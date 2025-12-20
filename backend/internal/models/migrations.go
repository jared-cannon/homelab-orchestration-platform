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

// MigrateDNSAndURLFields adds DNS/URL-related fields to deployments and devices
// This migration adds support for Traefik integration and app access URLs
func MigrateDNSAndURLFields(db *gorm.DB) error {
	migrator := db.Migrator()

	// ========== Device Table Migrations ==========

	// Add domain_suffix column to devices if it doesn't exist
	if !migrator.HasColumn(&Device{}, "domain_suffix") {
		if err := migrator.AddColumn(&Device{}, "domain_suffix"); err != nil {
			return fmt.Errorf("failed to add domain_suffix column to devices: %w", err)
		}
		fmt.Println("✅ Added domain_suffix column to devices")

		// Set default domain suffix for existing devices based on device name
		// This ensures backward compatibility
		var devices []Device
		if err := db.Find(&devices).Error; err == nil {
			for _, device := range devices {
				domainSuffix := device.GetDefaultDomainSuffix()
				db.Model(&Device{}).Where("id = ?", device.ID).Update("domain_suffix", domainSuffix)
			}
			if len(devices) > 0 {
				fmt.Printf("✅ Set default domain suffixes for %d existing devices\n", len(devices))
			}
		}
	}

	// ========== Deployment Table Migrations ==========

	// Add hostname column to deployments if it doesn't exist
	if !migrator.HasColumn(&Deployment{}, "hostname") {
		if err := migrator.AddColumn(&Deployment{}, "hostname"); err != nil {
			return fmt.Errorf("failed to add hostname column to deployments: %w", err)
		}
		fmt.Println("✅ Added hostname column to deployments")
	}

	// Add url column to deployments if it doesn't exist
	if !migrator.HasColumn(&Deployment{}, "url") {
		if err := migrator.AddColumn(&Deployment{}, "url"); err != nil {
			return fmt.Errorf("failed to add url column to deployments: %w", err)
		}
		fmt.Println("✅ Added url column to deployments")
	}

	// Add use_traefik column to deployments if it doesn't exist
	if !migrator.HasColumn(&Deployment{}, "use_traefik") {
		if err := migrator.AddColumn(&Deployment{}, "use_traefik"); err != nil {
			return fmt.Errorf("failed to add use_traefik column to deployments: %w", err)
		}
		// Set default value to true for new deployments
		if err := db.Model(&Deployment{}).Where("use_traefik IS NULL OR use_traefik = ?", false).Update("use_traefik", true).Error; err != nil {
			return fmt.Errorf("failed to set default use_traefik: %w", err)
		}
		fmt.Println("✅ Added use_traefik column to deployments")
	}

	// Add orchestrator_mode column to deployments if it doesn't exist
	if !migrator.HasColumn(&Deployment{}, "orchestrator_mode") {
		if err := migrator.AddColumn(&Deployment{}, "orchestrator_mode"); err != nil {
			return fmt.Errorf("failed to add orchestrator_mode column to deployments: %w", err)
		}
		// Set default value to "compose" for existing deployments
		if err := db.Model(&Deployment{}).Where("orchestrator_mode = ? OR orchestrator_mode IS NULL", "").Update("orchestrator_mode", "compose").Error; err != nil {
			return fmt.Errorf("failed to set default orchestrator_mode: %w", err)
		}
		fmt.Println("✅ Added orchestrator_mode column to deployments")
	}

	return nil
}
