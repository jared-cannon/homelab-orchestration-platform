package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/ssh"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CachePoolManager handles shared cache instance provisioning and management
// Similar to DatabasePoolManager but for Valkey/Redis/Memcached instances
type CachePoolManager struct {
	db           *gorm.DB
	sshClient    *ssh.Client
	infraConfig  *InfrastructureConfig
	orchestrator ContainerOrchestrator
}

// NewCachePoolManager creates a new CachePoolManager instance
func NewCachePoolManager(db *gorm.DB, sshClient *ssh.Client, infraConfig *InfrastructureConfig, orchestrator ContainerOrchestrator) *CachePoolManager {
	return &CachePoolManager{
		db:           db,
		sshClient:    sshClient,
		infraConfig:  infraConfig,
		orchestrator: orchestrator,
	}
}

// AutoMigrate creates the necessary database tables
func (cpm *CachePoolManager) AutoMigrate() error {
	return cpm.db.AutoMigrate(&models.SharedCacheInstance{}, &models.ProvisionedCacheConfig{})
}

// SharedInstanceExists checks if a shared cache instance exists for the given device and engine
// This is a wrapper method for DependencyService
func (cpm *CachePoolManager) SharedInstanceExists(ctx context.Context, deviceID uuid.UUID, engine string) (bool, error) {
	var count int64
	err := cpm.db.WithContext(ctx).Model(&models.SharedCacheInstance{}).
		Where("device_id = ? AND engine = ? AND status = ?", deviceID, engine, "running").
		Count(&count).Error

	if err != nil {
		return false, fmt.Errorf("failed to check for shared cache instance: %w", err)
	}

	return count > 0, nil
}

// GetOrCreateSharedInstance gets an existing shared cache instance or creates a new one
func (cpm *CachePoolManager) GetOrCreateSharedInstance(ctx context.Context, deviceID uuid.UUID, engine string, version string, name string) (*models.SharedCacheInstance, bool, error) {
	// Validate input parameters
	if deviceID == uuid.Nil {
		return nil, false, fmt.Errorf("deviceID cannot be nil")
	}

	// Validate engine using infrastructure config
	if err := cpm.infraConfig.ValidateCacheEngine(engine); err != nil {
		return nil, false, err
	}

	// Validate version
	if version == "" {
		return nil, false, fmt.Errorf("version cannot be empty")
	}

	// Name can be empty (will be generated), but validate if provided
	if name != "" && len(name) > 255 {
		return nil, false, fmt.Errorf("name too long: %d characters (max 255)", len(name))
	}

	// Check if a shared instance already exists
	var instance models.SharedCacheInstance
	err := cpm.db.WithContext(ctx).
		Where("device_id = ? AND engine = ? AND status = ?", deviceID, engine, "running").
		First(&instance).Error

	if err == nil {
		// Instance already exists
		return &instance, false, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, false, fmt.Errorf("failed to query cache instances: %w", err)
	}

	// Create a new shared instance
	masterPassword, err := generateSecurePassword(32)
	if err != nil {
		return nil, false, fmt.Errorf("failed to generate master password: %w", err)
	}

	// Find an available port using infrastructure config
	defaultPort := cpm.infraConfig.GetCachePort(engine)

	port, err := cpm.findAvailablePort(ctx, deviceID, defaultPort)
	if err != nil {
		return nil, false, fmt.Errorf("failed to find available port: %w", err)
	}

	if name == "" {
		name = fmt.Sprintf("shared-%s", engine)
	}

	containerName := fmt.Sprintf("%s_%s", name, deviceID.String()[:8])

	instance = models.SharedCacheInstance{
		DeviceID:       deviceID,
		Engine:         engine,
		Version:        version,
		Name:           name,
		Port:           port,
		ContainerName:  containerName,
		MasterPassword: masterPassword,
		MaxMemoryMB:    512, // Default 512MB for shared cache
		Status:         "provisioning",
	}

	if err := cpm.db.WithContext(ctx).Create(&instance).Error; err != nil {
		return nil, false, fmt.Errorf("failed to create cache instance: %w", err)
	}

	return &instance, true, nil
}

