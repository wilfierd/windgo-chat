// Package routes defines HTTP route configurations for the chat application.
// This file sets up authentication routes including registration, login, and protected endpoints.
package routes

import (
	"chat-backend-go/handlers"
	"chat-backend-go/middleware"
	"chat-backend-go/utils"

	"github.com/gofiber/fiber/v2"
)

// SetupAuthRoutes sets up authentication related routes
func SetupAuthRoutes(app *fiber.App) {
	// Create auth group
	auth := app.Group("/api/auth")

	// Public routes (no authentication required)
	auth.Post("/register", handlers.Register)
	auth.Post("/login", handlers.Login)

	// Protected routes (authentication required)
	auth.Get("/profile", middleware.AuthMiddleware, handlers.GetProfile)
	auth.Post("/refresh", middleware.AuthMiddleware, func(c *fiber.Ctx) error {
		// Get user ID and device ID from middleware
		userID := c.Locals("userID").(uint)
		deviceID := c.Locals("deviceID").(string)

		// Generate new token
		token, err := utils.GenerateJWT(userID, deviceID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "Failed to refresh token",
			})
		}

		return c.JSON(fiber.Map{
			"token": token,
		})
	})
}
