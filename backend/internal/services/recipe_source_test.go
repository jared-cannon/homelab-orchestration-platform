package services

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
)

// TestLocalRecipeSource tests loading local YAML recipes
func TestLocalRecipeSource(t *testing.T) {
	// Create temporary directory for test recipes
	tmpDir, err := os.MkdirTemp("", "test-recipes-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a valid test recipe
	validRecipe := `id: test-app
name: Test App
slug: test-app
category: testing
tagline: "A test application"
description: "This is a test application for unit testing"
icon_url: "https://example.com/icon.png"

resources:
  min_ram_mb: 512
  min_storage_gb: 1
  recommended_ram_mb: 1024
  recommended_storage_gb: 5
  cpu_cores: 1

compose_template: |
  version: '3.8'
  services:
    test:
      image: nginx:latest
      ports:
        - "8080:80"

config_options:
  - name: port
    label: "Port"
    type: number
    default: 8080
    required: true
    description: "Port to expose"

health_check:
  path: "/health"
  port: 80
  expected_status: 200
  timeout_seconds: 30
`

	// Write valid recipe to file
	err = os.WriteFile(filepath.Join(tmpDir, "test-app.yaml"), []byte(validRecipe), 0644)
	if err != nil {
		t.Fatalf("Failed to write test recipe: %v", err)
	}

	// Create recipe source
	source := NewLocalRecipeSource(tmpDir)

	// Load recipes
	recipes, err := source.LoadRecipes()
	if err != nil {
		t.Fatalf("Failed to load recipes: %v", err)
	}

	// Verify we got one recipe
	if len(recipes) != 1 {
		t.Fatalf("Expected 1 recipe, got %d", len(recipes))
	}

	// Verify recipe data
	recipe, exists := recipes["test-app"]
	if !exists {
		t.Fatal("Recipe 'test-app' not found")
	}

	if recipe.Name != "Test App" {
		t.Errorf("Expected name 'Test App', got '%s'", recipe.Name)
	}

	if recipe.Metadata.Source != "local" {
		t.Errorf("Expected source 'local', got '%s'", recipe.Metadata.Source)
	}

	if recipe.Resources.MinRAMMB != 512 {
		t.Errorf("Expected MinRAMMB 512, got %d", recipe.Resources.MinRAMMB)
	}

	// Verify config options
	if len(recipe.ConfigOptions) != 1 {
		t.Errorf("Expected 1 config option, got %d", len(recipe.ConfigOptions))
	}
}

// TestCoolifyRecipeSource_ParseFormat tests Coolify API response parsing
func TestCoolifyRecipeSource_ParseFormat(t *testing.T) {
	// Create a mock Coolify API server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a mock response in Coolify format
		composeContent := `services:
  nginx:
    image: nginx:latest
    ports:
      - "80:80"`

		composeEncoded := base64.StdEncoding.EncodeToString([]byte(composeContent))

		response := map[string]CoolifyTemplate{
			"nginx": {
				Documentation: "https://nginx.org/docs",
				Slogan:        "High performance web server",
				Compose:       composeEncoded,
				Tags:          []string{"web", "proxy"},
				Logo:          "svgs/nginx.svg",
				MinVersion:    "0.0.0",
			},
			"redis": {
				Documentation: "https://redis.io/docs",
				Slogan:        "In-memory data structure store",
				Compose:       base64.StdEncoding.EncodeToString([]byte("services:\n  redis:\n    image: redis:latest")),
				Tags:          []string{"database", "cache"},
				Logo:          "svgs/redis.svg",
				MinVersion:    "0.0.0",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Create temporary cache directory
	tmpDir, err := os.MkdirTemp("", "test-cache-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create Coolify source pointing to mock server
	source := &CoolifyRecipeSource{
		apiURL:        mockServer.URL,
		cacheDir:      tmpDir,
		cacheDuration: 24 * time.Hour,
		cache:         make(map[string]*models.Recipe),
	}

	// Load recipes
	recipes, err := source.LoadRecipes()
	if err != nil {
		t.Fatalf("Failed to load recipes: %v", err)
	}

	// Verify we got two recipes
	if len(recipes) != 2 {
		t.Fatalf("Expected 2 recipes, got %d", len(recipes))
	}

	// Verify nginx recipe
	nginx, exists := recipes["nginx"]
	if !exists {
		t.Fatal("Recipe 'nginx' not found")
	}

	if nginx.Name != "Nginx" {
		t.Errorf("Expected name 'Nginx', got '%s'", nginx.Name)
	}

	if nginx.Metadata.Source != "coolify" {
		t.Errorf("Expected source 'coolify', got '%s'", nginx.Metadata.Source)
	}

	if nginx.Description != "High performance web server" {
		t.Errorf("Expected correct description, got '%s'", nginx.Description)
	}

	// Verify compose was decoded correctly
	if !contains(nginx.ComposeTemplate, "nginx:latest") {
		t.Errorf("Compose template not properly decoded: %s", nginx.ComposeTemplate)
	}

	// Verify icon URL was constructed
	expectedIconURL := "https://cdn.coollabs.io/coolify/svgs/nginx.svg"
	if nginx.IconURL != expectedIconURL {
		t.Errorf("Expected icon URL '%s', got '%s'", expectedIconURL, nginx.IconURL)
	}

	// Verify category from tags
	if nginx.Category != "web" {
		t.Errorf("Expected category 'web', got '%s'", nginx.Category)
	}
}

// TestCoolifyRecipeSource_Base64Decoding tests base64 decode error handling
func TestCoolifyRecipeSource_Base64Decoding(t *testing.T) {
	source := &CoolifyRecipeSource{
		cacheDir:      os.TempDir(),
		cacheDuration: 24 * time.Hour,
		cache:         make(map[string]*models.Recipe),
	}

	// Test with invalid base64
	invalidTemplate := CoolifyTemplate{
		Documentation: "https://example.com",
		Slogan:        "Test",
		Compose:       "NOT_VALID_BASE64!!!",
		Tags:          []string{"test"},
		Logo:          "test.svg",
		MinVersion:    "0.0.0",
	}

	recipe := source.convertCoolifyTemplate("invalid", invalidTemplate)
	if recipe != nil {
		t.Error("Expected nil recipe for invalid base64, got valid recipe")
	}

	// Test with empty compose after decoding
	emptyTemplate := CoolifyTemplate{
		Documentation: "https://example.com",
		Slogan:        "Test",
		Compose:       base64.StdEncoding.EncodeToString([]byte("")),
		Tags:          []string{"test"},
		Logo:          "test.svg",
		MinVersion:    "0.0.0",
	}

	recipe = source.convertCoolifyTemplate("empty", emptyTemplate)
	if recipe != nil {
		t.Error("Expected nil recipe for empty compose, got valid recipe")
	}
}

// TestCoolifyRecipeSource_Caching tests the caching mechanism
func TestCoolifyRecipeSource_Caching(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-cache-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create mock server
	callCount := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		response := map[string]CoolifyTemplate{
			"test": {
				Documentation: "https://example.com",
				Slogan:        "Test app",
				Compose:       base64.StdEncoding.EncodeToString([]byte("services:\n  test:\n    image: test:latest")),
				Tags:          []string{"test"},
				Logo:          "test.svg",
				MinVersion:    "0.0.0",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	source := &CoolifyRecipeSource{
		apiURL:        mockServer.URL,
		cacheDir:      tmpDir,
		cacheDuration: 1 * time.Hour, // Long cache duration
		cache:         make(map[string]*models.Recipe),
	}

	// First load - should hit API
	_, err = source.LoadRecipes()
	if err != nil {
		t.Fatalf("Failed to load recipes: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 API call, got %d", callCount)
	}

	// Second load - should use cache
	_, err = source.LoadRecipes()
	if err != nil {
		t.Fatalf("Failed to load recipes from cache: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected cache to be used (still 1 API call), got %d calls", callCount)
	}
}

// TestCompositeRecipeSource tests combining multiple sources
func TestCompositeRecipeSource(t *testing.T) {
	// Create temporary directories
	tmpDir1, _ := os.MkdirTemp("", "test-recipes-1-*")
	tmpDir2, _ := os.MkdirTemp("", "test-recipes-2-*")
	defer os.RemoveAll(tmpDir1)
	defer os.RemoveAll(tmpDir2)

	// Create recipe in first source
	recipe1 := `id: app1
name: App 1
slug: app1
category: testing
tagline: "First app"
description: "From source 1"
icon_url: "https://example.com/icon1.png"

resources:
  min_ram_mb: 512
  min_storage_gb: 1
  cpu_cores: 1

compose_template: |
  version: '3.8'
  services:
    app1:
      image: app1:latest

config_options: []
`
	os.WriteFile(filepath.Join(tmpDir1, "app1.yaml"), []byte(recipe1), 0644)

	// Create recipe in second source (with same slug to test override)
	recipe2 := `id: app1
name: App 1 Override
slug: app1
category: testing
tagline: "Overridden app"
description: "From source 2 - should override"
icon_url: "https://example.com/icon2.png"

resources:
  min_ram_mb: 512
  min_storage_gb: 1
  cpu_cores: 1

compose_template: |
  version: '3.8'
  services:
    app1:
      image: app1:v2

config_options: []
`
	os.WriteFile(filepath.Join(tmpDir2, "app1.yaml"), []byte(recipe2), 0644)

	// Create sources
	source1 := NewLocalRecipeSource(tmpDir1)
	source2 := NewLocalRecipeSource(tmpDir2)

	// Create composite source (source2 should override source1)
	composite := NewCompositeRecipeSource(source1, source2)

	// Load recipes
	recipes, err := composite.LoadRecipes()
	if err != nil {
		t.Fatalf("Failed to load recipes: %v", err)
	}

	// Should have 1 recipe (override)
	if len(recipes) != 1 {
		t.Fatalf("Expected 1 recipe, got %d", len(recipes))
	}

	// Verify it's from source 2 (the override)
	recipe := recipes["app1"]
	if recipe.Name != "App 1 Override" {
		t.Errorf("Expected overridden name, got '%s'", recipe.Name)
	}

	if recipe.Description != "From source 2 - should override" {
		t.Errorf("Expected overridden description, got '%s'", recipe.Description)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