// ProvisionCacheInSharedInstance provisions cache access for an app in a shared cache instance
// This is a wrapper method for DependencyService
func (cpm *CachePoolManager) ProvisionCacheInSharedInstance(ctx context.Context, deviceID uuid.UUID, engine string, appSlug string) error {
	// Validate parameters
	if err := validateDeviceID(deviceID); err != nil {
		return err
	}
	if err := validateCacheEngine(engine); err != nil {
		return err
	}
	if err := validateAppSlug(appSlug); err != nil {
		return err
	}

	// Get or create the shared cache instance using config version
	version := cpm.infraConfig.GetCacheVersion(engine)

	instance, created, err := cpm.GetOrCreateSharedInstance(ctx, deviceID, engine, version, "")
	if err != nil {
		return fmt.Errorf("failed to get or create shared %s cache instance on device %s: %w", engine, deviceID, err)
	}

	// If the instance was just created, we need to deploy it
	if created {
		if err := cpm.deploySharedCacheInstance(ctx, instance); err != nil {
			// Cleanup on failure: remove Docker resources and database record
			log.Printf("[CachePool] Deployment failed, performing cleanup for instance %s", instance.ID)

			// Get device for Docker cleanup
			var device models.Device
			if getErr := cpm.db.WithContext(ctx).First(&device, deviceID).Error; getErr == nil {
				// Attempt Docker cleanup (best effort)
				if cleanupErr := cpm.cleanupDockerResources(device.GetPrimaryAddress(), instance.ContainerName, engine, deviceID); cleanupErr != nil {
					log.Printf("[CachePool] Warning: Failed to cleanup Docker resources: %v", cleanupErr)
				}
			}

			// Delete the database record
			if deleteErr := cpm.db.WithContext(ctx).Delete(instance).Error; deleteErr != nil {
				log.Printf("[ERROR] Failed to cleanup %s cache instance %s on device %s after deployment failure: %v", engine, instance.ID, deviceID, deleteErr)
			}

			return fmt.Errorf("failed to deploy shared %s cache instance on device %s: %w", engine, deviceID, err)
		}

		// Mark instance as running - use atomic update with transaction
		err := cpm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return tx.Model(&models.SharedCacheInstance{}).
				Where("id = ?", instance.ID).
				Update("status", "running").Error
		})
		if err != nil {
			return fmt.Errorf("failed to update cache instance status to running: %w", err)
		}
		instance.Status = "running" // Update in-memory object
	}

	// Check if app already has cache access
	var existingConfig models.ProvisionedCacheConfig
	err = cpm.db.WithContext(ctx).Where("cache_instance_id = ? AND app_slug = ?", instance.ID, appSlug).
		First(&existingConfig).Error

	if err == nil {
		// Already provisioned
		return nil
	}

	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to check existing cache config for app %s on instance %s: %w", appSlug, instance.ID, err)
	}

	// Assign a database number if the cache engine supports it (Redis/Valkey: 0-15)
	// For Memcached, this stays 0 (unused)
	var dbNumber int
	if cpm.infraConfig.SupportsDatabases(engine) {
		dbNumber, err = cpm.findAvailableDatabaseNumber(ctx, instance.ID, engine)
		if err != nil {
			return fmt.Errorf("failed to find available database number for app %s on instance %s: %w", appSlug, instance.ID, err)
		}
	}

	// Create cache config for the app
	config := models.ProvisionedCacheConfig{
		CacheInstanceID: instance.ID,
		AppSlug:         appSlug,
		DeviceID:        deviceID,
		DatabaseNumber:  dbNumber,
		KeyPrefix:       fmt.Sprintf("%s:", appSlug), // Use app slug as key prefix
		Password:        "", // Using master password by default
		MaxMemoryMB:     64, // Default 64MB per app
	}

	if err := cpm.db.WithContext(ctx).Create(&config).Error; err != nil {
		return fmt.Errorf("failed to create cache config for app %s on instance %s: %w", appSlug, instance.ID, err)
	}

	return nil
}

