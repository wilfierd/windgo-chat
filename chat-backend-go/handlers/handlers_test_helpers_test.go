package handlers_test

import (
	"chat-backend-go/models"
	"chat-backend-go/utils"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// setupTestDBShared initializes a test database
func setupTestDBShared(t *testing.T) *gorm.DB {
	dsn := "host=localhost user=postgres password=password dbname=windgo_chat_test port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("Skipping test: Failed to connect to test database: %v", err)
	}

	// Auto migrate tables
	err = db.AutoMigrate(&models.User{}, &models.Room{}, &models.RoomMembership{}, &models.Message{}, &models.MessageMention{})
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

// cleanupTestDBShared clears all test data
func cleanupTestDBShared(t *testing.T, db *gorm.DB) {
	db.Exec("DELETE FROM message_mentions")
	db.Exec("DELETE FROM messages")
	db.Exec("DELETE FROM room_memberships")
	db.Exec("DELETE FROM rooms")
	db.Exec("DELETE FROM users")
}

// createTestUserShared creates a test user and returns the user and JWT token
func createTestUserShared(t *testing.T, db *gorm.DB, username, email, role string) (*models.User, string) {
	user := &models.User{
		Username: username,
		Email:    email,
		Password: "$2a$10$xyz",
		Role:     role,
	}

	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	token, err := utils.GenerateJWT(user.ID)
	if err != nil {
		t.Fatalf("Failed to generate JWT token: %v", err)
	}

	return user, token
}
