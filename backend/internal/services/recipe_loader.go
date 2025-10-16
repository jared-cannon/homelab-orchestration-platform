package services

import (
	"fmt"
	"strings"
	"sync"

	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
)

// RecipeLoader loads and caches marketplace recipes from multiple sources
type RecipeLoader struct {
	compositeSource *CompositeRecipeSource
	mu              sync.RWMutex
}

// NewRecipeLoader creates a new recipe loader with local recipe source
func NewRecipeLoader(recipesPath string) *RecipeLoader {
	// Create local recipe source
	localSource := NewLocalRecipeSource(recipesPath)

	// Use composite source to allow future expansion
	compositeSource := NewCompositeRecipeSource(localSource)

	return &RecipeLoader{
		compositeSource: compositeSource,
	}
}

// LoadAll loads all recipes from all sources
func (r *RecipeLoader) LoadAll() (map[string]*models.Recipe, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Load from composite source
	recipes, err := r.compositeSource.LoadRecipes()
	if err != nil {
		return nil, err
	}

	// Validate all loaded recipes
	validRecipes := make(map[string]*models.Recipe)
	for slug, recipe := range recipes {
		if err := r.Validate(recipe); err != nil {
			fmt.Printf("Warning: Invalid recipe %s from %s: %v\n", slug, recipe.Metadata.Source, err)
			continue
		}
		validRecipes[slug] = recipe
	}

	return validRecipes, nil
}

// GetRecipe retrieves a recipe from the cache by slug
func (r *RecipeLoader) GetRecipe(slug string) (*models.Recipe, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.compositeSource.GetRecipe(slug)
}

// ListRecipes returns all cached recipes
func (r *RecipeLoader) ListRecipes() []*models.Recipe {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.compositeSource.ListRecipes()
}

// ListRecipesByCategory returns recipes filtered by category
func (r *RecipeLoader) ListRecipesByCategory(category string) []*models.Recipe {
	r.mu.RLock()
	defer r.mu.RUnlock()

	allRecipes := r.compositeSource.ListRecipes()
	if category == "" {
		return allRecipes
	}

	filtered := make([]*models.Recipe, 0)
	for _, recipe := range allRecipes {
		if recipe.Category == category {
			filtered = append(filtered, recipe)
		}
	}

	return filtered
}

// CheckForUpdates checks all sources for recipe updates
func (r *RecipeLoader) CheckForUpdates() (map[string][]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.compositeSource.CheckForUpdates()
}

// Validate validates a recipe's structure and required fields
func (r *RecipeLoader) Validate(recipe *models.Recipe) error {
	// First, run the Recipe model's built-in validation for new manifest format
	// This validates Requirements, Database config, Cache config, etc.
	if err := recipe.Validate(); err != nil {
		return fmt.Errorf("manifest validation failed: %w", err)
	}

	// Basic required fields
	if recipe.ID == "" {
		return fmt.Errorf("recipe missing required field: id")
	}
	if recipe.Name == "" {
		return fmt.Errorf("recipe missing required field: name")
	}
	if recipe.Slug == "" {
		return fmt.Errorf("recipe missing required field: slug")
	}
	if recipe.Category == "" {
		return fmt.Errorf("recipe missing required field: category")
	}
	if recipe.Description == "" {
		return fmt.Errorf("recipe missing required field: description")
	}

	// All recipes must have ComposeContent (docker-compose.yaml)
	if recipe.ComposeContent == "" {
		return fmt.Errorf("recipe must have compose content (docker-compose.yaml)")
	}

	// Validate legacy resources (if provided)
	// New recipes use Requirements instead, so these are optional
	if recipe.Resources.MinRAMMB < 0 {
		return fmt.Errorf("recipe has invalid min_ram_mb: %d", recipe.Resources.MinRAMMB)
	}
	if recipe.Resources.MinStorageGB < 0 {
		return fmt.Errorf("recipe has invalid min_storage_gb: %d", recipe.Resources.MinStorageGB)
	}
	if recipe.Resources.CPUCores < 0 {
		return fmt.Errorf("recipe has invalid cpu_cores: %d", recipe.Resources.CPUCores)
	}

	// Validate config options
	configVarMap := make(map[string]bool)
	for i, opt := range recipe.ConfigOptions {
		if opt.Name == "" {
			return fmt.Errorf("config option %d missing name", i)
		}
		if opt.Label == "" {
			return fmt.Errorf("config option %s missing label", opt.Name)
		}
		if opt.Type == "" {
			return fmt.Errorf("config option %s missing type", opt.Name)
		}
		// Validate type is one of the allowed types
		validTypes := map[string]bool{
			"string":   true,
			"number":   true,
			"boolean":  true,
			"secret":   true,
			"api_key":  true,
			"password": true,
			"email":    true,
			"domain":   true,
		}
		if !validTypes[opt.Type] {
			return fmt.Errorf("config option %s has invalid type: %s", opt.Name, opt.Type)
		}

		// Validate required fields have defaults
		if opt.Required && opt.Default == nil {
			return fmt.Errorf("config option %s is required but has no default value", opt.Name)
		}

		// Track variable names for template validation
		configVarMap[opt.Name] = true
	}

	// Validate health check if defined
	if recipe.HealthCheck.Port != 0 {
		if recipe.HealthCheck.Port < 1 || recipe.HealthCheck.Port > 65535 {
			return fmt.Errorf("health check has invalid port: %d (must be 1-65535)", recipe.HealthCheck.Port)
		}
	}
	if recipe.HealthCheck.ExpectedStatus != 0 {
		if recipe.HealthCheck.ExpectedStatus < 100 || recipe.HealthCheck.ExpectedStatus > 599 {
			return fmt.Errorf("health check has invalid expected status: %d (must be 100-599)", recipe.HealthCheck.ExpectedStatus)
		}
	}
	if recipe.HealthCheck.TimeoutSeconds < 0 {
		return fmt.Errorf("health check has invalid timeout: %d (must be >= 0)", recipe.HealthCheck.TimeoutSeconds)
	}

	// Validate compose template syntax
	if err := r.validateComposeTemplate(recipe); err != nil {
		return fmt.Errorf("invalid compose template: %w", err)
	}

	// Validate template variables are defined in config_options
	if err := r.validateTemplateVariables(recipe, configVarMap); err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	return nil
}

