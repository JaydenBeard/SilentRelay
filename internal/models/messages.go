package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// WebSocket message types
const (
	// Client -> Server
	MessageTypeSend        = "send"         // Send encrypted message
	MessageTypeDeliveryAck = "delivery_ack" // Acknowledge message delivery
	MessageTypeReadReceipt = "read_receipt" // Mark messages as read
	MessageTypeTyping      = "typing"       // Typing indicator
	MessageTypeHeartbeat   = "heartbeat"    // Keep-alive ping
	MessageTypePresence    = "presence"     // Update presence status

	// Server -> Client
	MessageTypeDeliver      = "deliver"       // Deliver message to recipient
	MessageTypeSentAck      = "sent_ack"      // Acknowledge message was sent
	MessageTypeStatusUpdate = "status_update" // Message status changed
	MessageTypeHeartbeatAck = "heartbeat_ack" // Heartbeat acknowledgment
	MessageTypeError        = "error"         // Error message
	MessageTypeUserOnline   = "user_online"   // User came online
	MessageTypeUserOffline  = "user_offline"  // User went offline

	// Call signaling (WebRTC)
	MessageTypeCallOffer    = "call_offer"    // Initiate call with SDP offer
	MessageTypeCallAnswer   = "call_answer"   // Accept call with SDP answer
	MessageTypeCallReject   = "call_reject"   // Reject incoming call
	MessageTypeCallEnd      = "call_end"      // End active call
	MessageTypeCallBusy     = "call_busy"     // User is busy
	MessageTypeIceCandidate = "ice_candidate" // ICE candidate for WebRTC

	// Device-to-device sync (encrypted, server can't read)
	MessageTypeSyncRequest = "sync_request" // New device requests state from primary
	MessageTypeSyncData    = "sync_data"    // Primary sends encrypted state to new device
	MessageTypeSyncAck     = "sync_ack"     // New device confirms receipt

	// Media key exchange (encrypted, server can't read)
	MessageTypeMediaKey = "media_key" // Exchange media encryption keys between clients
)

// WebSocketMessage is the envelope for all WebSocket communication
type WebSocketMessage struct {
	Type      string          `json:"type"`
	MessageID uuid.UUID       `json:"messageId,omitempty"` // camelCase to match frontend
	SenderID  uuid.UUID       `json:"sender_id,omitempty"`
	DeviceID  uuid.UUID       `json:"device_id,omitempty"`
	ServerID  string          `json:"server_id,omitempty"` // Track originating server for presence dedup
	Timestamp time.Time       `json:"timestamp,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Signature string          `json:"signature,omitempty"` // HMAC signature for message authentication
	Nonce     string          `json:"nonce,omitempty"`     // Unique nonce for replay protection
}

// EncryptedMessage represents the encrypted payload of a message
// The server CANNOT read the content - only the ciphertext bytes
type EncryptedMessage struct {
	// Target (one of these is required)
	ReceiverID *uuid.UUID `json:"receiver_id,omitempty"` // For direct messages
	GroupID    *uuid.UUID `json:"group_id,omitempty"`    // For group messages

	// Signal Protocol encrypted content
	Ciphertext  []byte `json:"ciphertext"`   // Encrypted message content
	MessageType string `json:"message_type"` // "prekey" or "whisper"

	// For PreKey messages (first message in session)
	PreKeyID       *uint32 `json:"prekey_id,omitempty"`
	SignedPreKeyID *uint32 `json:"signed_prekey_id,omitempty"`
	IdentityKey    []byte  `json:"identity_key,omitempty"` // Sender's identity key
	BaseKey        []byte  `json:"base_key,omitempty"`     // Ephemeral key for this session

	// Media attachment (encrypted separately)
	MediaID   *uuid.UUID `json:"media_id,omitempty"`
	MediaType string     `json:"media_type,omitempty"` // image, video, audio, document

	// Sealed Sender fields (when using sealed sender format)
	SealedSenderCertificateID *uuid.UUID `json:"sealed_sender_certificate_id,omitempty"`
	EphemeralPublicKey        []byte     `json:"ephemeral_public_key,omitempty"` // Ephemeral key for sealed sender decryption
}

// User represents a user in the system
type User struct {
	UserID                uuid.UUID `json:"user_id"`
	PhoneNumber           string    `json:"phone_number"`
	Username              *string   `json:"username,omitempty"`
	DisplayName           *string   `json:"display_name,omitempty"`
	AvatarURL             *string   `json:"avatar_url,omitempty"`
	PublicIdentityKey     string    `json:"public_identity_key"`
	PublicSignedPrekey    string    `json:"public_signed_prekey"`
	SignedPrekeySignature string    `json:"signed_prekey_signature"`
	CreatedAt             time.Time `json:"created_at"`
	LastSeen              time.Time `json:"last_seen"`
	IsActive              bool      `json:"is_active"`
}

// UserKeys contains a user's public keys for establishing E2EE session
type UserKeys struct {
	UserID                uuid.UUID `json:"user_id"`
	IdentityKey           string    `json:"identity_key"`  // Ed25519 public key
	SignedPreKey          string    `json:"signed_prekey"` // Current signed pre-key
	SignedPreKeySignature string    `json:"signed_prekey_signature"`
	SignedPreKeyID        uint32    `json:"signed_prekey_id"`
	OneTimePreKey         *string   `json:"onetime_prekey,omitempty"` // Optional one-time pre-key
	OneTimePreKeyID       *uint32   `json:"onetime_prekey_id,omitempty"`
}

// PreKey represents a one-time pre-key for X3DH
type PreKey struct {
	PreKeyID  uint32 `json:"prekey_id"`
	PublicKey string `json:"public_key"`
}

// Group represents a group chat
type Group struct {
	GroupID     uuid.UUID     `json:"group_id"`
	Name        string        `json:"name"`
	Description *string       `json:"description,omitempty"`
	AvatarURL   *string       `json:"avatar_url,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	CreatedBy   uuid.UUID     `json:"created_by"`
	Members     []GroupMember `json:"members,omitempty"`
}

