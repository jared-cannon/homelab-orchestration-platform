package api

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/models"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/services"
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
	deployments.Get("/:id", h.GetDeployment)
	deployments.Delete("/:id", h.DeleteDeployment)
	deployments.Post("/:id/cancel", h.CancelDeployment)
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
