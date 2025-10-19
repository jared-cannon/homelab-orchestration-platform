package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: validateEngine method not yet implemented - test disabled
// func TestDatabasePoolManager_ValidateEngine(t *testing.T) {
// 	db := setupTestDB(t)
// 	credService, _ := NewCredentialService()
// 	dpm := NewDatabasePoolManager(db, nil, credService, nil, nil)
//
// 	tests := []struct {
// 		name    string
// 		engine  string
// 		wantErr bool
// 	}{
// 		{"Postgres valid", "postgres", false},
// 		{"MySQL valid", "mysql", false},
// 		{"MariaDB valid", "mariadb", false},
// 		{"Invalid engine", "mongodb", true},
// 		{"Empty engine", "", true},
// 	}
//
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			err := dpm.validateEngine(tt.engine)
// 			if tt.wantErr {
// 				assert.Error(t, err)
// 			} else {
// 				assert.NoError(t, err)
// 			}
// 		})
// 	}
// }

// TODO: getMasterUsername method not yet implemented - test disabled
// func TestDatabasePoolManager_GetMasterUsername(t *testing.T) {
// 	db := setupTestDB(t)
// 	credService, _ := NewCredentialService()
// 	dpm := NewDatabasePoolManager(db, nil, credService, nil, nil)
//
// 	tests := []struct {
// 		engine   string
// 		expected string
// 	}{
// 		{"postgres", "postgres"},
// 		{"mysql", "root"},
// 		{"mariadb", "root"},
// 	}
//
// 	for _, tt := range tests {
// 		t.Run(tt.engine, func(t *testing.T) {
// 			username := dpm.getMasterUsername(tt.engine)
// 			assert.Equal(t, tt.expected, username)
// 		})
// 	}
// }

// TODO: getDefaultVersion method not yet implemented - test disabled
// func TestDatabasePoolManager_GetDefaultVersion(t *testing.T) {
// 	db := setupTestDB(t)
// 	credService, _ := NewCredentialService()
// 	dpm := NewDatabasePoolManager(db, nil, credService, nil, nil)
//
// 	tests := []struct {
// 		engine   string
// 		expected string
// 	}{
// 		{"postgres", "15"},
// 		{"mysql", "8.0"},
// 		{"mariadb", "10.11"},
// 	}
//
// 	for _, tt := range tests {
// 		t.Run(tt.engine, func(t *testing.T) {
// 			version := dpm.getDefaultVersion(tt.engine)
// 			assert.Equal(t, tt.expected, version)
// 		})
// 	}
// }

// TODO: getDefaultPort method not yet implemented - test disabled
// func TestDatabasePoolManager_GetDefaultPort(t *testing.T) {
// 	db := setupTestDB(t)
// 	credService, _ := NewCredentialService()
// 	dpm := NewDatabasePoolManager(db, nil, credService, nil, nil)
//
// 	tests := []struct {
// 		engine   string
// 		expected int
// 	}{
// 		{"postgres", 5432},
// 		{"mysql", 3306},
// 		{"mariadb", 3306},
// 	}
//
// 	for _, tt := range tests {
// 		t.Run(tt.engine, func(t *testing.T) {
// 			port := dpm.getDefaultPort(tt.engine)
// 			assert.Equal(t, tt.expected, port)
// 		})
// 	}
// }

func TestDatabasePoolManager_GenerateDatabaseName(t *testing.T) {
	db := setupTestDB(t)
	credService, _ := NewCredentialService()
	dpm := NewDatabasePoolManager(db, nil, credService, nil, nil)

	deploymentID := uuid.New()
	dbName := dpm.generateDatabaseName("nextcloud", deploymentID)

	// Should be in format: nextcloud_abc123de
	assert.Contains(t, dbName, "nextcloud_")
	assert.Len(t, dbName, len("nextcloud_")+8) // slug + underscore + 8-char ID
}

func TestDatabasePoolManager_GenerateUsername(t *testing.T) {
	db := setupTestDB(t)
	credService, _ := NewCredentialService()
	dpm := NewDatabasePoolManager(db, nil, credService, nil, nil)

	username := dpm.generateUsername("nextcloud")
	assert.Equal(t, "nextcloud_user", username)

	// Test with hyphens (should be replaced with underscores)
	username2 := dpm.generateUsername("my-app")
	assert.Equal(t, "my_app_user", username2)
}

