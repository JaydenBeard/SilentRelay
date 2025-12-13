package pubsub

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jaydenbeard/messaging-app/internal/models"
	"github.com/redis/go-redis/v9"
)

// RedisClient wraps the Redis connection for pub/sub and caching
type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

// Hub interface for message delivery callback
type Hub interface {
	DeliverFromRedis(userID uuid.UUID, msg *models.WebSocketMessage)
	BroadcastPresenceFromRedis(msg *models.WebSocketMessage)
}

// NewRedisClient creates a new Redis client with optional authentication
func NewRedisClient(addr string) (*RedisClient, error) {
	// Get Redis password from environment variable (optional)
	password := os.Getenv("REDIS_PASSWORD")

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password, // Empty string if not set (no auth)
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	ctx := context.Background()

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisClient{
		client: client,
		ctx:    ctx,
	}, nil
}

// GetClient returns the underlying Redis client
func (r *RedisClient) GetClient() *redis.Client {
	return r.client
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// ================== Connection Registry ==================

// RegisterConnection registers a user's connection to this server
// Used for: "Where is User B?" -> "User B is on Server B"
func (r *RedisClient) RegisterConnection(userID uuid.UUID, serverID string, deviceID uuid.UUID) {
	key := "connections:" + userID.String()

	// Store as hash: deviceID -> serverID
	r.client.HSet(r.ctx, key, deviceID.String(), serverID)

	// Set expiry (will be refreshed by heartbeats)
	r.client.Expire(r.ctx, key, 2*time.Minute)

	// Also track which users are on each server (for efficient server-to-users lookup)
	serverKey := "server_users:" + serverID
	r.client.SAdd(r.ctx, serverKey, userID.String())
	r.client.Expire(r.ctx, serverKey, 2*time.Minute)
}

// UnregisterConnection removes a user's connection
func (r *RedisClient) UnregisterConnection(userID uuid.UUID, deviceID uuid.UUID) {
	key := "connections:" + userID.String()
	r.client.HDel(r.ctx, key, deviceID.String())
}

// RefreshConnection refreshes the TTL on a connection
func (r *RedisClient) RefreshConnection(userID uuid.UUID, deviceID uuid.UUID) {
	key := "connections:" + userID.String()
	r.client.Expire(r.ctx, key, 2*time.Minute)
}

// GetUserConnectionInfo returns if user is online and which servers they're on
// This implements: "Where is User B?" query
func (r *RedisClient) GetUserConnectionInfo(userID uuid.UUID) (bool, []string) {
	key := "connections:" + userID.String()
	result, err := r.client.HGetAll(r.ctx, key).Result()
	if err != nil || len(result) == 0 {
		return false, nil
	}

	// Get unique servers
	serverSet := make(map[string]bool)
	for _, server := range result {
		serverSet[server] = true
	}

	servers := make([]string, 0, len(serverSet))
	for server := range serverSet {
		servers = append(servers, server)
	}

	return true, servers
}

// GetUserServers returns all servers a user is connected to
func (r *RedisClient) GetUserServers(userID uuid.UUID) ([]string, error) {
	key := "connections:" + userID.String()
	result, err := r.client.HGetAll(r.ctx, key).Result()
	if err != nil {
		return nil, err
	}

	servers := make([]string, 0, len(result))
	serverSet := make(map[string]bool)
	for _, server := range result {
		if !serverSet[server] {
			servers = append(servers, server)
			serverSet[server] = true
		}
	}
	return servers, nil
}

// ================== Presence ==================

// SetUserPresence updates a user's online status
func (r *RedisClient) SetUserPresence(userID uuid.UUID, isOnline bool) {
	key := "presence:" + userID.String()

	if isOnline {
		r.client.Set(r.ctx, key, "online", 2*time.Minute)
	} else {
		r.client.Set(r.ctx, key, time.Now().UTC().Format(time.RFC3339), 24*time.Hour)
	}
}

// GetUserPresence gets a user's online status
func (r *RedisClient) GetUserPresence(userID uuid.UUID) (isOnline bool, lastSeen time.Time) {
	key := "presence:" + userID.String()

	val, err := r.client.Get(r.ctx, key).Result()
	if err != nil || val == "" {
		return false, time.Time{}
	}

	if val == "online" {
		return true, time.Now().UTC()
	}

	// Parse last seen time
	t, err := time.Parse(time.RFC3339, val)
	if err != nil {
		return false, time.Time{}
	}
	return false, t
}

// UpdateLastActive refreshes the user's presence TTL
func (r *RedisClient) UpdateLastActive(userID uuid.UUID) {
	key := "presence:" + userID.String()
	r.client.Expire(r.ctx, key, 2*time.Minute)

	connKey := "connections:" + userID.String()
	r.client.Expire(r.ctx, connKey, 2*time.Minute)
}

// GetBatchPresence checks presence for multiple users efficiently
func (r *RedisClient) GetBatchPresence(userIDs []uuid.UUID) map[uuid.UUID]bool {
	pipe := r.client.Pipeline()
	cmds := make(map[uuid.UUID]*redis.StringCmd)

	for _, userID := range userIDs {
		key := "presence:" + userID.String()
		cmds[userID] = pipe.Get(r.ctx, key)
	}

	if _, err := pipe.Exec(r.ctx); err != nil {
		log.Printf("Warning: batch presence pipeline exec failed: %v", err)
	}

	result := make(map[uuid.UUID]bool)
	for userID, cmd := range cmds {
		val, err := cmd.Result()
		result[userID] = err == nil && val == "online"
	}

	return result
}

// ================== Pub/Sub for Cross-Server Messaging ==================

// PublishMessage publishes a message to be delivered to a user on another server
// Returns error if publishing fails after retries
func (r *RedisClient) PublishMessage(userID uuid.UUID, msg *models.WebSocketMessage) error {
	channel := "messages:" + userID.String()

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("ERROR: Failed to marshal message for pub/sub: %v", err)
		return err
	}

	// Retry logic for critical message delivery
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := r.client.Publish(r.ctx, channel, data).Err(); err != nil {
			if attempt == maxRetries {
				log.Printf("ERROR: Failed to publish message after %d attempts: %v", maxRetries, err)
				return err
			}
			log.Printf("WARN: Failed to publish message (attempt %d/%d): %v", attempt, maxRetries, err)
			time.Sleep(time.Duration(attempt*100) * time.Millisecond) // Exponential backoff
			continue
		}
		return nil
	}
	return nil
}

