package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretManager_GenerateSecret(t *testing.T) {
	credService := setupTestCredentialService(t)
	sm := NewSecretManager(credService)

	t.Run("Generate secret of length 32", func(t *testing.T) {
		secret, err := sm.generateSecret(32)
		require.NoError(t, err)
		assert.Len(t, secret, 32)
	})

	t.Run("Generate secret of length 16", func(t *testing.T) {
		secret, err := sm.generateSecret(16)
		require.NoError(t, err)
		assert.Len(t, secret, 16)
	})

	t.Run("Generate secret of length 64", func(t *testing.T) {
		secret, err := sm.generateSecret(64)
		require.NoError(t, err)
		assert.Len(t, secret, 64)
	})

	t.Run("Secrets are unique", func(t *testing.T) {
		secret1, err := sm.generateSecret(32)
		require.NoError(t, err)

		secret2, err := sm.generateSecret(32)
		require.NoError(t, err)

		assert.NotEqual(t, secret1, secret2, "Generated secrets should be unique")
	})

	t.Run("Secrets are URL-safe", func(t *testing.T) {
		secret, err := sm.generateSecret(32)
		require.NoError(t, err)

		// URL-safe base64 should only contain: A-Z, a-z, 0-9, -, _
		for _, char := range secret {
			valid := (char >= 'A' && char <= 'Z') ||
				(char >= 'a' && char <= 'z') ||
				(char >= '0' && char <= '9') ||
				char == '-' || char == '_'
			assert.True(t, valid, "Secret contains non-URL-safe character: %c", char)
		}
	})
}

func TestSecretManager_GenerateOrRetrieveSecrets(t *testing.T) {
	credService := setupTestCredentialService(t)
	sm := NewSecretManager(credService)

	deploymentID := uuid.New()

	t.Run("Auto-generate secrets for type=secret fields", func(t *testing.T) {
		recipe := &models.Recipe{
			ConfigOptions: []models.RecipeConfigOption{
				{Name: "api_key", Type: "secret", Required: false},
				{Name: "auth_token", Type: "api_key", Required: false},
				{Name: "regular_field", Type: "string", Required: false},
			},
		}

		userConfig := map[string]interface{}{
			"regular_field": "value",
		}

		secrets, err := sm.GenerateOrRetrieveSecrets(deploymentID, recipe, userConfig)
		require.NoError(t, err)

		// Should generate secrets for api_key and auth_token
		assert.Contains(t, secrets, "api_key")
		assert.Contains(t, secrets, "auth_token")
		assert.Len(t, secrets["api_key"], 32)
		assert.Len(t, secrets["auth_token"], 32)

		// Should not generate for regular_field
		assert.NotContains(t, secrets, "regular_field")
	})

	t.Run("Don't generate secrets if user provided them", func(t *testing.T) {
		recipe := &models.Recipe{
			ConfigOptions: []models.RecipeConfigOption{
				{Name: "api_key", Type: "secret", Required: false},
			},
		}

		userConfig := map[string]interface{}{
			"api_key": "user-provided-key",
		}

		secrets, err := sm.GenerateOrRetrieveSecrets(deploymentID, recipe, userConfig)
		require.NoError(t, err)

		// Should not generate since user provided it
		assert.NotContains(t, secrets, "api_key")
	})

	t.Run("Secrets persist across redeployments", func(t *testing.T) {
		recipe := &models.Recipe{
			ConfigOptions: []models.RecipeConfigOption{
				{Name: "api_key", Type: "secret", Required: false},
			},
		}

		userConfig := map[string]interface{}{}

		// First deployment
		secrets1, err := sm.GenerateOrRetrieveSecrets(deploymentID, recipe, userConfig)
		require.NoError(t, err)
		apiKey1 := secrets1["api_key"]
		assert.NotEmpty(t, apiKey1)

		// Second deployment with same ID should get same secret
		secrets2, err := sm.GenerateOrRetrieveSecrets(deploymentID, recipe, userConfig)
		require.NoError(t, err)
		apiKey2 := secrets2["api_key"]

		assert.Equal(t, apiKey1, apiKey2, "Secret should persist across redeployments")
	})

	t.Run("No hardcoded admin_token generation", func(t *testing.T) {
		recipe := &models.Recipe{
			ConfigOptions: []models.RecipeConfigOption{
				{Name: "some_field", Type: "string", Required: false},
			},
		}

		userConfig := map[string]interface{}{}

		secrets, err := sm.GenerateOrRetrieveSecrets(deploymentID, recipe, userConfig)
		require.NoError(t, err)

		// Should not auto-generate admin_token unless it's in recipe config_options
		assert.NotContains(t, secrets, "admin_token")
		assert.Empty(t, secrets)
	})

	t.Run("Generate admin_token only if defined in recipe", func(t *testing.T) {
		recipe := &models.Recipe{
			ConfigOptions: []models.RecipeConfigOption{
				{Name: "admin_token", Type: "secret", Required: false},
			},
		}

		userConfig := map[string]interface{}{}

		secrets, err := sm.GenerateOrRetrieveSecrets(deploymentID, recipe, userConfig)
		require.NoError(t, err)

		// Should generate admin_token because it's in recipe config_options
		assert.Contains(t, secrets, "admin_token")
		assert.Len(t, secrets["admin_token"], 32)
	})
}

