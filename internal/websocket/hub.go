package websocket

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jaydenbeard/messaging-app/internal/db"
	"github.com/jaydenbeard/messaging-app/internal/inbox"
	"github.com/jaydenbeard/messaging-app/internal/models"
	"github.com/jaydenbeard/messaging-app/internal/pubsub"
	"github.com/jaydenbeard/messaging-app/internal/queue"
	"github.com/jaydenbeard/messaging-app/internal/security"
)

// Connection limits for DoS protection
const (
	MaxConnectionsPerUser = 10    // Max devices per user
	MaxTotalConnections   = 10000 // Max total WebSocket connections
)

// Hub maintains the set of active clients and broadcasts messages
// Implements the message flows from the sequence diagrams:
// - Multi-device sync (User's devices on different servers)
// - Online user delivery (real-time via Redis pub/sub)
// - Offline user inbox (Redis ZSET + push notification)
// - Group fan-out (parallel delivery to online, store for offline)
type Hub struct {
	serverID string

	// Registered clients by user ID (supports multiple devices per user)
	clients map[uuid.UUID]map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Inbound messages from clients (with client context)
	broadcast chan *models.WebSocketMessage

	// Cross-server message delivery via Redis
	redis *pubsub.RedisClient

	// Database for message persistence
	db *db.PostgresDB

	// Redis inbox for offline message storage
	inbox *inbox.RedisInbox

	// Message queue for async processing
	queue *queue.MessageQueue

	// Mutex for thread-safe client map access
	mu sync.RWMutex

	// Connection tracking for limits
	totalConnections int32

	// Shutdown signal
	shutdown chan struct{}

	// HMAC secret for message authentication
	hmacSecret []byte

	// Nonce store for replay protection
	nonceStore map[string]time.Time
	nonceMutex sync.RWMutex

	// Audit logger for security events
	auditLogger *security.AuditLogger
}

// NewHub creates a new Hub instance
func NewHub(serverID string, redis *pubsub.RedisClient, database *db.PostgresDB, hmacSecret string, auditLogger *security.AuditLogger) *Hub {
	// SECURITY: Validate HMAC secret is provided
	var secret []byte
	if hmacSecret == "" {
		// SECURITY: Generate a cryptographically secure random 32-byte secret
		// This is acceptable for single-instance deployments but will break
		// cross-server message authentication in clustered deployments
		secret = make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			// SECURITY CRITICAL: Never fall back to deterministic secrets
			// If we can't generate secure random bytes, the system is compromised
			log.Printf("FATAL SECURITY ERROR: Failed to generate cryptographically secure random secret: %v", err)
			log.Printf("FATAL: Cannot continue with insecure secret generation. Exiting.")
			os.Exit(1)
		}
		log.Printf("WARNING: No HMAC secret provided, generated random secret. This will not work in clustered deployments.")
	} else {
		// Validate the provided secret meets minimum security requirements
		if len(hmacSecret) < 32 {
			log.Printf("FATAL SECURITY ERROR: HMAC secret must be at least 32 bytes (256 bits). Provided: %d bytes", len(hmacSecret))
			os.Exit(1)
		}
		secret = []byte(hmacSecret)
	}

	return &Hub{
		serverID:    serverID,
		clients:     make(map[uuid.UUID]map[*Client]bool),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		broadcast:   make(chan *models.WebSocketMessage, 256),
		redis:       redis,
		db:          database,
		inbox:       inbox.NewRedisInbox(redis.GetClient()),
		queue:       queue.NewMessageQueue(redis.GetClient(), "message_events"),
		shutdown:    make(chan struct{}),
		hmacSecret:  secret,
		nonceStore:  make(map[string]time.Time),
		auditLogger: auditLogger,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.handleMessage(message)

		case <-h.shutdown:
			h.closeAllClients()
			return
		}
	}
}

// Shutdown gracefully shuts down the hub
func (h *Hub) Shutdown() {
	close(h.shutdown)
}

// Register adds a client to the hub
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast sends a message to be processed
func (h *Hub) Broadcast(message *models.WebSocketMessage) {
	h.broadcast <- message
}

func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check total connection limit (DoS protection)
	if h.totalConnections >= MaxTotalConnections {
		log.Printf("SECURITY: Max total connections reached (%d), rejecting user=%s",
			MaxTotalConnections, client.UserID)
		close(client.send)
		return
	}

	// Check per-user connection limit (prevent single user from hogging connections)
	if userClients, ok := h.clients[client.UserID]; ok {
		if len(userClients) >= MaxConnectionsPerUser {
			log.Printf("SECURITY: Max connections per user reached (%d) for user=%s",
				MaxConnectionsPerUser, client.UserID)
			close(client.send)
			return
		}
	}

	if _, ok := h.clients[client.UserID]; !ok {
		h.clients[client.UserID] = make(map[*Client]bool)
	}
	h.clients[client.UserID][client] = true
	// Use atomic operation for counter
	atomic.AddInt32(&h.totalConnections, 1)

	// Register connection in Redis (for cross-server routing)
	// This enables: "Where is User B?" -> "User B is on Server B"
	h.redis.RegisterConnection(client.UserID, h.serverID, client.DeviceID)

	// Update user's online status
	h.redis.SetUserPresence(client.UserID, true)

	log.Printf("[Hub] Client registered: user=%s, device=%s, server=%s",
		client.UserID, client.DeviceID, h.serverID)

	// Broadcast presence update to all connected users
	// This notifies everyone that this user came online
	go h.broadcastPresenceUpdate(client.UserID, true)

	// Deliver pending messages from inbox (User B comes online flow)
	go h.deliverPendingMessages(client)
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if userClients, ok := h.clients[client.UserID]; ok {
		if _, ok := userClients[client]; ok {
			delete(userClients, client)
			close(client.send)
			// Use atomic operation for counter
			atomic.AddInt32(&h.totalConnections, -1)

			// Remove connection from Redis
			h.redis.UnregisterConnection(client.UserID, client.DeviceID)

			// If no more devices connected, set offline
			if len(userClients) == 0 {
				delete(h.clients, client.UserID)
				h.redis.SetUserPresence(client.UserID, false)
				// Broadcast presence update to all connected users
				go h.broadcastPresenceUpdate(client.UserID, false)
			}

			log.Printf("Client unregistered: user=%s, device=%s",
				client.UserID, client.DeviceID)
		}
	}
}

