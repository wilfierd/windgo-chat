// Package routes defines API route configurations for the chat application.
// This file contains WebSocket routes for real-time communication.
package routes

import (
	"chat-backend-go/handlers"
	"chat-backend-go/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// SetupWebSocketRoutes configures WebSocket routes
func SetupWebSocketRoutes(app *fiber.App) {
	// WebSocket endpoint with authentication (accepts token from query parameter)
	app.Use("/ws", middleware.WebSocketAuth(), handlers.WebSocketUpgrade)
	app.Get("/ws", websocket.New(handlers.HandleWebSocket))

	// WebSocket stats endpoints (REST API for monitoring)
	ws := app.Group("/api/v1/ws", middleware.AuthRequired())
	ws.Get("/online", handlers.GetOnlineUsers)
	ws.Get("/rooms/:roomId/stats", handlers.GetRoomStats)
}
