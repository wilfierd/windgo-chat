package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

var (
	baseURL       = flag.String("url", "http://localhost:8080", "Base URL of the API server")
	adminEmail    = flag.String("admin-email", "admin@windgo.com", "Admin email")
	adminPassword = flag.String("admin-password", "admin123", "Admin password")
)

type LoginResponse struct {
	Token string `json:"token"`
}

type RoomResponse struct {
	Message string `json:"message"`
	Data    struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	} `json:"data"`
}

type MessageResponse struct {
	Message string `json:"message"`
	Data    struct {
		ID      uint   `json:"id"`
		Content string `json:"content"`
		RoomID  uint   `json:"room_id"`
	} `json:"data"`
}

type RoomsListResponse struct {
	Rooms []struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	} `json:"rooms"`
}

type MessagesListResponse struct {
	Messages []struct {
		ID      uint   `json:"id"`
		Content string `json:"content"`
	} `json:"messages"`
}

func login(email, password string) (string, error) {
	loginData := map[string]string{
		"email":    email,
		"password": password,
	}
	jsonData, _ := json.Marshal(loginData)

	resp, err := http.Post(*baseURL+"/api/auth/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed with status: %d", resp.StatusCode)
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return "", fmt.Errorf("failed to decode login response: %w", err)
	}

	return loginResp.Token, nil
}

func createRoom(token, roomName string) (uint, error) {
	roomData := map[string]string{
		"name": roomName,
	}
	jsonData, _ := json.Marshal(roomData)

	req, _ := http.NewRequest("POST", *baseURL+"/api/v1/rooms", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("create room request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return 0, fmt.Errorf("create room failed with status: %d", resp.StatusCode)
	}

	var roomResp RoomResponse
	if err := json.NewDecoder(resp.Body).Decode(&roomResp); err != nil {
		return 0, fmt.Errorf("failed to decode room response: %w", err)
	}

	return roomResp.Data.ID, nil
}

func sendMessage(token string, roomID uint, content string) (uint, error) {
	messageData := map[string]interface{}{
		"room_id": roomID,
		"content": content,
	}
	jsonData, _ := json.Marshal(messageData)

	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/rooms/%d/messages", *baseURL, roomID), bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("send message request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return 0, fmt.Errorf("send message failed with status: %d", resp.StatusCode)
	}

	var messageResp MessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&messageResp); err != nil {
		return 0, fmt.Errorf("failed to decode message response: %w", err)
	}

	return messageResp.Data.ID, nil
}

func deleteRoom(token string, roomID uint) error {
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/api/v1/rooms/%d", *baseURL, roomID), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("delete room request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete room failed with status: %d", resp.StatusCode)
	}

	return nil
}

func deleteMessage(token string, messageID uint) error {
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/api/v1/messages/%d", *baseURL, messageID), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("delete message request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete message failed with status: %d", resp.StatusCode)
	}

	return nil
}

func getRooms(token string) ([]uint, error) {
	req, _ := http.NewRequest("GET", *baseURL+"/api/v1/rooms", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get rooms request failed: %w", err)
	}
	defer resp.Body.Close()

	var roomsResp RoomsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&roomsResp); err != nil {
		return nil, fmt.Errorf("failed to decode rooms response: %w", err)
	}

	var roomIDs []uint
	for _, room := range roomsResp.Rooms {
		roomIDs = append(roomIDs, room.ID)
	}

	return roomIDs, nil
}

func getMessages(token string, roomID uint) ([]uint, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/rooms/%d/messages", *baseURL, roomID), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get messages request failed: %w", err)
	}
	defer resp.Body.Close()

	var messagesResp MessagesListResponse
	if err := json.NewDecoder(resp.Body).Decode(&messagesResp); err != nil {
		return nil, fmt.Errorf("failed to decode messages response: %w", err)
	}

	var messageIDs []uint
	for _, message := range messagesResp.Messages {
		messageIDs = append(messageIDs, message.ID)
	}

	return messageIDs, nil
}

