package services

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jaredcannon/homelab-orchestration-platform/internal/models"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/ssh"
	"gorm.io/gorm"
)

// ResourceMonitoringService monitors resource usage across all devices
type ResourceMonitoringService struct {
	db               *gorm.DB
	sshClient        *ssh.Client
	deviceService    *DeviceService
	credService      *CredentialService
	pollInterval     time.Duration
	retentionPeriod  time.Duration
	maxConcurrent    int                          // Maximum concurrent device polls
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	running          bool
	mu               sync.RWMutex
	broadcastFunc    func(channel, event string, data interface{}) // WebSocket broadcast function
	failureCount     map[string]int               // Track consecutive failures per device
	failureCountMu   sync.Mutex

	// Observability metrics
	lastPollTime        time.Time
	lastPollDuration    time.Duration
	lastPollSuccessCount int
	lastPollFailureCount int
	totalPollsRun       int64
	totalMetricsCollected int64
	totalErrors         int64
	metricsMu           sync.RWMutex
}

// ResourceMonitoringConfig holds configuration for the monitoring service
type ResourceMonitoringConfig struct {
	PollInterval    time.Duration // How often to poll devices (default: 30s)
	RetentionPeriod time.Duration // How long to keep historical metrics (default: 24h)
	MaxConcurrent   int           // Maximum concurrent device polls (default: 10)
}

// NewResourceMonitoringService creates a new resource monitoring service
func NewResourceMonitoringService(db *gorm.DB, sshClient *ssh.Client, deviceService *DeviceService, credService *CredentialService, config *ResourceMonitoringConfig) *ResourceMonitoringService {
	if config == nil {
		config = &ResourceMonitoringConfig{
			PollInterval:    30 * time.Second,
			RetentionPeriod: 24 * time.Hour,
			MaxConcurrent:   10,
		}
	}

	// Set defaults for zero values
	if config.MaxConcurrent == 0 {
		config.MaxConcurrent = 10
	}

	return &ResourceMonitoringService{
		db:              db,
		sshClient:       sshClient,
		deviceService:   deviceService,
		credService:     credService,
		pollInterval:    config.PollInterval,
		retentionPeriod: config.RetentionPeriod,
		maxConcurrent:   config.MaxConcurrent,
		failureCount:    make(map[string]int),
	}
}

// SetBroadcastFunc sets the WebSocket broadcast function
func (rms *ResourceMonitoringService) SetBroadcastFunc(fn func(channel, event string, data interface{})) {
	rms.mu.Lock()
	defer rms.mu.Unlock()
	rms.broadcastFunc = fn
}

// Start begins monitoring all devices
func (rms *ResourceMonitoringService) Start() error {
	rms.mu.Lock()
	defer rms.mu.Unlock()

	if rms.running {
		return fmt.Errorf("resource monitoring service is already running")
	}

	ctx, cancel := context.WithCancel(context.Background())
	rms.cancel = cancel
	rms.running = true

	rms.wg.Add(1)
	go rms.monitoringLoop(ctx)

	log.Println("Resource monitoring service started")
	return nil
}

// Stop gracefully stops the monitoring service
func (rms *ResourceMonitoringService) Stop() error {
	rms.mu.Lock()
	defer rms.mu.Unlock()

	if !rms.running {
		return fmt.Errorf("resource monitoring service is not running")
	}

	if rms.cancel != nil {
		rms.cancel()
	}

	// Wait for goroutines to finish
	done := make(chan struct{})
	go func() {
		rms.wg.Wait()
		close(done)
	}()

	// Wait with timeout
	select {
	case <-done:
		log.Println("Resource monitoring service stopped")
	case <-time.After(10 * time.Second):
		return fmt.Errorf("timeout waiting for monitoring service to stop")
	}

	rms.running = false
	return nil
}

// IsRunning returns whether the service is currently running
func (rms *ResourceMonitoringService) IsRunning() bool {
	rms.mu.RLock()
	defer rms.mu.RUnlock()
	return rms.running
}

// MonitoringStatus represents the health and metrics of the monitoring service
type MonitoringStatus struct {
	Running             bool          `json:"running"`
	LastPollTime        *time.Time    `json:"last_poll_time,omitempty"`
	LastPollDuration    string        `json:"last_poll_duration,omitempty"`
	LastPollSuccess     int           `json:"last_poll_success"`
	LastPollFailure     int           `json:"last_poll_failure"`
	LastPollSuccessRate float64       `json:"last_poll_success_rate"`
	TotalPollsRun       int64         `json:"total_polls_run"`
	TotalMetricsCollected int64       `json:"total_metrics_collected"`
	TotalErrors         int64         `json:"total_errors"`
	OverallSuccessRate  float64       `json:"overall_success_rate"`
	TimeSinceLastPoll   string        `json:"time_since_last_poll,omitempty"`
	Healthy             bool          `json:"healthy"` // True if last poll was recent and successful
	HealthMessage       string        `json:"health_message,omitempty"`
}

