package handlers

import (
	"chat-backend-go/config"
	"chat-backend-go/middleware"
	"chat-backend-go/models"
	"chat-backend-go/utils"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// SearchMessages handles GET /api/v1/search
// Query params: q (required), room_id (optional), cursor (optional), limit (default 20)
// Returns: results with total count, next_cursor, has_more flag
func SearchMessages(c *fiber.Ctx) error {
	// Get authenticated user ID from context
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return utils.RespondUnauthorized(c, "User not authenticated")
	}

	// Check if search service is available
	searchClient := config.GetSearchClient()
	if searchClient == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "Search service temporarily unavailable",
		})
	}

	// Parse and validate query parameter
	query := c.Query("q")
	if query == "" {
		return utils.RespondBadRequest(c, "Search query is required")
	}

	// Parse optional room_id parameter
	var roomIDs []uint
	roomIDStr := c.Query("room_id")
	if roomIDStr != "" {
		roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
		if err != nil {
			return utils.RespondBadRequest(c, "Invalid room_id parameter")
		}

		// Verify user has access to the specified room
		if err := utils.VerifyRoomMembership(config.DB, userID, uint(roomID)); err != nil {
			return utils.RespondForbidden(c, err.Error())
		}

		roomIDs = []uint{uint(roomID)}
	} else {
		// Get all rooms the user has access to
		var memberships []models.RoomMembership
		if err := config.DB.Where("user_id = ?", userID).Find(&memberships).Error; err != nil {
			return utils.RespondInternalErrorWithLog(c, err, "SearchMessages - fetch user memberships")
		}

		roomIDs = make([]uint, len(memberships))
		for i, membership := range memberships {
			roomIDs[i] = membership.RoomID
		}
	}

	// Parse pagination parameters
	cursor := c.Query("cursor", "")
	limit := c.QueryInt("limit", 20)
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100 // Max limit
	}

	// Perform search
	results, err := searchClient.Search(query, roomIDs, cursor, limit)
	if err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "SearchMessages - search failed")
	}

	return c.JSON(results)
}

// GetMessageNavigation handles GET /api/v1/search/navigate/:messageId
// Query params: q (required), room_id (optional)
// Returns: NavigationContext with position (e.g., "3 of 47") and prev/next IDs
func GetMessageNavigation(c *fiber.Ctx) error {
	// Get authenticated user ID from context
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return utils.RespondUnauthorized(c, "User not authenticated")
	}

	// Check if search service is available
	searchClient := config.GetSearchClient()
	if searchClient == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "Search service temporarily unavailable",
		})
	}

	// Parse message ID from URL parameter
	messageIDStr := c.Params("messageId")
	messageID, err := strconv.ParseUint(messageIDStr, 10, 32)
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid message ID")
	}

	// Parse and validate query parameter
	query := c.Query("q")
	if query == "" {
		return utils.RespondBadRequest(c, "Search query is required")
	}

	// Parse optional room_id parameter
	var roomIDs []uint
	roomIDStr := c.Query("room_id")
	if roomIDStr != "" {
		roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
		if err != nil {
			return utils.RespondBadRequest(c, "Invalid room_id parameter")
		}

		// Verify user has access to the specified room
		if err := utils.VerifyRoomMembership(config.DB, userID, uint(roomID)); err != nil {
			return utils.RespondForbidden(c, err.Error())
		}

		roomIDs = []uint{uint(roomID)}
	} else {
		// Get all rooms the user has access to
		var memberships []models.RoomMembership
		if err := config.DB.Where("user_id = ?", userID).Find(&memberships).Error; err != nil {
			return utils.RespondInternalErrorWithLog(c, err, "GetMessageNavigation - fetch user memberships")
		}

		roomIDs = make([]uint, len(memberships))
		for i, membership := range memberships {
			roomIDs[i] = membership.RoomID
		}
	}

	// Get navigation context
	navContext, err := searchClient.GetNavigationContext(query, roomIDs, uint(messageID))
	if err != nil {
		return utils.RespondInternalErrorWithLog(c, err, "GetMessageNavigation - get context failed")
	}

	return c.JSON(navContext)
}
