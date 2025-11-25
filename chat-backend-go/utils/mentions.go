package utils

import (
	"chat-backend-go/models"
	"regexp"

	"gorm.io/gorm"
)

const (
	MaxMentionsPerMessage = 20
	MentionPattern        = `@([a-zA-Z0-9_]+)`
)

var mentionRegex *regexp.Regexp

func init() {
	mentionRegex = regexp.MustCompile(MentionPattern)
}

// ParseMentions extracts @username mentions from message content
// Returns a list of valid usernames found in the content
func ParseMentions(content string) []string {
	matches := mentionRegex.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return []string{}
	}

	// Use map for deduplication
	usernameMap := make(map[string]bool)
	usernames := []string{}

	for _, match := range matches {
		if len(match) > 1 {
			username := match[1]
			if !usernameMap[username] {
				usernameMap[username] = true
				usernames = append(usernames, username)

				// Stop after MaxMentionsPerMessage
				if len(usernames) >= MaxMentionsPerMessage {
					break
				}
			}
		}
	}

	return usernames
}

// ValidateMentions checks if usernames exist in the database
// Returns a list of valid user IDs for the given usernames
func ValidateMentions(db *gorm.DB, usernames []string) ([]uint, error) {
	if len(usernames) == 0 {
		return []uint{}, nil
	}

	var users []models.User
	err := db.Where("username IN ?", usernames).Find(&users).Error
	if err != nil {
		return nil, err
	}

	userIDs := make([]uint, 0, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}

	return userIDs, nil
}

// StoreMentions creates mention records for a message
// Limits to MaxMentionsPerMessage to prevent abuse
func StoreMentions(db *gorm.DB, messageID uint, userIDs []uint) error {
	if len(userIDs) == 0 {
		return nil
	}

	// Limit to MaxMentionsPerMessage
	if len(userIDs) > MaxMentionsPerMessage {
		userIDs = userIDs[:MaxMentionsPerMessage]
	}

	// Deduplicate user IDs
	userIDMap := make(map[uint]bool)
	uniqueUserIDs := []uint{}
	for _, userID := range userIDs {
		if !userIDMap[userID] {
			userIDMap[userID] = true
			uniqueUserIDs = append(uniqueUserIDs, userID)
		}
	}

	// Create mention records
	mentions := make([]models.MessageMention, 0, len(uniqueUserIDs))
	for _, userID := range uniqueUserIDs {
		mentions = append(mentions, models.MessageMention{
			MessageID:       messageID,
			MentionedUserID: userID,
		})
	}

	return db.Create(&mentions).Error
}

// UpdateMentions updates mention records when a message is edited
// Deletes old mentions and creates new ones
func UpdateMentions(db *gorm.DB, messageID uint, content string) error {
	// Parse mentions from new content
	usernames := ParseMentions(content)

	// Validate usernames
	userIDs, err := ValidateMentions(db, usernames)
	if err != nil {
		return err
	}

	// Use transaction to ensure atomicity
	return db.Transaction(func(tx *gorm.DB) error {
		// Delete existing mentions
		if err := tx.Where("message_id = ?", messageID).Delete(&models.MessageMention{}).Error; err != nil {
			return err
		}

		// Store new mentions
		return StoreMentions(tx, messageID, userIDs)
	})
}

// GetMentionedUserIDs returns user IDs mentioned in a message
func GetMentionedUserIDs(db *gorm.DB, messageID uint) ([]uint, error) {
	var mentions []models.MessageMention
	err := db.Where("message_id = ?", messageID).Find(&mentions).Error
	if err != nil {
		return nil, err
	}

	userIDs := make([]uint, 0, len(mentions))
	for _, mention := range mentions {
		userIDs = append(userIDs, mention.MentionedUserID)
	}

	return userIDs, nil
}
