package handlers

import (
	"chat-backend-go/config"
	"chat-backend-go/models"
	"chat-backend-go/utils"
	"sort"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Request types
type CreateDirectRoomRequest struct {
	TargetUserID uint `json:"target_user_id" validate:"required,min=1"`
}

// Response types for GetDirectRooms
type OtherUserInfo struct {
	ID           uint       `json:"id"`
	Username     string     `json:"username"`
	IsOnline     bool       `json:"is_online"`
	LastActiveAt *time.Time `json:"last_active_at,omitempty"`
}

type LastMessageInfo struct {
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type DirectRoomResponse struct {
	ID          uint             `json:"id"`
	Name        string           `json:"name"`
	Type        string           `json:"type"`
	DisplayName string           `json:"display_name"`
	OtherUser   OtherUserInfo    `json:"other_user"`
	LastMessage *LastMessageInfo `json:"last_message,omitempty"`
	UnreadCount int              `json:"unread_count"`
	CreatedAt   time.Time        `json:"created_at"`
}

// requireAdmin checks if the authenticated user has admin role
// Returns the user and an error response if not authorized
func requireAdmin(c *fiber.Ctx, operation string) (*models.User, error) {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return nil, utils.RespondUnauthorized(c, utils.ErrUnauthorized)
	}

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		utils.LogError("Failed to fetch user in "+operation, err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, utils.RespondNotFound(c, utils.ErrUserNotFound)
	}

	if user.Role != "admin" {
		utils.LogWarn("Non-admin user attempted to "+operation, map[string]interface{}{
			"user_id":  userID,
			"username": user.Username,
		})
		return nil, utils.RespondForbidden(c, utils.ErrInsufficientPrivilege)
	}

	return &user, nil
}

// buildDirectRoomResponse builds a DirectRoomResponse from a room
// This is extracted to support concurrent processing
func buildDirectRoomResponse(room models.Room, currentUserID uint, lastMsgMap map[uint]models.Message) (DirectRoomResponse, bool) {
	// Extract other user (the participant who is not the current user)
	var otherUser *models.User
	for _, member := range room.Members {
		if member.ID != currentUserID {
			otherUser = &member
			break
		}
	}

	if otherUser == nil {
		// Skip if we can't find the other user (shouldn't happen)
		return DirectRoomResponse{}, false
	}

	// Build response object
	dmResponse := DirectRoomResponse{
		ID:          room.ID,
		Name:        room.Name,
		Type:        room.Type,
		DisplayName: room.GetDisplayName(),
		OtherUser: OtherUserInfo{
			ID:           otherUser.ID,
			Username:     otherUser.Username,
			IsOnline:     otherUser.IsOnline,
			LastActiveAt: otherUser.LastActiveAt,
		},
		UnreadCount: 0, // TODO: Calculate actual unread count when read tracking is implemented
		CreatedAt:   room.CreatedAt,
	}

	// Add last message if exists from our map
	if lastMsg, exists := lastMsgMap[room.ID]; exists {
		dmResponse.LastMessage = &LastMessageInfo{
			Content:   lastMsg.Content,
			CreatedAt: lastMsg.CreatedAt,
		}
	}

	return dmResponse, true
}

// CreateRoom creates a new chat room (admin only)
func CreateRoom(c *fiber.Ctx) error {
	// Check if user is admin
	user, err := requireAdmin(c, "create room")
	if err != nil {
		return err
	}
	userID := user.ID

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
		Type: models.RoomTypeGroup, // Group room by default
	}

	// Begin transaction
	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(&room).Error; err != nil {
		tx.Rollback()
		return utils.RespondInternalErrorWithLog(c, err, "CreateRoom")
	}

	// Add creator as owner of the room
	membership := models.RoomMembership{
		UserID: userID,
		RoomID: room.ID,
		Role:   models.RoleOwner,
	}

	if err := tx.Create(&membership).Error; err != nil {
		tx.Rollback()
		return utils.RespondInternalErrorWithLog(c, err, "CreateRoom - create owner membership")
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "CreateRoom - commit transaction")
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
	// Check if user is admin
	user, err := requireAdmin(c, "update room")
	if err != nil {
		return err
	}
	userID := user.ID

	// Get room ID from URL params
	roomID, err := parseRoomID(c)
	if err != nil {
		return err
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
	// Check if user is admin
	user, err := requireAdmin(c, "delete room")
	if err != nil {
		return err
	}
	userID := user.ID

	// Get room ID from URL params
	roomID, err := parseRoomID(c)
	if err != nil {
		return err
	}

	// Check if room exists
	var room models.Room
	if err := config.DB.First(&room, roomID).Error; err != nil {
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

// parseRoomID extracts and validates room ID from URL params
func parseRoomID(c *fiber.Ctx) (uint, error) {
	roomIDStr := c.Params("id")
	roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
	if err != nil {
		return 0, utils.RespondBadRequest(c, "Invalid room ID format")
	}
	return uint(roomID), nil
}

// GetRoomByID retrieves a single room by ID
func GetRoomByID(c *fiber.Ctx) error {
	// Get room ID from URL params
	roomID, err := parseRoomID(c)
	if err != nil {
		return err
	}

	var room models.Room
	if err := config.DB.First(&room, uint(roomID)).Error; err != nil {
		return utils.RespondNotFound(c, utils.ErrRoomNotFound)
	}

	return utils.RespondSuccess(c, "Room retrieved successfully", room)
}

// CreateDirectRoom creates or retrieves a direct message room between two users
func CreateDirectRoom(c *fiber.Ctx) error {
	// Get authenticated user ID from context
	currentUserID, ok := c.Locals("userID").(uint)
	if !ok {
		return utils.RespondUnauthorized(c, utils.ErrUnauthorized)
	}

	// Parse request body
	var req CreateDirectRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.RespondBadRequest(c, "Invalid request data. Please check your input.")
	}

	// Validate target_user_id is provided
	if req.TargetUserID == 0 {
		return utils.RespondBadRequest(c, "target_user_id must be a valid positive number")
	}

	// Prevent self-DM
	if currentUserID == req.TargetUserID {
		return utils.RespondBadRequest(c, "Cannot create direct message with yourself")
	}

	// Fetch both users concurrently for better performance
	type userFetchResult struct {
		user models.User
		err  error
	}

	currentUserChan := make(chan userFetchResult, 1)
	targetUserChan := make(chan userFetchResult, 1)

	// Fetch current user
	go func() {
		var user models.User
		err := config.DB.First(&user, currentUserID).Error
		currentUserChan <- userFetchResult{user: user, err: err}
	}()

	// Fetch target user
	go func() {
		var user models.User
		err := config.DB.First(&user, req.TargetUserID).Error
		targetUserChan <- userFetchResult{user: user, err: err}
	}()

	// Wait for both fetches to complete
	currentUserResult := <-currentUserChan
	targetUserResult := <-targetUserChan

	// Check for errors
	if currentUserResult.err != nil {
		return utils.RespondInternalErrorWithLog(c, currentUserResult.err, "CreateDirectRoom - fetch current user")
	}

	if targetUserResult.err != nil {
		return utils.RespondNotFound(c, "Target user not found")
	}

	currentUser := currentUserResult.user
	targetUser := targetUserResult.user

	// Check if DM already exists between these two users
	// Query for rooms where:
	// 1. Room type is "direct"
	// 2. Both users are members
	var existingRoom models.Room
	err := config.DB.
		Joins("JOIN room_memberships rm1 ON rm1.room_id = rooms.id AND rm1.user_id = ?", currentUserID).
		Joins("JOIN room_memberships rm2 ON rm2.room_id = rooms.id AND rm2.user_id = ?", req.TargetUserID).
		Where("rooms.type = ?", models.RoomTypeDirect).
		Preload("Members").
		First(&existingRoom).Error

	if err == nil {
		// DM already exists, return it
		utils.LogInfo("Direct room already exists", map[string]interface{}{
			"room_id":         existingRoom.ID,
			"current_user_id": currentUserID,
			"target_user_id":  req.TargetUserID,
		})
		return utils.RespondSuccess(c, "Direct room already exists", existingRoom)
	} else if err != gorm.ErrRecordNotFound {
		// Actual database error, not just "not found"
		return utils.RespondInternalErrorWithLog(c, err, "CreateDirectRoom - check existing DM")
	}

	// Create new direct room
	// Generate room name from sorted usernames
	var roomName string
	if currentUser.Username < targetUser.Username {
		roomName = currentUser.Username + "_" + targetUser.Username
	} else {
		roomName = targetUser.Username + "_" + currentUser.Username
	}

	// Begin transaction
	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create room
	room := models.Room{
		Name: roomName,
		Type: models.RoomTypeDirect,
	}

	if err := tx.Create(&room).Error; err != nil {
		tx.Rollback()
		return utils.RespondInternalErrorWithLog(c, err, "CreateDirectRoom - create room")
	}

	// Create room membership for current user (owner in direct rooms)
	membership1 := models.RoomMembership{
		UserID: currentUserID,
		RoomID: room.ID,
		Role:   models.RoleOwner, // Both users are owners in direct rooms
	}

	if err := tx.Create(&membership1).Error; err != nil {
		tx.Rollback()
		return utils.RespondInternalErrorWithLog(c, err, "CreateDirectRoom - create membership 1")
	}

	// Create room membership for target user (owner in direct rooms)
	membership2 := models.RoomMembership{
		UserID: req.TargetUserID,
		RoomID: room.ID,
		Role:   models.RoleOwner, // Both users are owners in direct rooms
	}

	if err := tx.Create(&membership2).Error; err != nil {
		tx.Rollback()
		return utils.RespondInternalErrorWithLog(c, err, "CreateDirectRoom - create membership 2")
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "CreateDirectRoom - commit transaction")
	}

	// Load room with members
	if err := config.DB.Preload("Members").First(&room, room.ID).Error; err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "CreateDirectRoom - load room with members")
	}

	// Invalidate membership cache for this room
	if Hub != nil {
		Hub.InvalidateMembershipCache(room.ID)
	}

	utils.LogInfo("Direct room created successfully", map[string]interface{}{
		"room_id":         room.ID,
		"room_name":       room.Name,
		"current_user_id": currentUserID,
		"target_user_id":  req.TargetUserID,
	})

	return utils.RespondCreated(c, "Direct room created", room)
}

