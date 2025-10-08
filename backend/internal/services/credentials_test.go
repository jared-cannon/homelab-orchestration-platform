package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/99designs/keyring"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// setupTestCredentialService creates a credential service with file backend for testing
func setupCredService(t *testing.T) *CredentialService {
	// Create a temporary directory for test credentials
	tempDir := filepath.Join(os.TempDir(), "homelab-test-"+uuid.New().String())
	err := os.MkdirAll(tempDir, 0700)
	assert.NoError(t, err, "Failed to create temp directory")

	// Cleanup temp directory when test completes
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	// Set environment variable to force file backend and prevent keychain prompts
	os.Setenv("KEYRING_BACKEND", "file")
	t.Cleanup(func() {
		os.Unsetenv("KEYRING_BACKEND")
	})

	// Create keyring with ONLY file backend - no OS keychain access
	ring, err := keyring.Open(keyring.Config{
		ServiceName: "homelab-test",
		// Force file backend only - never try OS keychain
		AllowedBackends:         []keyring.BackendType{keyring.FileBackend},
		KeychainName:            "", // Disable macOS keychain
		KeychainTrustApplication: false,
		FileDir:                  tempDir,
		FilePasswordFunc: func(prompt string) (string, error) {
			return "test-password-123", nil
		},
	})
	assert.NoError(t, err, "Failed to create test keyring")

	return &CredentialService{ring: ring}
}

func TestCredentialService_PasswordAuth(t *testing.T) {
	credService := setupCredService(t)
	deviceID := uuid.New().String()

	t.Run("Store and retrieve password credentials", func(t *testing.T) {
		creds := &DeviceCredentials{
			Type:     "password",
			Username: "admin",
			Password: "secretpassword123",
		}

		// Store credentials
		err := credService.StoreCredentials(deviceID, creds)
		assert.NoError(t, err, "Should store credentials successfully")

		// Retrieve credentials
		retrieved, err := credService.GetCredentials(deviceID)
		assert.NoError(t, err, "Should retrieve credentials successfully")
		assert.Equal(t, creds.Type, retrieved.Type, "Type should match")
		assert.Equal(t, creds.Username, retrieved.Username, "Username should match")
		assert.Equal(t, creds.Password, retrieved.Password, "Password should match")
	})

	t.Run("Update existing credentials", func(t *testing.T) {
		// Store initial credentials
		creds1 := &DeviceCredentials{
			Type:     "password",
			Username: "admin",
			Password: "oldpassword",
		}
		err := credService.StoreCredentials(deviceID, creds1)
		assert.NoError(t, err)

		// Update with new credentials
		creds2 := &DeviceCredentials{
			Type:     "password",
			Username: "root",
			Password: "newpassword",
		}
		err = credService.StoreCredentials(deviceID, creds2)
		assert.NoError(t, err, "Should update credentials")

		// Retrieve and verify updated credentials
		retrieved, err := credService.GetCredentials(deviceID)
		assert.NoError(t, err)
		assert.Equal(t, creds2.Username, retrieved.Username, "Username should be updated")
		assert.Equal(t, creds2.Password, retrieved.Password, "Password should be updated")
	})

	// Cleanup
	t.Cleanup(func() {
		credService.DeleteCredentials(deviceID)
	})
}

func TestCredentialService_SSHKeyAuth(t *testing.T) {
	credService := setupCredService(t)
	deviceID := uuid.New().String()

	t.Run("Store and retrieve SSH key credentials", func(t *testing.T) {
		creds := &DeviceCredentials{
			Type:         "ssh_key",
			Username:     "root",
			SSHKey:       "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQE\n-----END PRIVATE KEY-----",
			SSHKeyPasswd: "keypassphrase",
		}

		// Store credentials
		err := credService.StoreCredentials(deviceID, creds)
		assert.NoError(t, err, "Should store SSH key credentials")

		// Retrieve credentials
		retrieved, err := credService.GetCredentials(deviceID)
		assert.NoError(t, err, "Should retrieve SSH key credentials")
		assert.Equal(t, creds.Type, retrieved.Type, "Type should match")
		assert.Equal(t, creds.Username, retrieved.Username, "Username should match")
		assert.Equal(t, creds.SSHKey, retrieved.SSHKey, "SSH key should match")
		assert.Equal(t, creds.SSHKeyPasswd, retrieved.SSHKeyPasswd, "SSH key passphrase should match")
	})

	t.Run("Store SSH key without passphrase", func(t *testing.T) {
		creds := &DeviceCredentials{
			Type:     "ssh_key",
			Username: "user",
			SSHKey:   "-----BEGIN PRIVATE KEY-----\ntest-key\n-----END PRIVATE KEY-----",
		}

		err := credService.StoreCredentials(deviceID, creds)
		assert.NoError(t, err)

		retrieved, err := credService.GetCredentials(deviceID)
		assert.NoError(t, err)
		assert.Equal(t, "", retrieved.SSHKeyPasswd, "Passphrase should be empty")
		assert.Equal(t, creds.SSHKey, retrieved.SSHKey, "SSH key should match")
	})

	// Cleanup
	t.Cleanup(func() {
		credService.DeleteCredentials(deviceID)
	})
}

