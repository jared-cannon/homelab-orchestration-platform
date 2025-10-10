package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/services"
)

// MarketplaceHandler handles marketplace-related API requests
type MarketplaceHandler struct {
	marketplaceService *services.MarketplaceService
	deviceScorer       *services.DeviceScorer
}

// NewMarketplaceHandler creates a new marketplace handler
func NewMarketplaceHandler(marketplaceService *services.MarketplaceService, deviceScorer *services.DeviceScorer) *MarketplaceHandler {
	return &MarketplaceHandler{
		marketplaceService: marketplaceService,
		deviceScorer:       deviceScorer,
	}
}

// RegisterRoutes RegisterMarketplaceRoutes registers all marketplace routes
func (h *MarketplaceHandler) RegisterRoutes(api fiber.Router) {
	marketplace := api.Group("/marketplace")

	marketplace.Get("/recipes", h.ListRecipes)
	marketplace.Get("/recipes/:slug", h.GetRecipe)
	marketplace.Post("/recipes/:slug/validate", h.ValidateDeployment)
	marketplace.Post("/recipes/:slug/recommend-device", h.RecommendDevice)
	marketplace.Get("/categories", h.GetCategories)
}

// ListRecipes godoc
// @Summary List all marketplace recipes
// @Description Get all available application recipes, optionally filtered by category
// @Tags marketplace
// @Produce json
// @Param category query string false "Filter by category"
// @Success 200 {array} models.Recipe
// @Failure 500 {object} ErrorResponse
// @Router /marketplace/recipes [get]
func (h *MarketplaceHandler) ListRecipes(c *fiber.Ctx) error {
	category := c.Query("category", "")

	recipes, err := h.marketplaceService.ListRecipes(category)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: "Failed to list recipes",
		})
	}

	return c.JSON(recipes)
}

// GetRecipe godoc
// @Summary Get a specific recipe
// @Description Get details of a specific recipe by slug
// @Tags marketplace
// @Produce json
// @Param slug path string true "Recipe slug"
// @Success 200 {object} models.Recipe
// @Failure 404 {object} ErrorResponse
// @Router /marketplace/recipes/{slug} [get]
func (h *MarketplaceHandler) GetRecipe(c *fiber.Ctx) error {
	slug := c.Params("slug")

	recipe, err := h.marketplaceService.GetRecipe(slug)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error: "Recipe not found",
		})
	}

	return c.JSON(recipe)
}

// ValidateDeploymentRequest represents a deployment validation request
type ValidateDeploymentRequest struct {
	DeviceID uuid.UUID              `json:"device_id"`
	Config   map[string]interface{} `json:"config"`
}

// ValidateDeployment godoc
// @Summary Validate a deployment
// @Description Validate that a recipe can be deployed with given configuration
// @Tags marketplace
// @Accept json
// @Produce json
// @Param slug path string true "Recipe slug"
// @Param body body ValidateDeploymentRequest true "Deployment configuration"
// @Success 200 {object} services.ValidationResult
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /marketplace/recipes/{slug}/validate [post]
func (h *MarketplaceHandler) ValidateDeployment(c *fiber.Ctx) error {
	slug := c.Params("slug")

	var req ValidateDeploymentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "Invalid request body",
		})
	}

	result, err := h.marketplaceService.ValidateDeployment(slug, req.DeviceID, req.Config)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: "Failed to validate deployment",
		})
	}

	return c.JSON(result)
}

// GetCategories godoc
// @Summary Get all recipe categories
// @Description Get list of all unique recipe categories
// @Tags marketplace
// @Produce json
// @Success 200 {array} string
// @Router /marketplace/categories [get]
func (h *MarketplaceHandler) GetCategories(c *fiber.Ctx) error {
	categories := h.marketplaceService.GetCategories()
	return c.JSON(categories)
}

// RecommendDevice godoc
// @Summary Recommend devices for a recipe
// @Description Score and rank all available devices for deploying a specific recipe
// @Tags marketplace
// @Produce json
// @Param slug path string true "Recipe slug"
// @Success 200 {array} services.DeviceScore
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /marketplace/recipes/{slug}/recommend-device [post]
func (h *MarketplaceHandler) RecommendDevice(c *fiber.Ctx) error {
	slug := c.Params("slug")

	// Get the recipe to extract requirements
	recipe, err := h.marketplaceService.GetRecipe(slug)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error: "Recipe not found",
		})
	}

	// Build requirements from recipe
	requirements := services.RecipeRequirements{
		MinRAMMB:     recipe.Resources.MinRAMMB,
		MinStorageGB: recipe.Resources.MinStorageGB,
		CPUCores:     recipe.Resources.CPUCores,
	}

	// Score devices
	scores, err := h.deviceScorer.ScoreDevicesForRecipe(requirements)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: "Failed to score devices",
		})
	}

	return c.JSON(scores)
}
