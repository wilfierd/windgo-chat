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