// GetCacheCredentials retrieves cache connection credentials for an app
// This is a wrapper method for DependencyService
func (cpm *CachePoolManager) GetCacheCredentials(ctx context.Context, deviceID uuid.UUID, engine string, appSlug string) (map[string]string, error) {
	// Validate parameters
	if err := validateDeviceID(deviceID); err != nil {
		return nil, err
	}
	if err := validateCacheEngine(engine); err != nil {
		return nil, err
	}
	if err := validateAppSlug(appSlug); err != nil {
		return nil, err
	}

	// Get the device to obtain its hostname/IP
	var device models.Device
	err := cpm.db.WithContext(ctx).First(&device, deviceID).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find device %s: %w", deviceID, err)
	}

	// Use device IP address for network connectivity
	// This allows apps to connect to cache instances on remote devices
	host := device.GetPrimaryAddress()
	if host == "" {
		// In a distributed system, empty IP address means misconfiguration
		// Don't fallback to localhost as it would connect to wrong host
		return nil, fmt.Errorf("device %s has no IP address configured - cannot generate cache credentials", deviceID)
	}

	// Get the cache instance
	var instance models.SharedCacheInstance
	err = cpm.db.WithContext(ctx).
		Where("device_id = ? AND engine = ? AND status = ?", deviceID, engine, "running").
		First(&instance).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find %s cache instance on device %s: %w", engine, deviceID, err)
	}

	// Get the app config
	var config models.ProvisionedCacheConfig
	err = cpm.db.WithContext(ctx).
		Where("cache_instance_id = ? AND app_slug = ?", instance.ID, appSlug).
		First(&config).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find cache config for app %s on instance %s: %w", appSlug, instance.ID, err)
	}

	// Build credentials map with correct host
	credentials := map[string]string{
		"CACHE_HOST":     host,
		"CACHE_PORT":     fmt.Sprintf("%d", instance.Port),
		"CACHE_PASSWORD": instance.MasterPassword,
		"CACHE_ENGINE":   instance.Engine,
	}

	// Redis/Valkey-specific credentials (they share the same protocol)
	if instance.Engine == "redis" || instance.Engine == "valkey" {
		credentials["REDIS_HOST"] = host
		credentials["REDIS_PORT"] = fmt.Sprintf("%d", instance.Port)
		credentials["REDIS_PASSWORD"] = instance.MasterPassword
		credentials["REDIS_DB"] = fmt.Sprintf("%d", config.DatabaseNumber)
		credentials["REDIS_PREFIX"] = config.KeyPrefix

		// Also provide valkey-specific credentials for clarity
		credentials["VALKEY_HOST"] = host
		credentials["VALKEY_PORT"] = fmt.Sprintf("%d", instance.Port)
		credentials["VALKEY_PASSWORD"] = instance.MasterPassword
		credentials["VALKEY_DB"] = fmt.Sprintf("%d", config.DatabaseNumber)
		credentials["VALKEY_PREFIX"] = config.KeyPrefix
	}

	// Memcached-specific credentials
	if instance.Engine == "memcached" {
		credentials["MEMCACHED_HOST"] = host
		credentials["MEMCACHED_PORT"] = fmt.Sprintf("%d", instance.Port)
	}

	return credentials, nil
}

// deploySharedCacheInstance deploys a shared cache instance using the orchestrator
func (cpm *CachePoolManager) deploySharedCacheInstance(ctx context.Context, instance *models.SharedCacheInstance) error {
	// Get device to determine host
	var device models.Device
	if err := cpm.db.First(&device, instance.DeviceID).Error; err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	host := device.GetSSHHost()
	deployDir := fmt.Sprintf("~/homelab-deployments/shared-%s-%s", instance.Engine, instance.DeviceID.String()[:8])

	// Generate docker-compose.yaml for the cache instance
	composeContent, err := cpm.generateCacheInstanceCompose(instance)
	if err != nil {
		return fmt.Errorf("failed to generate compose file: %w", err)
	}

	// Create deployment spec
	spec := DeploymentSpec{
		Host:           host,
		StackName:      instance.ContainerName,
		DeployDir:      deployDir,
		ComposeContent: composeContent,
		Timeout:        10 * time.Minute,
	}

	// Deploy using orchestrator
	if err := cpm.orchestrator.Deploy(ctx, spec); err != nil {
		return fmt.Errorf("orchestrator deployment failed: %w", err)
	}

	// Wait for cache to be ready
	if err := cpm.orchestrator.WaitForHealthy(ctx, instance.ContainerName, host, 5*time.Minute); err != nil {
		return fmt.Errorf("cache failed to become healthy: %w", err)
	}

	log.Printf("[CachePool] Shared %s instance deployed successfully on %s", instance.Engine, device.Name)
	return nil
}

