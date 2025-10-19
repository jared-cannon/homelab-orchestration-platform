package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	// testRecipeSlugPattern matches the pattern in the actual handler
	testRecipeSlugPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

// deploymentServiceInterface defines the interface for dependency checking (for testing)
type deploymentServiceInterface interface {
	CheckRecipeDependencies(recipeSlug string, deviceID uuid.UUID) (*services.DependencyCheckResult, error)
}

// MockDeploymentService is a mock for testing the handler
type MockDeploymentService struct {
	mock.Mock
}

func (m *MockDeploymentService) CheckRecipeDependencies(recipeSlug string, deviceID uuid.UUID) (*services.DependencyCheckResult, error) {
	args := m.Called(recipeSlug, deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.DependencyCheckResult), args.Error(1)
}

// testDeploymentHandler wraps the handler to use our interface
type testDeploymentHandler struct {
	service deploymentServiceInterface
}

func (h *testDeploymentHandler) CheckRecipeDependencies(c *fiber.Ctx) error {
	recipeSlug := strings.TrimSpace(c.Params("recipe_slug"))
	deviceIDStr := c.Params("device_id")

	// Validate recipe_slug parameter with regex whitelist
	// This prevents path traversal even after URL decoding (e.g., %2F becomes /)
	if recipeSlug == "" || len(recipeSlug) > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "Invalid recipe_slug parameter: must be 1-100 characters",
		})
	}
	if !testRecipeSlugPattern.MatchString(recipeSlug) {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "Invalid recipe_slug parameter: only alphanumeric characters, hyphens, and underscores allowed",
		})
	}

	// Parse device ID
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "Invalid device_id parameter",
		})
	}

	// Check dependencies
	result, err := h.service.CheckRecipeDependencies(recipeSlug, deviceID)
	if err != nil {
		// Return appropriate HTTP status based on typed errors
		if errors.Is(err, services.ErrRecipeNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error: err.Error(),
			})
		}
		if errors.Is(err, services.ErrRecipeValidationFailed) {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: err.Error(),
			})
		}
		if errors.Is(err, services.ErrDependencyServiceNotInit) {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Error: "Service initialization error",
			})
		}
		// Generic server error for unexpected errors (including ErrDependencyCheckFailed)
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: fmt.Sprintf("Failed to check dependencies: %v", err),
		})
	}

	return c.JSON(result)
}

// TestCheckRecipeDependencies_Success tests successful dependency check
func TestCheckRecipeDependencies_Success(t *testing.T) {
	app := fiber.New()
	mockService := new(MockDeploymentService)
	handler := &testDeploymentHandler{service: mockService}

	// Register route
	app.Get("/api/v1/deployments/check-dependencies/:recipe_slug/:device_id", handler.CheckRecipeDependencies)

	deviceID := uuid.New()
	expectedResult := &services.DependencyCheckResult{
		Satisfied: false,
		Missing: []services.MissingDependency{
			{
				Dependency: models.RecipeDependency{
					Type: "reverse_proxy",
					Name: "traefik",
				},
				Critical:     true,
				Reason:       "Not deployed",
				CanProvision: true,
			},
		},
		ToProvision: []services.ProvisionPlan{
			{
				Type:          "reverse_proxy",
				Name:          "Traefik",
				EstimatedTime: 30,
			},
		},
		Warnings:      []string{},
		EstimatedTime: 30,
		ResourceImpact: services.ResourceImpact{
			TotalRAMMB:     256,
			TotalStorageGB: 1,
			Breakdown:      "Traefik: 256MB RAM, 1GB storage",
		},
	}

	mockService.On("CheckRecipeDependencies", "nextcloud", deviceID).Return(expectedResult, nil)

	// Make request
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/deployments/check-dependencies/nextcloud/%s", deviceID), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result services.DependencyCheckResult
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.False(t, result.Satisfied)
	assert.Len(t, result.Missing, 1)
	assert.Equal(t, "reverse_proxy", result.Missing[0].Dependency.Type)

	mockService.AssertExpectations(t)
}

// TestCheckRecipeDependencies_RecipeNotFound tests 404 response for non-existent recipe
func TestCheckRecipeDependencies_RecipeNotFound(t *testing.T) {
	app := fiber.New()
	mockService := new(MockDeploymentService)
	handler := &testDeploymentHandler{service: mockService}

	app.Get("/api/v1/deployments/check-dependencies/:recipe_slug/:device_id", handler.CheckRecipeDependencies)

	deviceID := uuid.New()
	mockService.On("CheckRecipeDependencies", "non-existent", deviceID).
		Return(nil, fmt.Errorf("%w: non-existent", services.ErrRecipeNotFound))

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/deployments/check-dependencies/non-existent/%s", deviceID), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Should return 404
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

	var errorResp ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp.Error, "not found")

	mockService.AssertExpectations(t)
}

