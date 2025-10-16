package services

import (
	"fmt"
	"strings"

	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"gopkg.in/yaml.v3"
)

// RecipeValidator validates recipe structure and consistency
type RecipeValidator struct{}

// NewRecipeValidator creates a new recipe validator
func NewRecipeValidator() *RecipeValidator {
	return &RecipeValidator{}
}

// Validate performs comprehensive validation on a recipe
func (rv *RecipeValidator) Validate(recipe *models.Recipe) error {
	var errors []string

	// Basic field validation
	if recipe.Slug == "" {
		errors = append(errors, "slug is required")
	}
	if recipe.Name == "" {
		errors = append(errors, "name is required")
	}

	// Validate config options
	if err := rv.validateConfigOptions(recipe); err != nil {
		errors = append(errors, err.Error())
	}

	// Validate docker-compose content exists
	if recipe.ComposeContent == "" {
		errors = append(errors, "compose_content (docker-compose.yaml) is required")
	}

	// If docker-compose content exists, validate it references declared volumes
	if recipe.ComposeContent != "" {
		if err := rv.validateComposeVolumes(recipe); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("recipe validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// validateConfigOptions validates recipe config options
func (rv *RecipeValidator) validateConfigOptions(recipe *models.Recipe) error {
	seen := make(map[string]bool)

	for i, option := range recipe.ConfigOptions {
		if option.Name == "" {
			return fmt.Errorf("config_option[%d]: name is required", i)
		}

		// Check for duplicates
		if seen[option.Name] {
			return fmt.Errorf("duplicate config option: %s", option.Name)
		}
		seen[option.Name] = true

		// Validate type is recognized
		validTypes := map[string]bool{
			"string":   true,
			"number":   true,
			"boolean":  true,
			"password": true,
			"email":    true,
			"secret":   true,
			"api_key":  true,
			"domain":   true,
			"hostname": true,
		}

		if !validTypes[option.Type] {
			return fmt.Errorf("config option '%s' has invalid type '%s'", option.Name, option.Type)
		}

		// Email fields should not have placeholder defaults
		if option.Type == "email" && option.Required {
			if defaultEmail, ok := option.Default.(string); ok {
				if strings.Contains(strings.ToLower(defaultEmail), "example.com") {
					return fmt.Errorf("config option '%s': email field should not default to example.com (use empty string instead)", option.Name)
				}
			}
		}
	}

	return nil
}

// validateComposeVolumes checks that volumes declared in manifest exist in docker-compose
func (rv *RecipeValidator) validateComposeVolumes(recipe *models.Recipe) error {
	if len(recipe.Volumes) == 0 {
		return nil // No volumes declared, nothing to validate
	}

	// Parse docker-compose YAML
	var compose map[string]interface{}
	if err := yaml.Unmarshal([]byte(recipe.ComposeContent), &compose); err != nil {
		return fmt.Errorf("failed to parse docker-compose.yaml: %w", err)
	}

	// Extract volume names from compose
	composeVolumes := make(map[string]bool)
	if volumesSection, ok := compose["volumes"].(map[string]interface{}); ok {
		for volumeName := range volumesSection {
			composeVolumes[volumeName] = true
		}
	}

	// Also check for volumes used in services
	usedVolumes := make(map[string]bool)
	if services, ok := compose["services"].(map[string]interface{}); ok {
		for _, service := range services {
			if serviceMap, ok := service.(map[string]interface{}); ok {
				if volumesList, ok := serviceMap["volumes"].([]interface{}); ok {
					for _, vol := range volumesList {
						if volStr, ok := vol.(string); ok {
							// Extract volume name (before the :)
							parts := strings.Split(volStr, ":")
							if len(parts) > 0 && !strings.HasPrefix(parts[0], "/") && !strings.HasPrefix(parts[0], ".") {
								// Named volume (not a bind mount)
								usedVolumes[parts[0]] = true
							}
						}
					}
				}
			}
		}
	}

	// Check that volumes in manifest are declared in compose
	var errors []string
	for volumeName := range recipe.Volumes {
		if !composeVolumes[volumeName] {
			errors = append(errors, fmt.Sprintf("volume '%s' declared in manifest but not found in docker-compose volumes section", volumeName))
		}
	}

	// Warn about volumes used but not documented in manifest
	for volumeName := range usedVolumes {
		if _, documented := recipe.Volumes[volumeName]; !documented {
			// This is a warning, not an error - volumes might be intentionally undocumented
			// errors = append(errors, fmt.Sprintf("volume '%s' used in docker-compose but not documented in manifest", volumeName))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("volume validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

