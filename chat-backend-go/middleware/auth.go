package middleware

import (
	"chat-backend-go/utils"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Context key type for type-safe context values
type contextKey string

const (
	UserIDKey contextKey = "userID"
)

// AuthRequired middleware validates JWT token and extracts user ID
func AuthRequired() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(401).JSON(fiber.Map{
				"error": "Authorization header required",
			})
		}

		// Check if it's a Bearer token
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(401).JSON(fiber.Map{
				"error": "Invalid authorization header format",
			})
		}

		// Extract token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			return c.Status(401).JSON(fiber.Map{
				"error": "Token required",
			})
		}

		// Validate token and extract user ID
		userID, err := utils.ValidateJWT(tokenString)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		// Store user ID in context for use in handlers (as uint for type safety)
		c.Locals(string(UserIDKey), userID)

		return c.Next()
	}
}

// Optional auth middleware for routes that can work with or without auth
func OptionalAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if userID, err := utils.ValidateJWT(tokenString); err == nil {
				c.Locals(string(UserIDKey), userID)
			}
		}
		return c.Next()
	}
}

// WebSocketAuth middleware validates JWT token from query parameter for WebSocket connections
func WebSocketAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Try to get token from query parameter first (for WebSocket)
		tokenString := c.Query("token")

		// If not in query, try Authorization header as fallback
		if tokenString == "" {
			authHeader := c.Get("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		// If still no token, return error
		if tokenString == "" {
			return c.Status(401).JSON(fiber.Map{
				"error": "Token required in query parameter or Authorization header",
			})
		}

		// Validate token and extract user ID
		userID, err := utils.ValidateJWT(tokenString)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		// Store user ID in context for use in handlers (as uint for type safety)
		c.Locals(string(UserIDKey), userID)

		return c.Next()
	}
}
