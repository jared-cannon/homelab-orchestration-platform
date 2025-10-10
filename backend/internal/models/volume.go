package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// VolumeType represents the type of Docker volume
type VolumeType string

const (
	VolumeTypeLocal VolumeType = "local"
	VolumeTypeNFS   VolumeType = "nfs"
)

// Volume represents a Docker volume
type Volume struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	DeviceID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"device_id"`
	Device      Device     `gorm:"foreignKey:DeviceID" json:"-"`
	Name        string     `gorm:"not null" json:"name"`
	Type        VolumeType `gorm:"not null" json:"type"`
	Driver      string     `gorm:"default:local" json:"driver"`
	DriverOpts  []byte     `gorm:"type:json" json:"driver_opts,omitempty"` // JSON map of driver options
	NFSServerIP string     `json:"nfs_server_ip,omitempty"`
	NFSPath     string     `json:"nfs_path,omitempty"`
	Size        int64      `json:"size"` // bytes, if known
	InUse       bool       `gorm:"default:false" json:"in_use"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// BeforeCreate hook to generate UUID
func (v *Volume) BeforeCreate(tx *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	if v.Driver == "" {
		v.Driver = "local"
	}
	return nil
}

// TableName overrides the default table name
func (Volume) TableName() string {
	return "volumes"
}
