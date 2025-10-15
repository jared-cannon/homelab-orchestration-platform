package services

import (
	"testing"

	"github.com/99designs/keyring"
	"github.com/google/uuid"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/models"
	"github.com/stretchr/testify/assert"
)

// setupTestCredentialService creates a credential service with file backend for testing
func setupTestCredentialService(t *testing.T) *CredentialService {
	// Set test environment variable (also set in TestMain, but redundant for safety)
	t.Setenv("GO_ENV", "test")

	// Use file backend only for testing to avoid OS keychain dependencies
	ring, err := keyring.Open(keyring.Config{
		ServiceName: "homelab-test",
		AllowedBackends: []keyring.BackendType{
			keyring.FileBackend, // File backend only for tests
		},
		FileDir: t.TempDir(), // Use temporary directory
		FilePasswordFunc: func(prompt string) (string, error) {
			return "test-password", nil
		},
	})
	assert.NoError(t, err, "Failed to open test keyring")

	// Get encryption key (will use test default since GO_ENV=test)
	encKey, err := getEncryptionKey()
	assert.NoError(t, err, "Failed to get encryption key")

	return &CredentialService{
		ring:          ring,
		encryptionKey: encKey,
	}
}

func TestDeviceService_CreateDevice(t *testing.T) {
	db := setupTestDB(t)
	credService := setupTestCredentialService(t)
	// Use nil SSH client for tests
	deviceService := NewDeviceService(db, credService, nil)

	t.Run("Creates device with valid data", func(t *testing.T) {
		device := &models.Device{
			Name:      "Test Server",
			Type:      models.DeviceTypeServer,
			IPAddress: "192.168.1.100",
		}

		creds := &DeviceCredentials{
			Type:     "password",
			Username: "admin",
			Password: "secret",
		}

		err := deviceService.CreateDevice(device, creds)
		assert.NoError(t, err, "Should create device successfully")
		assert.NotEqual(t, uuid.Nil, device.ID, "Should generate UUID")
		assert.Equal(t, device.ID.String(), device.CredentialKey, "Credential key should match device ID")
	})

	t.Run("Rejects invalid IP address", func(t *testing.T) {
		device := &models.Device{
			Name:      "Invalid Device",
			Type:      models.DeviceTypeServer,
			IPAddress: "not-an-ip",
		}

		creds := &DeviceCredentials{
			Type:     "password",
			Username: "admin",
			Password: "secret",
		}

		err := deviceService.CreateDevice(device, creds)
		assert.Error(t, err, "Should reject invalid IP")
		assert.Contains(t, err.Error(), "invalid IP address", "Error should mention invalid IP")
	})

	t.Run("Rejects duplicate IP address", func(t *testing.T) {
		// Create first device
		device1 := &models.Device{
			Name:      "Server 1",
			Type:      models.DeviceTypeServer,
			IPAddress: "192.168.1.101",
		}
		creds := &DeviceCredentials{
			Type:     "password",
			Username: "admin",
			Password: "secret",
		}
		err := deviceService.CreateDevice(device1, creds)
		assert.NoError(t, err)

		// Attempt to create second device with same IP
		device2 := &models.Device{
			Name:      "Server 2",
			Type:      models.DeviceTypeServer,
			IPAddress: "192.168.1.101", // Same IP
		}
		err = deviceService.CreateDevice(device2, creds)
		assert.Error(t, err, "Should reject duplicate IP")
		assert.Contains(t, err.Error(), "already exists", "Error should mention duplicate")
	})

	t.Run("Stores and retrieves credentials", func(t *testing.T) {
		device := &models.Device{
			Name:      "Credential Test",
			Type:      models.DeviceTypeServer,
			IPAddress: "192.168.1.102",
		}

		creds := &DeviceCredentials{
			Type:     "ssh_key",
			Username: "root",
			SSHKey:   "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----",
		}

		err := deviceService.CreateDevice(device, creds)
		assert.NoError(t, err)

		// Retrieve credentials
		retrievedCreds, err := deviceService.GetDeviceCredentials(device.ID)
		assert.NoError(t, err, "Should retrieve credentials")
		assert.Equal(t, creds.Type, retrievedCreds.Type)
		assert.Equal(t, creds.Username, retrievedCreds.Username)
		assert.Equal(t, creds.SSHKey, retrievedCreds.SSHKey)
	})

	t.Run("Creates device with auto authentication type", func(t *testing.T) {
		device := &models.Device{
			Name:      "Auto Auth Test",
			Type:      models.DeviceTypeServer,
			IPAddress: "192.168.1.108",
		}

		creds := &DeviceCredentials{
			Type:     "auto",
			Username: "admin",
		}

		err := deviceService.CreateDevice(device, creds)
		assert.NoError(t, err, "Should create device with auto auth")
		assert.NotEqual(t, uuid.Nil, device.ID, "Should generate UUID")

		// Retrieve credentials
		retrievedCreds, err := deviceService.GetDeviceCredentials(device.ID)
		assert.NoError(t, err, "Should retrieve credentials")
		assert.Equal(t, "auto", retrievedCreds.Type)
		assert.Equal(t, "admin", retrievedCreds.Username)
		assert.Empty(t, retrievedCreds.Password, "Password should be empty for auto auth")
		assert.Empty(t, retrievedCreds.SSHKey, "SSH key should be empty for auto auth")
	})
}

