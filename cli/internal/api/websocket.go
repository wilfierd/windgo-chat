package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type    string      `json:"type"`
	RoomID  uint        `json:"room_id"`
	UserID  uint        `json:"user_id"`
	Content interface{} `json:"content"`
}

// WSClient manages the WebSocket connection
type WSClient struct {
	conn             *websocket.Conn
	token            string
	baseURL          string
	connected        bool
	mu               sync.RWMutex
	messageHandler   func(WSMessage)
	errorHandler     func(error)
	reconnectDelay   time.Duration
	stopChan         chan struct{}
	writeChan        chan []byte
	msgChan          chan WSMessage // Channel for subscribers
	autoReconnect    bool
	reconnecting     bool
	manualDisconnect bool
	currentRooms     map[uint]bool // Track joined rooms for reconnection
	currentRoomsMu   sync.RWMutex
}

// NewWSClient creates a new WebSocket client
func NewWSClient(baseURL, token string) *WSClient {
	return &WSClient{
		token:            token,
		baseURL:          baseURL,
		connected:        false,
		reconnectDelay:   5 * time.Second,
		stopChan:         make(chan struct{}),
		writeChan:        make(chan []byte, 256),
		msgChan:          make(chan WSMessage, 256),
		autoReconnect:    true,
		reconnecting:     false,
		manualDisconnect: false,
		currentRooms:     make(map[uint]bool),
	}
}

// SetMessageHandler sets the callback for incoming messages
func (ws *WSClient) SetMessageHandler(handler func(WSMessage)) {
	ws.messageHandler = handler
}

// SetAutoReconnect enables or disables automatic reconnection
func (ws *WSClient) SetAutoReconnect(enabled bool) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.autoReconnect = enabled
}

// Subscribe returns a channel to receive WebSocket messages
// This is useful for integrating with event loops like Bubble Tea
func (ws *WSClient) Subscribe() <-chan WSMessage {
	return ws.msgChan
}

// SetErrorHandler sets the callback for errors
func (ws *WSClient) SetErrorHandler(handler func(error)) {
	ws.errorHandler = handler
}

// Connect establishes the WebSocket connection
func (ws *WSClient) Connect() error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.connected {
		return fmt.Errorf("already connected")
	}

	// Parse base URL and convert to WebSocket URL
	u, err := url.Parse(ws.baseURL)
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}

	// Convert http/https to ws/wss
	scheme := "ws"
	if u.Scheme == "https" {
		scheme = "wss"
	}

	// Build WebSocket URL with token in query parameter
	wsURL := fmt.Sprintf("%s://%s/ws?token=%s", scheme, u.Host, url.QueryEscape(ws.token))

	// Connect to WebSocket
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	ws.conn = conn
	ws.connected = true

	// Start read and write pumps
	go ws.readPump()
	go ws.writePump()

	log.Println("WebSocket connected successfully")
	return nil
}

// Disconnect closes the WebSocket connection
func (ws *WSClient) Disconnect() error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if !ws.connected {
		return nil
	}

	ws.manualDisconnect = true
	close(ws.stopChan)
	ws.connected = false

	if ws.conn != nil {
		// Send close message
		ws.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		ws.conn.Close()
		ws.conn = nil
	}

	// Close message channel
	if ws.msgChan != nil {
		close(ws.msgChan)
		ws.msgChan = make(chan WSMessage, 256) // Recreate for potential reconnection
	}

	log.Println("WebSocket disconnected")
	return nil
}

// IsConnected returns whether the WebSocket is currently connected
func (ws *WSClient) IsConnected() bool {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.connected
}

// readPump reads messages from the WebSocket connection
func (ws *WSClient) readPump() {
	defer func() {
		ws.mu.Lock()
		wasConnected := ws.connected
		ws.connected = false
		if ws.conn != nil {
			ws.conn.Close()
		}
		shouldReconnect := ws.autoReconnect && !ws.manualDisconnect && wasConnected
		ws.mu.Unlock()

		// Attempt to reconnect if auto-reconnect is enabled
		if shouldReconnect {
			go ws.reconnect()
		}
	}()

	for {
		select {
		case <-ws.stopChan:
			return
		default:
			_, message, err := ws.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket read error: %v", err)
				}
				if ws.errorHandler != nil {
					ws.errorHandler(err)
				}
				return
			}

			// Parse message
			var msg WSMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Printf("Failed to parse WebSocket message: %v", err)
				continue
			}

			// Handle message via callback
			if ws.messageHandler != nil {
				ws.messageHandler(msg)
			}

			// Also send to channel for subscribers
			select {
			case ws.msgChan <- msg:
			default:
				// Channel full, drop message
				log.Printf("Message channel full, dropping message")
			}
		}
	}
}

