package models

import "fmt"

// Error codes for structured error handling
const (
	ErrCodeSudoNotConfigured   = "SUDO_NOT_CONFIGURED"
	ErrCodeSSHConnectionFailed = "SSH_CONNECTION_FAILED"
	ErrCodeAuthFailed          = "AUTH_FAILED"
	ErrCodeNotFound            = "NOT_FOUND"
	ErrCodeAlreadyExists       = "ALREADY_EXISTS"
	ErrCodeValidationFailed    = "VALIDATION_FAILED"
	ErrCodeInternalError       = "INTERNAL_ERROR"
)

// APIError represents a structured error with code and optional details
type APIError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
	Err     error                  `json:"-"` // Original error (not exposed to client)
}

// Error implements the error interface
func (e *APIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *APIError) Unwrap() error {
	return e.Err
}

// NewAPIError creates a new APIError
func NewAPIError(code, message string, details map[string]interface{}) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// WrapError wraps an existing error with an APIError
func WrapError(code, message string, err error, details map[string]interface{}) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Details: details,
		Err:     err,
	}
}

// NewSudoError creates a sudo configuration error
func NewSudoError(deviceIP string) *APIError {
	return NewAPIError(
		ErrCodeSudoNotConfigured,
		"Passwordless sudo is required for automated software installation",
		map[string]interface{}{
			"device_ip": deviceIP,
			"fix_steps": []string{
				"SSH into your device",
				"Run: sudo visudo",
				"Add this line at the end (replace YOUR_USERNAME):",
				"  YOUR_USERNAME ALL=(ALL) NOPASSWD: ALL",
				"Save and test with: sudo apt-get update",
			},
		},
	)
}

// NewSSHError creates an SSH connection error
func NewSSHError(deviceIP string, err error) *APIError {
	return WrapError(
		ErrCodeSSHConnectionFailed,
		"Unable to establish SSH connection to device",
		err,
		map[string]interface{}{
			"device_ip": deviceIP,
		},
	)
}

// NewNotFoundError creates a not found error
func NewNotFoundError(resource string) *APIError {
	return NewAPIError(
		ErrCodeNotFound,
		fmt.Sprintf("%s not found", resource),
		map[string]interface{}{
			"resource": resource,
		},
	)
}

// NewValidationError creates a validation error
func NewValidationError(message string, fields []string) *APIError {
	return NewAPIError(
		ErrCodeValidationFailed,
		message,
		map[string]interface{}{
			"invalid_fields": fields,
		},
	)
}
