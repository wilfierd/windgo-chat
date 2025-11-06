package models

import (
	"time"

	"gorm.io/gorm"
)

// UserRoomLastRead tracks the last message a user has read in each room
type UserRoomLastRead struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	UserID           uint           `json:"user_id" gorm:"not null;index:idx_last_read_user;uniqueIndex:idx_last_read_composite"`
	RoomID           uint           `json:"room_id" gorm:"not null;index:idx_last_read_room;uniqueIndex:idx_last_read_composite"`
	LastReadMessageID *uint         `json:"last_read_message_id" gorm:"index:idx_last_read_message"`
	LastReadAt       time.Time      `json:"last_read_at" gorm:"not null;index:idx_last_read_at"`
	User             User           `json:"user" gorm:"foreignKey:UserID"`
	Room             Room           `json:"room" gorm:"foreignKey:RoomID"`
	LastReadMessage  *Message       `json:"last_read_message,omitempty" gorm:"foreignKey:LastReadMessageID"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}
