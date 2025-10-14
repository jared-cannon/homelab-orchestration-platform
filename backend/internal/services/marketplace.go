package services

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/google/uuid"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/models"
	"gorm.io/gorm"
)

// MarketplaceService handles marketplace operations
type MarketplaceService struct {
	db                *gorm.DB
	recipeLoader      *RecipeLoader
	deviceService     *DeviceService
	validator         *ValidatorService
	resourceValidator *ResourceValidator
}

// NewMarketplaceService creates a new marketplace service
func NewMarketplaceService(db *gorm.DB, recipeLoader *RecipeLoader, deviceService *DeviceService, validator *ValidatorService, resourceValidator *ResourceValidator) *MarketplaceService {
	return &MarketplaceService{
		db:                db,
		recipeLoader:      recipeLoader,
		deviceService:     deviceService,
		validator:         validator,
		resourceValidator: resourceValidator,
	}
}

// ListRecipes returns all available recipes, optionally filtered by category
func (s *MarketplaceService) ListRecipes(category string) ([]*models.Recipe, error) {
	if category == "" {
		return s.recipeLoader.ListRecipes(), nil
	}
	return s.recipeLoader.ListRecipesByCategory(category), nil
}

// GetRecipe retrieves a single recipe by slug
func (s *MarketplaceService) GetRecipe(slug string) (*models.Recipe, error) {
	return s.recipeLoader.GetRecipe(slug)
}

// ValidationResult contains the result of deployment validation
type ValidationResult struct {
	Valid               bool                  `json:"valid"`
	Errors              []string              `json:"errors,omitempty"`
	Warnings            []string              `json:"warnings,omitempty"`
	ResourceCheck       *ResourceCheck        `json:"resource_check,omitempty"`
	PortConflicts       []int                 `json:"port_conflicts,omitempty"`
	RenderedCompose     string                `json:"rendered_compose,omitempty"` // Preview of what will be deployed
}

// ResourceCheck contains resource availability information
type ResourceCheck struct {
	RequiredRAMMB      int  `json:"required_ram_mb"`
	AvailableRAMMB     int  `json:"available_ram_mb"`
	RAMSufficient      bool `json:"ram_sufficient"`
	RequiredStorageGB  int  `json:"required_storage_gb"`
	AvailableStorageGB int  `json:"available_storage_gb"`
	StorageSufficient  bool `json:"storage_sufficient"`
	DockerInstalled    bool `json:"docker_installed"`
	DockerRunning      bool `json:"docker_running"`
}

