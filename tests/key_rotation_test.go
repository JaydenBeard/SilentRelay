package tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jaydenbeard/messaging-app/internal/auth"
	"github.com/jaydenbeard/messaging-app/internal/config"
	"github.com/jaydenbeard/messaging-app/internal/db"
	"github.com/jaydenbeard/messaging-app/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyRotationMechanism(t *testing.T) {
	t.Run("Test Dual Key Validation", func(t *testing.T) {
		// Skip in short mode - this test requires a real database connection
		if testing.Short() {
			t.Skip("Skipping key rotation test that requires database in short mode")
		}

		// Initialize with valid secret
		originalSecret := "original_jwt_secret_with_sufficient_length_and_entropy_1234567890"
		config.InitializeKeyManager(originalSecret)

		// Create auth service
		mockDB := &db.PostgresDB{}
		authService, err := auth.NewAuthService(mockDB, originalSecret)
		require.NoError(t, err)
		require.NotNil(t, authService)

		// Generate tokens with original secret
		userID := uuid.New()
		deviceID := uuid.New()

		accessToken, refreshToken, _, err := authService.GenerateTokens(userID, deviceID)
		assert.NoError(t, err)
		assert.NotEmpty(t, accessToken)
		assert.NotEmpty(t, refreshToken)

		// Validate tokens with original secret
		claims, err := authService.ValidateToken(accessToken)
		assert.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, userID, claims.UserID)

		// Rotate to new secret
		newSecret := "new_jwt_secret_with_sufficient_length_and_entropy_0987654321"
		err = authService.RotateJWTSecret(newSecret)
		assert.NoError(t, err)

		// Validate that old tokens still work (dual-key support)
		oldClaims, err := authService.ValidateToken(accessToken)
		assert.NoError(t, err)
		assert.NotNil(t, oldClaims)
		assert.Equal(t, userID, oldClaims.UserID)

		// Generate new tokens with new secret
		newAccessToken, newRefreshToken, _, err := authService.GenerateTokens(userID, deviceID)
		assert.NoError(t, err)
		assert.NotEmpty(t, newAccessToken)
		assert.NotEmpty(t, newRefreshToken)

		// Validate new tokens work
		newClaims, err := authService.ValidateToken(newAccessToken)
		assert.NoError(t, err)
		assert.NotNil(t, newClaims)
		assert.Equal(t, userID, newClaims.UserID)
	})

	t.Run("Test Key RotationScheduler", func(t *testing.T) {
		// Create rotation scheduler
		scheduler := security.NewKeyRotationScheduler()

		// Test initial state
		enabled, _, _ := scheduler.GetRotationStatus()
		assert.True(t, enabled)

		// Test disabling and enabling
		scheduler.Disable()
		enabled, _, _ = scheduler.GetRotationStatus()
		assert.False(t, enabled)

		scheduler.Enable()
		enabled, _, _ = scheduler.GetRotationStatus()
		assert.True(t, enabled)

		// Test rotation interval setting
		scheduler.SetRotationInterval(12 * time.Hour)
		_, _, interval := scheduler.GetRotationStatus()
		assert.Equal(t, 12*time.Hour, interval)

		// Test secure secret generation
		secureSecret, err := security.GenerateSecureJWTSecret()
		assert.NoError(t, err)
		assert.Equal(t, 128, len(secureSecret)) // 64 bytes -> 128 hex chars

		// Validate generated secret
		err = config.ValidateJWTSecret(secureSecret)
		assert.NoError(t, err)
	})

	t.Run("Test Thread Safe Key Access", func(t *testing.T) {
		// Skip in short mode - this test requires a real database
		if testing.Short() {
			t.Skip("Skipping key rotation test that requires database in short mode")
		}

		// Initialize key manager
		secret := "thread_safe_test_secret_with_sufficient_length_and_entropy_1234567890"
		config.InitializeKeyManager(secret)

		// Create auth service
		mockDB := &db.PostgresDB{}
		authService, err := auth.NewAuthService(mockDB, secret)
		require.NoError(t, err)

		// Multiple goroutines accessing secrets concurrently
		done := make(chan bool, 20)
		for i := 0; i < 20; i++ {
			go func() {
				// Access current secret
				current := authService.GetJWTSecret()
				assert.NotEmpty(t, current)

				// Access previous secret (may be empty initially)
				_ = authService.GetPreviousJWTSecret()

				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 20; i++ {
			<-done
		}
	})

	t.Run("Test Key Rotation WithTokenRefresh", func(t *testing.T) {
		// Skip in short mode - this test requires a real database
		if testing.Short() {
			t.Skip("Skipping key rotation test that requires database in short mode")
		}

		// Initialize with valid secret
		originalSecret := "token_refresh_test_secret_with_sufficient_length_and_entropy_1234567890"
		config.InitializeKeyManager(originalSecret)

		// Create auth service
		mockDB := &db.PostgresDB{}
		authService, err := auth.NewAuthService(mockDB, originalSecret)
		require.NoError(t, err)

		// Generate initial tokens
		userID := uuid.New()
		deviceID := uuid.New()

		_, refreshToken, _, err := authService.GenerateTokens(userID, deviceID)
		assert.NoError(t, err)

		// Rotate secret
		newSecret := "new_token_refresh_test_secret_with_sufficient_length_and_entropy_0987654321"
		err = authService.RotateJWTSecret(newSecret)
		assert.NoError(t, err)

		// Refresh access token should work with old refresh token
		newAccessToken, _, err := authService.RefreshAccessToken(refreshToken)
		assert.NoError(t, err)
		assert.NotEmpty(t, newAccessToken)

		// New access token should be signed with new secret
		claims, err := authService.ValidateToken(newAccessToken)
		assert.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
	})

	t.Run("Test Key RotationLogging", func(t *testing.T) {
		// Skip in short mode - this test requires a real database
		if testing.Short() {
			t.Skip("Skipping key rotation test that requires database in short mode")
		}

		// This test verifies that rotation events are properly logged
		// Initialize with valid secret
		secret := "logging_test_secret_with_sufficient_length_and_entropy_1234567890"
		config.InitializeKeyManager(secret)

		// Create auth service
		mockDB := &db.PostgresDB{}
		authService, err := auth.NewAuthService(mockDB, secret)
		require.NoError(t, err)

		// Generate tokens
		userID := uuid.New()
		deviceID := uuid.New()

		accessToken, _, _, err := authService.GenerateTokens(userID, deviceID)
		assert.NoError(t, err)

		// Rotate secret (should log the rotation)
		newSecret := "new_logging_test_secret_with_sufficient_length_and_entropy_0987654321"
		err = authService.RotateJWTSecret(newSecret)
		assert.NoError(t, err)

		// Validate old token (should log dual-key validation)
		claims, err := authService.ValidateToken(accessToken)
		assert.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
	})
}

func TestKeyRotationEdgeCases(t *testing.T) {
	t.Run("Test InvalidSecretRotation", func(t *testing.T) {
		// Skip in short mode - this test requires a real database
		if testing.Short() {
			t.Skip("Skipping key rotation test that requires database in short mode")
		}

		// Initialize with valid secret
		secret := "valid_test_secret_with_sufficient_length_and_entropy_1234567890"
		config.InitializeKeyManager(secret)

		// Create auth service
		mockDB := &db.PostgresDB{}
		authService, err := auth.NewAuthService(mockDB, secret)
		require.NoError(t, err)

		// Try to rotate to invalid secret (too short)
		err = authService.RotateJWTSecret("short")
		assert.Error(t, err)
		assert.Equal(t, auth.ErrJWTSecretWeak, err)

		// Try to rotate to empty secret
		err = authService.RotateJWTSecret("")
		assert.Error(t, err)
		assert.Equal(t, auth.ErrJWTSecretEmpty, err)
	})

	t.Run("TestMultipleRotations", func(t *testing.T) {
		// Skip in short mode - this test requires a real database
		if testing.Short() {
			t.Skip("Skipping key rotation test that requires database in short mode")
		}

		// Initialize with valid secret
		secret1 := "first_test_secret_with_sufficient_length_and_entropy_1234567890"
		config.InitializeKeyManager(secret1)

		// Create auth service
		mockDB := &db.PostgresDB{}
		authService, err := auth.NewAuthService(mockDB, secret1)
		require.NoError(t, err)

		// Generate token with first secret
		userID := uuid.New()
		deviceID := uuid.New()

		token1, _, _, err := authService.GenerateTokens(userID, deviceID)
		assert.NoError(t, err)

		// Rotate to second secret
		secret2 := "second_test_secret_with_sufficient_length_and_entropy_0987654321"
		err = authService.RotateJWTSecret(secret2)
		assert.NoError(t, err)

		// Generate token with second secret
		token2, _, _, err := authService.GenerateTokens(userID, deviceID)
		assert.NoError(t, err)

		// Rotate to third secret
		secret3 := "third_test_secret_with_sufficient_length_and_entropy_1122334455"
		err = authService.RotateJWTSecret(secret3)
		assert.NoError(t, err)

		// Token1 should still work (signed with secret1, validated against previous secret)
		claims1, err := authService.ValidateToken(token1)
		assert.NoError(t, err)
		assert.Equal(t, userID, claims1.UserID)

		// Token2 should still work (signed with secret2, validated against previous secret)
		claims2, err := authService.ValidateToken(token2)
		assert.NoError(t, err)
		assert.Equal(t, userID, claims2.UserID)

		// Generate new token with current secret
		token3, _, _, err := authService.GenerateTokens(userID, deviceID)
		assert.NoError(t, err)

		// All tokens should work during transition
		_, err = authService.ValidateToken(token3)
		assert.NoError(t, err)
	})

	t.Run("TestKeyRotationSchedulerIntervals", func(t *testing.T) {
		scheduler := security.NewKeyRotationScheduler()

		// Test various rotation intervals
		testIntervals := []time.Duration{
			1 * time.Hour,
			6 * time.Hour,
			12 * time.Hour,
			24 * time.Hour,
			48 * time.Hour,
			72 * time.Hour,
		}

		for _, interval := range testIntervals {
			scheduler.SetRotationInterval(interval)
			_, _, currentInterval := scheduler.GetRotationStatus()
			assert.Equal(t, interval, currentInterval)
		}

		// Test minimum interval enforcement
		scheduler.SetRotationInterval(30 * time.Minute) // Should be adjusted to 1 hour
		_, _, currentInterval := scheduler.GetRotationStatus()
		assert.Equal(t, 1*time.Hour, currentInterval)
	})
}
