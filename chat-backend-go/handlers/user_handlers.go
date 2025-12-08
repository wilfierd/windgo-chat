package handlers

import (
	"chat-backend-go/config"
	"chat-backend-go/middleware"
	"chat-backend-go/models"
	"chat-backend-go/utils"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

const (
	// OnlineThresholdMinutes defines how many minutes of inactivity before a user is considered offline
	OnlineThresholdMinutes = 5
)

// buildUserAvailableResponse builds a UserAvailableResponse from a user
// This is extracted to support concurrent processing
func buildUserAvailableResponse(user models.User, existingDMs map[uint]bool) UserAvailableResponse {
	// Calculate online status based on last_active_at
	isOnline := false
	if user.LastActiveAt != nil {
		timeSince := time.Since(*user.LastActiveAt)
		isOnline = timeSince < OnlineThresholdMinutes*time.Minute
	}

	return UserAvailableResponse{
		ID:       user.ID,
		Username: user.Username,
		IsOnline: isOnline,
		HasDM:    existingDMs[user.ID],
	}
}

// ListUsers returns other users for chat directory, optionally filtered by search query
func ListUsers(c *fiber.Ctx) error {
	// Get user ID from JWT middleware (type-safe)
	currentUserID, ok := middleware.GetUserID(c)
	if !ok {
		return utils.RespondUnauthorized(c, "User not authenticated")
	}
	search := strings.ToLower(strings.TrimSpace(c.Query("search")))

	var users []models.User
	query := config.DB.Model(&models.User{}).Where("id <> ?", currentUserID)

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("LOWER(username) LIKE ? OR LOWER(email) LIKE ?", like, like)
	}

	if err := query.Order("username ASC").Find(&users).Error; err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "ListUsers - fetch users")
	}

	// Calculate online status based on last_active_at
	for i := range users {
		if users[i].LastActiveAt != nil {
			timeSince := time.Since(*users[i].LastActiveAt)
			users[i].IsOnline = timeSince < OnlineThresholdMinutes*time.Minute
			if users[i].IsOnline {
				users[i].Status = "online"
			} else {
				users[i].Status = "offline"
			}
		} else {
			users[i].IsOnline = false
			users[i].Status = "offline"
		}
	}

	return c.JSON(fiber.Map{
		"users": users,
	})
}

// UserAvailableResponse represents a user available for starting DMs
type UserAvailableResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	IsOnline bool   `json:"is_online"`
	HasDM    bool   `json:"has_dm"`
}

// GetAvailableUsers returns all users available for starting DMs with current user
func GetAvailableUsers(c *fiber.Ctx) error {
	// Get user ID from JWT middleware (type-safe)
	currentUserID, ok := middleware.GetUserID(c)
	if !ok {
		return utils.RespondUnauthorized(c, "User not authenticated")
	}

	// Query all users except current user
	var users []models.User
	if err := config.DB.Where("id <> ?", currentUserID).Order("username ASC").Find(&users).Error; err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "GetAvailableUsers - fetch users")
	}

	// Get user IDs that have existing DMs with current user using optimized query
	var existingDMUserIDs []uint
	err := config.DB.Raw(`
		SELECT DISTINCT 
			CASE 
				WHEN rm1.user_id = ? THEN rm2.user_id
				ELSE rm1.user_id
			END as other_user_id
		FROM room_memberships rm1
		JOIN room_memberships rm2 ON rm1.room_id = rm2.room_id AND rm1.user_id != rm2.user_id
		JOIN rooms ON rooms.id = rm1.room_id
		WHERE (rm1.user_id = ? OR rm2.user_id = ?)
		AND rooms.type = ?
		AND rooms.deleted_at IS NULL
	`, currentUserID, currentUserID, currentUserID, models.RoomTypeDirect).Scan(&existingDMUserIDs).Error

	if err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "GetAvailableUsers - check existing DMs")
	}

	// Build a map of user IDs that have existing DMs with current user
	existingDMs := make(map[uint]bool)
	for _, userID := range existingDMUserIDs {
		existingDMs[userID] = true
	}

	// Build response with online status and has_dm flag
	// Use concurrent processing for better performance with many users
	response := make([]UserAvailableResponse, len(users))

	// For small number of users, sequential processing is faster
	if len(users) <= 20 {
		for i, user := range users {
			response[i] = buildUserAvailableResponse(user, existingDMs)
		}
	} else {
		// Concurrent processing for larger sets using worker pool
		const numWorkers = 10
		userChan := make(chan int, len(users))

		// Send user indices to channel
		for i := range users {
			userChan <- i
		}
		close(userChan)

		// Start workers
		var wg sync.WaitGroup
		for w := 0; w < numWorkers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range userChan {
					response[i] = buildUserAvailableResponse(users[i], existingDMs)
				}
			}()
		}

		wg.Wait()
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Users retrieved successfully",
		"data":    response,
	})
}

// GetUserById returns a specific user by ID
func GetUserById(c *fiber.Ctx) error {
	// Get user ID from JWT middleware (type-safe)
	_, ok := middleware.GetUserID(c)
	if !ok {
		return utils.RespondUnauthorized(c, "User not authenticated")
	}

	// Parse user ID from URL params
	userID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Calculate online status based on last_active_at
	if user.LastActiveAt != nil {
		timeSince := time.Since(*user.LastActiveAt)
		user.IsOnline = timeSince < OnlineThresholdMinutes*time.Minute
		if user.IsOnline {
			user.Status = "online"
		} else {
			user.Status = "offline"
		}
	} else {
		user.IsOnline = false
		user.Status = "offline"
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User retrieved successfully",
		"data":    user,
	})
}
