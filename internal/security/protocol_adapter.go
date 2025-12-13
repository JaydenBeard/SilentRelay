package security

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ProtocolAdapter provides compatibility between frontend Olm and backend Signal implementations
type ProtocolAdapter struct {
	signalProtocol *SignalProtocol
}

// NewProtocolAdapter creates a new protocol adapter
func NewProtocolAdapter() *ProtocolAdapter {
	return &ProtocolAdapter{
		signalProtocol: NewSignalProtocol(),
	}
}

// Frontend types (Olm format)
type FrontendKeyPair struct {
	PrivateKey string `json:"privateKey"` // Base64 encoded
	PublicKey  string `json:"publicKey"`  // Base64 encoded
}

type FrontendSignedPreKey struct {
	KeyId      int    `json:"keyId"`
	PublicKey  string `json:"publicKey"`  // Base64 encoded
	PrivateKey string `json:"privateKey"` // Base64 encoded
	Signature  string `json:"signature"`  // Base64 encoded
	Timestamp  int64  `json:"timestamp"`
}

type FrontendPreKey struct {
	KeyId      int    `json:"keyId"`
	PublicKey  string `json:"publicKey"`  // Base64 encoded
	PrivateKey string `json:"privateKey"` // Base64 encoded
}

type FrontendEncryptedMessage struct {
	Ciphertext  string `json:"ciphertext"`  // Base64 encoded
	MessageType string `json:"messageType"` // "prekey" or "whisper"
}

type FrontendPreKeyBundle struct {
	RegistrationId        int     `json:"registrationId"`
	DeviceId              int     `json:"deviceId"`
	IdentityKey           string  `json:"identityKey"`  // Base64 encoded
	SignedPreKey          string  `json:"signedPreKey"` // Base64 encoded
	SignedPreKeyId        int     `json:"signedPreKeyId"`
	SignedPreKeySignature string  `json:"signedPreKeySignature"` // Base64 encoded
	PreKeyId              *int    `json:"preKeyId,omitempty"`
	PreKey                *string `json:"preKey,omitempty"` // Base64 encoded
}

// Convert between frontend (Olm) and backend (Signal) formats

