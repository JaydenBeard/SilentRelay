package queue

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// MessageQueue handles async message processing using Redis Streams
// Used for analytics, archival, and async delivery retries
type MessageQueue struct {
	client    *redis.Client
	ctx       context.Context
	streamKey string
}

// QueuedMessage represents a message in the queue
type QueuedMessage struct {
	MessageID   uuid.UUID  `json:"message_id"`
	SenderID    uuid.UUID  `json:"sender_id"`
	ReceiverID  *uuid.UUID `json:"receiver_id,omitempty"`
	GroupID     *uuid.UUID `json:"group_id,omitempty"`
	Timestamp   time.Time  `json:"timestamp"`
	EventType   string     `json:"event_type"` // "sent", "delivered", "read", "archived"
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
}

// NewMessageQueue creates a new message queue
func NewMessageQueue(client *redis.Client, streamKey string) *MessageQueue {
	if streamKey == "" {
		streamKey = "message_events"
	}
	return &MessageQueue{
		client:    client,
		ctx:       context.Background(),
		streamKey: streamKey,
	}
}

// Enqueue adds a message event to the queue
func (q *MessageQueue) Enqueue(msg *QueuedMessage) (string, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}

	// Add to Redis Stream
	result, err := q.client.XAdd(q.ctx, &redis.XAddArgs{
		Stream: q.streamKey,
		Values: map[string]interface{}{
			"data":      string(data),
			"timestamp": time.Now().UnixNano(),
		},
	}).Result()

	if err != nil {
		return "", err
	}

	return result, nil
}

// EnqueueForArchival enqueues a message for storage/archival
func (q *MessageQueue) EnqueueForArchival(messageID, senderID uuid.UUID, receiverID, groupID *uuid.UUID) error {
	msg := &QueuedMessage{
		MessageID:  messageID,
		SenderID:   senderID,
		ReceiverID: receiverID,
		GroupID:    groupID,
		Timestamp:  time.Now().UTC(),
		EventType:  "archived",
	}
	_, err := q.Enqueue(msg)
	return err
}

// EnqueueDeliveryStatus enqueues a delivery status update
func (q *MessageQueue) EnqueueDeliveryStatus(messageID uuid.UUID, status string) error {
	msg := &QueuedMessage{
		MessageID: messageID,
		Timestamp: time.Now().UTC(),
		EventType: status,
	}
	_, err := q.Enqueue(msg)
	return err
}

// StartConsumer starts a consumer that processes queued messages
func (q *MessageQueue) StartConsumer(consumerGroup, consumerName string, handler func(*QueuedMessage) error) {
	// Create consumer group if it doesn't exist
	q.client.XGroupCreateMkStream(q.ctx, q.streamKey, consumerGroup, "0")

	for {
		// Read from stream
		streams, err := q.client.XReadGroup(q.ctx, &redis.XReadGroupArgs{
			Group:    consumerGroup,
			Consumer: consumerName,
			Streams:  []string{q.streamKey, ">"},
			Count:    10,
			Block:    5 * time.Second,
		}).Result()

		if err != nil {
			if err == redis.Nil {
				continue
			}
			log.Printf("Error reading from stream: %v", err)
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range streams {
			for _, message := range stream.Messages {
				data, ok := message.Values["data"].(string)
				if !ok {
					continue
				}

				var queuedMsg QueuedMessage
				if err := json.Unmarshal([]byte(data), &queuedMsg); err != nil {
					log.Printf("Failed to parse queued message: %v", err)
					continue
				}

				// Process the message
				if err := handler(&queuedMsg); err != nil {
					log.Printf("Failed to process message %s: %v", queuedMsg.MessageID, err)
					// Could add to dead letter queue here
					continue
				}

				// Acknowledge the message
				q.client.XAck(q.ctx, q.streamKey, consumerGroup, message.ID)
			}
		}
	}
}

// GetQueueLength returns the number of pending messages
func (q *MessageQueue) GetQueueLength() (int64, error) {
	return q.client.XLen(q.ctx, q.streamKey).Result()
}

// GetPendingMessages returns messages that haven't been processed yet
func (q *MessageQueue) GetPendingMessages(consumerGroup string, count int64) ([]QueuedMessage, error) {
	pending, err := q.client.XPendingExt(q.ctx, &redis.XPendingExtArgs{
		Stream: q.streamKey,
		Group:  consumerGroup,
		Start:  "-",
		End:    "+",
		Count:  count,
	}).Result()

	if err != nil {
		return nil, err
	}

	messages := make([]QueuedMessage, 0, len(pending))
	for _, p := range pending {
		// Fetch the actual message
		result, err := q.client.XRange(q.ctx, q.streamKey, p.ID, p.ID).Result()
		if err != nil || len(result) == 0 {
			continue
		}

		data, ok := result[0].Values["data"].(string)
		if !ok {
			continue
		}

		var msg QueuedMessage
		if err := json.Unmarshal([]byte(data), &msg); err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}
