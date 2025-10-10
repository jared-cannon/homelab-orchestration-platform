package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DeviceType represents the type of device
type DeviceType string

const (
	DeviceTypeRouter DeviceType = "router"
	DeviceTypeServer DeviceType = "server"
	DeviceTypeNAS    DeviceType = "nas"
	DeviceTypeSwitch DeviceType = "switch"
)

// DeviceStatus represents the current status of a device
type DeviceStatus string

const (
	DeviceStatusOnline  DeviceStatus = "online"
	DeviceStatusOffline DeviceStatus = "offline"
	DeviceStatusError   DeviceStatus = "error"
	DeviceStatusUnknown DeviceStatus = "unknown"
)

// AuthType represents the authentication method for SSH
type AuthType string

const (
	AuthTypeAuto     AuthType = "auto"     // SSH agent or default keys
	AuthTypePassword AuthType = "password" // Password authentication
	AuthTypeSSHKey   AuthType = "ssh_key"  // SSH key authentication
)

// Device represents a managed device (server, router, NAS, etc.)
type Device struct {
	ID            uuid.UUID    `gorm:"type:uuid;primaryKey" json:"id"`
	Name          string       `gorm:"not null" json:"name"`
	Type          DeviceType   `gorm:"not null" json:"type"`
	IPAddress     string       `gorm:"not null;uniqueIndex" json:"ip_address"`
	MACAddress    string       `json:"mac_address,omitempty"`
	Status        DeviceStatus `gorm:"default:unknown" json:"status"`
	Username      string       `gorm:"default:''" json:"username"`              // SSH username (not sensitive)
	AuthType      AuthType     `gorm:"default:auto" json:"auth_type"`           // Authentication method
	CredentialKey string       `json:"-"`                                       // Reference to credential in keychain (only for password/ssh_key), never expose in JSON
	Metadata      []byte       `gorm:"type:json" json:"metadata,omitempty"`
	LastSeen      *time.Time   `json:"last_seen,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

// BeforeCreate hook to generate UUID
func (d *Device) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	if d.Status == "" {
		d.Status = DeviceStatusUnknown
	}
	return nil
}

// TableName overrides the default table name
func (Device) TableName() string {
	return "devices"
}
