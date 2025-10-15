package services

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// MockWSHub is a mock WebSocket hub for testing
type MockWSHub struct {
	broadcasts []struct {
		channel string
		event   string
		data    interface{}
	}
}

func (m *MockWSHub) Broadcast(channel, event string, data interface{}) {
	m.broadcasts = append(m.broadcasts, struct {
		channel string
		event   string
		data    interface{}
	}{
		channel: channel,
		event:   event,
		data:    data,
	})
}

// MockPanicSSHClient is a mock SSH client that panics on certain commands
type MockPanicSSHClient struct {
	*MockSSHClient
	panicOnCommand string
}

func (m *MockPanicSSHClient) Execute(host, command string) (string, error) {
	if command == m.panicOnCommand {
		panic("simulated panic during execution")
	}
	return m.MockSSHClient.Execute(host, command)
}

func (m *MockPanicSSHClient) ExecuteWithTimeout(host, command string, timeout time.Duration) (string, error) {
	if command == m.panicOnCommand {
		panic("simulated panic during execution")
	}
	return m.MockSSHClient.ExecuteWithTimeout(host, command, timeout)
}

// MockFailingDB wraps gorm.DB and makes Create() fail
type MockFailingDB struct {
	*gorm.DB
	failCreate bool
}

func (m *MockFailingDB) Create(value interface{}) *gorm.DB {
	if m.failCreate {
		return &gorm.DB{Error: fmt.Errorf("mock database error")}
	}
	return m.DB.Create(value)
}

func TestSoftwareService_InstallationPanicRecovery(t *testing.T) {
	t.Run("Recovers from panic during installation and marks as failed", func(t *testing.T) {
		db := setupSoftwareTestDB(t)
		mockWSHub := &MockWSHub{}
		mockSSH := &MockPanicSSHClient{
			MockSSHClient:  NewMockSSHClient(),
			panicOnCommand: "test panic command",
		}

		// Create a minimal software definition that will trigger the panic
		registry := &SoftwareRegistry{
			definitions: map[string]*models.SoftwareDefinition{
				"docker": {
					Name: "Test Software",
					Commands: models.SoftwareCommands{
						Install: "test panic command",
					},
				},
			},
		}

		service := &SoftwareService{
			db:        db,
			sshClient: mockSSH,
			registry:  registry,
			wsHub:     mockWSHub,
		}

		// Create test device
		device := &models.Device{
			ID:        uuid.New(),
			Name:      "Test Server",
			IPAddress: "192.168.1.100",
		}
		db.Create(device)

		// Create installation
		installation := &models.SoftwareInstallation{
			DeviceID:     device.ID,
			SoftwareName: "docker",
			Status:       models.InstallationStatusPending,
		}
		db.Create(installation)

		def := registry.definitions["docker"]

		// Execute installation in a way that allows the panic to be recovered
		service.executeInstallation(installation, device, def, nil)

		// Give it a moment to process
		time.Sleep(100 * time.Millisecond)

		// Reload installation from database
		var updated models.SoftwareInstallation
		db.First(&updated, "id = ?", installation.ID)

		// Should have marked as failed
		assert.Equal(t, models.InstallationStatusFailed, updated.Status)
		assert.Contains(t, updated.ErrorDetails, "panic")
		assert.Contains(t, updated.InstallLogs, "Critical error")

		// Should have CompletedAt set
		assert.NotNil(t, updated.CompletedAt)
	})
}