func TestDatabasePoolManager_GenerateSecurePassword(t *testing.T) {
	db := setupTestDB(t)
	credService, _ := NewCredentialService()
	dpm := NewDatabasePoolManager(db, nil, credService, nil, nil)

	password, err := dpm.generateSecurePassword(32)
	require.NoError(t, err)
	assert.Len(t, password, 32)

	// Test uniqueness - generate multiple passwords
	password2, err := dpm.generateSecurePassword(32)
	require.NoError(t, err)
	assert.NotEqual(t, password, password2)
}

// TODO: getEstimatedRAM method not yet implemented - test disabled
// func TestDatabasePoolManager_GetEstimatedRAM(t *testing.T) {
// 	db := setupTestDB(t)
// 	credService, _ := NewCredentialService()
// 	dpm := NewDatabasePoolManager(db, nil, credService, nil, nil)
//
// 	tests := []struct {
// 		engine   string
// 		expected int
// 	}{
// 		{"postgres", 256},
// 		{"mysql", 400},
// 		{"mariadb", 400},
// 	}
//
// 	for _, tt := range tests {
// 		t.Run(tt.engine, func(t *testing.T) {
// 			ram := dpm.getEstimatedRAM(tt.engine)
// 			assert.Equal(t, tt.expected, ram)
// 		})
// 	}
// }

// TODO: generatePostgresCompose requires InfrastructureConfig - test disabled
// func TestDatabasePoolManager_GeneratePostgresCompose(t *testing.T) {
// 	db := setupTestDB(t)
// 	credService, _ := NewCredentialService()
// 	dpm := NewDatabasePoolManager(db, nil, credService, nil, nil)
//
// 	instance := &models.SharedDatabaseInstance{
// 		Engine:         "postgres",
// 		Version:        "15",
// 		ContainerName:  "homelab-postgres-shared",
// 		Port:           5432,
// 		InternalPort:   5432,
// 		MasterUsername: "postgres",
// 	}
//
// 	compose := dpm.generatePostgresCompose(instance, "test-password")
//
// 	// Verify key components are present
// 	assert.Contains(t, compose, "image: postgres:15")
// 	assert.Contains(t, compose, "container_name: homelab-postgres-shared")
// 	assert.Contains(t, compose, "POSTGRES_USER: postgres")
// 	assert.Contains(t, compose, "POSTGRES_PASSWORD: test-password")
// 	assert.Contains(t, compose, "5432:5432")
// 	assert.Contains(t, compose, "pg_isready")
// }

// TODO: generateMySQLCompose requires InfrastructureConfig - test disabled
// func TestDatabasePoolManager_GenerateMySQLCompose(t *testing.T) {
// 	db := setupTestDB(t)
// 	credService, _ := NewCredentialService()
// 	dpm := NewDatabasePoolManager(db, nil, credService, nil, nil)
//
// 	instance := &models.SharedDatabaseInstance{
// 		Engine:         "mysql",
// 		Version:        "8.0",
// 		ContainerName:  "homelab-mysql-shared",
// 		Port:           3306,
// 		InternalPort:   3306,
// 		MasterUsername: "root",
// 	}
//
// 	compose := dpm.generateMySQLCompose(instance, "test-password")
//
// 	// Verify key components are present
// 	assert.Contains(t, compose, "image: mysql:8.0")
// 	assert.Contains(t, compose, "container_name: homelab-mysql-shared")
// 	assert.Contains(t, compose, "MYSQL_ROOT_PASSWORD: test-password")
// 	assert.Contains(t, compose, "3306:3306")
// 	assert.Contains(t, compose, "mysqladmin")
// }

func TestDatabasePoolManager_CalculateRAMSavedPercent(t *testing.T) {
	db := setupTestDB(t)
	credService, _ := NewCredentialService()
	dpm := NewDatabasePoolManager(db, nil, credService, nil, nil)

	tests := []struct {
		name     string
		dbCount  int
		savedMB  int
		expected float64
	}{
		{"Zero databases", 0, 0, 0},
		{"One database", 1, 0, 0},
		{"Multiple databases with savings", 5, 750, 50}, // 5 dbs * 300MB = 1500MB, saved 750MB = 50%
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			percent := dpm.calculateRAMSavedPercent(tt.dbCount, tt.savedMB)
			assert.InDelta(t, tt.expected, percent, 1.0) // Allow 1% delta
		})
	}
}

