// Package models defines database models with optimized indexes for the chat application.
// This file contains the MessageMention model for tracking user mentions in messages.
package models

import (
	"time"

	"gorm.io/gorm"
)

type MessageMention struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	MessageID       uint           `json:"message_id" gorm:"not null;index:idx_mention_message;uniqueIndex:idx_unique_mention"`
	Message         Message        `json:"message" gorm:"foreignKey:MessageID;constraint:OnDelete:CASCADE"`
	MentionedUserID uint           `json:"mentioned_user_id" gorm:"not null;index:idx_mention_user;uniqueIndex:idx_unique_mention;index:idx_mention_user_created,priority:1"`
	MentionedUser   User           `json:"mentioned_user" gorm:"foreignKey:MentionedUserID;constraint:OnDelete:CASCADE"`
	CreatedAt       time.Time      `json:"created_at" gorm:"index:idx_mention_user_created,priority:2"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}
