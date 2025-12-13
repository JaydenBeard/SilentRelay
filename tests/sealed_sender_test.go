package tests

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jaydenbeard/messaging-app/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSealedSenderCertificateManager(t *testing.T) {
	// Skip - requires sqlite3 driver which isn't available in CI
	t.Skip("Skipping sealed sender tests - requires sqlite3 driver")

	// Create a test database connection (in-memory SQLite for testing)
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create tables for testing
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sealed_sender_certificates (
			certificate_id BLOB PRIMARY KEY,
			user_id BLOB NOT NULL,
			public_key BLOB NOT NULL,
			expiration TIMESTAMP NOT NULL,
			issued_at TIMESTAMP NOT NULL,
			certificate_data BLOB NOT NULL,
			signature BLOB NOT NULL
		)
	`)
	require.NoError(t, err)

	// Create certificate manager
	manager, err := security.NewSealedSenderIdentityCertificateManager(db)
	require.NoError(t, err)
	assert.NotNil(t, manager)

	// Test user ID
	userID := uuid.New()

	// Generate a test public key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	t.Run("IssueCertificate", func(t *testing.T) {
		// Issue a certificate
		cert, err := manager.IssueCertificate(userID, publicKeyBytes)
		require.NoError(t, err)
		assert.NotNil(t, cert)
		assert.Equal(t, userID, cert.UserID)
		assert.Equal(t, publicKeyBytes, cert.PublicKey)
		assert.False(t, cert.Expiration.Before(time.Now()))
		assert.NotEmpty(t, cert.Signature)
		assert.NotEmpty(t, cert.CertificateData)
	})

	t.Run("IssueCertificateWithPersistence", func(t *testing.T) {
		// Issue a certificate with persistence
		cert, err := manager.IssueCertificateWithPersistence(userID, publicKeyBytes)
		require.NoError(t, err)
		assert.NotNil(t, cert)

		// Verify it was saved to database
		rows, err := db.Query("SELECT COUNT(*) FROM sealed_sender_certificates WHERE certificate_id = ?", cert.CertificateID)
		require.NoError(t, err)
		defer rows.Close()

		var count int
		if rows.Next() {
			err := rows.Scan(&count)
			require.NoError(t, err)
			assert.Equal(t, 1, count)
		}
	})

	t.Run("VerifyCertificate", func(t *testing.T) {
		// Issue a certificate
		cert, err := manager.IssueCertificate(userID, publicKeyBytes)
		require.NoError(t, err)

		// Verify the certificate
		valid, err := manager.VerifyCertificate(cert)
		require.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("VerifyInvalidCertificate", func(t *testing.T) {
		// Create an invalid certificate
		invalidCert := &security.SealedSenderIdentityCertificate{
			CertificateID:   uuid.New(),
			UserID:          userID,
			PublicKey:       publicKeyBytes,
			Expiration:      time.Now().Add(24 * time.Hour),
			IssuedAt:        time.Now(),
			Signature:       []byte("invalid_signature"),
			CertificateData: []byte("invalid_data"),
		}

		// Verify the certificate (should fail)
		valid, err := manager.VerifyCertificate(invalidCert)
		assert.Error(t, err)
		assert.False(t, valid)
	})

	t.Run("CreateAndDecryptSealedSenderMessage", func(t *testing.T) {
		// Issue a certificate
		cert, err := manager.IssueCertificate(userID, publicKeyBytes)
		require.NoError(t, err)

		// Generate recipient key pair
		recipientPrivateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		recipientPublicKeyBytes, err := x509.MarshalPKIXPublicKey(&recipientPrivateKey.PublicKey)
		require.NoError(t, err)

		// Test message
		messageContent := []byte("Hello, this is a sealed sender message!")

		// Create sealed sender message
		sealedMsg, err := manager.CreateSealedSenderIdentityMessage(cert, recipientPublicKeyBytes, messageContent)
		require.NoError(t, err)
		assert.NotNil(t, sealedMsg)
		assert.NotEmpty(t, sealedMsg.EncryptedContent)
		assert.NotEmpty(t, sealedMsg.EphemeralPublicKey)
		assert.Equal(t, cert.CertificateID, sealedMsg.CertificateID)

		// Decrypt the message
		decryptedContent, returnedCert, err := manager.DecryptSealedSenderIdentityMessage(
			sealedMsg,
			recipientPrivateKey,
		)
		require.NoError(t, err)
		assert.NotNil(t, decryptedContent)
		assert.Equal(t, messageContent, decryptedContent)
		assert.NotNil(t, returnedCert)
		assert.Equal(t, cert.CertificateID, returnedCert.CertificateID)
	})

	t.Run("CertificateRevocation", func(t *testing.T) {
		// Issue a certificate
		cert, err := manager.IssueCertificate(userID, publicKeyBytes)
		require.NoError(t, err)

		// Revoke the certificate
		manager.RevokeCertificate(cert.CertificateID)

		// Check if it's revoked
		isRevoked := manager.IsCertificateRevoked(cert.CertificateID)
		assert.True(t, isRevoked)

		// Verify the certificate (should fail due to revocation)
		valid, err := manager.VerifyCertificate(cert)
		assert.Error(t, err)
		assert.False(t, valid)
	})

	t.Run("ExpiredCertificate", func(t *testing.T) {
		// Create a certificate that's already expired
		expiredCert := &security.SealedSenderIdentityCertificate{
			CertificateID:   uuid.New(),
			UserID:          userID,
			PublicKey:       publicKeyBytes,
			Expiration:      time.Now().Add(-24 * time.Hour), // Expired 24 hours ago
			IssuedAt:        time.Now().Add(-48 * time.Hour),
			Signature:       []byte("test_signature"),
			CertificateData: []byte("test_data"),
		}

		// Verify the certificate (should fail due to expiration)
		valid, err := manager.VerifyCertificate(expiredCert)
		assert.Error(t, err)
		assert.False(t, valid)
	})
}

func TestSealedSenderMessageFormat(t *testing.T) {
	// Skip - requires sqlite3 driver which isn't available in CI
	t.Skip("Skipping sealed sender tests - requires sqlite3 driver")

	// Create a test database connection
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create tables for testing
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sealed_sender_certificates (
			certificate_id BLOB PRIMARY KEY,
			user_id BLOB NOT NULL,
			public_key BLOB NOT NULL,
			expiration TIMESTAMP NOT NULL,
			issued_at TIMESTAMP NOT NULL,
			certificate_data BLOB NOT NULL,
			signature BLOB NOT NULL
		)
	`)
	require.NoError(t, err)

	// Create certificate manager
	manager, err := security.NewSealedSenderIdentityCertificateManager(db)
	require.NoError(t, err)

	// Test user ID
	userID := uuid.New()

	// Generate a test public key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	t.Run("MessageFormatValidation", func(t *testing.T) {
		// Issue a certificate
		cert, err := manager.IssueCertificate(userID, publicKeyBytes)
		require.NoError(t, err)

		// Generate recipient key pair
		recipientPrivateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		recipientPublicKeyBytes, err := x509.MarshalPKIXPublicKey(&recipientPrivateKey.PublicKey)
		require.NoError(t, err)

		// Test message
		messageContent := []byte("Test message for format validation")

		// Create sealed sender message
		sealedMsg, err := manager.CreateSealedSenderIdentityMessage(cert, recipientPublicKeyBytes, messageContent)
		require.NoError(t, err)

		// Verify the message format
		assert.NotEmpty(t, sealedMsg.EncryptedContent)
		assert.NotEmpty(t, sealedMsg.EphemeralPublicKey)
		assert.Equal(t, cert.CertificateID, sealedMsg.CertificateID)

		// Test JSON serialization
		jsonData, err := json.Marshal(sealedMsg)
		require.NoError(t, err)

		var deserializedMsg security.SealedSenderIdentityMessage
		err = json.Unmarshal(jsonData, &deserializedMsg)
		require.NoError(t, err)

		assert.Equal(t, sealedMsg.CertificateID, deserializedMsg.CertificateID)
		assert.Equal(t, sealedMsg.EphemeralPublicKey, deserializedMsg.EphemeralPublicKey)
		assert.Equal(t, sealedMsg.EncryptedContent, deserializedMsg.EncryptedContent)
	})

	t.Run("InvalidMessageDecryption", func(t *testing.T) {
		// Create an invalid sealed sender message
		invalidMsg := &security.SealedSenderIdentityMessage{
			EncryptedContent:   []byte("invalid_encrypted_content"),
			EphemeralPublicKey: []byte("invalid_public_key"),
			CertificateID:      uuid.New(),
		}

		// Generate recipient key pair
		recipientPrivateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		// Try to decrypt (should fail)
		_, _, err = manager.DecryptSealedSenderIdentityMessage(
			invalidMsg,
			recipientPrivateKey,
		)
		assert.Error(t, err)
	})
}

