package routes

import (
	"chat-backend-go/handlers"
	"chat-backend-go/middleware"

	"github.com/gofiber/fiber/v2"
)

func SearchRoutes(app *fiber.App) {
	api := app.Group("/api/v1")

	// Protected routes (require authentication and track activity)
	protected := api.Use(middleware.AuthRequired(), middleware.TrackActivity())

	// Search routes
	protected.Get("/search", handlers.SearchMessages)
	protected.Get("/search/navigate/:messageId", handlers.GetMessageNavigation)
}
