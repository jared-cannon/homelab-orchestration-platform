package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/services"
)

// NFSHandler handles NFS management HTTP requests
type NFSHandler struct {
	service *services.NFSService
}

// NewNFSHandler creates a new NFS handler
func NewNFSHandler(service *services.NFSService) *NFSHandler {
	return &NFSHandler{service: service}
}

// SetupNFSServerRequest represents the request body for setting up an NFS server
type SetupNFSServerRequest struct {
	ExportPath string `json:"export_path" validate:"required"`
	ClientCIDR string `json:"client_cidr,omitempty"` // defaults to "*"
	Options    string `json:"options,omitempty"`     // defaults to "rw,sync,no_subtree_check,no_root_squash"
}

// CreateExportRequest represents the request body for creating an NFS export
type CreateExportRequest struct {
	ExportPath string `json:"export_path" validate:"required"`
	ClientCIDR string `json:"client_cidr,omitempty"`
	Options    string `json:"options,omitempty"`
}

// MountNFSShareRequest represents the request body for mounting an NFS share
type MountNFSShareRequest struct {
	ServerIP   string `json:"server_ip" validate:"required"`
	RemotePath string `json:"remote_path" validate:"required"`
	LocalPath  string `json:"local_path" validate:"required"`
	Options    string `json:"options,omitempty"`
	Permanent  bool   `json:"permanent"`
}

// UnmountNFSShareRequest represents query parameters for unmounting
type UnmountNFSShareRequest struct {
	RemoveFromFstab bool `json:"remove_from_fstab"`
}

// SetupServer handles POST /api/v1/devices/:id/nfs/server/setup
func (h *NFSHandler) SetupServer(c *fiber.Ctx) error {
	deviceID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid device ID",
		})
	}

	var req SetupNFSServerRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if err := ValidateRequest(c, &req); err != nil {
		return err
	}

	// Set defaults
	if req.ClientCIDR == "" {
		req.ClientCIDR = "*"
	}
	if req.Options == "" {
		req.Options = "rw,sync,no_subtree_check,no_root_squash"
	}

	export, err := h.service.SetupServer(deviceID, req.ExportPath, req.ClientCIDR, req.Options)
	if err != nil {
		return HandleError(c, 500, err, "Failed to setup NFS server")
	}

	return c.Status(201).JSON(export)
}

// CreateExport handles POST /api/v1/devices/:id/nfs/exports
func (h *NFSHandler) CreateExport(c *fiber.Ctx) error {
	deviceID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid device ID",
		})
	}

	var req CreateExportRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if err := ValidateRequest(c, &req); err != nil {
		return err
	}

	// Set defaults
	if req.ClientCIDR == "" {
		req.ClientCIDR = "*"
	}
	if req.Options == "" {
		req.Options = "rw,sync,no_subtree_check,no_root_squash"
	}

	export, err := h.service.CreateExport(deviceID, req.ExportPath, req.ClientCIDR, req.Options)
	if err != nil {
		return HandleError(c, 500, err, "Failed to create export")
	}

	return c.Status(201).JSON(export)
}

// ListExports handles GET /api/v1/devices/:id/nfs/exports
func (h *NFSHandler) ListExports(c *fiber.Ctx) error {
	deviceID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid device ID",
		})
	}

	exports, err := h.service.ListExports(deviceID)
	if err != nil {
		return HandleError(c, 500, err, "Failed to list exports")
	}

	return c.JSON(exports)
}

// RemoveExport handles DELETE /api/v1/devices/:id/nfs/exports/:exportId
func (h *NFSHandler) RemoveExport(c *fiber.Ctx) error {
	deviceID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid device ID",
		})
	}

	exportID, err := uuid.Parse(c.Params("exportId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid export ID",
		})
	}

	if err := h.service.RemoveExport(deviceID, exportID); err != nil {
		return HandleError(c, 500, err, "Failed to remove export")
	}

	return c.Status(204).Send(nil)
}

// MountShare handles POST /api/v1/devices/:id/nfs/mounts
func (h *NFSHandler) MountShare(c *fiber.Ctx) error {
	deviceID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid device ID",
		})
	}

	var req MountNFSShareRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if err := ValidateRequest(c, &req); err != nil {
		return err
	}

	mount, err := h.service.MountShare(deviceID, req.ServerIP, req.RemotePath, req.LocalPath, req.Options, req.Permanent)
	if err != nil {
		return HandleError(c, 500, err, "Failed to mount NFS share")
	}

	return c.Status(201).JSON(mount)
}

// ListMounts handles GET /api/v1/devices/:id/nfs/mounts
func (h *NFSHandler) ListMounts(c *fiber.Ctx) error {
	deviceID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid device ID",
		})
	}

	mounts, err := h.service.ListMounts(deviceID)
	if err != nil {
		return HandleError(c, 500, err, "Failed to list mounts")
	}

	return c.JSON(mounts)
}

// UnmountShare handles DELETE /api/v1/devices/:id/nfs/mounts/:mountId
func (h *NFSHandler) UnmountShare(c *fiber.Ctx) error {
	deviceID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid device ID",
		})
	}

	mountID, err := uuid.Parse(c.Params("mountId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid mount ID",
		})
	}

	// Check query parameter for fstab removal
	removeFromFstab := c.QueryBool("remove_from_fstab", true)

	if err := h.service.UnmountShare(deviceID, mountID, removeFromFstab); err != nil {
		return HandleError(c, 500, err, "Failed to unmount share")
	}

	return c.Status(204).Send(nil)
}

// RegisterRoutes registers all NFS routes
func (h *NFSHandler) RegisterRoutes(api fiber.Router) {
	// NFS routes are nested under devices
	// They will be registered as /api/v1/devices/:id/nfs/*
}
