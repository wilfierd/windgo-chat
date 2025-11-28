package search

import (
	"chat-backend-go/models"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/meilisearch/meilisearch-go"
)

const (
	messagesIndexName = "messages"
	defaultLimit      = 20
)

// MessageDocument represents the indexed document structure in Meilisearch
type MessageDocument struct {
	ID        uint   `json:"id"`
	Content   string `json:"content"`
	UserID    uint   `json:"user_id"`
	Username  string `json:"username"`
	RoomID    uint   `json:"room_id"`
	RoomName  string `json:"room_name"`
	CreatedAt int64  `json:"created_at"`
}

// MeilisearchClient implements SearchClient using Meilisearch
type MeilisearchClient struct {
	client meilisearch.ServiceManager
	index  meilisearch.IndexManager
}

// NewMeilisearchClient creates a new Meilisearch client
func NewMeilisearchClient(host string, apiKey string) (*MeilisearchClient, error) {
	client := meilisearch.New(host, meilisearch.WithAPIKey(apiKey))

	return &MeilisearchClient{
		client: client,
		index:  client.Index(messagesIndexName),
	}, nil
}

// Ping checks if the search service is available
func (m *MeilisearchClient) Ping() error {
	_, err := m.client.Health()
	return err
}


// EnsureIndex creates the messages index if it doesn't exist and configures settings
func (m *MeilisearchClient) EnsureIndex() error {
	// Create index if it doesn't exist
	_, err := m.client.CreateIndex(&meilisearch.IndexConfig{
		Uid:        messagesIndexName,
		PrimaryKey: "id",
	})
	// Ignore error if index already exists
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("failed to create index: %w", err)
	}

	// Configure searchable attributes
	_, err = m.index.UpdateSearchableAttributes(&[]string{
		"content",
		"username",
		"room_name",
	})
	if err != nil {
		return fmt.Errorf("failed to update searchable attributes: %w", err)
	}

	// Configure filterable attributes
	filterableAttrs := []interface{}{"room_id", "user_id"}
	_, err = m.index.UpdateFilterableAttributes(&filterableAttrs)
	if err != nil {
		return fmt.Errorf("failed to update filterable attributes: %w", err)
	}

	// Configure sortable attributes
	_, err = m.index.UpdateSortableAttributes(&[]string{
		"created_at",
	})
	if err != nil {
		return fmt.Errorf("failed to update sortable attributes: %w", err)
	}

	return nil
}

// IndexMessage adds or updates a message in the search index
func (m *MeilisearchClient) IndexMessage(message *models.Message) error {
	doc := MessageDocument{
		ID:        message.ID,
		Content:   message.Content,
		UserID:    message.UserID,
		Username:  message.User.Username,
		RoomID:    message.RoomID,
		RoomName:  message.Room.Name,
		CreatedAt: message.CreatedAt.Unix(),
	}

	primaryKey := "id"
	_, err := m.index.AddDocuments([]MessageDocument{doc}, &primaryKey)
	if err != nil {
		return fmt.Errorf("failed to index message: %w", err)
	}

	return nil
}

// RemoveMessage removes a message from the search index
func (m *MeilisearchClient) RemoveMessage(messageID uint) error {
	_, err := m.index.DeleteDocument(strconv.FormatUint(uint64(messageID), 10))
	if err != nil {
		return fmt.Errorf("failed to remove message from index: %w", err)
	}
	return nil
}


