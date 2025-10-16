package services

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"gopkg.in/yaml.v3"
)

// RecipeSource is an interface for loading recipes from different sources
type RecipeSource interface {
	// GetName returns the name of this recipe source
	GetName() string

	// LoadRecipes loads all recipes from this source
	LoadRecipes() (map[string]*models.Recipe, error)

	// SupportsUpdates returns true if this source supports checking for updates
	SupportsUpdates() bool

	// CheckForUpdates checks if any recipes have updates available
	CheckForUpdates() ([]string, error)
}

// CompositeRecipeSource combines multiple recipe sources
type CompositeRecipeSource struct {
	sources []RecipeSource
	cache   map[string]*models.Recipe
	mu      sync.RWMutex
}

// NewCompositeRecipeSource creates a new composite recipe source
func NewCompositeRecipeSource(sources ...RecipeSource) *CompositeRecipeSource {
	return &CompositeRecipeSource{
		sources: sources,
		cache:   make(map[string]*models.Recipe),
	}
}

// LoadRecipes loads recipes from all sources
func (c *CompositeRecipeSource) LoadRecipes() (map[string]*models.Recipe, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	allRecipes := make(map[string]*models.Recipe)

	for _, source := range c.sources {
		recipes, err := source.LoadRecipes()
		if err != nil {
			// Log error but continue with other sources
			fmt.Printf("Warning: Failed to load recipes from %s: %v\n", source.GetName(), err)
			continue
		}

		// Merge recipes (later sources override earlier ones for same slug)
		for slug, recipe := range recipes {
			allRecipes[slug] = recipe
		}
	}

	c.cache = allRecipes
	return allRecipes, nil
}

// GetRecipe gets a recipe from the cache
func (c *CompositeRecipeSource) GetRecipe(slug string) (*models.Recipe, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	recipe, exists := c.cache[slug]
	if !exists {
		return nil, fmt.Errorf("recipe not found: %s", slug)
	}
	return recipe, nil
}

// ListRecipes returns all cached recipes
func (c *CompositeRecipeSource) ListRecipes() []*models.Recipe {
	c.mu.RLock()
	defer c.mu.RUnlock()

	recipes := make([]*models.Recipe, 0, len(c.cache))
	for _, recipe := range c.cache {
		recipes = append(recipes, recipe)
	}
	return recipes
}

// CheckForUpdates checks all sources for updates
func (c *CompositeRecipeSource) CheckForUpdates() (map[string][]string, error) {
	updates := make(map[string][]string)

	for _, source := range c.sources {
		if source.SupportsUpdates() {
			slugs, err := source.CheckForUpdates()
			if err != nil {
				fmt.Printf("Warning: Failed to check updates for %s: %v\n", source.GetName(), err)
				continue
			}
			if len(slugs) > 0 {
				updates[source.GetName()] = slugs
			}
		}
	}

	return updates, nil
}

// LocalRecipeSource loads recipes from local YAML files
type LocalRecipeSource struct {
	recipesPath string
	cache       map[string]*models.Recipe
	mu          sync.RWMutex
}

// NewLocalRecipeSource creates a new local recipe source
func NewLocalRecipeSource(recipesPath string) *LocalRecipeSource {
	return &LocalRecipeSource{
		recipesPath: recipesPath,
		cache:       make(map[string]*models.Recipe),
	}
}

// GetName returns the name of this source
func (l *LocalRecipeSource) GetName() string {
	return "local"
}

// LoadRecipes loads all recipes from local YAML files
func (l *LocalRecipeSource) LoadRecipes() (map[string]*models.Recipe, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	recipes := make(map[string]*models.Recipe)

	// Read directory entries
	entries, err := os.ReadDir(l.recipesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read recipes directory: %w", err)
	}

	for _, entry := range entries {
		// Only process directories (new manifest + docker-compose format)
		if !entry.IsDir() {
			continue
		}

		entryPath := filepath.Join(l.recipesPath, entry.Name())
		recipe, loadErr := l.loadDirectoryRecipe(entryPath)

		if loadErr != nil {
			fmt.Printf("Warning: Failed to load recipe %s: %v\n", entry.Name(), loadErr)
			continue
		}

		// Add metadata
		recipe.Metadata = models.RecipeMetadata{
			Source:       "local",
			Version:      "1.0.0",
			UpdatedAt:    time.Now(),
			Verified:     true,
			QualityScore: 80, // Default quality score for local recipes
		}

		recipes[recipe.Slug] = recipe
	}

	l.cache = recipes
	return recipes, nil
}

// loadDirectoryRecipe loads a recipe from a directory with manifest.yaml and docker-compose.yaml
func (l *LocalRecipeSource) loadDirectoryRecipe(dirPath string) (*models.Recipe, error) {
	// Load manifest.yaml
	manifestPath := filepath.Join(dirPath, "manifest.yaml")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest.yaml: %w", err)
	}

	var recipe models.Recipe
	if err := yaml.Unmarshal(manifestData, &recipe); err != nil {
		return nil, fmt.Errorf("failed to parse manifest.yaml: %w", err)
	}

	// Load docker-compose.yaml (required)
	composePath := filepath.Join(dirPath, "docker-compose.yaml")
	composeData, err := os.ReadFile(composePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read docker-compose.yaml: %w", err)
	}
	recipe.ComposeContent = string(composeData)

	return &recipe, nil
}

// SupportsUpdates returns false for local source
func (l *LocalRecipeSource) SupportsUpdates() bool {
	return false
}

// CheckForUpdates is not supported for local source
func (l *LocalRecipeSource) CheckForUpdates() ([]string, error) {
	return nil, fmt.Errorf("local source does not support update checking")
}
