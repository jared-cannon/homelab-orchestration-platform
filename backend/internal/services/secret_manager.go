package services

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// SecretManager handles generation, storage, and retrieval of deployment secrets
type SecretManager struct {
	credService *CredentialService
}

// NewSecretManager creates a new secret manager
func NewSecretManager(credService *CredentialService) *SecretManager {
	return &SecretManager{
		credService: credService,
	}
}

// GenerateOrRetrieveSecrets generates secrets for a deployment or retrieves existing ones
// This ensures secrets persist across redeployments
// Only generates secrets for fields defined in recipe config_options with type="secret" or "api_key"
func (sm *SecretManager) GenerateOrRetrieveSecrets(
	deploymentID uuid.UUID,
	recipe *models.Recipe,
	userConfig map[string]interface{},
) (map[string]string, error) {
	secrets := make(map[string]string)

	// Process each config option that needs secret generation
	for _, option := range recipe.ConfigOptions {
		// Generate secrets for fields marked as secret/api_key that aren't provided by user
		if option.Type == "secret" || option.Type == "api_key" {
			if _, exists := userConfig[option.Name]; !exists {
				secret, err := sm.getOrGenerateSecret(deploymentID, option.Name, 32)
				if err != nil {
					return nil, fmt.Errorf("failed to generate secret for %s: %w", option.Name, err)
				}
				secrets[option.Name] = secret
			}
		}
	}

	return secrets, nil
}

// getOrGenerateSecret retrieves an existing secret or generates a new one
func (sm *SecretManager) getOrGenerateSecret(deploymentID uuid.UUID, fieldName string, length int) (string, error) {
	credentialKey := fmt.Sprintf("deployment:%s:%s", deploymentID.String(), fieldName)

	// Try to retrieve existing secret
	existing, err := sm.credService.GetCredential(credentialKey)
	if err == nil && existing != "" {
		return existing, nil
	}

	// Generate new secret
	secret, err := sm.generateSecret(length)
	if err != nil {
		return "", err
	}

	// Store for future retrievals
	if err := sm.credService.StoreCredential(credentialKey, secret); err != nil {
		return "", fmt.Errorf("failed to store secret: %w", err)
	}

	return secret, nil
}

// generateSecret generates a cryptographically secure random secret of exact length
// Uses URL-safe base64 encoding for compatibility with various systems
func (sm *SecretManager) generateSecret(length int) (string, error) {
	// Calculate bytes needed for target length after base64 encoding
	// Base64 encoding expands by ~4/3, so we need length * 3/4 bytes
	// Add 1 to handle rounding and ensure we have enough
	bytesNeeded := ((length * 3) / 4) + 1
	if bytesNeeded < 1 {
		bytesNeeded = length // Fallback for very small lengths
	}

	bytes := make([]byte, bytesNeeded)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Encode to URL-safe base64 and truncate to exact length
	encoded := base64.URLEncoding.EncodeToString(bytes)
	if len(encoded) < length {
		// If somehow we don't have enough, generate more bytes
		bytes = make([]byte, length)
		if _, err := rand.Read(bytes); err != nil {
			return "", fmt.Errorf("failed to generate random bytes: %w", err)
		}
		encoded = base64.URLEncoding.EncodeToString(bytes)
	}

	return encoded[:length], nil
}

// HashPasswordForBasicAuth creates htpasswd-compatible bcrypt hash for HTTP basic auth
// Returns format: username:$2a$hash (compatible with Traefik basicauth middleware)
func (sm *SecretManager) HashPasswordForBasicAuth(username, password string) (string, error) {
	// Generate bcrypt hash with cost 10 (good balance of security and performance)
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	// Return in htpasswd format: username:$2y$hash
	return fmt.Sprintf("%s:%s", username, string(hash)), nil
}

// ProcessPasswordHashing looks for password fields and creates corresponding _HASH fields
// Only processes fields with type="password" in the recipe config_options
func (sm *SecretManager) ProcessPasswordHashing(recipe *models.Recipe, envMap map[string]string) error {
	// Look for password config options that need hashing
	for _, option := range recipe.ConfigOptions {
		// Only hash fields explicitly marked as password type
		if option.Type == "password" {
			passwordKey := toEnvVarName(option.Name)
			hashKey := passwordKey + "_HASH"

			// If password exists and hash doesn't, create the hash
			if password, exists := envMap[passwordKey]; exists && password != "" {
				// Look for corresponding username field
				usernameFieldName := findUsernameField(option.Name)
				usernameKey := toEnvVarName(usernameFieldName)

				username, hasUsername := envMap[usernameKey]
				if !hasUsername {
					username = "admin" // Default username
				}

				hash, err := sm.HashPasswordForBasicAuth(username, password)
				if err != nil {
					return fmt.Errorf("failed to hash %s: %w", passwordKey, err)
				}
				envMap[hashKey] = hash
			}
		}
	}

	return nil
}

// toEnvVarName converts a field name to environment variable format (uppercase)
func toEnvVarName(name string) string {
	return strings.ToUpper(name)
}

// findUsernameField finds the corresponding username field for a password field
// e.g., "dashboard_password" -> "dashboard_username"
func findUsernameField(passwordFieldName string) string {
	// Replace "password" suffix with "username"
	lowerName := strings.ToLower(passwordFieldName)
	if strings.HasSuffix(lowerName, "_password") {
		return strings.TrimSuffix(lowerName, "_password") + "_username"
	}
	if lowerName == "password" {
		return "username"
	}
	return "username" // fallback
}