// generateCacheInstanceCompose generates docker-compose for a cache instance
func (cpm *CachePoolManager) generateCacheInstanceCompose(instance *models.SharedCacheInstance) (string, error) {
	dockerImage := cpm.infraConfig.GetCacheImage(instance.Engine)

	switch instance.Engine {
	case "valkey", "redis":
		return cpm.generateValkeyRedisCompose(instance, dockerImage), nil
	case "memcached":
		return cpm.generateMemcachedCompose(instance, dockerImage), nil
	default:
		return "", fmt.Errorf("unsupported cache engine: %s", instance.Engine)
	}
}

// generateValkeyRedisCompose generates docker-compose for Valkey or Redis
func (cpm *CachePoolManager) generateValkeyRedisCompose(instance *models.SharedCacheInstance, dockerImage string) string {
	volumeName := fmt.Sprintf("%s-data", instance.ContainerName)
	networkName := fmt.Sprintf("%s-network", instance.ContainerName)

	// Both Valkey and Redis use the same binary name and CLI commands
	// Valkey is a fork maintaining Redis protocol compatibility
	// Use environment variables for password to avoid exposing in command line
	return fmt.Sprintf(`version: '3.8'
services:
  cache:
    image: %s:%s
    container_name: %s
    restart: unless-stopped
    environment:
      - REDIS_PASSWORD=%s
    command: redis-server --requirepass "${REDIS_PASSWORD}" --maxmemory %dmb --maxmemory-policy allkeys-lru
    volumes:
      - %s:/data
    ports:
      - "%d:6379"
    networks:
      - %s
    healthcheck:
      test: ["CMD", "sh", "-c", "redis-cli -a $$REDIS_PASSWORD ping"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  %s:
    driver: local

networks:
  %s:
    driver: bridge
`, dockerImage, instance.Version, instance.ContainerName, instance.MasterPassword, instance.MaxMemoryMB, volumeName, instance.Port, networkName, volumeName, networkName)
}

// generateMemcachedCompose generates docker-compose for Memcached
func (cpm *CachePoolManager) generateMemcachedCompose(instance *models.SharedCacheInstance, dockerImage string) string {
	networkName := fmt.Sprintf("%s-network", instance.ContainerName)

	return fmt.Sprintf(`version: '3.8'
services:
  cache:
    image: %s:%s
    container_name: %s
    restart: unless-stopped
    command: memcached -m %d
    ports:
      - "%d:11211"
    networks:
      - %s
    healthcheck:
      test: ["CMD", "sh", "-c", "echo stats | nc localhost 11211 | grep -q uptime"]
      interval: 10s
      timeout: 5s
      retries: 5

networks:
  %s:
    driver: bridge
`, dockerImage, instance.Version, instance.ContainerName, instance.MaxMemoryMB, instance.Port, networkName, networkName)
}

// cleanupDockerResources stops and removes Docker containers using the orchestrator
// This is a reusable helper for cleaning up cache deployments
func (cpm *CachePoolManager) cleanupDockerResources(deviceIP string, containerName string, engine string, deviceID uuid.UUID) error {
	host := fmt.Sprintf("%s:22", deviceIP)
	deployDir := fmt.Sprintf("~/homelab-deployments/shared-%s-%s", engine, deviceID.String()[:8])

	// Create removal spec
	spec := RemovalSpec{
		Host:           host,
		StackName:      containerName,
		DeployDir:      deployDir,
		ContainerName:  containerName,
		IncludeVolumes: true,
	}

	// Use orchestrator's cleanup method
	if err := cpm.orchestrator.RemoveWithCleanup(context.Background(), spec); err != nil {
		log.Printf("[CachePool] Warning: orchestrator cleanup failed for %s: %v", containerName, err)
	}

	log.Printf("[CachePool] Docker cleanup completed for %s", containerName)
	return nil
}

