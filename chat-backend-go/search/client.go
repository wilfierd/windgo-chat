package search

import "chat-backend-go/models"

// SearchResult represents a single search result with highlighting
type SearchResult struct {
	Message        models.Message      `json:"message"`
	Highlights     map[string][]string `json:"highlights,omitempty"`
	PreviewSnippet string              `json:"preview_snippet"`
}

// SearchResponse contains cursor-paginated search results
type SearchResponse struct {
	Results    []SearchResult `json:"results"`
	Total      int64          `json:"total"`
	NextCursor string         `json:"next_cursor"`
	HasMore    bool           `json:"has_more"`
	Query      string         `json:"query"`
}

// NavigationContext provides match position info for Previous/Next navigation
type NavigationContext struct {
	CurrentIndex int    `json:"current_index"`
	TotalMatches int    `json:"total_matches"`
	PrevID       *uint  `json:"prev_id"`
	NextID       *uint  `json:"next_id"`
	Query        string `json:"query"`
}

// SearchClient defines the interface for search operations
type SearchClient interface {
	// IndexMessage adds or updates a message in the search index
	IndexMessage(message *models.Message) error

	// RemoveMessage removes a message from the search index
	RemoveMessage(messageID uint) error

	// Search performs a text search with cursor-based pagination
	Search(query string, roomIDs []uint, cursor string, limit int) (*SearchResponse, error)

	// GetNavigationContext returns position info for a specific message in search results
	GetNavigationContext(query string, roomIDs []uint, messageID uint) (*NavigationContext, error)

	// Ping checks if the search service is available
	Ping() error

	// EnsureIndex creates the messages index if it doesn't exist
	EnsureIndex() error
}
