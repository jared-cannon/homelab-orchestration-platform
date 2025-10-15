package services

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/jaredcannon/homelab-orchestration-platform/internal/models"
)

// EnvironmentBuilder builds environment variable maps for docker-compose deployment
// Combines user config + generated secrets + database credentials
type EnvironmentBuilder struct {
	credService *CredentialService
	dbPoolManager *DatabasePoolManager
}

// NewEnvironmentBuilder creates a new environment builder
func NewEnvironmentBuilder(credService *CredentialService, dbPoolManager *DatabasePoolManager) *EnvironmentBuilder {
	return &EnvironmentBuilder{
		credService:   credService,
		dbPoolManager: dbPoolManager,
	}
}

// BuildEnvironment creates a complete environment variable map for deployment
// Returns both a map (for programmatic use) and a .env file content (for docker-compose)
func (eb *EnvironmentBuilder) BuildEnvironment(
	deployment *models.Deployment,
	recipe *models.Recipe,
	userConfig map[string]interface{},
	device *models.Device,
	provisionedDB *models.ProvisionedDatabase,
) (map[string]string, string, error) {

	envMap := make(map[string]string)

	// 1. Add user-provided configuration
	for key, value := range userConfig {
		envMap[strings.ToUpper(key)] = fmt.Sprintf("%v", value)
	}

	// 2. Add deployment-specific variables
	envMap["DEPLOYMENT_ID"] = deployment.ID.String()
	envMap["COMPOSE_PROJECT"] = deployment.ComposeProject
	envMap["DEVICE_IP"] = device.IPAddress

	// 3. Add database credentials if database was provisioned
	if provisionedDB != nil {
		dbEnvVars, err := eb.buildDatabaseEnvVars(provisionedDB, recipe.Database.EnvPrefix)
		if err != nil {
			return nil, "", fmt.Errorf("failed to build database env vars: %w", err)
		}
		for k, v := range dbEnvVars {
			envMap[k] = v
		}
	}

	// 4. Generate any required secrets (API keys, admin tokens, etc.)
	if err := eb.generateRequiredSecrets(recipe, deployment, envMap); err != nil {
		return nil, "", fmt.Errorf("failed to generate secrets: %w", err)
	}

	// 5. Build .env file content
	envFileContent := eb.buildEnvFileContent(envMap)

	return envMap, envFileContent, nil
}

// buildDatabaseEnvVars creates environment variables for database connection
func (eb *EnvironmentBuilder) buildDatabaseEnvVars(provisionedDB *models.ProvisionedDatabase, envPrefix string) (map[string]string, error) {
	if envPrefix == "" {
		envPrefix = "DB_"
	}

	// Get database password from credential store
	password, err := eb.credService.GetCredential(provisionedDB.CredentialKey)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve database password: %w", err)
	}

	envVars := provisionedDB.GetConnectionEnvVars(envPrefix)
	envVars[envPrefix+"PASSWORD"] = password

	// Also provide common alternative formats
	envVars[envPrefix+"CONNECTION_STRING"] = eb.buildConnectionString(provisionedDB, password)

	return envVars, nil
}

// buildConnectionString creates a database connection string
func (eb *EnvironmentBuilder) buildConnectionString(db *models.ProvisionedDatabase, password string) string {
	// Determine engine from shared instance
	if db.SharedDatabaseInstance == nil {
		return ""
	}

	switch db.SharedDatabaseInstance.Engine {
	case "postgres":
		return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s",
			db.Username, password, db.Host, db.Port, db.DatabaseName)
	case "mysql", "mariadb":
		return fmt.Sprintf("mysql://%s:%s@%s:%d/%s",
			db.Username, password, db.Host, db.Port, db.DatabaseName)
	default:
		return ""
	}
}

// generateRequiredSecrets generates any required secrets based on recipe config options
func (eb *EnvironmentBuilder) generateRequiredSecrets(recipe *models.Recipe, deployment *models.Deployment, envMap map[string]string) error {
	// Scan config options for secret types
	for _, option := range recipe.ConfigOptions {
		// If option type is "secret" or "password" and not provided, generate it
		if (option.Type == "secret" || option.Type == "password" || option.Type == "api_key") {
			envKey := strings.ToUpper(option.Name)
			if _, exists := envMap[envKey]; !exists {
				// Generate a secure random secret
				secret, err := eb.generateSecret(32)
				if err != nil {
					return fmt.Errorf("failed to generate secret for %s: %w", option.Name, err)
				}
				envMap[envKey] = secret
			}
		}
	}

	// Common secrets that apps might need
	// Generate ADMIN_TOKEN if not provided (for apps like Vaultwarden, Traefik, etc.)
	if _, exists := envMap["ADMIN_TOKEN"]; !exists {
		secret, err := eb.generateSecret(32)
		if err != nil {
			return err
		}
		envMap["ADMIN_TOKEN"] = secret
	}

	return nil
}

// generateSecret generates a cryptographically secure random secret
func (eb *EnvironmentBuilder) generateSecret(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// Use URL-safe base64 encoding
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// buildEnvFileContent creates the content for a .env file
func (eb *EnvironmentBuilder) buildEnvFileContent(envMap map[string]string) string {
	var builder strings.Builder
	builder.WriteString("# Auto-generated environment file\n")
	builder.WriteString("# Generated by Homelab Orchestration Platform\n\n")

	// Sort keys for consistent output (optional, for better debugging)
	for key, value := range envMap {
		// Escape values that contain special characters
		escapedValue := eb.escapeEnvValue(value)
		builder.WriteString(fmt.Sprintf("%s=%s\n", key, escapedValue))
	}

	return builder.String()
}

// escapeEnvValue escapes special characters in environment variable values
func (eb *EnvironmentBuilder) escapeEnvValue(value string) string {
	// If value contains spaces, quotes, or special characters, wrap in quotes
	needsQuotes := strings.ContainsAny(value, " \t\n\"'$`\\#")

	if !needsQuotes {
		return value
	}

	// Escape existing quotes
	escaped := strings.ReplaceAll(value, "\"", "\\\"")
	return fmt.Sprintf("\"%s\"", escaped)
}

// ValidateEnvironment checks that all required environment variables are present
func (eb *EnvironmentBuilder) ValidateEnvironment(recipe *models.Recipe, envMap map[string]string) error {
	var missing []string

	// Check required config options
	for _, option := range recipe.ConfigOptions {
		if option.Required {
			envKey := strings.ToUpper(option.Name)
			if _, exists := envMap[envKey]; !exists {
				missing = append(missing, option.Name)
			}
		}
	}

	// Check database requirements
	if recipe.Database.AutoProvision {
		dbPrefix := recipe.Database.EnvPrefix
		if dbPrefix == "" {
			dbPrefix = "DB_"
		}

		requiredDBKeys := []string{
			dbPrefix + "HOST",
			dbPrefix + "PORT",
			dbPrefix + "NAME",
			dbPrefix + "USER",
			dbPrefix + "PASSWORD",
		}

		for _, key := range requiredDBKeys {
			if _, exists := envMap[key]; !exists {
				missing = append(missing, key)
			}
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return nil
}