// GetStatus returns detailed status and metrics for the monitoring service
func (rms *ResourceMonitoringService) GetStatus() *MonitoringStatus {
	rms.mu.RLock()
	running := rms.running
	rms.mu.RUnlock()

	rms.metricsMu.RLock()
	defer rms.metricsMu.RUnlock()

	status := &MonitoringStatus{
		Running:             running,
		TotalPollsRun:       rms.totalPollsRun,
		TotalMetricsCollected: rms.totalMetricsCollected,
		TotalErrors:         rms.totalErrors,
	}

	// Add last poll info if available
	if !rms.lastPollTime.IsZero() {
		status.LastPollTime = &rms.lastPollTime
		status.LastPollDuration = rms.lastPollDuration.String()
		status.LastPollSuccess = rms.lastPollSuccessCount
		status.LastPollFailure = rms.lastPollFailureCount

		// Calculate success rate for last poll
		lastTotal := rms.lastPollSuccessCount + rms.lastPollFailureCount
		if lastTotal > 0 {
			status.LastPollSuccessRate = float64(rms.lastPollSuccessCount) / float64(lastTotal) * 100
		}

		// Calculate time since last poll
		status.TimeSinceLastPoll = time.Since(rms.lastPollTime).String()
	}

	// Calculate overall success rate
	totalAttempts := rms.totalMetricsCollected + rms.totalErrors
	if totalAttempts > 0 {
		status.OverallSuccessRate = float64(rms.totalMetricsCollected) / float64(totalAttempts) * 100
	}

	// Determine health status
	if !running {
		status.Healthy = false
		status.HealthMessage = "Monitoring service is not running"
	} else if rms.lastPollTime.IsZero() {
		status.Healthy = false
		status.HealthMessage = "No polls completed yet"
	} else {
		// Check if last poll was recent (within 2x poll interval)
		maxAge := rms.pollInterval * 2
		age := time.Since(rms.lastPollTime)

		if age > maxAge {
			status.Healthy = false
			status.HealthMessage = fmt.Sprintf("Last poll was %v ago (expected every %v)", age.Round(time.Second), rms.pollInterval)
		} else if status.LastPollSuccessRate < 50 {
			status.Healthy = false
			status.HealthMessage = fmt.Sprintf("Low success rate: %.1f%%", status.LastPollSuccessRate)
		} else {
			status.Healthy = true
			status.HealthMessage = "Monitoring service is healthy"
		}
	}

	return status
}

// monitoringLoop is the main monitoring loop
func (rms *ResourceMonitoringService) monitoringLoop(ctx context.Context) {
	defer rms.wg.Done()

	// Initial poll
	rms.pollAllDevices()

	// Cleanup old metrics
	rms.cleanupOldMetrics()

	ticker := time.NewTicker(rms.pollInterval)
	defer ticker.Stop()

	cleanupTicker := time.NewTicker(1 * time.Hour)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Monitoring loop stopped")
			return
		case <-ticker.C:
			rms.pollAllDevices()
		case <-cleanupTicker.C:
			rms.cleanupOldMetrics()
		}
	}
}

