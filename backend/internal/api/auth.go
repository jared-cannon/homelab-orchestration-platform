package api

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/middleware"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/services"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	userService *services.UserService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(userService *services.UserService) *AuthHandler {
	return &AuthHandler{userService: userService}
}

// LoginRequest represents the request body for login
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// RegisterRequest represents the request body for user registration
type RegisterRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
	Email    string `json:"email,omitempty"`
}

// ChangePasswordRequest represents the request body for changing password
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required"`
}

// AuthResponse represents the response after successful authentication
type AuthResponse struct {
	Token    string      `json:"token"`
	Username string      `json:"username"`
	IsAdmin  bool        `json:"is_admin"`
	User     interface{} `json:"user"`
}

// Login handles POST /api/v1/auth/login
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if err := ValidateRequest(c, &req); err != nil {
		return err
	}

	// Authenticate user
	user, err := h.userService.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			return c.Status(401).JSON(fiber.Map{
				"error": "Invalid username or password",
			})
		}
		return HandleError(c, 500, err, "Authentication failed")
	}

	// Generate JWT token
	token, err := middleware.GenerateToken(user.Username)
	if err != nil {
		return HandleError(c, 500, err, "Failed to generate authentication token")
	}

	return c.JSON(AuthResponse{
		Token:    token,
		Username: user.Username,
		IsAdmin:  user.IsAdmin,
		User:     user,
	})
}

// Register handles POST /api/v1/auth/register
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if err := ValidateRequest(c, &req); err != nil {
		return err
	}

	// Check if this is the first user (make them admin)
	count, err := h.userService.CountUsers()
	if err != nil {
		return HandleError(c, 500, err, "Failed to check user count")
	}
	isFirstUser := count == 0

	// Create user (first user becomes admin)
	user, err := h.userService.CreateUser(req.Username, req.Password, isFirstUser)
	if err != nil {
		if errors.Is(err, services.ErrUserAlreadyExists) {
			return c.Status(409).JSON(fiber.Map{
				"error": "Username already exists",
			})
		}
		if errors.Is(err, services.ErrInvalidUsername) {
			return c.Status(400).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		if errors.Is(err, services.ErrInvalidPassword) {
			return c.Status(400).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return HandleError(c, 500, err, "Failed to create user")
	}

	// Set email if provided
	if req.Email != "" {
		user.Email = req.Email
	}

	// Generate JWT token
	token, err := middleware.GenerateToken(user.Username)
	if err != nil {
		return HandleError(c, 500, err, "Failed to generate authentication token")
	}

	return c.Status(201).JSON(AuthResponse{
		Token:    token,
		Username: user.Username,
		IsAdmin:  user.IsAdmin,
		User:     user,
	})
}

// GetCurrentUser handles GET /api/v1/auth/me
func (h *AuthHandler) GetCurrentUser(c *fiber.Ctx) error {
	// Get username from JWT token (set by auth middleware)
	username, ok := c.Locals("username").(string)
	if !ok || username == "" {
		return c.Status(401).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Get user from database
	user, err := h.userService.GetUserByUsername(username)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return c.Status(404).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		return HandleError(c, 500, err, "Failed to get user")
	}

	return c.JSON(user)
}

// ChangePassword handles POST /api/v1/auth/change-password
func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	// Get username from JWT token
	username, ok := c.Locals("username").(string)
	if !ok || username == "" {
		return c.Status(401).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var req ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if err := ValidateRequest(c, &req); err != nil {
		return err
	}

	// Get user
	user, err := h.userService.GetUserByUsername(username)
	if err != nil {
		return HandleError(c, 500, err, "Failed to get user")
	}

	// Change password
	err = h.userService.UpdatePassword(user.ID, req.OldPassword, req.NewPassword)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			return c.Status(401).JSON(fiber.Map{
				"error": "Current password is incorrect",
			})
		}
		if errors.Is(err, services.ErrInvalidPassword) {
			return c.Status(400).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return HandleError(c, 500, err, "Failed to change password")
	}

	return c.JSON(fiber.Map{
		"message": "Password changed successfully",
	})
}

// RegisterRoutes registers all auth routes
func (h *AuthHandler) RegisterRoutes(api fiber.Router) {
	auth := api.Group("/auth")

	// Public routes (no authentication required)
	auth.Post("/login", h.Login)
	auth.Post("/register", h.Register)
}

// RegisterProtectedRoutes registers authenticated auth routes
func (h *AuthHandler) RegisterProtectedRoutes(api fiber.Router) {
	auth := api.Group("/auth")

	// Protected routes (authentication required)
	auth.Get("/me", h.GetCurrentUser)
	auth.Post("/change-password", h.ChangePassword)
}
