package models

import (
	"time"

	"github.com/google/uuid"
)

// CredentialType represents the type of authentication
type CredentialType string

const (
	CredentialTypePassword CredentialType = "password"
	CredentialTypeSSHKey   CredentialType = "ssh_key"
)

// Credential represents stored authentication credentials for devices
// tygo:emit
type Credential struct {
	ID        string         `json:"id" gorm:"primaryKey"`
	Name      string         `json:"name"` // User-friendly name like "Home Network Default"
	Type      CredentialType `json:"type"`
	Username  string         `json:"username"`
	Password  string         `json:"password,omitempty"` // Encrypted
	SSHKey    string         `json:"ssh_key,omitempty"`  // Encrypted private key
	SSHKeyPassphrase string  `json:"-" gorm:"column:ssh_key_passphrase"` // Encrypted passphrase

	// Matching criteria - used to auto-apply credentials
	NetworkCIDR string `json:"network_cidr,omitempty"` // e.g., "192.168.1.0/24"
	DeviceType  string `json:"device_type,omitempty"`  // e.g., "server", "nas"
	HostPattern string `json:"host_pattern,omitempty"` // e.g., "*synology*", "*nas*"

	// Usage tracking
	LastUsed  *time.Time `json:"last_used,omitempty"`
	UseCount  int        `json:"use_count"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// BeforeCreate hook to set UUID
func (c *Credential) BeforeCreate(tx interface{}) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}
