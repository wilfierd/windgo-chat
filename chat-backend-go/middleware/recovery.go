package middleware

import (
	"fmt"
	"log"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
)

// Recovery middleware catches panics and returns a 500 error
func Recovery() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				// Log the panic with stack trace
				log.Printf("PANIC RECOVERED: %v\n%s", r, debug.Stack())

				// Return error response
				err := c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":   "Internal server error",
					"message": "An unexpected error occurred. Please try again later.",
				})

				if err != nil {
					log.Printf("Failed to send error response: %v", err)
				}
			}
		}()

		return c.Next()
	}
}

// RecoveryWithLogger is a recovery middleware that uses a custom logger
func RecoveryWithLogger(logger func(c *fiber.Ctx, err interface{}, stack []byte)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()

				// Call custom logger
				if logger != nil {
					logger(c, r, stack)
				} else {
					log.Printf("PANIC RECOVERED: %v\n%s", r, stack)
				}

				// Return error response
				err := c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":   "Internal server error",
					"message": "An unexpected error occurred. Please try again later.",
				})

				if err != nil {
					log.Printf("Failed to send error response: %v", err)
				}
			}
		}()

		return c.Next()
	}
}

// RecoveryConfig defines config for Recovery middleware
type RecoveryConfig struct {
	// Next defines a function to skip this middleware
	Next func(c *fiber.Ctx) bool

	// EnableStackTrace enables printing stack trace
	EnableStackTrace bool

	// StackTraceHandler defines a function to handle stack trace
	StackTraceHandler func(c *fiber.Ctx, e interface{})
}

// RecoveryWithConfig creates a recovery middleware with custom config
func RecoveryWithConfig(config RecoveryConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if config.Next != nil && config.Next(c) {
			return c.Next()
		}

		defer func() {
			if r := recover(); r != nil {
				if config.EnableStackTrace {
					stack := debug.Stack()
					log.Printf("PANIC RECOVERED: %v\n%s", r, stack)
				} else {
					log.Printf("PANIC RECOVERED: %v", r)
				}

				if config.StackTraceHandler != nil {
					config.StackTraceHandler(c, r)
				}

				// Format error message based on error type
				var errMsg string
				switch e := r.(type) {
				case error:
					errMsg = e.Error()
				case string:
					errMsg = e
				default:
					errMsg = fmt.Sprintf("%v", r)
				}

				// Return error response
				err := c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":   "Internal server error",
					"message": "An unexpected error occurred",
					"detail":  errMsg,
				})

				if err != nil {
					log.Printf("Failed to send error response: %v", err)
				}
			}
		}()

		return c.Next()
	}
}
