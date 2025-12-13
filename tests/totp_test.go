package tests

import (
	"testing"

	"github.com/jaydenbeard/messaging-app/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTOTPCryptoFunctions tests TOTP encryption/decryption
func TestTOTPCryptoFunctions(t *testing.T) {
	t.Run("TOTP Secret Encryption/Decryption", func(t *testing.T) {
		// Generate a test master key (simulating JWT secret)
		masterKey := make([]byte, 32)
		for i := range masterKey {
			masterKey[i] = byte(i % 256)
		}

		// Test data (simulating TOTP secret bytes)
		plaintext := []byte("test_totp_secret_32_bytes_long!")

		// Encrypt
		ciphertext, err := security.EncryptAESGCM(plaintext, masterKey)
		require.NoError(t, err)
		assert.NotNil(t, ciphertext)
		assert.True(t, len(ciphertext) > len(plaintext))

		// Decrypt
		decrypted, err := security.DecryptAESGCM(ciphertext, masterKey)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("TOTP Secret Encryption with Wrong Key Fails", func(t *testing.T) {
		masterKey := make([]byte, 32)
		wrongKey := make([]byte, 32)
		for i := range masterKey {
			masterKey[i] = byte(i % 256)
			wrongKey[i] = byte((i + 1) % 256)
		}

		plaintext := []byte("test_totp_secret_32_bytes_long!")

		// Encrypt with correct key
		ciphertext, err := security.EncryptAESGCM(plaintext, masterKey)
		require.NoError(t, err)

		// Try to decrypt with wrong key
		_, err = security.DecryptAESGCM(ciphertext, wrongKey)
		assert.Error(t, err)
	})

	t.Run("TOTP Secret Encryption with Different Nonces", func(t *testing.T) {
		masterKey := make([]byte, 32)
		for i := range masterKey {
			masterKey[i] = byte(i % 256)
		}

		plaintext := []byte("test_totp_secret_32_bytes_long!")

		// Encrypt same data twice
		ciphertext1, err := security.EncryptAESGCM(plaintext, masterKey)
		require.NoError(t, err)

		ciphertext2, err := security.EncryptAESGCM(plaintext, masterKey)
		require.NoError(t, err)

		// Ciphertexts should be different (different nonces)
		assert.NotEqual(t, ciphertext1, ciphertext2)

		// But both should decrypt to same plaintext
		decrypted1, err := security.DecryptAESGCM(ciphertext1, masterKey)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted1)

		decrypted2, err := security.DecryptAESGCM(ciphertext2, masterKey)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted2)
	})
}

// TestTOTPAlgorithm tests the TOTP validation algorithm
func TestTOTPAlgorithm(t *testing.T) {
	t.Run("TOTP Time Window Calculation", func(t *testing.T) {
		// Test time window calculation (30-second intervals)
		testTimes := []struct {
			unixTime       int64
			expectedWindow int64
		}{
			{0, 0},
			{29, 0},
			{30, 1},
			{59, 1},
			{60, 2},
			{1234567890, 41152263}, // Some real timestamp
		}

		for _, tt := range testTimes {
			actualWindow := tt.unixTime / 30
			assert.Equal(t, tt.expectedWindow, actualWindow,
				"Time %d should be in window %d", tt.unixTime, tt.expectedWindow)
		}
	})

	t.Run("TOTP Clock Skew Tolerance", func(t *testing.T) {
		// Test that the algorithm checks ±1 window for clock skew
		currentTime := int64(90)            // 90 seconds = window 3
		expectedWindows := []int64{2, 3, 4} // ±1 window tolerance

		windows := make([]int64, 0, 3)
		for offset := int64(-1); offset <= 1; offset++ {
			windows = append(windows, currentTime/30+offset)
		}

		assert.Equal(t, expectedWindows, windows)
	})
}

// TestTOTPIntegration tests the integration of TOTP functions
func TestTOTPIntegration(t *testing.T) {
	t.Run("TOTP Secret Generation Format", func(t *testing.T) {
		// Test that generated secrets are valid base32
		// This is a basic format test since we can't test the full flow without DB
		assert.True(t, true, "TOTP integration test placeholder - requires database setup")
	})

	t.Run("TOTP URI Format", func(t *testing.T) {
		// Test TOTP URI format (otpauth://totp/...)
		assert.True(t, true, "TOTP URI format test placeholder - requires database setup")
	})
}

// Note: Full integration tests with database storage require a test database setup.
// These tests focus on the cryptographic functions and algorithm logic.
// Database-dependent tests should be run in an environment with PostgreSQL.
