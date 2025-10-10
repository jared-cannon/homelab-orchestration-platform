package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DeploymentStatus represents the status of a deployment
type DeploymentStatus string

const (
	DeploymentStatusValidating  DeploymentStatus = "validating"
	DeploymentStatusPreparing   DeploymentStatus = "preparing"
	DeploymentStatusDeploying   DeploymentStatus = "deploying"
	DeploymentStatusConfiguring DeploymentStatus = "configuring"
	DeploymentStatusHealthCheck DeploymentStatus = "health_check"
	DeploymentStatusRunning     DeploymentStatus = "running"
	DeploymentStatusStopped     DeploymentStatus = "stopped"
	DeploymentStatusFailed      DeploymentStatus = "failed"
	DeploymentStatusRollingBack DeploymentStatus = "rolling_back"
	DeploymentStatusRolledBack  DeploymentStatus = "rolled_back"
)

// Deployment represents a deployed application on a device
type Deployment struct {
	ID               uuid.UUID        `gorm:"type:uuid;primaryKey" json:"id"`
	RecipeSlug       string           `gorm:"not null" json:"recipe_slug"`                  // Marketplace recipe identifier
	RecipeName       string           `json:"recipe_name"`                                  // Cached for display
	ApplicationID    uuid.UUID        `gorm:"type:uuid" json:"application_id,omitempty"`    // Legacy - made nullable
	Application      *Application     `gorm:"foreignKey:ApplicationID" json:"application,omitempty"`
	DeviceID         uuid.UUID        `gorm:"type:uuid;not null" json:"device_id"`
	Device           *Device          `gorm:"foreignKey:DeviceID" json:"device,omitempty"`
	Status           DeploymentStatus `gorm:"default:validating" json:"status"`
	Config           []byte           `gorm:"type:json" json:"config,omitempty"`
	Domain           string           `json:"domain,omitempty"`
	InternalPort     int              `json:"internal_port"`
	ExternalPort     int              `json:"external_port,omitempty"`
	ContainerID      string           `json:"container_id,omitempty"`
	ComposeProject   string           `json:"compose_project,omitempty"`                    // Docker Compose project name
	GeneratedCompose string           `gorm:"type:text" json:"generated_compose,omitempty"` // For debugging/transparency
	DeploymentLogs   string           `gorm:"type:text" json:"deployment_logs,omitempty"`   // Logs from deployment process
	SSHCommands      []byte           `gorm:"type:json" json:"ssh_commands,omitempty"`      // For debugging
	RollbackLog      []byte           `gorm:"type:json" json:"rollback_log,omitempty"`      // For debugging
	ErrorDetails     string           `gorm:"type:text" json:"error_details,omitempty"`
	DeployedAt       *time.Time       `json:"deployed_at,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// BeforeCreate hook to generate UUID
func (d *Deployment) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	if d.Status == "" {
		d.Status = DeploymentStatusValidating
	}
	return nil
}

// TableName overrides the default table name
func (Deployment) TableName() string {
	return "deployments"
}
