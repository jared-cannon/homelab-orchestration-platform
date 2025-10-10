package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SoftwareInstallationStatus represents the status of a software installation
type SoftwareInstallationStatus string

const (
	InstallationStatusPending    SoftwareInstallationStatus = "pending"
	InstallationStatusInstalling SoftwareInstallationStatus = "installing"
	InstallationStatusSuccess    SoftwareInstallationStatus = "success"
	InstallationStatusFailed     SoftwareInstallationStatus = "failed"
)

// SoftwareInstallation represents a software installation job
type SoftwareInstallation struct {
	ID             uuid.UUID                   `gorm:"type:uuid;primary_key" json:"id"`
	DeviceID       uuid.UUID                   `gorm:"type:uuid;not null;index" json:"device_id"`
	SoftwareName   SoftwareType                `gorm:"type:varchar(100);not null" json:"software_name"`
	Status         SoftwareInstallationStatus  `gorm:"type:varchar(50);not null;default:'pending'" json:"status"`
	InstallLogs    string                      `gorm:"type:text" json:"install_logs,omitempty"`
	ErrorDetails   string                      `gorm:"type:text" json:"error_details,omitempty"`
	CreatedAt      time.Time                   `gorm:"autoCreateTime" json:"created_at"`
	CompletedAt    *time.Time                  `json:"completed_at,omitempty"`

	// Relationships
	Device         *Device                     `gorm:"foreignKey:DeviceID" json:"device,omitempty"`
}

// BeforeCreate hook to generate UUID for cross-database compatibility
func (s *SoftwareInstallation) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// TableName specifies the table name for SoftwareInstallation
func (SoftwareInstallation) TableName() string {
	return "software_installations"
}