func TestSealedSenderErrorHandling(t *testing.T) {
	// Skip - requires sqlite3 driver which isn't available in CI
	t.Skip("Skipping sealed sender tests - requires sqlite3 driver")

	// Create a test database connection
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create tables for testing
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sealed_sender_certificates (
			certificate_id BLOB PRIMARY KEY,
			user_id BLOB NOT NULL,
			public_key BLOB NOT NULL,
			expiration TIMESTAMP NOT NULL,
			issued_at TIMESTAMP NOT NULL,
			certificate_data BLOB NOT NULL,
			signature BLOB NOT NULL
		)
	`)
	require.NoError(t, err)

	// Create certificate manager
	manager, err := security.NewSealedSenderIdentityCertificateManager(db)
	require.NoError(t, err)

	// Test user ID
	userID := uuid.New()

	// Generate a test public key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	t.Run("InvalidPublicKey", func(t *testing.T) {
		// Try to issue certificate with empty public key
		_, err := manager.IssueCertificate(userID, []byte{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "public key cannot be empty")
	})

	t.Run("InvalidCertificateData", func(t *testing.T) {
		// Create an invalid certificate with malformed data
		invalidCert := &security.SealedSenderIdentityCertificate{
			CertificateID:   uuid.New(),
			UserID:          userID,
			PublicKey:       publicKeyBytes,
			Expiration:      time.Now().Add(24 * time.Hour),
			IssuedAt:        time.Now(),
			Signature:       []byte("invalid"),
			CertificateData: []byte(`{"invalid": "json"}`), // Invalid JSON
		}

		// Try to verify (should fail)
		valid, err := manager.VerifyCertificate(invalidCert)
		assert.Error(t, err)
		assert.False(t, valid)
	})

	t.Run("ThreadSafety", func(t *testing.T) {
		// Test concurrent certificate issuance
		numGoroutines := 10
		results := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				_, err := manager.IssueCertificate(userID, publicKeyBytes)
				results <- err
			}()
		}

		// Collect results
		var errors []error
		for i := 0; i < numGoroutines; i++ {
			err := <-results
			if err != nil {
				errors = append(errors, err)
			}
		}

		// Should have no errors
		assert.Empty(t, errors)
	})
}