// checkPortOnHost checks if a port is actually listening on the target host
// This method does not access the database and is safe to call within transactions
func (cpm *CachePoolManager) checkPortOnHost(host string, port int) (bool, error) {
	// Check if port is listening using ss (socket statistics)
	// ss is more portable and efficient than netstat
	checkCmd := fmt.Sprintf("ss -tuln | grep ':%d ' || true", port)
	output, err := cpm.sshClient.ExecuteWithTimeout(host, checkCmd, 10*time.Second)

	// If there's an error executing the command (not just empty output), return error
	if err != nil {
		return false, fmt.Errorf("failed to check port on host %s: %w", host, err)
	}

	// If output is non-empty, port is in use
	return strings.TrimSpace(output) != "", nil
}

// findAvailablePort finds an available port for a cache instance on the device
// Uses database transaction with row-level locking to prevent race conditions
// Also verifies port is not in use on the actual system
func (cpm *CachePoolManager) findAvailablePort(ctx context.Context, deviceID uuid.UUID, startPort int) (int, error) {
	// Query device BEFORE entering transaction to avoid nested database queries
	var device models.Device
	if err := cpm.db.WithContext(ctx).First(&device, deviceID).Error; err != nil {
		return 0, fmt.Errorf("failed to get device %s: %w", deviceID, err)
	}

	// Build SSH host string for port checking
	host := device.GetSSHHost()

	var port int

	// Use transaction with FOR UPDATE lock to prevent race condition
	err := cpm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get all used ports with row lock
		var usedPorts []int
		err := tx.Model(&models.SharedCacheInstance{}).
			Where("device_id = ?", deviceID).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Pluck("port", &usedPorts).Error

		if err != nil {
			return fmt.Errorf("failed to query used ports: %w", err)
		}

		// Find available port
		candidatePort := startPort
		maxAttempts := 100 // Prevent infinite loop
		attempts := 0

		for {
			attempts++
			if attempts > maxAttempts {
				return fmt.Errorf("failed to find available port after %d attempts", maxAttempts)
			}

			// Validate port range (avoid privileged ports < 1024)
			if candidatePort < 1024 {
				candidatePort = 1024
			}
			if candidatePort > 65535 {
				return fmt.Errorf("no available ports in valid range (1024-65535)")
			}

			// Check if port is in database
			usedInDB := false
			for _, usedPort := range usedPorts {
				if usedPort == candidatePort {
					usedInDB = true
					break
				}
			}

			if !usedInDB {
				// Port not in database, check if it's actually available on the system
				// Uses host string queried before transaction (no nested DB queries)
				inUse, err := cpm.checkPortOnHost(host, candidatePort)
				if err != nil {
					// Log warning but don't fail - continue trying other ports
					log.Printf("[CachePool] Warning: Could not check if port %d is in use on device %s: %v", candidatePort, deviceID, err)
					// Assume port is available if we can't check
					port = candidatePort
					return nil
				}

				if !inUse {
					port = candidatePort
					return nil
				}
			}

			candidatePort++
		}
	})

	if err != nil {
		return 0, err
	}

	return port, nil
}

// findAvailableDatabaseNumber finds an available database number for the cache instance
// Uses database transaction with row-level locking to prevent race conditions
func (cpm *CachePoolManager) findAvailableDatabaseNumber(ctx context.Context, instanceID uuid.UUID, engine string) (int, error) {
	var dbNumber int

	// Get max databases from config (Redis/Valkey: 16, Memcached: 0)
	maxDatabases := cpm.infraConfig.GetMaxDatabases(engine)
	if maxDatabases == 0 {
		return 0, fmt.Errorf("engine %s does not support database numbers", engine)
	}

	// Use transaction with FOR UPDATE lock to prevent race condition
	err := cpm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get all used database numbers with row lock
		var usedDBs []int
		err := tx.Model(&models.ProvisionedCacheConfig{}).
			Where("cache_instance_id = ?", instanceID).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Pluck("database_number", &usedDBs).Error

		if err != nil {
			return fmt.Errorf("failed to query used database numbers: %w", err)
		}

		// Find the next available database number (0 to maxDatabases-1)
		for candidateDB := 0; candidateDB < maxDatabases; candidateDB++ {
			used := false
			for _, usedDB := range usedDBs {
				if usedDB == candidateDB {
					used = true
					break
				}
			}

			if !used {
				dbNumber = candidateDB
				return nil
			}
		}

		return fmt.Errorf("no available database numbers (maximum %d databases per %s instance)", maxDatabases, engine)
	})

	if err != nil {
		return 0, err
	}

	return dbNumber, nil
}

