package services

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/models"
	"gorm.io/gorm"
)

// MarketplaceService handles marketplace operations
type MarketplaceService struct {
	db            *gorm.DB
	recipeLoader  *RecipeLoader
	deviceService *DeviceService
	validator     *ValidatorService
}

// NewMarketplaceService creates a new marketplace service
func NewMarketplaceService(db *gorm.DB, recipeLoader *RecipeLoader, deviceService *DeviceService, validator *ValidatorService) *MarketplaceService {
	return &MarketplaceService{
		db:            db,
		recipeLoader:  recipeLoader,
		deviceService: deviceService,
		validator:     validator,
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

	// TODO: Implement actual RAM and storage checking
	result.ResourceCheck = &ResourceCheck{
		RequiredRAMMB:      recipe.Resources.MinRAMMB,
		RequiredStorageGB:  recipe.Resources.MinStorageGB,
		DockerInstalled:    dockerInstalled,
		DockerRunning:      dockerRunning,
		RAMSufficient:      true, // TODO: Check actual RAM
		StorageSufficient:  true, // TODO: Check actual storage
		AvailableRAMMB:     4096, // Placeholder
		AvailableStorageGB: 100,  // Placeholder
	}

	// Check for port conflicts
	// TODO: Implement port conflict detection
	if internalPort, ok := config["internal_port"].(int); ok {
		_ = internalPort // TODO: Check if port is already in use
	}

	// Render compose template to validate
	// TODO: Implement template rendering
	// result.RenderedCompose = renderedTemplate

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
