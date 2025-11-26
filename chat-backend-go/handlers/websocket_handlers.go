// Package handlers contains HTTP request handlers for the chat application.
// This file contains WebSocket handlers for real-time communication.
package handlers

import (
	"chat-backend-go/middleware"
	"chat-backend-go/models"
	ws "chat-backend-go/websocket"
	"fmt"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"gorm.io/gorm"
)

var (
	// Hub is the global WebSocket hub instance
	Hub *ws.Hub
)

// InitWebSocketHub initializes the WebSocket hub with database connection
func InitWebSocketHub(db *gorm.DB) {
	Hub = ws.NewHub(db)
	go Hub.Run()
	log.Println("WebSocket hub initialized and running")
}

// WebSocketUpgrade upgrades the HTTP connection to WebSocket
func WebSocketUpgrade(c *fiber.Ctx) error {
	// Check if the request is a WebSocket upgrade request
	if websocket.IsWebSocketUpgrade(c) {
		// Get user ID from JWT middleware (type-safe)
		userID, ok := middleware.GetUserID(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not authenticated",
			})
		}

		// Store userID in context for the WebSocket handler
		c.Locals("wsUserID", userID)
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

// BroadcastMentionNotifications sends individual mention notifications to mentioned users
// Filters out self-mentions and verifies room membership before sending
func BroadcastMentionNotifications(message interface{}, mentionedUserIDs []uint) {
	if Hub == nil {
		log.Println("WebSocket hub not initialized, skipping mention notifications")
		return
	}

	// Extract message details for filtering and notification
	var messageID uint
	var roomID uint
	var authorID uint

	// Type assert to get message details
	switch msg := message.(type) {
	case models.Message:
		messageID = msg.ID
		roomID = msg.RoomID
		authorID = msg.UserID
	default:
		log.Printf("Invalid message type for mention notifications: %T", message)
		return
	}

	// Filter out self-mentions and send notifications
	for _, mentionedUserID := range mentionedUserIDs {
		// Skip self-mentions
		if mentionedUserID == authorID {
			log.Printf("Skipping self-mention for user %d in message %d", mentionedUserID, messageID)
			continue
		}

		// Verify room membership
		if !Hub.IsRoomMember(mentionedUserID, roomID) {
			log.Printf("User %d is not a member of room %d, skipping mention notification", mentionedUserID, roomID)
			continue
		}

		// Send individual mention notification to the mentioned user
		go sendMentionNotification(mentionedUserID, roomID, message)
	}
}

// sendMentionNotification sends a mention notification to a specific user
func sendMentionNotification(userID uint, roomID uint, message interface{}) {
	if Hub == nil {
		return
	}

	// Create mention notification message
	notification := &ws.Message{
		Type:    "mention",
		RoomID:  roomID,
		UserID:  userID,
		Content: map[string]interface{}{
			"message": message,
		},
	}

	// Send to specific user via Hub
	Hub.SendToUser(userID, notification)
}
