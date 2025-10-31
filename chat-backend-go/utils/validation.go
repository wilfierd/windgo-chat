package utils

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Validation constants
const (
	MaxUsernameLength = 50
	MinUsernameLength = 3
	MaxEmailLength    = 100
	MaxRoomNameLength = 50
	MinRoomNameLength = 1
	MaxMessageLength  = 3000
	MinMessageLength  = 1
	MaxPasswordLength = 28
	MinPasswordLength = 8
)

// ValidationError represents a validation error with field and message
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidateUsername validates username format and length
func ValidateUsername(username string) error {
	username = strings.TrimSpace(username)

	if utf8.RuneCountInString(username) < MinUsernameLength {
		return &ValidationError{
			Field:   "username",
			Message: fmt.Sprintf("Username must be at least %d characters", MinUsernameLength),
		}
	}

	if utf8.RuneCountInString(username) > MaxUsernameLength {
		return &ValidationError{
			Field:   "username",
			Message: fmt.Sprintf("Username must not exceed %d characters", MaxUsernameLength),
		}
	}

	// Username can only contain alphanumeric characters, underscores, and hyphens
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, username)
	if !matched {
		return &ValidationError{
			Field:   "username",
			Message: "Username can only contain letters, numbers, underscores, and hyphens",
		}
	}

	return nil
}

// ValidateEmail validates email format and length
func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)

	if email == "" {
		return &ValidationError{
			Field:   "email",
			Message: "Email is required",
		}
	}

	if utf8.RuneCountInString(email) > MaxEmailLength {
		return &ValidationError{
			Field:   "email",
			Message: fmt.Sprintf("Email must not exceed %d characters", MaxEmailLength),
		}
	}

	// Basic email regex validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return &ValidationError{
			Field:   "email",
			Message: "Invalid email format",
		}
	}

	return nil
}

// ValidatePassword validates password strength and length
func ValidatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return &ValidationError{
			Field:   "password",
			Message: fmt.Sprintf("Password must be at least %d characters", MinPasswordLength),
		}
	}

	if len(password) > MaxPasswordLength {
		return &ValidationError{
			Field:   "password",
			Message: fmt.Sprintf("Password must not exceed %d characters", MaxPasswordLength),
		}
	}

	// Check for at least one letter and one number (basic strength)
	hasLetter := regexp.MustCompile(`[a-zA-Z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)

	if !hasLetter || !hasNumber {
		return &ValidationError{
			Field:   "password",
			Message: "Password must contain at least one letter and one number",
		}
	}

	return nil
}

// ValidateRoomName validates room name format and length
func ValidateRoomName(name string) error {
	name = strings.TrimSpace(name)

	if utf8.RuneCountInString(name) < MinRoomNameLength {
		return &ValidationError{
			Field:   "name",
			Message: "Room name is required",
		}
	}

	if utf8.RuneCountInString(name) > MaxRoomNameLength {
		return &ValidationError{
			Field:   "name",
			Message: fmt.Sprintf("Room name must not exceed %d characters", MaxRoomNameLength),
		}
	}

	return nil
}

// ValidateMessageContent validates message content length
func ValidateMessageContent(content string) error {
	content = strings.TrimSpace(content)

	if utf8.RuneCountInString(content) < MinMessageLength {
		return &ValidationError{
			Field:   "content",
			Message: "Message content is required",
		}
	}

	if utf8.RuneCountInString(content) > MaxMessageLength {
		return &ValidationError{
			Field:   "content",
			Message: fmt.Sprintf("Message content must not exceed %d characters", MaxMessageLength),
		}
	}

	return nil
}

// SanitizeInput removes potentially dangerous HTML/script tags and trims whitespace
func SanitizeInput(input string) string {
	// Trim whitespace
	input = strings.TrimSpace(input)

	// Escape HTML to prevent XSS
	input = html.EscapeString(input)

	return input
}

// SanitizeMessage sanitizes message content while preserving formatting
func SanitizeMessage(content string) string {
	// Trim excessive whitespace but preserve line breaks
	content = strings.TrimSpace(content)

	// Remove null bytes
	content = strings.ReplaceAll(content, "\x00", "")

	// Escape HTML to prevent XSS
	content = html.EscapeString(content)

	return content
}

// ValidateAndSanitizeUsername validates and sanitizes username
func ValidateAndSanitizeUsername(username string) (string, error) {
	username = strings.TrimSpace(username)
	if err := ValidateUsername(username); err != nil {
		return "", err
	}
	return SanitizeInput(username), nil
}

// ValidateAndSanitizeEmail validates and sanitizes email
func ValidateAndSanitizeEmail(email string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if err := ValidateEmail(email); err != nil {
		return "", err
	}
	return email, nil
}

// ValidateAndSanitizeRoomName validates and sanitizes room name
func ValidateAndSanitizeRoomName(name string) (string, error) {
	if err := ValidateRoomName(name); err != nil {
		return "", err
	}
	return SanitizeInput(name), nil
}

// ValidateAndSanitizeMessage validates and sanitizes message content
func ValidateAndSanitizeMessage(content string) (string, error) {
	if err := ValidateMessageContent(content); err != nil {
		return "", err
	}
	return SanitizeMessage(content), nil
}

// IsValidRole checks if a role is valid
func IsValidRole(role string) bool {
	validRoles := map[string]bool{
		"user":  true,
		"admin": true,
	}
	return validRoles[role]
}

// ValidateRole validates user role
func ValidateRole(role string) error {
	if role == "" {
		return nil // Role is optional, defaults to "user"
	}

	if !IsValidRole(role) {
		return &ValidationError{
			Field:   "role",
			Message: "Invalid role. Must be 'user' or 'admin'",
		}
	}

	return nil
}