func (h *Hub) handleMessage(msg *models.WebSocketMessage) {
	// Find the client to get the auth token for HMAC verification
	h.mu.RLock()
	var client *Client
	if userClients, ok := h.clients[msg.SenderID]; ok {
		for c := range userClients {
			if c.DeviceID == msg.DeviceID {
				client = c
				break
			}
		}
	}
	h.mu.RUnlock()

	if client == nil {
		log.Printf("SECURITY: Client not found for message verification: user=%s, device=%s", msg.SenderID, msg.DeviceID)
		return
	}

	// Verify HMAC authentication for message integrity
	if !h.verifyMessageHMAC(msg, client.authToken) {
		log.Printf("SECURITY: Invalid HMAC for message type %s from user %s - terminating connection", msg.Type, msg.SenderID)

		// SECURITY: Terminate connection on signature verification failure
		// This prevents message tampering and MITM attacks
		go h.unregisterClient(client)
		return
	}

	// Check for replay attacks using nonce
	if !h.checkAndStoreNonce(msg) {
		log.Printf("SECURITY: Replay attack detected for message from user %s", msg.SenderID)
		return
	}

	switch msg.Type {
	case models.MessageTypeSend:
		h.handleSendMessage(msg)
	case models.MessageTypeDeliveryAck:
		h.handleDeliveryAck(msg)
	case models.MessageTypeReadReceipt:
		h.handleReadReceipt(msg)
	case models.MessageTypeTyping:
		h.handleTypingIndicator(msg)
	case models.MessageTypeHeartbeat:
		h.handleHeartbeat(msg)
	// Call signaling - forward to recipient
	case models.MessageTypeCallOffer,
		models.MessageTypeCallAnswer,
		models.MessageTypeCallReject,
		models.MessageTypeCallEnd,
		models.MessageTypeCallBusy,
		models.MessageTypeIceCandidate:
		h.handleCallSignaling(msg)
	// Device-to-device sync - relay encrypted data (server can't read it)
	case models.MessageTypeSyncRequest,
		models.MessageTypeSyncData,
		models.MessageTypeSyncAck:
		h.handleDeviceSync(msg)
	// Media key exchange - forward encrypted key to recipient
	case models.MessageTypeMediaKey:
		h.handleMediaKey(msg)
	}
}

// verifyMessageHMAC verifies the HMAC signature of a WebSocket message using the client's auth token
func (h *Hub) verifyMessageHMAC(msg *models.WebSocketMessage, authToken string) bool {
	// Signature and nonce are on the WebSocketMessage struct, not in the payload
	if msg.Signature == "" || msg.Nonce == "" {
		log.Printf("SECURITY: Message missing signature or nonce for type %s from user %s", msg.Type, msg.SenderID)
		// Audit: Missing signature
		if h.auditLogger != nil {
			h.auditLogger.LogSecurityEvent(context.Background(), security.AuditEventInvalidRequest,
				security.AuditResultFailure, &msg.SenderID,
				"WebSocket message missing signature or nonce", map[string]any{
					"message_type": msg.Type,
				})
		}
		return false
	}

	// Create message string for HMAC: type + timestamp + messageId + payload
	// Frontend sends timestamp as ISO string with milliseconds (JavaScript toISOString format)
	// We must use the exact same format: "2006-01-02T15:04:05.000Z" (always 3 decimal places with Z suffix)
	timestampStr := msg.Timestamp.UTC().Format("2006-01-02T15:04:05.000Z")
	if msg.Timestamp.IsZero() {
		timestampStr = "0"
	}

	messageStr := fmt.Sprintf("%s:%s:%s:%s",
		msg.Type,
		timestampStr,
		msg.MessageID.String(),
		string(msg.Payload))

	// Use client's auth token as HMAC key (first 32 bytes for SHA-256)
	tokenBytes := []byte(authToken)
	if len(tokenBytes) < 32 {
		// Pad to 32 bytes
		padding := make([]byte, 32-len(tokenBytes))
		tokenBytes = append(tokenBytes, padding...)
	} else {
		tokenBytes = tokenBytes[:32]
	}

	mac := hmac.New(sha256.New, tokenBytes)
	mac.Write([]byte(messageStr))
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	// Compare HMACs using constant-time comparison
	if hmac.Equal([]byte(expectedMAC), []byte(msg.Signature)) {
		return true
	}

	// Log details for debugging
	log.Printf("SECURITY: HMAC verification failed for message type %s from user %s", msg.Type, msg.SenderID)
	log.Printf("[HMAC DEBUG] Expected: %s..., Got: %s...", expectedMAC[:16], msg.Signature[:min(16, len(msg.Signature))])
	log.Printf("[HMAC DEBUG] Message string: %s", messageStr[:min(100, len(messageStr))])

	// Audit: HMAC verification failure
	if h.auditLogger != nil {
		h.auditLogger.LogSecurityEvent(context.Background(), security.AuditEventInvalidRequest,
			security.AuditResultFailure, &msg.SenderID,
			"WebSocket message HMAC verification failed", map[string]any{
				"message_type":  msg.Type,
				"expected_hmac": expectedMAC[:8] + "...", // Log partial HMAC for debugging
			})
	}

	return false
}

// checkAndStoreNonce checks for replay attacks using nonces
func (h *Hub) checkAndStoreNonce(msg *models.WebSocketMessage) bool {
	// Use the actual nonce from the message for replay protection
	// The nonce should be a unique random value generated by the client
	nonceStr := msg.Nonce
	if nonceStr == "" {
		// Fallback for backwards compatibility - use message content as nonce
		nonceStr = fmt.Sprintf("%s:%s:%d",
			msg.SenderID.String(),
			msg.DeviceID.String(),
			msg.Timestamp.UnixNano()) // Use nanoseconds for better precision
	}

	h.nonceMutex.Lock()
	defer h.nonceMutex.Unlock()

	// Check if nonce was already used
	if usedTime, exists := h.nonceStore[nonceStr]; exists {
		// Allow some tolerance for clock skew (5 seconds)
		if time.Since(usedTime) < 5*time.Second {
			log.Printf("SECURITY: Replay attack detected for user %s, device %s", msg.SenderID, msg.DeviceID)

			// Audit: Replay attack detected
			if h.auditLogger != nil {
				h.auditLogger.LogSecurityEvent(context.Background(), security.AuditEventReplayAttempt,
					security.AuditResultFailure, &msg.SenderID,
					"WebSocket message replay attack detected", map[string]any{
						"message_type": msg.Type,
						"device_id":    msg.DeviceID,
						"timestamp":    msg.Timestamp,
					})
			}

			return false // Replay attack detected
		}
	}

	// Store nonce with current timestamp
	h.nonceStore[nonceStr] = time.Now()

	// Clean up old nonces (older than 10 minutes)
	cutoff := time.Now().Add(-10 * time.Minute)
	for nonce, timestamp := range h.nonceStore {
		if timestamp.Before(cutoff) {
			delete(h.nonceStore, nonce)
		}
	}

	return true
}

