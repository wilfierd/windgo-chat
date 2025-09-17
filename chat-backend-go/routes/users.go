package routes

import (
	"chat-backend-go/handlers"
	"chat-backend-go/middleware"

	"github.com/gofiber/fiber/v2"
)

// UserRoutes exposes user directory endpoints required by the CLI
func UserRoutes(app *fiber.App) {
	api := app.Group("/api/v1")
	users := api.Group("/users", middleware.AuthRequired())
	users.Get("/", handlers.ListUsers)
}
