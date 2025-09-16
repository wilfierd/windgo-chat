package handlers

import (
	"chat-backend-go/config"
	"chat-backend-go/models"
	"strings"

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

	return c.JSON(fiber.Map{
		"users": users,
	})
}
