// Package models defines database models with optimized indexes for the chat application.
// This file contains the Room model with relationships to messages and performance indexes.
//
// Room Display Names:
// - Rooms can have duplicate names (e.g., multiple "Meeting Room")
// - Each room is uniquely identified by its ID
// - The display_name field automatically shows: "Room Name (#123)"
// - This allows users to distinguish between rooms with the same name
package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

const (
	// RoomTypeGroup represents a group chat room
	RoomTypeGroup = "group"
	// RoomTypeDirect represents a direct message room between two users
	RoomTypeDirect = "direct"
)

type Room struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"not null;index:idx_room_name"`
	Type        string         `json:"type" gorm:"not null;default:'group';index:idx_room_type"`
	DisplayName string         `json:"display_name" gorm:"-"` // Computed field: "Name (#ID)"
	Messages    []Message      `json:"messages,omitempty"`
	Members     []User         `json:"members,omitempty" gorm:"many2many:room_memberships;"`
	CreatedAt   time.Time      `json:"created_at" gorm:"index:idx_room_created"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// AfterFind hook to populate DisplayName
func (r *Room) AfterFind(tx *gorm.DB) error {
	r.DisplayName = r.GetDisplayName()
	return nil
}

// GetDisplayName returns the room name with ID: "Room Name (#123)"
func (r *Room) GetDisplayName() string {
	return fmt.Sprintf("%s (#%d)", r.Name, r.ID)
}
