package utils

import (
	"chat-backend-go/config"
	"chat-backend-go/models"
	"time"
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