func TestDeviceService_GetDevice(t *testing.T) {
	db := setupTestDB(t)
	credService := setupTestCredentialService(t)
	// Use nil SSH client for tests
	deviceService := NewDeviceService(db, credService, nil)

	t.Run("Returns device by ID", func(t *testing.T) {
		// Create device
		device := &models.Device{
			Name:      "Get Test",
			Type:      models.DeviceTypeNAS,
			IPAddress: "192.168.1.103",
		}
		creds := &DeviceCredentials{
			Type:     "password",
			Username: "admin",
			Password: "secret",
		}
		err := deviceService.CreateDevice(device, creds)
		assert.NoError(t, err)

		// Retrieve device
		retrieved, err := deviceService.GetDevice(device.ID)
		assert.NoError(t, err)
		assert.Equal(t, device.Name, retrieved.Name)
		assert.Equal(t, device.Type, retrieved.Type)
		assert.Equal(t, device.IPAddress, retrieved.IPAddress)
	})

	t.Run("Returns error for non-existent device", func(t *testing.T) {
		randomID := uuid.New()
		_, err := deviceService.GetDevice(randomID)
		assert.Error(t, err, "Should return error for non-existent device")
		assert.Contains(t, err.Error(), "not found", "Error should mention not found")
	})
}

func TestDeviceService_ListDevices(t *testing.T) {
	db := setupTestDB(t)
	credService := setupTestCredentialService(t)
	// Use nil SSH client for tests
	deviceService := NewDeviceService(db, credService, nil)

	t.Run("Returns empty list when no devices", func(t *testing.T) {
		devices, err := deviceService.ListDevices()
		assert.NoError(t, err)
		assert.Empty(t, devices, "Should return empty list")
	})

	t.Run("Returns all devices", func(t *testing.T) {
		// Create multiple devices
		creds := &DeviceCredentials{
			Type:     "password",
			Username: "admin",
			Password: "secret",
		}

		device1 := &models.Device{
			Name:      "Device 1",
			Type:      models.DeviceTypeServer,
			IPAddress: "192.168.1.104",
		}
		device2 := &models.Device{
			Name:      "Device 2",
			Type:      models.DeviceTypeRouter,
			IPAddress: "192.168.1.105",
		}

		err := deviceService.CreateDevice(device1, creds)
		assert.NoError(t, err)
		err = deviceService.CreateDevice(device2, creds)
		assert.NoError(t, err)

		// List devices
		devices, err := deviceService.ListDevices()
		assert.NoError(t, err)
		assert.Len(t, devices, 2, "Should return both devices")
	})
}

