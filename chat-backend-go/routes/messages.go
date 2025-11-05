package routes

import (
	"chat-backend-go/handlers"
	"chat-backend-go/middleware"

	"github.com/gofiber/fiber/v2"
)

func MessageRoutes(app *fiber.App) {
	api := app.Group("/api/v1")

	// Public routes
	api.Get("/rooms/:id", handlers.GetRoomByID)

	// Protected routes (require authentication and track activity)
	protected := api.Use(middleware.AuthRequired(), middleware.TrackActivity())

	// Room list (authenticated - shows user's rooms)
	protected.Get("/rooms", handlers.GetRooms)

	// Message routes with room membership verification
	protected.Get("/rooms/:roomId/messages", middleware.RequireRoomMembership(), handlers.GetMessages)
	protected.Post("/rooms/:id/messages", middleware.RequireRoomMembership(), handlers.SendMessage)

	// General message routes (without room-specific membership check)
	protected.Post("/messages", handlers.SendMessage)
	protected.Put("/messages/:id", handlers.UpdateMessage)
	protected.Delete("/messages/:id", handlers.DeleteMessage)

	// Room management routes (admin only - checked within handlers)
	protected.Post("/rooms", handlers.CreateRoom)
	protected.Put("/rooms/:id", handlers.UpdateRoom)
	protected.Delete("/rooms/:id", handlers.DeleteRoom)

	// Direct message routes
	protected.Post("/rooms/direct", handlers.CreateDirectRoom)
	protected.Get("/rooms/direct", handlers.GetDirectRooms)

	// Room participants route
	protected.Get("/rooms/:id/participants", handlers.GetRoomParticipants)

	// Room membership management routes (requires admin role or higher)
	protected.Post("/rooms/:id/members", handlers.InviteUserToRoom)
	protected.Delete("/rooms/:id/members/:userId", handlers.RemoveUserFromRoom)
}
