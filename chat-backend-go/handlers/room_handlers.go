package handlers

import (
	"chat-backend-go/config"
	"chat-backend-go/models"
	"chat-backend-go/utils"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// CreateRoom creates a new chat room (admin only)
func CreateRoom(c *fiber.Ctx) error {
	// Get user ID from context
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return utils.RespondUnauthorized(c, utils.ErrUnauthorized)
	}

	// Check if user is admin
	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		utils.LogError("Failed to fetch user in CreateRoom", err, map[string]interface{}{
			"user_id": userID,
		})
		return utils.RespondNotFound(c, utils.ErrUserNotFound)
	}

	if user.Role != "admin" {
		utils.LogWarn("Non-admin user attempted to create room", map[string]interface{}{
			"user_id":  userID,
			"username": user.Username,
		})
		return utils.RespondForbidden(c, utils.ErrInsufficientPrivilege)
	}

	// Parse request body
	type CreateRoomRequest struct {
		Name string `json:"name"`
	}

	var req CreateRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.RespondBadRequest(c, "Invalid request data. Please check your input.")
	}

	// Validate and sanitize room name
	sanitizedName, err := utils.ValidateAndSanitizeRoomName(req.Name)
	if err != nil {
		return utils.RespondWithValidationError(c, err)
	}

	// Create new room (allow duplicate names - they'll be distinguished by ID)
	room := models.Room{
		Name: sanitizedName,
	}

	if err := config.DB.Create(&room).Error; err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "CreateRoom")
	}

	utils.LogInfo("Room created successfully", map[string]interface{}{
		"room_id":   room.ID,
		"room_name": room.Name,
		"user_id":   userID,
	})

	return utils.RespondCreated(c, "Room created successfully", room)
}

// UpdateRoom updates an existing chat room (admin only)
func UpdateRoom(c *fiber.Ctx) error {
	// Get user ID from context
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return utils.RespondUnauthorized(c, utils.ErrUnauthorized)
	}

	// Check if user is admin
	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		utils.LogError("Failed to fetch user in UpdateRoom", err, map[string]interface{}{
			"user_id": userID,
		})
		return utils.RespondNotFound(c, utils.ErrUserNotFound)
	}

	if user.Role != "admin" {
		utils.LogWarn("Non-admin user attempted to update room", map[string]interface{}{
			"user_id":  userID,
			"username": user.Username,
		})
		return utils.RespondForbidden(c, utils.ErrInsufficientPrivilege)
	}

	// Get room ID from URL params
	roomIDStr := c.Params("id")
	roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid room ID format")
	}

	// Parse request body
	type UpdateRoomRequest struct {
		Name string `json:"name"`
	}

	var req UpdateRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.RespondBadRequest(c, "Invalid request data. Please check your input.")
	}

	// Validate and sanitize room name
	sanitizedName, err := utils.ValidateAndSanitizeRoomName(req.Name)
	if err != nil {
		return utils.RespondWithValidationError(c, err)
	}

	// Check if room exists
	var room models.Room
	if err := config.DB.First(&room, uint(roomID)).Error; err != nil {
		return utils.RespondNotFound(c, utils.ErrRoomNotFound)
	}

	// Update room (allow duplicate names - they'll be distinguished by ID)
	room.Name = sanitizedName
	if err := config.DB.Save(&room).Error; err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "UpdateRoom")
	}

	utils.LogInfo("Room updated successfully", map[string]interface{}{
		"room_id":   room.ID,
		"room_name": room.Name,
		"user_id":   userID,
	})

	return utils.RespondSuccess(c, "Room updated successfully", room)
}

// DeleteRoom soft deletes a chat room (admin only)
func DeleteRoom(c *fiber.Ctx) error {
	// Get user ID from context
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return utils.RespondUnauthorized(c, utils.ErrUnauthorized)
	}

	// Check if user is admin
	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		utils.LogError("Failed to fetch user in DeleteRoom", err, map[string]interface{}{
			"user_id": userID,
		})
		return utils.RespondNotFound(c, utils.ErrUserNotFound)
	}

	if user.Role != "admin" {
		utils.LogWarn("Non-admin user attempted to delete room", map[string]interface{}{
			"user_id":  userID,
			"username": user.Username,
		})
		return utils.RespondForbidden(c, utils.ErrInsufficientPrivilege)
	}

	// Get room ID from URL params
	roomIDStr := c.Params("id")
	roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid room ID format")
	}

	// Check if room exists
	var room models.Room
	if err := config.DB.First(&room, uint(roomID)).Error; err != nil {
		return utils.RespondNotFound(c, utils.ErrRoomNotFound)
	}

	// Soft delete room (GORM will set DeletedAt timestamp)
	if err := config.DB.Delete(&room).Error; err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "DeleteRoom")
	}

	utils.LogInfo("Room soft-deleted successfully", map[string]interface{}{
		"room_id":   room.ID,
		"room_name": room.Name,
		"user_id":   userID,
	})

	return utils.RespondSuccess(c, "Room deleted successfully", nil)
}

// GetRoomByID retrieves a single room by ID
func GetRoomByID(c *fiber.Ctx) error {
	// Get room ID from URL params
	roomIDStr := c.Params("id")
	roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid room ID format")
	}

	var room models.Room
	if err := config.DB.First(&room, uint(roomID)).Error; err != nil {
		return utils.RespondNotFound(c, utils.ErrRoomNotFound)
	}

	return utils.RespondSuccess(c, "Room retrieved successfully", room)
}