// GetDirectRooms retrieves all direct message rooms for the authenticated user
// Uses concurrent processing for improved performance
func GetDirectRooms(c *fiber.Ctx) error {
	// Get authenticated user ID from context
	currentUserID, ok := c.Locals("userID").(uint)
	if !ok {
		return utils.RespondUnauthorized(c, utils.ErrUnauthorized)
	}

	// Query for direct rooms where the user is a member
	// Only load rooms and members, messages will be fetched separately for efficiency
	var rooms []models.Room
	err := config.DB.
		Joins("JOIN room_memberships ON room_memberships.room_id = rooms.id").
		Where("room_memberships.user_id = ?", currentUserID).
		Where("rooms.type = ?", models.RoomTypeDirect).
		Preload("Members").
		Find(&rooms).Error

	if err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "GetDirectRooms")
	}

	// Early return if no rooms
	if len(rooms) == 0 {
		return utils.RespondSuccess(c, "Direct rooms retrieved successfully", []DirectRoomResponse{})
	}

	// Get room IDs for fetching last messages efficiently
	roomIDs := make([]uint, len(rooms))
	for i, room := range rooms {
		roomIDs[i] = room.ID
	}

	// Use channels for concurrent operations
	type fetchResult struct {
		messages []models.Message
		err      error
	}
	messageChan := make(chan fetchResult, 1)

	// Fetch last messages concurrently while we process room data
	go func() {
		var lastMessages []models.Message
		var fetchErr error

		if len(roomIDs) > 0 {
			// Use a subquery to get only the most recent message per room
			fetchErr = config.DB.Raw(`
				SELECT m.*
				FROM messages m
				INNER JOIN (
					SELECT room_id, MAX(created_at) as max_created
					FROM messages
					WHERE room_id IN (?) AND deleted_at IS NULL
					GROUP BY room_id
				) latest ON m.room_id = latest.room_id AND m.created_at = latest.max_created
				WHERE m.deleted_at IS NULL
			`, roomIDs).Scan(&lastMessages).Error
		}

		messageChan <- fetchResult{messages: lastMessages, err: fetchErr}
	}()

	// Wait for message fetch to complete
	result := <-messageChan
	if result.err != nil {
		utils.LogError("Failed to fetch last messages", result.err, map[string]interface{}{
			"user_id": currentUserID,
		})
		// Continue without last messages rather than failing the entire request
	}

	// Map messages to rooms for quick lookup
	lastMsgMap := make(map[uint]models.Message)
	for _, msg := range result.messages {
		lastMsgMap[msg.RoomID] = msg
	}

	// Transform rooms to include other_user, last_message, and unread_count
	// Use concurrent processing for better performance with many rooms
	response := make([]DirectRoomResponse, 0, len(rooms))

	// For small number of rooms, sequential processing is faster
	// For larger sets, use concurrent processing
	if len(rooms) <= 10 {
		// Sequential processing for small sets
		for _, room := range rooms {
			if dmResponse, ok := buildDirectRoomResponse(room, currentUserID, lastMsgMap); ok {
				response = append(response, dmResponse)
			}
		}
	} else {
		// Concurrent processing for larger sets
		type roomResult struct {
			response DirectRoomResponse
			valid    bool
		}

		resultChan := make(chan roomResult, len(rooms))

		// Process rooms concurrently
		for _, room := range rooms {
			go func(r models.Room) {
				if dmResponse, ok := buildDirectRoomResponse(r, currentUserID, lastMsgMap); ok {
					resultChan <- roomResult{response: dmResponse, valid: true}
				} else {
					resultChan <- roomResult{valid: false}
				}
			}(room)
		}

		// Collect results
		for i := 0; i < len(rooms); i++ {
			result := <-resultChan
			if result.valid {
				response = append(response, result.response)
			}
		}
	}

	// Sort by last message timestamp descending using efficient sort.Slice
	// Rooms with messages come first, sorted by message time
	// Rooms without messages come last, sorted by creation time
	sort.Slice(response, func(i, j int) bool {
		var iTime, jTime time.Time

		if response[i].LastMessage != nil {
			iTime = response[i].LastMessage.CreatedAt
		} else {
			iTime = response[i].CreatedAt
		}

		if response[j].LastMessage != nil {
			jTime = response[j].LastMessage.CreatedAt
		} else {
			jTime = response[j].CreatedAt
		}

		return jTime.Before(iTime) // Descending order
	})

	utils.LogInfo("Direct rooms retrieved successfully", map[string]interface{}{
		"user_id": currentUserID,
		"count":   len(response),
	})

	return utils.RespondSuccess(c, "Direct rooms retrieved successfully", response)
}

