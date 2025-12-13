package inbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// RedisInbox manages user message inboxes using Redis ZSETs
// This enables efficient offline message storage with timestamp ordering
type RedisInbox struct {
	client *redis.Client
	ctx    context.Context
}

// InboxMessage represents a message stored in the inbox
type InboxMessage struct {
	MessageID   uuid.UUID  `json:"message_id"`
	SenderID    uuid.UUID  `json:"sender_id"`
	GroupID     *uuid.UUID `json:"group_id,omitempty"`
	Ciphertext  []byte     `json:"ciphertext"`
	MessageType string     `json:"message_type"`
	MediaID     *uuid.UUID `json:"media_id,omitempty"`
	MediaType   string     `json:"media_type,omitempty"`
	Timestamp   time.Time  `json:"timestamp"`
}

// NewRedisInbox creates a new Redis inbox manager
func NewRedisInbox(client *redis.Client) *RedisInbox {
	return &RedisInbox{
		client: client,
		ctx:    context.Background(),
	}
}

// AddToInbox adds a message to a user's offline inbox using ZADD
// Score is the Unix timestamp for ordering
func (r *RedisInbox) AddToInbox(userID uuid.UUID, message *InboxMessage) error {
	key := fmt.Sprintf("inbox:%s", userID.String())

	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Use timestamp as score for ordering
	score := float64(message.Timestamp.UnixNano())

	return r.client.ZAdd(r.ctx, key, redis.Z{
		Score:  score,
		Member: string(data),
	}).Err()
}

// AddMultipleToInbox adds a message to multiple users' inboxes (for group messages)
// Uses pipelining for efficiency
func (r *RedisInbox) AddMultipleToInbox(userIDs []uuid.UUID, message *InboxMessage) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	score := float64(message.Timestamp.UnixNano())

	pipe := r.client.Pipeline()
	for _, userID := range userIDs {
		key := fmt.Sprintf("inbox:%s", userID.String())
		pipe.ZAdd(r.ctx, key, redis.Z{
			Score:  score,
			Member: string(data),
		})
	}

	_, err = pipe.Exec(r.ctx)
	return err
}

// GetPendingMessages retrieves all pending messages for a user
// Returns messages ordered by timestamp (oldest first)
func (r *RedisInbox) GetPendingMessages(userID uuid.UUID) ([]*InboxMessage, error) {
	key := fmt.Sprintf("inbox:%s", userID.String())

	// Get all messages ordered by score (timestamp)
	results, err := r.client.ZRangeByScore(r.ctx, key, &redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()

	if err != nil {
		return nil, err
	}

	messages := make([]*InboxMessage, 0, len(results))
	for _, data := range results {
		var msg InboxMessage
		if err := json.Unmarshal([]byte(data), &msg); err != nil {
			continue
		}
		messages = append(messages, &msg)
	}

	return messages, nil
}

// GetPendingCount returns the number of pending messages for a user
func (r *RedisInbox) GetPendingCount(userID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("inbox:%s", userID.String())
	return r.client.ZCard(r.ctx, key).Result()
}

// RemoveFromInbox removes specific messages from a user's inbox
// Called after successful delivery
func (r *RedisInbox) RemoveFromInbox(userID uuid.UUID, messageIDs []uuid.UUID) error {
	key := fmt.Sprintf("inbox:%s", userID.String())

	// Get all messages to find the ones to remove
	results, err := r.client.ZRange(r.ctx, key, 0, -1).Result()
	if err != nil {
		return err
	}

	messageIDSet := make(map[uuid.UUID]bool)
	for _, id := range messageIDs {
		messageIDSet[id] = true
	}

	pipe := r.client.Pipeline()
	for _, data := range results {
		var msg InboxMessage
		if err := json.Unmarshal([]byte(data), &msg); err != nil {
			continue
		}
		if messageIDSet[msg.MessageID] {
			pipe.ZRem(r.ctx, key, data)
		}
	}

	_, err = pipe.Exec(r.ctx)
	return err
}

// ClearInbox removes all messages from a user's inbox
func (r *RedisInbox) ClearInbox(userID uuid.UUID) error {
	key := fmt.Sprintf("inbox:%s", userID.String())
	return r.client.Del(r.ctx, key).Err()
}

// GetInboxStats returns statistics about a user's inbox
func (r *RedisInbox) GetInboxStats(userID uuid.UUID) (map[string]interface{}, error) {
	key := fmt.Sprintf("inbox:%s", userID.String())

	count, err := r.client.ZCard(r.ctx, key).Result()
	if err != nil {
		return nil, err
	}

	// Get oldest message timestamp
	oldest, err := r.client.ZRangeWithScores(r.ctx, key, 0, 0).Result()
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"pending_count": count,
	}

	if len(oldest) > 0 {
		oldestTime := time.Unix(0, int64(oldest[0].Score))
		stats["oldest_message"] = oldestTime
	}

	return stats, nil
}