// PublishToServer publishes a message for delivery to users on a specific server
// Returns error if publishing fails after retries
func (r *RedisClient) PublishToServer(serverID string, userID uuid.UUID, msg *models.WebSocketMessage) error {
	channel := "server:" + serverID + ":" + userID.String()

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("ERROR: Failed to marshal message for server pub/sub: %v", err)
		return err
	}

	// Retry logic for critical message delivery
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := r.client.Publish(r.ctx, channel, data).Err(); err != nil {
			if attempt == maxRetries {
				log.Printf("ERROR: Failed to publish to server after %d attempts: %v", maxRetries, err)
				return err
			}
			log.Printf("WARN: Failed to publish to server (attempt %d/%d): %v", attempt, maxRetries, err)
			time.Sleep(time.Duration(attempt*100) * time.Millisecond)
			continue
		}
		return nil
	}
	return nil
}

// PublishRaw publishes raw data to a user's channel (for system events like device approval)
// Returns error if publishing fails after retries
func (r *RedisClient) PublishRaw(userID uuid.UUID, data []byte) error {
	channel := "messages:" + userID.String()

	// Retry logic for critical events
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := r.client.Publish(r.ctx, channel, data).Err(); err != nil {
			if attempt == maxRetries {
				log.Printf("ERROR: Failed to publish raw message after %d attempts: %v", maxRetries, err)
				return err
			}
			log.Printf("WARN: Failed to publish raw message (attempt %d/%d): %v", attempt, maxRetries, err)
			time.Sleep(time.Duration(attempt*100) * time.Millisecond)
			continue
		}
		return nil
	}
	return nil
}

// PublishPresenceUpdate publishes a presence update to the global presence channel
// All servers subscribe to this channel to receive presence updates
func (r *RedisClient) PublishPresenceUpdate(userID uuid.UUID, isOnline bool, data []byte) {
	channel := "presence:updates"
	if err := r.client.Publish(r.ctx, channel, data).Err(); err != nil {
		log.Printf("Failed to publish presence update: %v", err)
	}
}

// PublishToDevice publishes a WebSocketMessage to a specific device channel
// Returns error if publishing fails after retries
func (r *RedisClient) PublishToDevice(userID, deviceID uuid.UUID, msg *models.WebSocketMessage) error {
	channel := "device:" + deviceID.String()
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("ERROR: Failed to marshal message for device: %v", err)
		return err
	}

	// Retry logic
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := r.client.Publish(r.ctx, channel, data).Err(); err != nil {
			if attempt == maxRetries {
				log.Printf("ERROR: Failed to publish to device after %d attempts: %v", maxRetries, err)
				return err
			}
			log.Printf("WARN: Failed to publish to device (attempt %d/%d): %v", attempt, maxRetries, err)
			time.Sleep(time.Duration(attempt*100) * time.Millisecond)
			continue
		}
		return nil
	}
	return nil
}

// PublishToDeviceRaw publishes raw bytes to a specific device channel
// Returns error if publishing fails after retries
func (r *RedisClient) PublishToDeviceRaw(deviceID uuid.UUID, data []byte) error {
	channel := "device:" + deviceID.String()

	// Retry logic
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := r.client.Publish(r.ctx, channel, data).Err(); err != nil {
			if attempt == maxRetries {
				log.Printf("ERROR: Failed to publish raw data to device after %d attempts: %v", maxRetries, err)
				return err
			}
			log.Printf("WARN: Failed to publish raw data to device (attempt %d/%d): %v", attempt, maxRetries, err)
			time.Sleep(time.Duration(attempt*100) * time.Millisecond)
			continue
		}
		return nil
	}
	return nil
}

