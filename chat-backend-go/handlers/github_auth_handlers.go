package handlers

import (
	"chat-backend-go/config"
	"chat-backend-go/models"
	"chat-backend-go/utils"
	"time"

	"github.com/gofiber/fiber/v2"
)

// NonceRequest represents nonce generation request
type NonceRequest struct{}

// NonceResponse represents nonce response
type NonceResponse struct {
	NonceID string `json:"nonce_id"`
	Nonce   string `json:"nonce"`
}

// SSHLoginRequest represents SSH login request
type SSHLoginRequest struct {
	GitHubUser     string `json:"github_user" validate:"required"`
	Signed         string `json:"signed" validate:"required"`
	PubFingerprint string `json:"pub_fingerprint" validate:"required"`
	NonceID        string `json:"nonce_id" validate:"required"`
	DeviceName     string `json:"device_name" validate:"required"`
	DeviceType     string `json:"device_type" validate:"required,oneof=web cli mobile"`
}

// DeviceResponse represents device information
type DeviceResponse struct {
	UserID   uint   `json:"user_id"`
	DeviceID string `json:"device_id"`
	User     models.User `json:"user"`
}

// RefreshRequest represents refresh token request
type RefreshRequest struct {
	DeviceID string `json:"device_id" validate:"required"`
}

// GenerateNonce creates a new nonce for authentication
func GenerateNonce(c *fiber.Ctx) error {
	// Generate nonce
	nonce, err := utils.GenerateNonce()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to generate nonce",
		})
	}

	// Generate nonce ID
	nonceID, err := utils.GenerateNonce()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to generate nonce ID",
		})
	}

	// Save nonce to database with TTL
	nonceModel := models.Nonce{
		NonceID:   nonceID,
		Nonce:     nonce,
		Used:      false,
		ExpiresAt: time.Now().Add(60 * time.Second), // 60 seconds TTL
	}

	if err := config.DB.Create(&nonceModel).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to save nonce",
		})
	}

	return c.JSON(NonceResponse{
		NonceID: nonceID,
		Nonce:   nonce,
	})
}

// SSHLogin handles SSH-based GitHub authentication
func SSHLogin(c *fiber.Ctx) error {
	var req SSHLoginRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Verify nonce
	var nonceModel models.Nonce
	if err := config.DB.Where("nonce_id = ? AND used = false", req.NonceID).First(&nonceModel).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid or expired nonce",
		})
	}

	// Check if nonce is expired
	if time.Now().After(nonceModel.ExpiresAt) {
		return c.Status(400).JSON(fiber.Map{
			"error": "Nonce expired",
		})
	}

	// Mark nonce as used
	config.DB.Model(&nonceModel).Update("used", true)

	// Verify SSH signature
	valid, err := utils.VerifySSHSignature(req.GitHubUser, req.Signed, req.PubFingerprint, nonceModel.Nonce)
	if err != nil || !valid {
		return c.Status(401).JSON(fiber.Map{
			"error": "SSH signature verification failed",
		})
	}

	// Get GitHub user info
	githubUser, err := utils.GetGitHubUser(req.GitHubUser)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Failed to fetch GitHub user",
		})
	}

	// Check if user exists or create new one
	var user models.User
	var githubUserModel models.GitHubUser

	// Check if GitHub user already exists
	if err := config.DB.Where("github_id = ?", githubUser.ID).First(&githubUserModel).Error; err != nil {
		// Create new user
		user = models.User{
			Username: githubUser.Login,
			Email:    githubUser.Email,
			Role:     "user",
			AuthType: "github",
		}

		if err := config.DB.Create(&user).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "Failed to create user",
			})
		}

		// Create GitHub user record
		githubUserModel = models.GitHubUser{
			UserID:    user.ID,
			GitHubID:  githubUser.ID,
			Login:     githubUser.Login,
			Name:      githubUser.Name,
			Email:     githubUser.Email,
			AvatarURL: githubUser.AvatarURL,
		}

		if err := config.DB.Create(&githubUserModel).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "Failed to create GitHub user record",
			})
		}
	} else {
		// Load existing user
		if err := config.DB.First(&user, githubUserModel.UserID).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "Failed to load user",
			})
		}
	}

	// Generate device ID
	deviceID, err := utils.GenerateDeviceID()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to generate device ID",
		})
	}

	// Generate refresh token
	refreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to generate refresh token",
		})
	}

	// Create device record
	device := models.Device{
		DeviceID:     deviceID,
		UserID:       user.ID,
		DeviceName:   req.DeviceName,
		DeviceType:   req.DeviceType,
		LastUsedAt:   time.Now(),
		IsActive:     true,
		RefreshToken: refreshToken,
	}

	if err := config.DB.Create(&device).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to create device record",
		})
	}

	// Generate JWT token
	token, err := utils.GenerateJWT(user.ID, deviceID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	// Set cookies
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    token,
		MaxAge:   24 * 60 * 60, // 24 hours
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
	})

	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
	})

	return c.JSON(DeviceResponse{
		UserID:   user.ID,
		DeviceID: deviceID,
		User:     user,
	})
}

