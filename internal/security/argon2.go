package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// ============================================
// ARGON2ID PASSWORD HASHING
// Memory-hard password hashing for secure credential storage
// ============================================

// Argon2Params contains the parameters for Argon2id hashing
type Argon2Params struct {
	// Time parameter (number of iterations)
	Time uint32
	// Memory parameter in KiB
	Memory uint32
	// Parallelism (number of threads)
	Threads uint8
	// Length of the generated key
	KeyLength uint32
	// Salt length
	SaltLength uint32
}

// DefaultArgon2Params returns the recommended parameters for Argon2id
// These parameters provide a good balance between security and performance
// OWASP recommends: time=1, memory=64MB, threads=4 for interactive logins
func DefaultArgon2Params() *Argon2Params {
	return &Argon2Params{
		Time:       1,         // 1 iteration
		Memory:     64 * 1024, // 64 MB (in KiB)
		Threads:    4,         // 4 parallel threads
		KeyLength:  32,        // 256-bit key
		SaltLength: 16,        // 128-bit salt
	}
}

// HighSecurityArgon2Params returns stronger parameters for high-security scenarios
// Use this for master passwords, admin accounts, or sensitive data encryption keys
func HighSecurityArgon2Params() *Argon2Params {
	return &Argon2Params{
		Time:       3,          // 3 iterations
		Memory:     128 * 1024, // 128 MB (in KiB)
		Threads:    4,          // 4 parallel threads
		KeyLength:  32,         // 256-bit key
		SaltLength: 16,         // 128-bit salt
	}
}

// Argon2Hasher provides Argon2id password hashing functionality
type Argon2Hasher struct {
	params *Argon2Params
}

// NewArgon2Hasher creates a new Argon2 hasher with default parameters
func NewArgon2Hasher() *Argon2Hasher {
	return &Argon2Hasher{
		params: DefaultArgon2Params(),
	}
}

// NewArgon2HasherWithParams creates a new Argon2 hasher with custom parameters
func NewArgon2HasherWithParams(params *Argon2Params) *Argon2Hasher {
	if params == nil {
		params = DefaultArgon2Params()
	}
	return &Argon2Hasher{
		params: params,
	}
}

// HashPassword generates an Argon2id hash of the provided password
// Returns a string in the format: $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
func (h *Argon2Hasher) HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("password cannot be empty")
	}

	// Generate a cryptographically secure random salt
	salt := make([]byte, h.params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Generate the Argon2id hash
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		h.params.Time,
		h.params.Memory,
		h.params.Threads,
		h.params.KeyLength,
	)

	// Encode the hash in the standard format
	// Format: $argon2id$v=19$m=<memory>,t=<time>,p=<parallelism>$<base64-salt>$<base64-hash>
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		h.params.Memory,
		h.params.Time,
		h.params.Threads,
		encodedSalt,
		encodedHash,
	)

	return encoded, nil
}

// VerifyPassword compares a password against an Argon2id hash
// Returns true if the password matches, false otherwise
func (h *Argon2Hasher) VerifyPassword(password, encodedHash string) (bool, error) {
	if password == "" || encodedHash == "" {
		return false, errors.New("password and hash cannot be empty")
	}

	// Parse the hash to extract parameters and salt
	params, salt, hash, err := decodeHash(encodedHash)
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	// Generate hash with the same parameters and salt
	computedHash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Time,
		params.Memory,
		params.Threads,
		params.KeyLength,
	)

	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare(hash, computedHash) == 1 {
		return true, nil
	}

	return false, nil
}

// NeedsRehash checks if a hash needs to be updated with new parameters
// This is useful when upgrading security parameters
func (h *Argon2Hasher) NeedsRehash(encodedHash string) (bool, error) {
	params, _, _, err := decodeHash(encodedHash)
	if err != nil {
		return true, err
	}

	// Check if current params are different from stored params
	if params.Memory != h.params.Memory ||
		params.Time != h.params.Time ||
		params.Threads != h.params.Threads ||
		params.KeyLength != h.params.KeyLength {
		return true, nil
	}

	return false, nil
}

