package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SoftwareType represents the type of software
type SoftwareType string

const (
	SoftwareDocker        SoftwareType = "docker"
	SoftwareDockerCompose SoftwareType = "docker-compose"
	SoftwareNFSServer     SoftwareType = "nfs-server"
	SoftwareNFSClient     SoftwareType = "nfs-client"
)

// InstalledSoftware tracks software installed on devices
type InstalledSoftware struct {
	ID          uuid.UUID    `gorm:"type:uuid;primaryKey" json:"id"`
	DeviceID    uuid.UUID    `gorm:"type:uuid;not null;uniqueIndex:idx_device_software" json:"device_id"`
	Device      Device       `gorm:"foreignKey:DeviceID" json:"-"`
	Name        SoftwareType `gorm:"not null;uniqueIndex:idx_device_software" json:"name"`
	Version     string       `json:"version"`
	InstalledAt time.Time    `json:"installed_at"`
	InstalledBy string       `json:"installed_by"` // username or "system"
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// BeforeCreate hook to generate UUID
func (s *InstalledSoftware) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	if s.InstalledAt.IsZero() {
		s.InstalledAt = time.Now()
	}
	return nil
}

// TableName overrides the default table name
func (InstalledSoftware) TableName() string {
	return "installed_software"
}