// GetAllInstances returns all cache instances for a device
func (cpm *CachePoolManager) GetAllInstances(ctx context.Context, deviceID uuid.UUID) ([]models.SharedCacheInstance, error) {
	var instances []models.SharedCacheInstance
	err := cpm.db.WithContext(ctx).Where("device_id = ?", deviceID).Find(&instances).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query cache instances: %w", err)
	}
	return instances, nil
}

// GetInstance returns a specific cache instance by ID
func (cpm *CachePoolManager) GetInstance(ctx context.Context, instanceID uuid.UUID) (*models.SharedCacheInstance, error) {
	var instance models.SharedCacheInstance
	err := cpm.db.WithContext(ctx).First(&instance, instanceID).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get cache instance: %w", err)
	}
	return &instance, nil
}

// GetAppConfigs returns all app configurations for a cache instance
func (cpm *CachePoolManager) GetAppConfigs(ctx context.Context, instanceID uuid.UUID) ([]models.ProvisionedCacheConfig, error) {
	var configs []models.ProvisionedCacheConfig
	err := cpm.db.WithContext(ctx).Where("cache_instance_id = ?", instanceID).Find(&configs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query app configs: %w", err)
	}
	return configs, nil
}

// DeleteInstance removes a cache instance and all its app configurations
func (cpm *CachePoolManager) DeleteInstance(ctx context.Context, instanceID uuid.UUID) error {
	// Get the instance first
	instance, err := cpm.GetInstance(ctx, instanceID)
	if err != nil {
		return err
	}

	// Get device to perform Docker cleanup
	var device models.Device
	if err := cpm.db.WithContext(ctx).First(&device, instance.DeviceID).Error; err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	// Stop and remove Docker containers
	if err := cpm.cleanupDockerResources(device.GetPrimaryAddress(), instance.ContainerName, instance.Engine, instance.DeviceID); err != nil {
		// Log error but continue with database cleanup
		log.Printf("[CachePool] Warning: Failed to cleanup Docker resources for instance %s: %v", instanceID, err)
	}

	// Delete all app configurations
	if err := cpm.db.WithContext(ctx).Where("cache_instance_id = ?", instanceID).Delete(&models.ProvisionedCacheConfig{}).Error; err != nil {
		return fmt.Errorf("failed to delete app configs: %w", err)
	}

	// Delete the instance
	if err := cpm.db.WithContext(ctx).Delete(instance).Error; err != nil {
		return fmt.Errorf("failed to delete cache instance: %w", err)
	}

	log.Printf("[CachePool] Successfully deleted cache instance %s", instanceID)
	return nil
}

