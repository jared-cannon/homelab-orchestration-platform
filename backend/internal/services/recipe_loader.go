package services

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jaredcannon/homelab-orchestration-platform/internal/models"
)

// RecipeLoader loads and caches marketplace recipes from multiple sources
type RecipeLoader struct {
	compositeSource *CompositeRecipeSource
	mu              sync.RWMutex
}

// NewRecipeLoader creates a new recipe loader with multiple sources
func NewRecipeLoader(recipesPath string) *RecipeLoader {
	// Create local recipe source
	localSource := NewLocalRecipeSource(recipesPath)

	// Create Coolify recipe source with cache directory
	cacheDir := filepath.Join(filepath.Dir(recipesPath), "recipe-cache")
	coolifySource := NewCoolifyRecipeSource(cacheDir)

	// Combine sources (local recipes override Coolify if same slug)
	compositeSource := NewCompositeRecipeSource(
		coolifySource, // Load Coolify first
		localSource,   // Local overrides Coolify
	)

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
	if recipe.ComposeTemplate == "" {
		return fmt.Errorf("recipe missing required field: compose_template")
	}

	// Validate resources
	if recipe.Resources.MinRAMMB <= 0 {
		return fmt.Errorf("recipe has invalid min_ram_mb: %d", recipe.Resources.MinRAMMB)
	}
	if recipe.Resources.MinStorageGB <= 0 {
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
		// Validate type is one of: string, number, boolean
		if opt.Type != "string" && opt.Type != "number" && opt.Type != "boolean" {
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

// validateComposeTemplate validates the Docker Compose template syntax
func (r *RecipeLoader) validateComposeTemplate(recipe *models.Recipe) error {
	template := recipe.ComposeTemplate

	// Check for basic Docker Compose structure
	if !strings.Contains(template, "services:") {
		return fmt.Errorf("compose template must contain 'services:' section")
	}

	// Validate it's not empty after trimming
	if len(strings.TrimSpace(template)) == 0 {
		return fmt.Errorf("compose template is empty")
	}

	// Check for common YAML syntax errors
	lines := strings.Split(template, "\n")
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
	// Extract variables from template ({{.VarName}})
	template := recipe.ComposeTemplate

	// Find all template variables
	start := 0
	for {
		idx := strings.Index(template[start:], "{{")
		if idx == -1 {
			break
		}
		idx += start

		endIdx := strings.Index(template[idx:], "}}")
		if endIdx == -1 {
			return fmt.Errorf("unclosed template variable at position %d", idx)
		}
		endIdx += idx

		// Extract variable name (remove {{. and }})
		varExpr := template[idx+2 : endIdx]
		varExpr = strings.TrimSpace(varExpr)

		// Remove leading dot if present
		if strings.HasPrefix(varExpr, ".") {
			varExpr = varExpr[1:]
		}

		// Skip if it's a built-in variable or control structure
		if strings.HasPrefix(varExpr, "if ") ||
		   strings.HasPrefix(varExpr, "range ") ||
		   strings.HasPrefix(varExpr, "end") ||
		   varExpr == "DEPLOYMENT_ID" ||
		   varExpr == "COMPOSE_PROJECT" {
			start = endIdx + 2
			continue
		}

		// Convert PascalCase to snake_case for lookup
		snakeCase := pascalToSnakeCase(varExpr)

		// Check if variable is defined in config options
		if !configVars[snakeCase] && !configVars[varExpr] {
			// Allow some common derived variables
			if !strings.HasSuffix(varExpr, "PasswordHash") && varExpr != "DashboardPasswordHash" {
				return fmt.Errorf("template variable '%s' is not defined in config_options (expected config option: %s)", varExpr, snakeCase)
			}
		}

		start = endIdx + 2
	}

	return nil
}

// pascalToSnakeCase converts PascalCase to snake_case
func pascalToSnakeCase(input string) string {
	var result strings.Builder
	for i, r := range input {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// Reload reloads all recipes from disk (useful for development/updates)
func (r *RecipeLoader) Reload() error {
	_, err := r.LoadAll()
	return err
}
