package models

import (
	"time"

	"gorm.io/gorm"
)

type Message struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Content   string         `json:"content" gorm:"not null"`
	UserID    uint           `json:"user_id" gorm:"not null;index:idx_message_user"`
	User      User           `json:"user" gorm:"foreignKey:UserID"`
	RoomID    uint           `json:"room_id" gorm:"not null;index:idx_message_room"`
	Room      Room           `json:"room" gorm:"foreignKey:RoomID"`
	CreatedAt time.Time      `json:"created_at" gorm:"index:idx_message_created"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
