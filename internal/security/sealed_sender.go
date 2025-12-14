package security

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SealedSenderIdentityCertificate represents a certificate that proves sender identity
// without revealing it to the server
type SealedSenderIdentityCertificate struct {
	CertificateID   uuid.UUID `json:"certificate_id"`
	UserID          uuid.UUID `json:"user_id"`
	PublicKey       []byte    `json:"public_key"`
	Expiration      time.Time `json:"expiration"`
	Signature       []byte    `json:"signature"`
	IssuedAt        time.Time `json:"issued_at"`
	CertificateData []byte    `json:"certificate_data"` // PEM-encoded certificate
}

// SealedSenderIdentityCertificateRequest represents a request for a new certificate
type SealedSenderIdentityCertificateRequest struct {
	UserID    uuid.UUID `json:"user_id"`
	PublicKey []byte    `json:"public_key"` // User's identity public key
}

// SealedSenderIdentityCertificateManager handles certificate issuance and verification
type SealedSenderIdentityCertificateManager struct {
	caPrivateKey *ecdsa.PrivateKey
	caPublicKey  *ecdsa.PublicKey
	revokedCerts map[uuid.UUID]time.Time // Certificate ID -> revocation time
	revokedMutex sync.RWMutex
	db           *sql.DB // Database connection for persistence
}

// NewSealedSenderIdentityCertificateManager creates a new certificate manager
func NewSealedSenderIdentityCertificateManager(db *sql.DB) (*SealedSenderIdentityCertificateManager, error) {
	// Generate CA key pair for signing certificates
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CA key: %w", err)
	}

	return &SealedSenderIdentityCertificateManager{
		caPrivateKey: privateKey,
		caPublicKey:  &privateKey.PublicKey,
		revokedCerts: make(map[uuid.UUID]time.Time),
		db:           db,
	}, nil
}

// IssueCertificate creates a new sealed sender certificate for a user
func (m *SealedSenderIdentityCertificateManager) IssueCertificate(userID uuid.UUID, publicKey []byte) (*SealedSenderIdentityCertificate, error) {
	// Validate public key
	if len(publicKey) == 0 {
		return nil, errors.New("public key cannot be empty")
	}

	// Create certificate ID
	certificateID := uuid.New()

	// Set expiration (30 days from now)
	expiration := time.Now().Add(30 * 24 * time.Hour)
	issuedAt := time.Now()

	// Create certificate data
	certData := fmt.Sprintf("%s:%s:%s", certificateID, userID, expiration.Format(time.RFC3339))

	// Sign the certificate data with CA private key
	signature, err := m.signData([]byte(certData))
	if err != nil {
		return nil, fmt.Errorf("failed to sign certificate: %w", err)
	}

	// Create PEM-encoded certificate
	pemData, err := m.createPEMCertificate(certificateID, userID, publicKey, expiration, issuedAt, signature)
	if err != nil {
		return nil, fmt.Errorf("failed to create PEM certificate: %w", err)
	}

	certificate := &SealedSenderIdentityCertificate{
		CertificateID:   certificateID,
		UserID:          userID,
		PublicKey:       publicKey,
		Expiration:      expiration,
		Signature:       signature,
		IssuedAt:        issuedAt,
		CertificateData: pemData,
	}

	return certificate, nil
}