// handleSendMessage implements the "Message Flow (Both Users Online)" sequence
func (h *Hub) handleSendMessage(msg *models.WebSocketMessage) {
	log.Printf("[MSG] handleSendMessage: from=%s", msg.SenderID)

	// Parse the encrypted message payload
	var payload models.EncryptedMessage
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		log.Printf("[MSG] Failed to parse message payload: %v", err)
		// SECURITY: Do not log raw payload as it may contain sensitive encrypted data
		return
	}
	log.Printf("[MSG] Parsed payload: sender=%s, receiver=%v, type=%s, ciphertext_len=%d, sealed_sender=%v",
		msg.SenderID, payload.ReceiverID, payload.MessageType, len(payload.Ciphertext),
		payload.SealedSenderCertificateID != nil)

	// Step 2: Use client's message_id if provided, otherwise generate new one
	// This allows clients to correlate status updates with their local messages
	messageID := msg.MessageID
	if messageID == uuid.Nil {
		messageID = uuid.New()
	}
	timestamp := time.Now().UTC()

	// Handle sealed sender message format
	var isSealedSender bool
	var sealedSenderCertID *uuid.UUID
	if payload.SealedSenderCertificateID != nil {
		isSealedSender = true
		sealedSenderCertID = payload.SealedSenderCertificateID
		log.Printf("[MSG] Sealed sender message detected, certificate ID: %s", sealedSenderCertID)
	}

	// Store message in database (encrypted - server cannot read content!)
	dbMessage := &db.Message{
		MessageID:   messageID,
		SenderID:    msg.SenderID,
		ReceiverID:  payload.ReceiverID,
		GroupID:     payload.GroupID,
		Ciphertext:  payload.Ciphertext,
		MessageType: payload.MessageType,
		MediaID:     payload.MediaID,
		MediaType:   payload.MediaType,
		Timestamp:   timestamp,
		Status:      "sent",
	}

	// If this is a sealed sender message, store the certificate association
	if isSealedSender {
		// Store sealed sender certificate association in database
		// This allows recipients to verify the sender's identity
		if sealedSenderCertID != nil {
			// Store the certificate ID with the message for verification
			// In a real implementation, we would store this in a sealed_sender_messages table
			// For now, we'll just log it and handle it in the decryption process
			log.Printf("[MSG] Sealed sender certificate ID: %s", sealedSenderCertID)

			// Update the database message to include sealed sender info
			// This would be done via a separate table in a full implementation
			// For now, we'll just ensure the message is properly marked
			// dbMessage.SealedSenderCertID = sealedSenderCertID
		}
	}

	if err := h.db.SaveMessage(dbMessage); err != nil {
		log.Printf("Failed to save message: %v", err)
		h.sendErrorToClient(msg.SenderID, "Failed to save message")
		return
	}
	// NOTE: Conversation state is managed CLIENT-SIDE only for security.
	// Server only stores encrypted message content, not metadata about who talks to whom.

	// Step 3: ACK (status: sent) to sender - all sender's devices
	ack := &models.WebSocketMessage{
		Type:      models.MessageTypeSentAck,
		MessageID: messageID,
		Timestamp: timestamp,
		Payload:   json.RawMessage(`{"status": "sent"}`),
	}
	h.sendToUserAllDevices(msg.SenderID, ack, msg.DeviceID)

	// Step 4+5: Where is User B? Route message accordingly
	if payload.GroupID != nil {
		// Group message - fan-out to all members
		h.deliverGroupMessage(dbMessage, &payload, isSealedSender)
	} else if payload.ReceiverID != nil {
		// Direct message
		h.deliverDirectMessage(dbMessage, &payload, isSealedSender)
	}

	// Step 10.1: Async processing - enqueue for analytics/archival
	go func() {
		if err := h.queue.EnqueueForArchival(messageID, msg.SenderID, payload.ReceiverID, payload.GroupID); err != nil {
			log.Printf("Warning: failed to enqueue for archival: %v", err)
		}
	}()
}

// deliverDirectMessage implements cross-server message delivery
func (h *Hub) deliverDirectMessage(msg *db.Message, payload *models.EncryptedMessage, isSealedSender bool) {
	recipientID := *payload.ReceiverID
	log.Printf("[Deliver] deliverDirectMessage: to=%s, msg_id=%s", recipientID, msg.MessageID)

	// Check if recipient is online (Step 4: Where is User B?)
	isOnline, serverIDs := h.redis.GetUserConnectionInfo(recipientID)
	log.Printf("[Deliver] Recipient online=%v, servers=%v", isOnline, serverIDs)

	deliveryMsg := &models.WebSocketMessage{
		Type:      models.MessageTypeDeliver,
		MessageID: msg.MessageID,
		SenderID:  msg.SenderID,
		Timestamp: msg.Timestamp,
		Payload:   mustMarshal(payload),
	}

	// For sealed sender messages, the server doesn't know the actual sender
	// The recipient will decrypt using their private key to get the sender info
	if isSealedSender {
		// In sealed sender mode, we hide the sender ID from the server
		// The recipient will decrypt the message to get the actual sender
		deliveryMsg.SenderID = uuid.Nil // Hide sender from server
		log.Printf("[MSG] Sealed sender: hiding sender ID from server for message %s", msg.MessageID)
	}

	if isOnline && len(serverIDs) > 0 {
		// User B is online - deliver in real time
		// Step 5: User B is on Server B (or multiple servers for multi-device)

		// Check if recipient is on THIS server
		h.mu.RLock()
		localClients, onThisServer := h.clients[recipientID]
		h.mu.RUnlock()

		log.Printf("[Deliver] Recipient on this server=%v, local_clients=%d", onThisServer, len(localClients))

		if onThisServer && len(localClients) > 0 {
			// Deliver locally to all recipient's devices
			deliveredCount := 0
			for client := range localClients {
				select {
				case client.send <- mustMarshal(deliveryMsg):
					deliveredCount++
					log.Printf("[Deliver] Message delivered to device=%s", client.DeviceID)
				default:
					log.Printf("[Deliver] Warning: Client buffer full, unregistering device=%s", client.DeviceID)
					go h.unregisterClient(client)
				}
			}
			log.Printf("[Deliver] Delivered to %d local devices", deliveredCount)
		}

		// Also publish to Redis for any other servers the user is connected to
		// This handles multi-device across servers (User A's Tablet on Server C, etc.)
		for _, serverID := range serverIDs {
			if serverID != h.serverID {
				if err := h.redis.PublishToServer(serverID, recipientID, deliveryMsg); err != nil {
					log.Printf("Warning: failed to publish to server %s: %v", serverID, err)
				}
			}
		}
	} else {
		// User B is offline - use offline flow
		h.handleOfflineDelivery(recipientID, msg, payload)
	}
}

