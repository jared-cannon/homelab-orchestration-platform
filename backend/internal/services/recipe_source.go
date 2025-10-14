package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jaredcannon/homelab-orchestration-platform/internal/models"
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

	// Read all YAML files from recipes directory
	files, err := filepath.Glob(filepath.Join(l.recipesPath, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to list recipe files: %w", err)
	}

	// Also check for .yml extension
	ymlFiles, err := filepath.Glob(filepath.Join(l.recipesPath, "*.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to list recipe files: %w", err)
	}
	files = append(files, ymlFiles...)

	// Load each recipe file
	for _, file := range files {
		recipe, err := l.loadRecipeFile(file)
		if err != nil {
			fmt.Printf("Warning: Failed to load recipe %s: %v\n", file, err)
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

// loadRecipeFile loads a single recipe from a YAML file
func (l *LocalRecipeSource) loadRecipeFile(filename string) (*models.Recipe, error) {
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

// SupportsUpdates returns false for local source
func (l *LocalRecipeSource) SupportsUpdates() bool {
	return false
}

// CheckForUpdates is not supported for local source
func (l *LocalRecipeSource) CheckForUpdates() ([]string, error) {
	return nil, fmt.Errorf("local source does not support update checking")
}

// CoolifyRecipeSource loads recipes from Coolify's template repository
type CoolifyRecipeSource struct {
	apiURL       string
	cacheDir     string
	cacheDuration time.Duration
	cache        map[string]*models.Recipe
	mu           sync.RWMutex
}

// NewCoolifyRecipeSource creates a new Coolify recipe source
func NewCoolifyRecipeSource(cacheDir string) *CoolifyRecipeSource {
	return &CoolifyRecipeSource{
		// Coolify's service templates from their GitHub repo
		apiURL:        "https://cdn.coollabs.io/coolify/service-templates.json",
		cacheDir:      cacheDir,
		cacheDuration: 24 * time.Hour, // Cache for 24 hours
		cache:         make(map[string]*models.Recipe),
	}
}

// GetName returns the name of this source
func (c *CoolifyRecipeSource) GetName() string {
	return "coolify"
}

// LoadRecipes loads recipes from Coolify's template repository
func (c *CoolifyRecipeSource) LoadRecipes() (map[string]*models.Recipe, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check cache first
	cacheFile := filepath.Join(c.cacheDir, "coolify-templates.json")
	if c.isCacheValid(cacheFile) {
		if err := c.loadFromCache(cacheFile); err == nil {
			return c.cache, nil
		}
	}

	// Fetch from Coolify API
	resp, err := http.Get(c.apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Coolify templates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch Coolify templates: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Coolify response: %w", err)
	}

	// Parse Coolify templates - it's an object/map, not an array
	var coolifyTemplates map[string]CoolifyTemplate
	if err := json.Unmarshal(body, &coolifyTemplates); err != nil {
		return nil, fmt.Errorf("failed to parse Coolify templates: %w", err)
	}

	// Convert to our recipe format
	recipes := make(map[string]*models.Recipe)
	for slug, template := range coolifyTemplates {
		recipe := c.convertCoolifyTemplate(slug, template)
		if recipe != nil {
			recipes[recipe.Slug] = recipe
		}
	}

	// Save to cache
	os.MkdirAll(c.cacheDir, 0755)
	if err := os.WriteFile(cacheFile, body, 0644); err != nil {
		fmt.Printf("Warning: Failed to cache Coolify templates: %v\n", err)
	}

	c.cache = recipes
	return recipes, nil
}

// CoolifyTemplate represents a template from Coolify's repository
type CoolifyTemplate struct {
	Documentation string   `json:"documentation"`
	Slogan        string   `json:"slogan"`
	Compose       string   `json:"compose"`       // Base64 encoded
	Tags          []string `json:"tags"`
	Logo          string   `json:"logo"`
	MinVersion    string   `json:"minversion"`
}

// convertCoolifyTemplate converts a Coolify template to our Recipe format
func (c *CoolifyRecipeSource) convertCoolifyTemplate(slug string, template CoolifyTemplate) *models.Recipe {
	// Decode base64 compose template
	composeDecoded, err := base64.StdEncoding.DecodeString(template.Compose)
	if err != nil {
		fmt.Printf("Warning: Failed to decode compose for %s: %v\n", slug, err)
		return nil
	}
	composeTemplate := string(composeDecoded)

	// Validate that compose is not empty
	if strings.TrimSpace(composeTemplate) == "" {
		fmt.Printf("Warning: Empty compose template for %s\n", slug)
		return nil
	}

	// Generate human-readable name from slug
	name := formatLabel(slug)

	// Determine category from tags
	category := "other"
	if len(template.Tags) > 0 {
		// Use first tag as category
		category = strings.ToLower(template.Tags[0])
	}

	// Extract tagline from slogan
	tagline := template.Slogan
	if tagline == "" {
		tagline = name
	}
	if len(tagline) > 100 {
		tagline = tagline[:97] + "..."
	}

	// Build icon URL (Coolify uses relative paths)
	iconURL := ""
	if template.Logo != "" {
		// Coolify logo paths are relative, construct full URL
		iconURL = "https://cdn.coollabs.io/coolify/" + template.Logo
	}

	// For now, we don't extract config options from Coolify templates
	// because they use Coolify-specific env var syntax ($SERVICE_PASSWORD_*, etc.)
	// Users can deploy as-is or customize via the deployment wizard
	configOptions := []models.RecipeConfigOption{}

	recipe := &models.Recipe{
		ID:                     slug,
		Name:                   name,
		Slug:                   slug,
		Category:               category,
		Tagline:                tagline,
		Description:            template.Slogan,
		IconURL:                iconURL,
		ComposeTemplate:        composeTemplate,
		ConfigOptions:          configOptions,
		PostDeployInstructions: fmt.Sprintf("✓ %s is now running!\n\nDocumentation: %s\n\nNote: This template uses Coolify-specific environment variables that may need configuration.", name, template.Documentation),

		// Estimate resource requirements (conservative defaults)
		Resources: models.RecipeResources{
			MinRAMMB:             512,
			MinStorageGB:         5,
			RecommendedRAMMB:     1024,
			RecommendedStorageGB: 10,
			CPUCores:             1,
		},

		// Basic health check (will be overridden if compose defines healthcheck)
		HealthCheck: models.RecipeHealthCheck{
			Path:           "/",
			Port:           80,
			ExpectedStatus: 200,
			TimeoutSeconds: 60,
		},

		Metadata: models.RecipeMetadata{
			Source:       "coolify",
			Version:      template.MinVersion,
			UpdatedAt:    time.Now(),
			SourceURL:    template.Documentation,
			Verified:     true,
			QualityScore: 85, // Coolify templates are generally high quality
		},
	}

	return recipe
}

// formatLabel converts an environment variable key to a readable label
func formatLabel(key string) string {
	// Convert UPPER_SNAKE_CASE to Title Case
	words := strings.Split(key, "_")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

// isCacheValid checks if the cache file is still valid
func (c *CoolifyRecipeSource) isCacheValid(cacheFile string) bool {
	info, err := os.Stat(cacheFile)
	if err != nil {
		return false
	}

	age := time.Since(info.ModTime())
	return age < c.cacheDuration
}

// loadFromCache loads recipes from the cache file
func (c *CoolifyRecipeSource) loadFromCache(cacheFile string) error {
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return err
	}

	var coolifyTemplates map[string]CoolifyTemplate
	if err := json.Unmarshal(data, &coolifyTemplates); err != nil {
		return err
	}

	recipes := make(map[string]*models.Recipe)
	for slug, template := range coolifyTemplates {
		recipe := c.convertCoolifyTemplate(slug, template)
		if recipe != nil {
			recipes[recipe.Slug] = recipe
		}
	}

	c.cache = recipes
	return nil
}

// SupportsUpdates returns true for Coolify source
func (c *CoolifyRecipeSource) SupportsUpdates() bool {
	return true
}

// CheckForUpdates checks for updated Coolify templates
func (c *CoolifyRecipeSource) CheckForUpdates() ([]string, error) {
	// For now, we'll just reload all recipes
	// In a more sophisticated implementation, we could compare versions
	oldCache := c.cache

	_, err := c.LoadRecipes()
	if err != nil {
		return nil, err
	}

	// Compare old and new cache to find updates
	var updated []string
	for slug := range c.cache {
		if _, existed := oldCache[slug]; !existed {
			updated = append(updated, slug)
		}
	}

	return updated, nil
}