// IssueCertificateWithPersistence creates a new sealed sender certificate and saves it to database
func (m *SealedSenderIdentityCertificateManager) IssueCertificateWithPersistence(userID uuid.UUID, publicKey []byte) (*SealedSenderIdentityCertificate, error) {
	// First issue the certificate
	cert, err := m.IssueCertificate(userID, publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to issue certificate: %w", err)
	}

	// Save to database if db connection is available
	if m.db != nil {
		// Create a temporary certificate struct that matches the database structure
		dbCert := &SealedSenderIdentityCertificate{
			CertificateID:   cert.CertificateID,
			UserID:          cert.UserID,
			PublicKey:       cert.PublicKey,
			Expiration:      cert.Expiration,
			Signature:       cert.Signature,
			IssuedAt:        cert.IssuedAt,
			CertificateData: cert.CertificateData,
		}

		// Save to database
		query := `
			INSERT INTO sealed_sender_certificates
			(certificate_id, user_id, public_key, expiration, issued_at, certificate_data, signature)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (certificate_id) DO UPDATE SET
				public_key = $3,
				expiration = $4,
				certificate_data = $6,
				signature = $7`

		_, err = m.db.Exec(query,
			dbCert.CertificateID,
			dbCert.UserID,
			dbCert.PublicKey,
			dbCert.Expiration,
			dbCert.IssuedAt,
			dbCert.CertificateData,
			dbCert.Signature,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to save certificate to database: %w", err)
		}
	}

	return cert, nil
}

// VerifyCertificate verifies the authenticity and validity of a certificate
func (m *SealedSenderIdentityCertificateManager) VerifyCertificate(cert *SealedSenderIdentityCertificate) (bool, error) {
	// Check if certificate is expired
	if time.Now().After(cert.Expiration) {
		return false, errors.New("certificate has expired")
	}

	// Verify the signature
	certData := fmt.Sprintf("%s:%s:%s", cert.CertificateID, cert.UserID, cert.Expiration.Format(time.RFC3339))
	isValid, err := m.verifySignature([]byte(certData), cert.Signature)
	if err != nil {
		return false, fmt.Errorf("signature verification failed: %w", err)
	}

	if !isValid {
		return false, errors.New("invalid certificate signature")
	}

	// Check if certificate has been revoked
	if m.IsCertificateRevoked(cert.CertificateID) {
		return false, errors.New("certificate has been revoked")
	}

	return true, nil
}

// RevokeCertificate revokes a certificate by its ID
func (m *SealedSenderIdentityCertificateManager) RevokeCertificate(certificateID uuid.UUID) {
	m.revokedMutex.Lock()
	defer m.revokedMutex.Unlock()

	m.revokedCerts[certificateID] = time.Now().UTC()
}

// IsCertificateRevoked checks if a certificate has been revoked
func (m *SealedSenderIdentityCertificateManager) IsCertificateRevoked(certificateID uuid.UUID) bool {
	m.revokedMutex.RLock()
	defer m.revokedMutex.RUnlock()

	_, revoked := m.revokedCerts[certificateID]
	return revoked
}

// CleanupExpiredRevocations removes old revocation entries
func (m *SealedSenderIdentityCertificateManager) CleanupExpiredRevocations() {
	m.revokedMutex.Lock()
	defer m.revokedMutex.Unlock()

	// Keep revocations for 30 days for audit purposes
	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	for certID, revocationTime := range m.revokedCerts {
		if revocationTime.Before(cutoff) {
			delete(m.revokedCerts, certID)
		}
	}
}

// signData signs data with the CA private key
func (m *SealedSenderIdentityCertificateManager) signData(data []byte) ([]byte, error) {
	// Hash the data
	hash := sha256.Sum256(data)

	// Sign the hash
	signature, err := ecdsa.SignASN1(rand.Reader, m.caPrivateKey, hash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to sign data: %w", err)
	}

	return signature, nil
}

// verifySignature verifies a signature with the CA public key
func (m *SealedSenderIdentityCertificateManager) verifySignature(data []byte, signature []byte) (bool, error) {
	// Hash the data
	hash := sha256.Sum256(data)

	// Verify the signature
	valid := ecdsa.VerifyASN1(m.caPublicKey, hash[:], signature)
	if !valid {
		return false, errors.New("signature verification failed")
	}

	return true, nil
}

