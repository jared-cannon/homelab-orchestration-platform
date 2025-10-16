package services

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
)

var (
	// emailRegex is a reasonable email validation pattern
	// Not RFC 5322 compliant but good enough for practical use
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

// ConfigValidator validates user configuration against recipe requirements
type ConfigValidator struct{}

// NewConfigValidator creates a new config validator
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

// Validate validates user config against recipe config options
func (cv *ConfigValidator) Validate(recipe *models.Recipe, config map[string]interface{}) error {
	var errors []string

	// Validate all config options
	for _, option := range recipe.ConfigOptions {
		value, exists := config[option.Name]

		// Check required fields are provided
		if !exists {
			if option.Required {
				errors = append(errors, fmt.Sprintf("missing required field: %s", option.Name))
			}
			continue
		}

		// Check for empty string values on required fields
		if option.Required {
			if strValue, ok := value.(string); ok && strValue == "" {
				errors = append(errors, fmt.Sprintf("required field cannot be empty: %s", option.Name))
				continue
			}
		}

		// Type-specific validation for ALL provided fields (required or optional)
		if err := cv.validateFieldByType(option, value); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// validateFieldByType performs type-specific validation
func (cv *ConfigValidator) validateFieldByType(option models.RecipeConfigOption, value interface{}) error {
	strValue, ok := value.(string)
	if !ok {
		return nil // Only validate string fields for now
	}

	switch option.Type {
	case "email":
		return cv.validateEmail(option.Name, strValue)
	case "password":
		return cv.validatePassword(option.Name, strValue)
	case "domain", "hostname":
		return cv.validateDomain(option.Name, strValue)
	}

	return nil
}

// validateEmail ensures email addresses are valid and not placeholders
func (cv *ConfigValidator) validateEmail(fieldName, email string) error {
	if email == "" {
		return nil // Empty check happens in required validation
	}

	// Validate email format using regex
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("%s is not a valid email address", fieldName)
	}

	lowerEmail := strings.ToLower(email)

	// Check for RFC 2606 reserved placeholder domains
	placeholderDomains := []string{
		"example.com",
		"example.org",
		"example.net",
		"test.com",
	}

	for _, domain := range placeholderDomains {
		if strings.HasSuffix(lowerEmail, "@"+domain) {
			return fmt.Errorf("%s contains placeholder domain %s - please use a real email address", fieldName, domain)
		}
	}

	return nil
}

// validatePassword ensures passwords meet basic requirements
func (cv *ConfigValidator) validatePassword(fieldName, password string) error {
	if password == "" {
		return nil // Empty check happens in required validation
	}

	// Minimum length check first
	if len(password) < 8 {
		return fmt.Errorf("%s must be at least 8 characters long", fieldName)
	}

	// Check for extremely weak/common placeholder passwords
	// Only reject exact matches of very weak passwords
	lowerPassword := strings.ToLower(password)
	weakPasswords := []string{
		"password",
		"password123",
		"12345678",
		"changeme",
		"letmein",
		"qwerty",
		"admin123",
	}

	for _, weak := range weakPasswords {
		if lowerPassword == weak {
			return fmt.Errorf("%s is too weak - please use a more secure password", fieldName)
		}
	}

	return nil
}

// validateDomain ensures domain names are not RFC 2606 reserved placeholders
func (cv *ConfigValidator) validateDomain(fieldName, domain string) error {
	if domain == "" {
		return nil // Empty check happens in required validation
	}

	lowerDomain := strings.ToLower(domain)

	// Only reject RFC 2606 reserved documentation domains
	// localhost and test.local are legitimate for internal/development use
	reservedDomains := []string{
		"example.com",
		"example.org",
		"example.net",
	}

	for _, reserved := range reservedDomains {
		if strings.Contains(lowerDomain, reserved) {
			return fmt.Errorf("%s contains placeholder domain '%s' - please use your actual domain", fieldName, reserved)
		}
	}

	return nil
}
