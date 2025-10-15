package api

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/services"
)

// ResourceHandler handles resource-related HTTP requests
type ResourceHandler struct {
	monitoringService *services.ResourceMonitoringService
	dbPoolManager     *services.DatabasePoolManager
}

// NewResourceHandler creates a new resource handler
func NewResourceHandler(monitoringService *services.ResourceMonitoringService, dbPoolManager *services.DatabasePoolManager) *ResourceHandler {
	return &ResourceHandler{
		monitoringService: monitoringService,
		dbPoolManager:     dbPoolManager,
	}
}

// GetAggregateResources handles GET /api/v1/resources/aggregate
// Now includes database pooling savings
func (h *ResourceHandler) GetAggregateResources(c *fiber.Ctx) error {
	resources, err := h.monitoringService.GetAggregateResources()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to get aggregate resources",
		})
	}

	// Add database pooling statistics
	if h.dbPoolManager != nil {
		dbStats, err := h.dbPoolManager.GetSharedInstanceStats()
		if err == nil {
			// Merge database pooling stats into response
			response := fiber.Map{
				// Resource aggregation
				"total_devices":          resources.TotalDevices,
				"online_devices":         resources.OnlineDevices,
				"offline_devices":        resources.OfflineDevices,
				"total_cpu_cores":        resources.TotalCPUCores,
				"used_cpu_cores":         resources.UsedCPUCores,
				"avg_cpu_usage_percent":  resources.AvgCPUUsagePercent,
				"total_ram_mb":           resources.TotalRAMMB,
				"used_ram_mb":            resources.UsedRAMMB,
				"available_ram_mb":       resources.AvailableRAMMB,
				"ram_usage_percent":      resources.RAMUsagePercent,
				"total_storage_gb":       resources.TotalStorageGB,
				"used_storage_gb":        resources.UsedStorageGB,
				"available_storage_gb":   resources.AvailableStorageGB,
				"storage_usage_percent":  resources.StorageUsagePercent,

				// Database pooling savings
				"database_pooling": fiber.Map{
					"shared_instances":          dbStats["shared_instances"],
					"total_databases":           dbStats["total_databases"],
					"estimated_ram_saved_mb":    dbStats["estimated_ram_saved_mb"],
					"estimated_ram_saved_percent": dbStats["estimated_ram_saved_percent"],
				},
			}
			return c.JSON(response)
		}
	}

	// Fallback: return resources without database stats
	return c.JSON(resources)
}

// GetDeviceResources handles GET /api/v1/devices/:id/resources
func (h *ResourceHandler) GetDeviceResources(c *fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid device ID",
		})
	}

	metrics, err := h.monitoringService.GetDeviceMetrics(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Device metrics not found",
		})
	}

	// Add calculated percentages
	response := fiber.Map{
		"device_id":              metrics.DeviceID,
		"cpu_usage_percent":      metrics.CPUUsagePercent,
		"cpu_cores":              metrics.CPUCores,
		"total_ram_mb":           metrics.TotalRAMMB,
		"used_ram_mb":            metrics.UsedRAMMB,
		"available_ram_mb":       metrics.AvailableRAMMB,
		"ram_usage_percent":      metrics.RAMUsagePercent(),
		"total_storage_gb":       metrics.TotalStorageGB,
		"used_storage_gb":        metrics.UsedStorageGB,
		"available_storage_gb":   metrics.AvailableStorageGB,
		"storage_usage_percent":  metrics.StorageUsagePercent(),
		"recorded_at":            metrics.RecordedAt,
	}

	return c.JSON(response)
}

// GetDeviceResourcesHistory handles GET /api/v1/devices/:id/resources/history
func (h *ResourceHandler) GetDeviceResourcesHistory(c *fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid device ID",
		})
	}

	// Parse query parameter for time range (default: last 24 hours)
	hoursStr := c.Query("hours", "24")
	var hours int
	if _, err := fmt.Sscanf(hoursStr, "%d", &hours); err != nil {
		hours = 24
	}

	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	metrics, err := h.monitoringService.GetDeviceMetricsHistory(id, since)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to get device metrics history",
		})
	}

	// Transform metrics to include calculated percentages
	response := make([]fiber.Map, len(metrics))
	for i, m := range metrics {
		response[i] = fiber.Map{
			"cpu_usage_percent":      m.CPUUsagePercent,
			"cpu_cores":              m.CPUCores,
			"total_ram_mb":           m.TotalRAMMB,
			"used_ram_mb":            m.UsedRAMMB,
			"available_ram_mb":       m.AvailableRAMMB,
			"ram_usage_percent":      m.RAMUsagePercent(),
			"total_storage_gb":       m.TotalStorageGB,
			"used_storage_gb":        m.UsedStorageGB,
			"available_storage_gb":   m.AvailableStorageGB,
			"storage_usage_percent":  m.StorageUsagePercent(),
			"recorded_at":            m.RecordedAt,
		}
	}

	return c.JSON(response)
}

// GetMonitoringStatus handles GET /api/v1/resources/status
// Returns detailed status including health check and metrics
func (h *ResourceHandler) GetMonitoringStatus(c *fiber.Ctx) error {
	status := h.monitoringService.GetStatus()
	return c.JSON(status)
}

// RegisterRoutes registers all resource routes
func (h *ResourceHandler) RegisterRoutes(api fiber.Router) {
	resources := api.Group("/resources")

	resources.Get("/aggregate", h.GetAggregateResources)
	resources.Get("/status", h.GetMonitoringStatus)
}

// RegisterDeviceResourceRoutes registers device-specific resource routes
// This should be called after device routes are set up
func (h *ResourceHandler) RegisterDeviceResourceRoutes(devices fiber.Router) {
	devices.Get("/:id/resources", h.GetDeviceResources)
	devices.Get("/:id/resources/history", h.GetDeviceResourcesHistory)
}
