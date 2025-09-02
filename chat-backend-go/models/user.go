// Package models defines database models with optimized indexes for the chat application.
// This file contains the User model with performance-optimized database indexes.
package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
    ID        uint           `json:"id" gorm:"primaryKey"`
    Username  string         `json:"username" gorm:"unique;not null;index:idx_user_username"`
    Email     string         `json:"email" gorm:"unique;not null;index:idx_user_email"`
    Password  string         `json:"-" gorm:"not null"`
    Role      string         `json:"role" gorm:"not null;default:'user';index:idx_user_role"`
    // Social login fields
    Provider  string         `json:"provider" gorm:"index:idx_user_provider"`
    GitHubID  string         `json:"github_id" gorm:"uniqueIndex"`
    AvatarURL string         `json:"avatar_url"`
    CreatedAt time.Time      `json:"created_at" gorm:"index:idx_user_created"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
