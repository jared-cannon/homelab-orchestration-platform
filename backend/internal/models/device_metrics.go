package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DeviceMetrics represents resource usage metrics for a device at a point in time
type DeviceMetrics struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	DeviceID         uuid.UUID `gorm:"type:uuid;not null;index" json:"device_id"`
	Device           *Device   `gorm:"foreignKey:DeviceID" json:"device,omitempty"`
	CPUUsagePercent  float64   `gorm:"not null" json:"cpu_usage_percent"`
	CPUCores         int       `gorm:"not null" json:"cpu_cores"`
	TotalRAMMB       int       `gorm:"not null" json:"total_ram_mb"`
	UsedRAMMB        int       `gorm:"not null" json:"used_ram_mb"`
	AvailableRAMMB   int       `gorm:"not null" json:"available_ram_mb"`
	TotalStorageGB   int       `gorm:"not null" json:"total_storage_gb"`
	UsedStorageGB    int       `gorm:"not null" json:"used_storage_gb"`
	AvailableStorageGB int     `gorm:"not null" json:"available_storage_gb"`
	RecordedAt       time.Time `gorm:"not null;index" json:"recorded_at"`
	CreatedAt        time.Time `json:"created_at"`
}

// BeforeCreate hook to generate UUID
func (dm *DeviceMetrics) BeforeCreate(tx *gorm.DB) error {
	if dm.ID == uuid.Nil {
		dm.ID = uuid.New()
	}
	if dm.RecordedAt.IsZero() {
		dm.RecordedAt = time.Now()
	}
	return nil
}

// TableName overrides the default table name
func (DeviceMetrics) TableName() string {
	return "device_metrics"
}

// RAMUsagePercent calculates the RAM usage percentage
func (dm *DeviceMetrics) RAMUsagePercent() float64 {
	if dm.TotalRAMMB == 0 {
		return 0
	}
	return (float64(dm.UsedRAMMB) / float64(dm.TotalRAMMB)) * 100
}

// StorageUsagePercent calculates the storage usage percentage
func (dm *DeviceMetrics) StorageUsagePercent() float64 {
	if dm.TotalStorageGB == 0 {
		return 0
	}
	return (float64(dm.UsedStorageGB) / float64(dm.TotalStorageGB)) * 100
}
