package services

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/models"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/ssh"
	"gorm.io/gorm"
)

// HealthCheckService handles background device health monitoring
type HealthCheckService struct {
	db             *gorm.DB
	sshClient      *ssh.Client
	deviceService  *DeviceService
	credService    *CredentialService
	checkInterval  time.Duration
	cancel         context.CancelFunc
}

// NewHealthCheckService creates a new health check service
func NewHealthCheckService(db *gorm.DB, sshClient *ssh.Client, credService *CredentialService) *HealthCheckService {
	return &HealthCheckService{
		db:            db,
		sshClient:     sshClient,
		credService:   credService,
		checkInterval: 30 * time.Second, // Check every 30 seconds
	}
}

// SetDeviceService sets the device service (to avoid circular dependency)
func (h *HealthCheckService) SetDeviceService(ds *DeviceService) {
	h.deviceService = ds
}

// Start begins the background health check loop
func (h *HealthCheckService) Start(ctx context.Context) {
	log.Println("[HealthCheck] Starting health check service")

	ctx, cancel := context.WithCancel(ctx)
	h.cancel = cancel

	// Run initial health check immediately
	h.checkAllDevices()

	// Then run periodic checks
	ticker := time.NewTicker(h.checkInterval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				log.Println("[HealthCheck] Health check service stopped")
				return
			case <-ticker.C:
				h.checkAllDevices()
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

// checkAllDevices checks health of all devices
func (h *HealthCheckService) checkAllDevices() {
	var devices []models.Device
	if err := h.db.Find(&devices).Error; err != nil {
		log.Printf("[HealthCheck] Error fetching devices: %v", err)
		return
	}

	if len(devices) == 0 {
		return
	}

	log.Printf("[HealthCheck] Checking health of %d devices", len(devices))

	for _, device := range devices {
		h.checkDeviceHealth(device.ID)
	}
}

// checkDeviceHealth checks the health of a single device
func (h *HealthCheckService) checkDeviceHealth(deviceID uuid.UUID) {
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
		// Get credentials only when needed for new connection
		creds, credErr := h.credService.GetCredentials(deviceID.String())
		if credErr != nil {
			log.Printf("[HealthCheck] Failed to get credentials for %s: %v", device.Name, credErr)
			h.updateDeviceStatus(deviceID, models.DeviceStatusError)
			return
		}

		// Create new connection based on credential type
		if creds.Type == "password" {
			client, err = h.sshClient.ConnectWithPassword(host, creds.Username, creds.Password)
		} else if creds.Type == "ssh_key" {
			client, err = h.sshClient.ConnectWithKey(host, creds.Username, creds.SSHKey, creds.SSHKeyPasswd)
		} else if creds.Type == "auto" {
			client, err = h.sshClient.TryAutoAuth(host, creds.Username)
		} else {
			log.Printf("[HealthCheck] Unknown credential type for %s: %s", device.Name, creds.Type)
			h.updateDeviceStatus(deviceID, models.DeviceStatusError)
			return
		}

		if err != nil {
			log.Printf("[HealthCheck] Device %s is offline: %v", device.Name, err)
			h.updateDeviceStatus(deviceID, models.DeviceStatusOffline)
			return
		}
		log.Printf("[HealthCheck] Created new SSH connection for %s", device.Name)
	} else {
		log.Printf("[HealthCheck] Reusing existing SSH connection for %s", device.Name)
	}

	// Connection successful - device is online
	// Run a simple command to verify SSH is actually working
	session, err := client.NewSession()
	if err != nil {
		log.Printf("[HealthCheck] Device %s SSH session failed: %v", device.Name, err)
		h.updateDeviceStatus(deviceID, models.DeviceStatusError)
		return
	}
	defer session.Close()

	_, err = session.CombinedOutput("echo ping")
	if err != nil {
		log.Printf("[HealthCheck] Device %s SSH command failed: %v", device.Name, err)
		h.updateDeviceStatus(deviceID, models.DeviceStatusError)
		return
	}

	log.Printf("[HealthCheck] Device %s is online", device.Name)
	h.updateDeviceStatus(deviceID, models.DeviceStatusOnline)
}

// updateDeviceStatus updates the device status and last_seen timestamp
func (h *HealthCheckService) updateDeviceStatus(deviceID uuid.UUID, status models.DeviceStatus) {
	now := time.Now()
	if err := h.db.Model(&models.Device{}).Where("id = ?", deviceID).Updates(map[string]interface{}{
		"status":    status,
		"last_seen": &now,
	}).Error; err != nil {
		log.Printf("[HealthCheck] Failed to update device status: %v", err)
	}
}
