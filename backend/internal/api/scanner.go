package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/services"
)

// ScannerHandler handles network scanning HTTP requests
type ScannerHandler struct {
	service *services.ScannerService
}

// NewScannerHandler creates a new scanner handler
func NewScannerHandler(service *services.ScannerService) *ScannerHandler {
	return &ScannerHandler{service: service}
}

// StartScanRequest represents the request body for starting a network scan
type StartScanRequest struct {
	CIDR string `json:"cidr,omitempty"` // Optional: auto-detect if not provided
}

// StartScan handles POST /api/v1/devices/scan
func (h *ScannerHandler) StartScan(c *fiber.Ctx) error {
	var req StartScanRequest
	if err := c.BodyParser(&req); err != nil {
		// If no body provided, that's okay - we'll auto-detect
		req.CIDR = ""
	}

	// Auto-detect network if CIDR not provided
	cidr := req.CIDR
	if cidr == "" {
		detectedCIDR, err := h.service.DetectLocalNetwork()
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "Could not detect local network. Please provide a CIDR range.",
			})
		}
		cidr = detectedCIDR
	}

	// Start the scan
	scanID, err := h.service.StartScan(c.Context(), cidr)
	if err != nil {
		return HandleError(c, 400, err, "Failed to start network scan")
	}

	return c.Status(201).JSON(fiber.Map{
		"scan_id": scanID,
		"cidr":    cidr,
		"message": "Network scan started",
	})
}

// GetScanProgress handles GET /api/v1/devices/scan/:id
func (h *ScannerHandler) GetScanProgress(c *fiber.Ctx) error {
	scanID := c.Params("id")

	progress, err := h.service.GetScanProgress(scanID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Scan not found",
		})
	}

	return c.JSON(progress)
}

// DetectNetwork handles GET /api/v1/devices/scan/detect-network
func (h *ScannerHandler) DetectNetwork(c *fiber.Ctx) error {
	cidr, err := h.service.DetectLocalNetwork()
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Could not detect local network",
		})
	}

	return c.JSON(fiber.Map{
		"cidr": cidr,
	})
}

// RegisterRoutes registers all scanner routes
func (h *ScannerHandler) RegisterRoutes(api fiber.Router) {
	devices := api.Group("/devices")

	// Scan routes - must be before /:id to avoid conflicts
	devices.Get("/scan/detect-network", h.DetectNetwork)
	devices.Post("/scan", h.StartScan)
	devices.Get("/scan/:id", h.GetScanProgress)
}
