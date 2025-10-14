package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupResourceMonitoringTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Run migrations
	if err := db.AutoMigrate(&models.Device{}, &models.DeviceMetrics{}); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

func TestNewResourceMonitoringService(t *testing.T) {
	db := setupResourceMonitoringTestDB(t)

	config := &ResourceMonitoringConfig{
		PollInterval:    10 * time.Second,
		RetentionPeriod: 1 * time.Hour,
		MaxConcurrent:   5,
	}

	service := NewResourceMonitoringService(db, nil, nil, nil, config)

	assert.NotNil(t, service)
	assert.Equal(t, 10*time.Second, service.pollInterval)
	assert.Equal(t, 1*time.Hour, service.retentionPeriod)
	assert.Equal(t, 5, service.maxConcurrent)
	assert.NotNil(t, service.failureCount)
}

func TestNewResourceMonitoringService_DefaultConfig(t *testing.T) {
	db := setupResourceMonitoringTestDB(t)

	service := NewResourceMonitoringService(db, nil, nil, nil, nil)

	assert.NotNil(t, service)
	assert.Equal(t, 30*time.Second, service.pollInterval)
	assert.Equal(t, 24*time.Hour, service.retentionPeriod)
	assert.Equal(t, 10, service.maxConcurrent)
}

func TestGetStatus_NotStarted(t *testing.T) {
	db := setupResourceMonitoringTestDB(t)
	service := NewResourceMonitoringService(db, nil, nil, nil, nil)

	status := service.GetStatus()

	assert.NotNil(t, status)
	assert.False(t, status.Running)
	assert.False(t, status.Healthy)
	assert.Equal(t, "Monitoring service is not running", status.HealthMessage)
	assert.Equal(t, int64(0), status.TotalPollsRun)
	assert.Equal(t, int64(0), status.TotalMetricsCollected)
	assert.Equal(t, int64(0), status.TotalErrors)
}

func TestGetAggregateResources_EmptyDatabase(t *testing.T) {
	db := setupResourceMonitoringTestDB(t)
	service := NewResourceMonitoringService(db, nil, nil, nil, nil)

	agg, err := service.GetAggregateResources()

	assert.NoError(t, err)
	assert.NotNil(t, agg)
	assert.Equal(t, 0, agg.TotalDevices)
	assert.Equal(t, 0, agg.OnlineDevices)
	assert.Equal(t, 0, agg.TotalCPUCores)
}

func TestGetAggregateResources_WithDevices(t *testing.T) {
	db := setupResourceMonitoringTestDB(t)
	service := NewResourceMonitoringService(db, nil, nil, nil, nil)

	// Create test devices with metrics
	now := time.Now()
	cpuUsage1 := 50.0
	cpuCores1 := 4
	cpuUsage2 := 75.0
	cpuCores2 := 8

	device1 := models.Device{
		ID:                 uuid.New(),
		Name:               "Test Device 1",
		Type:               models.DeviceTypeServer,
		IPAddress:          "192.168.1.100",
		Status:             models.DeviceStatusOnline,
		Username:           "admin",
		AuthType:           models.AuthTypeAuto,
		CPUUsagePercent:    &cpuUsage1,
		CPUCores:           &cpuCores1,
		ResourcesUpdatedAt: &now,
	}

	device2 := models.Device{
		ID:                 uuid.New(),
		Name:               "Test Device 2",
		Type:               models.DeviceTypeServer,
		IPAddress:          "192.168.1.101",
		Status:             models.DeviceStatusOnline,
		Username:           "admin",
		AuthType:           models.AuthTypeAuto,
		CPUUsagePercent:    &cpuUsage2,
		CPUCores:           &cpuCores2,
		ResourcesUpdatedAt: &now,
	}

	db.Create(&device1)
	db.Create(&device2)

	agg, err := service.GetAggregateResources()

	assert.NoError(t, err)
	assert.NotNil(t, agg)
	assert.Equal(t, 2, agg.TotalDevices)
	assert.Equal(t, 2, agg.OnlineDevices)
	assert.Equal(t, 12, agg.TotalCPUCores) // 4 + 8

	// Core-weighted CPU calculation:
	// Device 1: 4 cores * 50% = 2 cores used
	// Device 2: 8 cores * 75% = 6 cores used
	// Total: 8 cores used / 12 total = 66.67%
	expectedUsedCores := 2.0 + 6.0
	expectedPercent := (expectedUsedCores / 12.0) * 100.0

	assert.InDelta(t, expectedUsedCores, agg.UsedCPUCores, 0.01)
	assert.InDelta(t, expectedPercent, agg.AvgCPUUsagePercent, 0.01)
}

