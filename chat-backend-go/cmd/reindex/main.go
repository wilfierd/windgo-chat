package main

import (
	"chat-backend-go/config"
	"chat-backend-go/utils"
	"flag"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// Parse command line flags
	roomID := flag.Uint("room", 0, "Optional: Reindex only messages in specific room ID")
	flag.Parse()

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// Initialize database connection
	config.ConnectDB()
	if config.DB == nil {
		log.Fatal("Failed to connect to database")
	}

	log.Println("Database connected successfully")

	// Initialize search client
	if err := config.InitSearch(); err != nil {
		log.Fatalf("Failed to initialize search client: %v", err)
	}

	searchClient := config.GetSearchClient()
	if searchClient == nil {
		log.Fatal("Search client is not available")
	}

	log.Println("Search client connected successfully")

	// Ensure search index exists and is configured
	if err := searchClient.EnsureIndex(); err != nil {
		log.Fatalf("Failed to ensure search index: %v", err)
	}

	log.Println("Search index verified")

	// Run reindexing
	var err error
	if *roomID > 0 {
		// Reindex specific room
		log.Printf("Reindexing messages in room ID %d...\n", *roomID)
		err = utils.ReindexRoomMessages(config.DB, searchClient, uint(*roomID))
	} else {
		// Reindex all messages
		log.Println("Reindexing all messages...")
		err = utils.ReindexAllMessages(config.DB, searchClient)
	}

	if err != nil {
		log.Fatalf("Reindexing failed: %v", err)
	}

	log.Println("Reindexing completed successfully!")
	os.Exit(0)
}
