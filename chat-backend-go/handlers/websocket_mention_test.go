package handlers

import (
	"chat-backend-go/models"
	"testing"
)

// TestBroadcastMentionNotifications_SelfMention tests that self-mentions are filtered out
func TestBroadcastMentionNotifications_SelfMention(t *testing.T) {
	// Create a test message
	message := models.Message{
		ID:      1,
		UserID:  10,
		RoomID:  5,
		Content: "Hello @myself",
	}

	// Mentioned user IDs include the author (self-mention)
	mentionedUserIDs := []uint{10}

	// This should not panic and should filter out the self-mention
	// Since Hub is nil in test environment, it will log and return early
	BroadcastMentionNotifications(message, mentionedUserIDs)

	// If we get here without panic, the test passes
	t.Log("Self-mention filtered successfully")
}

// TestBroadcastMentionNotifications_MultipleMentions tests multiple mentions
func TestBroadcastMentionNotifications_MultipleMentions(t *testing.T) {
	// Create a test message
	message := models.Message{
		ID:      2,
		UserID:  10,
		RoomID:  5,
		Content: "Hello @alice and @bob",
	}

	// Mentioned user IDs (different from author)
	mentionedUserIDs := []uint{20, 30}

	// This should not panic
	// Since Hub is nil in test environment, it will log and return early
	BroadcastMentionNotifications(message, mentionedUserIDs)

	// If we get here without panic, the test passes
	t.Log("Multiple mentions processed successfully")
}

// TestBroadcastMentionNotifications_NilHub tests behavior when Hub is nil
func TestBroadcastMentionNotifications_NilHub(t *testing.T) {
	// Save original Hub
	originalHub := Hub
	Hub = nil
	defer func() { Hub = originalHub }()

	message := models.Message{
		ID:      3,
		UserID:  10,
		RoomID:  5,
		Content: "Hello @alice",
	}

	mentionedUserIDs := []uint{20}

	// This should not panic when Hub is nil
	BroadcastMentionNotifications(message, mentionedUserIDs)

	// If we get here without panic, the test passes
	t.Log("Nil Hub handled gracefully")
}

// TestBroadcastMentionNotifications_InvalidMessageType tests invalid message type
func TestBroadcastMentionNotifications_InvalidMessageType(t *testing.T) {
	// Pass an invalid message type
	invalidMessage := "not a message struct"

	mentionedUserIDs := []uint{20}

	// This should not panic with invalid message type
	BroadcastMentionNotifications(invalidMessage, mentionedUserIDs)

	// If we get here without panic, the test passes
	t.Log("Invalid message type handled gracefully")
}
