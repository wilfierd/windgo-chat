package handlers

import (
	"chat-backend-go/config"
	"chat-backend-go/models"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// SendMessage creates a new message in a room
func SendMessage(c *fiber.Ctx) error {
	// Get user ID from JWT middleware (we'll assume it's set)
	userID := c.Locals("userID")
	if userID == nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "User not authenticated",
		})
	}

	type MessageRequest struct {
		RoomID   uint   `json:"room_id" validate:"required"`
		Content  string `json:"content" validate:"required"`
		ParentID *uint  `json:"parent_id,omitempty"`
	}

	var req MessageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate room exists
	var room models.Room
	if err := config.DB.First(&room, req.RoomID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Room not found",
		})
	}

	// Validate parent message exists if ParentID is provided
	if req.ParentID != nil {
		var parentMsg models.Message
		if err := config.DB.First(&parentMsg, *req.ParentID).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{
				"error": "Parent message not found",
			})
		}
		// Ensure parent message is in the same room
		if parentMsg.RoomID != req.RoomID {
			return c.Status(400).JSON(fiber.Map{
				"error": "Parent message must be in the same room",
			})
		}
	}

	// Create message
	message := models.Message{
		UserID:   userID.(uint),
		RoomID:   req.RoomID,
		Content:  req.Content,
		ParentID: req.ParentID,
	}

	if err := config.DB.Create(&message).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to create message",
		})
	}

	// Load user data and parent message for response
	if err := config.DB.Preload("User").Preload("ParentMessage.User").First(&message, message.ID).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to load message data",
		})
	}

	// Broadcast message to WebSocket clients in the room
	BroadcastMessage(message.RoomID, "message", message, message.UserID)

	return c.Status(201).JSON(fiber.Map{
		"message": "Message sent successfully",
		"data":    message,
	})
}

// GetMessages retrieves messages for a specific room
func GetMessages(c *fiber.Ctx) error {
	roomIDStr := c.Params("roomId")
	roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid room ID",
		})
	}

	// Validate room exists
	var room models.Room
	if err := config.DB.First(&room, roomID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Room not found",
		})
	}

	// Get pagination parameters
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 50)
	if limit > 100 {
		limit = 100 // Max limit
	}
	offset := (page - 1) * limit

	var messages []models.Message
	if err := config.DB.
		Preload("User").
		Preload("ParentMessage.User").
		Where("room_id = ?", roomID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch messages",
		})
	}

	// Count total messages for pagination
	var total int64
	config.DB.Model(&models.Message{}).Where("room_id = ?", roomID).Count(&total)

	return c.JSON(fiber.Map{
		"messages": messages,
		"pagination": fiber.Map{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetRooms retrieves all available rooms
func GetRooms(c *fiber.Ctx) error {
	var rooms []models.Room
	if err := config.DB.Find(&rooms).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch rooms",
		})
	}

	return c.JSON(fiber.Map{
		"rooms": rooms,
	})
}

// UpdateMessage updates an existing message
func UpdateMessage(c *fiber.Ctx) error {
	// Get user ID from JWT middleware
	userID := c.Locals("userID")
	if userID == nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "User not authenticated",
		})
	}

	// Get message ID from URL params
	messageIDStr := c.Params("id")
	messageID, err := strconv.ParseUint(messageIDStr, 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid message ID",
		})
	}

	type UpdateRequest struct {
		Content string `json:"content" validate:"required"`
	}

	var req UpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Fetch the message
	var message models.Message
	if err := config.DB.First(&message, messageID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Message not found",
		})
	}

	// Check if user owns the message
	if message.UserID != userID.(uint) {
		return c.Status(403).JSON(fiber.Map{
			"error": "You can only edit your own messages",
		})
	}

	// Update the message content
	message.Content = req.Content
	if err := config.DB.Save(&message).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to update message",
		})
	}

	// Reload with user data and parent message for response
	if err := config.DB.Preload("User").Preload("ParentMessage.User").First(&message, message.ID).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to load message data",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Message updated successfully",
		"data":    message,
	})
}

// DeleteMessage soft deletes a message
func DeleteMessage(c *fiber.Ctx) error {
	// Get user ID from JWT middleware
	userID := c.Locals("userID")
	if userID == nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "User not authenticated",
		})
	}

	// Get message ID from URL params
	messageIDStr := c.Params("id")
	messageID, err := strconv.ParseUint(messageIDStr, 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid message ID",
		})
	}

	// Fetch the message
	var message models.Message
	if err := config.DB.First(&message, messageID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Message not found",
		})
	}

	// Check if user owns the message
	if message.UserID != userID.(uint) {
		return c.Status(403).JSON(fiber.Map{
			"error": "You can only delete your own messages",
		})
	}

	// Soft delete the message
	if err := config.DB.Delete(&message).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to delete message",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Message deleted successfully",
	})
}
