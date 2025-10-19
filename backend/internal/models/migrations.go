package models

import (
	"fmt"

	"gorm.io/gorm"
)

// MigrateDualIPAddresses handles migration from single ip_address to dual addresses
// Uses GORM's Migrator API for database independence
func MigrateDualIPAddresses(db *gorm.DB) error {
	migrator := db.Migrator()

	// Check if old column exists and rename it
	if migrator.HasColumn(&Device{}, "ip_address") {
		if err := migrator.RenameColumn(&Device{}, "ip_address", "local_ip_address"); err != nil {
			return fmt.Errorf("failed to rename ip_address column: %w", err)
		}
		fmt.Println("✅ Migrated ip_address → local_ip_address")
	}

	// Add tailscale_address column if it doesn't exist
	if !migrator.HasColumn(&Device{}, "tailscale_address") {
		if err := migrator.AddColumn(&Device{}, "tailscale_address"); err != nil {
			return fmt.Errorf("failed to add tailscale_address column: %w", err)
		}
		fmt.Println("✅ Added tailscale_address column")
	}

	// Add primary_connection column if it doesn't exist
	if !migrator.HasColumn(&Device{}, "primary_connection") {
		if err := migrator.AddColumn(&Device{}, "primary_connection"); err != nil {
			return fmt.Errorf("failed to add primary_connection column: %w", err)
		}
		// Set default value for existing rows
		if err := db.Model(&Device{}).Where("primary_connection = ? OR primary_connection IS NULL", "").Update("primary_connection", "local").Error; err != nil {
			return fmt.Errorf("failed to set default primary_connection: %w", err)
		}
		fmt.Println("✅ Added primary_connection column")
	}

	return nil
}
