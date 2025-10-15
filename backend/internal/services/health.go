package services

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/ssh"
	"gorm.io/gorm"
)

// WebSocketBroadcaster interface for broadcasting WebSocket messages
type WebSocketBroadcaster interface {
	Broadcast(channel string, event string, data interface{})
}

// HealthCheckService handles background device health monitoring
type HealthCheckService struct {
	db               *gorm.DB
	sshClient        *ssh.Client
	deviceService    *DeviceService
	credService      *CredentialService
	wsHub            WebSocketBroadcaster
	checkInterval    time.Duration
	cancel           context.CancelFunc
	maxConcurrency   int // Maximum concurrent health checks
}

// NewHealthCheckService creates a new health check service
func NewHealthCheckService(db *gorm.DB, sshClient *ssh.Client, credService *CredentialService) *HealthCheckService {
	return &HealthCheckService{
		db:             db,
		sshClient:      sshClient,
		credService:    credService,
		checkInterval:  30 * time.Second, // Check every 30 seconds
		maxConcurrency: 10,                // Max 10 concurrent health checks
	}
}

// SetDeviceService sets the device service (to avoid circular dependency)
func (h *HealthCheckService) SetDeviceService(ds *DeviceService) {
	h.deviceService = ds
}

// SetWebSocketHub sets the WebSocket hub for broadcasting status changes
func (h *HealthCheckService) SetWebSocketHub(hub WebSocketBroadcaster) {
	h.wsHub = hub
}

// Start begins the background health check loop
func (h *HealthCheckService) Start(ctx context.Context) {
	log.Println("[HealthCheck] Starting health check service")

	ctx, cancel := context.WithCancel(ctx)
	h.cancel = cancel

	// Run initial health check asynchronously to avoid blocking server startup
	// Devices will start with "unknown" status and update as checks complete
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(1 * time.Millisecond): // Non-blocking start
			log.Println("[HealthCheck] Running initial device health checks")
			h.checkAllDevices(ctx)
		}
	}()

	// Run periodic checks
	ticker := time.NewTicker(h.checkInterval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				log.Println("[HealthCheck] Health check service stopped")
				return
			case <-ticker.C:
				h.checkAllDevices(ctx)
			}
		}
	}()
}

// Stop stops the background health check loop
func (h *HealthCheckService) Stop() {
	if h.cancel != nil {
		h.cancel()
	}
}

// checkAllDevices checks health of all devices concurrently with worker pool
func (h *HealthCheckService) checkAllDevices(ctx context.Context) {
	var devices []models.Device
	if err := h.db.Find(&devices).Error; err != nil {
		log.Printf("[HealthCheck] Error fetching devices: %v", err)
		return
	}

	if len(devices) == 0 {
		return
	}

	log.Printf("[HealthCheck] Checking health of %d devices (max %d concurrent)", len(devices), h.maxConcurrency)

	// Create worker pool with bounded concurrency
	deviceChan := make(chan uuid.UUID, len(devices))
	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < h.maxConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case deviceID, ok := <-deviceChan:
					if !ok {
						return
					}
					h.checkDeviceHealth(ctx, deviceID)
				}
			}
		}()
	}

	// Send device IDs to workers
	for _, device := range devices {
		select {
		case <-ctx.Done():
			close(deviceChan)
			wg.Wait()
			return
		case deviceChan <- device.ID:
		}
	}

	// Close channel and wait for workers to finish
	close(deviceChan)
	wg.Wait()
}

// checkDeviceHealth checks the health of a single device
func (h *HealthCheckService) checkDeviceHealth(ctx context.Context, deviceID uuid.UUID) {
	// Check for cancellation before starting
	select {
	case <-ctx.Done():
		return
	default:
	}

	// Check if deviceService is initialized (avoid race condition)
	if h.deviceService == nil {
		log.Printf("[HealthCheck] Device service not yet initialized, skipping check")
		return
	}

	var device models.Device
	if err := h.db.First(&device, "id = ?", deviceID).Error; err != nil {
		return
	}

	// Try to establish SSH connection
	host := device.IPAddress + ":22"

	// First, try to get existing connection from pool (avoids re-authentication)
	client, err := h.sshClient.GetConnection(host)

	// If no existing connection, create a new one
	if err != nil {
		// Check for cancellation before expensive operation
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Use DeviceService to get credentials (handles both DB and keychain sources)
		creds, credErr := h.deviceService.GetDeviceCredentials(deviceID)
		if credErr != nil {
			log.Printf("[HealthCheck] Failed to get credentials for %s: %v", device.Name, credErr)
			h.updateDeviceStatus(deviceID, device.Name, models.DeviceStatusOffline)
			return
		}

		// Create new connection based on credential type
		if creds.Type == "password" {
			client, err = h.sshClient.ConnectWithPassword(host, creds.Username, creds.Password)
		} else if creds.Type == "ssh_key" {
			client, err = h.sshClient.ConnectWithKey(host, creds.Username, creds.SSHKey, creds.SSHKeyPasswd)
		} else if creds.Type == "auto" {
			client, err = h.sshClient.TryAutoAuth(host, creds.Username)
		} else if creds.Type == "tailscale" {
			client, err = h.sshClient.ConnectWithTailscale(host, creds.Username)
		} else {
			log.Printf("[HealthCheck] Unknown credential type for %s: %s", device.Name, creds.Type)
			h.updateDeviceStatus(deviceID, device.Name, models.DeviceStatusError)
			return
		}

		if err != nil {
			log.Printf("[HealthCheck] Device %s is offline: %v", device.Name, err)
			h.updateDeviceStatus(deviceID, device.Name, models.DeviceStatusOffline)
			return
		}
		log.Printf("[HealthCheck] Created new SSH connection for %s", device.Name)
	} else {
		log.Printf("[HealthCheck] Reusing existing SSH connection for %s", device.Name)
	}

	// Check for cancellation before running command
	select {
	case <-ctx.Done():
		return
	default:
	}

	// Connection successful - device is online
	// Run a simple command to verify SSH is actually working
	session, err := client.NewSession()
	if err != nil {
		log.Printf("[HealthCheck] Device %s SSH session failed: %v", device.Name, err)
		h.updateDeviceStatus(deviceID, device.Name, models.DeviceStatusError)
		return
	}
	defer session.Close()

	_, err = session.CombinedOutput("echo ping")
	if err != nil {
		log.Printf("[HealthCheck] Device %s SSH command failed: %v", device.Name, err)
		h.updateDeviceStatus(deviceID, device.Name, models.DeviceStatusError)
		return
	}

	log.Printf("[HealthCheck] Device %s is online", device.Name)
	h.updateDeviceStatus(deviceID, device.Name, models.DeviceStatusOnline)
}

// updateDeviceStatus updates the device status and last_seen timestamp, then broadcasts via WebSocket
func (h *HealthCheckService) updateDeviceStatus(deviceID uuid.UUID, deviceName string, status models.DeviceStatus) {
	now := time.Now()
	if err := h.db.Model(&models.Device{}).Where("id = ?", deviceID).Updates(map[string]interface{}{
		"status":    status,
		"last_seen": &now,
	}).Error; err != nil {
		log.Printf("[HealthCheck] Failed to update device status: %v", err)
		return
	}

	// Broadcast status change via WebSocket if hub is available
	if h.wsHub != nil {
		h.wsHub.Broadcast("devices", "status_change", map[string]interface{}{
			"device_id":   deviceID.String(),
			"device_name": deviceName,
			"status":      string(status),
			"last_seen":   now.Format(time.RFC3339),
		})
	}
}
