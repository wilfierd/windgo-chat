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
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Message represents a chat message from the API.
type Message struct {
	ID        uint      `json:"id"`
	Content   string    `json:"content"`
	UserID    uint      `json:"user_id"`
	RoomID    uint      `json:"room_id"`
	User      User      `json:"user"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Pagination holds the pagination data from the API.
type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"totalPages"`
}

// MessagesResponse is the expected response for getting messages.
type MessagesResponse struct {
	Messages   []Message  `json:"messages"`
	Pagination Pagination `json:"pagination"`
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

func (c *Client) postJSON(path string, reqBody any, v any, token ...string) error {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if len(token) > 0 {
		req.Header.Set("Authorization", "Bearer "+token[0])
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil || apiErr.Error == "" {
			return fmt.Errorf("api error: %s", resp.Status)
		}
		return errors.New(apiErr.Error)
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

func (c *Client) getJSON(path, token string, v any) error {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil || apiErr.Error == "" {
			return fmt.Errorf("api error: %s", resp.Status)
		}
		return errors.New(apiErr.Error)
	}

	if v != nil {
		return json.NewDecoder(resp.Body).Decode(v)
	}
	return nil
}

// Profile fetches the authenticated user using a bearer token.
func (c *Client) Profile(token string) (*User, error) {
	var user User
	err := c.getJSON("/api/auth/profile", token, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetRooms fetches the list of available chat rooms using a bearer token.
func (c *Client) GetRooms(token string) ([]Room, error) {
	var response struct {
		Rooms []Room `json:"rooms"`
	}
	err := c.getJSON("/api/v1/rooms", token, &response)
	if err != nil {
		return nil, err
	}
	return response.Rooms, nil
}

// GetUsers fetches the list of available users using a bearer token.
// Optionally filters by search query.
func (c *Client) GetUsers(token, search string) ([]User, error) {
	url := "/api/v1/users"
	if search != "" {
		url += "?search=" + search
	}
	var response struct {
		Users []User `json:"users"`
	}
	err := c.getJSON(url, token, &response)
	if err != nil {
		return nil, err
	}
	return response.Users, nil
}

// GetMessages fetches messages for a room with an auth token.
func (c *Client) GetMessages(token string, roomID uint, page, limit int) (*MessagesResponse, error) {
	path := fmt.Sprintf("/api/v1/rooms/%d/messages?page=%d&limit=%d", roomID, page, limit)
	var resp MessagesResponse
	err := c.getJSON(path, token, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// SendMessage posts a new message to a room.
func (c *Client) SendMessage(token string, roomID uint, content string) (*Message, error) {
	var resp struct {
		Data Message `json:"data"`
	}
	err := c.postJSON("/api/v1/messages", map[string]any{
		"room_id": roomID,
		"content": content,
	}, &resp, token)
	if err != nil {
		return nil, err
	}
	return &resp.Data, nil
}
