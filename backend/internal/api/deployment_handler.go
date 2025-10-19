package api

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/services"
)

var (
	// recipeSlugPattern defines allowed characters for recipe slugs (alphanumeric, hyphens, underscores)
	// This prevents path traversal attacks even after URL decoding
	recipeSlugPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

// DeploymentHandler handles deployment-related HTTP requests
type DeploymentHandler struct {
	deploymentService *services.DeploymentService
}

// NewDeploymentHandler creates a new deployment handler
func NewDeploymentHandler(deploymentService *services.DeploymentService) *DeploymentHandler {
	return &DeploymentHandler{
		deploymentService: deploymentService,
	}
}

// RegisterRoutes registers deployment routes
func (h *DeploymentHandler) RegisterRoutes(router fiber.Router) {
	deployments := router.Group("/deployments")
	deployments.Get("", h.ListDeployments)
	deployments.Post("", h.CreateDeployment)
	deployments.Delete("/cleanup", h.CleanupDeployments)
	deployments.Get("/check-dependencies/:recipe_slug/:device_id", h.CheckRecipeDependencies)
	deployments.Get("/:id", h.GetDeployment)
	deployments.Delete("/:id", h.DeleteDeployment)
	deployments.Post("/:id/cancel", h.CancelDeployment)
	deployments.Post("/:id/restart", h.RestartDeployment)
	deployments.Post("/:id/stop", h.StopDeployment)
	deployments.Post("/:id/start", h.StartDeployment)
	deployments.Get("/:id/urls", h.GetAccessURLs)
	deployments.Get("/:id/troubleshoot", h.TroubleshootDeployment)
}

// CreateDeployment creates a new deployment
func (h *DeploymentHandler) CreateDeployment(c *fiber.Ctx) error {
	var req services.CreateDeploymentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "Invalid request body",
		})
	}

	deployment, err := h.deploymentService.CreateDeployment(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: fmt.Sprintf("Failed to create deployment: %v", err),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(deployment)
}

// GetDeployment retrieves a deployment by ID
func (h *DeploymentHandler) GetDeployment(c *fiber.Ctx) error {
	id := c.Params("id")

	deployment, err := h.deploymentService.GetDeployment(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error: "Deployment not found",
		})
	}

	return c.JSON(deployment)
}

// ListDeployments lists all deployments with optional filters
func (h *DeploymentHandler) ListDeployments(c *fiber.Ctx) error {
	var deviceID *uuid.UUID
	var status *models.DeploymentStatus

	// Parse device_id query parameter
	if deviceIDStr := c.Query("device_id"); deviceIDStr != "" {
		parsedID, err := uuid.Parse(deviceIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "Invalid device_id parameter",
			})
		}
		deviceID = &parsedID
	}

	// Parse status query parameter
	if statusStr := c.Query("status"); statusStr != "" {
		statusValue := models.DeploymentStatus(statusStr)
		status = &statusValue
	}

	deployments, err := h.deploymentService.ListDeployments(deviceID, status)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: fmt.Sprintf("Failed to list deployments: %v", err),
		})
	}

	return c.JSON(deployments)
}

// DeleteDeployment removes a deployment
func (h *DeploymentHandler) DeleteDeployment(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.deploymentService.DeleteDeployment(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: fmt.Sprintf("Failed to delete deployment: %v", err),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// CancelDeployment cancels a running or pending deployment
func (h *DeploymentHandler) CancelDeployment(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.deploymentService.CancelDeployment(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: fmt.Sprintf("Failed to cancel deployment: %v", err),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// RestartDeployment restarts a deployment
func (h *DeploymentHandler) RestartDeployment(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.deploymentService.RestartDeployment(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: fmt.Sprintf("Failed to restart deployment: %v", err),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// StopDeployment stops a deployment
func (h *DeploymentHandler) StopDeployment(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.deploymentService.StopDeployment(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: fmt.Sprintf("Failed to stop deployment: %v", err),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// StartDeployment starts a stopped deployment
func (h *DeploymentHandler) StartDeployment(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.deploymentService.StartDeployment(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: fmt.Sprintf("Failed to start deployment: %v", err),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetAccessURLs returns access URLs for a deployment
func (h *DeploymentHandler) GetAccessURLs(c *fiber.Ctx) error {
	id := c.Params("id")

	urls, err := h.deploymentService.GetAccessURLs(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: fmt.Sprintf("Failed to get access URLs: %v", err),
		})
	}

	return c.JSON(urls)
}

// TroubleshootDeployment provides troubleshooting information for a deployment
func (h *DeploymentHandler) TroubleshootDeployment(c *fiber.Ctx) error {
	id := c.Params("id")

	troubleshoot, err := h.deploymentService.TroubleshootDeployment(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: fmt.Sprintf("Failed to troubleshoot deployment: %v", err),
		})
	}

	return c.JSON(troubleshoot)
}

// CleanupDeployments bulk deletes deployments by status
func (h *DeploymentHandler) CleanupDeployments(c *fiber.Ctx) error {
	// Parse status query parameter (defaults to "failed")
	status := c.Query("status", "failed")
	deploymentStatus := models.DeploymentStatus(status)

	// Call service to delete deployments
	count, err := h.deploymentService.BulkDeleteDeployments(deploymentStatus)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: fmt.Sprintf("Failed to cleanup deployments: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"deleted_count": count,
		"status":        status,
	})
}

// CheckRecipeDependencies checks dependencies for a recipe before deployment
// Allows frontend to preview what dependencies will be auto-provisioned
func (h *DeploymentHandler) CheckRecipeDependencies(c *fiber.Ctx) error {
	recipeSlug := strings.TrimSpace(c.Params("recipe_slug"))
	deviceIDStr := c.Params("device_id")

	// Validate recipe_slug parameter with regex whitelist
	// This prevents path traversal even after URL decoding (e.g., %2F becomes /)
	if recipeSlug == "" || len(recipeSlug) > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "Invalid recipe_slug parameter: must be 1-100 characters",
		})
	}
	if !recipeSlugPattern.MatchString(recipeSlug) {
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

	// TODO: When multi-user support is implemented, verify that the current user owns the device
	// Example:
	// device, err := h.deviceService.GetDevice(deviceID)
	// if err != nil || device.UserID != currentUserID {
	//     return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
	//         Error: "Access denied",
	//     })
	// }

	// Check dependencies
	result, err := h.deploymentService.CheckRecipeDependencies(recipeSlug, deviceID)
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
