package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	numClients  = flag.Int("clients", 10, "Number of concurrent WebSocket clients")
	duration    = flag.Duration("duration", 60*time.Second, "Test duration")
	serverURL   = flag.String("url", "ws://localhost:8080/ws", "WebSocket server URL")
	token       = flag.String("token", "", "JWT token for authentication")
	roomID      = flag.Int("room", 1, "Room ID to join")
	messageRate = flag.Duration("rate", 5*time.Second, "Message sending rate per client")
	verbose     = flag.Bool("verbose", false, "Enable verbose logging")
)

type WebSocketClient struct {
	ID           int
	conn         *websocket.Conn
	url          string
	token        string
	roomID       int
	messagesSent int
	messagesRecv int
	errors       int
	mu           sync.Mutex
}

type Message struct {
	Type    string                 `json:"type"`
	RoomID  int                    `json:"room_id"`
	Content map[string]interface{} `json:"content"`
}

func NewWebSocketClient(id int, serverURL, token string, roomID int) *WebSocketClient {
	return &WebSocketClient{
		ID:     id,
		url:    serverURL,
		token:  token,
		roomID: roomID,
	}
}

func (c *WebSocketClient) Connect() error {
	u, err := url.Parse(c.url)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	// Add token as query parameter
	q := u.Query()
	q.Set("token", c.token)
	u.RawQuery = q.Encode()

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.conn = conn
	if *verbose {
		log.Printf("Client #%d: Connected to %s", c.ID, u.String())
	}

	return nil
}

func (c *WebSocketClient) JoinRoom() error {
	msg := Message{
		Type:   "join",
		RoomID: c.roomID,
		Content: map[string]interface{}{
			"room_id": c.roomID,
		},
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		return fmt.Errorf("failed to join room: %w", err)
	}

	if *verbose {
		log.Printf("Client #%d: Joined room %d", c.ID, c.roomID)
	}

	c.messagesSent++
	return nil
}

func (c *WebSocketClient) SendMessage(content string) error {
	msg := Message{
		Type:   "message",
		RoomID: c.roomID,
		Content: map[string]interface{}{
			"content": content,
		},
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		c.mu.Lock()
		c.errors++
		c.mu.Unlock()
		return fmt.Errorf("failed to send message: %w", err)
	}

	c.mu.Lock()
	c.messagesSent++
	c.mu.Unlock()

	if *verbose {
		log.Printf("Client #%d: Sent message '%s'", c.ID, content)
	}

	return nil
}

func (c *WebSocketClient) ReadMessages(done chan bool) {
	defer func() {
		done <- true
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Client #%d: WebSocket error: %v", c.ID, err)
			}
			return
		}

		c.mu.Lock()
		c.messagesRecv++
		c.mu.Unlock()

		if *verbose {
			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err == nil {
				log.Printf("Client #%d: Received message: %v", c.ID, msg)
			}
		}
	}
}

func (c *WebSocketClient) SendHeartbeat() error {
	msg := Message{
		Type:    "ping",
		RoomID:  c.roomID,
		Content: map[string]interface{}{},
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		c.mu.Lock()
		c.errors++
		c.mu.Unlock()
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}

	return nil
}

func (c *WebSocketClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *WebSocketClient) GetStats() (sent, recv, errors int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.messagesSent, c.messagesRecv, c.errors
}

func runLoadTest() {
	log.Printf("Starting WebSocket load test with %d clients for %v", *numClients, *duration)
	log.Printf("Server URL: %s", *serverURL)
	log.Printf("Room ID: %d", *roomID)
	log.Printf("Message rate: %v per client", *messageRate)

	var wg sync.WaitGroup
	clients := make([]*WebSocketClient, *numClients)
	stopChan := make(chan bool)

	// Setup interrupt handler
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Start clients
	for i := 0; i < *numClients; i++ {
		client := NewWebSocketClient(i+1, *serverURL, *token, *roomID)
		clients[i] = client

		// Connect
		if err := client.Connect(); err != nil {
			log.Printf("Failed to connect client #%d: %v", i+1, err)
			continue
		}

		// Join room
		if err := client.JoinRoom(); err != nil {
			log.Printf("Failed to join room for client #%d: %v", i+1, err)
			client.Close()
			continue
		}

		wg.Add(1)

		// Start message reader
		go func(c *WebSocketClient) {
			done := make(chan bool)
			go c.ReadMessages(done)

			// Send messages periodically
			ticker := time.NewTicker(*messageRate)
			heartbeat := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			defer heartbeat.Stop()

			for {
				select {
				case <-stopChan:
					c.Close()
					wg.Done()
					return
				case <-ticker.C:
					msg := fmt.Sprintf("Test message from client #%d at %s", c.ID, time.Now().Format(time.RFC3339))
					if err := c.SendMessage(msg); err != nil {
						log.Printf("Client #%d: Error sending message: %v", c.ID, err)
					}
				case <-heartbeat.C:
					if err := c.SendHeartbeat(); err != nil {
						log.Printf("Client #%d: Error sending heartbeat: %v", c.ID, err)
					}
				case <-done:
					wg.Done()
					return
				}
			}
		}(client)
	}

	// Wait for duration or interrupt
	timer := time.NewTimer(*duration)
	select {
	case <-timer.C:
		log.Println("\nTest duration completed. Shutting down...")
	case <-interrupt:
		log.Println("\nInterrupt signal received. Shutting down...")
	}

	// Stop all clients
	close(stopChan)
	wg.Wait()

	// Print statistics
	totalSent := 0
	totalRecv := 0
	totalErrors := 0

	fmt.Println("\n=== Load Test Results ===")
	fmt.Printf("Duration: %v\n", *duration)
	fmt.Printf("Number of clients: %d\n\n", *numClients)

	for i, client := range clients {
		sent, recv, errors := client.GetStats()
		totalSent += sent
		totalRecv += recv
		totalErrors += errors

		if *verbose {
			fmt.Printf("Client #%d: Sent=%d, Received=%d, Errors=%d\n", i+1, sent, recv, errors)
		}
	}

	fmt.Printf("\nTotal Messages Sent: %d\n", totalSent)
	fmt.Printf("Total Messages Received: %d\n", totalRecv)
	fmt.Printf("Total Errors: %d\n", totalErrors)
	fmt.Printf("Messages/sec (sent): %.2f\n", float64(totalSent)/duration.Seconds())
	fmt.Printf("Messages/sec (received): %.2f\n", float64(totalRecv)/duration.Seconds())

	if totalErrors > 0 {
		fmt.Printf("\n⚠️  WARNING: %d errors occurred during the test\n", totalErrors)
	} else {
		fmt.Println("\n✅ All operations completed successfully!")
	}
}

func main() {
	flag.Parse()

	if *token == "" {
		log.Fatal("JWT token is required. Use -token flag to provide it.")
	}

	runLoadTest()
}
