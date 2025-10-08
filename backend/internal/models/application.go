package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Application represents an application that can be deployed
type Application struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name            string    `gorm:"not null" json:"name"`
	Slug            string    `gorm:"not null;uniqueIndex" json:"slug"`
	Category        string    `json:"category"`
	Description     string    `gorm:"type:text" json:"description"`
	IconURL         string    `json:"icon_url,omitempty"`
	DockerImage     string    `gorm:"not null" json:"docker_image"`
	RequiredRAM     int64     `json:"required_ram"`     // bytes
	RequiredStorage int64     `json:"required_storage"` // bytes
	ConfigTemplate  string    `gorm:"type:text" json:"config_template"`
	SetupSteps      []byte    `gorm:"type:json" json:"setup_steps,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// BeforeCreate hook to generate UUID
func (a *Application) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// TableName overrides the default table name
func (Application) TableName() string {
	return "applications"
}