// createPEMCertificate creates a PEM-encoded certificate
func (m *SealedSenderIdentityCertificateManager) createPEMCertificate(
	certificateID uuid.UUID,
	_ uuid.UUID,
	_ []byte,
	expiration time.Time,
	issuedAt time.Time,
	_ []byte,
) ([]byte, error) {
	// Create a simple certificate structure
	certTemplate := &x509.Certificate{
		SerialNumber:          m.bigIntFromUUID(certificateID),
		NotBefore:             issuedAt,
		NotAfter:              expiration,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	// Create certificate bytes
	certBytes, err := x509.CreateCertificate(rand.Reader, certTemplate, certTemplate, m.caPublicKey, m.caPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode as PEM
	pemBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}

	return pem.EncodeToMemory(pemBlock), nil
}

// GetCAPublicKey returns the CA public key for client verification
func (m *SealedSenderIdentityCertificateManager) GetCAPublicKey() ([]byte, error) {
	return x509.MarshalPKIXPublicKey(m.caPublicKey)
}

// bigIntFromUUID converts UUID to big.Int for certificate serial number
func (m *SealedSenderIdentityCertificateManager) bigIntFromUUID(id uuid.UUID) *big.Int {
	// Convert UUID bytes to big.Int
	bytes := id[:]
	var result big.Int
	result.SetBytes(bytes[:])
	return &result
}

// SealedSenderIdentityMessage represents a message with sealed sender format
type SealedSenderIdentityMessage struct {
	// Encrypted to recipient's identity key
	// Contains: sender certificate + actual message
	EncryptedContent []byte `json:"encrypted_content"`

	// Ephemeral key for decryption (visible to server, but not linked to sender)
	EphemeralPublicKey []byte `json:"ephemeral_public_key"`

	// Certificate ID for verification
	CertificateID uuid.UUID `json:"certificate_id"`
}

// CreateSealedSenderIdentityMessage creates a sealed sender message
func (m *SealedSenderIdentityCertificateManager) CreateSealedSenderIdentityMessage(
	senderCert *SealedSenderIdentityCertificate,
	recipientPublicKey []byte,
	messageContent []byte,
) (*SealedSenderIdentityMessage, error) {
	// Generate ephemeral key pair for this message
	ephemeralPrivateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ephemeral key: %w", err)
	}

	// Serialize the sender certificate
	certData, err := json.Marshal(senderCert)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize certificate: %w", err)
	}

	// Combine certificate and message content using proper format
	// Format: [certificate_length:4][certificate_json][message_content]

	// Security: Check for potential integer overflow before allocation
	totalSize := 4 + len(certData) + len(messageContent)
	if totalSize < 0 || totalSize > 100*1024*1024 { // Max 100MB to prevent DoS
		return nil, errors.New("combined data size exceeds maximum allowed limit")
	}

	// Create buffer for certificate length (4 bytes) + certificate JSON + message content
	combinedData := make([]byte, totalSize)

	// Write certificate length (big-endian)
	combinedData[0] = byte((len(certData) >> 24) & 0xFF)
	combinedData[1] = byte((len(certData) >> 16) & 0xFF)
	combinedData[2] = byte((len(certData) >> 8) & 0xFF)
	combinedData[3] = byte(len(certData) & 0xFF)

	// Copy certificate JSON
	copy(combinedData[4:], certData)

	// Copy message content
	copy(combinedData[4+len(certData):], messageContent)

	// Encrypt the combined data to recipient's public key
	// In a real implementation, this would use proper hybrid encryption
	// For now, we'll use a simple approach
	encryptedContent, err := m.encryptToRecipient(combinedData, recipientPublicKey, ephemeralPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt content: %w", err)
	}

	// Get ephemeral public key
	ephemeralPublicKey, err := x509.MarshalPKIXPublicKey(&ephemeralPrivateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ephemeral public key: %w", err)
	}

	return &SealedSenderIdentityMessage{
		EncryptedContent:   encryptedContent,
		EphemeralPublicKey: ephemeralPublicKey,
		CertificateID:      senderCert.CertificateID,
	}, nil
}