func TestCredentialService_DeleteCredentials(t *testing.T) {
	credService := setupCredService(t)
	deviceID := uuid.New().String()

	t.Run("Delete existing credentials", func(t *testing.T) {
		// Store credentials
		creds := &DeviceCredentials{
			Type:     "password",
			Username: "admin",
			Password: "password",
		}
		err := credService.StoreCredentials(deviceID, creds)
		assert.NoError(t, err)

		// Delete credentials
		err = credService.DeleteCredentials(deviceID)
		assert.NoError(t, err, "Should delete credentials successfully")

		// Verify credentials are deleted
		_, err = credService.GetCredentials(deviceID)
		assert.Error(t, err, "Should return error for deleted credentials")
		assert.Contains(t, err.Error(), "not found", "Error should mention credentials not found")
	})

	t.Run("Delete non-existent credentials", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		err := credService.DeleteCredentials(nonExistentID)
		assert.NoError(t, err, "Deleting non-existent credentials should not error")
	})
}

func TestCredentialService_GetCredentials_Errors(t *testing.T) {
	credService := setupCredService(t)

	t.Run("Get non-existent credentials", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		_, err := credService.GetCredentials(nonExistentID)
		assert.Error(t, err, "Should return error for non-existent credentials")
		assert.Contains(t, err.Error(), "not found", "Error should mention not found")
	})
}

func TestCredentialService_TestCredentials(t *testing.T) {
	credService := setupCredService(t)
	deviceID := uuid.New().String()

	t.Run("Test existing credentials", func(t *testing.T) {
		// Store credentials
		creds := &DeviceCredentials{
			Type:     "password",
			Username: "admin",
			Password: "password",
		}
		err := credService.StoreCredentials(deviceID, creds)
		assert.NoError(t, err)

		// Test credentials exist
		exists, err := credService.TestCredentials(deviceID)
		assert.NoError(t, err, "Should test credentials successfully")
		assert.True(t, exists, "Credentials should exist")
	})

	t.Run("Test non-existent credentials", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		exists, err := credService.TestCredentials(nonExistentID)
		assert.Error(t, err, "Should return error for non-existent credentials")
		assert.False(t, exists, "Credentials should not exist")
	})

	// Cleanup
	t.Cleanup(func() {
		credService.DeleteCredentials(deviceID)
	})
}

func TestCredentialService_MultipleDevices(t *testing.T) {
	credService := setupCredService(t)

	t.Run("Store credentials for multiple devices", func(t *testing.T) {
		// Create credentials for 3 devices
		devices := []struct {
			id    string
			creds *DeviceCredentials
		}{
			{
				id: uuid.New().String(),
				creds: &DeviceCredentials{
					Type:     "password",
					Username: "admin1",
					Password: "pass1",
				},
			},
			{
				id: uuid.New().String(),
				creds: &DeviceCredentials{
					Type:     "password",
					Username: "admin2",
					Password: "pass2",
				},
			},
			{
				id: uuid.New().String(),
				creds: &DeviceCredentials{
					Type:     "ssh_key",
					Username: "root",
					SSHKey:   "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----",
				},
			},
		}

		// Store all credentials
		for _, d := range devices {
			err := credService.StoreCredentials(d.id, d.creds)
			assert.NoError(t, err, "Should store credentials for device %s", d.id)
		}

		// Verify all credentials can be retrieved independently
		for _, d := range devices {
			retrieved, err := credService.GetCredentials(d.id)
			assert.NoError(t, err, "Should retrieve credentials for device %s", d.id)
			assert.Equal(t, d.creds.Username, retrieved.Username, "Username should match for device %s", d.id)
		}

		// Cleanup
		for _, d := range devices {
			credService.DeleteCredentials(d.id)
		}
	})
}
