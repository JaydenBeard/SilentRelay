package security

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
)

// ============================================
// KEY TRANSPARENCY
// Prevents server from lying about user keys
// Similar to Certificate Transparency for TLS
// ============================================

// KeyLogEntry represents a logged key change
type KeyLogEntry struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	DeviceID     uuid.UUID `json:"device_id"`
	KeyType      KeyType   `json:"key_type"`
	PublicKey    []byte    `json:"public_key"`
	KeyHash      string    `json:"key_hash"`
	PreviousHash string    `json:"previous_hash"` // Hash chain
	Timestamp    time.Time `json:"timestamp"`
	Signature    []byte    `json:"signature"` // Server's signature
}

// KeyType identifies the type of key
type KeyType string

const (
	KeyTypeIdentity  KeyType = "identity"
	KeyTypeSignedPre KeyType = "signed_prekey"
	KeyTypeOneTime   KeyType = "one_time_prekey"
)

// KeyTransparencyLog manages the key transparency log
type KeyTransparencyLog struct {
	db        *sql.DB
	serverKey []byte // Server's signing key
}

// NewKeyTransparencyLog creates a new key transparency log
func NewKeyTransparencyLog(db *sql.DB, serverKey []byte) *KeyTransparencyLog {
	return &KeyTransparencyLog{
		db:        db,
		serverKey: serverKey,
	}
}

