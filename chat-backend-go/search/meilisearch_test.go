package search_test

import (
	"chat-backend-go/search"
	"os"
	"testing"
)

// TestMeilisearchClient_Basic tests basic client initialization
func TestMeilisearchClient_Basic(t *testing.T) {
	host := os.Getenv("MEILISEARCH_HOST")
	if host == "" {
		host = "http://localhost:7700"
	}
	apiKey := os.Getenv("MEILISEARCH_API_KEY")

	client, err := search.NewMeilisearchClient(host, apiKey)
	if err != nil {
		t.Skipf("Skipping test: Failed to create Meilisearch client: %v", err)
	}

	if err := client.Ping(); err != nil {
		t.Skipf("Skipping test: Meilisearch not available: %v", err)
	}

	if err := client.EnsureIndex(); err != nil {
		t.Fatalf("Failed to ensure index: %v", err)
	}

	t.Log("Meilisearch client initialized successfully")
}
