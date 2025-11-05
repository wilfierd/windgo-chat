package handlers

import (
	"chat-backend-go/config"
	"chat-backend-go/middleware"
	"chat-backend-go/models"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// SendMessage creates a new message in a room
func SendMessage(c *fiber.Ctx) error {
	// Get user ID from JWT middleware (type-safe)
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return utils.RespondUnauthorized(c, "User not authenticated")
	}

	type MessageRequest struct {
		RoomID   uint   `json:"room_id" validate:"required"`
		Content  string `json:"content" validate:"required"`
		ParentID *uint  `json:"parent_id,omitempty"`
	}

	var req MessageRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.RespondBadRequest(c, "Invalid request body")
	}

	// Validate room exists
	var room models.Room
	if err := config.DB.First(&room, req.RoomID).Error; err != nil {
		return utils.RespondNotFound(c, "Room not found")
	}

	// Verify sender is a room participant (for direct messages and group rooms)
	if err := utils.VerifyRoomMembership(config.DB, userID, req.RoomID); err != nil {
		return utils.RespondForbidden(c, err.Error())
	}

	// Validate parent message exists if ParentID is provided
	if req.ParentID != nil {
		var parentMsg models.Message
		if err := config.DB.First(&parentMsg, *req.ParentID).Error; err != nil {
			return utils.RespondNotFound(c, "Parent message not found")
		}
		// Ensure parent message is in the same room
		if parentMsg.RoomID != req.RoomID {
			return utils.RespondBadRequest(c, "Parent message must be in the same room")
		}
	}

	// Create message
	message := models.Message{
		UserID:   userID,
		RoomID:   req.RoomID,
		Content:  req.Content,
		ParentID: req.ParentID,
	}

	if err := config.DB.Create(&message).Error; err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "SendMessage - create message")
	}

	// Update room's last activity timestamp (UpdatedAt)
	if err := config.DB.Model(&room).Update("updated_at", message.CreatedAt).Error; err != nil {
		// Log error but don't fail the request
		// The message was created successfully
	}

	// Load user data and parent message for response
	if err := config.DB.Preload("User").Preload("ParentMessage.User").First(&message, message.ID).Error; err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "SendMessage - load message data")
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
	// Get authenticated user ID from context
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return utils.RespondUnauthorized(c, "User not authenticated")
	}

	roomIDStr := c.Params("roomId")
	roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid room ID")
	}

	// Validate room exists
	var room models.Room
	if err := config.DB.First(&room, roomID).Error; err != nil {
		return utils.RespondNotFound(c, "Room not found")
	}

	// Verify requester is a room participant (for direct messages and group rooms)
	if err := utils.VerifyRoomMembership(config.DB, userID, uint(roomID)); err != nil {
		return utils.RespondForbidden(c, err.Error())
	}

	// Get pagination parameters
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 50)
	if limit > 100 {
		limit = 100 // Max limit
	}
	offset := (page - 1) * limit

	// Fetch messages - GORM automatically excludes soft-deleted messages
	var messages []models.Message
	if err := config.DB.
		Preload("User").
		Preload("ParentMessage.User").
		Where("room_id = ?", roomID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error; err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "GetMessages - fetch messages")
	}

	// Count total messages for pagination (excluding soft-deleted)
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

// GetRooms retrieves all rooms the authenticated user is a member of (excluding direct rooms)
func GetRooms(c *fiber.Ctx) error {
	// Get authenticated user ID from context
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return utils.RespondUnauthorized(c, utils.ErrUnauthorized)
	}

	// Query for group rooms where the user is a member
	var rooms []models.Room
	err := config.DB.
		Joins("JOIN room_memberships ON room_memberships.room_id = rooms.id").
		Where("room_memberships.user_id = ?", userID).
		Where("rooms.type = ?", models.RoomTypeGroup).
		Find(&rooms).Error

	if err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "GetRooms - fetch rooms")
	}

	return c.JSON(fiber.Map{
		"rooms": rooms,
	})
}

// UpdateMessage updates an existing message
func UpdateMessage(c *fiber.Ctx) error {
	// Get user ID from JWT middleware (type-safe)
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return utils.RespondUnauthorized(c, "User not authenticated")
	}

	// Get message ID from URL params
	messageIDStr := c.Params("id")
	messageID, err := strconv.ParseUint(messageIDStr, 10, 32)
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid message ID")
	}

	type UpdateRequest struct {
		Content string `json:"content" validate:"required"`
	}

	var req UpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.RespondBadRequest(c, "Invalid request body")
	}

	// Fetch the message
	var message models.Message
	if err := config.DB.First(&message, messageID).Error; err != nil {
		return utils.RespondNotFound(c, "Message not found")
	}

	// Check if user owns the message
	if message.UserID != userID {
		return utils.RespondForbidden(c, "You can only edit your own messages")
	}

	// Update the message content
	message.Content = req.Content
	if err := config.DB.Save(&message).Error; err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "UpdateMessage - save message")
	}

	// Reload with user data and parent message for response
	if err := config.DB.Preload("User").Preload("ParentMessage.User").First(&message, message.ID).Error; err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "UpdateMessage - load message data")
	}

	return c.JSON(fiber.Map{
		"message": "Message updated successfully",
		"data":    message,
	})
}

// DeleteMessage soft deletes a message
func DeleteMessage(c *fiber.Ctx) error {
	// Get user ID from JWT middleware (type-safe)
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return utils.RespondUnauthorized(c, "User not authenticated")
	}

	// Get message ID from URL params
	messageIDStr := c.Params("id")
	messageID, err := strconv.ParseUint(messageIDStr, 10, 32)
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid message ID")
	}

	// Fetch the message
	var message models.Message
	if err := config.DB.First(&message, messageID).Error; err != nil {
		return utils.RespondNotFound(c, "Message not found")
	}

	// Check if user owns the message
	if message.UserID != userID {
		return utils.RespondForbidden(c, "You can only delete your own messages")
	}

	// Soft delete the message
	if err := config.DB.Delete(&message).Error; err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "DeleteMessage - delete message")
	}

	return c.JSON(fiber.Map{
		"message": "Message deleted successfully",
	})
}
