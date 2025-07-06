package main

import (
    "log"
    "time"

    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/cors"
    "github.com/golang-jwt/jwt/v4"
    "golang.org/x/crypto/bcrypt"
)

// User struct đơn giản
type User struct {
    ID       int    `json:"id"`
    Username string `json:"username"`
    Email    string `json:"email"`
    Password string `json:"-"`
}

// LoginRequest
type LoginRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

// AuthResponse
type AuthResponse struct {
    User  User   `json:"user"`
    Token string `json:"token"`
}

// Fake database - chỉ để test
var users = []User{
    {ID: 1, Username: "admin", Email: "admin@example.com", Password: "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi"}, // password: "password"
}

var jwtSecret = []byte("your-secret-key")

func main() {
    app := fiber.New()

    // CORS
    app.Use(cors.New())

    // Login route
    app.Post("/login", login)

    // Test route
    app.Get("/", func(c *fiber.Ctx) error {
        return c.JSON(fiber.Map{"message": "Chat Backend API"})
    })

    log.Fatal(app.Listen(":8080"))
}

func login(c *fiber.Ctx) error {
    var req LoginRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
    }

    // Tìm user
    var user User
    found := false
    for _, u := range users {
        if u.Email == req.Email {
            user = u
            found = true
            break
        }
    }

    if !found {
        return c.Status(401).JSON(fiber.Map{"error": "Invalid credentials"})
    }

    // Check password
    if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
        return c.Status(401).JSON(fiber.Map{"error": "Invalid credentials"})
    }

    // Tạo JWT token
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "user_id": user.ID,
        "email":   user.Email,
        "exp":     time.Now().Add(time.Hour * 24).Unix(),
    })

    tokenString, err := token.SignedString(jwtSecret)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": "Could not generate token"})
    }

    return c.JSON(AuthResponse{
        User:  user,
        Token: tokenString,
    })
}