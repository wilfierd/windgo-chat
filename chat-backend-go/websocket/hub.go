// Package websocket provides real-time communication infrastructure for the chat application.
// This file contains the Hub implementation that manages all WebSocket connections and message broadcasting.
package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gofiber/websocket/v2"
	"gorm.io/gorm"
)

// Client represents a connected WebSocket client
type Client struct {
	ID     string          // Unique client identifier (user ID)
	UserID uint            // User ID from database
	Conn   *websocket.Conn // WebSocket connection
	Send   chan []byte     // Buffered channel for outbound messages
	Hub    *Hub            // Reference to the hub
	Rooms  map[uint]bool   // Rooms this client is subscribed to
	mu     sync.RWMutex    // Mutex for thread-safe room operations
}

// Message represents a WebSocket message
type Message struct {
	Type    string      `json:"type"`    // Message type: "message", "typing", "join", "leave", etc.
	RoomID  uint        `json:"room_id"` // Room ID
	UserID  uint        `json:"user_id"` // Sender user ID
	Content interface{} `json:"content"` // Message content (flexible type)
}

// Hub maintains the set of active clients and broadcasts messages to clients
type Hub struct {
	// Registered clients mapped by user ID
	clients map[string]*Client

	// Clients organized by room ID
	rooms map[uint]map[string]*Client

	// Inbound messages from clients
	broadcast chan *Message

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Database connection for membership verification
	db *gorm.DB

	// Cache of room memberships: roomID -> set of userIDs
	membershipCache map[uint]map[uint]bool

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

// NewHub creates a new Hub instance
func NewHub(db *gorm.DB) *Hub {
	return &Hub{
		clients:         make(map[string]*Client),
		rooms:           make(map[uint]map[string]*Client),
		broadcast:       make(chan *Message, 256),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		db:              db,
		membershipCache: make(map[uint]map[uint]bool),
	}
}

// Run starts the hub's main loop to handle registration, unregistration, and broadcasting
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// Register adds a client to the hub
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// registerClient adds a client to the hub
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client.ID] = client
	log.Printf("Client registered: %s (User ID: %d)", client.ID, client.UserID)
}

// unregisterClient removes a client from the hub and all rooms
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client.ID]; ok {
		// Remove from all rooms
		client.mu.RLock()
		for roomID := range client.Rooms {
			if roomClients, exists := h.rooms[roomID]; exists {
				delete(roomClients, client.ID)
				if len(roomClients) == 0 {
					delete(h.rooms, roomID)
				}
			}
		}
		client.mu.RUnlock()

		// Close send channel and remove from clients map
		close(client.Send)
		delete(h.clients, client.ID)
		log.Printf("Client unregistered: %s (User ID: %d)", client.ID, client.UserID)
	}
}

// broadcastMessage sends a message to all clients in a specific room
// Only broadcasts to clients who are verified room members
func (h *Hub) broadcastMessage(message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	roomClients, exists := h.rooms[message.RoomID]
	if !exists {
		return
	}

	// Get room members from cache or database
	members := h.getRoomMembers(message.RoomID)
	if len(members) == 0 {
		log.Printf("No members found for room %d, skipping broadcast", message.RoomID)
		return
	}

	// Marshal message to JSON once
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	// Send only to clients who are room members
	for _, client := range roomClients {
		// Verify client is a room member
		if !members[client.UserID] {
			log.Printf("Client %s (User %d) is not a member of room %d, skipping", client.ID, client.UserID, message.RoomID)
			continue
		}

		select {
		case client.Send <- data:
		default:
			// Client's send buffer is full, close and unregister
			log.Printf("Client send buffer full, closing: %s", client.ID)
			go func(c *Client) {
				h.unregister <- c
			}(client)
		}
	}
}