// GroupMember represents a member of a group
type GroupMember struct {
	UserID   uuid.UUID `json:"user_id"`
	Role     string    `json:"role"` // admin, member
	JoinedAt time.Time `json:"joined_at"`
}

// Device represents a user's device
type Device struct {
	DeviceID        uuid.UUID `json:"device_id"`
	UserID          uuid.UUID `json:"user_id"`
	DeviceName      *string   `json:"device_name,omitempty"`
	DeviceType      string    `json:"device_type"` // mobile, tablet, desktop, web
	PublicDeviceKey string    `json:"public_device_key"`
	RegisteredAt    time.Time `json:"registered_at"`
	LastSeen        time.Time `json:"last_seen"`
	IsActive        bool      `json:"is_active"`
}

// PresenceStatus represents a user's online status
type PresenceStatus struct {
	UserID   uuid.UUID `json:"user_id"`
	IsOnline bool      `json:"is_online"`
	LastSeen time.Time `json:"last_seen"`
}

// MessageStatus represents the delivery status of a message
type MessageStatus struct {
	MessageID uuid.UUID `json:"message_id"`
	Status    string    `json:"status"` // sent, delivered, read
	UpdatedAt time.Time `json:"updated_at"`
}

// AuthRequest represents a login/register request
type AuthRequest struct {
	PhoneNumber string `json:"phone_number"`
}

// AuthVerifyRequest represents a verification code submission
type AuthVerifyRequest struct {
	PhoneNumber string `json:"phone_number"`
	Code        string `json:"code"`
}

// RegisterRequest represents a new user registration
type RegisterRequest struct {
	PhoneNumber           string    `json:"phone_number"`
	Code                  string    `json:"code"`
	Username              *string   `json:"username,omitempty"`
	DisplayName           *string   `json:"display_name,omitempty"`
	PublicIdentityKey     string    `json:"public_identity_key"`
	PublicSignedPrekey    string    `json:"public_signed_prekey"`
	SignedPrekeySignature string    `json:"signed_prekey_signature"`
	PreKeys               []PreKey  `json:"prekeys"` // Initial batch of one-time pre-keys
	DeviceID              uuid.UUID `json:"device_id"`
	DeviceName            *string   `json:"device_name,omitempty"`
	DeviceType            string    `json:"device_type"`
	PublicDeviceKey       string    `json:"public_device_key"`
}

// LoginRequest represents a returning user login on a new device
type LoginRequest struct {
	PhoneNumber           string    `json:"phone_number"`
	Code                  string    `json:"code"`
	DeviceID              uuid.UUID `json:"device_id"`
	DeviceName            string    `json:"device_name"`
	DeviceType            string    `json:"device_type"`
	PublicDeviceKey       string    `json:"public_device_key"`
	PublicIdentityKey     string    `json:"public_identity_key"`
	PublicSignedPrekey    string    `json:"public_signed_prekey"`
	SignedPrekeySignature string    `json:"signed_prekey_signature"`
}

// AuthResponse contains authentication tokens
type AuthResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         User      `json:"user"`
}

// MediaUploadRequest represents a request for a presigned upload URL
type MediaUploadRequest struct {
	FileType       string    `json:"file_type"`       // MIME type
	FileSize       int64     `json:"file_size"`       // Size in bytes
	ConversationID uuid.UUID `json:"conversation_id"` // User or group ID
}

// MediaUploadResponse contains the presigned URL for upload
type MediaUploadResponse struct {
	MediaID   uuid.UUID `json:"media_id"`
	UploadURL string    `json:"upload_url"`
	ExpiresIn int       `json:"expires_in"` // Seconds until URL expires
}