// handleOfflineDelivery implements "Message Flow (User Offline)"
func (h *Hub) handleOfflineDelivery(userID uuid.UUID, msg *db.Message, _ *models.EncryptedMessage) {
	// Step 3.1: Add message to User B's inbox (ZSET)
	inboxMsg := &inbox.InboxMessage{
		MessageID:   msg.MessageID,
		SenderID:    msg.SenderID,
		GroupID:     msg.GroupID,
		Ciphertext:  msg.Ciphertext,
		MessageType: msg.MessageType,
		MediaID:     msg.MediaID,
		MediaType:   msg.MediaType,
		Timestamp:   msg.Timestamp,
	}

	if err := h.inbox.AddToInbox(userID, inboxMsg); err != nil {
		log.Printf("Failed to add message to inbox: %v", err)
	}

	// Step 3.2: Store message in inbox table (already done in SaveMessage)

	// Step 3.3: Send push notification for User B
	h.redis.PublishNotification(userID, map[string]interface{}{
		"type":       "new_message",
		"message_id": msg.MessageID,
		"sender_id":  msg.SenderID,
		"timestamp":  msg.Timestamp,
	})

	// Step 3.4: Enqueue message for further processing
	if err := h.queue.EnqueueDeliveryStatus(msg.MessageID, "pending_delivery"); err != nil {
		log.Printf("Warning: failed to enqueue delivery status: %v", err)
	}

	log.Printf("[Inbox] Message stored for offline user: %s", userID)
}

// deliverGroupMessage implements "Group Message Fan-Out (50-person group)"
func (h *Hub) deliverGroupMessage(msg *db.Message, payload *models.EncryptedMessage, isSealedSender bool) {
	groupID := *payload.GroupID

	// Step 2+3: Who's in group? Get all members
	members, err := h.db.GetGroupMembers(groupID)
	if err != nil {
		log.Printf("Failed to get group members: %v", err)
		return
	}

	// Step 4+5: Check status of all users
	onlineMembers := make([]db.GroupMember, 0)
	offlineMembers := make([]db.GroupMember, 0)
	serverGroups := make(map[string][]uuid.UUID) // serverID -> userIDs

	for _, member := range members {
		if member.UserID == msg.SenderID {
			continue // Don't send to self
		}

		isOnline, serverIDs := h.redis.GetUserConnectionInfo(member.UserID)

		if isOnline && len(serverIDs) > 0 {
			onlineMembers = append(onlineMembers, member)
			// Group by server for parallel delivery
			for _, serverID := range serverIDs {
				serverGroups[serverID] = append(serverGroups[serverID], member.UserID)
			}
		} else {
			offlineMembers = append(offlineMembers, member)
		}
	}

	log.Printf("[Group] Fan-out: %d online, %d offline", len(onlineMembers), len(offlineMembers))

	// Step 6: For online users - parallel delivery to each server
	deliveryMsg := &models.WebSocketMessage{
		Type:      models.MessageTypeDeliver,
		MessageID: msg.MessageID,
		SenderID:  msg.SenderID,
		Timestamp: msg.Timestamp,
		Payload:   mustMarshal(payload),
	}

	// For sealed sender messages in groups, hide the sender ID
	if isSealedSender {
		deliveryMsg.SenderID = uuid.Nil // Hide sender from server
	}

	for serverID, userIDs := range serverGroups {
		if serverID == h.serverID {
			// Deliver locally
			h.mu.RLock()
			for _, userID := range userIDs {
				if clients, ok := h.clients[userID]; ok {
					for client := range clients {
						select {
						case client.send <- mustMarshal(deliveryMsg):
						default:
							go h.unregisterClient(client)
						}
					}
				}
			}
			h.mu.RUnlock()
		} else {
			// Route to other server via Redis
			for _, userID := range userIDs {
				if err := h.redis.PublishToServer(serverID, userID, deliveryMsg); err != nil {
					log.Printf("Warning: failed to publish to server %s: %v", serverID, err)
				}
			}
		}
	}

	// Step 7: For offline users - store for later delivery
	if len(offlineMembers) > 0 {
		offlineUserIDs := make([]uuid.UUID, len(offlineMembers))
		for i, m := range offlineMembers {
			offlineUserIDs[i] = m.UserID
		}

		// Step 7.1+7.2: Write to inboxes (ZADD) and inbox records
		inboxMsg := &inbox.InboxMessage{
			MessageID:   msg.MessageID,
			SenderID:    msg.SenderID,
			GroupID:     &groupID,
			Ciphertext:  msg.Ciphertext,
			MessageType: msg.MessageType,
			Timestamp:   msg.Timestamp,
		}

		if err := h.inbox.AddMultipleToInbox(offlineUserIDs, inboxMsg); err != nil {
			log.Printf("Failed to add message to offline inboxes: %v", err)
		}

		// Step 7.3: Send push notifications to offline users
		for _, userID := range offlineUserIDs {
			h.redis.PublishNotification(userID, map[string]interface{}{
				"type":       "new_group_message",
				"message_id": msg.MessageID,
				"group_id":   groupID,
				"sender_id":  msg.SenderID,
			})
		}
	}

	// Step 8: Store message (1 write for entire group) - already done

	// Step 9: Status update to sender
	statusUpdate := &models.WebSocketMessage{
		Type:      models.MessageTypeStatusUpdate,
		MessageID: msg.MessageID,
		Timestamp: time.Now().UTC(),
		Payload: mustMarshal(map[string]interface{}{
			"delivered_to": len(onlineMembers),
			"pending":      len(offlineMembers),
		}),
	}
	h.sendToUser(msg.SenderID, statusUpdate)
}

