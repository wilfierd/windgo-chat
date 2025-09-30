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
    // OAuth with GitHub (web)
    auth.Get("/github/login", handlers.GitHubLogin)
    auth.Get("/github/callback", handlers.GitHubCallback)
    auth.Get("/github/status", handlers.GitHubConfigStatus)

    // OAuth with GitHub (Device Flow for CLI)
    auth.Post("/github/device/start", handlers.GitHubDeviceStart)
    auth.Post("/github/device/poll", handlers.GitHubDevicePoll)

    // Protected routes (authentication required with activity tracking)
    auth.Get("/profile", middleware.AuthRequired(), middleware.TrackActivity(), handlers.GetProfile)
	auth.Post("/refresh", middleware.AuthRequired(), middleware.TrackActivity(), func(c *fiber.Ctx) error {
		// Get user ID from middleware
		userID := c.Locals("userID").(uint)

		// Generate new token
		token, err := utils.GenerateJWT(userID)
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