// decodeHash parses an encoded Argon2id hash string
func decodeHash(encodedHash string) (*Argon2Params, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return nil, nil, nil, errors.New("invalid hash format")
	}

	// Verify algorithm
	if parts[1] != "argon2id" {
		return nil, nil, nil, errors.New("unsupported algorithm")
	}

	// Parse version
	var version int
	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse version: %w", err)
	}
	if version != argon2.Version {
		return nil, nil, nil, fmt.Errorf("unsupported argon2 version: %d", version)
	}

	// Parse parameters
	params := &Argon2Params{}
	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.Memory, &params.Time, &params.Threads)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Decode salt
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode salt: %w", err)
	}
	// Bounds check to prevent integer overflow (defensive - salt is typically small)
	if len(salt) > 0xFFFFFFFF {
		return nil, nil, nil, fmt.Errorf("salt length exceeds maximum uint32 value")
	}
	params.SaltLength = uint32(len(salt))

	// Decode hash
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode hash: %w", err)
	}
	// Bounds check to prevent integer overflow (defensive - hash is typically small)
	if len(hash) > 0xFFFFFFFF {
		return nil, nil, nil, fmt.Errorf("hash length exceeds maximum uint32 value")
	}
	params.KeyLength = uint32(len(hash))

	return params, salt, hash, nil
}

// ============================================
// CONVENIENCE FUNCTIONS
// ============================================

// HashPasswordDefault hashes a password using default parameters
func HashPasswordDefault(password string) (string, error) {
	hasher := NewArgon2Hasher()
	return hasher.HashPassword(password)
}

// VerifyPasswordDefault verifies a password against a hash using default settings
func VerifyPasswordDefault(password, encodedHash string) (bool, error) {
	hasher := NewArgon2Hasher()
	return hasher.VerifyPassword(password, encodedHash)
}

// HashPasswordHighSecurity hashes a password using high-security parameters
func HashPasswordHighSecurity(password string) (string, error) {
	hasher := NewArgon2HasherWithParams(HighSecurityArgon2Params())
	return hasher.HashPassword(password)
}

// ============================================
// KEY DERIVATION
// ============================================

// DeriveKey derives a cryptographic key from a password using Argon2id
// This is useful for encrypting data with a user-provided password
func DeriveKey(password string, salt []byte, keyLength uint32) ([]byte, error) {
	if password == "" {
		return nil, errors.New("password cannot be empty")
	}
	if len(salt) < 8 {
		return nil, errors.New("salt must be at least 8 bytes")
	}
	if keyLength < 16 {
		return nil, errors.New("key length must be at least 16 bytes")
	}

	params := DefaultArgon2Params()
	key := argon2.IDKey(
		[]byte(password),
		salt,
		params.Time,
		params.Memory,
		params.Threads,
		keyLength,
	)

	return key, nil
}

// DeriveKeyWithParams derives a key using custom parameters
func DeriveKeyWithParams(password string, salt []byte, keyLength uint32, params *Argon2Params) ([]byte, error) {
	if password == "" {
		return nil, errors.New("password cannot be empty")
	}
	if len(salt) < 8 {
		return nil, errors.New("salt must be at least 8 bytes")
	}
	if keyLength < 16 {
		return nil, errors.New("key length must be at least 16 bytes")
	}
	if params == nil {
		params = DefaultArgon2Params()
	}

	key := argon2.IDKey(
		[]byte(password),
		salt,
		params.Time,
		params.Memory,
		params.Threads,
		keyLength,
	)

	return key, nil
}

// GenerateSalt generates a cryptographically secure random salt
func GenerateSalt(length int) ([]byte, error) {
	if length < 8 {
		length = 16 // Minimum recommended salt length
	}
	salt := make([]byte, length)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}
