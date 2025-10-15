package services

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MockSSHClient is a mock SSH client for testing
type MockSSHClient struct {
	// Map of host+command -> (output, error)
	responses map[string]struct {
		output string
		err    error
	}
}

func NewMockSSHClient() *MockSSHClient {
	return &MockSSHClient{
		responses: make(map[string]struct {
			output string
			err    error
		}),
	}
}

// SetResponse sets the response for a specific command on a specific host
func (m *MockSSHClient) SetResponse(host, command, output string, err error) {
	key := fmt.Sprintf("%s:%s", host, command)
	m.responses[key] = struct {
		output string
		err    error
	}{output: output, err: err}
}

func (m *MockSSHClient) Execute(host, command string) (string, error) {
	key := fmt.Sprintf("%s:%s", host, command)
	if resp, ok := m.responses[key]; ok {
		return resp.output, resp.err
	}
	// Default: return error (not installed)
	return "", fmt.Errorf("command failed")
}

func (m *MockSSHClient) ExecuteWithTimeout(host, command string, timeout time.Duration) (string, error) {
	return m.Execute(host, command)
}

// setupSoftwareTestDB creates an in-memory SQLite database for testing
func setupSoftwareTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err, "Failed to open in-memory database")

	// Run migrations
	err = db.AutoMigrate(&models.Device{}, &models.InstalledSoftware{}, &models.SoftwareInstallation{})
	assert.NoError(t, err, "Failed to run migrations")

	return db
}

func TestSoftwareService_DetectInstalled(t *testing.T) {
	t.Run("Adds newly detected software to database", func(t *testing.T) {
		db := setupSoftwareTestDB(t)
		mockSSH := NewMockSSHClient()
		registry := NewSoftwareRegistry("../../software-definitions")

		// Create service with mock
		service := &SoftwareService{
			db:        db,
			sshClient: mockSSH,
			registry:  registry,
		}

		// Create test device
		device := &models.Device{
			ID:        uuid.New(),
			Name:      "Test Server",
			IPAddress: "192.168.1.100",
		}
		db.Create(device)

		host := device.IPAddress + ":22"

		// Mock: Docker is installed
		mockSSH.SetResponse(host, "docker --version", "Docker version 24.0.0, build abc123", nil)
		// Mock: NFS is not installed (will return error)

		// Run detection
		detected, err := service.DetectInstalled(device.ID)
		assert.NoError(t, err)

		// Should detect Docker
		assert.Len(t, detected, 1)
		assert.Equal(t, models.SoftwareDocker, detected[0].Name)
		assert.Contains(t, detected[0].Version, "24.0.0")

		// Verify it was saved to database
		var dbSoftware []models.InstalledSoftware
		db.Where("device_id = ?", device.ID).Find(&dbSoftware)
		assert.Len(t, dbSoftware, 1)
		assert.Equal(t, models.SoftwareDocker, dbSoftware[0].Name)
	})

	t.Run("Removes uninstalled software from database", func(t *testing.T) {
		db := setupSoftwareTestDB(t)
		mockSSH := NewMockSSHClient()
		registry := NewSoftwareRegistry("../../software-definitions")

		service := &SoftwareService{
			db:        db,
			sshClient: mockSSH,
			registry:  registry,
		}

		// Create test device
		device := &models.Device{
			ID:        uuid.New(),
			Name:      "Test Server",
			IPAddress: "192.168.1.101",
		}
		db.Create(device)

		// Manually add Docker as installed in database
		installedDocker := &models.InstalledSoftware{
			DeviceID:    device.ID,
			Name:        models.SoftwareDocker,
			Version:     "Docker version 24.0.0",
			InstalledBy: "system",
		}
		db.Create(installedDocker)

		// Verify it's in database
		var beforeDetection []models.InstalledSoftware
		db.Where("device_id = ?", device.ID).Find(&beforeDetection)
		assert.Len(t, beforeDetection, 1, "Docker should be in database before detection")

		// Mock: Docker is NOT installed (return error)
		// NFS is also not installed
		// (No SetResponse calls means all commands return error)

		// Run detection
		detected, err := service.DetectInstalled(device.ID)
		assert.NoError(t, err)

		// Should detect nothing
		assert.Len(t, detected, 0, "Should detect no installed software")

		// Verify Docker was removed from database
		var afterDetection []models.InstalledSoftware
		db.Where("device_id = ?", device.ID).Find(&afterDetection)
		assert.Len(t, afterDetection, 0, "Docker should be removed from database")
	})

	t.Run("Updates version when software version changes", func(t *testing.T) {
		db := setupSoftwareTestDB(t)
		mockSSH := NewMockSSHClient()
		registry := NewSoftwareRegistry("../../software-definitions")

		service := &SoftwareService{
			db:        db,
			sshClient: mockSSH,
			registry:  registry,
		}

		// Create test device
		device := &models.Device{
			ID:        uuid.New(),
			Name:      "Test Server",
			IPAddress: "192.168.1.102",
		}
		db.Create(device)

		// Add Docker with old version
		oldDocker := &models.InstalledSoftware{
			DeviceID:    device.ID,
			Name:        models.SoftwareDocker,
			Version:     "Docker version 23.0.0, build old123",
			InstalledBy: "system",
		}
		db.Create(oldDocker)

		host := device.IPAddress + ":22"

		// Mock: Docker is installed with new version
		mockSSH.SetResponse(host, "docker --version", "Docker version 24.0.0, build new456", nil)

		// Run detection
		detected, err := service.DetectInstalled(device.ID)
		assert.NoError(t, err)

		// Should still detect Docker
		assert.Len(t, detected, 1)
		assert.Equal(t, models.SoftwareDocker, detected[0].Name)

		// Version should be updated
		assert.Contains(t, detected[0].Version, "24.0.0", "Version should be updated to new version")
		assert.NotContains(t, detected[0].Version, "23.0.0", "Old version should not be present")

		// Verify database was updated
		var dbSoftware models.InstalledSoftware
		db.Where("device_id = ? AND name = ?", device.ID, models.SoftwareDocker).First(&dbSoftware)
		assert.Contains(t, dbSoftware.Version, "24.0.0")
	})

	t.Run("Handles mixed scenarios: add, remove, and update", func(t *testing.T) {
		db := setupSoftwareTestDB(t)
		mockSSH := NewMockSSHClient()
		registry := NewSoftwareRegistry("../../software-definitions")

		service := &SoftwareService{
			db:        db,
			sshClient: mockSSH,
			registry:  registry,
		}

		// Create test device
		device := &models.Device{
			ID:        uuid.New(),
			Name:      "Test Server",
			IPAddress: "192.168.1.103",
		}
		db.Create(device)

		// Start with Docker (will be removed) and NFS Server (will be updated)
		db.Create(&models.InstalledSoftware{
			DeviceID:    device.ID,
			Name:        models.SoftwareDocker,
			Version:     "Docker version 24.0.0",
			InstalledBy: "system",
		})
		db.Create(&models.InstalledSoftware{
			DeviceID:    device.ID,
			Name:        models.SoftwareNFSServer,
			Version:     "1.0.0",
			InstalledBy: "system",
		})

		host := device.IPAddress + ":22"

		// Mock responses:
		// - Docker is NOT installed (no response = error)
		// - NFS Server is installed with updated version
		mockSSH.SetResponse(host, "systemctl is-active nfs-kernel-server", "active", nil)
		// - NFS Client is newly installed
		mockSSH.SetResponse(host, "dpkg -l | grep nfs-common", "ii  nfs-common  1:2.6.1-1ubuntu1", nil)

		// Run detection
		detected, err := service.DetectInstalled(device.ID)
		assert.NoError(t, err)

		// Should detect NFS Server and NFS Client (Docker removed)
		assert.Len(t, detected, 2)

		// Find each software
		var foundNFSServer, foundNFSClient bool
		for _, sw := range detected {
			if sw.Name == models.SoftwareNFSServer {
				foundNFSServer = true
			}
			if sw.Name == models.SoftwareNFSClient {
				foundNFSClient = true
			}
		}

		assert.True(t, foundNFSServer, "NFS Server should still be detected")
		assert.True(t, foundNFSClient, "NFS Client should be newly detected")

		// Verify database state
		var dbSoftware []models.InstalledSoftware
		db.Where("device_id = ?", device.ID).Find(&dbSoftware)
		assert.Len(t, dbSoftware, 2, "Should have exactly 2 software entries")

		// Docker should be gone
		var dockerCount int64
		db.Model(&models.InstalledSoftware{}).Where("device_id = ? AND name = ?", device.ID, models.SoftwareDocker).Count(&dockerCount)
		assert.Equal(t, int64(0), dockerCount, "Docker should be removed from database")
	})

	t.Run("Returns empty list when no software installed", func(t *testing.T) {
		db := setupSoftwareTestDB(t)
		mockSSH := NewMockSSHClient()
		registry := NewSoftwareRegistry("../../software-definitions")

		service := &SoftwareService{
			db:        db,
			sshClient: mockSSH,
			registry:  registry,
		}

		// Create test device
		device := &models.Device{
			ID:        uuid.New(),
			Name:      "Bare Server",
			IPAddress: "192.168.1.104",
		}
		db.Create(device)

		// No mock responses = nothing installed

		// Run detection
		detected, err := service.DetectInstalled(device.ID)
		assert.NoError(t, err)

		// Should detect nothing
		assert.Len(t, detected, 0)

		// Database should also be empty
		var dbSoftware []models.InstalledSoftware
		db.Where("device_id = ?", device.ID).Find(&dbSoftware)
		assert.Len(t, dbSoftware, 0)
	})
}

