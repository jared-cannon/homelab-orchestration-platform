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

	var software *models.InstalledSoftware

	switch req.SoftwareType {
	case models.SoftwareDocker:
		software, err = h.service.InstallDocker(deviceID, req.AddUserToGroup)
	case models.SoftwareNFSServer:
		software, err = h.service.InstallNFSServer(deviceID)
	case models.SoftwareNFSClient:
		software, err = h.service.InstallNFSClient(deviceID)
	default:
		return c.Status(400).JSON(fiber.Map{
			"error": "Unsupported software type",
		})
	}

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

// RegisterRoutes registers all software routes
func (h *SoftwareHandler) RegisterRoutes(api fiber.Router) {
	// Software management routes are nested under devices
	// They will be registered as /api/v1/devices/:id/software
}