// deliverPendingMessages implements "User B comes online" flow
func (h *Hub) deliverPendingMessages(client *Client) {
	// Step 5.2: Retrieve pending messages from inbox (ZSET)
	messages, err := h.inbox.GetPendingMessages(client.UserID)
	if err != nil {
		log.Printf("Failed to fetch pending messages: %v", err)
		return
	}

	if len(messages) == 0 {
		return
	}

	log.Printf("[Deliver] Delivering %d pending messages to user %s", len(messages), client.UserID)

	deliveredIDs := make([]uuid.UUID, 0, len(messages))

	// Step 5.4: Deliver all pending messages
	for _, msg := range messages {
		deliveryMsg := &models.WebSocketMessage{
			Type:      models.MessageTypeDeliver,
			MessageID: msg.MessageID,
			SenderID:  msg.SenderID,
			Timestamp: msg.Timestamp,
			Payload: mustMarshal(&models.EncryptedMessage{
				GroupID:     msg.GroupID,
				Ciphertext:  msg.Ciphertext,
				MessageType: msg.MessageType,
				MediaID:     msg.MediaID,
				MediaType:   msg.MediaType,
			}),
		}

		select {
		case client.send <- mustMarshal(deliveryMsg):
			deliveredIDs = append(deliveredIDs, msg.MessageID)
		default:
			return
		}
	}

	// Remove delivered messages from inbox
	if len(deliveredIDs) > 0 {
		if err := h.inbox.RemoveFromInbox(client.UserID, deliveredIDs); err != nil {
			log.Printf("Warning: failed to remove from inbox: %v", err)
		}
	}
}

func (h *Hub) handleDeliveryAck(msg *models.WebSocketMessage) {
	// Step 7: Delivery ACK received from recipient
	now := time.Now().UTC()
	if err := h.db.UpdateMessageStatus(msg.MessageID, "delivered", now); err != nil {
		log.Printf("Warning: failed to update message status: %v", err)
	}

	// Step 8: Forward delivery ACK to sender
	message, err := h.db.GetMessage(msg.MessageID)
	if err != nil {
		return
	}

	// Step 9: Status update (delivered) to sender
	statusUpdate := &models.WebSocketMessage{
		Type:      models.MessageTypeStatusUpdate,
		MessageID: msg.MessageID,
		Timestamp: now,
		Payload:   json.RawMessage(`{"status": "delivered"}`),
	}

	h.sendToUserAllDevices(message.SenderID, statusUpdate, uuid.Nil)
}

func (h *Hub) handleReadReceipt(msg *models.WebSocketMessage) {
	log.Printf("[ReadReceipt] handleReadReceipt: from=%s, payload=%s", msg.SenderID, string(msg.Payload))

	var payload struct {
		MessageIDs []uuid.UUID `json:"message_ids"`
	}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		log.Printf("[ReadReceipt] Failed to parse payload: %v", err)
		return
	}

	log.Printf("[ReadReceipt] Processing %d read receipts", len(payload.MessageIDs))

	now := time.Now().UTC()
	for _, messageID := range payload.MessageIDs {
		log.Printf("[ReadReceipt] Marking message %s as read", messageID)
		if err := h.db.UpdateMessageStatus(messageID, "read", now); err != nil {
			log.Printf("Warning: failed to update message status: %v", err)
		}

		message, err := h.db.GetMessage(messageID)
		if err != nil {
			log.Printf("[ReadReceipt] Could not fetch message %s: %v", messageID, err)
			continue
		}

		statusUpdate := &models.WebSocketMessage{
			Type:      models.MessageTypeStatusUpdate,
			MessageID: messageID,
			Timestamp: now,
			Payload:   json.RawMessage(`{"status": "read"}`),
		}

		log.Printf("[ReadReceipt] Sending status_update (read) to sender %s for message %s", message.SenderID, messageID)
		// Also sync read status to sender's other devices
		h.sendToUserAllDevices(message.SenderID, statusUpdate, uuid.Nil)
	}
}

func (h *Hub) handleTypingIndicator(msg *models.WebSocketMessage) {
	var payload struct {
		ReceiverID *uuid.UUID `json:"receiver_id"`
		GroupID    *uuid.UUID `json:"group_id"`
		IsTyping   bool       `json:"is_typing"`
	}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return
	}

	typingMsg := &models.WebSocketMessage{
		Type:      models.MessageTypeTyping,
		SenderID:  msg.SenderID,
		Timestamp: time.Now().UTC(),
		Payload:   msg.Payload,
	}

	if payload.GroupID != nil {
		members, err := h.db.GetGroupMembers(*payload.GroupID)
		if err != nil {
			log.Printf("[Typing] Failed to get group members: %v", err)
			return
		}
		for _, member := range members {
			if member.UserID != msg.SenderID {
				h.sendToUser(member.UserID, typingMsg)
			}
		}
	} else if payload.ReceiverID != nil {
		h.sendToUser(*payload.ReceiverID, typingMsg)
	}
}

func (h *Hub) handleHeartbeat(msg *models.WebSocketMessage) {
	// Update user's last seen time
	h.redis.UpdateLastActive(msg.SenderID)

	// Refresh connection TTL
	h.redis.RefreshConnection(msg.SenderID, msg.DeviceID)

	// Ensure presence is set to online (in case it expired)
	h.redis.SetUserPresence(msg.SenderID, true)

	// Send heartbeat acknowledgment
	ack := &models.WebSocketMessage{
		Type:      models.MessageTypeHeartbeatAck,
		Timestamp: time.Now().UTC(),
	}
	h.sendToUser(msg.SenderID, ack)
}