// validateComposeTemplate validates the Docker Compose content syntax
func (r *RecipeLoader) validateComposeTemplate(recipe *models.Recipe) error {
	content := recipe.ComposeContent

	// Check for basic Docker Compose structure
	if !strings.Contains(content, "services:") {
		return fmt.Errorf("compose content must contain 'services:' section")
	}

	// Validate it's not empty after trimming
	if len(strings.TrimSpace(content)) == 0 {
		return fmt.Errorf("compose content is empty")
	}

	// Check for common YAML syntax errors
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check for tabs (YAML doesn't allow tabs for indentation)
		if strings.Contains(line, "\t") {
			return fmt.Errorf("line %d contains tabs (YAML requires spaces for indentation)", i+1)
		}
	}

	return nil
}

// validateTemplateVariables checks that template variables are defined in config_options
func (r *RecipeLoader) validateTemplateVariables(recipe *models.Recipe, configVars map[string]bool) error {
	// Extract variables from docker-compose content (${VAR_NAME})
	content := recipe.ComposeContent

	// Find all template variables
	start := 0
	for {
		idx := strings.Index(content[start:], "${")
		if idx == -1 {
			break
		}
		idx += start

		endIdx := strings.Index(content[idx:], "}")
		if endIdx == -1 {
			return fmt.Errorf("unclosed template variable at position %d", idx)
		}
		endIdx += idx

		// Extract variable name (remove ${ and })
		varName := content[idx+2 : endIdx]
		varName = strings.TrimSpace(varName)

		// Skip built-in deployment variables
		if varName == "DEPLOYMENT_ID" ||
			varName == "COMPOSE_PROJECT" ||
			varName == "DEVICE_IP" ||
			strings.HasPrefix(varName, "POSTGRES_") ||
			strings.HasPrefix(varName, "MYSQL_") ||
			strings.HasPrefix(varName, "REDIS_") {
			start = endIdx + 1
			continue
		}

		// Convert UPPER_SNAKE_CASE to snake_case for lookup in config options
		lowerVarName := strings.ToLower(varName)

		// Check if variable is defined in config options
		if !configVars[lowerVarName] {
			// Allow derived variables like PASSWORD_HASH
			if !strings.HasSuffix(varName, "_HASH") {
				return fmt.Errorf("docker-compose variable '%s' is not defined in config_options (expected config option: %s)", varName, lowerVarName)
			}
		}

		start = endIdx + 1
	}

	return nil
}

// Reload reloads all recipes from disk (useful for development/updates)
func (r *RecipeLoader) Reload() error {
	_, err := r.LoadAll()
	return err
}
