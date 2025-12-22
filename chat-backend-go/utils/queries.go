// Package utils provides optimized database query functions for the chat application.
// This file contains pre-built, indexed-optimized queries for better performance.
// These functions leverage the database indexes defined in the models package.
package utils

import (
	"chat-backend-go/config"
	"chat-backend-go/models"
	"errors"
	"time"

	"gorm.io/gorm"
)

// GetUserByEmail - Optimized query using email index
func GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := config.DB.Where("email = ?", email).First(&user).Error
	return &user, err
}

// GetUserByUsername - Optimized query using username index
func GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := config.DB.Where("username = ?", username).First(&user).Error
	return &user, err
}

// GetRecentMessages - Optimized query using room and created_at indexes
func GetRecentMessages(roomID uint, limit int) ([]models.Message, error) {
	var messages []models.Message
	err := config.DB.Where("room_id = ?", roomID).
		Order("created_at DESC").
		Limit(limit).
		Preload("User").
		Find(&messages).Error
	return messages, err
}

// GetMessagesByUser - Optimized query using user index
func GetMessagesByUser(userID uint, limit int) ([]models.Message, error) {
	var messages []models.Message
	err := config.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Preload("Room").
		Find(&messages).Error
	return messages, err
}

// GetRoomByName - Optimized query using room name index
func GetRoomByName(name string) (*models.Room, error) {
	var room models.Room
	err := config.DB.Where("name = ?", name).First(&room).Error
	return &room, err
}

// GetRecentRooms - Optimized query using created_at index
func GetRecentRooms(limit int) ([]models.Room, error) {
	var rooms []models.Room
	err := config.DB.Order("created_at DESC").
		Limit(limit).
		Find(&rooms).Error
	return rooms, err
}

// GetRoomWithRecentMessages - Optimized compound query
func GetRoomWithRecentMessages(roomID uint, messageLimit int) (*models.Room, error) {
	var room models.Room

	// First get the room
	err := config.DB.First(&room, roomID).Error
	if err != nil {
		return nil, err
	}

	// Then get recent messages using optimized query
	err = config.DB.Where("room_id = ?", roomID).
		Order("created_at DESC").
		Limit(messageLimit).
		Preload("User").
		Find(&room.Messages).Error

	return &room, err
}

// GetMessagesInTimeRange - Optimized time range query
func GetMessagesInTimeRange(roomID uint, start, end time.Time) ([]models.Message, error) {
	var messages []models.Message
	err := config.DB.Where("room_id = ? AND created_at BETWEEN ? AND ?", roomID, start, end).
		Order("created_at ASC").
		Preload("User").
		Find(&messages).Error
	return messages, err
}

// GetUserStats - Efficient aggregation query
func GetUserStats(userID uint) (map[string]interface{}, error) {
	var stats map[string]interface{} = make(map[string]interface{})

	// Count total messages
	var messageCount int64
	err := config.DB.Model(&models.Message{}).Where("user_id = ?", userID).Count(&messageCount).Error
	if err != nil {
		return nil, err
	}
	stats["total_messages"] = messageCount

	// Get first message date
	var firstMessage models.Message
	err = config.DB.Where("user_id = ?", userID).Order("created_at ASC").First(&firstMessage).Error
	if err == nil {
		stats["member_since"] = firstMessage.CreatedAt
	}

	return stats, nil
}

// VerifyRoomMembership checks if a user is a member of a room
// Returns an error if the user is not a member
func VerifyRoomMembership(db *gorm.DB, userID, roomID uint) error {
	var membership models.RoomMembership
	if err := db.Where("user_id = ? AND room_id = ?", userID, roomID).First(&membership).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("you are not a participant of this conversation")
		}
		return err
	}
	return nil
}

// GetLastReadMessage - Get the last read message for a user in a room
func GetLastReadMessage(userID, roomID uint) (*models.UserRoomLastRead, error) {
	var lastRead []models.UserRoomLastRead
	err := config.DB.Where("user_id = ? AND room_id = ?", userID, roomID).
		Preload("LastReadMessage").
		Limit(1).
		Find(&lastRead).Error

	if err != nil {
		return nil, err
	}

	if len(lastRead) == 0 {
		return nil, nil // No last read record exists yet
	}
	return &lastRead[0], nil
}

// UpdateLastReadMessage - Update or create the last read message for a user in a room
func UpdateLastReadMessage(userID, roomID, messageID uint) error {
	lastRead := models.UserRoomLastRead{
		UserID:            userID,
		RoomID:            roomID,
		LastReadMessageID: &messageID,
		LastReadAt:        time.Now(),
	}

	// Use GORM's upsert functionality
	// This will update if exists (based on unique index), or create if not
	return config.DB.Where(models.UserRoomLastRead{UserID: userID, RoomID: roomID}).
		Assign(map[string]interface{}{
			"last_read_message_id": messageID,
			"last_read_at":         time.Now(),
		}).
		FirstOrCreate(&lastRead).Error
}

// GetUnreadCount - Get the count of unread messages for a user in a room
func GetUnreadCount(userID, roomID uint) (int64, error) {
	// Get the last read message
	lastRead, err := GetLastReadMessage(userID, roomID)
	if err != nil {
		return 0, err
	}

	var count int64

	// If no last read record, count all messages in the room
	if lastRead == nil || lastRead.LastReadMessageID == nil {
		err := config.DB.Model(&models.Message{}).
			Where("room_id = ?", roomID).
			Count(&count).Error
		return count, err
	}

	// Count messages created after the last read message
	// We need to get the timestamp of the last read message
	var lastReadMessage models.Message
	if err := config.DB.First(&lastReadMessage, lastRead.LastReadMessageID).Error; err != nil {
		return 0, err
	}

	err = config.DB.Model(&models.Message{}).
		Where("room_id = ? AND created_at > ?", roomID, lastReadMessage.CreatedAt).
		Count(&count).Error

	return count, err
}

// GetUnreadCountsForUser - Get unread counts for all rooms the user is a member of
func GetUnreadCountsForUser(userID uint) (map[uint]int64, error) {
	// Get all rooms the user is a member of
	var memberships []models.RoomMembership
	if err := config.DB.Where("user_id = ?", userID).Find(&memberships).Error; err != nil {
		return nil, err
	}

	unreadCounts := make(map[uint]int64)

	// Get unread count for each room
	for _, membership := range memberships {
		count, err := GetUnreadCount(userID, membership.RoomID)
		if err != nil {
			// Log error but continue for other rooms
			continue
		}
		unreadCounts[membership.RoomID] = count
	}

	return unreadCounts, nil
}
