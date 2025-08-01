package utils

import (
	"chat-backend-go/config"
	"chat-backend-go/models"
	"log"

	"golang.org/x/crypto/bcrypt"
)

// SeedDemoUsers creates demo users if they don't exist
func SeedDemoUsers() {
	// Check if admin user exists
	var adminUser models.User
	err := config.DB.Where("email = ?", "admin@windgo.com").First(&adminUser).Error
	if err != nil {
		// Create admin user
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		admin := models.User{
			Username: "admin",
			Email:    "admin@windgo.com",
			Password: string(hashedPassword),
			Role:     "admin",
		}
		if err := config.DB.Create(&admin).Error; err != nil {
			log.Printf("Failed to create admin user: %v", err)
		} else {
			log.Println("Admin user created: admin@windgo.com / admin123")
		}
	}

	// Check if demo user exists
	var demoUser models.User
	err = config.DB.Where("email = ?", "demo@windgo.com").First(&demoUser).Error
	if err != nil {
		// Create demo user
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		demo := models.User{
			Username: "demo",
			Email:    "demo@windgo.com",
			Password: string(hashedPassword),
			Role:     "user",
		}
		if err := config.DB.Create(&demo).Error; err != nil {
			log.Printf("Failed to create demo user: %v", err)
		} else {
			log.Println("Demo user created: demo@windgo.com / admin123")
		}
	}

	// Create additional test user
	var testUser models.User
	err = config.DB.Where("email = ?", "test@windgo.com").First(&testUser).Error
	if err != nil {
		// Create test user
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("test123"), bcrypt.DefaultCost)
		test := models.User{
			Username: "testuser",
			Email:    "test@windgo.com",
			Password: string(hashedPassword),
			Role:     "user",
		}
		if err := config.DB.Create(&test).Error; err != nil {
			log.Printf("Failed to create test user: %v", err)
		} else {
			log.Println("Test user created: test@windgo.com / test123")
		}
	}
}
