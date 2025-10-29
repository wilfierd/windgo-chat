package handlers

import (
	"chat-backend-go/config"
	"chat-backend-go/models"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// CreateRoom creates a new chat room (admin only)
func CreateRoom(c *fiber.Ctx) error {
	// Get user ID from context
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Check if user is admin
	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	if user.Role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	// Parse request body
	type CreateRoomRequest struct {
		Name string `json:"name"`
	}

	var req CreateRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate room name
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Room name is required",
		})
	}

	// Create new room (allow duplicate names - they'll be distinguished by ID)
	room := models.Room{
		Name: req.Name,
	}

	if err := config.DB.Create(&room).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create room",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Room created successfully",
		"room":    room,
	})
}

// UpdateRoom updates an existing chat room (admin only)
func UpdateRoom(c *fiber.Ctx) error {
	// Get user ID from context
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Check if user is admin
	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	if user.Role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	// Get room ID from URL params
	roomIDStr := c.Params("id")
	roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid room ID",
		})
	}

	// Parse request body
	type UpdateRoomRequest struct {
		Name string `json:"name"`
	}

	var req UpdateRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate room name
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Room name is required",
		})
	}

	// Check if room exists
	var room models.Room
	if err := config.DB.First(&room, uint(roomID)).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Room not found",
		})
	}

	// Update room (allow duplicate names - they'll be distinguished by ID)
	room.Name = req.Name
	if err := config.DB.Save(&room).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update room",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Room updated successfully",
		"room":    room,
	})
}

// DeleteRoom soft deletes a chat room (admin only)
func DeleteRoom(c *fiber.Ctx) error {
	// Get user ID from context
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Check if user is admin
	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	if user.Role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	// Get room ID from URL params
	roomIDStr := c.Params("id")
	roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid room ID",
		})
	}

	// Check if room exists
	var room models.Room
	if err := config.DB.First(&room, uint(roomID)).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Room not found",
		})
	}

	// Soft delete room (GORM will set DeletedAt timestamp)
	if err := config.DB.Delete(&room).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete room",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Room deleted successfully",
	})
}

// GetRoomByID retrieves a single room by ID
func GetRoomByID(c *fiber.Ctx) error {
	// Get room ID from URL params
	roomIDStr := c.Params("id")
	roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid room ID",
		})
	}

	var room models.Room
	if err := config.DB.First(&room, uint(roomID)).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Room not found",
		})
	}

	return c.JSON(fiber.Map{
		"room": room,
	})
}