// pollAllDevices polls all online devices for resource metrics using a worker pool
func (rms *ResourceMonitoringService) pollAllDevices() {
	startTime := time.Now()
	successCount := 0
	failureCount := 0

	var devices []models.Device
	if err := rms.db.Where("status = ?", models.DeviceStatusOnline).Find(&devices).Error; err != nil {
		log.Printf("[ResourceMonitoring] Error fetching online devices: %v", err)
		return
	}

	if len(devices) == 0 {
		return
	}

	log.Printf("[ResourceMonitoring] Polling %d online devices with max %d concurrent workers", len(devices), rms.maxConcurrent)

	// Create work queue channel
	deviceQueue := make(chan models.Device, len(devices))

	// Create worker pool with semaphore pattern
	sem := make(chan struct{}, rms.maxConcurrent)
	var wg sync.WaitGroup
	var countMu sync.Mutex

	// Add jitter to device order to spread load over time
	// Shuffle devices using Fisher-Yates algorithm to avoid always polling in the same order
	for i := len(devices) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		devices[i], devices[j] = devices[j], devices[i]
	}

	// Queue all devices
	for _, device := range devices {
		deviceQueue <- device
	}
	close(deviceQueue)

	// Start workers
	for device := range deviceQueue {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(d models.Device) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			// Add small random delay (0-5s) to stagger requests
			jitter := time.Duration(time.Now().UnixNano()%5000) * time.Millisecond
			time.Sleep(jitter)

			success := rms.pollDevice(d)

			// Track success/failure for this poll cycle
			countMu.Lock()
			if success {
				successCount++
			} else {
				failureCount++
			}
			countMu.Unlock()
		}(device)
	}

	// Wait for all workers to complete
	wg.Wait()

	// Update metrics
	duration := time.Since(startTime)
	rms.metricsMu.Lock()
	rms.lastPollTime = startTime
	rms.lastPollDuration = duration
	rms.lastPollSuccessCount = successCount
	rms.lastPollFailureCount = failureCount
	rms.totalPollsRun++
	rms.totalMetricsCollected += int64(successCount)
	rms.totalErrors += int64(failureCount)
	rms.metricsMu.Unlock()

	log.Printf("[ResourceMonitoring] Completed polling cycle: %d succeeded, %d failed in %v", successCount, failureCount, duration)
}

// pollDevice polls a single device for resource metrics
// Returns true if successful, false if failed
func (rms *ResourceMonitoringService) pollDevice(device models.Device) bool {
	deviceIDStr := device.ID.String()

	metrics, err := rms.collectDeviceMetrics(&device)
	if err != nil {
		log.Printf("[ResourceMonitoring] Error collecting metrics for device %s (%s): %v", device.Name, device.IPAddress, err)

		// Track consecutive failures
		rms.failureCountMu.Lock()
		rms.failureCount[deviceIDStr]++
		failCount := rms.failureCount[deviceIDStr]
		rms.failureCountMu.Unlock()

		// After 3 consecutive failures, clear device metrics (mark as stale)
		if failCount >= 3 {
			log.Printf("[ResourceMonitoring] Device %s has failed %d times, clearing stale metrics", device.Name, failCount)
			if err := rms.clearDeviceMetrics(&device); err != nil {
				log.Printf("[ResourceMonitoring] Error clearing metrics for %s: %v", device.Name, err)
			}
		}
		return false
	}

	// Success - reset failure count
	rms.failureCountMu.Lock()
	rms.failureCount[deviceIDStr] = 0
	rms.failureCountMu.Unlock()

	// Store metrics in database
	if err := rms.db.Create(metrics).Error; err != nil {
		log.Printf("[ResourceMonitoring] Error storing metrics for device %s: %v", device.Name, err)
		return false
	}

	// Update device with current metrics
	if err := rms.updateDeviceMetrics(&device, metrics); err != nil {
		log.Printf("[ResourceMonitoring] Error updating device metrics for %s: %v", device.Name, err)
		return false
	}

	// Broadcast update via WebSocket if available
	rms.broadcastResourceUpdate(&device, metrics)
	return true
}

// ensureConnection ensures an SSH connection exists for the device
func (rms *ResourceMonitoringService) ensureConnection(device *models.Device) error {
	host := fmt.Sprintf("%s:22", device.IPAddress)

	// Check if connection already exists
	_, err := rms.sshClient.GetConnection(host)
	if err == nil {
		// Connection exists and is alive
		return nil
	}

	// No connection exists, need to establish one
	// Get device credentials
	creds, err := rms.deviceService.GetDeviceCredentials(device.ID)
	if err != nil {
		return fmt.Errorf("failed to get credentials: %w", err)
	}

	// Establish connection based on auth type
	var connErr error
	switch creds.Type {
	case "auto":
		_, connErr = rms.sshClient.TryAutoAuth(host, creds.Username)
	case "password":
		_, connErr = rms.sshClient.ConnectWithPassword(host, creds.Username, creds.Password)
	case "ssh_key":
		_, connErr = rms.sshClient.ConnectWithKey(host, creds.Username, creds.SSHKey, creds.SSHKeyPasswd)
	case "tailscale":
		_, connErr = rms.sshClient.ConnectWithTailscale(host, creds.Username)
	default:
		return fmt.Errorf("unknown credential type: %s", creds.Type)
	}

	if connErr != nil {
		return fmt.Errorf("failed to establish SSH connection: %w", connErr)
	}

	log.Printf("[ResourceMonitoring] Established new SSH connection to %s (%s)", device.Name, device.IPAddress)
	return nil
}

