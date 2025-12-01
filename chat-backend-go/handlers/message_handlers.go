package handlers

import (
	"chat-backend-go/config"
	"chat-backend-go/middleware"
	"chat-backend-go/models"
	"chat-backend-go/utils"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
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

	// Parse and store mentions
	var mentionedUserIDs []uint
	usernames := utils.ParseMentions(req.Content)
	if len(usernames) > 0 {
		// Validate mentioned usernames exist
		validatedUserIDs, err := utils.ValidateMentions(config.DB, usernames)
		if err != nil {
			// Log error but don't fail message creation
			utils.LogError("Failed to validate mentions", err, map[string]interface{}{
				"message_id": message.ID,
				"usernames":  usernames,
			})
		} else if len(validatedUserIDs) > 0 {
			mentionedUserIDs = validatedUserIDs
			// Store mention records
			if err := utils.StoreMentions(config.DB, message.ID, mentionedUserIDs); err != nil {
				// Log error but don't fail message creation
				utils.LogError("Failed to store mentions", err, map[string]interface{}{
					"message_id": message.ID,
					"user_ids":   mentionedUserIDs,
				})
			}
		}
	}

	// Load user data, room, and parent message for response and search indexing
	if err := config.DB.Preload("User").Preload("Room").Preload("ParentMessage.User").First(&message, message.ID).Error; err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "SendMessage - load message data")
	}

	// Index message in search engine (non-blocking)
	if config.SearchClient != nil {
		if err := config.SearchClient.IndexMessage(&message); err != nil {
			// Log error but don't fail message creation
			utils.LogError("Failed to index message in search", err, map[string]interface{}{
				"message_id": message.ID,
				"room_id":    message.RoomID,
			})
		}
	}

	// Broadcast message to WebSocket clients in the room
	BroadcastMessage(message.RoomID, "message", message, message.UserID)

	// Send mention notifications to mentioned users
	if len(mentionedUserIDs) > 0 {
		BroadcastMentionNotifications(message, mentionedUserIDs)
	}

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

	// Get unread counts for all rooms
	unreadCounts, err := utils.GetUnreadCountsForUser(userID)
	if err != nil {
		// Log error but continue with rooms without unread counts
		unreadCounts = make(map[uint]int64)
	}

	// Build response with unread counts
	type RoomWithUnread struct {
		models.Room
		UnreadCount int64 `json:"unread_count"`
	}

	roomsWithUnread := make([]RoomWithUnread, len(rooms))
	for i, room := range rooms {
		roomsWithUnread[i] = RoomWithUnread{
			Room:        room,
			UnreadCount: unreadCounts[room.ID],
		}
	}

	return c.JSON(fiber.Map{
		"rooms": roomsWithUnread,
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

	// Update mention records to reflect content changes
	if err := utils.UpdateMentions(config.DB, message.ID, req.Content); err != nil {
		// Log error but don't fail message update
		utils.LogError("Failed to update mentions", err, map[string]interface{}{
			"message_id": message.ID,
			"content":    req.Content,
		})
	}

	// Reload with user data, room, and parent message for response and search indexing
	if err := config.DB.Preload("User").Preload("Room").Preload("ParentMessage.User").First(&message, message.ID).Error; err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "UpdateMessage - load message data")
	}

	// Re-index message in search engine (non-blocking)
	if config.SearchClient != nil {
		if err := config.SearchClient.IndexMessage(&message); err != nil {
			// Log error but don't fail message update
			utils.LogError("Failed to re-index message in search", err, map[string]interface{}{
				"message_id": message.ID,
				"room_id":    message.RoomID,
			})
		}
	}

	return c.JSON(fiber.Map{
		"message": "Message updated successfully",
		"data":    message,
	})
}

