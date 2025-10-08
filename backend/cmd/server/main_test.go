package main

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// setupTestApp creates a Fiber app for testing
func setupTestApp() *fiber.App {
	app := fiber.New(fiber.Config{
		AppName: "Homelab Orchestration Platform",
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	api := app.Group("/api/v1")

	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "homelab-orchestration-platform",
			"version": "0.1.0",
		})
	})

	return app
}

func TestHealthCheckEndpoint(t *testing.T) {
	app := setupTestApp()

	// Create a test HTTP request
	req := httptest.NewRequest("GET", "/api/v1/health", nil)

	// Perform the request
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}

	// Check status code
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check Content-Type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Parse response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Validate response fields
	if result["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%v'", result["status"])
	}

	if result["service"] != "homelab-orchestration-platform" {
		t.Errorf("Expected service 'homelab-orchestration-platform', got '%v'", result["service"])
	}

	if result["version"] != "0.1.0" {
		t.Errorf("Expected version '0.1.0', got '%v'", result["version"])
	}
}

func TestHealthCheckResponseStructure(t *testing.T) {
	app := setupTestApp()

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	// Ensure all required fields are present
	requiredFields := []string{"status", "service", "version"}
	for _, field := range requiredFields {
		if _, exists := result[field]; !exists {
			t.Errorf("Missing required field: %s", field)
		}
	}
}
