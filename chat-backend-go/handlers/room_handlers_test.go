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



// TestCreateRoom_AsAdmin tests room creation by admin user
func TestCreateRoom_AsAdmin(t *testing.T) {
	// Setup
	db := setupTestDBShared(t)
	defer cleanupTestDBShared(t, db)
	config.DB = db

	adminUser, adminToken := createTestUserShared(t, db, "testuser_admin", "test_admin@example.com", "admin")

	app := fiber.New()
	app.Post("/rooms", middleware.AuthRequired(), handlers.CreateRoom)

	// Test data
	reqBody := map[string]string{
		"name": "Test Room",
	}
	bodyJSON, _ := json.Marshal(reqBody)

	// Make request
	req := httptest.NewRequest("POST", "/rooms", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Assert
	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	if response["message"] != "Room created successfully" {
		t.Errorf("Unexpected response message: %v", response["message"])
	}

	// Verify room was created in database
	var room models.Room
	if err := db.Where("name = ?", "Test Room").First(&room).Error; err != nil {
		t.Errorf("Room was not created in database: %v", err)
	}

	t.Logf("Admin user %s successfully created room", adminUser.Username)
}

// TestCreateRoom_AsNonAdmin tests that non-admin users cannot create rooms
func TestCreateRoom_AsNonAdmin(t *testing.T) {
	// Setup
	db := setupTestDBShared(t)
	defer cleanupTestDBShared(t, db)
	config.DB = db

	regularUser, regularToken := createTestUserShared(t, db, "testuser_user", "test_user@example.com", "user")

	app := fiber.New()
	app.Post("/rooms", middleware.AuthRequired(), handlers.CreateRoom)

	// Test data
	reqBody := map[string]string{
		"name": "Unauthorized Room",
	}
	bodyJSON, _ := json.Marshal(reqBody)

	// Make request
	req := httptest.NewRequest("POST", "/rooms", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+regularToken)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Assert - should be forbidden
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("Expected status 403, got %d", resp.StatusCode)
	}

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	if response["error"] != "Admin access required" {
		t.Errorf("Unexpected error message: %v", response["error"])
	}

	t.Logf("Non-admin user %s correctly denied room creation", regularUser.Username)
}

// TestUpdateRoom_AsAdmin tests room update by admin
func TestUpdateRoom_AsAdmin(t *testing.T) {
	// Setup
	db := setupTestDBShared(t)
	defer cleanupTestDBShared(t, db)
	config.DB = db

	adminUser, adminToken := createTestUserShared(t, db, "testuser_admin", "test_admin@example.com", "admin")

	// Create a room first
	room := &models.Room{Name: "Original Room"}
	if err := db.Create(room).Error; err != nil {
		t.Fatalf("Failed to create test room: %v", err)
	}

	app := fiber.New()
	app.Put("/rooms/:id", middleware.AuthRequired(), handlers.UpdateRoom)

	// Test data
	reqBody := map[string]string{
		"name": "Updated Room",
	}
	bodyJSON, _ := json.Marshal(reqBody)

	// Make request
	req := httptest.NewRequest("PUT", "/rooms/"+string(rune(room.ID+'0')), bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Assert
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify room was updated in database
	var updatedRoom models.Room
	if err := db.First(&updatedRoom, room.ID).Error; err != nil {
		t.Fatalf("Failed to fetch updated room: %v", err)
	}

	if updatedRoom.Name != "Updated Room" {
		t.Errorf("Room name was not updated. Expected 'Updated Room', got '%s'", updatedRoom.Name)
	}

	t.Logf("Admin user %s successfully updated room", adminUser.Username)
}

// TestDeleteRoom_AsAdmin tests room deletion by admin (soft delete)
func TestDeleteRoom_AsAdmin(t *testing.T) {
	// Setup
	db := setupTestDBShared(t)
	defer cleanupTestDBShared(t, db)
	config.DB = db

	adminUser, adminToken := createTestUserShared(t, db, "testuser_admin", "test_admin@example.com", "admin")

	// Create a room first
	room := &models.Room{Name: "Room to Delete"}
	if err := db.Create(room).Error; err != nil {
		t.Fatalf("Failed to create test room: %v", err)
	}

	app := fiber.New()
	app.Delete("/rooms/:id", middleware.AuthRequired(), handlers.DeleteRoom)

	// Make request
	req := httptest.NewRequest("DELETE", "/rooms/"+string(rune(room.ID+'0')), nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Assert
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify room was soft deleted (should not be found in normal query)
	var deletedRoom models.Room
	if err := db.First(&deletedRoom, room.ID).Error; err == nil {
		t.Errorf("Room should be soft deleted but was still found")
	}

	// Verify room exists with Unscoped (including soft deleted)
	var roomWithDeleted models.Room
	if err := db.Unscoped().First(&roomWithDeleted, room.ID).Error; err != nil {
		t.Errorf("Room should exist in database with DeletedAt set: %v", err)
	}

	if !roomWithDeleted.DeletedAt.Valid {
		t.Errorf("Room DeletedAt should be set")
	}

	t.Logf("Admin user %s successfully soft-deleted room", adminUser.Username)
}

// TestGetRoomByID tests retrieving a room by ID
func TestGetRoomByID(t *testing.T) {
	// Setup
	db := setupTestDBShared(t)
	defer cleanupTestDBShared(t, db)
	config.DB = db

	// Create a room
	room := &models.Room{Name: "Test Room"}
	if err := db.Create(room).Error; err != nil {
		t.Fatalf("Failed to create test room: %v", err)
	}

	app := fiber.New()
	app.Get("/rooms/:id", handlers.GetRoomByID)

	// Make request
	req := httptest.NewRequest("GET", "/rooms/"+string(rune(room.ID+'0')), nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Assert
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	roomData := response["room"].(map[string]interface{})
	if roomData["name"] != "Test Room" {
		t.Errorf("Unexpected room name: %v", roomData["name"])
	}

	t.Logf("Successfully retrieved room by ID")
}

// TestCreateRoom_EmptyName tests validation for empty room name
func TestCreateRoom_EmptyName(t *testing.T) {
	// Setup
	db := setupTestDBShared(t)
	defer cleanupTestDBShared(t, db)
	config.DB = db

	_, adminToken := createTestUserShared(t, db, "testuser_admin", "test_admin@example.com", "admin")

	app := fiber.New()
	app.Post("/rooms", middleware.AuthRequired(), handlers.CreateRoom)

	// Test data with empty name
	reqBody := map[string]string{
		"name": "",
	}
	bodyJSON, _ := json.Marshal(reqBody)

	// Make request
	req := httptest.NewRequest("POST", "/rooms", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Assert - should be bad request
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	if response["error"] != "Room name is required" {
		t.Errorf("Unexpected error message: %v", response["error"])
	}

	t.Logf("Empty room name correctly rejected")
}
