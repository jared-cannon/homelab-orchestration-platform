package services

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLocalRecipeSource tests loading local recipes from directories
func TestLocalRecipeSource(t *testing.T) {
	// Create temporary directory for test recipes
	tmpDir, err := os.MkdirTemp("", "test-recipes-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a recipe directory
	recipeDir := filepath.Join(tmpDir, "test-app")
	if err := os.MkdirAll(recipeDir, 0755); err != nil {
		t.Fatalf("Failed to create recipe directory: %v", err)
	}

	// Create manifest.yaml
	manifest := `id: test-app
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

	// Create docker-compose.yaml
	compose := `version: '3.8'
services:
  test:
    image: nginx:latest
    ports:
      - "8080:80"
`

	// Write files to recipe directory
	err = os.WriteFile(filepath.Join(recipeDir, "manifest.yaml"), []byte(manifest), 0644)
	if err != nil {
		t.Fatalf("Failed to write manifest.yaml: %v", err)
	}

	err = os.WriteFile(filepath.Join(recipeDir, "docker-compose.yaml"), []byte(compose), 0644)
	if err != nil {
		t.Fatalf("Failed to write docker-compose.yaml: %v", err)
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

	// Verify compose content was loaded
	if recipe.ComposeContent == "" {
		t.Error("Expected ComposeContent to be populated from docker-compose.yaml")
	}

	if !contains(recipe.ComposeContent, "nginx:latest") {
		t.Errorf("ComposeContent should contain 'nginx:latest': %s", recipe.ComposeContent)
	}
}

// TestCompositeRecipeSource tests combining multiple sources
func TestCompositeRecipeSource(t *testing.T) {
	// Create temporary directories
	tmpDir1, _ := os.MkdirTemp("", "test-recipes-1-*")
	tmpDir2, _ := os.MkdirTemp("", "test-recipes-2-*")
	defer os.RemoveAll(tmpDir1)
	defer os.RemoveAll(tmpDir2)

	// Create recipe directory in first source
	recipeDir1 := filepath.Join(tmpDir1, "app1")
	os.MkdirAll(recipeDir1, 0755)

	manifest1 := `id: app1
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

config_options: []
`
	compose1 := `version: '3.8'
services:
  app1:
    image: app1:latest
`
	os.WriteFile(filepath.Join(recipeDir1, "manifest.yaml"), []byte(manifest1), 0644)
	os.WriteFile(filepath.Join(recipeDir1, "docker-compose.yaml"), []byte(compose1), 0644)

	// Create recipe directory in second source (with same slug to test override)
	recipeDir2 := filepath.Join(tmpDir2, "app1")
	os.MkdirAll(recipeDir2, 0755)

	manifest2 := `id: app1
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

config_options: []
`
	compose2 := `version: '3.8'
services:
  app1:
    image: app1:v2
`
	os.WriteFile(filepath.Join(recipeDir2, "manifest.yaml"), []byte(manifest2), 0644)
	os.WriteFile(filepath.Join(recipeDir2, "docker-compose.yaml"), []byte(compose2), 0644)

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