// ValidateDeployment validates that a deployment can proceed
func (s *MarketplaceService) ValidateDeployment(recipeSlug string, deviceID uuid.UUID, config map[string]interface{}) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Get recipe
	recipe, err := s.recipeLoader.GetRecipe(recipeSlug)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Recipe not found: %s", recipeSlug))
		return result, nil
	}

	// Get device
	device, err := s.deviceService.GetDevice(deviceID)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Device not found: %s", deviceID))
		return result, nil
	}

	// Check device is online
	if device.Status != models.DeviceStatusOnline {
		result.Valid = false
		result.Errors = append(result.Errors, "Device is not online")
		return result, nil
	}

	// Validate all required config options are provided
	for _, opt := range recipe.ConfigOptions {
		if opt.Required {
			if _, exists := config[opt.Name]; !exists {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Missing required configuration: %s", opt.Label))
			}
		}
	}

	// Check resources via SSH
	host := device.IPAddress + ":22"

	// Check Docker installation and status
	dockerInstalled, _, err := s.validator.DockerInstalled(host)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, "Docker is not installed on this device")
	}

	dockerRunning := false
	if dockerInstalled {
		dockerRunning, err = s.validator.DockerRunning(host)
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, "Docker daemon is not running on this device")
		}
	}

	// Extract required ports from config
	requiredPorts := ExtractPortsFromConfig(config)

	// Validate resources using ResourceValidator
	resourceValidation, err := s.resourceValidator.ValidateResourceRequirements(
		device,
		recipe.Resources.MinRAMMB,
		recipe.Resources.MinStorageGB,
		recipe.Resources.CPUCores,
		requiredPorts,
	)

	if err != nil {
		// If resource checking fails, log warning but don't fail validation
		result.Warnings = append(result.Warnings, fmt.Sprintf("Could not verify resources: %v", err))

		// Use conservative defaults
		result.ResourceCheck = &ResourceCheck{
			RequiredRAMMB:      recipe.Resources.MinRAMMB,
			RequiredStorageGB:  recipe.Resources.MinStorageGB,
			DockerInstalled:    dockerInstalled,
			DockerRunning:      dockerRunning,
			RAMSufficient:      true,
			StorageSufficient:  true,
			AvailableRAMMB:     0,
			AvailableStorageGB: 0,
		}
	} else {
		// Use actual resource check results
		result.ResourceCheck = &ResourceCheck{
			RequiredRAMMB:      recipe.Resources.MinRAMMB,
			AvailableRAMMB:     resourceValidation.DeviceResources.AvailableRAMMB,
			RAMSufficient:      resourceValidation.RAMSufficient,
			RequiredStorageGB:  recipe.Resources.MinStorageGB,
			AvailableStorageGB: resourceValidation.DeviceResources.AvailableStorageGB,
			StorageSufficient:  resourceValidation.StorageSufficient,
			DockerInstalled:    dockerInstalled,
			DockerRunning:      dockerRunning,
		}

		// Add validation errors if resources insufficient
		if !resourceValidation.RAMSufficient {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf(
				"Insufficient RAM: need %d MB, only %d MB available",
				recipe.Resources.MinRAMMB,
				resourceValidation.DeviceResources.AvailableRAMMB,
			))
		}

		if !resourceValidation.StorageSufficient {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf(
				"Insufficient storage: need %d GB, only %d GB available",
				recipe.Resources.MinStorageGB,
				resourceValidation.DeviceResources.AvailableStorageGB,
			))
		}

		if !resourceValidation.PortsAvailable {
			result.Valid = false
			result.PortConflicts = resourceValidation.PortConflicts
			result.Errors = append(result.Errors, fmt.Sprintf(
				"Port conflicts detected: ports %v are already in use",
				resourceValidation.PortConflicts,
			))
		}
	}

	// Render compose template to validate and preview
	renderedCompose, err := s.renderComposePreview(recipe, config, device)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Template rendering failed: %v", err))
	} else {
		result.RenderedCompose = renderedCompose
	}

	return result, nil
}

// GetCategories returns all unique recipe categories
func (s *MarketplaceService) GetCategories() []string {
	recipes := s.recipeLoader.ListRecipes()
	categoryMap := make(map[string]bool)

	for _, recipe := range recipes {
		categoryMap[recipe.Category] = true
	}

	categories := make([]string, 0, len(categoryMap))
	for category := range categoryMap {
		categories = append(categories, category)
	}

	return categories
}

// ReloadRecipes reloads all recipes from disk (useful for development)
func (s *MarketplaceService) ReloadRecipes() error {
	return s.recipeLoader.Reload()
}

// renderComposePreview renders the Docker Compose template for preview/validation
func (s *MarketplaceService) renderComposePreview(recipe *models.Recipe, config map[string]interface{}, device *models.Device) (string, error) {
	// Import deployment service's template rendering logic
	// We'll use a simplified version here for preview

	// Normalize config keys: Convert snake_case to PascalCase for Go templates
	normalizedConfig := make(map[string]interface{})
	for key, value := range config {
		// Convert snake_case to PascalCase
		pascalKey := snakeToPascalCase(key)
		normalizedConfig[pascalKey] = value
		// Also keep original key for backwards compatibility
		normalizedConfig[key] = value
	}

	// Add preview-specific variables
	normalizedConfig["DEPLOYMENT_ID"] = "preview"
	normalizedConfig["COMPOSE_PROJECT"] = "preview-" + recipe.Slug

	// Parse and execute template
	tmpl, err := template.New("compose").Parse(recipe.ComposeTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, normalizedConfig); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// snakeToPascalCase converts snake_case to PascalCase
func snakeToPascalCase(input string) string {
	parts := strings.Split(input, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// CheckForUpdates checks all recipe sources for available updates
func (s *MarketplaceService) CheckForUpdates() (map[string][]string, error) {
	return s.recipeLoader.CheckForUpdates()
}
