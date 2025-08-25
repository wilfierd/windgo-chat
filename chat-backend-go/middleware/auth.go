package middleware

import (
	"chat-backend-go/utils"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware validates JWT token and extracts user ID and device ID
func AuthMiddleware(c *fiber.Ctx) error {
	// Get Authorization header
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		// Try to get token from cookie
		token := c.Cookies("access_token")
		if token != "" {
			authHeader = "Bearer " + token
		}
	}

	if authHeader == "" {
		return c.Status(401).JSON(fiber.Map{
			"error": "Authorization token required",
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

	// Validate token and extract user ID and device ID
	userID, deviceID, err := utils.ValidateJWT(tokenString)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid or expired token",
		})
	}

	// Store user ID and device ID in context for use in handlers
	c.Locals("userID", userID)
	c.Locals("deviceID", deviceID)

	return c.Next()
}

// Optional auth middleware for routes that can work with or without auth
func OptionalAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if userID, deviceID, err := utils.ValidateJWT(tokenString); err == nil {
				c.Locals("userID", userID)
				c.Locals("deviceID", deviceID)
			}
		}
		return c.Next()
	}
}