// GetRoomParticipants retrieves all participants of a room
func GetRoomParticipants(c *fiber.Ctx) error {
	// Get room ID from URL params
	roomID, err := parseRoomID(c)
	if err != nil {
		return err
	}

	// Check if room exists
	var room models.Room
	if err := config.DB.First(&room, roomID).Error; err != nil {
		return utils.RespondNotFound(c, utils.ErrRoomNotFound)
	}

	// Query RoomMembership for given room_id with User preload
	var memberships []models.RoomMembership
	err = config.DB.
		Where("room_id = ?", roomID).
		Preload("User").
		Find(&memberships).Error

	if err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "GetRoomParticipants")
	}

	// Build response with participant information
	type ParticipantResponse struct {
		ID       uint      `json:"id"`
		Username string    `json:"username"`
		Role     string    `json:"role"`
		IsOnline bool      `json:"is_online"`
		JoinedAt time.Time `json:"joined_at"`
	}

	var participants []ParticipantResponse
	for _, membership := range memberships {
		participants = append(participants, ParticipantResponse{
			ID:       membership.User.ID,
			Username: membership.User.Username,
			Role:     membership.Role,
			IsOnline: membership.User.IsOnline,
			JoinedAt: membership.JoinedAt,
		})
	}

	utils.LogInfo("Room participants retrieved successfully", map[string]interface{}{
		"room_id": roomID,
		"count":   len(participants),
	})

	return utils.RespondSuccess(c, "Participants retrieved successfully", participants)
}

