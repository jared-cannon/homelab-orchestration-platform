package services

import (
	"os"
	"testing"

	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestMain runs once before all tests in this package
// It sets up the test environment including GO_ENV=test for encryption key handling
func TestMain(m *testing.M) {
	// Set test environment variable for all tests in this package
	// This allows NewCredentialService() to use a default encryption key in tests
	os.Setenv("GO_ENV", "test")

	// Run all tests
	exitCode := m.Run()

	// Exit with the test result code
	os.Exit(exitCode)
}

// setupTestDB creates an in-memory SQLite database for testing
// This is a shared helper used by all test files in the services package
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "Failed to open in-memory database")

	// Auto-migrate all models used across tests
	err = db.AutoMigrate(
		&models.Device{},
		&models.DeviceMetrics{},
		&models.Application{},
		&models.Deployment{},
		&models.Credential{},
		&models.SharedDatabaseInstance{},
		&models.ProvisionedDatabase{},
		&models.InstalledSoftware{},
		&models.SoftwareInstallation{},
		&models.NFSExport{},
		&models.NFSMount{},
		&models.Volume{},
	)
	require.NoError(t, err, "Failed to run migrations")

	return db
}
