package middleware

import (
	"chat-backend-go/config"
	"chat-backend-go/models"
	"chat-backend-go/utils"
	"strconv"
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

// RequireRoomMembership middleware verifies that the authenticated user is a participant of the room
// This middleware should be applied after AuthRequired() to ensure userID is available in context
func RequireRoomMembership() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get authenticated user ID from context
		userID, ok := c.Locals(string(UserIDKey)).(uint)
		if !ok {
			return c.Status(401).JSON(fiber.Map{
				"error": "Authentication required",
			})
		}

		// Get room ID from URL parameter
		roomIDParam := c.Params("id")
		if roomIDParam == "" {
			// Try alternative parameter name used in some routes
			roomIDParam = c.Params("roomId")
		}

		if roomIDParam == "" {
			return c.Status(400).JSON(fiber.Map{
				"error": "Room ID is required",
			})
		}

		// Convert room ID to uint
		roomID, err := strconv.ParseUint(roomIDParam, 10, 32)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "Invalid room ID format",
			})
		}

		// Check if user is a member of the room
		var membership models.RoomMembership
		result := config.DB.Where("user_id = ? AND room_id = ?", userID, uint(roomID)).First(&membership)

		if result.Error != nil {
			// User is not a member of this room
			return c.Status(403).JSON(fiber.Map{
				"error": "You are not a participant of this conversation",
			})
		}

		// User is a member, allow access
		return c.Next()
	}
}
