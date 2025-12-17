package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jaydenbeard/messaging-app/internal/security"
	_ "github.com/lib/pq"
)

// PostgresDB wraps the database connection
type PostgresDB struct {
	db *sql.DB
}

// Message represents a stored message
type Message struct {
	MessageID   uuid.UUID
	SenderID    uuid.UUID
	ReceiverID  *uuid.UUID
	GroupID     *uuid.UUID
	Ciphertext  []byte
	MessageType string
	MediaID     *uuid.UUID
	MediaType   string
	Timestamp   time.Time
	Status      string
	DeliveredAt *time.Time
	ReadAt      *time.Time
}

// GroupMember represents a group member from DB
type GroupMember struct {
	UserID   uuid.UUID
	Role     string
	JoinedAt time.Time
}

// NewPostgresDB creates a new database connection
func NewPostgresDB(connStr string) (*PostgresDB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresDB{db: db}, nil
}

// Close closes the database connection
func (p *PostgresDB) Close() error {
	return p.db.Close()
}

// GetDB returns the underlying *sql.DB connection (for audit logging)
func (p *PostgresDB) GetDB() *sql.DB {
	return p.db
}

// SaveMessage stores an encrypted message
func (p *PostgresDB) SaveMessage(msg *Message) error {
	query := `
		INSERT INTO messages (message_id, sender_id, receiver_id, group_id, ciphertext, message_type, media_id, media_type, timestamp, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := p.db.Exec(query,
		msg.MessageID,
		msg.SenderID,
		msg.ReceiverID,
		msg.GroupID,
		msg.Ciphertext,
		msg.MessageType,
		msg.MediaID,
		msg.MediaType,
		msg.Timestamp,
		msg.Status,
	)
	return err
}

// GetMessage retrieves a message by ID
func (p *PostgresDB) GetMessage(messageID uuid.UUID) (*Message, error) {
	query := `
		SELECT message_id, sender_id, receiver_id, group_id, ciphertext, message_type, media_id, media_type, timestamp, status, delivered_at, read_at
		FROM messages WHERE message_id = $1`

	msg := &Message{}
	err := p.db.QueryRow(query, messageID).Scan(
		&msg.MessageID,
		&msg.SenderID,
		&msg.ReceiverID,
		&msg.GroupID,
		&msg.Ciphertext,
		&msg.MessageType,
		&msg.MediaID,
		&msg.MediaType,
		&msg.Timestamp,
		&msg.Status,
		&msg.DeliveredAt,
		&msg.ReadAt,
	)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// GetPendingMessages gets undelivered messages for a user
func (p *PostgresDB) GetPendingMessages(userID uuid.UUID) ([]*Message, error) {
	query := `
		SELECT message_id, sender_id, receiver_id, group_id, ciphertext, message_type, media_id, media_type, timestamp, status
		FROM messages 
		WHERE (receiver_id = $1 OR group_id IN (SELECT group_id FROM group_members WHERE user_id = $1))
		AND status = 'sent'
		ORDER BY timestamp ASC
		LIMIT 100`

	rows, err := p.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var messages []*Message
	for rows.Next() {
		msg := &Message{}
		if err := rows.Scan(
			&msg.MessageID,
			&msg.SenderID,
			&msg.ReceiverID,
			&msg.GroupID,
			&msg.Ciphertext,
			&msg.MessageType,
			&msg.MediaID,
			&msg.MediaType,
			&msg.Timestamp,
			&msg.Status,
		); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

// UpdateMessageStatus updates the delivery status of a message
func (p *PostgresDB) UpdateMessageStatus(messageID uuid.UUID, status string, timestamp time.Time) error {
	var query string
	switch status {
	case "delivered":
		query = `UPDATE messages SET status = $1, delivered_at = $2 WHERE message_id = $3`
	case "read":
		query = `UPDATE messages SET status = $1, read_at = $2 WHERE message_id = $3`
	default:
		query = `UPDATE messages SET status = $1 WHERE message_id = $2`
		_, err := p.db.Exec(query, status, messageID)
		return err
	}
	_, err := p.db.Exec(query, status, timestamp, messageID)
	return err
}

// GetMessagedUsers returns all user IDs who have exchanged messages with the given user
// This is used for targeted presence broadcasting (privacy-first: only send presence to contacts)
func (p *PostgresDB) GetMessagedUsers(userID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT DISTINCT
			CASE
				WHEN sender_id = $1 THEN receiver_id
				ELSE sender_id
			END as contact_id
		FROM messages
		WHERE (sender_id = $1 OR receiver_id = $1)
			AND receiver_id IS NOT NULL
			AND group_id IS NULL
		ORDER BY contact_id`

	rows, err := p.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var contacts []uuid.UUID
	for rows.Next() {
		var contactID uuid.UUID
		if err := rows.Scan(&contactID); err != nil {
			return nil, err
		}
		contacts = append(contacts, contactID)
	}
	return contacts, nil
}

