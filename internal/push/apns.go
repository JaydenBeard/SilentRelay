package push

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// APNs endpoints
const (
	APNsProductionURL = "https://api.push.apple.com"
	APNsSandboxURL    = "https://api.sandbox.push.apple.com"
)

// NotificationType for different notification categories
type NotificationType string

const (
	NotificationTypeMessage        NotificationType = "new_message"
	NotificationTypeMessageRequest NotificationType = "message_request"
	NotificationTypeFriendRequest  NotificationType = "friend_request"
	NotificationTypeFriendAccepted NotificationType = "friend_accepted"
	NotificationTypeIncomingCall   NotificationType = "incoming_call"
	NotificationTypeMissedCall     NotificationType = "missed_call"
)

// APNsConfig holds the configuration for APNs
type APNsConfig struct {
	KeyPath    string
	KeyID      string
	TeamID     string
	BundleID   string
	Production bool
}

// APNsClient handles sending push notifications to iOS devices
type APNsClient struct {
	config     APNsConfig
	privateKey *ecdsa.PrivateKey
	httpClient *http.Client

	// JWT token caching (valid for 1 hour)
	token       string
	tokenExpiry time.Time
	tokenMu     sync.RWMutex
}

// Notification represents a push notification payload
type Notification struct {
	DeviceToken string
	Title       string
	Body        string
	Sound       string
	Badge       int
	Category    string
	ThreadID    string
	Data        map[string]interface{}
	Priority    int    // 5 for normal, 10 for immediate
	PushType    string // "alert", "background", "voip"
}

// NewAPNsClient creates a new APNs client
func NewAPNsClient(config APNsConfig) (*APNsClient, error) {
	// Load the private key
	keyData, err := os.ReadFile(config.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read APNs key file: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from APNs key")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse APNs private key: %w", err)
	}

	ecdsaKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("APNs key is not an ECDSA key")
	}

	return &APNsClient{
		config:     config,
		privateKey: ecdsaKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// getToken returns a valid JWT token for APNs authentication
func (c *APNsClient) getToken() (string, error) {
	c.tokenMu.RLock()
	if c.token != "" && time.Now().Before(c.tokenExpiry) {
		token := c.token
		c.tokenMu.RUnlock()
		return token, nil
	}
	c.tokenMu.RUnlock()

	// Generate new token
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	// Double-check after acquiring write lock
	if c.token != "" && time.Now().Before(c.tokenExpiry) {
		return c.token, nil
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss": c.config.TeamID,
		"iat": now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = c.config.KeyID

	signedToken, err := token.SignedString(c.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT token: %w", err)
	}

	c.token = signedToken
	c.tokenExpiry = now.Add(50 * time.Minute) // Refresh 10 min before expiry

	return c.token, nil
}

// getAPNsURL returns the appropriate APNs endpoint
func (c *APNsClient) getAPNsURL() string {
	if c.config.Production {
		return APNsProductionURL
	}
	return APNsSandboxURL
}

// Send sends a push notification to an iOS device
func (c *APNsClient) Send(notification *Notification) error {
	token, err := c.getToken()
	if err != nil {
		return err
	}

	// Build the payload
	payload := c.buildPayload(notification)

	url := fmt.Sprintf("%s/3/device/%s", c.getAPNsURL(), notification.DeviceToken)

	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apns-topic", c.config.BundleID)
	req.Header.Set("apns-push-type", notification.PushType)

	if notification.Priority > 0 {
		req.Header.Set("apns-priority", fmt.Sprintf("%d", notification.Priority))
	}

	if notification.ThreadID != "" {
		req.Header.Set("apns-collapse-id", notification.ThreadID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("APNs error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// buildPayload creates the JSON payload for the notification
func (c *APNsClient) buildPayload(n *Notification) io.Reader {
	aps := map[string]interface{}{
		"alert": map[string]string{
			"title": n.Title,
			"body":  n.Body,
		},
	}

	if n.Sound != "" {
		aps["sound"] = n.Sound
	} else {
		aps["sound"] = "default"
	}

	if n.Badge >= 0 {
		aps["badge"] = n.Badge
	}

	if n.Category != "" {
		aps["category"] = n.Category
	}

	if n.ThreadID != "" {
		aps["thread-id"] = n.ThreadID
	}

	payload := map[string]interface{}{
		"aps": aps,
	}

	// Add custom data
	for k, v := range n.Data {
		payload[k] = v
	}

	jsonData, _ := json.Marshal(payload)
	return bytes.NewReader(jsonData)
}

// Helper methods for common notification types

// SendMessageNotification sends a new message notification
func (c *APNsClient) SendMessageNotification(deviceToken, senderName, messagePreview, conversationID string) error {
	return c.Send(&Notification{
		DeviceToken: deviceToken,
		Title:       senderName,
		Body:        messagePreview,
		Sound:       "default",
		Category:    "MESSAGE_CATEGORY",
		ThreadID:    conversationID,
		Priority:    10,
		PushType:    "alert",
		Data: map[string]interface{}{
			"type":            string(NotificationTypeMessage),
			"conversation_id": conversationID,
		},
	})
}

// SendFriendRequestNotification sends a friend request notification
func (c *APNsClient) SendFriendRequestNotification(deviceToken, fromUsername, fromUserID string) error {
	return c.Send(&Notification{
		DeviceToken: deviceToken,
		Title:       "Friend Request",
		Body:        fmt.Sprintf("%s sent you a friend request", fromUsername),
		Sound:       "default",
		Category:    "FRIEND_REQUEST_CATEGORY",
		Priority:    10,
		PushType:    "alert",
		Data: map[string]interface{}{
			"type":    string(NotificationTypeFriendRequest),
			"user_id": fromUserID,
		},
	})
}

// SendIncomingCallNotification sends a VoIP call notification
func (c *APNsClient) SendIncomingCallNotification(deviceToken, callerName, callType, callID string) error {
	return c.Send(&Notification{
		DeviceToken: deviceToken,
		Title:       fmt.Sprintf("Incoming %s Call", callType),
		Body:        callerName,
		Sound:       "default",
		Category:    "CALL_CATEGORY",
		Priority:    10,
		PushType:    "voip",
		Data: map[string]interface{}{
			"type":      string(NotificationTypeIncomingCall),
			"call_id":   callID,
			"call_type": callType,
		},
	})
}