func TestSoftwareService_InstallationCompletedAt(t *testing.T) {
	t.Run("Sets CompletedAt on success", func(t *testing.T) {
		db := setupSoftwareTestDB(t)
		mockWSHub := &MockWSHub{}
		mockSSH := NewMockSSHClient()
		registry := NewSoftwareRegistry("../../software-definitions")

		service := &SoftwareService{
			db:        db,
			sshClient: mockSSH,
			registry:  registry,
			wsHub:     mockWSHub,
		}

		// Create test device
		device := &models.Device{
			ID:        uuid.New(),
			Name:      "Test Server",
			IPAddress: "192.168.1.100",
		}
		db.Create(device)

		host := device.IPAddress + ":22"

		// Mock: Software already installed (use exact command from docker.yaml)
		mockSSH.SetResponse(host, "docker --version 2>/dev/null", "Docker version 24.0.0", nil)
		mockSSH.SetResponse(host, "docker --version 2>/dev/null | awk '{print $3}' | sed 's/,//'", "24.0.0", nil)

		// Create installation
		installation := &models.SoftwareInstallation{
			DeviceID:     device.ID,
			SoftwareName: models.SoftwareDocker,
			Status:       models.InstallationStatusPending,
		}
		db.Create(installation)

		def, _ := registry.GetDefinition("docker")

		// Execute installation
		service.executeInstallation(installation, device, def, nil)

		// Give it a moment to process
		time.Sleep(100 * time.Millisecond)

		// Reload from database
		var updated models.SoftwareInstallation
		db.First(&updated, "id = ?", installation.ID)

		// Should have CompletedAt set
		assert.NotNil(t, updated.CompletedAt, "CompletedAt should be set on success")
		assert.Equal(t, models.InstallationStatusSuccess, updated.Status)
	})

	t.Run("Sets CompletedAt on failure", func(t *testing.T) {
		db := setupSoftwareTestDB(t)
		mockWSHub := &MockWSHub{}
		mockSSH := NewMockSSHClient()

		registry := &SoftwareRegistry{
			definitions: map[string]*models.SoftwareDefinition{
				"docker": {
					Name: "Test Software",
					Commands: models.SoftwareCommands{
						Install: "failing command",
					},
				},
			},
		}

		service := &SoftwareService{
			db:        db,
			sshClient: mockSSH,
			registry:  registry,
			wsHub:     mockWSHub,
		}

		// Create test device
		device := &models.Device{
			ID:        uuid.New(),
			Name:      "Test Server",
			IPAddress: "192.168.1.100",
		}
		db.Create(device)

		// Create installation
		installation := &models.SoftwareInstallation{
			DeviceID:     device.ID,
			SoftwareName: "docker",
			Status:       models.InstallationStatusPending,
		}
		db.Create(installation)

		def := registry.definitions["docker"]

		// Execute installation (will fail because command not mocked)
		service.executeInstallation(installation, device, def, nil)

		// Give it a moment to process
		time.Sleep(100 * time.Millisecond)

		// Reload from database
		var updated models.SoftwareInstallation
		db.First(&updated, "id = ?", installation.ID)

		// Should have CompletedAt set even on failure
		assert.NotNil(t, updated.CompletedAt, "CompletedAt should be set on failure")
		assert.Equal(t, models.InstallationStatusFailed, updated.Status)
	})
}

func TestSoftwareService_DatabaseErrorHandling(t *testing.T) {
	t.Run("Fails installation when InstalledSoftware record creation fails", func(t *testing.T) {
		db := setupSoftwareTestDB(t)
		mockWSHub := &MockWSHub{}
		mockSSH := NewMockSSHClient()
		registry := NewSoftwareRegistry("../../software-definitions")

		service := &SoftwareService{
			db:        db,
			sshClient: mockSSH,
			registry:  registry,
			wsHub:     mockWSHub,
		}

		// Create test device
		device := &models.Device{
			ID:        uuid.New(),
			Name:      "Test Server",
			IPAddress: "192.168.1.100",
		}
		db.Create(device)

		host := device.IPAddress + ":22"

		// Mock: Software already installed (use exact command from docker.yaml)
		mockSSH.SetResponse(host, "docker --version 2>/dev/null", "Docker version 24.0.0", nil)
		mockSSH.SetResponse(host, "docker --version 2>/dev/null | awk '{print $3}' | sed 's/,//'", "24.0.0", nil)

		// Create installation
		installation := &models.SoftwareInstallation{
			DeviceID:     device.ID,
			SoftwareName: models.SoftwareDocker,
			Status:       models.InstallationStatusPending,
		}
		db.Create(installation)

		// Pre-create an InstalledSoftware record to cause unique constraint violation
		existingSoftware := &models.InstalledSoftware{
			ID:          uuid.New(),
			DeviceID:    device.ID,
			Name:        models.SoftwareDocker,
			Version:     "Old version",
			InstalledBy: "test",
		}
		db.Create(existingSoftware)

		def, _ := registry.GetDefinition("docker")

		// Execute installation (should fail due to duplicate record)
		service.executeInstallation(installation, device, def, nil)

		// Give it a moment to process
		time.Sleep(100 * time.Millisecond)

		// Reload from database
		var updated models.SoftwareInstallation
		db.First(&updated, "id = ?", installation.ID)

		// Should have marked as failed due to database error
		assert.Equal(t, models.InstallationStatusFailed, updated.Status)
		assert.Contains(t, updated.ErrorDetails, "database", "Error should mention database issue")
		assert.NotNil(t, updated.CompletedAt)
	})
}

