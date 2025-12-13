package reactions

import (
	"database/sql"
	"log"
	"time"

	"github.com/google/uuid"
)

// Reaction represents a message reaction
type Reaction struct {
	MessageID uuid.UUID `json:"message_id"`
	UserID    uuid.UUID `json:"user_id"`
	Emoji     string    `json:"emoji"`
	CreatedAt time.Time `json:"created_at"`
}

// ReactionSummary summarizes reactions on a message
type ReactionSummary struct {
	MessageID uuid.UUID              `json:"message_id"`
	Reactions map[string][]uuid.UUID `json:"reactions"` // emoji -> user IDs
	Total     int                    `json:"total"`
}

// ReactionsService handles message reactions
type ReactionsService struct {
	db *sql.DB
}

// NewReactionsService creates a new reactions service
func NewReactionsService(db *sql.DB) *ReactionsService {
	return &ReactionsService{db: db}
}

// AddReaction adds a reaction to a message
func (r *ReactionsService) AddReaction(messageID, userID uuid.UUID, emoji string) error {
	_, err := r.db.Exec(`
		INSERT INTO message_reactions (message_id, user_id, emoji)
		VALUES ($1, $2, $3)
		ON CONFLICT (message_id, user_id, emoji) DO NOTHING
	`, messageID, userID, emoji)
	return err
}

// RemoveReaction removes a reaction from a message
func (r *ReactionsService) RemoveReaction(messageID, userID uuid.UUID, emoji string) error {
	_, err := r.db.Exec(`
		DELETE FROM message_reactions 
		WHERE message_id = $1 AND user_id = $2 AND emoji = $3
	`, messageID, userID, emoji)
	return err
}

// GetReactions gets all reactions for a message
func (r *ReactionsService) GetReactions(messageID uuid.UUID) (*ReactionSummary, error) {
	rows, err := r.db.Query(`
		SELECT user_id, emoji FROM message_reactions WHERE message_id = $1
	`, messageID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	summary := &ReactionSummary{
		MessageID: messageID,
		Reactions: make(map[string][]uuid.UUID),
	}

	for rows.Next() {
		var userID uuid.UUID
		var emoji string
		if err := rows.Scan(&userID, &emoji); err != nil {
			return nil, err
		}

		summary.Reactions[emoji] = append(summary.Reactions[emoji], userID)
		summary.Total++
	}

	return summary, nil
}

// GetUserReaction gets a user's reaction on a message (if any)
func (r *ReactionsService) GetUserReaction(messageID, userID uuid.UUID) ([]string, error) {
	rows, err := r.db.Query(`
		SELECT emoji FROM message_reactions 
		WHERE message_id = $1 AND user_id = $2
	`, messageID, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var emojis []string
	for rows.Next() {
		var emoji string
		if err := rows.Scan(&emoji); err != nil {
			return nil, err
		}
		emojis = append(emojis, emoji)
	}

	return emojis, nil
}

// GetReactionsForMessages gets reactions for multiple messages (batch)
func (r *ReactionsService) GetReactionsForMessages(messageIDs []uuid.UUID) (map[uuid.UUID]*ReactionSummary, error) {
	if len(messageIDs) == 0 {
		return make(map[uuid.UUID]*ReactionSummary), nil
	}

	// Build query with IN clause
	query := `SELECT message_id, user_id, emoji FROM message_reactions WHERE message_id = ANY($1)`

	rows, err := r.db.Query(query, messageIDs)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	summaries := make(map[uuid.UUID]*ReactionSummary)

	for rows.Next() {
		var messageID, userID uuid.UUID
		var emoji string
		if err := rows.Scan(&messageID, &userID, &emoji); err != nil {
			return nil, err
		}

		if summaries[messageID] == nil {
			summaries[messageID] = &ReactionSummary{
				MessageID: messageID,
				Reactions: make(map[string][]uuid.UUID),
			}
		}

		summaries[messageID].Reactions[emoji] = append(summaries[messageID].Reactions[emoji], userID)
		summaries[messageID].Total++
	}

	return summaries, nil
}

// Common reaction emojis
var SupportedEmojis = []string{
	"👍", "❤️", "😂", "😮", "😢", "😡", "🎉", "🤔",
}

// IsValidEmoji checks if an emoji is in the supported list
func IsValidEmoji(emoji string) bool {
	for _, e := range SupportedEmojis {
		if e == emoji {
			return true
		}
	}
	return false
}
