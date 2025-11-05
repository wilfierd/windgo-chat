// Package models defines database models with optimized indexes for the chat application.
// This file contains the RoomMembership model for tracking user participation in rooms.
package models

import (
	"time"
)

const (
	// RoleMember represents a regular room member with basic permissions
	RoleMember = "member"
	// RoleAdmin represents a room admin with elevated permissions (can invite/kick)
	RoleAdmin = "admin"
	// RoleOwner represents the room owner with full permissions
	RoleOwner = "owner"
)

type RoomMembership struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null;index:idx_membership_user;uniqueIndex:idx_membership_composite"`
	RoomID    uint      `json:"room_id" gorm:"not null;index:idx_membership_room;uniqueIndex:idx_membership_composite"`
	Role      string    `json:"role" gorm:"not null;default:'member';index:idx_membership_role"`
	User      User      `json:"user" gorm:"foreignKey:UserID"`
	Room      Room      `json:"room" gorm:"foreignKey:RoomID"`
	JoinedAt  time.Time `json:"joined_at" gorm:"autoCreateTime"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
