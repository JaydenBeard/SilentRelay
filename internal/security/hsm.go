package security

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
)

// ============================================
// HARDWARE SECURITY MODULE (HSM) INTEGRATION
// Keys never leave the HSM - only sign/decrypt ops
// ============================================

// HSMProvider defines the interface for HSM operations
type HSMProvider interface {
	// Key operations
	GenerateKey(ctx context.Context, keyID string, keyType KeyAlgorithm) error
	GetPublicKey(ctx context.Context, keyID string) (crypto.PublicKey, error)
	DeleteKey(ctx context.Context, keyID string) error

	// Cryptographic operations (key never leaves HSM)
	Sign(ctx context.Context, keyID string, digest []byte) ([]byte, error)
	Verify(ctx context.Context, keyID string, digest, signature []byte) (bool, error)
	Decrypt(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error)

	// Key wrapping (for secure key export/import)
	WrapKey(ctx context.Context, wrappingKeyID string, keyToWrap []byte) ([]byte, error)
	UnwrapKey(ctx context.Context, wrappingKeyID string, wrappedKey []byte) ([]byte, error)

	// Health check
	HealthCheck(ctx context.Context) error
}

// KeyAlgorithm defines supported key algorithms
type KeyAlgorithm string

const (
	KeyAlgorithmECDSAP256 KeyAlgorithm = "ECDSA_P256"
	KeyAlgorithmECDSAP384 KeyAlgorithm = "ECDSA_P384"
	KeyAlgorithmRSA2048   KeyAlgorithm = "RSA_2048"
	KeyAlgorithmRSA4096   KeyAlgorithm = "RSA_4096"
	KeyAlgorithmAES256    KeyAlgorithm = "AES_256"
	KeyAlgorithmEd25519   KeyAlgorithm = "ED25519"
)

// HSMKeyID defines well-known key identifiers
const (
	HSMKeyServerSigning    = "server-signing-key"
	HSMKeyServerEncryption = "server-encryption-key"
	HSMKeyBackupMaster     = "backup-master-key"
	HSMKeyAuditSigning     = "audit-signing-key"
)

// ============================================
// AWS CloudHSM / PKCS#11 Implementation
// ============================================

// CloudHSMProvider implements HSMProvider for AWS CloudHSM
type CloudHSMProvider struct {
	clusterID string
	pin       string
	// In production, this would use the PKCS#11 library
}

// NewCloudHSMProvider creates a new CloudHSM provider
func NewCloudHSMProvider(clusterID, pin string) (*CloudHSMProvider, error) {
	return &CloudHSMProvider{
		clusterID: clusterID,
		pin:       pin,
	}, nil
}

// ============================================
// SOFTWARE HSM (Development/Testing Only)
// NEVER use in production!
// ============================================

// SoftwareHSM is a software-based HSM for development
// WARNING: Keys are stored in memory - NOT SECURE FOR PRODUCTION
type SoftwareHSM struct {
	mu      sync.RWMutex
	keys    map[string]*softwareKey
	warning bool
}

type softwareKey struct {
	algorithm  KeyAlgorithm
	privateKey crypto.PrivateKey
	publicKey  crypto.PublicKey
}

// NewSoftwareHSM creates a software HSM (DEVELOPMENT ONLY)
func NewSoftwareHSM() *SoftwareHSM {
	fmt.Println("⚠️  WARNING: Using Software HSM - NOT FOR PRODUCTION USE!")
	return &SoftwareHSM{
		keys:    make(map[string]*softwareKey),
		warning: true,
	}
}

func (s *SoftwareHSM) GenerateKey(ctx context.Context, keyID string, keyType KeyAlgorithm) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var privateKey crypto.PrivateKey
	var publicKey crypto.PublicKey

	switch keyType {
	case KeyAlgorithmECDSAP256:
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return err
		}
		privateKey = key
		publicKey = &key.PublicKey

	case KeyAlgorithmECDSAP384:
		key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		if err != nil {
			return err
		}
		privateKey = key
		publicKey = &key.PublicKey

	default:
		return fmt.Errorf("unsupported key algorithm: %s", keyType)
	}

	s.keys[keyID] = &softwareKey{
		algorithm:  keyType,
		privateKey: privateKey,
		publicKey:  publicKey,
	}

	return nil
}

func (s *SoftwareHSM) GetPublicKey(ctx context.Context, keyID string) (crypto.PublicKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key, ok := s.keys[keyID]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}

	return key.publicKey, nil
}

func (s *SoftwareHSM) DeleteKey(ctx context.Context, keyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.keys, keyID)
	return nil
}

