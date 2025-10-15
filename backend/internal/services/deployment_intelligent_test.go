package services

import (
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRecipeLoader is a simple mock for testing
type MockRecipeLoader struct {
	recipes map[string]*models.Recipe
	mu      sync.RWMutex
}

func NewMockRecipeLoader(recipes map[string]*models.Recipe) *MockRecipeLoader {
	return &MockRecipeLoader{
		recipes: recipes,
	}
}

func (m *MockRecipeLoader) GetRecipe(slug string) (*models.Recipe, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	recipe, exists := m.recipes[slug]
	if !exists {
		return nil, fmt.Errorf("recipe not found: %s", slug)
	}
	return recipe, nil
}

func (m *MockRecipeLoader) ListRecipes() []models.Recipe {
	m.mu.RLock()
	defer m.mu.RUnlock()

	recipes := make([]models.Recipe, 0, len(m.recipes))
	for _, r := range m.recipes {
		recipes = append(recipes, *r)
	}
	return recipes
}

func (m *MockRecipeLoader) LoadAll() ([]models.Recipe, error) {
	return m.ListRecipes(), nil
}

func TestDeploymentService_ParseMemoryRequirement(t *testing.T) {
	db := setupTestDB(t)
	credService, _ := NewCredentialService()
	deviceService := NewDeviceService(db, credService, nil)
	mockRecipeLoader := NewMockRecipeLoader(map[string]*models.Recipe{})
	deploymentService := NewDeploymentService(db, nil, mockRecipeLoader, deviceService, credService, nil)

	tests := []struct {
		input    string
		expected int
	}{
		{"512MB", 512},
		{"1GB", 1024},
		{"2GB", 2048},
		{"256MB", 256},
		{"4GB", 4096},
		{"", 512},           // Default
		{"invalid", 512},   // Default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := deploymentService.parseMemoryRequirement(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDeploymentService_ParseStorageRequirement(t *testing.T) {
	db := setupTestDB(t)
	credService, _ := NewCredentialService()
	deviceService := NewDeviceService(db, credService, nil)
	mockRecipeLoader := NewMockRecipeLoader(map[string]*models.Recipe{})
	deploymentService := NewDeploymentService(db, nil, mockRecipeLoader, deviceService, credService, nil)

	tests := []struct {
		input    string
		expected int
	}{
		{"1GB", 1},
		{"10GB", 10},
		{"100GB", 100},
		{"5GB", 5},
		{"", 1},           // Default
		{"invalid", 1},   // Default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := deploymentService.parseStorageRequirement(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test intelligent device selection in CreateDeployment
// Skipped: requires SSH connection
func TestDeploymentService_CreateDeploymentAutoSelectDevice(t *testing.T) {
	t.Skip("Skipping test that requires SSH connection to devices")
	db := setupTestDB(t)
	credService, _ := NewCredentialService()

	// Create mock devices with different resource profiles
	device1 := &models.Device{
		ID:                 uuid.New(),
		Name:               "low-resource-device",
		IPAddress:          "192.168.1.101",
		Status:             models.DeviceStatusOnline,
		TotalRAMMB:         &[]int{2048}[0],      // 2GB
		AvailableRAMMB:     &[]int{512}[0],       // 512MB available
		TotalStorageGB:     &[]int{50}[0],
		AvailableStorageGB: &[]int{10}[0],
		CPUCores:           &[]int{2}[0],
	}

	device2 := &models.Device{
		ID:                 uuid.New(),
		Name:               "high-resource-device",
		IPAddress:          "192.168.1.102",
		Status:             models.DeviceStatusOnline,
		TotalRAMMB:         &[]int{16384}[0],     // 16GB
		AvailableRAMMB:     &[]int{8192}[0],      // 8GB available
		TotalStorageGB:     &[]int{500}[0],
		AvailableStorageGB: &[]int{300}[0],
		CPUCores:           &[]int{8}[0],
	}

	require.NoError(t, db.Create(device1).Error)
	require.NoError(t, db.Create(device2).Error)

	// Create a mock recipe loader with a test recipe
	recipeLoader := NewMockRecipeLoader(map[string]*models.Recipe{
		"test-app": {
			ID:   "test-app",
			Name: "Test App",
			Slug: "test-app",
			Requirements: models.RecipeRequirements{
				Memory: struct {
					Minimum     string `yaml:"minimum" json:"minimum"`
					Recommended string `yaml:"recommended" json:"recommended"`
				}{
					Minimum:     "1GB",
					Recommended: "2GB",
				},
				Storage: struct {
					Minimum     string `yaml:"minimum" json:"minimum"`
					Recommended string `yaml:"recommended" json:"recommended"`
					Type        string `yaml:"type" json:"type"`
				}{
					Minimum:     "10GB",
					Recommended: "50GB",
				},
				CPU: struct {
					MinimumCores     int `yaml:"minimum_cores" json:"minimum_cores"`
					RecommendedCores int `yaml:"recommended_cores" json:"recommended_cores"`
				}{
					MinimumCores:     2,
					RecommendedCores: 4,
				},
			},
			ComposeContent: "version: '3.8'\nservices:\n  app:\n    image: test:latest",
		},
	})

	deviceService := NewDeviceService(db, credService, nil)
	deploymentService := NewDeploymentService(db, nil, recipeLoader, deviceService, credService, nil)

	// Test auto-select device (should select device2 as it has more resources)
	req := CreateDeploymentRequest{
		RecipeSlug:       "test-app",
		AutoSelectDevice: true,
		Config: map[string]interface{}{
			"port": 8080,
		},
	}

	deployment, err := deploymentService.CreateDeployment(req)
	require.NoError(t, err)
	assert.NotNil(t, deployment)

	// Should have selected the high-resource device
	assert.Equal(t, device2.ID, deployment.DeviceID)
}

func TestDeploymentService_RecommendDevicesForRecipe(t *testing.T) {
	t.Skip("Skipping test that requires SSH connection to devices")
	db := setupTestDB(t)
	credService, _ := NewCredentialService()

	// Create devices with different specs
	device1 := &models.Device{
		ID:                 uuid.New(),
		Name:               "minimal-device",
		IPAddress:          "192.168.1.101",
		Status:             models.DeviceStatusOnline,
		TotalRAMMB:         &[]int{1024}[0],   // 1GB
		AvailableRAMMB:     &[]int{512}[0],
		TotalStorageGB:     &[]int{20}[0],
		AvailableStorageGB: &[]int{5}[0],
		CPUCores:           &[]int{1}[0],
	}

	device2 := &models.Device{
		ID:                 uuid.New(),
		Name:               "good-device",
		IPAddress:          "192.168.1.102",
		Status:             models.DeviceStatusOnline,
		TotalRAMMB:         &[]int{4096}[0],   // 4GB
		AvailableRAMMB:     &[]int{2048}[0],
		TotalStorageGB:     &[]int{100}[0],
		AvailableStorageGB: &[]int{50}[0],
		CPUCores:           &[]int{4}[0],
	}

	device3 := &models.Device{
		ID:                 uuid.New(),
		Name:               "best-device",
		IPAddress:          "192.168.1.103",
		Status:             models.DeviceStatusOnline,
		TotalRAMMB:         &[]int{16384}[0],  // 16GB
		AvailableRAMMB:     &[]int{8192}[0],
		TotalStorageGB:     &[]int{500}[0],
		AvailableStorageGB: &[]int{300}[0],
		CPUCores:           &[]int{8}[0],
	}

	require.NoError(t, db.Create(device1).Error)
	require.NoError(t, db.Create(device2).Error)
	require.NoError(t, db.Create(device3).Error)

	recipeLoader := NewMockRecipeLoader(map[string]*models.Recipe{
		"test-app": {
			ID:   "test-app",
			Name: "Test App",
			Slug: "test-app",
			Requirements: models.RecipeRequirements{
				Memory: struct {
					Minimum     string `yaml:"minimum" json:"minimum"`
					Recommended string `yaml:"recommended" json:"recommended"`
				}{
					Minimum:     "1GB",
					Recommended: "2GB",
				},
				Storage: struct {
					Minimum     string `yaml:"minimum" json:"minimum"`
					Recommended string `yaml:"recommended" json:"recommended"`
					Type        string `yaml:"type" json:"type"`
				}{
					Minimum:     "10GB",
					Recommended: "20GB",
				},
				CPU: struct {
					MinimumCores     int `yaml:"minimum_cores" json:"minimum_cores"`
					RecommendedCores int `yaml:"recommended_cores" json:"recommended_cores"`
				}{
					MinimumCores:     1,
					RecommendedCores: 2,
				},
			},
		},
	})

	deviceService := NewDeviceService(db, credService, nil)
	deploymentService := NewDeploymentService(db, nil, recipeLoader, deviceService, credService, nil)

	recommendations, err := deploymentService.RecommendDevicesForRecipe("test-app")
	require.NoError(t, err)
	require.Len(t, recommendations, 3)

	// Verify they're sorted by score (highest first)
	assert.Greater(t, recommendations[0].Score, recommendations[1].Score)
	assert.Greater(t, recommendations[1].Score, recommendations[2].Score)

	// Best device should be first
	assert.Equal(t, "best-device", recommendations[0].DeviceName)
	assert.Equal(t, "best", recommendations[0].Recommendation)

	// Good device should be second
	assert.Equal(t, "good-device", recommendations[1].DeviceName)
	assert.Contains(t, []string{"best", "good"}, recommendations[1].Recommendation)

	// Minimal device should be last (may or may not be available)
	assert.Equal(t, "minimal-device", recommendations[2].DeviceName)
}

// Test that device must meet minimum requirements
// Skipped: requires SSH connection
func TestDeploymentService_RecommendDevices_InsufficientResources(t *testing.T) {
	t.Skip("Skipping test that requires SSH connection to devices")
	db := setupTestDB(t)
	credService, _ := NewCredentialService()

	// Create a device that doesn't meet minimum requirements
	device := &models.Device{
		ID:                 uuid.New(),
		Name:               "insufficient-device",
		IPAddress:          "192.168.1.101",
		Status:             models.DeviceStatusOnline,
		TotalRAMMB:         &[]int{512}[0],    // Only 512MB
		AvailableRAMMB:     &[]int{256}[0],
		TotalStorageGB:     &[]int{10}[0],     // Only 10GB
		AvailableStorageGB: &[]int{5}[0],
		CPUCores:           &[]int{1}[0],
	}
	require.NoError(t, db.Create(device).Error)

	recipeLoader := NewMockRecipeLoader(map[string]*models.Recipe{
		"heavy-app": {
			ID:   "heavy-app",
			Name: "Heavy App",
			Slug: "heavy-app",
			Requirements: models.RecipeRequirements{
				Memory: struct {
					Minimum     string `yaml:"minimum" json:"minimum"`
					Recommended string `yaml:"recommended" json:"recommended"`
				}{
					Minimum:     "4GB",  // Requires 4GB
					Recommended: "8GB",
				},
				Storage: struct {
					Minimum     string `yaml:"minimum" json:"minimum"`
					Recommended string `yaml:"recommended" json:"recommended"`
					Type        string `yaml:"type" json:"type"`
				}{
					Minimum:     "100GB",  // Requires 100GB
					Recommended: "200GB",
				},
			},
		},
	})

	deviceService := NewDeviceService(db, credService, nil)
	deploymentService := NewDeploymentService(db, nil, recipeLoader, deviceService, credService, nil)

	recommendations, err := deploymentService.RecommendDevicesForRecipe("heavy-app")
	require.NoError(t, err)
	require.Len(t, recommendations, 1)

	// Device should be marked as not available
	assert.False(t, recommendations[0].Available)
	assert.Equal(t, "not-recommended", recommendations[0].Recommendation)
	assert.Contains(t, recommendations[0].Reasons[0], "Insufficient RAM")
}

// Test RecipeRequirements struct parsing
func TestRecipeRequirements_Parsing(t *testing.T) {
	recipe := models.Recipe{
		Requirements: models.RecipeRequirements{
			Memory: struct {
				Minimum     string `yaml:"minimum" json:"minimum"`
				Recommended string `yaml:"recommended" json:"recommended"`
			}{
				Minimum:     "512MB",
				Recommended: "1GB",
			},
			Storage: struct {
				Minimum     string `yaml:"minimum" json:"minimum"`
				Recommended string `yaml:"recommended" json:"recommended"`
				Type        string `yaml:"type" json:"type"`
			}{
				Minimum:     "10GB",
				Recommended: "50GB",
				Type:        "ssd",
			},
			CPU: struct {
				MinimumCores     int `yaml:"minimum_cores" json:"minimum_cores"`
				RecommendedCores int `yaml:"recommended_cores" json:"recommended_cores"`
			}{
				MinimumCores:     2,
				RecommendedCores: 4,
			},
			Reliability: "high",
			AlwaysOn:    true,
		},
	}

	assert.Equal(t, "512MB", recipe.Requirements.Memory.Minimum)
	assert.Equal(t, "1GB", recipe.Requirements.Memory.Recommended)
	assert.Equal(t, "10GB", recipe.Requirements.Storage.Minimum)
	assert.Equal(t, "ssd", recipe.Requirements.Storage.Type)
	assert.Equal(t, 2, recipe.Requirements.CPU.MinimumCores)
	assert.Equal(t, "high", recipe.Requirements.Reliability)
	assert.True(t, recipe.Requirements.AlwaysOn)
}

// Test database config in recipe
func TestRecipeDatabaseConfig(t *testing.T) {
	recipe := models.Recipe{
		Database: models.RecipeDatabaseConfig{
			Engine:        "postgres",
			AutoProvision: true,
			Version:       "15",
			EnvPrefix:     "POSTGRES_",
		},
	}

	assert.Equal(t, "postgres", recipe.Database.Engine)
	assert.True(t, recipe.Database.AutoProvision)
	assert.Equal(t, "15", recipe.Database.Version)
	assert.Equal(t, "POSTGRES_", recipe.Database.EnvPrefix)
}