func TestDeviceService_UpdateDevice(t *testing.T) {
	db := setupTestDB(t)
	credService := setupTestCredentialService(t)
	// Use nil SSH client for tests
	deviceService := NewDeviceService(db, credService, nil)

	t.Run("Updates device fields", func(t *testing.T) {
		// Create device
		device := &models.Device{
			Name:      "Original Name",
			Type:      models.DeviceTypeServer,
			IPAddress: "192.168.1.106",
		}
		creds := &DeviceCredentials{
			Type:     "password",
			Username: "admin",
			Password: "secret",
		}
		err := deviceService.CreateDevice(device, creds)
		assert.NoError(t, err)

		// Update device
		updates := map[string]interface{}{
			"name": "Updated Name",
		}
		err = deviceService.UpdateDevice(device.ID, updates)
		assert.NoError(t, err)

		// Verify update
		retrieved, err := deviceService.GetDevice(device.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Name", retrieved.Name)
	})
}

func TestDeviceService_DeleteDevice(t *testing.T) {
	db := setupTestDB(t)
	credService := setupTestCredentialService(t)
	// Use nil SSH client for tests
	deviceService := NewDeviceService(db, credService, nil)

	t.Run("Deletes device and credentials", func(t *testing.T) {
		// Create device
		device := &models.Device{
			Name:      "To Delete",
			Type:      models.DeviceTypeServer,
			IPAddress: "192.168.1.107",
		}
		creds := &DeviceCredentials{
			Type:     "password",
			Username: "admin",
			Password: "secret",
		}
		err := deviceService.CreateDevice(device, creds)
		assert.NoError(t, err)

		// Delete device
		err = deviceService.DeleteDevice(device.ID)
		assert.NoError(t, err)

		// Verify deletion
		_, err = deviceService.GetDevice(device.ID)
		assert.Error(t, err, "Device should no longer exist")

		// Verify credentials deleted
		_, err = deviceService.GetDeviceCredentials(device.ID)
		assert.Error(t, err, "Credentials should no longer exist")
	})
}

func TestDeviceService_CreateDevice_TailscaleHostnames(t *testing.T) {
	db := setupTestDB(t)
	credService := setupTestCredentialService(t)
	deviceService := NewDeviceService(db, credService, nil)

	t.Run("Accepts Tailscale hostname - standard format", func(t *testing.T) {
		device := &models.Device{
			Name:      "Tailscale Device",
			Type:      models.DeviceTypeServer,
			IPAddress: "myserver.wolf-bear.ts.net",
		}

		creds := &DeviceCredentials{
			Type:     "tailscale",
			Username: "admin",
		}

		err := deviceService.CreateDevice(device, creds)
		assert.NoError(t, err, "Should accept Tailscale hostname format")
		assert.NotEqual(t, uuid.Nil, device.ID, "Should generate UUID")
		assert.Equal(t, models.AuthTypeTailscale, device.AuthType, "Auth type should be tailscale")
		assert.Empty(t, device.CredentialKey, "Tailscale devices should not have credential key")
	})

	t.Run("Accepts Tailscale hostname - different tailnet names", func(t *testing.T) {
		device := &models.Device{
			Name:      "Another Tailscale Device",
			Type:      models.DeviceTypeServer,
			IPAddress: "api.red-blue.ts.net",
		}

		creds := &DeviceCredentials{
			Type:     "tailscale",
			Username: "root",
		}

		err := deviceService.CreateDevice(device, creds)
		assert.NoError(t, err, "Should accept different tailnet format")
	})

	t.Run("Accepts Tailscale IP address", func(t *testing.T) {
		device := &models.Device{
			Name:      "Tailscale IP Device",
			Type:      models.DeviceTypeServer,
			IPAddress: "100.64.1.5",
		}

		creds := &DeviceCredentials{
			Type:     "tailscale",
			Username: "admin",
		}

		err := deviceService.CreateDevice(device, creds)
		assert.NoError(t, err, "Should accept Tailscale IP address")
	})

	t.Run("Rejects invalid hostname for Tailscale", func(t *testing.T) {
		device := &models.Device{
			Name:      "Invalid Tailscale",
			Type:      models.DeviceTypeServer,
			IPAddress: "invalid..hostname",
		}

		creds := &DeviceCredentials{
			Type:     "tailscale",
			Username: "admin",
		}

		err := deviceService.CreateDevice(device, creds)
		assert.Error(t, err, "Should reject invalid hostname")
		assert.Contains(t, err.Error(), "invalid hostname", "Error should mention invalid hostname")
	})

	t.Run("Rejects hostname for non-Tailscale auth types", func(t *testing.T) {
		device := &models.Device{
			Name:      "Regular Device with Hostname",
			Type:      models.DeviceTypeServer,
			IPAddress: "myserver.local",
		}

		creds := &DeviceCredentials{
			Type:     "password",
			Username: "admin",
			Password: "secret",
		}

		err := deviceService.CreateDevice(device, creds)
		assert.Error(t, err, "Should reject hostname for non-Tailscale auth")
		assert.Contains(t, err.Error(), "invalid IP address", "Error should mention invalid IP")
	})

	t.Run("Retrieves Tailscale credentials correctly", func(t *testing.T) {
		device := &models.Device{
			Name:      "Tailscale Cred Test",
			Type:      models.DeviceTypeServer,
			IPAddress: "machine.happy-sunny.ts.net",
		}

		creds := &DeviceCredentials{
			Type:     "tailscale",
			Username: "user123",
		}

		err := deviceService.CreateDevice(device, creds)
		assert.NoError(t, err)

		// Retrieve credentials
		retrievedCreds, err := deviceService.GetDeviceCredentials(device.ID)
		assert.NoError(t, err, "Should retrieve Tailscale credentials")
		assert.Equal(t, "tailscale", retrievedCreds.Type)
		assert.Equal(t, "user123", retrievedCreds.Username)
		assert.Empty(t, retrievedCreds.Password, "Password should be empty for Tailscale")
		assert.Empty(t, retrievedCreds.SSHKey, "SSH key should be empty for Tailscale")
	})
}
