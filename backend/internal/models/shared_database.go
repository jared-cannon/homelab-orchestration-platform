package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SharedDatabaseInstance represents a shared database container running on a device
// Multiple applications can have isolated databases within a single shared instance
type SharedDatabaseInstance struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	DeviceID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_device_engine" json:"device_id"`
	Device     *Device   `gorm:"foreignKey:DeviceID" json:"device,omitempty"`
	Engine     string    `gorm:"not null;uniqueIndex:idx_device_engine" json:"engine"` // "postgres", "mysql", "mariadb"
	Version    string    `gorm:"not null" json:"version"`                               // e.g., "15", "8.0"
	Status     string    `gorm:"default:provisioning" json:"status"`                    // "provisioning", "running", "failed", "stopped"

	// Container information
	ContainerID      string `json:"container_id,omitempty"`
	ContainerName    string `gorm:"not null" json:"container_name"`    // e.g., "homelab-postgres-shared"
	ComposeProject   string `gorm:"not null" json:"compose_project"`   // Docker Compose project name

	// Connection details
	Port             int    `gorm:"not null" json:"port"`              // Exposed port on device
	InternalPort     int    `gorm:"not null" json:"internal_port"`     // Container internal port (5432 for postgres, 3306 for mysql)

	// Master credentials (encrypted in credential store)
	MasterUsername   string `gorm:"not null" json:"master_username"`   // Usually "postgres" or "root"
	CredentialKey    string `gorm:"not null" json:"-"`                 // Reference to encrypted password in credential store

	// Resource tracking
	EstimatedRAMMB   int    `json:"estimated_ram_mb"`                  // Estimated RAM usage
	DatabaseCount    int    `gorm:"default:0" json:"database_count"`   // Number of databases in this instance

	// Metadata
	DeployedAt       *time.Time `json:"deployed_at,omitempty"`
	LastHealthCheck  *time.Time `json:"last_health_check,omitempty"`
	ErrorDetails     string     `gorm:"type:text" json:"error_details,omitempty"`

	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// BeforeCreate hook to generate UUID
func (s *SharedDatabaseInstance) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// TableName overrides the default table name
func (SharedDatabaseInstance) TableName() string {
	return "shared_database_instances"
}

// ProvisionedDatabase represents an isolated database within a shared database instance
type ProvisionedDatabase struct {
	ID                         uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	SharedDatabaseInstanceID   uuid.UUID `gorm:"type:uuid;not null;index" json:"shared_database_instance_id"`
	SharedDatabaseInstance     *SharedDatabaseInstance `gorm:"foreignKey:SharedDatabaseInstanceID" json:"shared_instance,omitempty"`

	DeploymentID               uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"deployment_id"` // One database per deployment
	Deployment                 *Deployment `gorm:"foreignKey:DeploymentID" json:"deployment,omitempty"`

	// Database details
	DatabaseName               string `gorm:"not null" json:"database_name"`           // e.g., "nextcloud_abc123"
	Username                   string `gorm:"not null" json:"username"`                // e.g., "nextcloud_user"
	CredentialKey              string `gorm:"not null" json:"-"`                       // Reference to encrypted password

	// Connection string components (injected as env vars into application)
	Host                       string `json:"host"`                                    // Device IP or hostname
	Port                       int    `json:"port"`                                    // Shared instance port

	// Status
	Status                     string `gorm:"default:provisioning" json:"status"`     // "provisioning", "ready", "failed"
	ErrorDetails               string `gorm:"type:text" json:"error_details,omitempty"`

	// Metadata
	ProvisionedAt              *time.Time `json:"provisioned_at,omitempty"`
	CreatedAt                  time.Time  `json:"created_at"`
	UpdatedAt                  time.Time  `json:"updated_at"`
}

// BeforeCreate hook to generate UUID
func (p *ProvisionedDatabase) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// TableName overrides the default table name
func (ProvisionedDatabase) TableName() string {
	return "provisioned_databases"
}

// GetConnectionEnvVars returns environment variables for connecting to this database
func (p *ProvisionedDatabase) GetConnectionEnvVars(prefix string) map[string]string {
	if prefix == "" {
		prefix = "DB_"
	}

	return map[string]string{
		prefix + "HOST":     p.Host,
		prefix + "PORT":     fmt.Sprintf("%d", p.Port),
		prefix + "NAME":     p.DatabaseName,
		prefix + "USER":     p.Username,
		// Password retrieved separately from credential store
	}
}