func TestSoftwareService_WebSocketBroadcasting(t *testing.T) {
	t.Run("Broadcasts log and status updates via WebSocket", func(t *testing.T) {
		db := setupSoftwareTestDB(t)
		mockWSHub := &MockWSHub{}
		mockSSH := NewMockSSHClient()
		registry := NewSoftwareRegistry("../../software-definitions")

		service := &SoftwareService{
			db:        db,
			sshClient: mockSSH,
			registry:  registry,
			wsHub:     mockWSHub,
		}

		// Create test device
		device := &models.Device{
			ID:        uuid.New(),
			Name:      "Test Server",
			IPAddress: "192.168.1.100",
		}
		db.Create(device)

		host := device.IPAddress + ":22"

		// Mock: Software already installed (use exact command from docker.yaml)
		mockSSH.SetResponse(host, "docker --version 2>/dev/null", "Docker version 24.0.0", nil)
		mockSSH.SetResponse(host, "docker --version 2>/dev/null | awk '{print $3}' | sed 's/,//'", "24.0.0", nil)

		// Create installation
		installation := &models.SoftwareInstallation{
			DeviceID:     device.ID,
			SoftwareName: models.SoftwareDocker,
			Status:       models.InstallationStatusPending,
		}
		db.Create(installation)

		def, _ := registry.GetDefinition("docker")

		// Execute installation
		service.executeInstallation(installation, device, def, nil)

		// Give it a moment to process
		time.Sleep(100 * time.Millisecond)

		// Should have multiple broadcasts
		assert.Greater(t, len(mockWSHub.broadcasts), 0, "Should have broadcasted messages")

		// Check for software:log broadcasts
		logBroadcasts := 0
		statusBroadcasts := 0
		for _, broadcast := range mockWSHub.broadcasts {
			assert.Equal(t, "software", broadcast.channel, "Should broadcast on software channel")

			if broadcast.event == "software:log" {
				logBroadcasts++
				dataMap, ok := broadcast.data.(map[string]interface{})
				assert.True(t, ok, "Log data should be a map")
				assert.Contains(t, dataMap, "id")
				assert.Contains(t, dataMap, "message")
			}

			if broadcast.event == "software:status" {
				statusBroadcasts++
				dataMap, ok := broadcast.data.(map[string]interface{})
				assert.True(t, ok, "Status data should be a map")
				assert.Contains(t, dataMap, "id")
				assert.Contains(t, dataMap, "status")
			}
		}

		assert.Greater(t, logBroadcasts, 0, "Should have log broadcasts")
		assert.Greater(t, statusBroadcasts, 0, "Should have status broadcasts")
	})

	t.Run("Handles nil WSHub gracefully", func(t *testing.T) {
		db := setupSoftwareTestDB(t)
		mockSSH := NewMockSSHClient()
		registry := NewSoftwareRegistry("../../software-definitions")

		// Create service WITHOUT wsHub
		service := &SoftwareService{
			db:        db,
			sshClient: mockSSH,
			registry:  registry,
			wsHub:     nil, // No WebSocket hub
		}

		// Create test device
		device := &models.Device{
			ID:        uuid.New(),
			Name:      "Test Server",
			IPAddress: "192.168.1.100",
		}
		db.Create(device)

		host := device.IPAddress + ":22"

		// Mock: Software already installed (use exact command from docker.yaml)
		mockSSH.SetResponse(host, "docker --version 2>/dev/null", "Docker version 24.0.0", nil)
		mockSSH.SetResponse(host, "docker --version 2>/dev/null | awk '{print $3}' | sed 's/,//'", "24.0.0", nil)

		// Create installation
		installation := &models.SoftwareInstallation{
			DeviceID:     device.ID,
			SoftwareName: models.SoftwareDocker,
			Status:       models.InstallationStatusPending,
		}
		db.Create(installation)

		def, _ := registry.GetDefinition("docker")

		// Execute installation (should not panic with nil wsHub)
		assert.NotPanics(t, func() {
			service.executeInstallation(installation, device, def, nil)
			time.Sleep(100 * time.Millisecond)
		})

		// Should still complete successfully
		var updated models.SoftwareInstallation
		db.First(&updated, "id = ?", installation.ID)
		assert.Equal(t, models.InstallationStatusSuccess, updated.Status)
	})
}
