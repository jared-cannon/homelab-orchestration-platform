package api

import (
	"log"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
)

// Global validator instance
var validate = validator.New()

// ErrorResponse represents a sanitized error response for API clients
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Code    string                 `json:"code,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// sanitizeError returns a user-friendly error message and logs the detailed error
func sanitizeError(err error, userMessage string) string {
	if err == nil {
		return userMessage
	}

	// Log the detailed error server-side for debugging
	log.Printf("[API Error] %s: %v", userMessage, err)

	// Check for common error patterns and return sanitized messages
	errStr := err.Error()

	// Database errors
	if strings.Contains(errStr, "UNIQUE constraint") {
		return "A resource with this value already exists"
	}
	if strings.Contains(errStr, "record not found") || strings.Contains(errStr, "not found") {
		return "Resource not found"
	}
	if strings.Contains(errStr, "failed to create") {
		return "Failed to create resource"
	}
	if strings.Contains(errStr, "failed to update") {
		return "Failed to update resource"
	}
	if strings.Contains(errStr, "failed to delete") {
		return "Failed to delete resource"
	}

	// Keychain/credential errors
	if strings.Contains(errStr, "keyring") || strings.Contains(errStr, "keychain") {
		return "Failed to manage credentials securely"
	}

	// SSH connection errors
	if strings.Contains(errStr, "connection failed") || strings.Contains(errStr, "failed to dial") {
		return "Unable to connect to device"
	}
	if strings.Contains(errStr, "authentication failed") || strings.Contains(errStr, "unable to authenticate") {
		return "Authentication failed - check credentials"
	}
	if strings.Contains(errStr, "no active connection") {
		return "Device connection unavailable"
	}

	// Network scanning errors
	if strings.Contains(errStr, "invalid CIDR") {
		return "Invalid network range provided"
	}
	if strings.Contains(errStr, "scan not found") {
		return "Scan not found"
	}

	// IP/validation errors
	if strings.Contains(errStr, "invalid IP") {
		return "Invalid IP address"
	}
	if strings.Contains(errStr, "device with IP") && strings.Contains(errStr, "already exists") {
		return "A device with this IP address already exists"
	}

	// Default to the provided user message
	return userMessage
}

// HandleError is a helper to return sanitized error responses
func HandleError(c *fiber.Ctx, statusCode int, err error, defaultMessage string) error {
	// Check if this is a structured APIError
	if apiErr, ok := err.(*models.APIError); ok {
		return c.Status(statusCode).JSON(ErrorResponse{
			Error:   apiErr.Message,
			Code:    apiErr.Code,
			Details: apiErr.Details,
		})
	}

	// Otherwise sanitize the error
	sanitized := sanitizeError(err, defaultMessage)
	return c.Status(statusCode).JSON(ErrorResponse{
		Error: sanitized,
	})
}

// HandleErrorWithDetails returns sanitized error with additional safe details
func HandleErrorWithDetails(c *fiber.Ctx, statusCode int, err error, defaultMessage string, details interface{}) error {
	sanitized := sanitizeError(err, defaultMessage)
	return c.Status(statusCode).JSON(fiber.Map{
		"error":   sanitized,
		"details": details,
	})
}

// ValidateRequest validates a request struct and returns a sanitized error if validation fails
func ValidateRequest(c *fiber.Ctx, req interface{}) error {
	if err := validate.Struct(req); err != nil {
		// Log detailed validation error
		log.Printf("[Validation Error] %v", err)

		// Return user-friendly error
		return c.Status(400).JSON(ErrorResponse{
			Error: "Invalid request - please check your input and try again",
		})
	}
	return nil
}
