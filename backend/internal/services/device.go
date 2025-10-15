package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/models"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/ssh"
	"gorm.io/gorm"
)

// DeviceService handles device management operations
type DeviceService struct {
	db          *gorm.DB
	credService *CredentialService
	sshClient   *ssh.Client
	validator   *ValidatorService
}

// NewDeviceService creates a new device service
func NewDeviceService(db *gorm.DB, credService *CredentialService, sshClient *ssh.Client) *DeviceService {
	return &DeviceService{
		db:          db,
		credService: credService,
		sshClient:   sshClient,
		validator:   NewValidatorService(sshClient),
	}
}

// CreateDevice creates a new device and stores its credentials
func (s *DeviceService) CreateDevice(device *models.Device, creds *DeviceCredentials) error {
	// Validate IP address or hostname
	// For Tailscale, allow hostnames (e.g., "machine.tail-scale.ts.net")
	// For other auth types, require IP addresses for consistency
	if creds.Type == "tailscale" {
		if !ValidateHostname(device.IPAddress) {
			return fmt.Errorf("invalid hostname or IP address: %s", device.IPAddress)
		}
	} else {
		if !ValidateIPAddress(device.IPAddress) {
			return fmt.Errorf("invalid IP address: %s", device.IPAddress)
		}
	}

	// Check if device with this IP already exists
	var existing models.Device
	if err := s.db.Where("ip_address = ?", device.IPAddress).First(&existing).Error; err == nil {
		return fmt.Errorf("device with IP %s already exists", device.IPAddress)
	}

	// Generate UUID for device
	if device.ID == uuid.Nil {
		device.ID = uuid.New()
	}

	// Set username and auth type from credentials (stored in DB, not sensitive)
	device.Username = creds.Username
	device.AuthType = models.AuthType(creds.Type)

	// Store secrets in keychain (only for password/ssh_key types)
	// For "auto" and "tailscale" types, this is a no-op since no secrets to store
	if err := s.credService.StoreCredentials(device.ID.String(), creds, device.Name, device.IPAddress); err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	// Set credential key reference (only used for password/ssh_key types)
	// "auto" and "tailscale" types don't use keychain, so no credential key needed
	if creds.Type == "password" || creds.Type == "ssh_key" {
		device.CredentialKey = device.ID.String()
	}

	// Create device in database
	if err := s.db.Create(device).Error; err != nil {
		// Cleanup credentials if database insert fails
		s.credService.DeleteCredentials(device.ID.String())
		return fmt.Errorf("failed to create device: %w", err)
	}

	return nil
}

// GetDevice retrieves a device by ID
func (s *DeviceService) GetDevice(id uuid.UUID) (*models.Device, error) {
	var device models.Device
	if err := s.db.First(&device, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("device not found")
		}
		return nil, err
	}
	return &device, nil
}

// ListDevices retrieves all devices
func (s *DeviceService) ListDevices() ([]models.Device, error) {
	var devices []models.Device
	if err := s.db.Order("created_at desc").Find(&devices).Error; err != nil {
		return nil, err
	}
	return devices, nil
}

