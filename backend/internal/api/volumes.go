package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/models"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/services"
)

// VolumeHandler handles Docker volume management HTTP requests
type VolumeHandler struct {
	service *services.VolumeService
}

// NewVolumeHandler creates a new volume handler
func NewVolumeHandler(service *services.VolumeService) *VolumeHandler {
	return &VolumeHandler{service: service}
}

// CreateVolumeRequest represents the request body for creating a volume
type CreateVolumeRequest struct {
	Name        string            `json:"name" validate:"required"`
	Type        models.VolumeType `json:"type" validate:"required"`
	NFSServerIP string            `json:"nfs_server_ip,omitempty"` // Required if type is "nfs"
	NFSPath     string            `json:"nfs_path,omitempty"`      // Required if type is "nfs"
	Options     map[string]string `json:"options,omitempty"`       // NFS mount options
}

// RemoveVolumeRequest represents query parameters for removing a volume
type RemoveVolumeRequest struct {
	Force bool `json:"force"`
}

// ListVolumes handles GET /api/v1/devices/:id/volumes
func (h *VolumeHandler) ListVolumes(c *fiber.Ctx) error {
	deviceID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid device ID",
		})
	}

	volumes, err := h.service.ListVolumes(deviceID)
	if err != nil {
		return HandleError(c, 500, err, "Failed to list volumes")
	}

	return c.JSON(volumes)
}

// CreateVolume handles POST /api/v1/devices/:id/volumes
func (h *VolumeHandler) CreateVolume(c *fiber.Ctx) error {
	deviceID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid device ID",
		})
	}

	var req CreateVolumeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if err := ValidateRequest(c, &req); err != nil {
		return err
	}

	var volume *models.Volume

	switch req.Type {
	case models.VolumeTypeLocal:
		volume, err = h.service.CreateLocalVolume(deviceID, req.Name)
	case models.VolumeTypeNFS:
		// Validate NFS-specific fields
		if req.NFSServerIP == "" || req.NFSPath == "" {
			return c.Status(400).JSON(fiber.Map{
				"error": "nfs_server_ip and nfs_path are required for NFS volumes",
			})
		}
		volume, err = h.service.CreateNFSVolume(deviceID, req.Name, req.NFSServerIP, req.NFSPath, req.Options)
	default:
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid volume type",
		})
	}

	if err != nil {
		return HandleError(c, 500, err, "Failed to create volume")
	}

	return c.Status(201).JSON(volume)
}

// GetVolume handles GET /api/v1/devices/:id/volumes/:name
func (h *VolumeHandler) GetVolume(c *fiber.Ctx) error {
	deviceID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid device ID",
		})
	}

	volumeName := c.Params("name")

	volume, err := h.service.GetVolume(deviceID, volumeName)
	if err != nil {
		return HandleError(c, 404, err, "Volume not found")
	}

	return c.JSON(volume)
}

// InspectVolume handles GET /api/v1/devices/:id/volumes/:name/inspect
func (h *VolumeHandler) InspectVolume(c *fiber.Ctx) error {
	deviceID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid device ID",
		})
	}

	volumeName := c.Params("name")

	details, err := h.service.InspectVolume(deviceID, volumeName)
	if err != nil {
		return HandleError(c, 500, err, "Failed to inspect volume")
	}

	return c.JSON(details)
}

// RemoveVolume handles DELETE /api/v1/devices/:id/volumes/:name
func (h *VolumeHandler) RemoveVolume(c *fiber.Ctx) error {
	deviceID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid device ID",
		})
	}

	volumeName := c.Params("name")
	force := c.QueryBool("force", false)

	if err := h.service.RemoveVolume(deviceID, volumeName, force); err != nil {
		return HandleError(c, 500, err, "Failed to remove volume")
	}

	return c.Status(204).Send(nil)
}

// RegisterRoutes registers all volume routes
func (h *VolumeHandler) RegisterRoutes(api fiber.Router) {
	// Volume routes are nested under devices
	// They will be registered as /api/v1/devices/:id/volumes
}
