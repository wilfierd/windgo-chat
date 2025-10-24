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
	api.Get("/rooms/:id", handlers.GetRoomByID)

	// Protected routes (require authentication and track activity)
	protected := api.Use(middleware.AuthRequired(), middleware.TrackActivity())

	// Message routes
	protected.Post("/messages", handlers.SendMessage)
	protected.Get("/rooms/:roomId/messages", handlers.GetMessages)
	protected.Put("/messages/:id", handlers.UpdateMessage)
	protected.Delete("/messages/:id", handlers.DeleteMessage)

	// Room management routes (admin only - checked within handlers)
	protected.Post("/rooms", handlers.CreateRoom)
	protected.Put("/rooms/:id", handlers.UpdateRoom)
	protected.Delete("/rooms/:id", handlers.DeleteRoom)
}
