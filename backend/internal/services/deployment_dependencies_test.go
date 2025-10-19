package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeploymentService_CheckRecipeDependencies_Success(t *testing.T) {
	db := setupTestDB(t)
	credService, _ := NewCredentialService()

	// Create a test device
	device := &models.Device{
		ID:        uuid.New(),
		Name:      "test-device",
		LocalIPAddress: "192.168.1.100",
		Status:    models.DeviceStatusOnline,
	}
	require.NoError(t, db.Create(device).Error)

	// Create a recipe with dependencies
	recipeWithDeps := &models.Recipe{
		ID:   "test-app",
		Name: "Test App",
		Slug: "test-app",
		Dependencies: models.RecipeDependencies{
			Required: []models.RecipeDependency{
				{
					Type: "reverse_proxy",
					Name: "traefik",
				},
			},
		},
		ComposeContent: "version: '3.8'\nservices:\n  app:\n    image: test:latest",
	}

	mockRecipeLoader := NewMockRecipeLoader(map[string]*models.Recipe{
		"test-app": recipeWithDeps,
	})

	deviceService := NewDeviceService(db, credService, nil)
	deploymentService := NewDeploymentService(db, nil, mockRecipeLoader, deviceService, credService, nil, nil, nil)

	// Check dependencies
	result, err := deploymentService.CheckRecipeDependencies("test-app", device.ID)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify the result structure
	assert.False(t, result.Satisfied) // Dependencies not yet satisfied
	assert.NotEmpty(t, result.Missing)
}

func TestDeploymentService_CheckRecipeDependencies_RecipeNotFound(t *testing.T) {
	db := setupTestDB(t)
	credService, _ := NewCredentialService()

	device := &models.Device{
		ID:        uuid.New(),
		Name:      "test-device",
		LocalIPAddress: "192.168.1.100",
	}
	require.NoError(t, db.Create(device).Error)

	mockRecipeLoader := NewMockRecipeLoader(map[string]*models.Recipe{})
	deviceService := NewDeviceService(db, credService, nil)
	deploymentService := NewDeploymentService(db, nil, mockRecipeLoader, deviceService, credService, nil, nil, nil)

	// Try to check dependencies for non-existent recipe
	result, err := deploymentService.CheckRecipeDependencies("non-existent", device.ID)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "recipe not found")
}

func TestDeploymentService_CheckRecipeDependencies_InvalidRecipe(t *testing.T) {
	db := setupTestDB(t)
	credService, _ := NewCredentialService()

	device := &models.Device{
		ID:        uuid.New(),
		Name:      "test-device",
		LocalIPAddress: "192.168.1.100",
	}
	require.NoError(t, db.Create(device).Error)

	// Create an invalid recipe (missing required fields)
	invalidRecipe := &models.Recipe{
		// Missing required fields like ID, Name, Slug
		ComposeContent: "version: '3.8'\nservices:\n  app:\n    image: test:latest",
	}

	mockRecipeLoader := NewMockRecipeLoader(map[string]*models.Recipe{
		"invalid-app": invalidRecipe,
	})

	deviceService := NewDeviceService(db, credService, nil)
	deploymentService := NewDeploymentService(db, nil, mockRecipeLoader, deviceService, credService, nil, nil, nil)

	// Check dependencies should fail due to validation
	result, err := deploymentService.CheckRecipeDependencies("invalid-app", device.ID)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestDeploymentService_CheckRecipeDependencies_NoDependencies(t *testing.T) {
	db := setupTestDB(t)
	credService, _ := NewCredentialService()

	device := &models.Device{
		ID:        uuid.New(),
		Name:      "test-device",
		LocalIPAddress: "192.168.1.100",
	}
	require.NoError(t, db.Create(device).Error)

	// Recipe without dependencies
	recipeNoDeps := &models.Recipe{
		ID:             "simple-app",
		Name:           "Simple App",
		Slug:           "simple-app",
		Dependencies:   models.RecipeDependencies{}, // No dependencies
		ComposeContent: "version: '3.8'\nservices:\n  app:\n    image: test:latest",
	}

	mockRecipeLoader := NewMockRecipeLoader(map[string]*models.Recipe{
		"simple-app": recipeNoDeps,
	})

	deviceService := NewDeviceService(db, credService, nil)
	deploymentService := NewDeploymentService(db, nil, mockRecipeLoader, deviceService, credService, nil, nil, nil)

	// Check dependencies
	result, err := deploymentService.CheckRecipeDependencies("simple-app", device.ID)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Should be satisfied (no dependencies to check)
	assert.True(t, result.Satisfied)
	assert.Empty(t, result.Missing)
	assert.Empty(t, result.ToProvision)
}

func TestDeploymentService_CheckRecipeDependencies_NilDependencyService(t *testing.T) {
	db := setupTestDB(t)
	credService, _ := NewCredentialService()

	device := &models.Device{
		ID:        uuid.New(),
		Name:      "test-device",
		LocalIPAddress: "192.168.1.100",
	}
	require.NoError(t, db.Create(device).Error)

	recipeWithDeps := &models.Recipe{
		ID:   "test-app",
		Name: "Test App",
		Slug: "test-app",
		Dependencies: models.RecipeDependencies{
			Required: []models.RecipeDependency{
				{Type: "reverse_proxy", Name: "traefik"},
			},
		},
		ComposeContent: "version: '3.8'\nservices:\n  app:\n    image: test:latest",
	}

	mockRecipeLoader := NewMockRecipeLoader(map[string]*models.Recipe{
		"test-app": recipeWithDeps,
	})

	deviceService := NewDeviceService(db, credService, nil)

	// Create deployment service manually without dependency service
	deploymentService := &DeploymentService{
		db:                db,
		recipeLoader:      mockRecipeLoader,
		deviceService:     deviceService,
		dependencyService: nil, // Explicitly nil
	}

	// Should fail with appropriate error
	result, err := deploymentService.CheckRecipeDependencies("test-app", device.ID)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "dependency service not initialized")
}

