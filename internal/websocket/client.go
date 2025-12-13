package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jaydenbeard/messaging-app/internal/models"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer (10MB for media references)
	maxMessageSize = 10 * 1024 * 1024
)

// Client represents a single WebSocket connection
type Client struct {
	hub *Hub

	// The WebSocket connection
	conn *websocket.Conn

	// Buffered channel of outbound messages
	send chan []byte

	// User information
	UserID   uuid.UUID
	DeviceID uuid.UUID

	// Authentication token for HMAC verification
	authToken string

	// Rate limiting (token bucket algorithm)
	messageTokens int
	lastRefill    time.Time
	tokenMu       sync.Mutex
}

// NewClient creates a new Client instance
func NewClient(hub *Hub, conn *websocket.Conn, userID, deviceID uuid.UUID, authToken string) *Client {
	return &Client{
		hub:           hub,
		conn:          conn,
		send:          make(chan []byte, 100), // Reduced buffer size for better backpressure
		UserID:        userID,
		DeviceID:      deviceID,
		authToken:     authToken,
		messageTokens: 200, // Start with 200 tokens (full burst capacity)
		lastRefill:    time.Now(),
	}
}

// canSendMessage checks if client can send a message (rate limiting)
// Rate limit: 50 messages/second with burst capacity of 200
// This is generous enough for rapid ICE candidates during calls while still preventing abuse
func (c *Client) canSendMessage() bool {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	// Refill tokens (50 messages per second for call signaling support)
	now := time.Now()
	elapsed := now.Sub(c.lastRefill)
	tokensToAdd := int(elapsed.Seconds() * 50) // 50 tokens/sec

	if tokensToAdd > 0 {
		c.messageTokens = min(c.messageTokens+tokensToAdd, 200) // max 200 tokens burst
		c.lastRefill = now
	}

	if c.messageTokens > 0 {
		c.messageTokens--
		return true
	}

	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		if err := c.conn.Close(); err != nil {
			log.Printf("Warning: failed to close WebSocket connection: %v", err)
		}
	}()

	c.conn.SetReadLimit(maxMessageSize)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Printf("Warning: failed to set read deadline: %v", err)
	}
	c.conn.SetPongHandler(func(string) error {
		if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			return err
		}
		return nil
	})

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Parse the incoming message
		var msg models.WebSocketMessage
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			log.Printf("Failed to parse WebSocket message: %v", err)
			log.Printf("Raw message (first 300 chars): %s", string(messageBytes[:min(300, len(messageBytes))]))
			continue
		}

		// DEBUG: Log incoming message
		log.Printf("[WS DEBUG] Received message type=%s from user=%s", msg.Type, c.UserID)

		// SECURITY: Rate limit check
		if !c.canSendMessage() {
			// Send rate limit error to client
			errorMsg := mustMarshal(map[string]interface{}{
				"type":    "error",
				"message": "Rate limit exceeded. Please slow down.",
			})
			select {
			case c.send <- errorMsg:
			default:
			}
			continue
		}

		// Set sender information
		msg.SenderID = c.UserID
		msg.DeviceID = c.DeviceID

		// Send to hub for processing
		c.hub.Broadcast(&msg)
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		if err := c.conn.Close(); err != nil {
			log.Printf("Warning: failed to close WebSocket connection: %v", err)
		}
	}()

	for {
		select {
		case message, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Printf("Warning: failed to set write deadline: %v", err)
			}
			if !ok {
				// Hub closed the channel
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					log.Printf("Warning: failed to write close message: %v", err)
				}
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if _, err := w.Write(message); err != nil {
				log.Printf("WebSocket write error for client %s: %v", c.DeviceID, err)
				if closeErr := w.Close(); closeErr != nil {
					log.Printf("Warning: failed to close writer: %v", closeErr)
				}
				return
			}

			// Add queued messages to the current WebSocket frame with backpressure
			n := len(c.send)
			processed := 0
			for i := 0; i < n; i++ {
				// Check buffer capacity before processing more messages
				if len(c.send) > 50 { // Backpressure threshold
					log.Printf("Backpressure: Client buffer full (%d messages), slowing down", len(c.send))
					time.Sleep(10 * time.Millisecond) // Slow down processing
				}

				select {
				case nextMessage := <-c.send:
					if _, err := w.Write([]byte{'\n'}); err != nil {
						log.Printf("WebSocket write error for client %s: %v", c.DeviceID, err)
						if closeErr := w.Close(); closeErr != nil {
							log.Printf("Warning: failed to close writer: %v", closeErr)
						}
						return
					}
					if _, err := w.Write(nextMessage); err != nil {
						log.Printf("WebSocket write error for client %s: %v", c.DeviceID, err)
						if closeErr := w.Close(); closeErr != nil {
							log.Printf("Warning: failed to close writer: %v", closeErr)
						}
						return
					}
					processed++
				default:
					// Buffer is empty, break out of the for loop
					goto endProcessing
				}
			}
		endProcessing:

			if err := w.Close(); err != nil {
				return
			}

			// Log if we're falling behind
			if processed > 20 {
				log.Printf("Client %s processing %d queued messages, buffer size: %d",
					c.DeviceID, processed, len(c.send))
			}

		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Printf("Warning: failed to set write deadline: %v", err)
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