// GetGroupMembers returns all members of a group
func (p *PostgresDB) GetGroupMembers(groupID uuid.UUID) ([]GroupMember, error) {
	query := `SELECT user_id, role, joined_at FROM group_members WHERE group_id = $1`

	rows, err := p.db.Query(query, groupID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var members []GroupMember
	for rows.Next() {
		var m GroupMember
		if err := rows.Scan(&m.UserID, &m.Role, &m.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, nil
}

// User operations

// CreateUser creates a new user
func (p *PostgresDB) CreateUser(phoneNumber, displayName, identityKey, signedPrekey, prekeySignature string) (*uuid.UUID, error) {
	// Generate phone_hash for privacy-preserving contact discovery
	phoneHash := hashPhoneNumber(phoneNumber)

	query := `
		INSERT INTO users (phone_number, phone_hash, display_name, public_identity_key, public_signed_prekey, signed_prekey_signature)
		VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6)
		RETURNING user_id`

	var userID uuid.UUID
	err := p.db.QueryRow(query, phoneNumber, phoneHash, displayName, identityKey, signedPrekey, prekeySignature).Scan(&userID)
	if err != nil {
		return nil, err
	}
	return &userID, nil
}

// hashPhoneNumber creates a SHA-256 hash of a phone number for privacy-preserving contact discovery
func hashPhoneNumber(phoneNumber string) string {
	hash := sha256.Sum256([]byte(phoneNumber))
	return hex.EncodeToString(hash[:])
}

// GetUserByPhone finds a user by phone number
func (p *PostgresDB) GetUserByPhone(phoneNumber string) (*uuid.UUID, error) {
	query := `SELECT user_id FROM users WHERE phone_number = $1 AND is_active = true`

	var userID uuid.UUID
	err := p.db.QueryRow(query, phoneNumber).Scan(&userID)
	if err != nil {
		return nil, err
	}
	return &userID, nil
}

// GetUserByID retrieves a user by ID
func (p *PostgresDB) GetUserByID(userID uuid.UUID) (map[string]interface{}, error) {
	query := `
		SELECT user_id, phone_number, username, display_name, avatar_url, 
		       public_identity_key, public_signed_prekey, signed_prekey_signature,
		       created_at, last_seen, is_active
		FROM users WHERE user_id = $1`

	var user struct {
		UserID                uuid.UUID
		PhoneNumber           string
		Username              sql.NullString
		DisplayName           sql.NullString
		AvatarURL             sql.NullString
		PublicIdentityKey     string
		PublicSignedPrekey    string
		SignedPrekeySignature string
		CreatedAt             time.Time
		LastSeen              time.Time
		IsActive              bool
	}

	err := p.db.QueryRow(query, userID).Scan(
		&user.UserID,
		&user.PhoneNumber,
		&user.Username,
		&user.DisplayName,
		&user.AvatarURL,
		&user.PublicIdentityKey,
		&user.PublicSignedPrekey,
		&user.SignedPrekeySignature,
		&user.CreatedAt,
		&user.LastSeen,
		&user.IsActive,
	)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"user_id":                 user.UserID,
		"phone_number":            user.PhoneNumber,
		"public_identity_key":     user.PublicIdentityKey,
		"public_signed_prekey":    user.PublicSignedPrekey,
		"signed_prekey_signature": user.SignedPrekeySignature,
		"created_at":              user.CreatedAt,
		"last_seen":               user.LastSeen,
		"is_active":               user.IsActive,
	}

	if user.Username.Valid {
		result["username"] = user.Username.String
	}
	if user.DisplayName.Valid {
		result["display_name"] = user.DisplayName.String
	}
	if user.AvatarURL.Valid {
		result["avatar_url"] = user.AvatarURL.String
	}

	return result, nil
}

// GetUserKeys retrieves a user's public keys for E2EE session establishment
func (p *PostgresDB) GetUserKeys(userID uuid.UUID) (map[string]interface{}, error) {
	// Get identity and signed pre-key, plus display name for UI
	query := `
		SELECT public_identity_key, public_signed_prekey, signed_prekey_signature,
		       COALESCE(display_name, username, '') as display_name,
		       COALESCE(username, '') as username
		FROM users WHERE user_id = $1`

	var identityKey, signedPrekey, signedPrekeySig, displayName, username string
	err := p.db.QueryRow(query, userID).Scan(&identityKey, &signedPrekey, &signedPrekeySig, &displayName, &username)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"user_id":                 userID,
		"identity_key":            identityKey,
		"signed_prekey":           signedPrekey,
		"signed_prekey_signature": signedPrekeySig,
		"display_name":            displayName,
		"username":                username,
	}

	// Try to get an unused one-time pre-key
	prekeyQuery := `
		UPDATE prekeys SET used_at = NOW()
		WHERE id = (
			SELECT id FROM prekeys 
			WHERE user_id = $1 AND used_at IS NULL 
			ORDER BY prekey_id LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING prekey_id, public_key`

	var prekeyID int
	var prekeyPublic string
	err = p.db.QueryRow(prekeyQuery, userID).Scan(&prekeyID, &prekeyPublic)
	if err == nil {
		result["onetime_prekey_id"] = prekeyID
		result["onetime_prekey"] = prekeyPublic
	}
	// If no one-time pre-key available, session still works (just less forward secrecy)

	return result, nil
}

// UpdateUserKeys updates a user's public cryptographic keys
// Returns true if the identity key changed (triggers security notification)
// This is used when a user sets up encryption on a new device
func (p *PostgresDB) UpdateUserKeys(userID uuid.UUID, identityKey, signedPrekey, signedPrekeySig string) (bool, error) {
	// First, get the current identity key to check if it changed
	var currentIdentityKey string
	err := p.db.QueryRow(`SELECT public_identity_key FROM users WHERE user_id = $1`, userID).Scan(&currentIdentityKey)
	if err != nil {
		return false, fmt.Errorf("failed to get current identity key: %w", err)
	}

	// Check if identity key is actually changing
	identityKeyChanged := currentIdentityKey != identityKey

	// Update the keys
	query := `
		UPDATE users 
		SET public_identity_key = $2, 
		    public_signed_prekey = $3, 
		    signed_prekey_signature = $4,
		    last_seen = NOW()
		WHERE user_id = $1`

	result, err := p.db.Exec(query, userID, identityKey, signedPrekey, signedPrekeySig)
	if err != nil {
		return false, fmt.Errorf("failed to update keys: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return false, fmt.Errorf("user not found")
	}

	if identityKeyChanged {
		log.Printf("[Security] Identity key changed for user %s", userID)
	}

	return identityKeyChanged, nil
}

// SavePreKeys stores a batch of one-time pre-keys
func (p *PostgresDB) SavePreKeys(userID uuid.UUID, prekeys []struct {
	ID        int
	PublicKey string
}) error {
	query := `INSERT INTO prekeys (user_id, prekey_id, public_key) VALUES ($1, $2, $3)`

	for _, pk := range prekeys {
		_, err := p.db.Exec(query, userID, pk.ID, pk.PublicKey)
		if err != nil {
			return err
		}
	}
	return nil
}

// CheckUsernameAvailable checks if a username is available
func (p *PostgresDB) CheckUsernameAvailable(username string) (bool, error) {
	// Validate username format
	if !security.ValidateUsername(username) {
		return false, fmt.Errorf("invalid username format")
	}

	var count int
	err := p.db.QueryRow("SELECT COUNT(*) FROM users WHERE LOWER(username) = LOWER($1)", username).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// UpdateUser updates user profile fields
func (p *PostgresDB) UpdateUser(userID uuid.UUID, fields map[string]interface{}) error {
	// SECURITY: Whitelist of allowed fields
	allowedFields := map[string]bool{
		"username":     true,
		"display_name": true,
		"avatar_url":   true,
	}

	// Validate all fields are in whitelist
	for field := range fields {
		if !allowedFields[field] {
			return fmt.Errorf("field '%s' is not allowed to be updated", field)
		}
	}

	if len(fields) == 0 {
		return fmt.Errorf("no fields to update")
	}

	// Build dynamic update query with proper parameter numbering
	// SECURITY NOTE: This is safe from SQL injection because:
	// 1. Field names are hardcoded strings (not user input)
	// 2. Field names are validated against allowedFields whitelist above
	// 3. All values are parameterized with $n placeholders
	setClauses := []string{}
	args := []interface{}{}
	i := 1

	if username, ok := fields["username"]; ok {
		setClauses = append(setClauses, fmt.Sprintf("username = $%d", i))
		args = append(args, username)
		i++
	}
	if displayName, ok := fields["display_name"]; ok {
		setClauses = append(setClauses, fmt.Sprintf("display_name = $%d", i))
		args = append(args, displayName)
		i++
	}
	if avatarURL, ok := fields["avatar_url"]; ok {
		setClauses = append(setClauses, fmt.Sprintf("avatar_url = $%d", i))
		args = append(args, avatarURL)
		i++
	}

	args = append(args, userID)
	query := fmt.Sprintf("UPDATE users SET %s WHERE user_id = $%d",
		strings.Join(setClauses, ", "), i)

	_, err := p.db.Exec(query, args...)
	return err
}

// DeleteUser permanently deletes a user and all associated data
func (p *PostgresDB) DeleteUser(userID uuid.UUID) error {
	// First, get the user's phone number (outside transaction)
	var phoneNumber string
	_ = p.db.QueryRow("SELECT phone_number FROM users WHERE user_id = $1", userID).Scan(&phoneNumber)

	// Pre-cleanup: delete data from tables that might not have proper CASCADE
	// These are done outside the main transaction so they don't abort it

	// Delete messages (might not exist)
	_, _ = p.db.Exec("DELETE FROM messages WHERE sender_id = $1 OR receiver_id = $1", userID)

	// Delete group memberships
	_, _ = p.db.Exec("DELETE FROM group_members WHERE user_id = $1", userID)

	// Delete device approval requests
	_, _ = p.db.Exec("DELETE FROM device_approval_requests WHERE user_id = $1", userID)

	// Delete verification codes if we have the phone number
	if phoneNumber != "" {
		_, _ = p.db.Exec("DELETE FROM verification_codes WHERE phone_number = $1", phoneNumber)
	}

	// Delete devices (in case CASCADE isn't set up)
	_, _ = p.db.Exec("DELETE FROM devices WHERE user_id = $1", userID)

	// Delete user pins
	_, _ = p.db.Exec("DELETE FROM user_pins WHERE user_id = $1", userID)

	// Delete prekeys
	_, _ = p.db.Exec("DELETE FROM prekeys WHERE user_id = $1", userID)

	// Now delete the user record itself
	result, err := p.db.Exec("DELETE FROM users WHERE user_id = $1", userID)
	if err != nil {
		log.Printf("Failed to delete user %s: %v", userID, err)
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Warning: Failed to get rows affected: %v", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("user not found: %s", userID)
	}

	log.Printf("Successfully deleted user %s and all associated data", userID)
	return nil
}

// ============================================
// DEVICE MANAGEMENT
// ============================================

// Device represents a user's device
type Device struct {
	DeviceID     uuid.UUID `json:"device_id"`
	UserID       uuid.UUID `json:"user_id"`
	DeviceName   string    `json:"device_name"`
	DeviceType   string    `json:"device_type"`
	IsPrimary    bool      `json:"is_primary"`
	RegisteredAt time.Time `json:"registered_at"`
	LastSeen     time.Time `json:"last_seen"`
	IsActive     bool      `json:"is_active"`
}

// RegisterDevice adds a new device for a user
func (p *PostgresDB) RegisterDevice(userID uuid.UUID, deviceID uuid.UUID, deviceName, deviceType, publicKey string, isPrimary bool) error {
	query := `
		INSERT INTO devices (device_id, user_id, device_name, device_type, public_device_key, is_primary)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (device_id) DO UPDATE SET
			device_name = $3,
			last_seen = NOW(),
			is_active = true,
			is_primary = CASE WHEN $6 = true THEN $6 ELSE devices.is_primary END`
	_, err := p.db.Exec(query, deviceID, userID, deviceName, deviceType, publicKey, isPrimary)
	return err
}

// GetUserDevices returns all active devices for a user
func (p *PostgresDB) GetUserDevices(userID uuid.UUID) ([]Device, error) {
	query := `
		SELECT device_id, user_id, COALESCE(device_name, ''), COALESCE(device_type, 'web'), 
			   is_primary, registered_at, last_seen, is_active
		FROM devices 
		WHERE user_id = $1 AND is_active = true
		ORDER BY is_primary DESC, last_seen DESC`

	rows, err := p.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var devices []Device
	for rows.Next() {
		var d Device
		err := rows.Scan(&d.DeviceID, &d.UserID, &d.DeviceName, &d.DeviceType,
			&d.IsPrimary, &d.RegisteredAt, &d.LastSeen, &d.IsActive)
		if err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, nil
}

// UpdateDeviceLastSeen updates the last seen time for a device
func (p *PostgresDB) UpdateDeviceLastSeen(deviceID uuid.UUID) error {
	query := `UPDATE devices SET last_seen = NOW() WHERE device_id = $1`
	_, err := p.db.Exec(query, deviceID)
	return err
}

// RemoveDevice deactivates a device
func (p *PostgresDB) RemoveDevice(userID, deviceID uuid.UUID) error {
	query := `UPDATE devices SET is_active = false WHERE device_id = $1 AND user_id = $2`
	_, err := p.db.Exec(query, deviceID, userID)
	return err
}

// ============================================
// PIN MANAGEMENT (Server-side sync)
// ============================================

// SaveUserPIN stores or updates the user's PIN hash on the server
func (p *PostgresDB) SaveUserPIN(userID uuid.UUID, pinHash string, pinLength int) error {
	query := `
		INSERT INTO user_pins (user_id, pin_hash, pin_length)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE SET
			pin_hash = $2,
			pin_length = $3,
			failed_attempts = 0,
			locked_until = NULL,
			updated_at = NOW()`
	_, err := p.db.Exec(query, userID, pinHash, pinLength)
	return err
}

// GetUserPIN retrieves the user's PIN hash
func (p *PostgresDB) GetUserPIN(userID uuid.UUID) (string, int, error) {
	query := `SELECT pin_hash, pin_length FROM user_pins WHERE user_id = $1`
	var pinHash string
	var pinLength int
	err := p.db.QueryRow(query, userID).Scan(&pinHash, &pinLength)
	if err == sql.ErrNoRows {
		return "", 0, nil // No PIN set
	}
	return pinHash, pinLength, err
}

// DeleteUserPIN removes the user's PIN
func (p *PostgresDB) DeleteUserPIN(userID uuid.UUID) error {
	query := `DELETE FROM user_pins WHERE user_id = $1`
	_, err := p.db.Exec(query, userID)
	return err
}

// ============================================
// TOTP SECRET MANAGEMENT
// ============================================

// SaveTOTPSecret stores an encrypted TOTP secret for a user
func (p *PostgresDB) SaveTOTPSecret(userID uuid.UUID, encryptedSecret string) error {
	query := `
		UPDATE users SET totp_secret = $2, last_seen = NOW()
		WHERE user_id = $1`
	_, err := p.db.Exec(query, userID, encryptedSecret)
	return err
}

// GetTOTPSecret retrieves the encrypted TOTP secret for a user
func (p *PostgresDB) GetTOTPSecret(userID uuid.UUID) (string, error) {
	query := `SELECT totp_secret FROM users WHERE user_id = $1 AND totp_secret IS NOT NULL`
	var encryptedSecret string
	err := p.db.QueryRow(query, userID).Scan(&encryptedSecret)
	if err == sql.ErrNoRows {
		return "", nil // No TOTP secret set
	}
	return encryptedSecret, err
}

// ============================================
// DEVICE APPROVAL (Secure Device Linking)
// ============================================

// DeviceApprovalRequest represents a pending device approval
type DeviceApprovalRequest struct {
	RequestID     uuid.UUID `json:"request_id"`
	UserID        uuid.UUID `json:"user_id"`
	NewDeviceID   uuid.UUID `json:"new_device_id"`
	NewDeviceName string    `json:"new_device_name"`
	NewDeviceType string    `json:"new_device_type"`
	ApprovalCode  string    `json:"approval_code"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
}

// CreateDeviceApprovalRequest creates a new device approval request
func (p *PostgresDB) CreateDeviceApprovalRequest(userID, newDeviceID uuid.UUID, deviceName, deviceType, code string) (*DeviceApprovalRequest, error) {
	query := `
		INSERT INTO device_approval_requests (user_id, new_device_id, new_device_name, new_device_type, approval_code)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING request_id, created_at, expires_at`

	var req DeviceApprovalRequest
	req.UserID = userID
	req.NewDeviceID = newDeviceID
	req.NewDeviceName = deviceName
	req.NewDeviceType = deviceType
	req.ApprovalCode = code
	req.Status = "pending"

	err := p.db.QueryRow(query, userID, newDeviceID, deviceName, deviceType, code).Scan(
		&req.RequestID, &req.CreatedAt, &req.ExpiresAt,
	)
	return &req, err
}

// GetPendingApprovalRequests gets all pending approval requests for a user
func (p *PostgresDB) GetPendingApprovalRequests(userID uuid.UUID) ([]DeviceApprovalRequest, error) {
	query := `
		SELECT request_id, user_id, new_device_id, COALESCE(new_device_name, ''), 
			   COALESCE(new_device_type, 'web'), approval_code, status, created_at, expires_at
		FROM device_approval_requests
		WHERE user_id = $1 AND status = 'pending' AND expires_at > NOW()
		ORDER BY created_at DESC`

	rows, err := p.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var requests []DeviceApprovalRequest
	for rows.Next() {
		var req DeviceApprovalRequest
		err := rows.Scan(&req.RequestID, &req.UserID, &req.NewDeviceID, &req.NewDeviceName,
			&req.NewDeviceType, &req.ApprovalCode, &req.Status, &req.CreatedAt, &req.ExpiresAt)
		if err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}
	return requests, nil
}

// ApproveDeviceRequest approves a device linking request
func (p *PostgresDB) ApproveDeviceRequest(requestID, approverDeviceID uuid.UUID) error {
	query := `
		UPDATE device_approval_requests 
		SET status = 'approved', approved_by_device_id = $2, responded_at = NOW()
		WHERE request_id = $1 AND status = 'pending' AND expires_at > NOW()`

	result, err := p.db.Exec(query, requestID, approverDeviceID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Printf("Warning: Failed to get rows affected: %v", err)
	}
	if rows == 0 {
		return fmt.Errorf("request not found, expired, or already processed")
	}
	return nil
}

// DenyDeviceRequest denies a device linking request
func (p *PostgresDB) DenyDeviceRequest(requestID uuid.UUID) error {
	query := `
		UPDATE device_approval_requests 
		SET status = 'denied', responded_at = NOW()
		WHERE request_id = $1 AND status = 'pending'`

	_, err := p.db.Exec(query, requestID)
	return err
}

// VerifyApprovalCode checks if the code is valid and returns the request
func (p *PostgresDB) VerifyApprovalCode(userID uuid.UUID, code string) (*DeviceApprovalRequest, error) {
	query := `
		SELECT request_id, user_id, new_device_id, COALESCE(new_device_name, ''),
			   COALESCE(new_device_type, 'web'), approval_code, status, created_at, expires_at
		FROM device_approval_requests
		WHERE user_id = $1 AND approval_code = $2 AND status = 'pending' AND expires_at > NOW()`

	var req DeviceApprovalRequest
	err := p.db.QueryRow(query, userID, code).Scan(
		&req.RequestID, &req.UserID, &req.NewDeviceID, &req.NewDeviceName,
		&req.NewDeviceType, &req.ApprovalCode, &req.Status, &req.CreatedAt, &req.ExpiresAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid or expired code")
	}
	return &req, err
}

// CheckApprovalStatus checks the status of a device approval request
func (p *PostgresDB) CheckApprovalStatus(requestID uuid.UUID) (string, error) {
	query := `SELECT status FROM device_approval_requests WHERE request_id = $1`
	var status string
	err := p.db.QueryRow(query, requestID).Scan(&status)
	return status, err
}

// IsDeviceLinked checks if a device is already linked to the user (and active)
func (p *PostgresDB) IsDeviceLinked(userID, deviceID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM devices WHERE user_id = $1 AND device_id = $2 AND is_active = true)`
	var exists bool
	err := p.db.QueryRow(query, userID, deviceID).Scan(&exists)
	return exists, err
}

// WasDeviceEverLinked checks if a device was ever linked to the user (even if now inactive)
// This allows recognizing devices that were previously used on this account
func (p *PostgresDB) WasDeviceEverLinked(userID, deviceID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM devices WHERE user_id = $1 AND device_id = $2)`
	var exists bool
	err := p.db.QueryRow(query, userID, deviceID).Scan(&exists)
	return exists, err
}

// HasLinkedDevices checks if user has any linked devices (for first device case)
func (p *PostgresDB) HasLinkedDevices(userID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM devices WHERE user_id = $1 AND is_active = true)`
	var exists bool
	err := p.db.QueryRow(query, userID).Scan(&exists)
	return exists, err
}

// IsDeviceActive checks if a specific device exists, belongs to user, and is active
func (p *PostgresDB) IsDeviceActive(userID, deviceID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM devices WHERE user_id = $1 AND device_id = $2 AND is_active = true)`
	var exists bool
	err := p.db.QueryRow(query, userID, deviceID).Scan(&exists)
	return exists, err
}

// GetPrimaryDevice returns the primary device for a user
func (p *PostgresDB) GetPrimaryDevice(userID uuid.UUID) (*Device, error) {
	query := `
		SELECT device_id, user_id, COALESCE(device_name, ''), COALESCE(device_type, 'web'), 
			   is_primary, registered_at, last_seen, is_active
		FROM devices 
		WHERE user_id = $1 AND is_primary = true AND is_active = true
		LIMIT 1`

	var d Device
	err := p.db.QueryRow(query, userID).Scan(
		&d.DeviceID, &d.UserID, &d.DeviceName, &d.DeviceType,
		&d.IsPrimary, &d.RegisteredAt, &d.LastSeen, &d.IsActive,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// IsPrimaryDevice checks if a specific device is the primary device
func (p *PostgresDB) IsPrimaryDevice(userID, deviceID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM devices WHERE user_id = $1 AND device_id = $2 AND is_primary = true AND is_active = true)`
	var isPrimary bool
	err := p.db.QueryRow(query, userID, deviceID).Scan(&isPrimary)
	return isPrimary, err
}

// SetPrimaryDevice changes which device is the primary device
func (p *PostgresDB) SetPrimaryDevice(userID, newPrimaryDeviceID uuid.UUID) error {
	tx, err := p.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("Warning: failed to rollback: %v", err)
		}
	}()

	// Remove primary status from all user's devices
	_, err = tx.Exec(`UPDATE devices SET is_primary = false WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}

	// Set new primary device
	result, err := tx.Exec(`UPDATE devices SET is_primary = true WHERE user_id = $1 AND device_id = $2 AND is_active = true`, userID, newPrimaryDeviceID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Printf("Warning: Failed to get rows affected: %v", err)
	}
	if rows == 0 {
		return fmt.Errorf("device not found or not active")
	}

	return tx.Commit()
}

// EnsurePrimaryDevice makes sure user has a primary device (promotes oldest active if none)
func (p *PostgresDB) EnsurePrimaryDevice(userID uuid.UUID) error {
	// Check if user has a primary device
	primary, err := p.GetPrimaryDevice(userID)
	if err != nil {
		return err
	}
	if primary != nil {
		return nil // Already has primary device
	}

	// Promote oldest active device to primary
	query := `
		UPDATE devices SET is_primary = true
		WHERE device_id = (
			SELECT device_id FROM devices
			WHERE user_id = $1 AND is_active = true
			ORDER BY registered_at ASC
			LIMIT 1
		)`
	_, err = p.db.Exec(query, userID)
	return err
}

// ============================================
// PRIVACY SETTINGS
// ============================================

// GetPrivacySettings retrieves a user's privacy settings
func (p *PostgresDB) GetPrivacySettings(userID uuid.UUID) (map[string]interface{}, error) {
	query := `
		SELECT show_read_receipts, show_online_status, show_last_seen, show_typing_indicator
		FROM privacy_settings WHERE user_id = $1`

	var showReadReceipts, showOnlineStatus, showLastSeen, showTypingIndicator bool
	err := p.db.QueryRow(query, userID).Scan(&showReadReceipts, &showOnlineStatus, &showLastSeen, &showTypingIndicator)
	if err == sql.ErrNoRows {
		// Return defaults if no settings exist
		return map[string]interface{}{
			"show_read_receipts":    true,
			"show_online_status":    true,
			"show_last_seen":        true,
			"show_typing_indicator": true,
		}, nil
	}
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"show_read_receipts":    showReadReceipts,
		"show_online_status":    showOnlineStatus,
		"show_last_seen":        showLastSeen,
		"show_typing_indicator": showTypingIndicator,
	}, nil
}

// UpdatePrivacySetting updates a specific privacy setting
func (p *PostgresDB) UpdatePrivacySetting(userID uuid.UUID, setting string, value bool) error {
	// First ensure row exists
	_, err := p.db.Exec(`
		INSERT INTO privacy_settings (user_id, show_read_receipts, show_online_status, show_last_seen, show_typing_indicator)
		VALUES ($1, true, true, true, true)
		ON CONFLICT (user_id) DO NOTHING`, userID)
	if err != nil {
		return err
	}

	// Update the specific setting
	var query string
	switch setting {
	case "show_read_receipts":
		query = `UPDATE privacy_settings SET show_read_receipts = $2, updated_at = NOW() WHERE user_id = $1`
	case "show_online_status":
		query = `UPDATE privacy_settings SET show_online_status = $2, updated_at = NOW() WHERE user_id = $1`
	case "show_last_seen":
		query = `UPDATE privacy_settings SET show_last_seen = $2, updated_at = NOW() WHERE user_id = $1`
	case "show_typing_indicator":
		query = `UPDATE privacy_settings SET show_typing_indicator = $2, updated_at = NOW() WHERE user_id = $1`
	default:
		return fmt.Errorf("unknown privacy setting: %s", setting)
	}

	_, err = p.db.Exec(query, userID, value)
	return err
}

// ExpireOldApprovalRequests expires old pending requests
func (p *PostgresDB) ExpireOldApprovalRequests() error {
	query := `UPDATE device_approval_requests SET status = 'expired' WHERE status = 'pending' AND expires_at < NOW()`
	_, err := p.db.Exec(query)
	return err
}

// Verification codes

// SaveVerificationCode stores a verification code
func (p *PostgresDB) SaveVerificationCode(phoneNumber, code string, expiresAt time.Time) error {
	query := `INSERT INTO verification_codes (phone_number, code, expires_at) VALUES ($1, $2, $3)`
	_, err := p.db.Exec(query, phoneNumber, code, expiresAt)
	return err
}

// CheckCode validates a code without marking it as verified (for pre-checking)
func (p *PostgresDB) CheckCode(phoneNumber, code string) (bool, error) {
	query := `
		SELECT id FROM verification_codes 
		WHERE phone_number = $1 AND code = $2 AND expires_at > NOW() AND verified = false AND attempts < 5
		LIMIT 1`

	var id int
	err := p.db.QueryRow(query, phoneNumber, code).Scan(&id)
	if err == sql.ErrNoRows {
		// Increment attempt counter even for wrong codes
		if _, execErr := p.db.Exec(`UPDATE verification_codes SET attempts = attempts + 1 WHERE phone_number = $1 AND expires_at > NOW() AND verified = false`, phoneNumber); execErr != nil {
			log.Printf("Warning: failed to increment attempt counter: %v", execErr)
		}
		return false, nil
	}
	return err == nil, err
}

// VerifyCode checks if a code is valid and marks it as verified
func (p *PostgresDB) VerifyCode(phoneNumber, code string) (bool, error) {
	query := `
		UPDATE verification_codes
		SET verified = true, attempts = attempts + 1
		WHERE phone_number = $1 AND code = $2 AND expires_at > NOW() AND verified = false AND attempts < 5
		RETURNING id`

	var id int
	err := p.db.QueryRow(query, phoneNumber, code).Scan(&id)
	if err == sql.ErrNoRows {
		// Increment attempt counter even for wrong codes
		if _, execErr := p.db.Exec(`UPDATE verification_codes SET attempts = attempts + 1 WHERE phone_number = $1 AND expires_at > NOW() AND verified = false`, phoneNumber); execErr != nil {
			log.Printf("Warning: failed to increment attempt counter: %v", execErr)
		}
		return false, nil
	}
	return err == nil, err
}

// MarkCodeVerified marks a verification code as verified (used after successful user creation)
func (p *PostgresDB) MarkCodeVerified(phoneNumber, code string) error {
	query := `
		UPDATE verification_codes
		SET verified = true
		WHERE phone_number = $1 AND code = $2 AND expires_at > NOW() AND verified = false`
	_, err := p.db.Exec(query, phoneNumber, code)
	return err
}

// Group operations

// CreateGroup creates a new group
func (p *PostgresDB) CreateGroup(name string, creatorID uuid.UUID) (*uuid.UUID, error) {
	tx, err := p.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("Warning: failed to rollback: %v", err)
		}
	}()

	// Create group
	var groupID uuid.UUID
	err = tx.QueryRow(`
		INSERT INTO groups (name, created_by)
		VALUES ($1, $2)
		RETURNING group_id`, name, creatorID).Scan(&groupID)
	if err != nil {
		return nil, err
	}

	// Add creator as admin
	_, err = tx.Exec(`
		INSERT INTO group_members (group_id, user_id, role, encrypted_group_key)
		VALUES ($1, $2, 'admin', '')`, groupID, creatorID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &groupID, nil
}

// AddGroupMember adds a user to a group
func (p *PostgresDB) AddGroupMember(groupID, userID uuid.UUID, encryptedKey string) error {
	query := `INSERT INTO group_members (group_id, user_id, role, encrypted_group_key) VALUES ($1, $2, 'member', $3)`
	_, err := p.db.Exec(query, groupID, userID, encryptedKey)
	return err
}

// RemoveGroupMember removes a user from a group
func (p *PostgresDB) RemoveGroupMember(groupID, userID uuid.UUID) error {
	query := `DELETE FROM group_members WHERE group_id = $1 AND user_id = $2`
	_, err := p.db.Exec(query, groupID, userID)
	return err
}

// IsGroupAdmin checks if a user is an admin of a group
func (p *PostgresDB) IsGroupAdmin(groupID, userID uuid.UUID) (bool, error) {
	query := `SELECT role FROM group_members WHERE group_id = $1 AND user_id = $2`

	var role string
	err := p.db.QueryRow(query, groupID, userID).Scan(&role)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return false, nil // User is not a member
		}
		return false, err
	}

	return role == "admin", nil
}

// IsGroupMember checks if a user is a member (admin or regular member) of a group
func (p *PostgresDB) IsGroupMember(groupID, userID uuid.UUID) (bool, error) {
	query := `SELECT COUNT(*) FROM group_members WHERE group_id = $1 AND user_id = $2`

	var count int
	err := p.db.QueryRow(query, groupID, userID).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// Session operations

// CreateSession stores a new session
func (p *PostgresDB) CreateSession(userID uuid.UUID, tokenHash string, expiresAt time.Time) (*uuid.UUID, error) {
	var sessionID uuid.UUID
	err := p.db.QueryRow(`
		INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING session_id`, userID, tokenHash, expiresAt).Scan(&sessionID)
	return &sessionID, err
}

// ValidateSession checks if a session is valid
// Updated to handle both old and new hash formats for backward compatibility
func (p *PostgresDB) ValidateSession(tokenHash string) (*uuid.UUID, error) {
	var userID uuid.UUID

	// Try new salted hash format first (longer hash)
	if len(tokenHash) > 64 { // Salted hashes are longer than 64 chars
		err := p.db.QueryRow(`
			SELECT user_id FROM sessions
			WHERE token_hash = $1 AND expires_at > NOW() AND revoked_at IS NULL`, tokenHash).Scan(&userID)
		if err == nil {
			// Update last used
			if _, execErr := p.db.Exec(`UPDATE sessions SET last_used = NOW() WHERE token_hash = $1`, tokenHash); execErr != nil {
				log.Printf("Warning: failed to update session last_used: %v", execErr)
			}
			return &userID, nil
		}
	}

	// Fallback to old format for backward compatibility
	err := p.db.QueryRow(`
		SELECT user_id FROM sessions
		WHERE token_hash = $1 AND expires_at > NOW() AND revoked_at IS NULL`, tokenHash).Scan(&userID)
	if err != nil {
		return nil, err
	}

	// Update last used
	if _, err := p.db.Exec(`UPDATE sessions SET last_used = NOW() WHERE token_hash = $1`, tokenHash); err != nil {
		log.Printf("Warning: failed to update session last_used: %v", err)
	}

	return &userID, nil
}

// RevokeSession invalidates a session
func (p *PostgresDB) RevokeSession(tokenHash string) error {
	_, err := p.db.Exec(`UPDATE sessions SET revoked_at = NOW() WHERE token_hash = $1`, tokenHash)
	return err
}

// RevokeAllUserSessions invalidates all active sessions for a user (used when password/PIN changes)
func (p *PostgresDB) RevokeAllUserSessions(userID uuid.UUID) error {
	_, err := p.db.Exec(`UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	return err
}

// SearchUsers searches for users by username only (privacy-first: no phone number search)
func (p *PostgresDB) SearchUsers(query string, limit int) ([]map[string]interface{}, error) {
	// Only search by username - phone numbers stay private
	// Strip @ prefix if user included it
	searchQuery := query
	if len(searchQuery) > 0 && searchQuery[0] == '@' {
		searchQuery = searchQuery[1:]
	}

	sqlQuery := `
		SELECT user_id, username, display_name, avatar_url
		FROM users
		WHERE is_active = true
			AND username IS NOT NULL
			AND username ILIKE $1
		ORDER BY
			CASE WHEN LOWER(username) = LOWER($2) THEN 0 ELSE 1 END,
			username ASC
		LIMIT $3`

	rows, err := p.db.Query(sqlQuery, "%"+searchQuery+"%", searchQuery, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var users []map[string]interface{}
	for rows.Next() {
		var userID uuid.UUID
		var username, displayName, avatarURL sql.NullString

		if err := rows.Scan(&userID, &username, &displayName, &avatarURL); err != nil {
			return nil, err
		}

		// Only include users who have set a username
		if !username.Valid {
			continue
		}

		user := map[string]interface{}{
			"user_id":  userID,
			"username": username.String,
		}
		if displayName.Valid {
			user["display_name"] = displayName.String
		}
		if avatarURL.Valid {
			user["avatar_url"] = avatarURL.String
		}
		users = append(users, user)
	}
	return users, nil
}

// SearchUsersExcludingBlockers searches for users but excludes those who have blocked the searcher
func (p *PostgresDB) SearchUsersExcludingBlockers(query string, searcherID uuid.UUID, limit int) ([]map[string]interface{}, error) {
	// Only search by username - phone numbers stay private
	// Strip @ prefix if user included it
	searchQuery := query
	if len(searchQuery) > 0 && searchQuery[0] == '@' {
		searchQuery = searchQuery[1:]
	}

	sqlQuery := `
		SELECT user_id, username, display_name, avatar_url
		FROM users
		WHERE is_active = true
			AND username IS NOT NULL
			AND username ILIKE $1
			AND user_id != $4
			AND user_id NOT IN (
				SELECT blocker_id FROM blocked_users WHERE blocked_id = $4
			)
		ORDER BY
			CASE WHEN LOWER(username) = LOWER($2) THEN 0 ELSE 1 END,
			username ASC
		LIMIT $3`

	rows, err := p.db.Query(sqlQuery, "%"+searchQuery+"%", searchQuery, limit, searcherID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var users []map[string]interface{}
	for rows.Next() {
		var userID uuid.UUID
		var username, displayName, avatarURL sql.NullString

		if err := rows.Scan(&userID, &username, &displayName, &avatarURL); err != nil {
			return nil, err
		}

		// Only include users who have set a username
		if !username.Valid {
			continue
		}

		user := map[string]interface{}{
			"user_id":  userID,
			"username": username.String,
		}
		if displayName.Valid {
			user["display_name"] = displayName.String
		}
		if avatarURL.Valid {
			user["avatar_url"] = avatarURL.String
		}
		users = append(users, user)
	}
	return users, nil
}

// Note: Conversation metadata is now synced device-to-device, not stored server-side.
// This is for security - server should not know who talks to whom.

// ============================================
// BLOCKED USERS
// ============================================

// BlockUser blocks a user
func (p *PostgresDB) BlockUser(blockerID, blockedID uuid.UUID) error {
	_, err := p.db.Exec(`
		INSERT INTO blocked_users (blocker_id, blocked_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, blockerID, blockedID)
	return err
}

// UnblockUser unblocks a user
func (p *PostgresDB) UnblockUser(blockerID, blockedID uuid.UUID) error {
	_, err := p.db.Exec(`
		DELETE FROM blocked_users
		WHERE blocker_id = $1 AND blocked_id = $2
	`, blockerID, blockedID)
	return err
}

// GetBlockedUsers returns list of users blocked by a user
func (p *PostgresDB) GetBlockedUsers(blockerID uuid.UUID) ([]map[string]interface{}, error) {
	rows, err := p.db.Query(`
		SELECT u.user_id, u.username, u.display_name, u.avatar_url, bu.blocked_at
		FROM blocked_users bu
		JOIN users u ON bu.blocked_id = u.user_id
		WHERE bu.blocker_id = $1
		ORDER BY bu.blocked_at DESC
	`, blockerID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var users []map[string]interface{}
	for rows.Next() {
		var userID uuid.UUID
		var username, displayName, avatarURL sql.NullString
		var blockedAt time.Time
		if err := rows.Scan(&userID, &username, &displayName, &avatarURL, &blockedAt); err != nil {
			continue
		}
		user := map[string]interface{}{
			"user_id":    userID.String(),
			"blocked_at": blockedAt.Format(time.RFC3339),
		}
		if username.Valid {
			user["username"] = username.String
		}
		if displayName.Valid {
			user["display_name"] = displayName.String
		}
		if avatarURL.Valid {
			user["avatar_url"] = avatarURL.String
		}
		users = append(users, user)
	}
	return users, nil
}

// IsBlocked checks if a user is blocked by another user
func (p *PostgresDB) IsBlocked(blockerID, blockedID uuid.UUID) (bool, error) {
	var isBlocked bool
	err := p.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM blocked_users
			WHERE blocker_id = $1 AND blocked_id = $2
		)
	`, blockerID, blockedID).Scan(&isBlocked)
	return isBlocked, err
}

// ============================================
// FRIENDSHIPS (Facebook-style friend requests)
// ============================================

// FriendInfo represents a friend with their profile info
type FriendInfo struct {
	UserID       uuid.UUID `json:"user_id"`
	Username     string    `json:"username,omitempty"`
	DisplayName  string    `json:"display_name,omitempty"`
	AvatarURL    string    `json:"avatar_url,omitempty"`
	IsOnline     bool      `json:"is_online"`
	LastSeen     time.Time `json:"last_seen,omitempty"`
	FriendsSince time.Time `json:"friends_since"`
}

// FriendRequest represents a pending friend request
type FriendRequest struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	Username    string    `json:"username,omitempty"`
	DisplayName string    `json:"display_name,omitempty"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	Type        string    `json:"type"` // "incoming" or "outgoing"
}

// SendFriendRequest sends a friend request from requester to addressee
func (p *PostgresDB) SendFriendRequest(requesterID, addresseeID uuid.UUID) error {
	// Check if a friendship already exists (in either direction)
	var existingStatus string
	err := p.db.QueryRow(`
		SELECT status FROM friendships 
		WHERE (requester_id = $1 AND addressee_id = $2) 
		   OR (requester_id = $2 AND addressee_id = $1)
	`, requesterID, addresseeID).Scan(&existingStatus)

	if err == nil {
		if existingStatus == "accepted" {
			return fmt.Errorf("already friends")
		}
		if existingStatus == "pending" {
			return fmt.Errorf("friend request already pending")
		}
	}

	// Insert new friend request
	_, err = p.db.Exec(`
		INSERT INTO friendships (requester_id, addressee_id, status)
		VALUES ($1, $2, 'pending')
		ON CONFLICT (requester_id, addressee_id) DO UPDATE SET
			status = 'pending',
			updated_at = NOW()
	`, requesterID, addresseeID)
	return err
}

// AcceptFriendRequest accepts a pending friend request
func (p *PostgresDB) AcceptFriendRequest(addresseeID, requesterID uuid.UUID) error {
	result, err := p.db.Exec(`
		UPDATE friendships 
		SET status = 'accepted', updated_at = NOW()
		WHERE requester_id = $1 AND addressee_id = $2 AND status = 'pending'
	`, requesterID, addresseeID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("no pending friend request found")
	}
	return nil
}

// DeclineFriendRequest declines a pending friend request
func (p *PostgresDB) DeclineFriendRequest(addresseeID, requesterID uuid.UUID) error {
	result, err := p.db.Exec(`
		UPDATE friendships 
		SET status = 'declined', updated_at = NOW()
		WHERE requester_id = $1 AND addressee_id = $2 AND status = 'pending'
	`, requesterID, addresseeID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("no pending friend request found")
	}
	return nil
}

// RemoveFriend removes a friendship (unfriend)
func (p *PostgresDB) RemoveFriend(userID, friendID uuid.UUID) error {
	_, err := p.db.Exec(`
		DELETE FROM friendships 
		WHERE (requester_id = $1 AND addressee_id = $2)
		   OR (requester_id = $2 AND addressee_id = $1)
	`, userID, friendID)
	return err
}

// GetFriends returns all friends of a user
func (p *PostgresDB) GetFriends(userID uuid.UUID) ([]FriendInfo, error) {
	query := `
		SELECT 
			u.user_id, 
			COALESCE(u.username, '') as username,
			COALESCE(u.display_name, '') as display_name, 
			COALESCE(u.avatar_url, '') as avatar_url,
			u.is_active as is_online,
			u.last_seen,
			f.updated_at as friends_since
		FROM friendships f
		JOIN users u ON (
			CASE 
				WHEN f.requester_id = $1 THEN f.addressee_id
				ELSE f.requester_id
			END = u.user_id
		)
		WHERE (f.requester_id = $1 OR f.addressee_id = $1)
		  AND f.status = 'accepted'
		ORDER BY u.display_name, u.username`

	rows, err := p.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var friends []FriendInfo
	for rows.Next() {
		var f FriendInfo
		if err := rows.Scan(&f.UserID, &f.Username, &f.DisplayName, &f.AvatarURL, &f.IsOnline, &f.LastSeen, &f.FriendsSince); err != nil {
			return nil, err
		}
		friends = append(friends, f)
	}
	return friends, nil
}

// GetPendingFriendRequests returns incoming friend requests for a user
func (p *PostgresDB) GetPendingFriendRequests(userID uuid.UUID) ([]FriendRequest, error) {
	query := `
		SELECT 
			f.id,
			u.user_id, 
			COALESCE(u.username, '') as username,
			COALESCE(u.display_name, '') as display_name, 
			COALESCE(u.avatar_url, '') as avatar_url,
			f.created_at
		FROM friendships f
		JOIN users u ON f.requester_id = u.user_id
		WHERE f.addressee_id = $1 AND f.status = 'pending'
		ORDER BY f.created_at DESC`

	rows, err := p.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var requests []FriendRequest
	for rows.Next() {
		var r FriendRequest
		if err := rows.Scan(&r.ID, &r.UserID, &r.Username, &r.DisplayName, &r.AvatarURL, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Type = "incoming"
		requests = append(requests, r)
	}
	return requests, nil
}

// GetSentFriendRequests returns outgoing friend requests from a user
func (p *PostgresDB) GetSentFriendRequests(userID uuid.UUID) ([]FriendRequest, error) {
	query := `
		SELECT 
			f.id,
			u.user_id, 
			COALESCE(u.username, '') as username,
			COALESCE(u.display_name, '') as display_name, 
			COALESCE(u.avatar_url, '') as avatar_url,
			f.created_at
		FROM friendships f
		JOIN users u ON f.addressee_id = u.user_id
		WHERE f.requester_id = $1 AND f.status = 'pending'
		ORDER BY f.created_at DESC`

	rows, err := p.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var requests []FriendRequest
	for rows.Next() {
		var r FriendRequest
		if err := rows.Scan(&r.ID, &r.UserID, &r.Username, &r.DisplayName, &r.AvatarURL, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Type = "outgoing"
		requests = append(requests, r)
	}
	return requests, nil
}

// AreFriends checks if two users are friends
func (p *PostgresDB) AreFriends(userID1, userID2 uuid.UUID) (bool, error) {
	var areFriends bool
	err := p.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM friendships
			WHERE ((requester_id = $1 AND addressee_id = $2)
			    OR (requester_id = $2 AND addressee_id = $1))
			  AND status = 'accepted'
		)
	`, userID1, userID2).Scan(&areFriends)
	return areFriends, err
}

// GetFriendshipStatus returns the friendship status between two users
// Returns: "none", "pending_sent", "pending_received", "friends"
func (p *PostgresDB) GetFriendshipStatus(currentUserID, otherUserID uuid.UUID) (string, error) {
	var requesterID uuid.UUID
	var status string

	err := p.db.QueryRow(`
		SELECT requester_id, status FROM friendships
		WHERE (requester_id = $1 AND addressee_id = $2)
		   OR (requester_id = $2 AND addressee_id = $1)
	`, currentUserID, otherUserID).Scan(&requesterID, &status)

	if err == sql.ErrNoRows {
		return "none", nil
	}
	if err != nil {
		return "", err
	}

	if status == "accepted" {
		return "friends", nil
	}
	if status == "pending" {
		if requesterID == currentUserID {
			return "pending_sent", nil
		}
		return "pending_received", nil
	}
	return "none", nil
}

// GetFriendIDs returns just the user IDs of all friends (for efficient checks)
func (p *PostgresDB) GetFriendIDs(userID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT 
			CASE 
				WHEN requester_id = $1 THEN addressee_id
				ELSE requester_id
			END as friend_id
		FROM friendships
		WHERE (requester_id = $1 OR addressee_id = $1)
		  AND status = 'accepted'`

	rows, err := p.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var friendIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		friendIDs = append(friendIDs, id)
	}
	return friendIDs, nil
}

// ============================================
// SEALED SENDER CERTIFICATES
// ============================================

// SealedSenderCertificate represents a sealed sender certificate stored in the database
type SealedSenderCertificate struct {
	CertificateID   uuid.UUID  `json:"certificate_id"`
	UserID          uuid.UUID  `json:"user_id"`
	PublicKey       []byte     `json:"public_key"`
	Expiration      time.Time  `json:"expiration"`
	IssuedAt        time.Time  `json:"issued_at"`
	CertificateData []byte     `json:"certificate_data"` // PEM-encoded certificate
	IsRevoked       bool       `json:"is_revoked"`
	RevokedAt       *time.Time `json:"revoked_at,omitempty"`
}

// SaveSealedSenderCertificate stores a sealed sender certificate
func (p *PostgresDB) SaveSealedSenderCertificate(cert *security.SealedSenderIdentityCertificate) error {
	query := `
		INSERT INTO sealed_sender_certificates
		(certificate_id, user_id, public_key, expiration, issued_at, certificate_data, signature)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (certificate_id) DO UPDATE SET
			public_key = $3,
			expiration = $4,
			certificate_data = $6,
			signature = $7`

	_, err := p.db.Exec(query,
		cert.CertificateID,
		cert.UserID,
		cert.PublicKey,
		cert.Expiration,
		cert.IssuedAt,
		cert.CertificateData,
		cert.Signature,
	)
	return err
}

// GetSealedSenderCertificate retrieves a sealed sender certificate by ID
func (p *PostgresDB) GetSealedSenderCertificate(certificateID uuid.UUID) (*security.SealedSenderIdentityCertificate, error) {
	query := `
		SELECT certificate_id, user_id, public_key, expiration, issued_at, certificate_data, signature
		FROM sealed_sender_certificates
		WHERE certificate_id = $1`

	var cert security.SealedSenderIdentityCertificate

	err := p.db.QueryRow(query, certificateID).Scan(
		&cert.CertificateID,
		&cert.UserID,
		&cert.PublicKey,
		&cert.Expiration,
		&cert.IssuedAt,
		&cert.CertificateData,
		&cert.Signature,
	)
	if err != nil {
		return nil, err
	}

	return &cert, nil
}

// GetUserSealedSenderCertificates retrieves all valid (non-revoked, non-expired) certificates for a user
func (p *PostgresDB) GetUserSealedSenderCertificates(userID uuid.UUID) ([]*security.SealedSenderIdentityCertificate, error) {
	query := `
		SELECT certificate_id, user_id, public_key, expiration, issued_at, certificate_data, signature
		FROM sealed_sender_certificates
		WHERE user_id = $1 AND expiration > NOW()
		ORDER BY issued_at DESC`

	rows, err := p.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var certificates []*security.SealedSenderIdentityCertificate
	for rows.Next() {
		var cert security.SealedSenderIdentityCertificate

		err := rows.Scan(
			&cert.CertificateID,
			&cert.UserID,
			&cert.PublicKey,
			&cert.Expiration,
			&cert.IssuedAt,
			&cert.CertificateData,
			&cert.Signature,
		)
		if err != nil {
			return nil, err
		}

		certificates = append(certificates, &cert)
	}

	return certificates, nil
}

// RevokeSealedSenderCertificate marks a certificate as revoked
func (p *PostgresDB) RevokeSealedSenderCertificate(certificateID uuid.UUID) error {
	query := `
		DELETE FROM sealed_sender_certificates
		WHERE certificate_id = $1`

	_, err := p.db.Exec(query, certificateID)
	return err
}

// IsSealedSenderCertificateRevoked checks if a certificate has been revoked
func (p *PostgresDB) IsSealedSenderCertificateRevoked(certificateID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(SELECT 1 FROM sealed_sender_certificates WHERE certificate_id = $1)`

	var exists bool
	err := p.db.QueryRow(query, certificateID).Scan(&exists)
	if err != nil {
		return false, err
	}

	return !exists, nil // If certificate doesn't exist, treat as revoked
}

// CleanupExpiredSealedSenderCertificates removes expired certificates
func (p *PostgresDB) CleanupExpiredSealedSenderCertificates() error {
	query := `
		DELETE FROM sealed_sender_certificates
		WHERE expiration < NOW()`

	_, err := p.db.Exec(query)
	return err
}

// GetSealedSenderCertificateByUserAndKey finds a certificate by user ID and public key
func (p *PostgresDB) GetSealedSenderCertificateByUserAndKey(userID uuid.UUID, publicKey []byte) (*security.SealedSenderIdentityCertificate, error) {
	query := `
		SELECT certificate_id, user_id, public_key, expiration, issued_at, certificate_data, signature
		FROM sealed_sender_certificates
		WHERE user_id = $1 AND public_key = $2 AND expiration > NOW()
		LIMIT 1`

	var cert security.SealedSenderIdentityCertificate

	err := p.db.QueryRow(query, userID, publicKey).Scan(
		&cert.CertificateID,
		&cert.UserID,
		&cert.PublicKey,
		&cert.Expiration,
		&cert.IssuedAt,
		&cert.CertificateData,
		&cert.Signature,
	)
	if err != nil {
		return nil, err
	}

	return &cert, nil
}