// JoinRoom adds a client to a room after verifying membership
func (h *Hub) JoinRoom(client *Client, roomID uint) bool {
	// Verify user is a member of the room
	if !h.isRoomMember(client.UserID, roomID) {
		log.Printf("Client %s (User %d) denied access to room %d: not a member", client.ID, client.UserID, roomID)
		return false
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Add room to client's room list
	client.mu.Lock()
	client.Rooms[roomID] = true
	client.mu.Unlock()

	// Add client to room's client list
	if _, exists := h.rooms[roomID]; !exists {
		h.rooms[roomID] = make(map[string]*Client)
	}
	h.rooms[roomID][client.ID] = client

	log.Printf("Client %s joined room %d", client.ID, roomID)
	return true
}

// LeaveRoom removes a client from a room
func (h *Hub) LeaveRoom(client *Client, roomID uint) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Remove room from client's room list
	client.mu.Lock()
	delete(client.Rooms, roomID)
	client.mu.Unlock()

	// Remove client from room's client list
	if roomClients, exists := h.rooms[roomID]; exists {
		delete(roomClients, client.ID)
		if len(roomClients) == 0 {
			delete(h.rooms, roomID)
		}
	}

	log.Printf("Client %s left room %d", client.ID, roomID)
}

// BroadcastToRoom sends a message to all clients in a room
func (h *Hub) BroadcastToRoom(message *Message) {
	h.broadcast <- message
}

// GetRoomClientCount returns the number of clients in a room
func (h *Hub) GetRoomClientCount(roomID uint) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if roomClients, exists := h.rooms[roomID]; exists {
		return len(roomClients)
	}
	return 0
}

// GetOnlineUsers returns a list of online user IDs
func (h *Hub) GetOnlineUsers() []uint {
	h.mu.RLock()
	defer h.mu.RUnlock()

	users := make(map[uint]bool)
	for _, client := range h.clients {
		users[client.UserID] = true
	}

	result := make([]uint, 0, len(users))
	for userID := range users {
		result = append(result, userID)
	}
	return result
}

// isRoomMember checks if a user is a member of a room
func (h *Hub) isRoomMember(userID uint, roomID uint) bool {
	members := h.getRoomMembers(roomID)
	return members[userID]
}

// getRoomMembers returns all member user IDs for a room
func (h *Hub) getRoomMembers(roomID uint) map[uint]bool {
	// Check cache first
	h.mu.RLock()
	if members, exists := h.membershipCache[roomID]; exists {
		h.mu.RUnlock()
		return members
	}
	h.mu.RUnlock()

	// Query database if not in cache
	if h.db == nil {
		log.Printf("Database not available for getting room members")
		return make(map[uint]bool)
	}

	var userIDs []uint
	err := h.db.Table("room_memberships").
		Where("room_id = ?", roomID).
		Pluck("user_id", &userIDs).Error

	if err != nil {
		log.Printf("Error getting room members: %v", err)
		return make(map[uint]bool)
	}

	// Build member set
	members := make(map[uint]bool)
	for _, id := range userIDs {
		members[id] = true
	}

	// Update cache
	h.mu.Lock()
	h.membershipCache[roomID] = members
	h.mu.Unlock()

	return members
}

// InvalidateMembershipCache clears the membership cache for a room
// Should be called when room memberships change
func (h *Hub) InvalidateMembershipCache(roomID uint) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.membershipCache, roomID)
	log.Printf("Membership cache invalidated for room %d", roomID)
}

// InvalidateAllMembershipCache clears the entire membership cache
func (h *Hub) InvalidateAllMembershipCache() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.membershipCache = make(map[uint]map[uint]bool)
	log.Printf("All membership cache invalidated")
}

// IsRoomMember checks if a user is a member of a room (public method)
func (h *Hub) IsRoomMember(userID uint, roomID uint) bool {
	return h.isRoomMember(userID, roomID)
}

// SendToUser sends a message to a specific user across all their connected clients
func (h *Hub) SendToUser(userID uint, message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Marshal message to JSON once
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message for user %d: %v", userID, err)
		return
	}

	// Find all clients for this user and send the message
	sentCount := 0
	for _, client := range h.clients {
		if client.UserID == userID {
			select {
			case client.Send <- data:
				sentCount++
			default:
				// Client's send buffer is full, log but continue
				log.Printf("Client %s send buffer full, dropping mention notification", client.ID)
			}
		}
	}

	if sentCount > 0 {
		log.Printf("Sent mention notification to user %d (%d clients)", userID, sentCount)
	} else {
		log.Printf("User %d has no connected clients, mention notification not sent", userID)
	}
}
