// Package handlers contains HTTP request handlers for the chat application.
// This file contains WebSocket handlers for real-time communication.
package handlers

import (
	ws "chat-backend-go/websocket"
	"fmt"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

var (
	// Hub is the global WebSocket hub instance
	Hub *ws.Hub
)

// InitWebSocketHub initializes the WebSocket hub
func InitWebSocketHub() {
	Hub = ws.NewHub()
	go Hub.Run()
	log.Println("WebSocket hub initialized and running")
}

// WebSocketUpgrade upgrades the HTTP connection to WebSocket
func WebSocketUpgrade(c *fiber.Ctx) error {
	// Check if the request is a WebSocket upgrade request
	if websocket.IsWebSocketUpgrade(c) {
		// Get user ID from JWT middleware
		userID := c.Locals("userID")
		if userID == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not authenticated",
			})
		}

		// Store userID in context for the WebSocket handler
		c.Locals("wsUserID", userID.(uint))
		return c.Next()
	}

	return c.Status(fiber.StatusUpgradeRequired).JSON(fiber.Map{
		"error": "WebSocket upgrade required",
	})
}

// HandleWebSocket handles WebSocket connections
func HandleWebSocket(c *websocket.Conn) {
	// Get user ID from fiber context
	userID, ok := c.Locals("wsUserID").(uint)
	if !ok {
		log.Println("Failed to get user ID from WebSocket context")
		c.Close()
		return
	}

	// Create a new client
	client := &ws.Client{
		ID:     fmt.Sprintf("user_%d_%p", userID, c),
		UserID: userID,
		Conn:   c,
		Send:   make(chan []byte, 256),
		Hub:    Hub,
		Rooms:  make(map[uint]bool),
	}

	// Register client with hub
	Hub.Register(client)

	// Log connection
	log.Printf("WebSocket client connected: User ID %d", userID)

	// Start reading and writing goroutines
	go client.WritePump()
	client.ReadPump() // This blocks until connection closes
}

// BroadcastMessage broadcasts a message to all clients in a room
func BroadcastMessage(roomID uint, messageType string, content interface{}, userID uint) {
	if Hub == nil {
		log.Println("WebSocket hub not initialized")
		return
	}

	msg := &ws.Message{
		Type:    messageType,
		RoomID:  roomID,
		UserID:  userID,
		Content: content,
	}

	Hub.BroadcastToRoom(msg)
}

// GetOnlineUsers returns a list of online user IDs
func GetOnlineUsers(c *fiber.Ctx) error {
	if Hub == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "WebSocket service not available",
		})
	}

	onlineUsers := Hub.GetOnlineUsers()
	return c.JSON(fiber.Map{
		"online_users": onlineUsers,
		"count":        len(onlineUsers),
	})
}

// GetRoomStats returns statistics about a room's active connections
func GetRoomStats(c *fiber.Ctx) error {
	if Hub == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "WebSocket service not available",
		})
	}

	roomIDStr := c.Params("roomId")
	roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid room ID",
		})
	}

	clientCount := Hub.GetRoomClientCount(uint(roomID))
	return c.JSON(fiber.Map{
		"room_id":        roomID,
		"active_clients": clientCount,
	})
}
