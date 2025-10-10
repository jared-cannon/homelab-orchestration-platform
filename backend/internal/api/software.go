package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/models"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/services"
)

// SoftwareHandler handles software management HTTP requests
type SoftwareHandler struct {
	service *services.SoftwareService
}

// NewSoftwareHandler creates a new software handler
func NewSoftwareHandler(service *services.SoftwareService) *SoftwareHandler {
	return &SoftwareHandler{service: service}
}

// InstallSoftwareRequest represents the request body for installing software
type InstallSoftwareRequest struct {
	SoftwareType   models.SoftwareType `json:"software_type" validate:"required"`
	AddUserToGroup bool                `json:"add_user_to_group,omitempty"` // For Docker only
}

// ListInstalled handles GET /api/v1/devices/:id/software
func (h *SoftwareHandler) ListInstalled(c *fiber.Ctx) error {
	deviceID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid device ID",
		})
	}

	software, err := h.service.ListInstalled(deviceID)
	if err != nil {
		return HandleError(c, 500, err, "Failed to list installed software")
	}

	return c.JSON(software)
}

// InstallSoftware handles POST /api/v1/devices/:id/software
func (h *SoftwareHandler) InstallSoftware(c *fiber.Ctx) error {
	deviceID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid device ID",
		})
	}

	var req InstallSoftwareRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if err := ValidateRequest(c, &req); err != nil {
		return err
	}

	// Build options map from request
	options := make(map[string]interface{})
	if req.AddUserToGroup {
		options["add_user_to_group"] = true
	}

	// Use plugin-based installation system
	software, err := h.service.Install(deviceID, req.SoftwareType, options)
	if err != nil {
		return HandleError(c, 500, err, "Failed to install software")
	}

	return c.Status(201).JSON(software)
}

// UninstallSoftware handles DELETE /api/v1/devices/:id/software/:name
func (h *SoftwareHandler) UninstallSoftware(c *fiber.Ctx) error {
	deviceID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid device ID",
		})
	}

	softwareName := models.SoftwareType(c.Params("name"))

	if err := h.service.Uninstall(deviceID, softwareName); err != nil {
		return HandleError(c, 500, err, "Failed to uninstall software")
	}

	return c.Status(204).Send(nil)
}

// DetectInstalled handles POST /api/v1/devices/:id/software/detect
func (h *SoftwareHandler) DetectInstalled(c *fiber.Ctx) error {
	deviceID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid device ID",
		})
	}

	software, err := h.service.DetectInstalled(deviceID)
	if err != nil {
		return HandleError(c, 500, err, "Failed to detect installed software")
	}

	return c.JSON(software)
}

// CheckUpdates handles GET /api/v1/devices/:id/software/updates
func (h *SoftwareHandler) CheckUpdates(c *fiber.Ctx) error {
	deviceID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid device ID",
		})
	}

	updates, err := h.service.CheckUpdates(deviceID)
	if err != nil {
		return HandleError(c, 500, err, "Failed to check for updates")
	}

	return c.JSON(updates)
}

// UpdateSoftware handles POST /api/v1/devices/:id/software/:name/update
func (h *SoftwareHandler) UpdateSoftware(c *fiber.Ctx) error {
	deviceID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid device ID",
		})
	}

	softwareName := models.SoftwareType(c.Params("name"))

	software, err := h.service.UpdateSoftware(deviceID, softwareName)
	if err != nil {
		return HandleError(c, 500, err, "Failed to update software")
	}

	return c.JSON(software)
}

// ListAvailable handles GET /api/v1/software/available
func (h *SoftwareHandler) ListAvailable(c *fiber.Ctx) error {
	definitions := h.service.ListAvailableSoftware()
	return c.JSON(definitions)
}

// RegisterRoutes registers all software routes
func (h *SoftwareHandler) RegisterRoutes(api fiber.Router) {
	// Software management routes are nested under devices
	// They will be registered as /api/v1/devices/:id/software
}
