package config

import (
	"chat-backend-go/models"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDB() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Get database URL from environment or use default
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Use environment variables or defaults
		host := os.Getenv("DB_HOST")
		if host == "" {
			host = "localhost"
		}

		port := os.Getenv("DB_PORT")
		if port == "" {
			port = "5432"
		}

		user := os.Getenv("DB_USER")
		if user == "" {
			user = "postgres"
		}

		password := os.Getenv("DB_PASSWORD")
		if password == "" {
			password = "password"
		}

		dbname := os.Getenv("DB_NAME")
		if dbname == "" {
			dbname = "windgo_chat"
		}

		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
			host, user, password, dbname, port)
	}

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Configure connection pool
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatal("Failed to get underlying sql.DB:", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)                  // Maximum number of idle connections
	sqlDB.SetMaxOpenConns(100)                 // Maximum number of open connections
	sqlDB.SetConnMaxLifetime(time.Hour)        // Connection lifetime
	sqlDB.SetConnMaxIdleTime(30 * time.Minute) // Maximum idle time

	log.Println("Database connected successfully!")
	log.Printf("Connection pool configured: MaxIdle=%d, MaxOpen=%d", 10, 100)

	// Auto migrate models
	err = DB.AutoMigrate(&models.User{}, &models.Room{}, &models.Message{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	log.Println("Database migration completed!")
}

func GetDB() *gorm.DB {
	return DB
}
