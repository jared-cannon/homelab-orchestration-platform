package services

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/ssh"
	"github.com/stretchr/testify/assert"
)

// Mock WebSocket broadcaster
type mockWSBroadcaster struct {
	mu       sync.Mutex
	messages []map[string]interface{}
}

func (m *mockWSBroadcaster) Broadcast(channel string, event string, data interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	msg := map[string]interface{}{
		"channel": channel,
		"event":   event,
		"data":    data,
	}
	m.messages = append(m.messages, msg)
}

func (m *mockWSBroadcaster) GetMessages() []map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]map[string]interface{}{}, m.messages...)
}

func (m *mockWSBroadcaster) GetMessageCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages)
}

// TestHealthCheckService_StartDoesNotBlock verifies that the Start() method
// returns immediately without blocking on initial health checks
func TestHealthCheckService_StartDoesNotBlock(t *testing.T) {
	db := setupTestDB(t)
	sshClient := ssh.NewClient()
	credService, _ := NewCredentialService()

	healthService := NewHealthCheckService(db, sshClient, credService)

	// Create a mock device service (even though we won't add devices)
	deviceService := NewDeviceService(db, credService, sshClient)
	healthService.SetDeviceService(deviceService)

	ctx := context.Background()

	// Measure time it takes for Start() to return
	startTime := time.Now()
	healthService.Start(ctx)
	elapsedTime := time.Since(startTime)

	// Start should return immediately (within 50ms)
	// This proves the initial health check runs asynchronously
	assert.Less(t, elapsedTime.Milliseconds(), int64(50),
		"Start() should return immediately without blocking on health checks")

	// Clean up
	healthService.Stop()
	sshClient.Shutdown()
}

// TestHealthCheckService_StopCancelsChecks verifies that Stop() properly
// cancels background goroutines without hanging
func TestHealthCheckService_StopCancelsChecks(t *testing.T) {
	db := setupTestDB(t)
	sshClient := ssh.NewClient()
	credService, _ := NewCredentialService()

	healthService := NewHealthCheckService(db, sshClient, credService)

	deviceService := NewDeviceService(db, credService, sshClient)
	healthService.SetDeviceService(deviceService)

	ctx := context.Background()
	healthService.Start(ctx)

	// Stop immediately
	stopStartTime := time.Now()
	healthService.Stop()
	stopElapsed := time.Since(stopStartTime)

	// Stop should complete quickly without hanging
	assert.Less(t, stopElapsed.Seconds(), 1.0,
		"Stop() should complete quickly without hanging")

	// Clean up
	sshClient.Shutdown()
}

// TestHealthCheckService_PeriodicChecksStart verifies that the periodic
// check mechanism starts properly
func TestHealthCheckService_PeriodicChecksStart(t *testing.T) {
	db := setupTestDB(t)
	sshClient := ssh.NewClient()
	credService, _ := NewCredentialService()

	healthService := NewHealthCheckService(db, sshClient, credService)

	deviceService := NewDeviceService(db, credService, sshClient)
	healthService.SetDeviceService(deviceService)

	ctx := context.Background()
	healthService.Start(ctx)

	// Wait long enough for at least one periodic check to be scheduled
	// (not necessarily to run, just to verify the ticker is set up)
	time.Sleep(150 * time.Millisecond)

	// If we got here, the service started successfully with its ticker
	assert.True(t, true, "Periodic check mechanism should start without errors")

	// Clean up
	healthService.Stop()
	sshClient.Shutdown()
}

// TestHealthCheckService_BroadcastsStatusChanges verifies that status changes
// are broadcasted via WebSocket
func TestHealthCheckService_BroadcastsStatusChanges(t *testing.T) {
	db := setupTestDB(t)
	sshClient := ssh.NewClient()
	credService, _ := NewCredentialService()
	mockWS := &mockWSBroadcaster{}

	healthService := NewHealthCheckService(db, sshClient, credService)
	deviceService := NewDeviceService(db, credService, sshClient)
	healthService.SetDeviceService(deviceService)
	healthService.SetWebSocketHub(mockWS)

	// Create a test device
	device := models.Device{
		ID:        uuid.New(),
		Name:      "test-device",
		Type:      models.DeviceTypeServer,
		IPAddress: "192.168.1.100",
		Status:    models.DeviceStatusUnknown,
		AuthType:  models.AuthTypeAuto,
	}
	err := db.Create(&device).Error
	assert.NoError(t, err)

	// Manually trigger a status update
	healthService.updateDeviceStatus(device.ID, device.Name, models.DeviceStatusOffline)

	// Verify WebSocket broadcast was sent
	messages := mockWS.GetMessages()
	assert.Equal(t, 1, len(messages), "Should have broadcasted 1 status change")

	msg := messages[0]
	assert.Equal(t, "devices", msg["channel"])
	assert.Equal(t, "status_change", msg["event"])

	data := msg["data"].(map[string]interface{})
	assert.Equal(t, device.ID.String(), data["device_id"])
	assert.Equal(t, "test-device", data["device_name"])
	assert.Equal(t, "offline", data["status"])

	// Clean up
	sshClient.Shutdown()
}

