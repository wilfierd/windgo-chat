package middleware

import (
	"chat-backend-go/config"
	"chat-backend-go/models"
	"time"

	"github.com/gofiber/fiber/v2"
)

// GetUserID safely retrieves the userID from context
func GetUserID(c *fiber.Ctx) (uint, bool) {
	userID := c.Locals(string(UserIDKey))
	if userID == nil {
		return 0, false
	}
	id, ok := userID.(uint)
	return id, ok
}

// TrackActivity updates the user's last_active_at timestamp on each request
func TrackActivity() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Continue with the request first
		err := c.Next()

		// After request is processed, update user activity
		if userID, ok := GetUserID(c); ok {
			now := time.Now()
			// Update asynchronously to not slow down response
			go func() {
				config.DB.Model(&models.User{}).
					Where("id = ?", userID).
					Updates(map[string]interface{}{
						"last_active_at": now,
						"is_online":      true,
						"status":         "online",
					})
			}()
		}

		return err
	}
}
