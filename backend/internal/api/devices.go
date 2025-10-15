package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/services"
)

// DeviceHandler handles device-related HTTP requests
type DeviceHandler struct {
	service *services.DeviceService
}

// NewDeviceHandler creates a new device handler
func NewDeviceHandler(service *services.DeviceService) *DeviceHandler {
	return &DeviceHandler{service: service}
}

// CreateDeviceRequest represents the request body for creating a device
type CreateDeviceRequest struct {
	Name       string                       `json:"name" validate:"required"`
	Type       models.DeviceType            `json:"type" validate:"required"`
	IPAddress  string                       `json:"ip_address" validate:"required"`
	MACAddress string                       `json:"mac_address,omitempty"`
	Metadata   map[string]interface{}       `json:"metadata,omitempty"`
	Credentials services.DeviceCredentials `json:"credentials" validate:"required"`
}

// TestConnectionRequest represents the request body for testing a connection
type TestConnectionRequest struct {
	IPAddress  string                       `json:"ip_address" validate:"required"`
	Credentials services.DeviceCredentials `json:"credentials" validate:"required"`
}

// UpdateDeviceRequest represents the request body for updating a device
type UpdateDeviceRequest struct {
	Name       *string                `json:"name,omitempty"`
	Type       *models.DeviceType     `json:"type,omitempty"`
	IPAddress  *string                `json:"ip_address,omitempty"`
	MACAddress *string                `json:"mac_address,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateCredentialsRequest represents the request body for updating device credentials
type UpdateCredentialsRequest struct {
	Credentials services.DeviceCredentials `json:"credentials" validate:"required"`
}

// ListDevices handles GET /api/v1/devices
func (h *DeviceHandler) ListDevices(c *fiber.Ctx) error {
	devices, err := h.service.ListDevices()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to list devices",
		})
	}

	return c.JSON(devices)
}

// CreateDevice handles POST /api/v1/devices
func (h *DeviceHandler) CreateDevice(c *fiber.Ctx) error {
	var req CreateDeviceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if err := ValidateRequest(c, &req); err != nil {
		return err
	}

	device := &models.Device{
		Name:       req.Name,
		Type:       req.Type,
		IPAddress:  req.IPAddress,
		MACAddress: req.MACAddress,
		Status:     models.DeviceStatusUnknown,
	}

	if err := h.service.CreateDevice(device, &req.Credentials); err != nil {
		return HandleError(c, 400, err, "Failed to create device")
	}

	return c.Status(201).JSON(device)
}

// GetDevice handles GET /api/v1/devices/:id
func (h *DeviceHandler) GetDevice(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid device ID",
		})
	}

	device, err := h.service.GetDevice(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Device not found",
		})
	}

	return c.JSON(device)
}

// UpdateDevice handles PATCH /api/v1/devices/:id
func (h *DeviceHandler) UpdateDevice(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid device ID",
		})
	}

	var req UpdateDeviceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Type != nil {
		updates["type"] = *req.Type
	}
	if req.IPAddress != nil {
		updates["ip_address"] = *req.IPAddress
	}
	if req.MACAddress != nil {
		updates["mac_address"] = *req.MACAddress
	}
	if req.Metadata != nil {
		updates["metadata"] = req.Metadata
	}

	if err := h.service.UpdateDevice(id, updates); err != nil {
		return HandleError(c, 400, err, "Failed to update device")
	}

	device, _ := h.service.GetDevice(id)
	return c.JSON(device)
}

// DeleteDevice handles DELETE /api/v1/devices/:id
func (h *DeviceHandler) DeleteDevice(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid device ID",
		})
	}

	if err := h.service.DeleteDevice(id); err != nil {
		return HandleError(c, 500, err, "Failed to delete device")
	}

	return c.Status(204).Send(nil)
}

// TestConnectionBeforeCreate handles POST /api/v1/devices/test
func (h *DeviceHandler) TestConnectionBeforeCreate(c *fiber.Ctx) error {
	var req TestConnectionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if err := ValidateRequest(c, &req); err != nil {
		return err
	}

	result, err := h.service.TestConnectionWithCredentials(req.IPAddress, &req.Credentials)
	if err != nil {
		return HandleErrorWithDetails(c, 400, err, "Connection test failed", fiber.Map{
			"success": false,
			"details": result,
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"details": result,
	})
}

// TestConnection handles POST /api/v1/devices/:id/test-connection
func (h *DeviceHandler) TestConnection(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid device ID",
		})
	}

	result, err := h.service.TestConnection(id)
	if err != nil {
		return HandleErrorWithDetails(c, 400, err, "Connection test failed", result)
	}

	return c.JSON(result)
}

// UpdateDeviceCredentials handles PATCH /api/v1/devices/:id/credentials
func (h *DeviceHandler) UpdateDeviceCredentials(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid device ID",
		})
	}

	var req UpdateCredentialsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if err := ValidateRequest(c, &req); err != nil {
		return err
	}

	if err := h.service.UpdateDeviceCredentials(id, &req.Credentials); err != nil {
		return HandleError(c, 400, err, "Failed to update credentials")
	}

	return c.JSON(fiber.Map{
		"message": "Credentials updated successfully",
	})
}

// RegisterRoutes registers all device routes
func (h *DeviceHandler) RegisterRoutes(api fiber.Router) {
	devices := api.Group("/devices")

	devices.Get("/", h.ListDevices)
	devices.Post("/", h.CreateDevice)
	devices.Post("/test", h.TestConnectionBeforeCreate) // Must be before /:id routes
	devices.Get("/:id", h.GetDevice)
	devices.Patch("/:id", h.UpdateDevice)
	devices.Delete("/:id", h.DeleteDevice)
	devices.Post("/:id/test-connection", h.TestConnection)
	devices.Patch("/:id/credentials", h.UpdateDeviceCredentials)
}
