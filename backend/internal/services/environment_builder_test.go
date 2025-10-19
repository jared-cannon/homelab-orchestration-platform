package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironmentBuilder_BuildEnvironment(t *testing.T) {
	db := setupTestDB(t)
	credService := setupTestCredentialService(t)
	dbPoolManager := NewDatabasePoolManager(db, nil, credService, nil, nil)
	envBuilder := NewEnvironmentBuilder(credService, dbPoolManager)

	device := &models.Device{
		ID:        uuid.New(),
		Name:      "test-device",
		LocalIPAddress: "192.168.1.100",
	}

	deployment := &models.Deployment{
		ID:             uuid.New(),
		RecipeSlug:     "test-app",
		RecipeName:     "Test App",
		DeviceID:       device.ID,
		ComposeProject: "test-app-abc123",
	}

	recipe := &models.Recipe{
		ConfigOptions: []models.RecipeConfigOption{
			{Name: "port", Type: "number", Required: true},
			{Name: "domain", Type: "string", Required: true},
			{Name: "api_token", Type: "secret", Required: false}, // Will be auto-generated if not provided
		},
	}

	userConfig := map[string]interface{}{
		"port":   8080,
		"domain": "test.local",
		// api_token not provided - should be auto-generated
	}

	envMap, envFileContent, err := envBuilder.BuildEnvironment(deployment, recipe, userConfig, device, nil)
	require.NoError(t, err)

	// Check basic env vars
	assert.Equal(t, "8080", envMap["PORT"])
	assert.Equal(t, "test.local", envMap["DOMAIN"])
	assert.Equal(t, deployment.ID.String(), envMap["DEPLOYMENT_ID"])
	assert.Equal(t, "test-app-abc123", envMap["COMPOSE_PROJECT"])
	assert.Equal(t, "192.168.1.100", envMap["DEVICE_IP"])

	// Check api_token was auto-generated (since it's type="secret" and not provided by user)
	assert.NotEmpty(t, envMap["API_TOKEN"])
	assert.Len(t, envMap["API_TOKEN"], 32)

	// Check env file content
	assert.Contains(t, envFileContent, "PORT=8080")
	assert.Contains(t, envFileContent, "DOMAIN=test.local")
	assert.Contains(t, envFileContent, "Auto-generated environment file")
}

func TestEnvironmentBuilder_BuildEnvironmentWithDatabase(t *testing.T) {
	db := setupTestDB(t)

	// Use a credential service with file backend for testing
	credService := setupTestCredentialService(t)
	dbPoolManager := NewDatabasePoolManager(db, nil, credService, nil, nil)
	envBuilder := NewEnvironmentBuilder(credService, dbPoolManager)

	// Store a test password in credential service
	dbPassword := "test-db-password-123"
	credKey := "test-db-cred"
	err := credService.StoreCredential(credKey, dbPassword)
	require.NoError(t, err)

	device := &models.Device{
		ID:        uuid.New(),
		Name:      "test-device",
		LocalIPAddress: "192.168.1.100",
	}

	deployment := &models.Deployment{
		ID:             uuid.New(),
		RecipeSlug:     "nextcloud",
		ComposeProject: "nextcloud-xyz",
	}

	recipe := &models.Recipe{
		Database: models.RecipeDatabaseConfig{
			Engine:        "postgres",
			AutoProvision: true,
			EnvPrefix:     "POSTGRES_",
		},
	}

	// Create a provisioned database
	provisionedDB := &models.ProvisionedDatabase{
		DatabaseName:  "nextcloud_db",
		Username:      "nextcloud_user",
		Host:          "192.168.1.100",
		Port:          5432,
		CredentialKey: credKey,
		SharedDatabaseInstance: &models.SharedDatabaseInstance{
			Engine: "postgres",
		},
	}

	envMap, envFileContent, err := envBuilder.BuildEnvironment(deployment, recipe, map[string]interface{}{}, device, provisionedDB)
	require.NoError(t, err)

	// Check database env vars
	assert.Equal(t, "192.168.1.100", envMap["POSTGRES_HOST"])
	assert.Equal(t, "5432", envMap["POSTGRES_PORT"])
	assert.Equal(t, "nextcloud_db", envMap["POSTGRES_NAME"])
	assert.Equal(t, "nextcloud_user", envMap["POSTGRES_USER"])
	assert.Equal(t, dbPassword, envMap["POSTGRES_PASSWORD"])

	// Check connection string
	expectedConnString := "postgresql://nextcloud_user:test-db-password-123@192.168.1.100:5432/nextcloud_db"
	assert.Equal(t, expectedConnString, envMap["POSTGRES_CONNECTION_STRING"])

	// Check env file
	assert.Contains(t, envFileContent, "POSTGRES_HOST=192.168.1.100")
	assert.Contains(t, envFileContent, "POSTGRES_NAME=nextcloud_db")
}

func TestEscapeEnvValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Simple value", "hello", "hello"},
		{"With spaces", "hello world", `"hello world"`},
		{"With quotes", `hello "world"`, `"hello \"world\""`},
		{"With newline", "hello\nworld", "\"hello\nworld\""},
		{"With special chars", "hello$world", `"hello$world"`},
		{"With backslash", `path\to\file`, `"path\\to\\file"`},
		{"With backslash and quotes", `say \"hello\"`, `"say \\\"hello\\\""`},
		{"Empty value", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeEnvValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		db       *models.ProvisionedDatabase
		password string
		expected string
	}{
		{
			name: "Postgres connection string",
			db: &models.ProvisionedDatabase{
				DatabaseName: "mydb",
				Username:     "myuser",
				Host:         "192.168.1.100",
				Port:         5432,
				SharedDatabaseInstance: &models.SharedDatabaseInstance{
					Engine: "postgres",
				},
			},
			password: "mypassword",
			expected: "postgresql://myuser:mypassword@192.168.1.100:5432/mydb",
		},
		{
			name: "MySQL connection string",
			db: &models.ProvisionedDatabase{
				DatabaseName: "mydb",
				Username:     "myuser",
				Host:         "192.168.1.100",
				Port:         3306,
				SharedDatabaseInstance: &models.SharedDatabaseInstance{
					Engine: "mysql",
				},
			},
			password: "mypassword",
			expected: "mysql://myuser:mypassword@192.168.1.100:3306/mydb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connString := buildConnectionString(tt.db, tt.password)
			assert.Equal(t, tt.expected, connString)
		})
	}
}

func TestEnvironmentBuilder_BuildEnvFileContent(t *testing.T) {
	credService := setupTestCredentialService(t)
	envBuilder := NewEnvironmentBuilder(credService, nil)

	envMap := map[string]string{
		"PORT":         "8080",
		"DOMAIN":       "test.local",
		"API_KEY":      "secret-key-123",
		"WITH_SPACES":  "hello world",
		"WITH_QUOTES":  `say "hello"`,
	}

	content := envBuilder.buildEnvFileContent(envMap)

	// Check header
	assert.Contains(t, content, "Auto-generated environment file")
	assert.Contains(t, content, "Generated by Homelab Orchestration Platform")

	// Check all variables are present
	assert.Contains(t, content, "PORT=8080")
	assert.Contains(t, content, "DOMAIN=test.local")
	assert.Contains(t, content, "API_KEY=secret-key-123")

	// Check proper escaping
	assert.Contains(t, content, "WITH_SPACES=\"hello world\"")
	assert.Contains(t, content, "WITH_QUOTES=\"say \\\"hello\\\"\"")
}

