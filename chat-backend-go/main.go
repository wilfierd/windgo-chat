package main

import (
    "log"
    "chat-backend-go/config"
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/cors"
)

func main() {
    // Initialize database
    config.ConnectDB()

    // Create Fiber app
    app := fiber.New()

    // CORS middleware
    app.Use(cors.New(cors.Config{
        AllowOrigins: "http://localhost:3000",
        AllowHeaders: "Origin, Content-Type, Accept, Authorization",
        AllowMethods: "GET, POST, PUT, DELETE",
    }))

    // Basic route
    app.Get("/", func(c *fiber.Ctx) error {
        return c.JSON(fiber.Map{
            "message": "WindGo Chat API is running!",
        })
    })

    // Health check
    app.Get("/health", func(c *fiber.Ctx) error {
        return c.JSON(fiber.Map{
            "status": "healthy",
            "database": "connected",
        })
    })

    log.Println("Server starting on :8080")
    log.Fatal(app.Listen(":8080"))
}