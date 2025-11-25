package handlers_test

import (
	"bytes"
	"chat-backend-go/config"
	"chat-backend-go/handlers"
	"chat-backend-go/middleware"
	"chat-backend-go/models"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// TestSendMessage_WithMentions tests that mentions are parsed and stored when sending a message
func TestSendMessage_WithMentions(t *testing.T) {
	db := setupTestDBShared(t)
	defer cleanupTestDBShared(t, db)
	config.DB = db

	// Create test users
	sender, senderToken := createTestUserShared(t, db, "sender", "sender@example.com", "user")
	mentioned1, _ := createTestUserShared(t, db, "alice", "alice@example.com", "user")
	mentioned2, _ := createTestUserShared(t, db, "bob", "bob@example.com", "user")

	// Create a test room
	room := &models.Room{
		Name: "Test Room",
		Type: models.RoomTypeGroup,
	}
	if err := db.Create(room).Error; err != nil {
		t.Fatalf("Failed to create test room: %v", err)
	}

	// Add users as room members
	memberships := []models.RoomMembership{
		{UserID: sender.ID, RoomID: room.ID},
		{UserID: mentioned1.ID, RoomID: room.ID},
		{UserID: mentioned2.ID, RoomID: room.ID},
	}
	if err := db.Create(&memberships).Error; err != nil {
		t.Fatalf("Failed to create room memberships: %v", err)
	}

	app := fiber.New()
	app.Post("/messages", middleware.AuthRequired(), handlers.SendMessage)

	// Test data with mentions
	reqBody := map[string]interface{}{
		"room_id": room.ID,
		"content": "Hey @alice and @bob, check this out!",
	}
	bodyJSON, _ := json.Marshal(reqBody)

	// Make request
	req := httptest.NewRequest("POST", "/messages", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+senderToken)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Assert response
	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	messageData := response["data"].(map[string]interface{})
	messageID := uint(messageData["id"].(float64))

	// Verify mentions were stored in database
	var mentions []models.MessageMention
	if err := db.Where("message_id = ?", messageID).Find(&mentions).Error; err != nil {
		t.Fatalf("Failed to fetch mentions: %v", err)
	}

	if len(mentions) != 2 {
		t.Errorf("Expected 2 mentions, got %d", len(mentions))
	}

	// Verify the correct users were mentioned
	mentionedUserIDs := make(map[uint]bool)
	for _, mention := range mentions {
		mentionedUserIDs[mention.MentionedUserID] = true
	}

	if !mentionedUserIDs[mentioned1.ID] {
		t.Errorf("Expected alice to be mentioned")
	}
	if !mentionedUserIDs[mentioned2.ID] {
		t.Errorf("Expected bob to be mentioned")
	}

	t.Logf("Successfully parsed and stored mentions for message %d", messageID)
}

// TestSendMessage_WithInvalidMentions tests that invalid mentions are ignored
func TestSendMessage_WithInvalidMentions(t *testing.T) {
	db := setupTestDBShared(t)
	defer cleanupTestDBShared(t, db)
	config.DB = db

	// Create test user
	sender, senderToken := createTestUserShared(t, db, "sender", "sender@example.com", "user")

	// Create a test room
	room := &models.Room{
		Name: "Test Room",
		Type: models.RoomTypeGroup,
	}
	if err := db.Create(room).Error; err != nil {
		t.Fatalf("Failed to create test room: %v", err)
	}

	// Add sender as room member
	membership := models.RoomMembership{UserID: sender.ID, RoomID: room.ID}
	if err := db.Create(&membership).Error; err != nil {
		t.Fatalf("Failed to create room membership: %v", err)
	}

	app := fiber.New()
	app.Post("/messages", middleware.AuthRequired(), handlers.SendMessage)

	// Test data with non-existent user mention
	reqBody := map[string]interface{}{
		"room_id": room.ID,
		"content": "Hey @nonexistentuser, are you there?",
	}
	bodyJSON, _ := json.Marshal(reqBody)

	// Make request
	req := httptest.NewRequest("POST", "/messages", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+senderToken)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Assert response - message should still be created
	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	messageData := response["data"].(map[string]interface{})
	messageID := uint(messageData["id"].(float64))

	// Verify no mentions were stored
	var mentions []models.MessageMention
	if err := db.Where("message_id = ?", messageID).Find(&mentions).Error; err != nil {
		t.Fatalf("Failed to fetch mentions: %v", err)
	}

	if len(mentions) != 0 {
		t.Errorf("Expected 0 mentions for non-existent user, got %d", len(mentions))
	}

	t.Logf("Successfully handled invalid mention without failing message creation")
}

// TestSendMessage_WithoutMentions tests that messages without mentions work normally
func TestSendMessage_WithoutMentions(t *testing.T) {
	db := setupTestDBShared(t)
	defer cleanupTestDBShared(t, db)
	config.DB = db

	// Create test user
	sender, senderToken := createTestUserShared(t, db, "sender", "sender@example.com", "user")

	// Create a test room
	room := &models.Room{
		Name: "Test Room",
		Type: models.RoomTypeGroup,
	}
	if err := db.Create(room).Error; err != nil {
		t.Fatalf("Failed to create test room: %v", err)
	}

	// Add sender as room member
	membership := models.RoomMembership{UserID: sender.ID, RoomID: room.ID}
	if err := db.Create(&membership).Error; err != nil {
		t.Fatalf("Failed to create room membership: %v", err)
	}

	app := fiber.New()
	app.Post("/messages", middleware.AuthRequired(), handlers.SendMessage)

	// Test data without mentions
	reqBody := map[string]interface{}{
		"room_id": room.ID,
		"content": "This is a regular message without any mentions",
	}
	bodyJSON, _ := json.Marshal(reqBody)

	// Make request
	req := httptest.NewRequest("POST", "/messages", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+senderToken)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Assert response
	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	messageData := response["data"].(map[string]interface{})
	messageID := uint(messageData["id"].(float64))

	// Verify no mentions were stored
	var mentions []models.MessageMention
	if err := db.Where("message_id = ?", messageID).Find(&mentions).Error; err != nil {
		t.Fatalf("Failed to fetch mentions: %v", err)
	}

	if len(mentions) != 0 {
		t.Errorf("Expected 0 mentions, got %d", len(mentions))
	}

	t.Logf("Successfully created message without mentions")
}
