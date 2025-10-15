package middleware

import (
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/services"
)

// Claims represents JWT claims
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// getJWTSecret returns the JWT secret from environment or a default (for dev only)
func getJWTSecret() []byte {
	secret := os.Getenv("APP_KEY")
	if secret == "" {
		// For development only - in production this should be required
		secret = "homelab-default-jwt-secret-CHANGE-IN-PRODUCTION"
	}
	return []byte(secret)
}

// AuthMiddleware validates JWT tokens from Authorization header
func AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authorization header",
			})
		}

		// Extract token from "Bearer <token>" format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authorization header format",
			})
		}

		tokenString := parts[1]

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return getJWTSecret(), nil
		})

		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		if !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token",
			})
		}

		// Extract claims and store in context
		if claims, ok := token.Claims.(*Claims); ok {
			c.Locals("username", claims.Username)
		}

		return c.Next()
	}
}

// GenerateToken generates a new JWT token for a username
func GenerateToken(username string) (string, error) {
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

// OptionalAuthMiddleware validates JWT tokens but doesn't require them
// Useful for endpoints that work differently when authenticated
func OptionalAuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Next()
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.Next()
		}

		tokenString := parts[1]
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return getJWTSecret(), nil
		})

		if err == nil && token.Valid {
			if claims, ok := token.Claims.(*Claims); ok {
				c.Locals("username", claims.Username)
			}
		}

		return c.Next()
	}
}

// AdminMiddleware validates JWT tokens and checks if user is admin
// Must be used after AuthMiddleware
func AdminMiddleware(userService *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get username from context (set by AuthMiddleware)
		username, ok := c.Locals("username").(string)
		if !ok || username == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized - authentication required",
			})
		}

		// Get user from database to check admin status
		user, err := userService.GetUserByUsername(username)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not found",
			})
		}

		// Check if user is admin
		if !user.IsAdmin {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Forbidden - admin access required",
			})
		}

		// Store user in context for later use
		c.Locals("user", user)
		c.Locals("is_admin", true)

		return c.Next()
	}
}