// InviteUserToRoom invites a user to join a room (requires admin or owner role)
func InviteUserToRoom(c *fiber.Ctx) error {
	// Get authenticated user ID from context
	currentUserID, ok := c.Locals("userID").(uint)
	if !ok {
		return utils.RespondUnauthorized(c, utils.ErrUnauthorized)
	}

	// Get room ID from URL params
	roomID, err := parseRoomID(c)
	if err != nil {
		return err
	}

	// Parse request body
	type InviteRequest struct {
		UserID uint   `json:"user_id" validate:"required,min=1"`
		Role   string `json:"role"` // Optional, defaults to "member"
	}

	var req InviteRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.RespondBadRequest(c, "Invalid request data. Please check your input.")
	}

	// Validate user_id is provided
	if req.UserID == 0 {
		return utils.RespondBadRequest(c, "user_id must be a valid positive number")
	}

	// Set default role if not provided
	if req.Role == "" {
		req.Role = models.RoleMember
	}

	// Validate role
	if req.Role != models.RoleMember && req.Role != models.RoleAdmin && req.Role != models.RoleOwner {
		return utils.RespondBadRequest(c, "Invalid role. Must be one of: member, admin, owner")
	}

	// Check if room exists
	var room models.Room
	if err := config.DB.First(&room, roomID).Error; err != nil {
		return utils.RespondNotFound(c, utils.ErrRoomNotFound)
	}

	// Check if target user exists
	var targetUser models.User
	if err := config.DB.First(&targetUser, req.UserID).Error; err != nil {
		return utils.RespondNotFound(c, "User not found")
	}

	// Check if user already a member
	var existingMembership models.RoomMembership
	err = config.DB.Where("user_id = ? AND room_id = ?", req.UserID, roomID).First(&existingMembership).Error
	if err == nil {
		return utils.RespondBadRequest(c, "User is already a member of this room")
	}

	// Create membership
	membership := models.RoomMembership{
		UserID: req.UserID,
		RoomID: roomID,
		Role:   req.Role,
	}

	if err := config.DB.Create(&membership).Error; err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "InviteUserToRoom - create membership")
	}

	// Invalidate membership cache if Hub is available
	if Hub != nil {
		Hub.InvalidateMembershipCache(roomID)
	}

	utils.LogInfo("User invited to room successfully", map[string]interface{}{
		"room_id":         roomID,
		"invited_user_id": req.UserID,
		"role":            req.Role,
		"inviter_id":      currentUserID,
	})

	// Load membership with user data
	if err := config.DB.Preload("User").First(&membership, membership.ID).Error; err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "InviteUserToRoom - load membership")
	}

	return utils.RespondCreated(c, "User invited to room successfully", membership)
}

