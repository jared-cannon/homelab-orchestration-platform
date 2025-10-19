package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SharedCacheInstance represents a shared cache server (Redis/Memcached)
// Used by CachePoolManager for resource-efficient cache provisioning
type SharedCacheInstance struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	DeviceID        uuid.UUID `gorm:"type:uuid;not null;index:idx_device_port" json:"device_id"`
	Engine          string    `gorm:"type:varchar(50);not null" json:"engine"` // redis, memcached
	Version         string    `gorm:"type:varchar(20);not null" json:"version"`
	Name            string    `gorm:"type:varchar(255);not null" json:"name"` // e.g., "shared-redis"
	Port            int       `gorm:"not null;index:idx_device_port" json:"port"` // Composite index with DeviceID for efficient port lookups
	ContainerName   string    `gorm:"type:varchar(255);not null;uniqueIndex" json:"container_name"`
	MasterPassword  string    `gorm:"type:varchar(255);not null" json:"-"` // Master password for the cache instance (hidden from JSON)
	MaxMemoryMB     int       `gorm:"not null;default:512" json:"max_memory_mb"`
	Status          string    `gorm:"type:varchar(50);not null;default:'provisioning';index" json:"status"` // provisioning, running, stopped, error (indexed for efficient status queries)
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// BeforeCreate hook to generate UUID (SQLite-compatible approach)
func (s *SharedCacheInstance) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// TableName overrides the default table name
func (SharedCacheInstance) TableName() string {
	return "shared_cache_instances"
}

// ProvisionedCacheConfig represents per-application cache configuration within a shared instance
type ProvisionedCacheConfig struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	CacheInstanceID uuid.UUID `gorm:"type:uuid;not null;index:idx_cache_app,unique" json:"cache_instance_id"`
	AppSlug         string    `gorm:"type:varchar(255);not null;index:idx_cache_app,unique" json:"app_slug"`
	DeviceID        uuid.UUID `gorm:"type:uuid;not null;index" json:"device_id"`
	DatabaseNumber  int       `gorm:"not null" json:"database_number"` // Redis database number (0-15), set to 0 for Memcached (unused)
	KeyPrefix       string    `gorm:"type:varchar(100)" json:"key_prefix"` // Optional key prefix for isolation
	Password        string    `gorm:"type:varchar(255)" json:"-"` // Optional per-app password (if using Redis ACLs), hidden from JSON
	MaxMemoryMB     int       `gorm:"not null;default:64" json:"max_memory_mb"` // Memory limit for this app
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Foreign key
	CacheInstance SharedCacheInstance `gorm:"foreignKey:CacheInstanceID" json:"cache_instance,omitempty"`
}

// BeforeCreate hook to generate UUID (SQLite-compatible approach)
func (p *ProvisionedCacheConfig) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// TableName overrides the default table name
func (ProvisionedCacheConfig) TableName() string {
	return "provisioned_cache_configs"
}