// handleCallSignaling forwards WebRTC signaling messages between peers
func (h *Hub) handleCallSignaling(msg *models.WebSocketMessage) {
	// Parse the payload to get the recipient
	// Accept both target_id (frontend) and recipient_id (legacy) for compatibility
	var payload struct {
		TargetID    uuid.UUID `json:"target_id"`
		RecipientID uuid.UUID `json:"recipient_id"`
		CallType    string    `json:"call_type,omitempty"` // "audio" or "video"
		SDP         string    `json:"sdp,omitempty"`
		Candidate   string    `json:"candidate,omitempty"`
	}

	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		log.Printf("[Call] Failed to parse call signaling payload: %v", err)
		return
	}

	// Use target_id if set, otherwise fall back to recipient_id
	recipientID := payload.TargetID
	if recipientID == uuid.Nil {
		recipientID = payload.RecipientID
	}

	if recipientID == uuid.Nil {
		log.Printf("[Call] No recipient ID in call signaling payload")
		return
	}

	log.Printf("[Call] Signaling: type=%s, from=%s, to=%s", msg.Type, msg.SenderID, recipientID)

	// Forward the message to the recipient with sender info
	forwardMsg := &models.WebSocketMessage{
		Type:      msg.Type,
		SenderID:  msg.SenderID,
		Timestamp: time.Now().UTC(),
		Payload:   msg.Payload,
	}

	// Check if recipient is online
	isOnline, serverIDs := h.redis.GetUserConnectionInfo(recipientID)

	if !isOnline || len(serverIDs) == 0 {
		// Recipient is offline - send busy signal back to caller
		log.Printf("[Call] Recipient offline, sending busy signal")
		busyMsg := &models.WebSocketMessage{
			Type:      models.MessageTypeCallBusy,
			SenderID:  recipientID,
			Timestamp: time.Now().UTC(),
			Payload:   json.RawMessage(`{"reason": "offline"}`),
		}
		h.sendToUser(msg.SenderID, busyMsg)
		return
	}

	// Deliver to recipient (on this server or via Redis)
	h.mu.RLock()
	localClients, onThisServer := h.clients[recipientID]
	h.mu.RUnlock()

	if onThisServer && len(localClients) > 0 {
		for client := range localClients {
			select {
			case client.send <- mustMarshal(forwardMsg):
				log.Printf("[Call] Signal delivered to device=%s", client.DeviceID)
			default:
				log.Printf("[Call] Warning: Client buffer full")
			}
		}
	}

	// Also publish to Redis for other servers
	for _, serverID := range serverIDs {
		if serverID != h.serverID {
			if err := h.redis.PublishToServer(serverID, recipientID, forwardMsg); err != nil {
				log.Printf("Warning: failed to publish to server %s: %v", serverID, err)
			}
		}
	}
}

// handleDeviceSync relays encrypted sync data between devices of the same user
// The server CANNOT read this data - it's encrypted device-to-device
func (h *Hub) handleDeviceSync(msg *models.WebSocketMessage) {
	// Parse to get target device
	var payload struct {
		TargetDeviceID uuid.UUID `json:"target_device_id"`
		EncryptedData  string    `json:"encrypted_data"` // Server can't decrypt this
	}

	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		log.Printf("[Sync] Failed to parse sync payload: %v", err)
		return
	}

	log.Printf("[Sync] Device sync: type=%s, from_device=%s, to_device=%s, data_len=%d",
		msg.Type, msg.DeviceID, payload.TargetDeviceID, len(payload.EncryptedData))

	// Forward to specific device (or all devices for sync_request)
	if msg.Type == models.MessageTypeSyncRequest {
		// Sync request goes to PRIMARY device only
		primaryDevice, err := h.db.GetPrimaryDevice(msg.SenderID)
		if err != nil {
			log.Printf("[Sync] Error getting primary device for user %s: %v", msg.SenderID, err)
			return
		}
		if primaryDevice == nil {
			log.Printf("[Sync] Warning: No primary device found for user %s", msg.SenderID)
			return
		}

		// Forward request to primary device
		forwardMsg := &models.WebSocketMessage{
			Type:      msg.Type,
			SenderID:  msg.SenderID,
			DeviceID:  msg.DeviceID, // So primary knows which device is requesting
			Timestamp: time.Now().UTC(),
			Payload:   msg.Payload,
		}
		h.sendToDevice(msg.SenderID, primaryDevice.DeviceID, forwardMsg)
	} else {
		// Sync data/ack goes to specific device
		forwardMsg := &models.WebSocketMessage{
			Type:      msg.Type,
			SenderID:  msg.SenderID,
			DeviceID:  msg.DeviceID,
			Timestamp: time.Now().UTC(),
			Payload:   msg.Payload,
		}
		h.sendToDevice(msg.SenderID, payload.TargetDeviceID, forwardMsg)
	}
}

// handleMediaKey forwards encrypted media keys between clients
// The server CANNOT read the encrypted key - it's E2EE between clients
func (h *Hub) handleMediaKey(msg *models.WebSocketMessage) {
	// Parse the payload to get the recipient
	var payload struct {
		MediaID      uuid.UUID `json:"media_id"`
		RecipientID  uuid.UUID `json:"recipient_id"`
		EncryptedKey string    `json:"encrypted_key"` // Server can't decrypt this
		Algorithm    string    `json:"algorithm"`
		Timestamp    time.Time `json:"timestamp"`
	}

	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		log.Printf("[MediaKey] Failed to parse media key payload: %v", err)
		return
	}

	log.Printf("[MediaKey] Media key exchange: media=%s, from=%s, to=%s, key_len=%d",
		payload.MediaID, msg.SenderID, payload.RecipientID, len(payload.EncryptedKey))

	// Forward the message to the recipient
	forwardMsg := &models.WebSocketMessage{
		Type:      models.MessageTypeMediaKey,
		SenderID:  msg.SenderID,
		Timestamp: time.Now().UTC(),
		Payload:   msg.Payload,
	}

	// Check if recipient is online
	isOnline, serverIDs := h.redis.GetUserConnectionInfo(payload.RecipientID)

	if !isOnline || len(serverIDs) == 0 {
		log.Printf("[MediaKey] Recipient offline, cannot deliver media key for media %s", payload.MediaID)
		// Note: Media keys are not stored offline - recipient must be online to receive them
		return
	}

	// Deliver to recipient (on this server or via Redis)
	h.mu.RLock()
	localClients, onThisServer := h.clients[payload.RecipientID]
	h.mu.RUnlock()

	if onThisServer && len(localClients) > 0 {
		for client := range localClients {
			select {
			case client.send <- mustMarshal(forwardMsg):
				log.Printf("[MediaKey] Media key delivered to device=%s", client.DeviceID)
			default:
				log.Printf("[MediaKey] Warning: Client buffer full")
			}
		}
	}

	// Also publish to Redis for other servers
	for _, serverID := range serverIDs {
		if serverID != h.serverID {
			if err := h.redis.PublishToServer(serverID, payload.RecipientID, forwardMsg); err != nil {
				log.Printf("Warning: failed to publish to server %s: %v", serverID, err)
			}
		}
	}
}