// collectDeviceMetrics collects resource metrics from a device
func (rms *ResourceMonitoringService) collectDeviceMetrics(device *models.Device) (*models.DeviceMetrics, error) {
	// Ensure SSH connection exists before collecting metrics
	if err := rms.ensureConnection(device); err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}

	host := fmt.Sprintf("%s:22", device.IPAddress)
	metrics := &models.DeviceMetrics{
		DeviceID:   device.ID,
		RecordedAt: time.Now(),
	}

	// Collect CPU usage (1 second average) - use LANG=C to avoid locale issues
	cpuOutput, err := rms.sshClient.Execute(host, "LANG=C top -bn1 | grep 'Cpu(s)' | sed 's/.*, *\\([0-9.]*\\)%* id.*/\\1/' | awk '{print 100 - $1}'")
	if err == nil {
		cpuOutput = strings.TrimSpace(cpuOutput)
		if cpuOutput != "" {
			if cpu, err := strconv.ParseFloat(cpuOutput, 64); err == nil {
				metrics.CPUUsagePercent = cpu
			} else {
				log.Printf("[ResourceMonitoring] Failed to parse CPU output for %s: %s", device.Name, cpuOutput)
			}
		}
	} else {
		log.Printf("[ResourceMonitoring] Failed to collect CPU metrics for %s: %v", device.Name, err)
	}

	// Collect CPU cores
	cpuCoresOutput, err := rms.sshClient.Execute(host, "nproc")
	if err == nil {
		cpuCoresOutput = strings.TrimSpace(cpuCoresOutput)
		if cpuCoresOutput != "" {
			if cores, err := strconv.Atoi(cpuCoresOutput); err == nil {
				metrics.CPUCores = cores
			} else {
				log.Printf("[ResourceMonitoring] Failed to parse CPU cores for %s: %s", device.Name, cpuCoresOutput)
			}
		}
	} else {
		log.Printf("[ResourceMonitoring] Failed to collect CPU cores for %s: %v", device.Name, err)
	}

	// Collect RAM usage - use LANG=C to ensure consistent output
	ramOutput, err := rms.sshClient.Execute(host, "LANG=C free -m | grep Mem:")
	if err != nil {
		log.Printf("[ResourceMonitoring] Failed to collect RAM metrics for %s: %v", device.Name, err)
		return nil, fmt.Errorf("failed to collect RAM metrics: %w", err)
	}

	// Parse RAM output: Mem: total used free shared buff/cache available
	ramOutput = strings.TrimSpace(ramOutput)
	ramFields := strings.Fields(ramOutput)
	if len(ramFields) >= 7 {
		if total, err := strconv.Atoi(ramFields[1]); err == nil {
			metrics.TotalRAMMB = total
		} else {
			log.Printf("[ResourceMonitoring] Failed to parse RAM total for %s: %s", device.Name, ramFields[1])
		}
		if used, err := strconv.Atoi(ramFields[2]); err == nil {
			metrics.UsedRAMMB = used
		} else {
			log.Printf("[ResourceMonitoring] Failed to parse RAM used for %s: %s", device.Name, ramFields[2])
		}
		if available, err := strconv.Atoi(ramFields[6]); err == nil {
			metrics.AvailableRAMMB = available
		} else {
			log.Printf("[ResourceMonitoring] Failed to parse RAM available for %s: %s", device.Name, ramFields[6])
		}
	} else {
		log.Printf("[ResourceMonitoring] Unexpected RAM output format for %s (got %d fields): %s", device.Name, len(ramFields), ramOutput)
		return nil, fmt.Errorf("unexpected RAM output format: expected at least 7 fields, got %d", len(ramFields))
	}

	// Collect storage usage (root filesystem) - use LANG=C for consistent output
	storageOutput, err := rms.sshClient.Execute(host, "LANG=C df -BG / | tail -n 1")
	if err != nil {
		log.Printf("[ResourceMonitoring] Failed to collect storage metrics for %s: %v", device.Name, err)
		return nil, fmt.Errorf("failed to collect storage metrics: %w", err)
	}

	// Parse storage output: Filesystem 1G-blocks Used Available Use% Mounted on
	storageOutput = strings.TrimSpace(storageOutput)
	storageFields := strings.Fields(storageOutput)
	if len(storageFields) >= 4 {
		if total, err := strconv.Atoi(strings.TrimSuffix(storageFields[1], "G")); err == nil {
			metrics.TotalStorageGB = total
		} else {
			log.Printf("[ResourceMonitoring] Failed to parse storage total for %s: %s", device.Name, storageFields[1])
		}
		if used, err := strconv.Atoi(strings.TrimSuffix(storageFields[2], "G")); err == nil {
			metrics.UsedStorageGB = used
		} else {
			log.Printf("[ResourceMonitoring] Failed to parse storage used for %s: %s", device.Name, storageFields[2])
		}
		if available, err := strconv.Atoi(strings.TrimSuffix(storageFields[3], "G")); err == nil {
			metrics.AvailableStorageGB = available
		} else {
			log.Printf("[ResourceMonitoring] Failed to parse storage available for %s: %s", device.Name, storageFields[3])
		}
	} else {
		log.Printf("[ResourceMonitoring] Unexpected storage output format for %s (got %d fields): %s", device.Name, len(storageFields), storageOutput)
		return nil, fmt.Errorf("unexpected storage output format: expected at least 4 fields, got %d", len(storageFields))
	}

	return metrics, nil
}

