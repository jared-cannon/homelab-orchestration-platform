package services

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
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
	host := device.GetSSHHost()

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

// renderComposePreview renders the Docker Compose content for preview/validation
func (s *MarketplaceService) renderComposePreview(recipe *models.Recipe, config map[string]interface{}, device *models.Device) (string, error) {
	// Start with docker-compose.yaml content
	content := recipe.ComposeContent

	// Build environment variable map for substitution
	envVars := make(map[string]string)

	// Add preview-specific variables
	envVars["DEPLOYMENT_ID"] = "preview"
	envVars["COMPOSE_PROJECT"] = "preview-" + recipe.Slug

	// Add user config (convert to UPPER_SNAKE_CASE)
	for key, value := range config {
		upperKey := toEnvVarName(key)
		envVars[upperKey] = fmt.Sprintf("%v", value)
	}

	// Simple variable substitution for preview
	for varName, varValue := range envVars {
		placeholder := "${" + varName + "}"
		content = strings.ReplaceAll(content, placeholder, varValue)
	}

	return content, nil
}

// CheckForUpdates checks all recipe sources for available updates
func (s *MarketplaceService) CheckForUpdates() (map[string][]string, error) {
	return s.recipeLoader.CheckForUpdates()
}

// CuratedMarketplaceResponse contains curated recipes with user deployment status
type CuratedMarketplaceResponse struct {
	Recipes          []*CuratedRecipeWithStatus `json:"recipes"`
	UserDeployments  map[string]*DeploymentInfo `json:"user_deployments"` // Keyed by recipe slug
	Stats            *CuratedMarketplaceStats   `json:"stats"`
}

// CuratedRecipeWithStatus contains recipe info and deployment status
type CuratedRecipeWithStatus struct {
	*models.Recipe
}

// DeploymentInfo contains deployment status for a recipe
type DeploymentInfo struct {
	Status     models.DeploymentStatus `json:"status"` // "running", "pending", "failed", etc.
	DeviceName string                  `json:"device_name"`
	AccessURL  string                  `json:"access_url,omitempty"`
	DeployedAt *string                 `json:"deployed_at,omitempty"` // ISO 8601 timestamp
}

// CuratedMarketplaceStats contains aggregate statistics
type CuratedMarketplaceStats struct {
	TotalCurated int `json:"total_curated"`
	Deployed     int `json:"deployed"`
	Percentage   int `json:"percentage"`
}

// GetCuratedMarketplace returns curated recipes with user deployment status
// TODO: Add userID parameter when multi-user support is implemented to filter deployments by user
func (s *MarketplaceService) GetCuratedMarketplace() (*CuratedMarketplaceResponse, error) {
	// Get all recipes
	allRecipes := s.recipeLoader.ListRecipes()

	// Filter to curated recipes (those with SaaS replacements)
	var curatedRecipes []*models.Recipe
	for _, recipe := range allRecipes {
		if len(recipe.SaaSReplacements) > 0 {
			curatedRecipes = append(curatedRecipes, recipe)
		}
	}

	// Get all deployments
	// TODO: Filter by userID when multi-user support is added: .Where("user_id = ?", userID)
	var deployments []models.Deployment
	if err := s.db.Preload("Device").Find(&deployments).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch deployments: %w", err)
	}

	// Create deployment map keyed by recipe slug
	deploymentMap := make(map[string]*DeploymentInfo)
	deployedCount := 0

	for i := range deployments {
		deployment := &deployments[i]

		// Only include running deployments in the map
		if deployment.Status == models.DeploymentStatusRunning {
			var accessURL string
			if deployment.Domain != "" {
				// Use HTTPS if domain is configured
				accessURL = "https://" + deployment.Domain
			} else if deployment.ExternalPort > 0 && deployment.Device != nil {
				// Fall back to IP:port
				accessURL = fmt.Sprintf("http://%s:%d", deployment.Device.GetPrimaryAddress(), deployment.ExternalPort)
			}

			deployedAt := ""
			if deployment.DeployedAt != nil {
				deployedAt = deployment.DeployedAt.Format("2006-01-02T15:04:05Z07:00")
			}

			deviceName := ""
			if deployment.Device != nil {
				deviceName = deployment.Device.Name
			}

			deploymentMap[deployment.RecipeSlug] = &DeploymentInfo{
				Status:     deployment.Status,
				DeviceName: deviceName,
				AccessURL:  accessURL,
				DeployedAt: &deployedAt,
			}

			// Check if this recipe is in our curated list
			for _, recipe := range curatedRecipes {
				if recipe.Slug == deployment.RecipeSlug {
					deployedCount++
					break
				}
			}
		}
	}

	// Build response
	recipesWithStatus := make([]*CuratedRecipeWithStatus, len(curatedRecipes))
	for i, recipe := range curatedRecipes {
		recipesWithStatus[i] = &CuratedRecipeWithStatus{
			Recipe: recipe,
		}
	}

	// Calculate stats
	totalCurated := len(curatedRecipes)
	percentage := 0
	if totalCurated > 0 {
		percentage = (deployedCount * 100) / totalCurated
	}

	stats := &CuratedMarketplaceStats{
		TotalCurated: totalCurated,
		Deployed:     deployedCount,
		Percentage:   percentage,
	}

	return &CuratedMarketplaceResponse{
		Recipes:         recipesWithStatus,
		UserDeployments: deploymentMap,
		Stats:           stats,
	}, nil
}
