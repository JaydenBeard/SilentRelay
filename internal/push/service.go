package push

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// DeviceToken represents a registered push notification device
type DeviceToken struct {
	ID        int64     `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	Platform  string    `json:"platform"` // "ios" or "android"
	BundleID  string    `json:"bundle_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DeviceStore handles device token storage
type DeviceStore struct {
	db *sql.DB
}

// NewDeviceStore creates a new device store
func NewDeviceStore(db *sql.DB) *DeviceStore {
	return &DeviceStore{db: db}
}

// CreateTable creates the device_tokens table
func (s *DeviceStore) CreateTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS device_tokens (
			id SERIAL PRIMARY KEY,
			user_id TEXT NOT NULL,
			token TEXT NOT NULL UNIQUE,
			platform TEXT NOT NULL DEFAULT 'ios',
			bundle_id TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_device_tokens_user_id ON device_tokens(user_id);
		CREATE INDEX IF NOT EXISTS idx_device_tokens_token ON device_tokens(token);
	`
	_, err := s.db.ExecContext(ctx, query)
	return err
}

// RegisterToken registers or updates a device token for a user
func (s *DeviceStore) RegisterToken(ctx context.Context, userID, token, platform, bundleID string) error {
	query := `
		INSERT INTO device_tokens (user_id, token, platform, bundle_id, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
		ON CONFLICT (token) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			platform = EXCLUDED.platform,
			bundle_id = EXCLUDED.bundle_id,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := s.db.ExecContext(ctx, query, userID, token, platform, bundleID)
	return err
}

// GetTokensForUser returns all device tokens for a user
func (s *DeviceStore) GetTokensForUser(ctx context.Context, userID string) ([]DeviceToken, error) {
	query := `SELECT id, user_id, token, platform, bundle_id, created_at, updated_at 
			  FROM device_tokens WHERE user_id = $1`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []DeviceToken
	for rows.Next() {
		var t DeviceToken
		var bundleID sql.NullString
		err := rows.Scan(&t.ID, &t.UserID, &t.Token, &t.Platform, &bundleID, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, err
		}
		t.BundleID = bundleID.String
		tokens = append(tokens, t)
	}
	return tokens, nil
}

// RemoveToken removes a specific device token
func (s *DeviceStore) RemoveToken(ctx context.Context, token string) error {
	query := `DELETE FROM device_tokens WHERE token = $1`
	_, err := s.db.ExecContext(ctx, query, token)
	return err
}

// RemoveAllTokensForUser removes all device tokens for a user (e.g., on logout)
func (s *DeviceStore) RemoveAllTokensForUser(ctx context.Context, userID string) error {
	query := `DELETE FROM device_tokens WHERE user_id = $1`
	_, err := s.db.ExecContext(ctx, query, userID)
	return err
}

// PushService coordinates sending push notifications
type PushService struct {
	apns        *APNsClient
	deviceStore *DeviceStore
}

// NewPushService creates a new push service
func NewPushService(apns *APNsClient, deviceStore *DeviceStore) *PushService {
	return &PushService{
		apns:        apns,
		deviceStore: deviceStore,
	}
}

// SendToUser sends a notification to all devices for a user
func (s *PushService) SendToUser(ctx context.Context, userID string, notification *Notification) error {
	tokens, err := s.deviceStore.GetTokensForUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get device tokens: %w", err)
	}

	if len(tokens) == 0 {
		log.Printf("[Push] No devices registered for user %s", userID)
		return nil
	}

	var lastErr error
	successCount := 0

	for _, device := range tokens {
		if device.Platform != "ios" {
			// Skip non-iOS devices (Android would use FCM)
			continue
		}

		notification.DeviceToken = device.Token
		err := s.apns.Send(notification)
		if err != nil {
			log.Printf("[Push] Failed to send to device %s: %v", device.Token[:16]+"...", err)
			lastErr = err

			// Remove invalid tokens
			if isInvalidTokenError(err) {
				_ = s.deviceStore.RemoveToken(ctx, device.Token)
			}
		} else {
			successCount++
		}
	}

	log.Printf("[Push] Sent notification to %d/%d devices for user %s", successCount, len(tokens), userID)

	if successCount == 0 && lastErr != nil {
		return lastErr
	}
	return nil
}

// NotifyNewMessage sends a new message notification
func (s *PushService) NotifyNewMessage(ctx context.Context, recipientUserID, senderName, messagePreview, conversationID string) error {
	return s.SendToUser(ctx, recipientUserID, &Notification{
		Title:    senderName,
		Body:     messagePreview,
		Sound:    "default",
		Category: "MESSAGE_CATEGORY",
		ThreadID: conversationID,
		Priority: 10,
		PushType: "alert",
		Data: map[string]interface{}{
			"type":            string(NotificationTypeMessage),
			"conversation_id": conversationID,
		},
	})
}

// NotifyFriendRequest sends a friend request notification
func (s *PushService) NotifyFriendRequest(ctx context.Context, recipientUserID, fromUsername, fromUserID string) error {
	return s.SendToUser(ctx, recipientUserID, &Notification{
		Title:    "Friend Request",
		Body:     fmt.Sprintf("%s sent you a friend request", fromUsername),
		Sound:    "default",
		Category: "FRIEND_REQUEST_CATEGORY",
		Priority: 10,
		PushType: "alert",
		Data: map[string]interface{}{
			"type":    string(NotificationTypeFriendRequest),
			"user_id": fromUserID,
		},
	})
}

// NotifyFriendAccepted sends a friend accepted notification
func (s *PushService) NotifyFriendAccepted(ctx context.Context, recipientUserID, accepterUsername string) error {
	return s.SendToUser(ctx, recipientUserID, &Notification{
		Title:    "Friend Request Accepted",
		Body:     fmt.Sprintf("%s accepted your friend request", accepterUsername),
		Sound:    "default",
		Priority: 10,
		PushType: "alert",
		Data: map[string]interface{}{
			"type": string(NotificationTypeFriendAccepted),
		},
	})
}

// NotifyMissedCall sends a missed call notification
func (s *PushService) NotifyMissedCall(ctx context.Context, recipientUserID, callerName string) error {
	return s.SendToUser(ctx, recipientUserID, &Notification{
		Title:    "Missed Call",
		Body:     fmt.Sprintf("You missed a call from %s", callerName),
		Sound:    "default",
		Priority: 10,
		PushType: "alert",
		Data: map[string]interface{}{
			"type": string(NotificationTypeMissedCall),
		},
	})
}

// isInvalidTokenError checks if the error indicates an invalid device token
func isInvalidTokenError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "BadDeviceToken") ||
		contains(errStr, "Unregistered") ||
		contains(errStr, "DeviceTokenNotForTopic")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
