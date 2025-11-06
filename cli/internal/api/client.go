package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// Client knows how to talk to the WindGo backend API.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient constructs a client using WINDGO_BASE_URL or the local default.
func NewClient() *Client {
	base := os.Getenv("WINDGO_BASE_URL")
	if base == "" {
		base = "http://localhost:8080"
	}
	base = strings.TrimRight(base, "/")
	return &Client{
		BaseURL: base,
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// AuthResponse mirrors the backend login payload.
type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// User is a trimmed down view for the CLI.
type User struct {
	ID           uint       `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	Role         string     `json:"role"`
	Provider     string     `json:"provider"`
	GitHubID     string     `json:"github_id"`
	AvatarURL    string     `json:"avatar_url"`
	LastActiveAt *time.Time `json:"last_active_at"` // NEW: Track user activity
	IsOnline     bool       `json:"is_online"`      // NEW: Online status
	Status       string     `json:"status"`         // NEW: User status (online/away/busy/offline)
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Room represents a chat room from the API.
type Room struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"` // "direct" or "group"
	UnreadCount int64     `json:"unread_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DirectRoom represents a direct message conversation from the API.
type DirectRoom struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	OtherUser   User      `json:"other_user"`
	LastMessage *Message  `json:"last_message,omitempty"`
	UnreadCount int       `json:"unread_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Message represents a chat message from the API.
type Message struct {
	ID            uint      `json:"id"`
	UserID        uint      `json:"user_id"`
	RoomID        uint      `json:"room_id"`
	Content       string    `json:"content"`
	User          User      `json:"user"`
	ParentID      *uint     `json:"parent_id,omitempty"`
	ParentMessage *Message  `json:"parent_message,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// DeviceStartResponse is returned when initiating a GitHub device flow.
type DeviceStartResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// APIError captures {"error":"..."} replies.
type APIError struct {
	Error string `json:"error"`
}

// handleErrorResponse processes error responses from the API
func (c *Client) handleErrorResponse(resp *http.Response) error {
	if resp.StatusCode < 400 {
		return nil
	}

	var apiErr APIError
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil || apiErr.Error == "" {
		return fmt.Errorf("api error: %s", resp.Status)
	}
	return errors.New(apiErr.Error)
}

func (c *Client) postJSON(path string, reqBody any, v any) error {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := c.handleErrorResponse(resp); err != nil {
		return err
	}

	if v != nil {
		return json.NewDecoder(resp.Body).Decode(v)
	}
	return nil
}

// Login performs email/password authentication.
func (c *Client) Login(email, password string) (*AuthResponse, error) {
	var resp AuthResponse
	err := c.postJSON("/api/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// StartDeviceFlow kicks off the GitHub OAuth device flow.
func (c *Client) StartDeviceFlow() (*DeviceStartResponse, error) {
	var resp DeviceStartResponse
	err := c.postJSON("/api/auth/github/device/start", map[string]any{}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// PollDevice waits for the GitHub device flow to complete.
func (c *Client) PollDevice(deviceCode string, timeoutSeconds int) (*AuthResponse, error) {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 90
	}
	var resp AuthResponse
	err := c.postJSON("/api/auth/github/device/poll", map[string]any{
		"device_code": deviceCode,
		"timeout":     timeoutSeconds,
		"interval":    5,
	}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Profile fetches the authenticated user using a bearer token.
func (c *Client) Profile(token string) (*User, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/api/auth/profile", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.handleErrorResponse(resp); err != nil {
		return nil, err
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

// GetRooms fetches the list of available chat rooms using a bearer token.
func (c *Client) GetRooms(token string) ([]Room, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/api/v1/rooms", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.handleErrorResponse(resp); err != nil {
		return nil, err
	}

	var response struct {
		Rooms []Room `json:"rooms"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response.Rooms, nil
}

// CreateRoom creates a new chat room (admin only)
func (c *Client) CreateRoom(token, name string) (*Room, error) {
	payload := map[string]string{
		"name": name,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/api/v1/rooms", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil || apiErr.Error == "" {
			return nil, fmt.Errorf("api error: %s", resp.Status)
		}
		return nil, errors.New(apiErr.Error)
	}

	var response struct {
		Message string `json:"message"`
		Room    Room   `json:"room"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return &response.Room, nil
}

// UpdateRoom updates an existing chat room (admin only)
func (c *Client) UpdateRoom(token string, roomID uint, name string) (*Room, error) {
	payload := map[string]string{
		"name": name,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/v1/rooms/%d", c.BaseURL, roomID)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.handleErrorResponse(resp); err != nil {
		return nil, err
	}

	var response struct {
		Message string `json:"message"`
		Room    Room   `json:"room"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return &response.Room, nil
}

// DeleteRoom deletes a chat room (admin only)
func (c *Client) DeleteRoom(token string, roomID uint) error {
	url := fmt.Sprintf("%s/api/v1/rooms/%d", c.BaseURL, roomID)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.handleErrorResponse(resp)
}

// GetUsers fetches the list of available users using a bearer token.
// Optionally filters by search query.
func (c *Client) GetUsers(token, search string) ([]User, error) {
	url := c.BaseURL + "/api/v1/users"
	if search != "" {
		url += "?search=" + search
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.handleErrorResponse(resp); err != nil {
		return nil, err
	}

	var response struct {
		Users []User `json:"users"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response.Users, nil
}

// GetMessages fetches messages for a specific room using a bearer token.
// Supports pagination with page and limit parameters.
func (c *Client) GetMessages(token string, roomID uint, page, limit int) ([]Message, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	url := fmt.Sprintf("%s/api/v1/rooms/%d/messages?page=%d&limit=%d", c.BaseURL, roomID, page, limit)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.handleErrorResponse(resp); err != nil {
		return nil, err
	}

	var response struct {
		Messages []Message `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response.Messages, nil
}

// SendMessage sends a new message to a room using a bearer token.
func (c *Client) SendMessage(token string, roomID uint, content string, parentID *uint) (*Message, error) {
	reqBody := map[string]any{
		"room_id": roomID,
		"content": content,
	}

	if parentID != nil {
		reqBody["parent_id"] = *parentID
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/api/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.handleErrorResponse(resp); err != nil {
		return nil, err
	}

	var response struct {
		Message string  `json:"message"`
		Data    Message `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

// UpdateMessage updates an existing message using a bearer token.
func (c *Client) UpdateMessage(token string, messageID uint, content string) (*Message, error) {
	reqBody := map[string]any{
		"content": content,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/v1/messages/%d", c.BaseURL, messageID)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.handleErrorResponse(resp); err != nil {
		return nil, err
	}

	var response struct {
		Message string  `json:"message"`
		Data    Message `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

// DeleteMessage deletes a message using a bearer token.
func (c *Client) DeleteMessage(token string, messageID uint) error {
	url := fmt.Sprintf("%s/api/v1/messages/%d", c.BaseURL, messageID)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.handleErrorResponse(resp)
}

// GetDirectRooms fetches the list of direct message conversations using a bearer token.
func (c *Client) GetDirectRooms(token string) ([]DirectRoom, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/api/v1/rooms/direct", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.handleErrorResponse(resp); err != nil {
		return nil, err
	}

	var response struct {
		Success bool         `json:"success"`
		Message string       `json:"message"`
		Data    []DirectRoom `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response.Data, nil
}

// CreateDirectRoom creates or retrieves a direct message room with a target user.
func (c *Client) CreateDirectRoom(token string, targetUserID uint) (*DirectRoom, error) {
	payload := map[string]uint{
		"target_user_id": targetUserID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/api/v1/rooms/direct", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.handleErrorResponse(resp); err != nil {
		return nil, err
	}

	var response struct {
		Success bool       `json:"success"`
		Message string     `json:"message"`
		Data    DirectRoom `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

// AvailableUser represents a user available for starting a DM.
type AvailableUser struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	IsOnline bool   `json:"is_online"`
	HasDM    bool   `json:"has_dm"`
}

// GetAvailableUsers fetches the list of users available for starting a DM.
func (c *Client) GetAvailableUsers(token string) ([]AvailableUser, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/api/v1/users/available", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.handleErrorResponse(resp); err != nil {
		return nil, err
	}

	var response struct {
		Success bool            `json:"success"`
		Message string          `json:"message"`
		Data    []AvailableUser `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response.Data, nil
}

// MarkRoomAsRead marks all messages in a room as read.
func (c *Client) MarkRoomAsRead(token string, roomID uint) error {
	url := fmt.Sprintf("%s/api/v1/rooms/%d/read", c.BaseURL, roomID)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := c.handleErrorResponse(resp); err != nil {
		return err
	}

	return nil
}