func TestGetAggregateResources_ExcludesStaleDevices(t *testing.T) {
	db := setupResourceMonitoringTestDB(t)
	service := NewResourceMonitoringService(db, nil, nil, nil, nil)

	now := time.Now()
	cpuUsage := 50.0
	cpuCores := 4

	// Device with metrics
	deviceWithMetrics := models.Device{
		ID:                 uuid.New(),
		Name:               "Device With Metrics",
		Type:               models.DeviceTypeServer,
		IPAddress:          "192.168.1.100",
		Status:             models.DeviceStatusOnline,
		Username:           "admin",
		AuthType:           models.AuthTypeAuto,
		CPUUsagePercent:    &cpuUsage,
		CPUCores:           &cpuCores,
		ResourcesUpdatedAt: &now,
	}

	// Device without metrics (stale)
	deviceWithoutMetrics := models.Device{
		ID:                 uuid.New(),
		Name:               "Device Without Metrics",
		Type:               models.DeviceTypeServer,
		IPAddress:          "192.168.1.101",
		Status:             models.DeviceStatusOnline,
		Username:           "admin",
		AuthType:           models.AuthTypeAuto,
		ResourcesUpdatedAt: nil, // No metrics - should be excluded
	}

	db.Create(&deviceWithMetrics)
	db.Create(&deviceWithoutMetrics)

	agg, err := service.GetAggregateResources()

	assert.NoError(t, err)
	assert.Equal(t, 2, agg.TotalDevices)
	assert.Equal(t, 2, agg.OnlineDevices)
	// Only device with metrics should be counted
	assert.Equal(t, 4, agg.TotalCPUCores)
	assert.InDelta(t, 2.0, agg.UsedCPUCores, 0.01) // 4 * 0.5
}

func TestClearDeviceMetrics(t *testing.T) {
	db := setupResourceMonitoringTestDB(t)
	service := NewResourceMonitoringService(db, nil, nil, nil, nil)

	// Create device with metrics
	now := time.Now()
	cpuUsage := 50.0
	cpuCores := 4
	device := models.Device{
		ID:                 uuid.New(),
		Name:               "Test Device",
		Type:               models.DeviceTypeServer,
		IPAddress:          "192.168.1.100",
		Status:             models.DeviceStatusOnline,
		Username:           "admin",
		AuthType:           models.AuthTypeAuto,
		CPUUsagePercent:    &cpuUsage,
		CPUCores:           &cpuCores,
		ResourcesUpdatedAt: &now,
	}

	db.Create(&device)

	// Clear metrics
	err := service.clearDeviceMetrics(&device)
	assert.NoError(t, err)

	// Verify metrics are cleared
	var updated models.Device
	db.First(&updated, "id = ?", device.ID)

	assert.Nil(t, updated.CPUUsagePercent)
	assert.Nil(t, updated.CPUCores)
	assert.Nil(t, updated.ResourcesUpdatedAt)
}

func TestUpdateDeviceMetrics(t *testing.T) {
	db := setupResourceMonitoringTestDB(t)
	service := NewResourceMonitoringService(db, nil, nil, nil, nil)

	// Create device
	device := models.Device{
		ID:        uuid.New(),
		Name:      "Test Device",
		Type:      models.DeviceTypeServer,
		IPAddress: "192.168.1.100",
		Status:    models.DeviceStatusOnline,
		Username:  "admin",
		AuthType:  models.AuthTypeAuto,
	}

	db.Create(&device)

	// Create metrics
	metrics := &models.DeviceMetrics{
		ID:               uuid.New(),
		DeviceID:         device.ID,
		CPUUsagePercent:  45.5,
		CPUCores:         8,
		TotalRAMMB:       16384,
		UsedRAMMB:        8192,
		AvailableRAMMB:   8192,
		TotalStorageGB:   500,
		UsedStorageGB:    250,
		AvailableStorageGB: 250,
		RecordedAt:       time.Now(),
	}

	// Update device with metrics
	err := service.updateDeviceMetrics(&device, metrics)
	assert.NoError(t, err)

	// Verify device was updated
	var updated models.Device
	db.First(&updated, "id = ?", device.ID)

	assert.NotNil(t, updated.CPUUsagePercent)
	assert.Equal(t, 45.5, *updated.CPUUsagePercent)
	assert.NotNil(t, updated.CPUCores)
	assert.Equal(t, 8, *updated.CPUCores)
	assert.NotNil(t, updated.ResourcesUpdatedAt)
}