// Convert FrontendKeyPair to backend KeyPair
func (pa *ProtocolAdapter) convertFrontendKeyPair(frontendKeyPair FrontendKeyPair) (*KeyPair, error) {
	privateKeyBytes, err := base64.StdEncoding.DecodeString(frontendKeyPair.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	publicKeyBytes, err := base64.StdEncoding.DecodeString(frontendKeyPair.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	if len(privateKeyBytes) != 32 || len(publicKeyBytes) != 32 {
		return nil, errors.New("invalid key length - must be 32 bytes")
	}

	var privateKey, publicKey [32]byte
	copy(privateKey[:], privateKeyBytes)
	copy(publicKey[:], publicKeyBytes)

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}, nil
}

// Convert backend KeyPair to FrontendKeyPair
func (pa *ProtocolAdapter) convertBackendKeyPair(backendKeyPair *KeyPair) FrontendKeyPair {
	return FrontendKeyPair{
		PrivateKey: base64.StdEncoding.EncodeToString(backendKeyPair.PrivateKey[:]),
		PublicKey:  base64.StdEncoding.EncodeToString(backendKeyPair.PublicKey[:]),
	}
}

// Convert FrontendSignedPreKey to backend SignedPreKey
func (pa *ProtocolAdapter) convertFrontendSignedPreKey(frontendSignedPreKey FrontendSignedPreKey) (*SignedPreKey, error) {
	publicKeyBytes, err := base64.StdEncoding.DecodeString(frontendSignedPreKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	privateKeyBytes, err := base64.StdEncoding.DecodeString(frontendSignedPreKey.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	signatureBytes, err := base64.StdEncoding.DecodeString(frontendSignedPreKey.Signature)
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %w", err)
	}

	if len(publicKeyBytes) != 32 || len(privateKeyBytes) != 32 {
		return nil, errors.New("invalid key length - must be 32 bytes")
	}

	var publicKey, privateKey [32]byte
	copy(publicKey[:], publicKeyBytes)
	copy(privateKey[:], privateKeyBytes)

	return &SignedPreKey{
		KeyPair: KeyPair{
			PrivateKey: privateKey,
			PublicKey:  publicKey,
		},
		Signature: signatureBytes,
		KeyID:     uint32(frontendSignedPreKey.KeyId),
	}, nil
}

// Convert backend SignedPreKey to FrontendSignedPreKey
func (pa *ProtocolAdapter) convertBackendSignedPreKey(backendSignedPreKey *SignedPreKey) FrontendSignedPreKey {
	return FrontendSignedPreKey{
		KeyId:      int(backendSignedPreKey.KeyID),
		PublicKey:  base64.StdEncoding.EncodeToString(backendSignedPreKey.PublicKey[:]),
		PrivateKey: base64.StdEncoding.EncodeToString(backendSignedPreKey.PrivateKey[:]),
		Signature:  base64.StdEncoding.EncodeToString(backendSignedPreKey.Signature),
		Timestamp:  int64(backendSignedPreKey.KeyID), // Use key ID as timestamp for compatibility
	}
}

// Convert FrontendPreKey to backend OneTimePreKey
func (pa *ProtocolAdapter) convertFrontendPreKey(frontendPreKey FrontendPreKey) (*OneTimePreKey, error) {
	publicKeyBytes, err := base64.StdEncoding.DecodeString(frontendPreKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	privateKeyBytes, err := base64.StdEncoding.DecodeString(frontendPreKey.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	if len(publicKeyBytes) != 32 || len(privateKeyBytes) != 32 {
		return nil, errors.New("invalid key length - must be 32 bytes")
	}

	var publicKey, privateKey [32]byte
	copy(publicKey[:], publicKeyBytes)
	copy(privateKey[:], privateKeyBytes)

	return &OneTimePreKey{
		KeyPair: KeyPair{
			PrivateKey: privateKey,
			PublicKey:  publicKey,
		},
		KeyID: uint32(frontendPreKey.KeyId),
	}, nil
}

// Convert backend OneTimePreKey to FrontendPreKey
func (pa *ProtocolAdapter) convertBackendPreKey(backendPreKey *OneTimePreKey) FrontendPreKey {
	return FrontendPreKey{
		KeyId:      int(backendPreKey.KeyID),
		PublicKey:  base64.StdEncoding.EncodeToString(backendPreKey.PublicKey[:]),
		PrivateKey: base64.StdEncoding.EncodeToString(backendPreKey.PrivateKey[:]),
	}
}

// Convert FrontendEncryptedMessage to backend format
func (pa *ProtocolAdapter) convertFrontendEncryptedMessage(frontendMessage FrontendEncryptedMessage) ([]byte, string, error) {
	ciphertextBytes, err := base64.StdEncoding.DecodeString(frontendMessage.Ciphertext)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	messageType := frontendMessage.MessageType
	if messageType != "prekey" && messageType != "whisper" {
		return nil, "", fmt.Errorf("invalid message type: %s", messageType)
	}

	return ciphertextBytes, messageType, nil
}

// Convert backend encrypted message to FrontendEncryptedMessage
func (pa *ProtocolAdapter) convertBackendEncryptedMessage(ciphertext []byte, messageType string) FrontendEncryptedMessage {
	return FrontendEncryptedMessage{
		Ciphertext:  base64.StdEncoding.EncodeToString(ciphertext),
		MessageType: messageType,
	}
}

// Convert FrontendPreKeyBundle to backend X3DHKeyBundle
func (pa *ProtocolAdapter) convertFrontendPreKeyBundle(frontendBundle FrontendPreKeyBundle) (X3DHKeyBundle, error) {
	identityKeyBytes, err := base64.StdEncoding.DecodeString(frontendBundle.IdentityKey)
	if err != nil {
		return X3DHKeyBundle{}, fmt.Errorf("failed to decode identity key: %w", err)
	}

	signedPreKeyBytes, err := base64.StdEncoding.DecodeString(frontendBundle.SignedPreKey)
	if err != nil {
		return X3DHKeyBundle{}, fmt.Errorf("failed to decode signed pre-key: %w", err)
	}

	signedPreKeySignatureBytes, err := base64.StdEncoding.DecodeString(frontendBundle.SignedPreKeySignature)
	if err != nil {
		return X3DHKeyBundle{}, fmt.Errorf("failed to decode signed pre-key signature: %w", err)
	}

	if len(identityKeyBytes) != 32 || len(signedPreKeyBytes) != 32 {
		return X3DHKeyBundle{}, errors.New("invalid key length - must be 32 bytes")
	}

	var identityKey, signedPreKey [32]byte
	copy(identityKey[:], identityKeyBytes)
	copy(signedPreKey[:], signedPreKeyBytes)

	bundle := X3DHKeyBundle{
		IdentityKey:     identityKey,
		SignedPreKey:    signedPreKey,
		SignedPreKeyID:  uint32(frontendBundle.SignedPreKeyId),
		SignedPreKeySig: signedPreKeySignatureBytes,
		OneTimePreKey:   nil,
		OneTimePreKeyID: nil,
	}

	// Handle optional one-time pre-key
	if frontendBundle.PreKey != nil && frontendBundle.PreKeyId != nil {
		preKeyBytes, err := base64.StdEncoding.DecodeString(*frontendBundle.PreKey)
		if err != nil {
			return X3DHKeyBundle{}, fmt.Errorf("failed to decode one-time pre-key: %w", err)
		}

		if len(preKeyBytes) != 32 {
			return X3DHKeyBundle{}, errors.New("invalid one-time pre-key length - must be 32 bytes")
		}

		var preKey [32]byte
		copy(preKey[:], preKeyBytes)
		bundle.OneTimePreKey = &preKey
		preKeyID := uint32(*frontendBundle.PreKeyId)
		bundle.OneTimePreKeyID = &preKeyID
	}

	return bundle, nil
}

// Convert backend X3DHKeyBundle to FrontendPreKeyBundle
func (pa *ProtocolAdapter) convertBackendPreKeyBundle(backendBundle X3DHKeyBundle) FrontendPreKeyBundle {
	bundle := FrontendPreKeyBundle{
		RegistrationId:        0, // Will be set by caller
		DeviceId:              0, // Will be set by caller
		IdentityKey:           base64.StdEncoding.EncodeToString(backendBundle.IdentityKey[:]),
		SignedPreKey:          base64.StdEncoding.EncodeToString(backendBundle.SignedPreKey[:]),
		SignedPreKeyId:        int(backendBundle.SignedPreKeyID),
		SignedPreKeySignature: base64.StdEncoding.EncodeToString(backendBundle.SignedPreKeySig),
		PreKeyId:              nil,
		PreKey:                nil,
	}

	// Handle optional one-time pre-key
	if backendBundle.OneTimePreKey != nil && backendBundle.OneTimePreKeyID != nil {
		preKeyStr := base64.StdEncoding.EncodeToString(backendBundle.OneTimePreKey[:])
		preKeyId := int(*backendBundle.OneTimePreKeyID)
		bundle.PreKey = &preKeyStr
		bundle.PreKeyId = &preKeyId
	}

	return bundle
}

// Public API methods that handle protocol conversion

// GenerateKeyPair generates a key pair in frontend format
func (pa *ProtocolAdapter) GenerateKeyPair() (FrontendKeyPair, error) {
	backendKeyPair, err := pa.signalProtocol.GenerateKeyPair()
	if err != nil {
		return FrontendKeyPair{}, fmt.Errorf("failed to generate key pair: %w", err)
	}

	return pa.convertBackendKeyPair(backendKeyPair), nil
}

// GenerateSignedPreKey generates a signed pre-key in frontend format
func (pa *ProtocolAdapter) GenerateSignedPreKey(keyId int) (FrontendSignedPreKey, error) {
	// Generate a key pair for the signed pre-key
	keyPair, err := pa.signalProtocol.GenerateKeyPair()
	if err != nil {
		return FrontendSignedPreKey{}, fmt.Errorf("failed to generate key pair: %w", err)
	}

	// For simplicity, we'll use a zero signature for now
	// In a real implementation, this would be signed with the identity key
	signature := make([]byte, 64) // ECDSA signature length

	return FrontendSignedPreKey{
		KeyId:      keyId,
		PublicKey:  base64.StdEncoding.EncodeToString(keyPair.PublicKey[:]),
		PrivateKey: base64.StdEncoding.EncodeToString(keyPair.PrivateKey[:]),
		Signature:  base64.StdEncoding.EncodeToString(signature),
		Timestamp:  0,
	}, nil
}

// GeneratePreKeys generates pre-keys in frontend format
func (pa *ProtocolAdapter) GeneratePreKeys(startId, count int) ([]FrontendPreKey, error) {
	var preKeys []FrontendPreKey

	for i := 0; i < count; i++ {
		keyPair, err := pa.signalProtocol.GenerateKeyPair()
		if err != nil {
			return nil, fmt.Errorf("failed to generate pre-key %d: %w", startId+i, err)
		}

		preKey := FrontendPreKey{
			KeyId:      startId + i,
			PublicKey:  base64.StdEncoding.EncodeToString(keyPair.PublicKey[:]),
			PrivateKey: base64.StdEncoding.EncodeToString(keyPair.PrivateKey[:]),
		}
		preKeys = append(preKeys, preKey)
	}

	return preKeys, nil
}

// EstablishSession establishes a session using frontend format bundle
func (pa *ProtocolAdapter) EstablishSession(localIdentityKey [32]byte, localID, remoteID string, isInitiator bool, frontendBundle FrontendPreKeyBundle) (*SignalSession, error) {
	backendBundle, err := pa.convertFrontendPreKeyBundle(frontendBundle)
	if err != nil {
		return nil, fmt.Errorf("failed to convert pre-key bundle: %w", err)
	}

	session := pa.signalProtocol.NewSignalSession(localIdentityKey, localID, remoteID, isInitiator)
	err = pa.signalProtocol.EstablishSession(session, backendBundle)
	if err != nil {
		return nil, fmt.Errorf("failed to establish session: %w", err)
	}

	return session, nil
}

// EncryptMessageForSession encrypts a message for a session and returns frontend format
func (pa *ProtocolAdapter) EncryptMessageForSession(session *SignalSession, plaintext []byte) (FrontendEncryptedMessage, error) {
	ciphertext, err := pa.signalProtocol.EncryptMessageForSession(session, plaintext)
	if err != nil {
		return FrontendEncryptedMessage{}, fmt.Errorf("failed to encrypt message: %w", err)
	}

	// Determine message type based on session state
	// For simplicity, we'll use "whisper" for all messages after session establishment
	messageType := "whisper"
	if session.State.SendCount == 0 {
		messageType = "prekey"
	}

	return pa.convertBackendEncryptedMessage(ciphertext, messageType), nil
}

// DecryptMessageForSession decrypts a message from frontend format
func (pa *ProtocolAdapter) DecryptMessageForSession(session *SignalSession, frontendMessage FrontendEncryptedMessage) ([]byte, error) {
	ciphertext, messageType, err := pa.convertFrontendEncryptedMessage(frontendMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to convert message: %w", err)
	}

	// Convert message type to backend format
	if messageType == "prekey" {
		// olmMsgType = 0
	} else {
		// olmMsgType = 1
	}

	// Note: The backend Signal implementation doesn't directly support Olm message types
	// We'll use the session-based decryption which handles the protocol differences
	return pa.signalProtocol.DecryptMessageForSession(session, ciphertext)
}

// GetPreKeyBundle returns a pre-key bundle in frontend format
func (pa *ProtocolAdapter) GetPreKeyBundle(identityKey [32]byte, deviceId int) (FrontendPreKeyBundle, error) {
	// Generate a signed pre-key
	signedPreKey, err := pa.GenerateSignedPreKey(1)
	if err != nil {
		return FrontendPreKeyBundle{}, fmt.Errorf("failed to generate signed pre-key: %w", err)
	}

	// For simplicity, we'll use the signed pre-key as the one-time pre-key too
	// In a real implementation, these would be separate
	preKeyId := 1
	preKeyStr := signedPreKey.PublicKey

	return FrontendPreKeyBundle{
		RegistrationId:        1, // Fixed registration ID for simplicity
		DeviceId:              deviceId,
		IdentityKey:           base64.StdEncoding.EncodeToString(identityKey[:]),
		SignedPreKey:          signedPreKey.PublicKey,
		SignedPreKeyId:        signedPreKey.KeyId,
		SignedPreKeySignature: signedPreKey.Signature,
		PreKeyId:              &preKeyId,
		PreKey:                &preKeyStr,
	}, nil
}

// Helper functions for JSON serialization

func (pa *ProtocolAdapter) FrontendKeyPairToJSON(keyPair FrontendKeyPair) (string, error) {
	jsonBytes, err := json.Marshal(keyPair)
	if err != nil {
		return "", fmt.Errorf("failed to marshal key pair: %w", err)
	}
	return string(jsonBytes), nil
}

func (pa *ProtocolAdapter) FrontendKeyPairFromJSON(jsonStr string) (FrontendKeyPair, error) {
	var keyPair FrontendKeyPair
	err := json.Unmarshal([]byte(jsonStr), &keyPair)
	if err != nil {
		return FrontendKeyPair{}, fmt.Errorf("failed to unmarshal key pair: %w", err)
	}
	return keyPair, nil
}

func (pa *ProtocolAdapter) FrontendEncryptedMessageToJSON(message FrontendEncryptedMessage) (string, error) {
	jsonBytes, err := json.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("failed to marshal encrypted message: %w", err)
	}
	return string(jsonBytes), nil
}

func (pa *ProtocolAdapter) FrontendEncryptedMessageFromJSON(jsonStr string) (FrontendEncryptedMessage, error) {
	var message FrontendEncryptedMessage
	err := json.Unmarshal([]byte(jsonStr), &message)
	if err != nil {
		return FrontendEncryptedMessage{}, fmt.Errorf("failed to unmarshal encrypted message: %w", err)
	}
	return message, nil
}

func (pa *ProtocolAdapter) FrontendPreKeyBundleToJSON(bundle FrontendPreKeyBundle) (string, error) {
	jsonBytes, err := json.Marshal(bundle)
	if err != nil {
		return "", fmt.Errorf("failed to marshal pre-key bundle: %w", err)
	}
	return string(jsonBytes), nil
}

func (pa *ProtocolAdapter) FrontendPreKeyBundleFromJSON(jsonStr string) (FrontendPreKeyBundle, error) {
	var bundle FrontendPreKeyBundle
	err := json.Unmarshal([]byte(jsonStr), &bundle)
	if err != nil {
		return FrontendPreKeyBundle{}, fmt.Errorf("failed to unmarshal pre-key bundle: %w", err)
	}
	return bundle, nil
}

// Utility function to convert base64 string to bytes
func Base64ToBytes(base64Str string) ([]byte, error) {
	// Handle both standard and URL-encoded base64
	base64Str = strings.ReplaceAll(base64Str, "-", "+")
	base64Str = strings.ReplaceAll(base64Str, "_", "/")
	// Add padding if needed
	switch len(base64Str) % 4 {
	case 2:
		base64Str += "=="
	case 3:
		base64Str += "="
	}

	return base64.StdEncoding.DecodeString(base64Str)
}

// Utility function to convert bytes to base64 string
func BytesToBase64(bytes []byte) string {
	return base64.StdEncoding.EncodeToString(bytes)
}
