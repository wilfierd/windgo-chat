// Package websocket provides real-time communication infrastructure for the chat application.
// This file contains client connection handlers for reading and writing WebSocket messages.
package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gofiber/websocket/v2"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 8192
)

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		messageType, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Unexpected close error from client %s: %v", c.ID, err)
			} else {
				log.Printf("Client %s disconnected", c.ID)
			}
			break
		}

		// Only process text messages
		if messageType != websocket.TextMessage {
			continue
		}

		// Reset read deadline
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))

		// Parse message
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Error parsing message from client %s: %v", c.ID, err)
			continue
		}

		// Handle different message types
		c.handleMessage(&msg)
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := c.Conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Printf("Error writing to client %s: %v", c.ID, err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			// Send a ping frame
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming messages from the client
func (c *Client) handleMessage(msg *Message) {
	// Set the user ID from the authenticated client
	msg.UserID = c.UserID

	switch msg.Type {
	case "join":
		// Client wants to join a room
		if roomID, ok := msg.Content.(float64); ok {
			c.Hub.JoinRoom(c, uint(roomID))
			c.sendAck("joined", msg.RoomID)
			log.Printf("Client %s joined room %d", c.ID, uint(roomID))
		} else if roomMap, ok := msg.Content.(map[string]interface{}); ok {
			if roomID, ok := roomMap["room_id"].(float64); ok {
				c.Hub.JoinRoom(c, uint(roomID))
				c.sendAck("joined", uint(roomID))
				log.Printf("Client %s joined room %d", c.ID, uint(roomID))
			}
		}

	case "leave":
		// Client wants to leave a room
		if roomID, ok := msg.Content.(float64); ok {
			c.Hub.LeaveRoom(c, uint(roomID))
			c.sendAck("left", msg.RoomID)
			log.Printf("Client %s left room %d", c.ID, uint(roomID))
		} else if roomMap, ok := msg.Content.(map[string]interface{}); ok {
			if roomID, ok := roomMap["room_id"].(float64); ok {
				c.Hub.LeaveRoom(c, uint(roomID))
				c.sendAck("left", uint(roomID))
				log.Printf("Client %s left room %d", c.ID, uint(roomID))
			}
		}

	case "typing":
		// Client is typing in a room
		c.Hub.BroadcastToRoom(&Message{
			Type:    "typing",
			RoomID:  msg.RoomID,
			UserID:  c.UserID,
			Content: msg.Content,
		})

	case "ping":
		// Heartbeat ping from client
		c.sendAck("pong", 0)

	default:
		log.Printf("Unknown message type from client %s: %s", c.ID, msg.Type)
	}
}

// sendAck sends an acknowledgment message to the client
func (c *Client) sendAck(ackType string, roomID uint) {
	ack := Message{
		Type:    ackType,
		RoomID:  roomID,
		UserID:  c.UserID,
		Content: map[string]interface{}{"status": "ok"},
	}

	data, err := json.Marshal(ack)
	if err != nil {
		log.Printf("Error marshaling ack: %v", err)
		return
	}

	select {
	case c.Send <- data:
	default:
		log.Printf("Client %s send buffer full, dropping ack", c.ID)
	}
}

// SendMessage sends a message directly to this client
func (c *Client) SendMessage(msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	select {
	case c.Send <- data:
		return nil
	default:
		return ErrSendBufferFull
	}
}

// ErrSendBufferFull is returned when the client's send buffer is full
var ErrSendBufferFull = &ClientError{Message: "send buffer full"}

// ClientError represents a client-specific error
type ClientError struct {
	Message string
}

func (e *ClientError) Error() string {
	return e.Message
}