// writePump writes messages to the WebSocket connection
func (ws *WSClient) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		ws.mu.Lock()
		wasConnected := ws.connected
		if ws.conn != nil {
			ws.conn.Close()
		}
		shouldReconnect := ws.autoReconnect && !ws.manualDisconnect && wasConnected
		ws.mu.Unlock()

		// Attempt to reconnect if auto-reconnect is enabled
		if shouldReconnect {
			go ws.reconnect()
		}
	}()

	for {
		select {
		case <-ws.stopChan:
			return
		case message := <-ws.writeChan:
			ws.mu.RLock()
			conn := ws.conn
			ws.mu.RUnlock()

			if conn == nil {
				continue
			}

			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}
		case <-ticker.C:
			// Send ping
			ws.mu.RLock()
			conn := ws.conn
			ws.mu.RUnlock()

			if conn == nil {
				continue
			}

			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// reconnect attempts to reconnect to the WebSocket server
func (ws *WSClient) reconnect() {
	ws.mu.Lock()
	if ws.reconnecting || ws.manualDisconnect {
		ws.mu.Unlock()
		return
	}
	ws.reconnecting = true
	ws.mu.Unlock()

	defer func() {
		ws.mu.Lock()
		ws.reconnecting = false
		ws.mu.Unlock()
	}()

	log.Println("WebSocket connection lost, attempting to reconnect...")

	// Exponential backoff for reconnection attempts
	delay := ws.reconnectDelay
	maxDelay := 60 * time.Second
	attempts := 0

	for {
		ws.mu.RLock()
		shouldStop := ws.manualDisconnect
		ws.mu.RUnlock()

		if shouldStop {
			log.Println("Reconnection cancelled - manual disconnect")
			return
		}

		attempts++
		log.Printf("Reconnection attempt %d (waiting %v)...", attempts, delay)
		time.Sleep(delay)

		// Attempt to reconnect
		ws.mu.Lock()
		ws.stopChan = make(chan struct{})
		ws.writeChan = make(chan []byte, 256)
		ws.mu.Unlock()

		if err := ws.Connect(); err != nil {
			log.Printf("Reconnection attempt %d failed: %v", attempts, err)
			// Exponential backoff with max delay
			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
			continue
		}

		log.Println("WebSocket reconnected successfully")

		// Rejoin all previously joined rooms
		ws.currentRoomsMu.RLock()
		rooms := make([]uint, 0, len(ws.currentRooms))
		for roomID := range ws.currentRooms {
			rooms = append(rooms, roomID)
		}
		ws.currentRoomsMu.RUnlock()

		for _, roomID := range rooms {
			if err := ws.JoinRoom(roomID); err != nil {
				log.Printf("Failed to rejoin room %d: %v", roomID, err)
			} else {
				log.Printf("Rejoined room %d", roomID)
			}
		}

		return
	}
}

// JoinRoom sends a join room message
func (ws *WSClient) JoinRoom(roomID uint) error {
	if !ws.IsConnected() {
		return fmt.Errorf("not connected")
	}

	msg := WSMessage{
		Type:    "join",
		RoomID:  roomID,
		Content: map[string]interface{}{"room_id": roomID},
	}

	// Track joined room for reconnection
	ws.currentRoomsMu.Lock()
	ws.currentRooms[roomID] = true
	ws.currentRoomsMu.Unlock()

	return ws.sendMessage(msg)
}

// LeaveRoom sends a leave room message
func (ws *WSClient) LeaveRoom(roomID uint) error {
	if !ws.IsConnected() {
		return fmt.Errorf("not connected")
	}

	msg := WSMessage{
		Type:    "leave",
		RoomID:  roomID,
		Content: map[string]interface{}{"room_id": roomID},
	}

	// Remove from tracked rooms
	ws.currentRoomsMu.Lock()
	delete(ws.currentRooms, roomID)
	ws.currentRoomsMu.Unlock()

	return ws.sendMessage(msg)
}

// SendTyping sends a typing indicator
func (ws *WSClient) SendTyping(roomID uint, isTyping bool) error {
	if !ws.IsConnected() {
		return fmt.Errorf("not connected")
	}

	msg := WSMessage{
		Type:    "typing",
		RoomID:  roomID,
		Content: map[string]interface{}{"typing": isTyping},
	}

	return ws.sendMessage(msg)
}

// sendMessage sends a message through the WebSocket
func (ws *WSClient) sendMessage(msg WSMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	select {
	case ws.writeChan <- data:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("send timeout")
	}
}
