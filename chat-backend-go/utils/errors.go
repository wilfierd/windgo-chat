package utils

import (
	"github.com/gofiber/fiber/v2"
)

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Common error messages
const (
	ErrInternalServer        = "An unexpected error occurred. Please try again later."
	ErrUnauthorized          = "You must be logged in to perform this action."
	ErrForbidden             = "You don't have permission to perform this action."
	ErrInvalidRequest        = "The request data is invalid or malformed."
	ErrResourceNotFound      = "The requested resource was not found."
	ErrValidationFailed      = "Validation failed. Please check your input."
	ErrDatabaseOperation     = "A database error occurred. Please try again."
	ErrInvalidToken          = "Your session has expired. Please log in again."
	ErrRateLimitExceeded     = "Too many requests. Please slow down."
	ErrConflict              = "The resource already exists or conflicts with existing data."
	ErrMessageTooLong        = "Your message is too long. Please shorten it."
	ErrRoomNotFound          = "The chat room you're looking for doesn't exist."
	ErrMessageNotFound       = "The message you're looking for doesn't exist."
	ErrUserNotFound          = "The user you're looking for doesn't exist."
	ErrInsufficientPrivilege = "This action requires administrator privileges."
)

// RespondWithError sends a structured error response
func RespondWithError(c *fiber.Ctx, statusCode int, error string, message string) error {
	return c.Status(statusCode).JSON(ErrorResponse{
		Error:   error,
		Message: message,
	})
}

// RespondWithErrorDetails sends an error response with additional details
func RespondWithErrorDetails(c *fiber.Ctx, statusCode int, error string, message string, details map[string]interface{}) error {
	return c.Status(statusCode).JSON(ErrorResponse{
		Error:   error,
		Message: message,
		Details: details,
	})
}

// RespondWithValidationError sends a validation error response
func RespondWithValidationError(c *fiber.Ctx, validationErr error) error {
	if ve, ok := validationErr.(*ValidationError); ok {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "validation_failed",
			Message: ve.Message,
			Details: map[string]interface{}{
				"field": ve.Field,
			},
		})
	}
	return RespondWithError(c, fiber.StatusBadRequest, "validation_failed", ErrValidationFailed)
}

// RespondBadRequest sends a 400 Bad Request response
func RespondBadRequest(c *fiber.Ctx, message string) error {
	if message == "" {
		message = ErrInvalidRequest
	}
	return RespondWithError(c, fiber.StatusBadRequest, "bad_request", message)
}

// RespondUnauthorized sends a 401 Unauthorized response
func RespondUnauthorized(c *fiber.Ctx, message string) error {
	if message == "" {
		message = ErrUnauthorized
	}
	return RespondWithError(c, fiber.StatusUnauthorized, "unauthorized", message)
}

// RespondForbidden sends a 403 Forbidden response
func RespondForbidden(c *fiber.Ctx, message string) error {
	if message == "" {
		message = ErrForbidden
	}
	return RespondWithError(c, fiber.StatusForbidden, "forbidden", message)
}

// RespondNotFound sends a 404 Not Found response
func RespondNotFound(c *fiber.Ctx, resource string) error {
	message := ErrResourceNotFound
	if resource != "" {
		message = resource + " not found"
	}
	return RespondWithError(c, fiber.StatusNotFound, "not_found", message)
}

// RespondConflict sends a 409 Conflict response
func RespondConflict(c *fiber.Ctx, message string) error {
	if message == "" {
		message = ErrConflict
	}
	return RespondWithError(c, fiber.StatusConflict, "conflict", message)
}

// RespondInternalError sends a 500 Internal Server Error response
func RespondInternalError(c *fiber.Ctx) error {
	return RespondWithError(c, fiber.StatusInternalServerError, "internal_error", ErrInternalServer)
}

// RespondInternalErrorWithLog sends a 500 error and logs the error details
func RespondInternalErrorWithLog(c *fiber.Ctx, err error, context string) error {
	// Log the error with context
	LogError("Internal server error", err, map[string]interface{}{
		"context": context,
		"path":    c.Path(),
		"method":  c.Method(),
	})

	return RespondInternalError(c)
}

// RespondSuccess sends a successful response with data
func RespondSuccess(c *fiber.Ctx, message string, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": message,
		"data":    data,
	})
}

// RespondCreated sends a 201 Created response
func RespondCreated(c *fiber.Ctx, message string, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": message,
		"data":    data,
	})
}

// RespondNoContent sends a 204 No Content response
func RespondNoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

// GetUserFriendlyDBError converts database errors to user-friendly messages
func GetUserFriendlyDBError(err error) string {
	if err == nil {
		return ""
	}

	errStr := err.Error()

	// Check for common database errors
	switch {
	case contains(errStr, "unique constraint"):
		return "This entry already exists in the system."
	case contains(errStr, "foreign key constraint"):
		return "This operation cannot be completed because related data exists."
	case contains(errStr, "not null constraint"):
		return "Required information is missing."
	case contains(errStr, "connection refused"):
		return "Database connection error. Please try again."
	case contains(errStr, "timeout"):
		return "The operation took too long. Please try again."
	default:
		return ErrDatabaseOperation
	}
}

// contains is a helper function to check if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(hasPrefix(s, substr) || hasSuffix(s, substr) || containsMiddle(s, substr)))
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