// LogKeyChange records a key change in the transparency log
func (ktl *KeyTransparencyLog) LogKeyChange(ctx context.Context, userID, deviceID uuid.UUID, keyType KeyType, publicKey []byte) (*KeyLogEntry, error) {
	// Get previous entry for hash chain
	var previousHash string
	err := ktl.db.QueryRowContext(ctx, `
		SELECT key_hash FROM key_transparency_log 
		WHERE user_id = $1 
		ORDER BY created_at DESC 
		LIMIT 1
	`, userID).Scan(&previousHash)
	if err == sql.ErrNoRows {
		previousHash = "genesis" // First key for this user
	} else if err != nil {
		return nil, err
	}

	// Create hash of this key
	hasher := sha256.New()
	hasher.Write(publicKey)
	hasher.Write([]byte(previousHash))
	hasher.Write([]byte(userID.String()))
	keyHash := hex.EncodeToString(hasher.Sum(nil))

	// Create entry
	entry := &KeyLogEntry{
		ID:           uuid.New(),
		UserID:       userID,
		DeviceID:     deviceID,
		KeyType:      keyType,
		PublicKey:    publicKey,
		KeyHash:      keyHash,
		PreviousHash: previousHash,
		Timestamp:    time.Now().UTC(),
	}

	// Sign the entry (server attestation)
	entry.Signature = ktl.signEntry(entry)

	// Store in database
	_, err = ktl.db.ExecContext(ctx, `
		INSERT INTO key_transparency_log 
		(id, user_id, device_id, key_type, public_key, key_hash, previous_hash, signature, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, entry.ID, entry.UserID, entry.DeviceID, entry.KeyType, entry.PublicKey,
		entry.KeyHash, entry.PreviousHash, entry.Signature, entry.Timestamp)

	if err != nil {
		return nil, err
	}

	return entry, nil
}

// GetKeyHistory returns the key history for a user
func (ktl *KeyTransparencyLog) GetKeyHistory(ctx context.Context, userID uuid.UUID, limit int) ([]*KeyLogEntry, error) {
	rows, err := ktl.db.QueryContext(ctx, `
		SELECT id, user_id, device_id, key_type, public_key, key_hash, previous_hash, signature, created_at
		FROM key_transparency_log
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var entries []*KeyLogEntry
	for rows.Next() {
		entry := &KeyLogEntry{}
		err := rows.Scan(
			&entry.ID, &entry.UserID, &entry.DeviceID, &entry.KeyType,
			&entry.PublicKey, &entry.KeyHash, &entry.PreviousHash,
			&entry.Signature, &entry.Timestamp,
		)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// VerifyKeyChain verifies the integrity of a user's key chain
func (ktl *KeyTransparencyLog) VerifyKeyChain(ctx context.Context, userID uuid.UUID) (bool, error) {
	entries, err := ktl.GetKeyHistory(ctx, userID, 1000)
	if err != nil {
		return false, err
	}

	if len(entries) == 0 {
		return true, nil // No keys = valid
	}

	// Verify chain in reverse order (oldest first)
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]

		// Verify signature
		if !ktl.verifyEntry(entry) {
			return false, nil
		}

		// Verify hash chain (except for genesis)
		if i < len(entries)-1 {
			expected := entries[i+1].KeyHash
			if entry.PreviousHash != expected {
				return false, nil // Chain broken
			}
		}

		// Verify key hash
		hasher := sha256.New()
		hasher.Write(entry.PublicKey)
		hasher.Write([]byte(entry.PreviousHash))
		hasher.Write([]byte(entry.UserID.String()))
		computed := hex.EncodeToString(hasher.Sum(nil))

		if entry.KeyHash != computed {
			return false, nil // Hash mismatch
		}
	}

	return true, nil
}

// GetLatestKey returns the latest key of a specific type for a user
func (ktl *KeyTransparencyLog) GetLatestKey(ctx context.Context, userID uuid.UUID, keyType KeyType) (*KeyLogEntry, error) {
	entry := &KeyLogEntry{}
	err := ktl.db.QueryRowContext(ctx, `
		SELECT id, user_id, device_id, key_type, public_key, key_hash, previous_hash, signature, created_at
		FROM key_transparency_log
		WHERE user_id = $1 AND key_type = $2
		ORDER BY created_at DESC
		LIMIT 1
	`, userID, keyType).Scan(
		&entry.ID, &entry.UserID, &entry.DeviceID, &entry.KeyType,
		&entry.PublicKey, &entry.KeyHash, &entry.PreviousHash,
		&entry.Signature, &entry.Timestamp,
	)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

// signEntry creates a server signature for the entry
func (ktl *KeyTransparencyLog) signEntry(entry *KeyLogEntry) []byte {
	data, err := json.Marshal(map[string]any{
		"user_id":       entry.UserID,
		"device_id":     entry.DeviceID,
		"key_type":      entry.KeyType,
		"key_hash":      entry.KeyHash,
		"previous_hash": entry.PreviousHash,
		"timestamp":     entry.Timestamp,
	})
	if err != nil {
		// Return empty signature on error - will fail verification
		return []byte{}
	}

	hasher := sha256.New()
	hasher.Write(data)
	hasher.Write(ktl.serverKey)
	return hasher.Sum(nil)
}

// verifyEntry verifies a server signature
func (ktl *KeyTransparencyLog) verifyEntry(entry *KeyLogEntry) bool {
	expected := ktl.signEntry(entry)
	return ConstantTimeCompare(string(entry.Signature), string(expected))
}

// ============================================
// SEALED SENDER
// Hide who's talking to whom from the server
// ============================================

// SealedSenderCertificate proves sender identity without revealing it to server
type SealedSenderCertificate struct {
	SenderID       uuid.UUID `json:"sender_id"`
	SenderDeviceID uuid.UUID `json:"sender_device_id"`
	Expiry         time.Time `json:"expiry"`
	Signature      []byte    `json:"signature"`
}

// SealedSenderMessage wraps a message to hide sender from server
type SealedSenderMessage struct {
	// Encrypted to recipient's identity key
	// Contains: sender certificate + actual message
	EncryptedContent []byte `json:"encrypted_content"`

	// Ephemeral key for decryption (visible to server, but not linked to sender)
	EphemeralKey []byte `json:"ephemeral_key"`
}

// SealedSenderManager handles sealed sender operations
type SealedSenderManager struct {
	db        *sql.DB
	serverKey []byte
}

// NewSealedSenderManager creates a new sealed sender manager
func NewSealedSenderManager(db *sql.DB, serverKey []byte) *SealedSenderManager {
	return &SealedSenderManager{
		db:        db,
		serverKey: serverKey,
	}
}

// IssueCertificate issues a sealed sender certificate for a user
func (ssm *SealedSenderManager) IssueCertificate(userID, deviceID uuid.UUID) (*SealedSenderCertificate, error) {
	cert := &SealedSenderCertificate{
		SenderID:       userID,
		SenderDeviceID: deviceID,
		Expiry:         time.Now().Add(24 * time.Hour),
	}

	// Sign certificate
	data, err := json.Marshal(map[string]any{
		"sender_id":        cert.SenderID,
		"sender_device_id": cert.SenderDeviceID,
		"expiry":           cert.Expiry,
	})
	if err != nil {
		return nil, err
	}

	hasher := sha256.New()
	hasher.Write(data)
	hasher.Write(ssm.serverKey)
	cert.Signature = hasher.Sum(nil)

	return cert, nil
}

// VerifyCertificate verifies a sealed sender certificate
func (ssm *SealedSenderManager) VerifyCertificate(cert *SealedSenderCertificate) bool {
	if time.Now().After(cert.Expiry) {
		return false
	}

	data, err := json.Marshal(map[string]any{
		"sender_id":        cert.SenderID,
		"sender_device_id": cert.SenderDeviceID,
		"expiry":           cert.Expiry,
	})
	if err != nil {
		return false // Cannot verify if marshal fails
	}

	hasher := sha256.New()
	hasher.Write(data)
	hasher.Write(ssm.serverKey)
	expected := hasher.Sum(nil)

	return ConstantTimeCompare(string(cert.Signature), string(expected))
}