// updateDeviceMetrics updates the device record with current metrics
func (rms *ResourceMonitoringService) updateDeviceMetrics(device *models.Device, metrics *models.DeviceMetrics) error {
	now := time.Now()
	updates := map[string]interface{}{
		"cpu_usage_percent":     metrics.CPUUsagePercent,
		"cpu_cores":             metrics.CPUCores,
		"total_ram_mb":          metrics.TotalRAMMB,
		"used_ram_mb":           metrics.UsedRAMMB,
		"available_ram_mb":      metrics.AvailableRAMMB,
		"total_storage_gb":      metrics.TotalStorageGB,
		"used_storage_gb":       metrics.UsedStorageGB,
		"available_storage_gb":  metrics.AvailableStorageGB,
		"resources_updated_at":  now,
	}

	return rms.db.Model(device).Updates(updates).Error
}

// clearDeviceMetrics clears stale metrics from a device after repeated collection failures
func (rms *ResourceMonitoringService) clearDeviceMetrics(device *models.Device) error {
	updates := map[string]interface{}{
		"cpu_usage_percent":     nil,
		"cpu_cores":             nil,
		"total_ram_mb":          nil,
		"used_ram_mb":           nil,
		"available_ram_mb":      nil,
		"total_storage_gb":      nil,
		"used_storage_gb":       nil,
		"available_storage_gb":  nil,
		"resources_updated_at":  nil,
	}

	return rms.db.Model(device).Updates(updates).Error
}

// broadcastResourceUpdate broadcasts resource updates via WebSocket
func (rms *ResourceMonitoringService) broadcastResourceUpdate(device *models.Device, metrics *models.DeviceMetrics) {
	rms.mu.RLock()
	broadcastFunc := rms.broadcastFunc
	rms.mu.RUnlock()

	if broadcastFunc == nil {
		return
	}

	data := map[string]interface{}{
		"device_id":              device.ID,
		"device_name":            device.Name,
		"cpu_usage_percent":      metrics.CPUUsagePercent,
		"cpu_cores":              metrics.CPUCores,
		"total_ram_mb":           metrics.TotalRAMMB,
		"used_ram_mb":            metrics.UsedRAMMB,
		"available_ram_mb":       metrics.AvailableRAMMB,
		"total_storage_gb":       metrics.TotalStorageGB,
		"used_storage_gb":        metrics.UsedStorageGB,
		"available_storage_gb":   metrics.AvailableStorageGB,
		"ram_usage_percent":      metrics.RAMUsagePercent(),
		"storage_usage_percent":  metrics.StorageUsagePercent(),
		"recorded_at":            metrics.RecordedAt,
	}

	broadcastFunc("resources", "device_metrics_updated", data)
}

// cleanupOldMetrics removes metrics older than the retention period
func (rms *ResourceMonitoringService) cleanupOldMetrics() {
	cutoff := time.Now().Add(-rms.retentionPeriod)
	result := rms.db.Where("recorded_at < ?", cutoff).Delete(&models.DeviceMetrics{})

	if result.Error != nil {
		log.Printf("Error cleaning up old metrics: %v", result.Error)
		return
	}

	if result.RowsAffected > 0 {
		log.Printf("Cleaned up %d old metric records", result.RowsAffected)
	}
}