// UpdateDevice updates a device
func (s *DeviceService) UpdateDevice(id uuid.UUID, updates map[string]interface{}) error {
	// If updating IP address, validate and check for conflicts
	if newIP, ok := updates["ip_address"]; ok {
		ipStr, ok := newIP.(string)
		if !ok {
			return fmt.Errorf("invalid IP address format")
		}

		// Get the current device to check auth type and old IP
		device, err := s.GetDevice(id)
		if err != nil {
			return fmt.Errorf("device not found")
		}

		// Validate IP address or hostname based on auth type
		if device.AuthType == models.AuthTypeTailscale {
			if !ValidateHostname(ipStr) {
				return fmt.Errorf("invalid hostname or IP address: %s", ipStr)
			}
		} else {
			if !ValidateIPAddress(ipStr) {
				return fmt.Errorf("invalid IP address: %s", ipStr)
			}
		}

		// Check if another device already has this IP (excluding current device)
		var existing models.Device
		if err := s.db.Where("ip_address = ? AND id != ?", ipStr, id).First(&existing).Error; err == nil {
			return fmt.Errorf("device with IP %s already exists", ipStr)
		}

		// Close existing SSH connection if IP is changing
		if s.sshClient != nil && device.IPAddress != ipStr {
			oldHost := device.IPAddress + ":22"
			if err := s.sshClient.Close(oldHost); err == nil {
				fmt.Printf("[DeviceService] Closed SSH connection to old IP %s after IP update\n", device.IPAddress)
			}
		}
	}

	if err := s.db.Model(&models.Device{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update device: %w", err)
	}
	return nil
}

// DeleteDevice deletes a device and its credentials
func (s *DeviceService) DeleteDevice(id uuid.UUID) error {
	// Delete credentials from keychain
	if err := s.credService.DeleteCredentials(id.String()); err != nil {
		// Log error but continue with device deletion
		fmt.Printf("Warning: failed to delete credentials: %v\n", err)
	}

	// Close SSH connection if any
	if s.sshClient != nil {
		device, err := s.GetDevice(id)
		if err == nil {
			s.sshClient.Close(device.IPAddress + ":22")
		}
	}

	// Delete device from database
	if err := s.db.Delete(&models.Device{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete device: %w", err)
	}

	return nil
}

// TestConnectionWithCredentials tests SSH connection with provided credentials (no device required)
func (s *DeviceService) TestConnectionWithCredentials(ipAddress string, creds *DeviceCredentials) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Validate IP address or hostname
	// For Tailscale, allow hostnames (e.g., "machine.tail-scale.ts.net")
	// For other auth types, require IP addresses for consistency
	if creds.Type == "tailscale" {
		if !ValidateHostname(ipAddress) {
			return result, fmt.Errorf("invalid hostname or IP address: %s", ipAddress)
		}
	} else {
		if !ValidateIPAddress(ipAddress) {
			return result, fmt.Errorf("invalid IP address: %s", ipAddress)
		}
	}

	// Establish SSH connection
	host := ipAddress + ":22"

	var err error
	if creds.Type == "password" {
		_, err = s.sshClient.ConnectWithPassword(host, creds.Username, creds.Password)
	} else if creds.Type == "ssh_key" {
		_, err = s.sshClient.ConnectWithKey(host, creds.Username, creds.SSHKey, creds.SSHKeyPasswd)
	} else if creds.Type == "auto" {
		_, err = s.sshClient.TryAutoAuth(host, creds.Username)
	} else if creds.Type == "tailscale" {
		_, err = s.sshClient.ConnectWithTailscale(host, creds.Username)
	} else {
		return result, fmt.Errorf("unknown credential type: %s", creds.Type)
	}

	if err != nil {
		result["ssh_connection"] = false
		result["error"] = err.Error()
		return result, fmt.Errorf("connection failed: %w", err)
	}

	result["ssh_connection"] = true

	// Check Docker installation
	dockerInstalled, dockerVersion, err := s.validator.DockerInstalled(host)
	result["docker_installed"] = dockerInstalled
	if dockerInstalled {
		result["docker_version"] = dockerVersion
	}

	// Check Docker running
	if dockerInstalled {
		dockerRunning, err := s.validator.DockerRunning(host)
		result["docker_running"] = dockerRunning
		if err != nil {
			result["docker_error"] = err.Error()
		}
	}

	// Check Docker Compose
	composeInstalled, composeVersion, _ := s.validator.ValidateDockerCompose(host)
	result["docker_compose_installed"] = composeInstalled
	if composeInstalled {
		result["docker_compose_version"] = composeVersion
	}

	// Get system info
	sysInfo, err := s.validator.GetSystemInfo(host)
	if err == nil {
		result["system_info"] = sysInfo
	}

	return result, nil
}

// TestConnection tests SSH connection and Docker availability
func (s *DeviceService) TestConnection(id uuid.UUID) (map[string]interface{}, error) {
	device, err := s.GetDevice(id)
	if err != nil {
		return nil, err
	}

	// Get credentials (handles both DB and keychain sources)
	creds, err := s.GetDeviceCredentials(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	// Use TestConnectionWithCredentials for the actual test
	result, err := s.TestConnectionWithCredentials(device.IPAddress, creds)
	if err != nil {
		return result, err
	}

	// Update device status
	s.UpdateDeviceStatus(id, models.DeviceStatusOnline)

	return result, nil
}

// UpdateDeviceStatus updates the status and last_seen timestamp of a device
func (s *DeviceService) UpdateDeviceStatus(id uuid.UUID, status models.DeviceStatus) error {
	now := time.Now()
	return s.db.Model(&models.Device{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":    status,
		"last_seen": &now,
	}).Error
}

// GetDeviceCredentials retrieves credentials for a device
func (s *DeviceService) GetDeviceCredentials(id uuid.UUID) (*DeviceCredentials, error) {
	// Get device to check auth type
	device, err := s.GetDevice(id)
	if err != nil {
		return nil, err
	}

	// Migration path: if username is empty, this is a legacy device - try to get from keychain and migrate
	if device.Username == "" {
		fmt.Printf("[DeviceService] Migrating legacy device %s - retrieving credentials from keychain\n", device.Name)
		creds, err := s.credService.GetCredentials(id.String())
		if err != nil {
			return nil, fmt.Errorf("failed to migrate credentials: %w", err)
		}

		// Update device with username and auth_type for future use
		updates := map[string]interface{}{
			"username":  creds.Username,
			"auth_type": models.AuthType(creds.Type),
		}
		if creds.Type == "password" || creds.Type == "ssh_key" {
			updates["credential_key"] = id.String()
		}

		if err := s.db.Model(&models.Device{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			fmt.Printf("[DeviceService] Warning: failed to migrate device metadata: %v\n", err)
		} else {
			fmt.Printf("[DeviceService] Successfully migrated device %s to new credential system\n", device.Name)
		}

		return creds, nil
	}

	// For "auto" and "tailscale" types, construct credentials from device table
	// These types don't require keychain storage
	if device.AuthType == models.AuthTypeAuto {
		return &DeviceCredentials{
			Type:     "auto",
			Username: device.Username,
		}, nil
	}
	if device.AuthType == models.AuthTypeTailscale {
		return &DeviceCredentials{
			Type:     "tailscale",
			Username: device.Username,
		}, nil
	}

	// For password/ssh_key types, retrieve from keychain
	return s.credService.GetCredentials(id.String())
}

// UpdateDeviceCredentials updates credentials for an existing device
func (s *DeviceService) UpdateDeviceCredentials(id uuid.UUID, creds *DeviceCredentials) error {
	// Verify device exists
	device, err := s.GetDevice(id)
	if err != nil {
		return err
	}

	// Close existing SSH connection to force reconnection with new credentials
	if s.sshClient != nil {
		host := device.IPAddress + ":22"
		if err := s.sshClient.Close(host); err == nil {
			fmt.Printf("[DeviceService] Closed existing SSH connection to %s after credential update\n", device.Name)
		}
	}

	// Update username and auth type in device table
	updates := map[string]interface{}{
		"username":  creds.Username,
		"auth_type": models.AuthType(creds.Type),
	}

	// Update credential key reference (only for password/ssh_key types)
	if creds.Type == "password" || creds.Type == "ssh_key" {
		updates["credential_key"] = id.String()
	} else {
		updates["credential_key"] = "" // Clear for "auto" and "tailscale" types
	}

	if err := s.db.Model(&models.Device{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update device credentials metadata: %w", err)
	}

	// Update secrets in keychain (only for password/ssh_key types)
	// For "auto" type, this is a no-op
	if err := s.credService.StoreCredentials(id.String(), creds, device.Name, device.IPAddress); err != nil {
		return fmt.Errorf("failed to update credentials: %w", err)
	}

	return nil
}
