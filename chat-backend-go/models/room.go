// Package models defines database models with optimized indexes for the chat application.
// This file contains the Room model with relationships to messages and performance indexes.
package models

import (
	"time"

	"gorm.io/gorm"
)

type Room struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Name      string         `json:"name" gorm:"not null;index:idx_room_name"`
	Messages  []Message      `json:"messages,omitempty"`
	CreatedAt time.Time      `json:"created_at" gorm:"index:idx_room_created"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