// Search performs a text search with cursor-based pagination
func (m *MeilisearchClient) Search(query string, roomIDs []uint, cursor string, limit int) (*SearchResponse, error) {
	if limit <= 0 {
		limit = defaultLimit
	}

	// Build filter for room IDs
	var filter string
	if len(roomIDs) > 0 {
		roomFilters := make([]string, len(roomIDs))
		for i, id := range roomIDs {
			roomFilters[i] = fmt.Sprintf("room_id = %d", id)
		}
		filter = strings.Join(roomFilters, " OR ")
	}

	// Decode cursor to get offset timestamp
	var cursorTimestamp int64
	if cursor != "" {
		decoded, err := base64.StdEncoding.DecodeString(cursor)
		if err == nil {
			cursorTimestamp, _ = strconv.ParseInt(string(decoded), 10, 64)
		}
	}

	// Build search request
	searchReq := &meilisearch.SearchRequest{
		Limit:                 int64(limit + 1), // Request one extra to check if there are more
		AttributesToHighlight: []string{"content"},
		HighlightPreTag:       "<mark>",
		HighlightPostTag:      "</mark>",
		Sort:                  []string{"created_at:desc"},
	}

	if filter != "" {
		searchReq.Filter = filter
	}

	// If we have a cursor, add filter for older messages
	if cursorTimestamp > 0 {
		timestampFilter := fmt.Sprintf("created_at < %d", cursorTimestamp)
		if filter != "" {
			searchReq.Filter = fmt.Sprintf("(%s) AND %s", filter, timestampFilter)
		} else {
			searchReq.Filter = timestampFilter
		}
	}

	// Execute search
	result, err := m.index.Search(query, searchReq)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Process results
	hasMore := len(result.Hits) > limit
	hits := result.Hits
	if hasMore {
		hits = hits[:limit]
	}

	results := make([]SearchResult, 0, len(hits))
	var lastTimestamp int64

	for _, hit := range hits {
		var doc MessageDocument
		if err := hit.DecodeInto(&doc); err != nil {
			continue
		}

		msg := models.Message{
			ID:        doc.ID,
			Content:   doc.Content,
			UserID:    doc.UserID,
			RoomID:    doc.RoomID,
			CreatedAt: time.Unix(doc.CreatedAt, 0),
		}
		msg.User.Username = doc.Username
		msg.Room.Name = doc.RoomName

		lastTimestamp = doc.CreatedAt

		// Extract highlights from formatted results
		var previewSnippet string
		highlights := make(map[string][]string)

		var formatted map[string]interface{}
		if err := hit.Decode(&formatted); err == nil {
			if formattedMap, ok := formatted["_formatted"].(map[string]interface{}); ok {
				if content, ok := formattedMap["content"].(string); ok {
					previewSnippet = content
					highlights["content"] = []string{content}
				}
			}
		}

		// Fallback to content if no highlight
		if previewSnippet == "" {
			previewSnippet = doc.Content
		}

		results = append(results, SearchResult{
			Message:        msg,
			Highlights:     highlights,
			PreviewSnippet: previewSnippet,
		})
	}

	// Build next cursor
	var nextCursor string
	if hasMore && lastTimestamp > 0 {
		nextCursor = base64.StdEncoding.EncodeToString([]byte(strconv.FormatInt(lastTimestamp, 10)))
	}

	return &SearchResponse{
		Results:    results,
		Total:      result.EstimatedTotalHits,
		NextCursor: nextCursor,
		HasMore:    hasMore,
		Query:      query,
	}, nil
}


// GetNavigationContext returns position info for a specific message in search results
func (m *MeilisearchClient) GetNavigationContext(query string, roomIDs []uint, messageID uint) (*NavigationContext, error) {
	// Build filter for room IDs
	var filter string
	if len(roomIDs) > 0 {
		roomFilters := make([]string, len(roomIDs))
		for i, id := range roomIDs {
			roomFilters[i] = fmt.Sprintf("room_id = %d", id)
		}
		filter = strings.Join(roomFilters, " OR ")
	}

	// Search for all matching messages to find position
	searchReq := &meilisearch.SearchRequest{
		Limit: 1000, // Get enough results to find position
		Sort:  []string{"created_at:desc"},
	}

	if filter != "" {
		searchReq.Filter = filter
	}

	result, err := m.index.Search(query, searchReq)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Find the message position
	var currentIndex int
	var prevID, nextID *uint
	totalMatches := len(result.Hits)

	for i, hit := range result.Hits {
		var doc MessageDocument
		if err := hit.DecodeInto(&doc); err != nil {
			continue
		}

		if doc.ID == messageID {
			currentIndex = i + 1 // 1-based index

			// Get previous message ID (earlier in results = newer message)
			if i > 0 {
				var prevDoc MessageDocument
				if err := result.Hits[i-1].DecodeInto(&prevDoc); err == nil {
					prevID = &prevDoc.ID
				}
			}

			// Get next message ID (later in results = older message)
			if i < len(result.Hits)-1 {
				var nextDoc MessageDocument
				if err := result.Hits[i+1].DecodeInto(&nextDoc); err == nil {
					nextID = &nextDoc.ID
				}
			}

			break
		}
	}

	if currentIndex == 0 {
		return nil, fmt.Errorf("message not found in search results")
	}

	return &NavigationContext{
		CurrentIndex: currentIndex,
		TotalMatches: totalMatches,
		PrevID:       prevID,
		NextID:       nextID,
		Query:        query,
	}, nil
}