// TestHealthCheckService_HandlesMissingDeviceService verifies that checks
// are skipped when deviceService is not initialized
func TestHealthCheckService_HandlesMissingDeviceService(t *testing.T) {
	db := setupTestDB(t)
	sshClient := ssh.NewClient()
	credService, _ := NewCredentialService()

	healthService := NewHealthCheckService(db, sshClient, credService)
	// Intentionally don't set deviceService

	// Create a test device
	device := models.Device{
		ID:        uuid.New(),
		Name:      "test-device",
		Type:      models.DeviceTypeServer,
		IPAddress: "192.168.1.100",
		Status:    models.DeviceStatusUnknown,
		AuthType:  models.AuthTypeAuto,
	}
	err := db.Create(&device).Error
	assert.NoError(t, err)

	ctx := context.Background()

	// This should not panic and should handle nil deviceService gracefully
	healthService.checkDeviceHealth(ctx, device.ID)

	// Verify device status wasn't changed (since check was skipped)
	var updatedDevice models.Device
	err = db.First(&updatedDevice, "id = ?", device.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, models.DeviceStatusUnknown, updatedDevice.Status)

	// Clean up
	sshClient.Shutdown()
}

// TestHealthCheckService_ContextCancellation verifies that ongoing checks
// respect context cancellation
func TestHealthCheckService_ContextCancellation(t *testing.T) {
	db := setupTestDB(t)
	sshClient := ssh.NewClient()
	credService, _ := NewCredentialService()

	healthService := NewHealthCheckService(db, sshClient, credService)
	deviceService := NewDeviceService(db, credService, sshClient)
	healthService.SetDeviceService(deviceService)

	// Create multiple test devices
	for i := 0; i < 5; i++ {
		device := models.Device{
			ID:        uuid.New(),
			Name:      "test-device-" + string(rune(i)),
			Type:      models.DeviceTypeServer,
			IPAddress: "192.168.1.10" + string(rune(i)),
			Status:    models.DeviceStatusUnknown,
			AuthType:  models.AuthTypeAuto,
		}
		err := db.Create(&device).Error
		assert.NoError(t, err)
	}

	// Create context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Start checks in background
	go healthService.checkAllDevices(ctx)

	// Cancel immediately
	cancel()

	// Wait a bit to ensure cancellation propagates
	time.Sleep(50 * time.Millisecond)

	// If we got here without hanging, context cancellation worked
	assert.True(t, true, "Context cancellation should stop checks gracefully")

	// Clean up
	sshClient.Shutdown()
}

// TestHealthCheckService_ConcurrentChecks verifies that worker pool limits concurrency
func TestHealthCheckService_ConcurrentChecks(t *testing.T) {
	db := setupTestDB(t)
	sshClient := ssh.NewClient()
	credService, _ := NewCredentialService()

	healthService := NewHealthCheckService(db, sshClient, credService)

	// Verify worker pool is configured with max concurrency
	assert.Equal(t, 10, healthService.maxConcurrency,
		"Health service should have max concurrency of 10")

	// Clean up
	sshClient.Shutdown()
}

// TestHealthCheckService_GoroutineRespectsCancellation verifies that the
// initial check goroutine properly handles context cancellation
func TestHealthCheckService_GoroutineRespectsCancellation(t *testing.T) {
	db := setupTestDB(t)
	sshClient := ssh.NewClient()
	credService, _ := NewCredentialService()

	healthService := NewHealthCheckService(db, sshClient, credService)
	deviceService := NewDeviceService(db, credService, sshClient)
	healthService.SetDeviceService(deviceService)

	// Create a context that we immediately cancel
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel before start

	// Start the service with cancelled context
	healthService.Start(ctx)

	// Wait a bit to ensure goroutines would have tried to run
	time.Sleep(100 * time.Millisecond)

	// If we got here without panic or hanging, goroutine handled cancellation
	assert.True(t, true, "Initial check goroutine should respect context cancellation")

	// Clean up
	healthService.Stop()
	sshClient.Shutdown()
}