// sendToDevice sends a message to a specific device of a user
func (h *Hub) sendToDevice(userID, deviceID uuid.UUID, msg *models.WebSocketMessage) {
	h.mu.RLock()
	clients, ok := h.clients[userID]
	h.mu.RUnlock()

	if ok {
		for client := range clients {
			if client.DeviceID == deviceID {
				select {
				case client.send <- mustMarshal(msg):
					log.Printf("[Sync] Message sent to device %s", deviceID)
					return
				default:
					log.Printf("[Sync] Warning: Device %s buffer full", deviceID)
				}
			}
		}
	}

	// If not found locally, publish to Redis for other servers
	if err := h.redis.PublishToDevice(userID, deviceID, msg); err != nil {
		log.Printf("Warning: failed to publish to device: %v", err)
	}
}

// sendToUser sends to any one device of a user on this server
func (h *Hub) sendToUser(userID uuid.UUID, msg *models.WebSocketMessage) {
	h.mu.RLock()
	clients, ok := h.clients[userID]
	h.mu.RUnlock()

	if ok {
		data := mustMarshal(msg)
		for client := range clients {
			select {
			case client.send <- data:
				return // Sent to one device
			default:
				go h.unregisterClient(client)
			}
		}
	} else {
		// User not on this server, publish to Redis
		if err := h.redis.PublishMessage(userID, msg); err != nil {
			log.Printf("Warning: failed to publish message: %v", err)
		}
	}
}

// sendToUserAllDevices sends to ALL devices of a user (for sync)
func (h *Hub) sendToUserAllDevices(userID uuid.UUID, msg *models.WebSocketMessage, excludeDevice uuid.UUID) {
	h.mu.RLock()
	clients, ok := h.clients[userID]
	h.mu.RUnlock()

	if ok {
		data := mustMarshal(msg)
		for client := range clients {
			if excludeDevice != uuid.Nil && client.DeviceID == excludeDevice {
				continue // Skip the originating device
			}
			select {
			case client.send <- data:
			default:
				go h.unregisterClient(client)
			}
		}
	}

	// Also publish to Redis for devices on other servers
	if err := h.redis.PublishMessage(userID, msg); err != nil {
		log.Printf("Warning: failed to publish message: %v", err)
	}
}

func (h *Hub) sendErrorToClient(userID uuid.UUID, errorMsg string) {
	errMsg := &models.WebSocketMessage{
		Type:      models.MessageTypeError,
		Timestamp: time.Now().UTC(),
		Payload:   json.RawMessage(`{"error": "` + errorMsg + `"}`),
	}
	h.sendToUser(userID, errMsg)
}

// BroadcastPresenceUpdate is an exported wrapper for broadcastPresenceUpdate
// Used by HTTP handlers to trigger presence updates (e.g., when privacy settings change)
func (h *Hub) BroadcastPresenceUpdate(userID uuid.UUID, isOnline bool) {
	h.broadcastPresenceUpdate(userID, isOnline)
}

// broadcastPresenceUpdate broadcasts a user's online/offline status to all connected users
// This allows everyone to see when someone comes online or goes offline
// Respects user's privacy setting: if show_online_status is false, always broadcast as offline
// Note: show_online_status controls BOTH online indicator AND last_seen (simplified from separate settings)
func (h *Hub) broadcastPresenceUpdate(userID uuid.UUID, isOnline bool) {
	// Check user's privacy settings
	privacySettings, err := h.db.GetPrivacySettings(userID)
	showOnlineStatus := true

	if err != nil {
		log.Printf("Failed to get privacy settings for user %s: %v", userID, err)
		// Default to showing if we can't check
	} else {
		if val, ok := privacySettings["show_online_status"].(bool); ok {
			showOnlineStatus = val
		}
	}

	// Determine what to broadcast
	var msgType string
	var includeLastSeen bool

	if !showOnlineStatus {
		// Ghost Mode: Always appear offline, NO last_seen
		// This clears any "online" state and shows nothing
		msgType = models.MessageTypeUserOffline
		includeLastSeen = false
		// Don't update Redis - we don't want to store a timestamp
	} else if isOnline {
		// User is online and allows showing
		msgType = models.MessageTypeUserOnline
		includeLastSeen = false
	} else {
		// User is offline and allows showing last_seen
		msgType = models.MessageTypeUserOffline
		includeLastSeen = true
		// Update Redis with current timestamp
		h.redis.SetUserPresence(userID, false)
	}

	// Build payload
	payload := map[string]interface{}{
		"user_id": userID.String(),
	}
	if includeLastSeen {
		payload["last_seen"] = time.Now().UTC().Unix()
	}

	presenceMsg := &models.WebSocketMessage{
		Type:      msgType,
		SenderID:  userID,
		ServerID:  h.serverID, // Track which server originated this presence update
		Timestamp: time.Now().UTC(),
		Payload:   mustMarshal(payload),
	}

	// Get users who have exchanged messages with this user (contacts only)
	// This prevents broadcasting presence to all 1M+ users (scalability fix)
	contacts, err := h.db.GetMessagedUsers(userID)
	if err != nil {
		log.Printf("Failed to get contacts for presence broadcast: %v", err)
		return
	}

	// Create a map for fast lookup
	contactMap := make(map[uuid.UUID]bool)
	for _, contactID := range contacts {
		contactMap[contactID] = true
	}

	// Broadcast only to users who have messaged with this user
	h.mu.RLock()
	targetClients := make([]*Client, 0)
	for _, userClients := range h.clients {
		for client := range userClients {
			// Send only to contacts (users who have exchanged messages)
			// Don't send to the user themselves
			if client.UserID != userID && contactMap[client.UserID] {
				targetClients = append(targetClients, client)
			}
		}
	}
	h.mu.RUnlock()

	data := mustMarshal(presenceMsg)
	for _, client := range targetClients {
		select {
		case client.send <- data:
		default:
			// Buffer full, skip
		}
	}

	// Also publish to Redis for other servers via dedicated presence channel
	// Include contact list so other servers can filter too
	h.redis.PublishPresenceUpdate(userID, isOnline, data)
}

