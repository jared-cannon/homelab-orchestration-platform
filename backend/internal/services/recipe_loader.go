package services

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/jaredcannon/homelab-orchestration-platform/internal/models"
	"gopkg.in/yaml.v3"
)

// RecipeLoader loads and caches marketplace recipes from YAML files
type RecipeLoader struct {
	recipesPath string
	cache       map[string]*models.Recipe
	mu          sync.RWMutex
}

// NewRecipeLoader creates a new recipe loader
func NewRecipeLoader(recipesPath string) *RecipeLoader {
	return &RecipeLoader{
		recipesPath: recipesPath,
		cache:       make(map[string]*models.Recipe),
	}
}

// LoadAll loads all recipes from the recipes directory
func (r *RecipeLoader) LoadAll() (map[string]*models.Recipe, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	recipes := make(map[string]*models.Recipe)

	// Read all YAML files from recipes directory
	files, err := filepath.Glob(filepath.Join(r.recipesPath, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to list recipe files: %w", err)
	}

	// Also check for .yml extension
	ymlFiles, err := filepath.Glob(filepath.Join(r.recipesPath, "*.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to list recipe files: %w", err)
	}
	files = append(files, ymlFiles...)

	if len(files) == 0 {
		return recipes, nil // No recipes found, return empty map
	}

	// Load each recipe file
	for _, file := range files {
		recipe, err := r.loadRecipeFile(file)
		if err != nil {
			// Log error but continue loading other recipes
			fmt.Printf("Warning: Failed to load recipe %s: %v\n", file, err)
			continue
		}

		// Validate recipe
		if err := r.Validate(recipe); err != nil {
			fmt.Printf("Warning: Invalid recipe %s: %v\n", file, err)
			continue
		}

		recipes[recipe.Slug] = recipe
	}

	// Update cache
	r.cache = recipes

	return recipes, nil
}

// loadRecipeFile loads a single recipe from a YAML file
func (r *RecipeLoader) loadRecipeFile(filename string) (*models.Recipe, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var recipe models.Recipe
	if err := yaml.Unmarshal(data, &recipe); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &recipe, nil
}

// GetRecipe retrieves a recipe from the cache by slug
func (r *RecipeLoader) GetRecipe(slug string) (*models.Recipe, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	recipe, exists := r.cache[slug]
	if !exists {
		return nil, fmt.Errorf("recipe not found: %s", slug)
	}

	return recipe, nil
}

// ListRecipes returns all cached recipes
func (r *RecipeLoader) ListRecipes() []*models.Recipe {
	r.mu.RLock()
	defer r.mu.RUnlock()

	recipes := make([]*models.Recipe, 0, len(r.cache))
	for _, recipe := range r.cache {
		recipes = append(recipes, recipe)
	}

	return recipes
}

// ListRecipesByCategory returns recipes filtered by category
func (r *RecipeLoader) ListRecipesByCategory(category string) []*models.Recipe {
	r.mu.RLock()
	defer r.mu.RUnlock()

	recipes := make([]*models.Recipe, 0)
	for _, recipe := range r.cache {
		if category == "" || recipe.Category == category {
			recipes = append(recipes, recipe)
		}
	}

	return recipes
}

// Validate validates a recipe's structure and required fields
func (r *RecipeLoader) Validate(recipe *models.Recipe) error {
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

	// Validate config options
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
	}

	return nil
}

// Reload reloads all recipes from disk (useful for development/updates)
func (r *RecipeLoader) Reload() error {
	_, err := r.LoadAll()
	return err
}