// GetDeviceMetrics retrieves the current metrics for a device
func (rms *ResourceMonitoringService) GetDeviceMetrics(deviceID string) (*models.DeviceMetrics, error) {
	var metrics models.DeviceMetrics
	err := rms.db.Where("device_id = ?", deviceID).
		Order("recorded_at DESC").
		First(&metrics).Error

	if err != nil {
		return nil, err
	}

	return &metrics, nil
}

// GetDeviceMetricsHistory retrieves historical metrics for a device
func (rms *ResourceMonitoringService) GetDeviceMetricsHistory(deviceID string, since time.Time) ([]models.DeviceMetrics, error) {
	var metrics []models.DeviceMetrics
	err := rms.db.Where("device_id = ? AND recorded_at >= ?", deviceID, since).
		Order("recorded_at ASC").
		Find(&metrics).Error

	if err != nil {
		return nil, err
	}

	return metrics, nil
}

// AggregateResources represents aggregate resource metrics across all devices
type AggregateResources struct {
	TotalDevices        int     `json:"total_devices"`
	OnlineDevices       int     `json:"online_devices"`
	OfflineDevices      int     `json:"offline_devices"`
	TotalCPUCores       int     `json:"total_cpu_cores"`
	UsedCPUCores        float64 `json:"used_cpu_cores"`        // Absolute number of cores in use
	AvgCPUUsagePercent  float64 `json:"avg_cpu_usage_percent"` // Core-weighted percentage
	TotalRAMMB          int     `json:"total_ram_mb"`
	UsedRAMMB           int     `json:"used_ram_mb"`
	AvailableRAMMB      int     `json:"available_ram_mb"`
	RAMUsagePercent     float64 `json:"ram_usage_percent"`
	TotalStorageGB      int     `json:"total_storage_gb"`
	UsedStorageGB       int     `json:"used_storage_gb"`
	AvailableStorageGB  int     `json:"available_storage_gb"`
	StorageUsagePercent float64 `json:"storage_usage_percent"`
}

// GetAggregateResources calculates aggregate resource metrics across all devices
// Excludes devices with null/stale metrics (no data or collection failures)
func (rms *ResourceMonitoringService) GetAggregateResources() (*AggregateResources, error) {
	var devices []models.Device
	if err := rms.db.Find(&devices).Error; err != nil {
		return nil, err
	}

	agg := &AggregateResources{
		TotalDevices: len(devices),
	}

	for _, device := range devices {
		if device.Status == models.DeviceStatusOnline {
			agg.OnlineDevices++
		} else if device.Status == models.DeviceStatusOffline {
			agg.OfflineDevices++
		}

		// Only aggregate metrics from devices with valid data (not stale/null)
		// A device has valid metrics if resources_updated_at is not null
		if device.ResourcesUpdatedAt == nil {
			continue // Skip devices with stale/missing metrics
		}

		// Aggregate CPU with core-weighted calculation
		if device.CPUCores != nil {
			agg.TotalCPUCores += *device.CPUCores

			// Calculate actual cores in use for this device
			if device.CPUUsagePercent != nil {
				coresInUse := (float64(*device.CPUCores) * (*device.CPUUsagePercent)) / 100.0
				agg.UsedCPUCores += coresInUse
			}
		}

		// Aggregate RAM
		if device.TotalRAMMB != nil {
			agg.TotalRAMMB += *device.TotalRAMMB
		}
		if device.UsedRAMMB != nil {
			agg.UsedRAMMB += *device.UsedRAMMB
		}
		if device.AvailableRAMMB != nil {
			agg.AvailableRAMMB += *device.AvailableRAMMB
		}

		// Aggregate Storage
		if device.TotalStorageGB != nil {
			agg.TotalStorageGB += *device.TotalStorageGB
		}
		if device.UsedStorageGB != nil {
			agg.UsedStorageGB += *device.UsedStorageGB
		}
		if device.AvailableStorageGB != nil {
			agg.AvailableStorageGB += *device.AvailableStorageGB
		}
	}

	// Calculate percentages
	// CPU percentage is now core-weighted: (used cores / total cores) * 100
	if agg.TotalCPUCores > 0 {
		agg.AvgCPUUsagePercent = (agg.UsedCPUCores / float64(agg.TotalCPUCores)) * 100.0
	}

	if agg.TotalRAMMB > 0 {
		agg.RAMUsagePercent = (float64(agg.UsedRAMMB) / float64(agg.TotalRAMMB)) * 100
	}

	if agg.TotalStorageGB > 0 {
		agg.StorageUsagePercent = (float64(agg.UsedStorageGB) / float64(agg.TotalStorageGB)) * 100
	}

	return agg, nil
}
