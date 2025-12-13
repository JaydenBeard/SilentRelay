package tests

import (
	"testing"

	"github.com/jaydenbeard/messaging-app/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackwardCompatibilityAESGCM(t *testing.T) {
	t.Run("ExistingEncryptDecryptCompatibility", func(t *testing.T) {
		key := make([]byte, 32)
		for i := range key {
			key[i] = byte(i % 256)
		}

		plaintext := []byte("Test message for backward compatibility")

		// Encrypt using the existing function
		ciphertext, err := security.EncryptAESGCM(plaintext, key)
		require.NoError(t, err)
		assert.NotNil(t, ciphertext)
		assert.True(t, len(ciphertext) > len(plaintext))

		// Decrypt using the existing function
		decrypted, err := security.DecryptAESGCM(ciphertext, key)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("SignalProtocolAESGCMCompatibility", func(t *testing.T) {
		sp := security.NewSignalProtocol()

		key := make([]byte, 32)
		for i := range key {
			key[i] = byte((i + 100) % 256)
		}

		plaintext := []byte("Test message for Signal Protocol AES-GCM")

		// Encrypt using Signal Protocol function
		ciphertext, err := sp.EncryptAESGCM(plaintext, key)
		require.NoError(t, err)
		assert.NotNil(t, ciphertext)
		assert.True(t, len(ciphertext) > len(plaintext))

		// Decrypt using Signal Protocol function
		decrypted, err := sp.DecryptAESGCM(ciphertext, key)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("CrossCompatibilityEncryptExistingDecryptSignal", func(t *testing.T) {
		key := make([]byte, 32)
		for i := range key {
			key[i] = byte(i % 256)
		}

		plaintext := []byte("Cross-compatibility test: existing -> signal")

		// Encrypt with existing function
		ciphertext, err := security.EncryptAESGCM(plaintext, key)
		require.NoError(t, err)

		// Try to decrypt with Signal Protocol function
		sp := security.NewSignalProtocol()
		decrypted, err := sp.DecryptAESGCM(ciphertext, key)

		// This should work since both use the same AES-GCM implementation
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("CrossCompatibilityEncryptSignalDecryptExisting", func(t *testing.T) {
		sp := security.NewSignalProtocol()

		key := make([]byte, 32)
		for i := range key {
			key[i] = byte((i + 50) % 256)
		}

		plaintext := []byte("Cross-compatibility test: signal -> existing")

		// Encrypt with Signal Protocol function
		ciphertext, err := sp.EncryptAESGCM(plaintext, key)
		require.NoError(t, err)

		// Try to decrypt with existing function
		decrypted, err := security.DecryptAESGCM(ciphertext, key)

		// This should work since both use the same AES-GCM implementation
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("ExistingCryptoFunctionsStillWork", func(t *testing.T) {
		// Test that existing crypto functions still work as expected
		phone := "+14155551234"
		hashed := security.HashPhoneNumber(phone)
		assert.NotEmpty(t, hashed)
		assert.Equal(t, 64, len(hashed)) // SHA-256 hex encoded

		// Test safety number computation
		key1 := "test_key_1_12345678901234567890123456789012"
		key2 := "test_key_2_12345678901234567890123456789012"
		phone1 := "+14155551234"
		phone2 := "+14155559876"

		safetyNumber := security.ComputeSafetyNumber(key1, key2, phone1, phone2)
		assert.NotEmpty(t, safetyNumber)
		assert.Equal(t, 60, len(safetyNumber))

		formatted := security.FormatSafetyNumber(safetyNumber)
		assert.Contains(t, formatted, "\n")
		assert.True(t, len(formatted) > 60) // Should include formatting

		// Test master key generation
		masterKey, err := security.GenerateMasterKey()
		require.NoError(t, err)
		assert.Len(t, masterKey, 32)
	})
}