// SubscribeToMessages subscribes to messages for users on this server
func (r *RedisClient) SubscribeToMessages(hub Hub) {
	// Pattern subscribe to all user message channels
	pubsub := r.client.PSubscribe(r.ctx, "messages:*")
	defer func() {
		if err := pubsub.Close(); err != nil {
			log.Printf("Warning: failed to close pubsub: %v", err)
		}
	}()

	ch := pubsub.Channel()

	for msg := range ch {
		// Extract user ID from channel name
		userIDStr := msg.Channel[len("messages:"):]
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			continue
		}

		// Parse message
		var wsMsg models.WebSocketMessage
		if err := json.Unmarshal([]byte(msg.Payload), &wsMsg); err != nil {
			log.Printf("Failed to parse pub/sub message: %v", err)
			continue
		}

		// Deliver to hub
		hub.DeliverFromRedis(userID, &wsMsg)
	}
}

// SubscribeToServerMessages subscribes to messages specifically for this server
func (r *RedisClient) SubscribeToServerMessages(serverID string, hub Hub) {
	pattern := "server:" + serverID + ":*"
	pubsub := r.client.PSubscribe(r.ctx, pattern)
	defer func() {
		if err := pubsub.Close(); err != nil {
			log.Printf("Warning: failed to close pubsub: %v", err)
		}
	}()

	ch := pubsub.Channel()

	for msg := range ch {
		// Extract user ID from channel name: "server:serverID:userID"
		prefix := "server:" + serverID + ":"
		userIDStr := msg.Channel[len(prefix):]
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			continue
		}

		var wsMsg models.WebSocketMessage
		if err := json.Unmarshal([]byte(msg.Payload), &wsMsg); err != nil {
			continue
		}

		hub.DeliverFromRedis(userID, &wsMsg)
	}
}

// SubscribeToPresenceUpdates subscribes to the global presence channel
// All servers receive presence updates from all other servers
func (r *RedisClient) SubscribeToPresenceUpdates(hub Hub) {
	pubsub := r.client.Subscribe(r.ctx, "presence:updates")
	defer func() {
		if err := pubsub.Close(); err != nil {
			log.Printf("Warning: failed to close pubsub: %v", err)
		}
	}()

	ch := pubsub.Channel()

	for msg := range ch {
		var wsMsg models.WebSocketMessage
		if err := json.Unmarshal([]byte(msg.Payload), &wsMsg); err != nil {
			log.Printf("Failed to parse presence update: %v", err)
			continue
		}

		// Broadcast to all local clients
		hub.BroadcastPresenceFromRedis(&wsMsg)
	}
}

// ================== Notifications ==================

// PublishNotification sends a notification event for push notification delivery
func (r *RedisClient) PublishNotification(userID uuid.UUID, data map[string]interface{}) {
	channel := "notifications:" + userID.String()

	payload, err := json.Marshal(data)
	if err != nil {
		return
	}

	r.client.Publish(r.ctx, channel, payload)
}

// ================== Typing Indicators ==================

// PublishTyping broadcasts a typing indicator
func (r *RedisClient) PublishTyping(fromUserID, toUserID uuid.UUID, isTyping bool) {
	channel := "typing:" + toUserID.String()

	data := map[string]interface{}{
		"user_id":   fromUserID,
		"is_typing": isTyping,
		"timestamp": time.Now().UTC(),
	}

	payload, err := json.Marshal(data)
	if err != nil {
		log.Printf("Warning: Failed to marshal typing indicator: %v", err)
		return
	}
	if err := r.client.Publish(r.ctx, channel, payload).Err(); err != nil {
		log.Printf("Warning: Failed to publish typing indicator: %v", err)
	}
}

// ================== Rate Limiting ==================

// CheckRateLimit checks if an action is rate limited
func (r *RedisClient) CheckRateLimit(key string, limit int, window time.Duration) (bool, error) {
	current, err := r.client.Incr(r.ctx, "ratelimit:"+key).Result()
	if err != nil {
		return false, err
	}

	if current == 1 {
		r.client.Expire(r.ctx, "ratelimit:"+key, window)
	}

	return current <= int64(limit), nil
}

// ================== Session Caching ==================

// CacheSession caches a validated session for faster auth
func (r *RedisClient) CacheSession(tokenHash string, userID uuid.UUID, ttl time.Duration) {
	r.client.Set(r.ctx, "session:"+tokenHash, userID.String(), ttl)
}

// GetCachedSession retrieves a cached session
func (r *RedisClient) GetCachedSession(tokenHash string) (*uuid.UUID, error) {
	val, err := r.client.Get(r.ctx, "session:"+tokenHash).Result()
	if err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(val)
	if err != nil {
		return nil, err
	}
	return &userID, nil
}

// InvalidateSession removes a session from cache
func (r *RedisClient) InvalidateSession(tokenHash string) {
	r.client.Del(r.ctx, "session:"+tokenHash)
}