// DeleteMessage soft deletes a message and its associated mentions
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

	// Use transaction to ensure atomicity of message and mention deletion
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		// Count mentions before deletion for logging
		var mentionCount int64
		if err := tx.Model(&models.MessageMention{}).Where("message_id = ?", messageID).Count(&mentionCount).Error; err != nil {
			utils.LogError("Failed to count mentions before cascade delete", err, map[string]interface{}{
				"message_id": messageID,
			})
			// Continue with deletion even if count fails
		}

		// Soft delete associated mentions (cascade delete for soft deletes)
		if err := tx.Where("message_id = ?", messageID).Delete(&models.MessageMention{}).Error; err != nil {
			utils.LogError("Failed to cascade delete mentions", err, map[string]interface{}{
				"message_id": messageID,
			})
			return err
		}

		// Log cascade delete
		if mentionCount > 0 {
			utils.LogInfo("Cascade deleted mentions for message", map[string]interface{}{
				"message_id":    messageID,
				"mention_count": mentionCount,
				"user_id":       userID,
			})
		}

		// Soft delete the message
		if err := tx.Delete(&message).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "DeleteMessage - delete message")
	}

	// Remove message from search index (non-blocking)
	if config.SearchClient != nil {
		if err := config.SearchClient.RemoveMessage(uint(messageID)); err != nil {
			// Log error but don't fail message deletion
			utils.LogError("Failed to remove message from search index", err, map[string]interface{}{
				"message_id": messageID,
			})
		}
	}

	return c.JSON(fiber.Map{
		"message": "Message deleted successfully",
	})
}

// MarkRoomAsRead marks all messages in a room as read for the authenticated user
func MarkRoomAsRead(c *fiber.Ctx) error {
	// Get authenticated user ID from context
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return utils.RespondUnauthorized(c, "User not authenticated")
	}

	// Get room ID from URL params
	roomIDStr := c.Params("id")
	roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid room ID")
	}

	// Validate room exists
	var room models.Room
	if err := config.DB.First(&room, roomID).Error; err != nil {
		return utils.RespondNotFound(c, "Room not found")
	}

	// Verify user is a room participant
	if err := utils.VerifyRoomMembership(config.DB, userID, uint(roomID)); err != nil {
		return utils.RespondForbidden(c, err.Error())
	}

	// Get the most recent message in the room
	var lastMessage models.Message
	if err := config.DB.
		Where("room_id = ?", roomID).
		Order("created_at DESC").
		First(&lastMessage).Error; err != nil {
		// If there are no messages in the room, still return success
		// No need to track last read for empty rooms
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(fiber.Map{
				"message": "Room marked as read",
				"data": fiber.Map{
					"room_id":      roomID,
					"unread_count": 0,
				},
			})
		}
		return utils.RespondInternalErrorWithLog(c, err, "MarkRoomAsRead - fetch last message")
	}

	// Update or create the last read record
	if err := utils.UpdateLastReadMessage(userID, uint(roomID), lastMessage.ID); err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "MarkRoomAsRead - update last read")
	}

	return c.JSON(fiber.Map{
		"message": "Room marked as read",
		"data": fiber.Map{
			"room_id":              roomID,
			"last_read_message_id": lastMessage.ID,
			"unread_count":         0,
		},
	})
}

// GetMentions retrieves messages where the authenticated user was mentioned
func GetMentions(c *fiber.Ctx) error {
	// Get authenticated user ID from JWT context
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return utils.RespondUnauthorized(c, "User not authenticated")
	}

	// Parse query parameters
	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}

	limit := c.QueryInt("limit", 50)
	if limit > 100 {
		limit = 100 // Max limit
	}
	if limit < 1 {
		limit = 50
	}

	// Optional room_id filter
	roomIDStr := c.Query("room_id")
	var roomID *uint
	if roomIDStr != "" {
		parsedRoomID, err := strconv.ParseUint(roomIDStr, 10, 32)
		if err != nil {
			return utils.RespondBadRequest(c, "Invalid room_id parameter")
		}
		roomIDUint := uint(parsedRoomID)
		roomID = &roomIDUint
	}

	offset := (page - 1) * limit

	// Build query for message_mentions
	query := config.DB.Model(&models.MessageMention{}).
		Preload("Message.User").
		Preload("Message.Room").
		Preload("Message.ParentMessage.User").
		Preload("MentionedUser").
		Joins("JOIN messages ON messages.id = message_mentions.message_id").
		Where("message_mentions.mentioned_user_id = ?", userID).
		Where("messages.deleted_at IS NULL") // Filter out soft-deleted messages

	// Apply room filter if provided
	if roomID != nil {
		query = query.Where("messages.room_id = ?", *roomID)
	}

	// Count total mentions for pagination
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "GetMentions - count mentions")
	}

	// Fetch mentions ordered by message creation time (newest first)
	var mentions []models.MessageMention
	if err := query.
		Order("messages.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&mentions).Error; err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "GetMentions - fetch mentions")
	}

	return c.JSON(fiber.Map{
		"mentions": mentions,
		"pagination": fiber.Map{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}
