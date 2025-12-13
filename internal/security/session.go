package security

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Session represents a secure user session
type Session struct {
	ID           uuid.UUID              `json:"id"`
	UserID       uuid.UUID              `json:"user_id"`
	DeviceID     uuid.UUID              `json:"device_id"`
	TokenHash    string                 `json:"-"`
	Fingerprint  string                 `json:"-"`
	CreatedAt    time.Time              `json:"created_at"`
	ExpiresAt    time.Time              `json:"expires_at"`
	LastUsedAt   time.Time              `json:"last_used_at"`
	LastRotation time.Time              `json:"last_rotation"`
	Metadata     map[string]interface{} `json:"metadata"`
	IsRevoked    bool                   `json:"is_revoked"`
}

// SessionManager handles secure session operations
type SessionManager struct {
	db              *sql.DB
	auditLogger     *AuditLogger
	anomalyDetector *AnomalyDetector
	tokenTTL        time.Duration
	rotationPeriod  time.Duration
}

// NewSessionManager creates a new session manager
func NewSessionManager(db *sql.DB, auditLogger *AuditLogger) *SessionManager {
	return &SessionManager{
		db:              db,
		auditLogger:     auditLogger,
		anomalyDetector: NewAnomalyDetector(),
		tokenTTL:        24 * time.Hour,
		rotationPeriod:  1 * time.Hour,
	}
}

// CreateSession creates a new secure session
func (sm *SessionManager) CreateSession(ctx context.Context, userID, deviceID uuid.UUID, r *http.Request) (*Session, string, error) {
	token, err := SecureRandomToken(64)
	if err != nil {
		return nil, "", err
	}

	hasher := sha256.New()
	hasher.Write([]byte(token))
	tokenHash := hex.EncodeToString(hasher.Sum(nil))

	fingerprint := CreateFingerprint(r)

	now := time.Now().UTC()
	session := &Session{
		ID:           uuid.New(),
		UserID:       userID,
		DeviceID:     deviceID,
		TokenHash:    tokenHash,
		Fingerprint:  fingerprint.Hash(),
		CreatedAt:    now,
		ExpiresAt:    now.Add(sm.tokenTTL),
		LastUsedAt:   now,
		LastRotation: now,
		Metadata: map[string]interface{}{
			"ip":         GetRealIP(r),
			"user_agent": r.UserAgent(),
		},
		IsRevoked: false,
	}

	metadata, err := json.Marshal(session.Metadata)
	if err != nil {
		return nil, "", err
	}
	_, err = sm.db.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, device_id, token_hash, fingerprint, created_at, expires_at, last_used_at, last_rotation, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, session.ID, session.UserID, session.DeviceID, session.TokenHash, session.Fingerprint,
		session.CreatedAt, session.ExpiresAt, session.LastUsedAt, session.LastRotation, metadata)

	if err != nil {
		return nil, "", err
	}

	if sm.auditLogger != nil {
		sm.auditLogger.LogFromRequest(r, &userID, AuditEventSessionCreated, map[string]any{
			"session_id": session.ID,
			"device_id":  deviceID,
		})
	}

	return session, token, nil
}

// ValidateSession validates a session token
func (sm *SessionManager) ValidateSession(ctx context.Context, token string, r *http.Request) (*Session, error) {
	hasher := sha256.New()
	hasher.Write([]byte(token))
	tokenHash := hex.EncodeToString(hasher.Sum(nil))

	session := &Session{}
	var metadata []byte
	err := sm.db.QueryRowContext(ctx, `
		SELECT id, user_id, device_id, fingerprint, created_at, expires_at, last_used_at, last_rotation, metadata
		FROM sessions
		WHERE token_hash = $1 AND is_revoked = false
	`, tokenHash).Scan(
		&session.ID, &session.UserID, &session.DeviceID, &session.Fingerprint,
		&session.CreatedAt, &session.ExpiresAt, &session.LastUsedAt, &session.LastRotation, &metadata,
	)
	if err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiresAt) {
		if err := sm.RevokeSession(ctx, session.ID); err != nil {
			log.Printf("Warning: failed to revoke expired session: %v", err)
		}
		return nil, sql.ErrNoRows
	}

	currentFP := CreateFingerprint(r)
	if session.Fingerprint != currentFP.Hash() {
		ip := GetRealIP(r)
		sm.anomalyDetector.RecordFailure(ip)

		if sm.auditLogger != nil {
			sm.auditLogger.LogFromRequest(r, &session.UserID, AuditEventDeviceSuspicious, map[string]any{
				"session_id": session.ID,
				"reason":     "fingerprint_mismatch",
			})
		}
	}

	if _, err := sm.db.ExecContext(ctx, `UPDATE sessions SET last_used_at = $1 WHERE id = $2`, time.Now().UTC(), session.ID); err != nil {
		log.Printf("Warning: failed to update session last_used_at: %v", err)
	}
	if err := json.Unmarshal(metadata, &session.Metadata); err != nil {
		log.Printf("Warning: failed to unmarshal session metadata: %v", err)
	}
	return session, nil
}

// RotateSession rotates the session token
func (sm *SessionManager) RotateSession(ctx context.Context, oldToken string, r *http.Request) (string, error) {
	session, err := sm.ValidateSession(ctx, oldToken, r)
	if err != nil {
		return "", err
	}

	newToken, err := SecureRandomToken(64)
	if err != nil {
		return "", err
	}

	hasher := sha256.New()
	hasher.Write([]byte(newToken))
	newTokenHash := hex.EncodeToString(hasher.Sum(nil))

	now := time.Now().UTC()
	_, err = sm.db.ExecContext(ctx, `
		UPDATE sessions SET token_hash = $1, last_rotation = $2, expires_at = $3, fingerprint = $4 WHERE id = $5
	`, newTokenHash, now, now.Add(sm.tokenTTL), CreateFingerprint(r).Hash(), session.ID)

	if err != nil {
		return "", err
	}

	return newToken, nil
}

// ShouldRotate checks if session should be rotated
func (sm *SessionManager) ShouldRotate(session *Session) bool {
	return time.Since(session.LastRotation) > sm.rotationPeriod
}

// RevokeSession revokes a session
func (sm *SessionManager) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	_, err := sm.db.ExecContext(ctx, `UPDATE sessions SET is_revoked = true WHERE id = $1`, sessionID)
	return err
}

// RevokeAllSessions revokes all sessions for a user
func (sm *SessionManager) RevokeAllSessions(ctx context.Context, userID uuid.UUID) error {
	_, err := sm.db.ExecContext(ctx, `UPDATE sessions SET is_revoked = true WHERE user_id = $1`, userID)
	return err
}

// GetActiveSessions returns all active sessions for a user
func (sm *SessionManager) GetActiveSessions(ctx context.Context, userID uuid.UUID) ([]*Session, error) {
	rows, err := sm.db.QueryContext(ctx, `
		SELECT id, device_id, created_at, last_used_at, metadata
		FROM sessions WHERE user_id = $1 AND is_revoked = false AND expires_at > NOW()
		ORDER BY last_used_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var sessions []*Session
	for rows.Next() {
		session := &Session{UserID: userID}
		var metadata []byte
		err := rows.Scan(&session.ID, &session.DeviceID, &session.CreatedAt, &session.LastUsedAt, &metadata)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(metadata, &session.Metadata); err != nil {
			log.Printf("Warning: failed to unmarshal session metadata: %v", err)
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}