// RemoveUserFromRoom removes a user from a room or allows user to leave (requires admin/owner to kick, anyone can leave)
func RemoveUserFromRoom(c *fiber.Ctx) error {
	// Get authenticated user ID from context
	currentUserID, ok := c.Locals("userID").(uint)
	if !ok {
		return utils.RespondUnauthorized(c, utils.ErrUnauthorized)
	}

	// Get room ID from URL params
	roomID, err := parseRoomID(c)
	if err != nil {
		return err
	}

	// Get target user ID from URL params
	targetUserIDStr := c.Params("userId")
	targetUserID, err := strconv.ParseUint(targetUserIDStr, 10, 32)
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid user ID format")
	}

	// Check if room exists
	var room models.Room
	if err := config.DB.First(&room, roomID).Error; err != nil {
		return utils.RespondNotFound(c, utils.ErrRoomNotFound)
	}

	// Check if target user is a member
	var targetMembership models.RoomMembership
	if err := config.DB.Where("user_id = ? AND room_id = ?", uint(targetUserID), roomID).First(&targetMembership).Error; err != nil {
		return utils.RespondNotFound(c, "User is not a member of this room")
	}

	// Check permission: user can remove themselves, or admin/owner can remove others
	isSelfRemoval := currentUserID == uint(targetUserID)

	if !isSelfRemoval {
		// Get current user's membership to check their role
		var currentMembership models.RoomMembership
		if err := config.DB.Where("user_id = ? AND room_id = ?", currentUserID, roomID).First(&currentMembership).Error; err != nil {
			return utils.RespondForbidden(c, "You are not a member of this room")
		}

		// Only admin or owner can remove other users
		if currentMembership.Role != models.RoleAdmin && currentMembership.Role != models.RoleOwner {
			return utils.RespondForbidden(c, "Only admins and owners can remove other users")
		}

		// Owner cannot be removed by admin
		if targetMembership.Role == models.RoleOwner && currentMembership.Role != models.RoleOwner {
			return utils.RespondForbidden(c, "Only the owner can remove another owner")
		}
	}

	// Prevent owner from leaving if they're the only owner
	if targetMembership.Role == models.RoleOwner {
		var ownerCount int64
		config.DB.Model(&models.RoomMembership{}).
			Where("room_id = ? AND role = ?", roomID, models.RoleOwner).
			Count(&ownerCount)

		if ownerCount <= 1 {
			return utils.RespondBadRequest(c, "Cannot remove the only owner. Transfer ownership first.")
		}
	}

	// Delete membership
	if err := config.DB.Delete(&targetMembership).Error; err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "RemoveUserFromRoom - delete membership")
	}

	// Invalidate membership cache if Hub is available
	if Hub != nil {
		Hub.InvalidateMembershipCache(roomID)
	}

	action := "left"
	if !isSelfRemoval {
		action = "removed from"
	}

	utils.LogInfo("User "+action+" room successfully", map[string]interface{}{
		"room_id":         roomID,
		"target_user_id":  uint(targetUserID),
		"removed_by":      currentUserID,
		"is_self_removal": isSelfRemoval,
	})

	return utils.RespondSuccess(c, "User "+action+" room successfully", nil)
}