func TestSoftwareService_IsInstalled(t *testing.T) {
	registry := NewSoftwareRegistry("../../software-definitions")
	host := "192.168.1.100:22"

	t.Run("Detects Docker when installed", func(t *testing.T) {
		mockSSH := NewMockSSHClient()
		service := &SoftwareService{
			db:        nil,
			sshClient: mockSSH,
			registry:  registry,
		}
		mockSSH.SetResponse(host, "docker --version", "Docker version 24.0.0, build abc123", nil)

		installed, version, err := service.IsInstalled(host, models.SoftwareDocker)
		assert.NoError(t, err)
		assert.True(t, installed)
		assert.Contains(t, version, "24.0.0")
	})

	t.Run("Detects software as not installed when command fails", func(t *testing.T) {
		mockSSH := NewMockSSHClient()
		service := &SoftwareService{
			db:        nil,
			sshClient: mockSSH,
			registry:  registry,
		}
		// No mock response = command will fail

		installed, version, err := service.IsInstalled(host, models.SoftwareDocker)
		assert.NoError(t, err, "Should not return error for uninstalled software")
		assert.False(t, installed, "Should detect as not installed")
		assert.Empty(t, version)
	})

	t.Run("Detects NFS Server when active", func(t *testing.T) {
		mockSSH := NewMockSSHClient()
		service := &SoftwareService{
			db:        nil,
			sshClient: mockSSH,
			registry:  registry,
		}
		mockSSH.SetResponse(host, "systemctl is-active nfs-kernel-server", "active", nil)

		installed, version, err := service.IsInstalled(host, models.SoftwareNFSServer)
		assert.NoError(t, err)
		assert.True(t, installed)
		assert.Equal(t, "active", version)
	})
}