// TestCheckRecipeDependencies_InvalidRecipeSlug tests validation of recipe_slug parameter
func TestCheckRecipeDependencies_InvalidRecipeSlug(t *testing.T) {
	app := fiber.New()
	mockService := new(MockDeploymentService)
	handler := &testDeploymentHandler{service: mockService}

	app.Get("/api/v1/deployments/check-dependencies/:recipe_slug/:device_id", handler.CheckRecipeDependencies)

	deviceID := uuid.New()

	// Test recipe slug too long
	t.Run("recipe slug too long", func(t *testing.T) {
		longSlug := strings.Repeat("a", 101)
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/deployments/check-dependencies/%s/%s", longSlug, deviceID), nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var errorResp ErrorResponse
		err = json.NewDecoder(resp.Body).Decode(&errorResp)
		require.NoError(t, err)
		assert.Contains(t, errorResp.Error, "must be 1-100 characters")

		mockService.AssertNotCalled(t, "CheckRecipeDependencies")
	})
}

// TestCheckRecipeDependencies_InvalidDeviceID tests validation of device_id parameter
func TestCheckRecipeDependencies_InvalidDeviceID(t *testing.T) {
	app := fiber.New()
	mockService := new(MockDeploymentService)
	handler := &testDeploymentHandler{service: mockService}

	app.Get("/api/v1/deployments/check-dependencies/:recipe_slug/:device_id", handler.CheckRecipeDependencies)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/deployments/check-dependencies/nextcloud/not-a-uuid", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Should return 400
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var errorResp ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp.Error, "Invalid device_id")

	mockService.AssertNotCalled(t, "CheckRecipeDependencies")
}

// TestCheckRecipeDependencies_ValidationError tests 400 response for validation failures
func TestCheckRecipeDependencies_ValidationError(t *testing.T) {
	app := fiber.New()
	mockService := new(MockDeploymentService)
	handler := &testDeploymentHandler{service: mockService}

	app.Get("/api/v1/deployments/check-dependencies/:recipe_slug/:device_id", handler.CheckRecipeDependencies)

	deviceID := uuid.New()
	mockService.On("CheckRecipeDependencies", "invalid-recipe", deviceID).
		Return(nil, fmt.Errorf("%w: missing required field", services.ErrRecipeValidationFailed))

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/deployments/check-dependencies/invalid-recipe/%s", deviceID), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Should return 400 for validation errors
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var errorResp ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp.Error, "validation failed")

	mockService.AssertExpectations(t)
}

// TestCheckRecipeDependencies_InternalServerError tests 500 response for unexpected errors
func TestCheckRecipeDependencies_InternalServerError(t *testing.T) {
	app := fiber.New()
	mockService := new(MockDeploymentService)
	handler := &testDeploymentHandler{service: mockService}

	app.Get("/api/v1/deployments/check-dependencies/:recipe_slug/:device_id", handler.CheckRecipeDependencies)

	deviceID := uuid.New()
	mockService.On("CheckRecipeDependencies", "nextcloud", deviceID).
		Return(nil, fmt.Errorf("database connection error"))

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/deployments/check-dependencies/nextcloud/%s", deviceID), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Should return 500 for internal errors
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

	var errorResp ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp.Error, "Failed to check dependencies")

	mockService.AssertExpectations(t)
}

// TestCheckRecipeDependencies_NoDependencies tests response when recipe has no dependencies
func TestCheckRecipeDependencies_NoDependencies(t *testing.T) {
	app := fiber.New()
	mockService := new(MockDeploymentService)
	handler := &testDeploymentHandler{service: mockService}

	app.Get("/api/v1/deployments/check-dependencies/:recipe_slug/:device_id", handler.CheckRecipeDependencies)

	deviceID := uuid.New()
	expectedResult := &services.DependencyCheckResult{
		Satisfied:      true,
		Missing:        []services.MissingDependency{},
		ToProvision:    []services.ProvisionPlan{},
		Warnings:       []string{},
		EstimatedTime:  0,
		ResourceImpact: services.ResourceImpact{},
	}

	mockService.On("CheckRecipeDependencies", "simple-app", deviceID).Return(expectedResult, nil)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/deployments/check-dependencies/simple-app/%s", deviceID), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result services.DependencyCheckResult
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.True(t, result.Satisfied)
	assert.Empty(t, result.Missing)
	assert.Empty(t, result.ToProvision)

	mockService.AssertExpectations(t)
}