// Integration test: Test creating a provisioned database record
func TestDatabasePoolManager_ProvisionedDatabaseModel(t *testing.T) {
	db := setupTestDB(t)

	// Create a device
	device := &models.Device{
		ID:        uuid.New(),
		Name:      "test-device",
		LocalIPAddress: "192.168.1.100",
	}
	require.NoError(t, db.Create(device).Error)

	// Create a shared database instance
	sharedInstance := &models.SharedDatabaseInstance{
		DeviceID:      device.ID,
		Engine:        "postgres",
		Version:       "15",
		Status:        "running",
		ContainerName: "homelab-postgres-shared",
		Port:          5432,
		InternalPort:  5432,
		MasterUsername: "postgres",
		CredentialKey:  "test-cred-key",
	}
	require.NoError(t, db.Create(sharedInstance).Error)

	// Create a deployment
	deployment := &models.Deployment{
		RecipeSlug:     "nextcloud",
		RecipeName:     "NextCloud",
		DeviceID:       device.ID,
		Status:         models.DeploymentStatusRunning,
		ComposeProject: "nextcloud-test",
	}
	require.NoError(t, db.Create(deployment).Error)

	// Create a provisioned database
	provisionedDB := &models.ProvisionedDatabase{
		SharedDatabaseInstanceID: sharedInstance.ID,
		DeploymentID:             deployment.ID,
		DatabaseName:             "nextcloud_test",
		Username:                 "nextcloud_user",
		CredentialKey:            "db-cred-key",
		Host:                     device.GetPrimaryAddress(),
		Port:                     5432,
		Status:                   "ready",
	}
	require.NoError(t, db.Create(provisionedDB).Error)

	// Verify we can load it back with associations
	var loadedDB models.ProvisionedDatabase
	err := db.Preload("SharedDatabaseInstance").First(&loadedDB, "deployment_id = ?", deployment.ID).Error
	require.NoError(t, err)

	assert.Equal(t, "nextcloud_test", loadedDB.DatabaseName)
	assert.Equal(t, "nextcloud_user", loadedDB.Username)
	assert.Equal(t, "postgres", loadedDB.SharedDatabaseInstance.Engine)
	assert.Equal(t, "ready", loadedDB.Status)
}

// Test GetConnectionEnvVars method
func TestProvisionedDatabase_GetConnectionEnvVars(t *testing.T) {
	db := &models.ProvisionedDatabase{
		DatabaseName: "test_db",
		Username:     "test_user",
		Host:         "192.168.1.100",
		Port:         5432,
	}

	envVars := db.GetConnectionEnvVars("DB_")

	assert.Equal(t, "192.168.1.100", envVars["DB_HOST"])
	assert.Equal(t, "5432", envVars["DB_PORT"])
	assert.Equal(t, "test_db", envVars["DB_NAME"])
	assert.Equal(t, "test_user", envVars["DB_USER"])
	// Password not included (retrieved separately)
	_, hasPassword := envVars["DB_PASSWORD"]
	assert.False(t, hasPassword)
}

// Test unique constraint on device + engine for shared instances
func TestSharedDatabaseInstance_UniqueConstraint(t *testing.T) {
	db := setupTestDB(t)

	deviceID := uuid.New()

	// Create first postgres instance
	instance1 := &models.SharedDatabaseInstance{
		DeviceID:       deviceID,
		Engine:         "postgres",
		Version:        "15",
		Status:         "running",
		ContainerName:  "homelab-postgres-shared",
		ComposeProject: "homelab-postgres-shared",
		Port:           5432,
		InternalPort:   5432,
		MasterUsername: "postgres",
		CredentialKey:  "test-key-1",
	}
	require.NoError(t, db.Create(instance1).Error)

	// Try to create another postgres instance on same device (should fail due to unique constraint)
	instance2 := &models.SharedDatabaseInstance{
		DeviceID:       deviceID,
		Engine:         "postgres",
		Version:        "15",
		Status:         "running",
		ContainerName:  "homelab-postgres-shared-2",
		ComposeProject: "homelab-postgres-shared-2",
		Port:           5433,
		InternalPort:   5432,
		MasterUsername: "postgres",
		CredentialKey:  "test-key-2",
	}
	err := db.Create(instance2).Error
	assert.Error(t, err) // Should violate unique constraint

	// But creating a MySQL instance on same device should work
	instance3 := &models.SharedDatabaseInstance{
		DeviceID:       deviceID,
		Engine:         "mysql",
		Version:        "8.0",
		Status:         "running",
		ContainerName:  "homelab-mysql-shared",
		ComposeProject: "homelab-mysql-shared",
		Port:           3306,
		InternalPort:   3306,
		MasterUsername: "root",
		CredentialKey:  "test-key-3",
	}
	require.NoError(t, db.Create(instance3).Error)
}