func TestSecretManager_HashPasswordForBasicAuth(t *testing.T) {
	credService := setupTestCredentialService(t)
	sm := NewSecretManager(credService)

	t.Run("Generate valid htpasswd hash", func(t *testing.T) {
		hash, err := sm.HashPasswordForBasicAuth("admin", "password123")
		require.NoError(t, err)

		// Should be in format: username:$2a$...
		assert.Contains(t, hash, "admin:")
		assert.Contains(t, hash, "$2a$")
		assert.Greater(t, len(hash), 60, "bcrypt hash should be long")
	})

	t.Run("Different salts for same password", func(t *testing.T) {
		hash1, err := sm.HashPasswordForBasicAuth("admin", "password123")
		require.NoError(t, err)

		hash2, err := sm.HashPasswordForBasicAuth("admin", "password123")
		require.NoError(t, err)

		// bcrypt uses random salt, so hashes should differ
		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("Different usernames produce different hashes", func(t *testing.T) {
		hash1, err := sm.HashPasswordForBasicAuth("admin", "password123")
		require.NoError(t, err)

		hash2, err := sm.HashPasswordForBasicAuth("user", "password123")
		require.NoError(t, err)

		assert.Contains(t, hash1, "admin:")
		assert.Contains(t, hash2, "user:")
	})
}

func TestSecretManager_ProcessPasswordHashing(t *testing.T) {
	credService := setupTestCredentialService(t)
	sm := NewSecretManager(credService)

	t.Run("Create hash for password fields", func(t *testing.T) {
		recipe := &models.Recipe{
			ConfigOptions: []models.RecipeConfigOption{
				{Name: "dashboard_username", Type: "string"},
				{Name: "dashboard_password", Type: "password"},
			},
		}

		envMap := map[string]string{
			"DASHBOARD_USERNAME": "admin",
			"DASHBOARD_PASSWORD": "secret123",
		}

		err := sm.ProcessPasswordHashing(recipe, envMap)
		require.NoError(t, err)

		// Should create _HASH field
		assert.Contains(t, envMap, "DASHBOARD_PASSWORD_HASH")
		assert.Contains(t, envMap["DASHBOARD_PASSWORD_HASH"], "admin:")
		assert.Contains(t, envMap["DASHBOARD_PASSWORD_HASH"], "$2a$")

		// Original password should still exist
		assert.Equal(t, "secret123", envMap["DASHBOARD_PASSWORD"])
	})

	t.Run("Use default username if not provided", func(t *testing.T) {
		recipe := &models.Recipe{
			ConfigOptions: []models.RecipeConfigOption{
				{Name: "api_password", Type: "password"},
			},
		}

		envMap := map[string]string{
			"API_PASSWORD": "secret123",
		}

		err := sm.ProcessPasswordHashing(recipe, envMap)
		require.NoError(t, err)

		// Should use default username "admin"
		assert.Contains(t, envMap, "API_PASSWORD_HASH")
		assert.Contains(t, envMap["API_PASSWORD_HASH"], "admin:")
	})

	t.Run("Only hash fields marked as type=password", func(t *testing.T) {
		recipe := &models.Recipe{
			ConfigOptions: []models.RecipeConfigOption{
				{Name: "regular_field", Type: "string"},
				{Name: "secret_field", Type: "secret"},
			},
		}

		envMap := map[string]string{
			"REGULAR_FIELD": "value1",
			"SECRET_FIELD":  "value2",
		}

		err := sm.ProcessPasswordHashing(recipe, envMap)
		require.NoError(t, err)

		// Should not create hashes for non-password fields
		assert.NotContains(t, envMap, "REGULAR_FIELD_HASH")
		assert.NotContains(t, envMap, "SECRET_FIELD_HASH")
	})

	t.Run("Skip empty passwords", func(t *testing.T) {
		recipe := &models.Recipe{
			ConfigOptions: []models.RecipeConfigOption{
				{Name: "dashboard_password", Type: "password"},
			},
		}

		envMap := map[string]string{
			"DASHBOARD_PASSWORD": "",
		}

		err := sm.ProcessPasswordHashing(recipe, envMap)
		require.NoError(t, err)

		// Should not create hash for empty password
		assert.NotContains(t, envMap, "DASHBOARD_PASSWORD_HASH")
	})
}
