package main

import (
	"chat-backend-go/config"
	"chat-backend-go/handlers"
	"chat-backend-go/middleware"
	"chat-backend-go/routes"
	"chat-backend-go/utils"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	// Initialize structured logger
	utils.InitLogger()
	utils.Logger.Info("Starting WindGo Chat Backend...")

	// Initialize database
	config.ConnectDB()
	utils.Logger.Info("Database connected successfully")

	// Initialize search
	if err := config.InitSearch(); err != nil {
		utils.Logger.Warn("Search initialization failed, continuing without search functionality", "error", err)
	} else {
		utils.Logger.Info("Search service connected successfully")
	}

	// Seed demo users and rooms
	utils.SeedDemoUsers()
	utils.SeedDemoRooms()
	utils.Logger.Info("Demo data seeded")

	// Initialize WebSocket hub with database connection
	handlers.InitWebSocketHub(config.GetDB())
	utils.Logger.Info("WebSocket hub initialized")

	// Create Fiber app with custom error handler
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			utils.LogError("Fiber error handler", err, map[string]interface{}{
				"path":   c.Path(),
				"method": c.Method(),
				"code":   code,
			})
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Recovery middleware - must be first to catch all panics
	app.Use(middleware.Recovery())

	// Logger middleware for debugging
	app.Use(logger.New(logger.Config{
		Format: "[${ip}]:${port} ${status} - ${method} ${path}\n",
	}))

	// CORS middleware
	corsOrigin := os.Getenv("CORS_ORIGIN")
	if corsOrigin == "" {
		corsOrigin = "http://localhost:3000"
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins: corsOrigin,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE",
	}))

	// Basic route
	app.Get("/", func(c *fiber.Ctx) error {
		hostname, _ := os.Hostname()
		return c.JSON(fiber.Map{
			"message": "WindGo Chat API is running!",
			"server":  hostname,
		})
	})

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		hostname, _ := os.Hostname()
		return c.JSON(fiber.Map{
			"status":   "healthy",
			"database": "connected",
			"server":   hostname,
		})
	})

	// Setup routes
	routes.SetupAuthRoutes(app)
	routes.UserRoutes(app)
	routes.MessageRoutes(app)
	routes.SearchRoutes(app)
	routes.SetupWebSocketRoutes(app)
	log.Println("All routes registered including WebSocket")

	// Read port from environment (default 8080)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on :%s (CORS_ORIGIN=%s)", port, corsOrigin)
	log.Fatal(app.Listen(":" + port))
}
