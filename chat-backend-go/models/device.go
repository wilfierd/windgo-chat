package models

import (
	"time"

	"gorm.io/gorm"
)

// Device represents a user's authenticated device
type Device struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	DeviceID     string         `json:"device_id" gorm:"unique;not null;index:idx_device_id"`
	UserID       uint           `json:"user_id" gorm:"not null;index:idx_device_user"`
	User         User           `json:"user" gorm:"foreignKey:UserID"`
	DeviceName   string         `json:"device_name" gorm:"not null"`
	DeviceType   string         `json:"device_type" gorm:"not null"` // "web", "cli", "mobile"
	LastUsedAt   time.Time      `json:"last_used_at" gorm:"index:idx_device_last_used"`
	IsActive     bool           `json:"is_active" gorm:"default:true;index:idx_device_active"`
	RefreshToken string         `json:"-" gorm:"unique;index:idx_device_refresh"`
	CreatedAt    time.Time      `json:"created_at" gorm:"index:idx_device_created"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

// Nonce represents a temporary nonce for authentication
type Nonce struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	NonceID   string    `json:"nonce_id" gorm:"unique;not null;index:idx_nonce_id"`
	Nonce     string    `json:"nonce" gorm:"not null"`
	Used      bool      `json:"used" gorm:"default:false;index:idx_nonce_used"`
	ExpiresAt time.Time `json:"expires_at" gorm:"index:idx_nonce_expires"`
	CreatedAt time.Time `json:"created_at"`
}

// GitHubUser represents GitHub user data
type GitHubUser struct {
	ID       uint   `json:"id" gorm:"primaryKey"`
	UserID   uint   `json:"user_id" gorm:"unique;not null;index:idx_github_user"`
	User     User   `json:"user" gorm:"foreignKey:UserID"`
	GitHubID int    `json:"github_id" gorm:"unique;not null;index:idx_github_id"`
	Login    string `json:"login" gorm:"not null;index:idx_github_login"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
