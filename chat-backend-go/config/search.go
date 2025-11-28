package config

import (
	"chat-backend-go/search"
	"log"
	"os"
)

var SearchClient search.SearchClient

// InitSearch initializes the search client from environment variables
// with graceful error handling when Meilisearch is unavailable
func InitSearch() error {
	// Get Meilisearch configuration from environment
	host := os.Getenv("MEILISEARCH_HOST")
	if host == "" {
		host = "http://localhost:7700"
	}

	apiKey := os.Getenv("MEILISEARCH_API_KEY")
	// API key can be empty for development

	// Create Meilisearch client
	client, err := search.NewMeilisearchClient(host, apiKey)
	if err != nil {
		log.Printf("Warning: Failed to create Meilisearch client: %v", err)
		return err
	}

	// Test connection
	if err := client.Ping(); err != nil {
		log.Printf("Warning: Meilisearch is unavailable at %s: %v", host, err)
		log.Println("Search functionality will be disabled. Message operations will continue normally.")
		// Don't set SearchClient, leaving it nil to indicate unavailability
		return err
	}

	// Ensure index exists and is configured
	if err := client.EnsureIndex(); err != nil {
		log.Printf("Warning: Failed to ensure Meilisearch index: %v", err)
		log.Println("Search functionality may not work correctly.")
		return err
	}

	// Set global search client
	SearchClient = client

	log.Printf("Meilisearch connected successfully at %s", host)
	log.Println("Search index configured and ready")

	return nil
}

// GetSearchClient returns the global search client
// Returns nil if search is unavailable
func GetSearchClient() search.SearchClient {
	return SearchClient
}
