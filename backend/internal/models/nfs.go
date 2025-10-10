package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NFSExport represents an NFS export configured on a server device
type NFSExport struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	DeviceID   uuid.UUID `gorm:"type:uuid;not null;index" json:"device_id"` // The NFS server
	Device     Device    `gorm:"foreignKey:DeviceID" json:"-"`
	Path       string    `gorm:"not null" json:"path"`                  // e.g., "/srv/nfs/shared"
	ClientCIDR string    `gorm:"default:*" json:"client_cidr"`          // e.g., "*", "192.168.1.0/24"
	Options    string    `gorm:"default:rw,sync,no_subtree_check,no_root_squash" json:"options"`
	Active     bool      `gorm:"default:true" json:"active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// BeforeCreate hook to generate UUID
func (n *NFSExport) BeforeCreate(tx *gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	if n.Options == "" {
		n.Options = "rw,sync,no_subtree_check,no_root_squash"
	}
	if n.ClientCIDR == "" {
		n.ClientCIDR = "*"
	}
	return nil
}

// TableName overrides the default table name
func (NFSExport) TableName() string {
	return "nfs_exports"
}

// NFSMount represents an NFS mount on a client device
type NFSMount struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	DeviceID   uuid.UUID `gorm:"type:uuid;not null;index" json:"device_id"` // The NFS client
	Device     Device    `gorm:"foreignKey:DeviceID" json:"-"`
	ServerIP   string    `gorm:"not null" json:"server_ip"`     // NFS server IP
	RemotePath string    `gorm:"not null" json:"remote_path"`   // e.g., "/srv/nfs/shared"
	LocalPath  string    `gorm:"not null" json:"local_path"`    // e.g., "/mnt/nfs/shared"
	Options    string    `gorm:"default:defaults" json:"options"`
	Permanent  bool      `gorm:"default:true" json:"permanent"` // Add to /etc/fstab
	Active     bool      `gorm:"default:true" json:"active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// BeforeCreate hook to generate UUID
func (n *NFSMount) BeforeCreate(tx *gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	if n.Options == "" {
		n.Options = "defaults"
	}
	return nil
}

// TableName overrides the default table name
func (NFSMount) TableName() string {
	return "nfs_mounts"
}