// encryptToRecipient encrypts data to a recipient's public key
func (m *SealedSenderIdentityCertificateManager) encryptToRecipient(
	data []byte,
	recipientPublicKey []byte,
	ephemeralPrivateKey *ecdsa.PrivateKey,
) ([]byte, error) {
	// Parse recipient public key
	pubKey, err := x509.ParsePKIXPublicKey(recipientPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse recipient public key: %w", err)
	}

	ecdhPubKey, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("recipient public key is not ECDSA")
	}

	// Perform ECDH key exchange
	sharedSecret, err := m.deriveSharedSecret(ephemeralPrivateKey, ecdhPubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive shared secret: %w", err)
	}

	// Derive encryption key from shared secret
	encryptionKey := sha256.Sum256(sharedSecret)

	// Encrypt the data using AES-GCM
	ciphertext, err := EncryptAESGCM(data, encryptionKey[:])
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt data: %w", err)
	}

	return ciphertext, nil
}

// deriveSharedSecret performs ECDH key exchange
func (m *SealedSenderIdentityCertificateManager) deriveSharedSecret(
	privateKey *ecdsa.PrivateKey,
	publicKey *ecdsa.PublicKey,
) ([]byte, error) {
	// Perform ECDH
	x, err := privateKey.Curve.ScalarMult(publicKey.X, publicKey.Y, privateKey.D.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to perform ECDH: %v", err)
	}

	// Convert to bytes
	sharedSecret := make([]byte, (privateKey.Curve.Params().BitSize+7)>>3)
	x.FillBytes(sharedSecret)

	return sharedSecret, nil
}

// DecryptSealedSenderIdentityMessage decrypts a sealed sender message
func (m *SealedSenderIdentityCertificateManager) DecryptSealedSenderIdentityMessage(
	sealedMsg *SealedSenderIdentityMessage,
	recipientPrivateKey *ecdsa.PrivateKey,
) ([]byte, *SealedSenderIdentityCertificate, error) {
	// Parse ephemeral public key
	ephemeralPubKey, err := x509.ParsePKIXPublicKey(sealedMsg.EphemeralPublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse ephemeral public key: %w", err)
	}

	ecdhEphemeralKey, ok := ephemeralPubKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, nil, errors.New("ephemeral public key is not ECDSA")
	}

	// Perform ECDH key exchange
	sharedSecret, err := m.deriveSharedSecret(recipientPrivateKey, ecdhEphemeralKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to derive shared secret: %w", err)
	}

	// Derive decryption key
	decryptionKey := sha256.Sum256(sharedSecret)

	// Decrypt the content
	decryptedData, err := DecryptAESGCM(sealedMsg.EncryptedContent, decryptionKey[:])
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt content: %w", err)
	}

	// Extract certificate and message using proper format
	// Format: [certificate_length:4][certificate_json][message_content]
	if len(decryptedData) < 4 {
		return nil, nil, errors.New("invalid decrypted data format: too short")
	}

	// Read certificate length (first 4 bytes)
	certLength := int(decryptedData[0])<<24 | int(decryptedData[1])<<16 | int(decryptedData[2])<<8 | int(decryptedData[3])
	if certLength <= 0 || certLength+4 > len(decryptedData) {
		return nil, nil, errors.New("invalid certificate length")
	}

	// Extract certificate JSON
	certJSON := decryptedData[4 : 4+certLength]
	var cert SealedSenderIdentityCertificate
	err = json.Unmarshal(certJSON, &cert)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal certificate: %w", err)
	}

	// Verify the certificate
	valid, err := m.VerifyCertificate(&cert)
	if err != nil {
		return nil, nil, fmt.Errorf("certificate verification failed: %w", err)
	}

	if !valid {
		return nil, nil, errors.New("invalid certificate")
	}

	// Extract actual message content (after certificate)
	messageContent := decryptedData[4+certLength:]

	return messageContent, &cert, nil
}