// DeleteAppConfig removes cache access for a specific app
func (cpm *CachePoolManager) DeleteAppConfig(ctx context.Context, instanceID uuid.UUID, appSlug string) error {
	result := cpm.db.WithContext(ctx).
		Where("cache_instance_id = ? AND app_slug = ?", instanceID, appSlug).
		Delete(&models.ProvisionedCacheConfig{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete app config: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("app config not found")
	}

	return nil
}

// UpdateInstanceStatus updates the status of a cache instance
func (cpm *CachePoolManager) UpdateInstanceStatus(ctx context.Context, instanceID uuid.UUID, status string) error {
	// Validate status
	if err := validateCacheInstanceStatus(status); err != nil {
		return err
	}

	result := cpm.db.WithContext(ctx).Model(&models.SharedCacheInstance{}).
		Where("id = ?", instanceID).
		Update("status", status)

	if result.Error != nil {
		return fmt.Errorf("failed to update instance %s status: %w", instanceID, result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("cache instance %s not found", instanceID)
	}

	return nil
}

// GetInstanceStats returns usage statistics for a cache instance
func (cpm *CachePoolManager) GetInstanceStats(ctx context.Context, instanceID uuid.UUID) (map[string]interface{}, error) {
	// Get the instance
	instance, err := cpm.GetInstance(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	// Get app configs
	configs, err := cpm.GetAppConfigs(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	// Calculate total allocated memory
	totalAllocatedMB := 0
	for _, config := range configs {
		totalAllocatedMB += config.MaxMemoryMB
	}

	// Calculate utilization (safe division)
	utilizationPercent := 0.0
	if instance.MaxMemoryMB > 0 {
		utilizationPercent = float64(totalAllocatedMB) / float64(instance.MaxMemoryMB) * 100
	}

	stats := map[string]interface{}{
		"instance_id":         instance.ID,
		"engine":              instance.Engine,
		"status":              instance.Status,
		"max_memory_mb":       instance.MaxMemoryMB,
		"allocated_memory_mb": totalAllocatedMB,
		"available_memory_mb": instance.MaxMemoryMB - totalAllocatedMB,
		"app_count":           len(configs),
		"utilization_percent": utilizationPercent,
	}

	// Add database stats for engines that support databases (Redis/Valkey)
	if instance.Engine == "redis" || instance.Engine == "valkey" {
		maxDBs := cpm.infraConfig.GetMaxDatabases(instance.Engine)
		stats["databases_used"] = len(configs)
		stats["databases_available"] = maxDBs - len(configs)
		stats["supports_databases"] = true
	} else {
		stats["supports_databases"] = false
	}

	return stats, nil
}

// generateSecurePassword generates a cryptographically secure random password
func generateSecurePassword(length int) (string, error) {
	// Validate length parameter
	if length < 8 {
		return "", fmt.Errorf("password length must be at least 8 characters, got %d", length)
	}
	if length > 128 {
		return "", fmt.Errorf("password length must be at most 128 characters, got %d", length)
	}

	// Generate enough random bytes
	// Base64 encoding expands size by ~33%, so we need more bytes
	numBytes := (length * 3) / 4
	if numBytes < length {
		numBytes = length
	}

	bytes := make([]byte, numBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	encoded := base64.URLEncoding.EncodeToString(bytes)

	// Ensure we have enough characters
	if len(encoded) < length {
		return "", fmt.Errorf("generated password too short: %d < %d", len(encoded), length)
	}

	return encoded[:length], nil
}

// Validation helper functions (reusable across methods)

// validateDeviceID validates that a device ID is not nil
func validateDeviceID(deviceID uuid.UUID) error {
	if deviceID == uuid.Nil {
		return fmt.Errorf("deviceID cannot be nil")
	}
	return nil
}

// validateCacheEngine validates that the engine is supported (deprecated - use infraConfig instead)
// Kept for backward compatibility but should use infraConfig.ValidateCacheEngine
func validateCacheEngine(engine string) error {
	validEngines := []string{"redis", "valkey", "memcached"}
	for _, valid := range validEngines {
		if engine == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid cache engine: %s (must be 'redis', 'valkey', or 'memcached')", engine)
}

// validateAppSlug validates that an app slug is valid
func validateAppSlug(appSlug string) error {
	if appSlug == "" {
		return fmt.Errorf("appSlug cannot be empty")
	}
	if len(appSlug) > 255 {
		return fmt.Errorf("appSlug too long: %d characters (max 255)", len(appSlug))
	}
	// Additional validation: app slug should be alphanumeric with hyphens/underscores
	for _, ch := range appSlug {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			 (ch >= '0' && ch <= '9') || ch == '-' || ch == '_') {
			return fmt.Errorf("appSlug contains invalid character '%c' (only alphanumeric, hyphens, and underscores allowed)", ch)
		}
	}
	return nil
}

// validateCacheInstanceStatus validates that a status is valid
func validateCacheInstanceStatus(status string) error {
	validStatuses := map[string]bool{
		"provisioning": true,
		"running":      true,
		"stopped":      true,
		"error":        true,
		"failed":       true,
	}
	if !validStatuses[status] {
		return fmt.Errorf("invalid status: %s (must be one of: provisioning, running, stopped, error, failed)", status)
	}
	return nil
}
