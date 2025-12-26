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

	// Create additional test users
	testUsers := []struct {
		Username string
		Email    string
		Password string
	}{
		{"testuser", "test@windgo.com", "test123"},
		{"alice", "alice@windgo.com", "alice123"},
		{"bob", "bob@windgo.com", "bob123"},
	}

	for _, userData := range testUsers {
		var user models.User
		err = config.DB.Where("email = ?", userData.Email).First(&user).Error
		if err != nil {
			hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(userData.Password), bcrypt.DefaultCost)
			newUser := models.User{
				Username: userData.Username,
				Email:    userData.Email,
				Password: string(hashedPassword),
				Role:     "user",
			}
			if err := config.DB.Create(&newUser).Error; err != nil {
				log.Printf("Failed to create user %s: %v", userData.Username, err)
			} else {
				log.Printf("User created: %s / %s", userData.Email, userData.Password)
			}
		}
	}
}

// SeedDemoRooms creates demo chat rooms if they don't exist
func SeedDemoRooms() {
	defaultRooms := []struct {
		Name string
	}{
		{"General"},
		{"Random"},
		{"Tech Talk"},
		{"Gaming"},
	}

	for _, roomData := range defaultRooms {
		var room models.Room
		err := config.DB.Where("name = ?", roomData.Name).First(&room).Error
		if err != nil {
			newRoom := models.Room{
				Name: roomData.Name,
				Type: models.RoomTypeGroup,
			}
			if err := config.DB.Create(&newRoom).Error; err != nil {
				log.Printf("Failed to create room %s: %v", roomData.Name, err)
			} else {
				log.Printf("Room created: %s", roomData.Name)
			}
		}
	}

	// Add all users to all group rooms
	SeedRoomMemberships()
}

// SeedRoomMemberships adds all users to all group rooms
func SeedRoomMemberships() {
	var users []models.User
	var rooms []models.Room

	// Get all users
	if err := config.DB.Find(&users).Error; err != nil {
		log.Printf("Failed to fetch users for room memberships: %v", err)
		return
	}

	// Get all group rooms
	if err := config.DB.Where("type = ?", models.RoomTypeGroup).Find(&rooms).Error; err != nil {
		log.Printf("Failed to fetch rooms for memberships: %v", err)
		return
	}

	// Add each user to each room
	for _, user := range users {
		for _, room := range rooms {
			var existing models.RoomMembership
			err := config.DB.Where("user_id = ? AND room_id = ?", user.ID, room.ID).First(&existing).Error
			if err != nil {
				// Membership doesn't exist, create it
				role := models.RoleMember
				if user.Role == "admin" {
					role = models.RoleAdmin
				}
				membership := models.RoomMembership{
					UserID: user.ID,
					RoomID: room.ID,
					Role:   role,
				}
				if err := config.DB.Create(&membership).Error; err != nil {
					log.Printf("Failed to add user %s to room %s: %v", user.Username, room.Name, err)
				}
			}
		}
	}
	log.Println("Room memberships seeded")
}
