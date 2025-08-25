package routes

import (
	"chat-backend-go/handlers"
	"chat-backend-go/middleware"

	"github.com/gofiber/fiber/v2"
)

func MessageRoutes(app *fiber.App) {
	api := app.Group("/api/v1")

	// Public routes
	api.Get("/rooms", handlers.GetRooms)

	// Protected routes (require authentication)
	protected := api.Use(middleware.AuthMiddleware)
	protected.Post("/messages", handlers.SendMessage)
	protected.Get("/rooms/:roomId/messages", handlers.GetMessages)
}
