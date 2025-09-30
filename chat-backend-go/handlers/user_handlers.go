package handlers

import (
	"chat-backend-go/config"
	"chat-backend-go/models"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// ListUsers returns other users for chat directory, optionally filtered by search query
func ListUsers(c *fiber.Ctx) error {
	currentUserID := c.Locals("userID").(uint)
	search := strings.ToLower(strings.TrimSpace(c.Query("search")))

	var users []models.User
	query := config.DB.Model(&models.User{}).Where("id <> ?", currentUserID)

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("LOWER(username) LIKE ? OR LOWER(email) LIKE ?", like, like)
	}

	if err := query.Order("username ASC").Find(&users).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch users",
		})
	}

	// Calculate online status based on last_active_at
	// User is online if active within last 5 minutes
	for i := range users {
		if users[i].LastActiveAt != nil {
			timeSince := time.Since(*users[i].LastActiveAt)
			users[i].IsOnline = timeSince < 5*time.Minute
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
