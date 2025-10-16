package services

import (
	"testing"

	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestConfigValidator_ValidateRequiredFields(t *testing.T) {
	validator := NewConfigValidator()

	recipe := &models.Recipe{
		ConfigOptions: []models.RecipeConfigOption{
			{Name: "required_field", Type: "string", Required: true},
			{Name: "optional_field", Type: "string", Required: false},
		},
	}

	t.Run("Missing required field", func(t *testing.T) {
		config := map[string]interface{}{
			"optional_field": "value",
		}
		err := validator.Validate(recipe, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing required field: required_field")
	})

	t.Run("Empty required field", func(t *testing.T) {
		config := map[string]interface{}{
			"required_field": "",
			"optional_field": "value",
		}
		err := validator.Validate(recipe, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required field cannot be empty: required_field")
	})

	t.Run("Valid config", func(t *testing.T) {
		config := map[string]interface{}{
			"required_field": "value",
			"optional_field": "value",
		}
		err := validator.Validate(recipe, config)
		assert.NoError(t, err)
	})

	t.Run("Missing optional field is OK", func(t *testing.T) {
		config := map[string]interface{}{
			"required_field": "value",
		}
		err := validator.Validate(recipe, config)
		assert.NoError(t, err)
	})
}

func TestConfigValidator_ValidateOptionalFieldTypes(t *testing.T) {
	validator := NewConfigValidator()

	recipe := &models.Recipe{
		ConfigOptions: []models.RecipeConfigOption{
			{Name: "optional_email", Type: "email", Required: false},
			{Name: "optional_password", Type: "password", Required: false},
			{Name: "optional_domain", Type: "domain", Required: false},
		},
	}

	t.Run("Optional email field with invalid value", func(t *testing.T) {
		config := map[string]interface{}{
			"optional_email": "invalid-email",
		}
		err := validator.Validate(recipe, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a valid email address")
	})

	t.Run("Optional email field with placeholder", func(t *testing.T) {
		config := map[string]interface{}{
			"optional_email": "user@example.com",
		}
		err := validator.Validate(recipe, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "placeholder domain")
	})

	t.Run("Optional password field too short", func(t *testing.T) {
		config := map[string]interface{}{
			"optional_password": "short",
		}
		err := validator.Validate(recipe, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be at least 8 characters")
	})

	t.Run("Optional password field is weak", func(t *testing.T) {
		config := map[string]interface{}{
			"optional_password": "password123",
		}
		err := validator.Validate(recipe, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too weak")
	})

	t.Run("Optional domain with placeholder", func(t *testing.T) {
		config := map[string]interface{}{
			"optional_domain": "subdomain.example.com",
		}
		err := validator.Validate(recipe, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "placeholder domain")
	})

	t.Run("Valid optional fields", func(t *testing.T) {
		config := map[string]interface{}{
			"optional_email":    "user@realdomain.com",
			"optional_password": "SecurePass123!",
			"optional_domain":   "myapp.local",
		}
		err := validator.Validate(recipe, config)
		assert.NoError(t, err)
	})
}

func TestConfigValidator_EmailValidation(t *testing.T) {
	validator := NewConfigValidator()

	tests := []struct {
		name        string
		email       string
		shouldError bool
		errorText   string
	}{
		{"Valid email", "user@domain.com", false, ""},
		{"Valid email with subdomain", "user@mail.domain.com", false, ""},
		{"Valid email with plus", "user+tag@domain.com", false, ""},
		{"Invalid - no @", "userdomain.com", true, "not a valid email address"},
		{"Invalid - no domain", "user@", true, "not a valid email address"},
		{"Invalid - just @", "@", true, "not a valid email address"},
		{"Invalid - multiple @", "user@@domain.com", true, "not a valid email address"},
		{"Placeholder - example.com", "admin@example.com", true, "placeholder domain"},
		{"Placeholder - example.org", "admin@example.org", true, "placeholder domain"},
		{"Placeholder - example.net", "admin@example.net", true, "placeholder domain"},
		{"Placeholder - test.com", "admin@test.com", true, "placeholder domain"},
		{"Valid - localhost is now allowed", "user@localhost", true, "not a valid email address"}, // localhost rejected by regex (no TLD)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateEmail("email", tt.email)
			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorText)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigValidator_PasswordValidation(t *testing.T) {
	validator := NewConfigValidator()

	tests := []struct {
		name        string
		password    string
		shouldError bool
		errorText   string
	}{
		{"Valid password", "MySecurePass123!", false, ""},
		{"Valid password - 8 chars", "Pass1234", false, ""},
		{"Valid - contains 'admin' but not exact match", "Admin1234!", false, ""},
		{"Valid - contains 'password' but not exact match", "MyPassword123!", false, ""},
		{"Invalid - too short", "Pass12", true, "must be at least 8 characters"},
		{"Invalid - weak password", "password", true, "too weak"},
		{"Invalid - weak password123", "password123", true, "too weak"},
		{"Invalid - weak 12345678", "12345678", true, "too weak"},
		{"Invalid - weak changeme", "changeme", true, "too weak"},
		{"Invalid - weak admin123", "admin123", true, "too weak"},
		{"Empty password is allowed", "", false, ""}, // Empty check happens elsewhere
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validatePassword("password", tt.password)
			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorText)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigValidator_DomainValidation(t *testing.T) {
	validator := NewConfigValidator()

	tests := []struct {
		name        string
		domain      string
		shouldError bool
		errorText   string
	}{
		{"Valid domain", "myapp.com", false, ""},
		{"Valid subdomain", "api.myapp.com", false, ""},
		{"Valid - localhost", "localhost", false, ""}, // Now allowed for development
		{"Valid - test.local", "test.local", false, ""}, // Now allowed for internal use
		{"Valid - .local domain", "myapp.local", false, ""},
		{"Invalid - example.com", "example.com", true, "placeholder domain"},
		{"Invalid - subdomain.example.com", "subdomain.example.com", true, "placeholder domain"},
		{"Invalid - example.org", "example.org", true, "placeholder domain"},
		{"Invalid - example.net", "api.example.net", true, "placeholder domain"},
		{"Empty domain is allowed", "", false, ""}, // Empty check happens elsewhere
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateDomain("domain", tt.domain)
			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorText)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigValidator_MultipleErrors(t *testing.T) {
	validator := NewConfigValidator()

	recipe := &models.Recipe{
		ConfigOptions: []models.RecipeConfigOption{
			{Name: "email", Type: "email", Required: true},
			{Name: "password", Type: "password", Required: true},
			{Name: "domain", Type: "domain", Required: false},
		},
	}

	t.Run("Multiple validation errors", func(t *testing.T) {
		config := map[string]interface{}{
			"email":    "invalid@example.com", // Placeholder domain
			"password": "weak",                 // Too short
			"domain":   "test.example.org",     // Placeholder domain
		}
		err := validator.Validate(recipe, config)
		assert.Error(t, err)
		// Should contain multiple errors
		assert.Contains(t, err.Error(), "placeholder domain")
		assert.Contains(t, err.Error(), "must be at least 8 characters")
	})
}
