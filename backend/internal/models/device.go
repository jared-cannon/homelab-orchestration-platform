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
	AuthTypeAuto      AuthType = "auto"      // SSH agent or default keys
	AuthTypePassword  AuthType = "password"  // Password authentication
	AuthTypeSSHKey    AuthType = "ssh_key"   // SSH key authentication
	AuthTypeTailscale AuthType = "tailscale" // Tailscale SSH (uses Tailscale's built-in SSH)
)

// PrimaryConnection represents which connection type to use first
type PrimaryConnection string

const (
	PrimaryConnectionLocal     PrimaryConnection = "local"     // Use local IP address first
	PrimaryConnectionTailscale PrimaryConnection = "tailscale" // Use Tailscale address first
)

// Device represents a managed device (server, router, NAS, etc.)
type Device struct {
	ID                uuid.UUID         `gorm:"type:uuid;primaryKey" json:"id"`
	Name              string            `gorm:"not null" json:"name"`
	Type              DeviceType        `gorm:"not null" json:"type"`
	LocalIPAddress    string            `gorm:"not null;uniqueIndex" json:"local_ip_address"`
	TailscaleAddress  string            `json:"tailscale_address,omitempty"`                           // Tailscale IP or hostname (optional)
	PrimaryConnection PrimaryConnection `gorm:"default:local" json:"primary_connection"`               // Which connection to try first
	MACAddress        string            `json:"mac_address,omitempty"`
	Status            DeviceStatus      `gorm:"default:unknown" json:"status"`
	Username          string            `gorm:"default:''" json:"username"`              // SSH username (not sensitive)
	AuthType          AuthType          `gorm:"default:auto" json:"auth_type"`           // Authentication method
	CredentialKey     string            `json:"-"`                                       // Reference to credential in keychain (only for password/ssh_key), never expose in JSON
	Metadata          []byte            `gorm:"type:json" json:"metadata,omitempty"`

	// Current resource metrics (updated by ResourceMonitoringService)
	CPUUsagePercent    *float64   `json:"cpu_usage_percent,omitempty"`
	CPUCores           *int       `json:"cpu_cores,omitempty"`
	TotalRAMMB         *int       `json:"total_ram_mb,omitempty"`
	UsedRAMMB          *int       `json:"used_ram_mb,omitempty"`
	AvailableRAMMB     *int       `json:"available_ram_mb,omitempty"`
	TotalStorageGB     *int       `json:"total_storage_gb,omitempty"`
	UsedStorageGB      *int       `json:"used_storage_gb,omitempty"`
	AvailableStorageGB *int       `json:"available_storage_gb,omitempty"`
	ResourcesUpdatedAt *time.Time `json:"resources_updated_at,omitempty"`

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

// GetPrimaryAddress returns the primary connection address based on PrimaryConnection setting
func (d *Device) GetPrimaryAddress() string {
	if d.PrimaryConnection == PrimaryConnectionTailscale && d.TailscaleAddress != "" {
		return d.TailscaleAddress
	}
	return d.LocalIPAddress
}

// GetFallbackAddress returns the fallback connection address
func (d *Device) GetFallbackAddress() string {
	if d.PrimaryConnection == PrimaryConnectionTailscale {
		return d.LocalIPAddress
	}
	if d.TailscaleAddress != "" {
		return d.TailscaleAddress
	}
	return ""
}

// GetSSHHost returns the primary SSH connection host (address:22)
func (d *Device) GetSSHHost() string {
	return d.GetPrimaryAddress() + ":22"
}
