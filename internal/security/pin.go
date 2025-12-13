package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

var (
	ErrPINTooShort   = errors.New("PIN must be 4 or 6 digits")
	ErrPINNotNumeric = errors.New("PIN must contain only digits")
	ErrPINLocked     = errors.New("PIN locked due to too many failed attempts")
	ErrInvalidPIN    = errors.New("invalid PIN")
)

// Argon2 parameters (OWASP recommended for 2024)
const (
	argonTime    = 3
	argonMemory  = 64 * 1024 // 64MB
	argonThreads = 4
	argonKeyLen  = 32
	saltLength   = 16
)

// PINConfig holds PIN configuration
type PINConfig struct {
	Length int // 4 or 6
}

// HashPIN creates an Argon2id hash of the PIN
func HashPIN(pin string) (string, error) {
	// Validate PIN
	if len(pin) != 4 && len(pin) != 6 {
		return "", ErrPINTooShort
	}

	for _, c := range pin {
		if c < '0' || c > '9' {
			return "", ErrPINNotNumeric
		}
	}

	// Generate random salt
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// Hash with Argon2id
	hash := argon2.IDKey([]byte(pin), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	// Encode as: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory, argonTime, argonThreads, b64Salt, b64Hash), nil
}

// VerifyPIN checks if the PIN matches the hash
func VerifyPIN(pin, encodedHash string) (bool, error) {
	// Validate PIN format
	if len(pin) != 4 && len(pin) != 6 {
		return false, ErrPINTooShort
	}

	for _, c := range pin {
		if c < '0' || c > '9' {
			return false, ErrPINNotNumeric
		}
	}

	// Parse the encoded hash
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, errors.New("invalid hash format")
	}

	// Decode salt and hash
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}

	// Compute hash of provided PIN
	computedHash := argon2.IDKey([]byte(pin), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	// Constant-time comparison
	if subtle.ConstantTimeCompare(computedHash, expectedHash) == 1 {
		return true, nil
	}

	return false, nil
}

// PINLockStatus represents the lock state of a PIN
type PINLockStatus struct {
	IsLocked       bool
	LockedUntil    *time.Time
	FailedAttempts int
	RemainingTime  time.Duration
}

// CheckPINLock checks if PIN is currently locked
func CheckPINLock(lockedUntil *time.Time, failedAttempts int) PINLockStatus {
	status := PINLockStatus{
		FailedAttempts: failedAttempts,
	}

	if lockedUntil != nil && lockedUntil.After(time.Now()) {
		status.IsLocked = true
		status.LockedUntil = lockedUntil
		status.RemainingTime = time.Until(*lockedUntil)
	}

	return status
}

// GetLockDuration returns how long PIN should be locked based on failed attempts
func GetLockDuration(failedAttempts int) time.Duration {
	switch {
	case failedAttempts >= 5:
		return 1 * time.Hour
	case failedAttempts >= 3:
		return 5 * time.Minute
	default:
		return 0
	}
}