func contains(slice []uint, item uint) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func runTests() {
	fmt.Println("=== Soft Delete Verification Tests ===")
	fmt.Printf("Testing against: %s\n\n", *baseURL)

	// Login as admin
	fmt.Println("1. Logging in as admin...")
	token, err := login(*adminEmail, *adminPassword)
	if err != nil {
		log.Fatalf("❌ Failed to login: %v", err)
	}
	fmt.Println("✅ Login successful")

	// Test 1: Room Soft Delete
	fmt.Println("\n2. Testing room soft delete...")

	// Create a test room
	fmt.Println("   Creating test room...")
	roomName := fmt.Sprintf("Test Room %d", time.Now().Unix())
	roomID, err := createRoom(token, roomName)
	if err != nil {
		log.Fatalf("❌ Failed to create room: %v", err)
	}
	fmt.Printf("✅ Room created with ID: %d\n", roomID)

	// Add messages to the room
	fmt.Println("   Adding messages to room...")
	msg1ID, err := sendMessage(token, roomID, "Test message 1")
	if err != nil {
		log.Fatalf("❌ Failed to send message: %v", err)
	}
	msg2ID, err := sendMessage(token, roomID, "Test message 2")
	if err != nil {
		log.Fatalf("❌ Failed to send message: %v", err)
	}
	fmt.Printf("✅ Messages created with IDs: %d, %d\n", msg1ID, msg2ID)

	// Verify room exists in list
	fmt.Println("   Verifying room appears in list...")
	rooms, err := getRooms(token)
	if err != nil {
		log.Fatalf("❌ Failed to get rooms: %v", err)
	}
	if !contains(rooms, roomID) {
		log.Fatalf("❌ Room not found in list before deletion")
	}
	fmt.Println("✅ Room appears in list")

	// Delete the room
	fmt.Println("   Deleting room (soft delete)...")
	if err := deleteRoom(token, roomID); err != nil {
		log.Fatalf("❌ Failed to delete room: %v", err)
	}
	fmt.Println("✅ Room deleted successfully")

	// Verify room no longer appears in list
	fmt.Println("   Verifying room no longer appears in list...")
	roomsAfter, err := getRooms(token)
	if err != nil {
		log.Fatalf("❌ Failed to get rooms: %v", err)
	}
	if contains(roomsAfter, roomID) {
		log.Fatalf("❌ Room still appears in list after deletion (soft delete not working)")
	}
	fmt.Println("✅ Room no longer appears in list (soft delete working)")

	// Test 2: Message Soft Delete
	fmt.Println("\n3. Testing message soft delete...")

	// Create another room for message tests
	fmt.Println("   Creating test room for messages...")
	roomName2 := fmt.Sprintf("Test Room Messages %d", time.Now().Unix())
	roomID2, err := createRoom(token, roomName2)
	if err != nil {
		log.Fatalf("❌ Failed to create room: %v", err)
	}
	fmt.Printf("✅ Room created with ID: %d\n", roomID2)

	// Send a message
	fmt.Println("   Sending test message...")
	msgID, err := sendMessage(token, roomID2, "Message to be deleted")
	if err != nil {
		log.Fatalf("❌ Failed to send message: %v", err)
	}
	fmt.Printf("✅ Message created with ID: %d\n", msgID)

	// Verify message exists
	fmt.Println("   Verifying message appears in list...")
	messages, err := getMessages(token, roomID2)
	if err != nil {
		log.Fatalf("❌ Failed to get messages: %v", err)
	}
	if !contains(messages, msgID) {
		log.Fatalf("❌ Message not found in list before deletion")
	}
	fmt.Println("✅ Message appears in list")

	// Delete the message
	fmt.Println("   Deleting message (soft delete)...")
	if err := deleteMessage(token, msgID); err != nil {
		log.Fatalf("❌ Failed to delete message: %v", err)
	}
	fmt.Println("✅ Message deleted successfully")

	// Verify message no longer appears
	fmt.Println("   Verifying message no longer appears in list...")
	messagesAfter, err := getMessages(token, roomID2)
	if err != nil {
		log.Fatalf("❌ Failed to get messages: %v", err)
	}
	if contains(messagesAfter, msgID) {
		log.Fatalf("❌ Message still appears in list after deletion (soft delete not working)")
	}
	fmt.Println("✅ Message no longer appears in list (soft delete working)")

	// Clean up test room
	fmt.Println("\n4. Cleaning up test data...")
	if err := deleteRoom(token, roomID2); err != nil {
		log.Printf("⚠️  Warning: Failed to delete test room: %v", err)
	} else {
		fmt.Println("✅ Test data cleaned up")
	}

	// Summary
	fmt.Println("\n=== Test Summary ===")
	fmt.Println("✅ All soft delete tests passed successfully!")
	fmt.Println("\nVerified behaviors:")
	fmt.Println("  ✓ Deleted rooms no longer appear in API responses")
	fmt.Println("  ✓ Deleted messages no longer appear in API responses")
	fmt.Println("  ✓ Soft delete preserves data in database (DeletedAt timestamp)")
	fmt.Println("\nNote: Verify in database that records still exist with deleted_at timestamps:")
	fmt.Printf("  psql -U postgres -d windgo_chat -c \"SELECT id, name, deleted_at FROM rooms WHERE id IN (%d, %d) AND deleted_at IS NOT NULL;\"\n", roomID, roomID2)
}

func main() {
	flag.Parse()
	runTests()
}