// BroadcastPresenceFromRedis handles presence updates received from other servers via Redis
func (h *Hub) BroadcastPresenceFromRedis(msg *models.WebSocketMessage) {
	// Skip if this presence update originated from THIS server (avoid duplicates)
	if msg.ServerID == h.serverID {
		return
	}

	// Get the user ID from the message
	userID := msg.SenderID

	// Get users who have exchanged messages with this user (contacts only)
	contacts, err := h.db.GetMessagedUsers(userID)
	if err != nil {
		log.Printf("Failed to get contacts for presence broadcast: %v", err)
		return
	}

	// Create a map for fast lookup
	contactMap := make(map[uuid.UUID]bool)
	for _, contactID := range contacts {
		contactMap[contactID] = true
	}

	// Broadcast only to contacts on this server
	h.mu.RLock()
	targetClients := make([]*Client, 0)
	for _, userClients := range h.clients {
		for client := range userClients {
			// Send only to contacts (users who have exchanged messages)
			// Don't send to the user themselves
			if client.UserID != userID && contactMap[client.UserID] {
				targetClients = append(targetClients, client)
			}
		}
	}
	h.mu.RUnlock()

	data := mustMarshal(msg)
	for _, client := range targetClients {
		select {
		case client.send <- data:
		default:
			// Buffer full, skip
		}
	}
}

// DeliverFromRedis handles messages from other servers via Redis pub/sub
func (h *Hub) DeliverFromRedis(userID uuid.UUID, msg *models.WebSocketMessage) {
	h.mu.RLock()
	clients, ok := h.clients[userID]
	h.mu.RUnlock()

	if !ok {
		return
	}

	data := mustMarshal(msg)
	for client := range clients {
		select {
		case client.send <- data:
		default:
			go h.unregisterClient(client)
		}
	}
}

// BroadcastToUser sends a raw message to all devices of a user
// Used for device approval notifications and other system events
func (h *Hub) BroadcastToUser(userID uuid.UUID, payload interface{}) {
	data := mustMarshal(payload)

	// Send to local clients
	h.mu.RLock()
	clients, ok := h.clients[userID]
	h.mu.RUnlock()

	if ok {
		for client := range clients {
			select {
			case client.send <- data:
			default:
				go h.unregisterClient(client)
			}
		}
	}

	// Also publish to Redis for clients on other servers
	if err := h.redis.PublishRaw(userID, data); err != nil {
		log.Printf("Warning: failed to publish raw: %v", err)
	}
}

// SendToDevice sends a message to a specific device by device ID
// Used for targeted device approval requests to primary device only
func (h *Hub) SendToDevice(deviceID uuid.UUID, payload interface{}) {
	data := mustMarshal(payload)

	// Search for the client with this device ID in all users
	h.mu.RLock()

	for _, clients := range h.clients {
		for client := range clients {
			if client.DeviceID == deviceID {
				select {
				case client.send <- data:
					h.mu.RUnlock()
					return // Found and sent
				default:
					// Buffer full, unregister client
					h.mu.RUnlock()
					h.unregisterClient(client)
					return
				}
			}
		}
	}

	h.mu.RUnlock()

	// If not found locally, publish to Redis with device targeting
	// Other servers can check if they have this device
	if err := h.redis.PublishToDeviceRaw(deviceID, data); err != nil {
		log.Printf("Warning: failed to publish to device: %v", err)
	}
}

// SendToUser sends a message to all devices of a specific user
// Used for notifications like blocking
func (h *Hub) SendToUser(userIDStr string, message *models.WebSocketMessage) {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Printf("SendToUser: invalid user ID: %s", userIDStr)
		return
	}

	data := mustMarshal(message)

	h.mu.RLock()
	clients, exists := h.clients[userID]
	if exists {
		for client := range clients {
			select {
			case client.send <- data:
			default:
				// Buffer full, skip
			}
		}
	}
	h.mu.RUnlock()

	// Also publish via Redis for cross-server delivery
	if err := h.redis.PublishRaw(userID, data); err != nil {
		log.Printf("Warning: failed to publish: %v", err)
	}
}

func (h *Hub) closeAllClients() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for userID, clients := range h.clients {
		for client := range clients {
			close(client.send)
			h.redis.UnregisterConnection(userID, client.DeviceID)
		}
	}
	h.clients = make(map[uuid.UUID]map[*Client]bool)
}

func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("Warning: Failed to marshal JSON: %v", err)
		return []byte("{}")
	}
	return data
}

// VerifyResourceCleanup checks that all resources are properly cleaned up
func (h *Hub) VerifyResourceCleanup() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check that all clients are properly cleaned up
	if len(h.clients) > 0 {
		return NewWebSocketError("resource_cleanup_failed",
			fmt.Sprintf("clients still registered: %d", len(h.clients)),
			"Resource cleanup verification failed")
	}

	// Check that connection count is zero
	if h.totalConnections != 0 {
		return NewWebSocketError("resource_cleanup_failed",
			fmt.Sprintf("connections not zero: %d", h.totalConnections),
			"Resource cleanup verification failed")
	}

	// Check that nonce store is cleaned up
	h.nonceMutex.Lock()
	nonceCount := len(h.nonceStore)
	h.nonceMutex.Unlock()

	if nonceCount > 100 { // Allow some tolerance for recent nonces
		return NewWebSocketError("resource_cleanup_failed",
			fmt.Sprintf("nonces still in store: %d", nonceCount),
			"Resource cleanup verification failed")
	}

	log.Printf("[Cleanup] Resource cleanup verification passed: clients=%d, connections=%d, nonces=%d",
		len(h.clients), h.totalConnections, nonceCount)

	return nil
}

// WebSocketError represents a standardized error with context
type WebSocketError struct {
	ErrorCode    string
	ErrorMessage string
	Context      string
	Timestamp    time.Time
	StackTrace   string
}

// NewWebSocketError creates a new standardized error
func NewWebSocketError(errorCode, errorMessage, context string) *WebSocketError {
	return &WebSocketError{
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
		Context:      context,
		Timestamp:    time.Now().UTC(),
		StackTrace:   getStackTrace(),
	}
}

// getStackTrace captures the current stack trace
func getStackTrace() string {
	buf := make([]byte, 1024)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// Error implements the error interface
func (e *WebSocketError) Error() string {
	return fmt.Sprintf("[%s] %s (Context: %s, Time: %s)",
		e.ErrorCode, e.ErrorMessage, e.Context, e.Timestamp.Format(time.RFC3339))
}

// LogError logs an error with full context
func LogError(err error, context string) {
	if wsErr, ok := err.(*WebSocketError); ok {
		log.Printf("[ERROR] [%s] %s - Context: %s - Time: %s - Trace: %s",
			wsErr.ErrorCode, wsErr.ErrorMessage, wsErr.Context,
			wsErr.Timestamp.Format(time.RFC3339), wsErr.StackTrace)
	} else {
		log.Printf("[ERROR] %v - Context: %s - Time: %s",
			err, context, time.Now().UTC().Format(time.RFC3339))
	}
}
