package utils

import (
	"chat-backend-go/models"
	"chat-backend-go/search"
	"fmt"
	"log"

	"gorm.io/gorm"
)

const (
	reindexBatchSize = 100 // Process messages in batches to avoid memory issues
)

// ReindexAllMessages indexes all existing messages in the search engine
// This should be run when enabling search for the first time or after data migration
func ReindexAllMessages(db *gorm.DB, searchClient search.SearchClient) error {
	if searchClient == nil {
		return fmt.Errorf("search client is nil")
	}

	log.Println("Starting bulk reindex of all messages...")

	// Get total message count
	var totalCount int64
	if err := db.Model(&models.Message{}).Count(&totalCount).Error; err != nil {
		return fmt.Errorf("failed to count messages: %w", err)
	}

	log.Printf("Found %d messages to index\n", totalCount)

	if totalCount == 0 {
		log.Println("No messages to index")
		return nil
	}

	// Process messages in batches
	var indexed int64
	var failed int64
	offset := 0

	for {
		var messages []models.Message

		// Fetch batch with all required associations
		err := db.
			Preload("User").
			Preload("Room").
			Limit(reindexBatchSize).
			Offset(offset).
			Find(&messages).Error

		if err != nil {
			return fmt.Errorf("failed to fetch messages at offset %d: %w", offset, err)
		}

		// No more messages
		if len(messages) == 0 {
			break
		}

		// Index each message in the batch
		for _, message := range messages {
			if err := searchClient.IndexMessage(&message); err != nil {
				log.Printf("Warning: Failed to index message ID %d: %v\n", message.ID, err)
				failed++
			} else {
				indexed++
			}
		}

		// Update progress
		log.Printf("Progress: %d/%d indexed (%d failed)\n", indexed, totalCount, failed)

		// Move to next batch
		offset += reindexBatchSize
	}

	log.Printf("Reindexing complete: %d indexed, %d failed out of %d total\n", indexed, failed, totalCount)

	if failed > 0 {
		return fmt.Errorf("reindexing completed with %d failures", failed)
	}

	return nil
}

// ReindexRoomMessages indexes all messages in a specific room
// Useful for partial reindexing after room-specific changes
func ReindexRoomMessages(db *gorm.DB, searchClient search.SearchClient, roomID uint) error {
	if searchClient == nil {
		return fmt.Errorf("search client is nil")
	}

	log.Printf("Starting reindex of messages in room %d...\n", roomID)

	// Verify room exists
	var room models.Room
	if err := db.First(&room, roomID).Error; err != nil {
		return fmt.Errorf("room not found: %w", err)
	}

	// Get total message count for the room
	var totalCount int64
	if err := db.Model(&models.Message{}).Where("room_id = ?", roomID).Count(&totalCount).Error; err != nil {
		return fmt.Errorf("failed to count messages: %w", err)
	}

	log.Printf("Found %d messages in room '%s' to index\n", totalCount, room.Name)

	if totalCount == 0 {
		log.Println("No messages to index")
		return nil
	}

	// Process messages in batches
	var indexed int64
	var failed int64
	offset := 0

	for {
		var messages []models.Message

		// Fetch batch with all required associations
		err := db.
			Preload("User").
			Preload("Room").
			Where("room_id = ?", roomID).
			Limit(reindexBatchSize).
			Offset(offset).
			Find(&messages).Error

		if err != nil {
			return fmt.Errorf("failed to fetch messages at offset %d: %w", offset, err)
		}

		// No more messages
		if len(messages) == 0 {
			break
		}

		// Index each message in the batch
		for _, message := range messages {
			if err := searchClient.IndexMessage(&message); err != nil {
				log.Printf("Warning: Failed to index message ID %d: %v\n", message.ID, err)
				failed++
			} else {
				indexed++
			}
		}

		// Update progress
		log.Printf("Progress: %d/%d indexed (%d failed)\n", indexed, totalCount, failed)

		// Move to next batch
		offset += reindexBatchSize
	}

	log.Printf("Room reindexing complete: %d indexed, %d failed out of %d total\n", indexed, failed, totalCount)

	if failed > 0 {
		return fmt.Errorf("reindexing completed with %d failures", failed)
	}

	return nil
}
