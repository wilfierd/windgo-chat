package routes

import (
	"chat-backend-go/handlers"
	"chat-backend-go/middleware"

	"github.com/gofiber/fiber/v2"
)

// UserRoutes exposes user directory endpoints required by the CLI
func UserRoutes(app *fiber.App) {
	api := app.Group("/api/v1")
	// Apply activity tracking to all authenticated routes
	users := api.Group("/users", middleware.AuthRequired(), middleware.TrackActivity())
	users.Get("/", handlers.ListUsers)
}
