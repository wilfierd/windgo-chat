package routes

import (
	"chat-backend-go/handlers"
	"chat-backend-go/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupGitHubAuthRoutes sets up GitHub OAuth routes
func SetupGitHubAuthRoutes(app *fiber.App) {
	// Auth group
	auth := app.Group("/auth")

	// GitHub OAuth routes
	auth.Post("/nonce", handlers.GenerateNonce)
	auth.Post("/login/ssh", handlers.SSHLogin)
	auth.Post("/refresh", handlers.RefreshAuth)
	auth.Post("/logout", handlers.Logout)

	// Protected routes
	protected := app.Group("/", middleware.AuthMiddleware)
	protected.Get("/me", handlers.GetMe)

	// Device management
	devices := app.Group("/devices", middleware.AuthMiddleware)
	devices.Post("/revoke/:deviceId", handlers.RevokeDevice)
}