func (s *SoftwareHSM) Sign(ctx context.Context, keyID string, digest []byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key, ok := s.keys[keyID]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}

	switch k := key.privateKey.(type) {
	case *ecdsa.PrivateKey:
		return ecdsa.SignASN1(rand.Reader, k, digest)
	default:
		return nil, errors.New("unsupported key type for signing")
	}
}

func (s *SoftwareHSM) Verify(ctx context.Context, keyID string, digest, signature []byte) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key, ok := s.keys[keyID]
	if !ok {
		return false, fmt.Errorf("key not found: %s", keyID)
	}

	switch k := key.publicKey.(type) {
	case *ecdsa.PublicKey:
		return ecdsa.VerifyASN1(k, digest, signature), nil
	default:
		return false, errors.New("unsupported key type for verification")
	}
}

func (s *SoftwareHSM) Decrypt(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error) {
	return nil, errors.New("decrypt not implemented in software HSM")
}

func (s *SoftwareHSM) WrapKey(ctx context.Context, wrappingKeyID string, keyToWrap []byte) ([]byte, error) {
	return nil, errors.New("key wrapping not implemented in software HSM")
}

func (s *SoftwareHSM) UnwrapKey(ctx context.Context, wrappingKeyID string, wrappedKey []byte) ([]byte, error) {
	return nil, errors.New("key unwrapping not implemented in software HSM")
}

func (s *SoftwareHSM) HealthCheck(ctx context.Context) error {
	return nil
}

// ============================================
// HSM-BACKED SIGNER
// Use this for all server signing operations
// ============================================

// HSMSigner provides signing operations backed by HSM
type HSMSigner struct {
	hsm   HSMProvider
	keyID string
}

// NewHSMSigner creates a new HSM-backed signer
func NewHSMSigner(hsm HSMProvider, keyID string) *HSMSigner {
	return &HSMSigner{
		hsm:   hsm,
		keyID: keyID,
	}
}

// Sign signs the message using the HSM key
func (s *HSMSigner) Sign(message []byte) ([]byte, error) {
	digest := sha256.Sum256(message)
	return s.hsm.Sign(context.Background(), s.keyID, digest[:])
}

// Verify verifies a signature using the HSM key
func (s *HSMSigner) Verify(message, signature []byte) (bool, error) {
	digest := sha256.Sum256(message)
	return s.hsm.Verify(context.Background(), s.keyID, digest[:], signature)
}

// ============================================
// SECURE RANDOM FROM HSM
// Use HSM's TRNG for highest quality randomness
// ============================================

// HSMRandom provides random bytes from HSM's TRNG
type HSMRandom struct {
	// In production, this would use HSMProvider
	// For now, it's a simple wrapper around system random
}

// Read implements io.Reader for crypto/rand compatibility
func (r *HSMRandom) Read(b []byte) (int, error) {
	// In production, this would call HSM's TRNG
	// For now, fall back to system random
	return rand.Read(b)
}

var _ io.Reader = (*HSMRandom)(nil)

// ============================================
// KEY CEREMONY SUPPORT
// For initial HSM key generation with split knowledge
// ============================================

// KeyCeremonyShare represents one share of a split key
type KeyCeremonyShare struct {
	ShareIndex int
	ShareData  []byte
	Checksum   string
}

// GenerateKeyCeremonyShares splits a secret into N shares
// requiring K shares to reconstruct (Shamir's Secret Sharing)
func GenerateKeyCeremonyShares(secret []byte, n, k int) ([]KeyCeremonyShare, error) {
	if k > n {
		return nil, errors.New("threshold cannot exceed total shares")
	}

	// In production, use proper Shamir's Secret Sharing
	// This is a placeholder implementation
	shares := make([]KeyCeremonyShare, n)
	for i := 0; i < n; i++ {
		shareData := make([]byte, len(secret))
		if _, err := rand.Read(shareData); err != nil {
			return nil, fmt.Errorf("failed to generate random share data: %w", err)
		}

		checksum := sha256.Sum256(shareData)
		shares[i] = KeyCeremonyShare{
			ShareIndex: i + 1,
			ShareData:  shareData,
			Checksum:   hex.EncodeToString(checksum[:8]),
		}
	}

	return shares, nil
}

// ReconstructFromShares reconstructs the secret from K shares
func ReconstructFromShares(shares []KeyCeremonyShare, k int) ([]byte, error) {
	if len(shares) < k {
		return nil, fmt.Errorf("need at least %d shares, got %d", k, len(shares))
	}

	// In production, use proper Shamir reconstruction
	return nil, errors.New("not implemented - use proper Shamir library")
}