// TestCheckRecipeDependencies_ValidRecipeSlugFormats tests various valid recipe slug formats
func TestCheckRecipeDependencies_ValidRecipeSlugFormats(t *testing.T) {
	validSlugs := []string{
		"nextcloud",
		"nginx-proxy-manager",
		"pi-hole",
		"home_assistant",
		"jellyfin-2024",
		"app123",
	}

	for _, slug := range validSlugs {
		t.Run(slug, func(t *testing.T) {
			app := fiber.New()
			mockService := new(MockDeploymentService)
			handler := &testDeploymentHandler{service: mockService}

			app.Get("/api/v1/deployments/check-dependencies/:recipe_slug/:device_id", handler.CheckRecipeDependencies)

			deviceID := uuid.New()
			expectedResult := &services.DependencyCheckResult{
				Satisfied:      true,
				Missing:        []services.MissingDependency{},
				ToProvision:    []services.ProvisionPlan{},
				Warnings:       []string{},
				EstimatedTime:  0,
				ResourceImpact: services.ResourceImpact{},
			}

			mockService.On("CheckRecipeDependencies", slug, deviceID).Return(expectedResult, nil)

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/deployments/check-dependencies/%s/%s", slug, deviceID), nil)
			resp, err := app.Test(req)
			require.NoError(t, err)

			// Should succeed for valid slugs
			assert.Equal(t, fiber.StatusOK, resp.StatusCode)

			mockService.AssertExpectations(t)
		})
	}
}

// TestCheckRecipeDependencies_PathTraversalProtection tests protection against path traversal attacks
func TestCheckRecipeDependencies_PathTraversalProtection(t *testing.T) {
	maliciousSlugs := []struct {
		name string
		slug string
	}{
		{"URL-encoded forward slash", "..%2Fetc%2Fpasswd"},
		{"Mixed encoding", "..%2F..%2Fsecret"},
		{"Backslash in slug", "nextcloud\\\\secret"},
		{"Dot-dot in slug", ".."},
	}

	for _, tc := range maliciousSlugs {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			mockService := new(MockDeploymentService)
			handler := &testDeploymentHandler{service: mockService}

			app.Get("/api/v1/deployments/check-dependencies/:recipe_slug/:device_id", handler.CheckRecipeDependencies)

			deviceID := uuid.New()

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/deployments/check-dependencies/%s/%s", tc.slug, deviceID), nil)
			resp, err := app.Test(req)
			require.NoError(t, err)

			// Should return 400 for malicious slugs
			assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

			var errorResp ErrorResponse
			err = json.NewDecoder(resp.Body).Decode(&errorResp)
			require.NoError(t, err)
			// Error message should indicate validation failure
			assert.True(t,
				strings.Contains(errorResp.Error, "alphanumeric") ||
				strings.Contains(errorResp.Error, "illegal characters") ||
				strings.Contains(errorResp.Error, "1-100 characters"),
				"Error should mention validation: %s", errorResp.Error)

			mockService.AssertNotCalled(t, "CheckRecipeDependencies")
		})
	}
}

// TestCheckRecipeDependencies_SpecialCharactersRejection tests rejection of special characters
func TestCheckRecipeDependencies_SpecialCharactersRejection(t *testing.T) {
	invalidSlugs := []struct {
		name string
		slug string
	}{
		{"Dot-dot", ".."},
		{"With exclamation", "app!test"},
		{"With equals", "app=test"},
	}

	for _, tc := range invalidSlugs {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			mockService := new(MockDeploymentService)
			handler := &testDeploymentHandler{service: mockService}

			app.Get("/api/v1/deployments/check-dependencies/:recipe_slug/:device_id", handler.CheckRecipeDependencies)

			deviceID := uuid.New()

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/deployments/check-dependencies/%s/%s", tc.slug, deviceID), nil)
			resp, err := app.Test(req)
			require.NoError(t, err)

			// Should return 400 for invalid slugs
			assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

			var errorResp ErrorResponse
			err = json.NewDecoder(resp.Body).Decode(&errorResp)
			require.NoError(t, err)
			assert.Contains(t, errorResp.Error, "alphanumeric")

			mockService.AssertNotCalled(t, "CheckRecipeDependencies")
		})
	}
}