func TestDeploymentService_CheckRecipeDependencies_WithRecommendedDependencies(t *testing.T) {
	db := setupTestDB(t)
	credService, _ := NewCredentialService()

	device := &models.Device{
		ID:        uuid.New(),
		Name:      "test-device",
		LocalIPAddress: "192.168.1.100",
	}
	require.NoError(t, db.Create(device).Error)

	// Recipe with both required and recommended dependencies
	recipeWithBothDeps := &models.Recipe{
		ID:   "full-app",
		Name: "Full App",
		Slug: "full-app",
		Dependencies: models.RecipeDependencies{
			Required: []models.RecipeDependency{
				{Type: "reverse_proxy", Name: "traefik"},
			},
			Recommended: []models.RecipeDependency{
				{Type: "cache", Name: "redis", MinVersion: "7"},
			},
		},
		ComposeContent: "version: '3.8'\nservices:\n  app:\n    image: test:latest",
	}

	mockRecipeLoader := NewMockRecipeLoader(map[string]*models.Recipe{
		"full-app": recipeWithBothDeps,
	})

	deviceService := NewDeviceService(db, credService, nil)
	deploymentService := NewDeploymentService(db, nil, mockRecipeLoader, deviceService, credService, nil, nil, nil)

	// Check dependencies
	result, err := deploymentService.CheckRecipeDependencies("full-app", device.ID)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Should have missing dependencies
	assert.False(t, result.Satisfied)
	assert.NotEmpty(t, result.Missing)
}

func TestDeploymentService_CheckRecipeDependencies_InvalidDeviceID(t *testing.T) {
	db := setupTestDB(t)
	credService, _ := NewCredentialService()

	recipeSimple := &models.Recipe{
		ID:             "simple-app",
		Name:           "Simple App",
		Slug:           "simple-app",
		ComposeContent: "version: '3.8'\nservices:\n  app:\n    image: test:latest",
	}

	mockRecipeLoader := NewMockRecipeLoader(map[string]*models.Recipe{
		"simple-app": recipeSimple,
	})

	deviceService := NewDeviceService(db, credService, nil)
	deploymentService := NewDeploymentService(db, nil, mockRecipeLoader, deviceService, credService, nil, nil, nil)

	// Use a non-existent device ID
	nonExistentDeviceID := uuid.New()

	// Should succeed for simple recipe without dependencies (doesn't need to query device)
	result, err := deploymentService.CheckRecipeDependencies("simple-app", nonExistentDeviceID)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Satisfied)
}
