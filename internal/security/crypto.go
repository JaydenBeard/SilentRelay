package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strings"
)

// EncryptAESGCM encrypts data with AES-256-GCM
func EncryptAESGCM(plaintext, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes for AES-256")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptAESGCM decrypts data encrypted with AES-256-GCM
func DecryptAESGCM(ciphertext, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes for AES-256")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}

	nonce := ciphertext[:gcm.NonceSize()]
	ciphertext = ciphertext[gcm.NonceSize():]

	return gcm.Open(nil, nonce, ciphertext, nil)
}

// HashPhoneNumber creates a SHA-256 hash of a normalized phone number
// Used for privacy-preserving contact discovery
func HashPhoneNumber(phone string) string {
	// Normalize: remove spaces, dashes, parentheses
	normalized := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' || r == '+' {
			return r
		}
		return -1
	}, phone)

	hash := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(hash[:])
}

// ComputeSafetyNumber computes a safety number for key verification
// This is a simplified version - Signal uses a more complex algorithm
func ComputeSafetyNumber(identityKey1, identityKey2 string, phone1, phone2 string) string {
	// Sort by phone number to ensure same result regardless of order
	var combined string
	if phone1 < phone2 {
		combined = identityKey1 + phone1 + identityKey2 + phone2
	} else {
		combined = identityKey2 + phone2 + identityKey1 + phone1
	}

	hash := sha256.Sum256([]byte(combined))

	// Convert to 60-digit number (12 groups of 5 digits)
	// Each group uses 2.5 bytes (20 bits) for 5 digits (0-99999)
	result := make([]byte, 0, 60)
	for i := 0; i < 12; i++ {
		// Take 20 bits worth and mod by 100000
		offset := i * 5 / 2
		var value uint32
		if i%2 == 0 {
			value = uint32(hash[offset])<<12 | uint32(hash[offset+1])<<4 | uint32(hash[offset+2])>>4
		} else {
			value = uint32(hash[offset]&0x0F)<<16 | uint32(hash[offset+1])<<8 | uint32(hash[offset+2])
		}
		value = value % 100000

		// Format as 5 digits with leading zeros
		digits := []byte{
			'0' + byte((value/10000)%10),
			'0' + byte((value/1000)%10),
			'0' + byte((value/100)%10),
			'0' + byte((value/10)%10),
			'0' + byte(value%10),
		}
		result = append(result, digits...)
	}

	return string(result)
}

// FormatSafetyNumber formats a 60-digit safety number for display
// Returns 12 groups of 5 digits
func FormatSafetyNumber(safetyNumber string) string {
	if len(safetyNumber) != 60 {
		return safetyNumber
	}

	groups := make([]string, 12)
	for i := 0; i < 12; i++ {
		groups[i] = safetyNumber[i*5 : i*5+5]
	}

	// Format as 2 rows of 6 groups
	return strings.Join(groups[:6], " ") + "\n" + strings.Join(groups[6:], " ")
}

// GenerateMasterKey generates a new random master key
func GenerateMasterKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}