// RefreshAuth refreshes access token using refresh token
func RefreshAuth(c *fiber.Ctx) error {
	refreshToken := c.Cookies("refresh_token")
	if refreshToken == "" {
		return c.Status(401).JSON(fiber.Map{
			"error": "No refresh token provided",
		})
	}

	// Find device by refresh token
	var device models.Device
	if err := config.DB.Where("refresh_token = ? AND is_active = true", refreshToken).First(&device).Error; err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid refresh token",
		})
	}

	// Load user
	var user models.User
	if err := config.DB.First(&user, device.UserID).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to load user",
		})
	}

	// Generate new tokens
	newAccessToken, err := utils.GenerateJWT(user.ID, device.DeviceID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to generate access token",
		})
	}

	newRefreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to generate refresh token",
		})
	}

	// Update device with new refresh token
	config.DB.Model(&device).Updates(models.Device{
		RefreshToken: newRefreshToken,
		LastUsedAt:   time.Now(),
	})

	// Set new cookies
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    newAccessToken,
		MaxAge:   24 * 60 * 60, // 24 hours
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
	})

	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    newRefreshToken,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
	})

	return c.JSON(fiber.Map{
		"message": "Token refreshed successfully",
	})
}

// Logout revokes the current device
func Logout(c *fiber.Ctx) error {
	refreshToken := c.Cookies("refresh_token")
	if refreshToken == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "No refresh token provided",
		})
	}

	// Revoke device
	config.DB.Model(&models.Device{}).Where("refresh_token = ?", refreshToken).Update("is_active", false)

	// Clear cookies
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    "",
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
	})

	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
	})

	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}

// GetMe returns current user information and devices
func GetMe(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		// Try to get token from cookie
		token := c.Cookies("access_token")
		if token != "" {
			authHeader = "Bearer " + token
		}
	}

	if authHeader == "" {
		return c.Status(401).JSON(fiber.Map{
			"error": "No authorization token provided",
		})
	}

	userID, deviceID, err := utils.ExtractUserAndDevice(authHeader)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid token",
		})
	}

	// Load user
	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Load user devices
	var devices []models.Device
	config.DB.Where("user_id = ? AND is_active = true", userID).Find(&devices)

	return c.JSON(fiber.Map{
		"user":       user,
		"current_device": deviceID,
		"devices":    devices,
	})
}

// RevokeDevice revokes a specific device
func RevokeDevice(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		// Try to get token from cookie
		token := c.Cookies("access_token")
		if token != "" {
			authHeader = "Bearer " + token
		}
	}

	if authHeader == "" {
		return c.Status(401).JSON(fiber.Map{
			"error": "No authorization token provided",
		})
	}

	userID, _, err := utils.ExtractUserAndDevice(authHeader)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid token",
		})
	}

	deviceIDParam := c.Params("deviceId")
	if deviceIDParam == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Device ID required",
		})
	}

	// Check if device belongs to user
	var device models.Device
	if err := config.DB.Where("device_id = ? AND user_id = ?", deviceIDParam, userID).First(&device).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Device not found",
		})
	}

	// Revoke device
	config.DB.Model(&device).Update("is_active", false)

	return c.JSON(fiber.Map{
		"message": "Device revoked successfully",
	})
}

// CleanupExpiredNonces removes expired nonces from database
func CleanupExpiredNonces() {
	config.DB.Where("expires_at < ?", time.Now()).Delete(&models.Nonce{})
}
