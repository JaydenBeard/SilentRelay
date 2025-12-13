package security

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ============================================
// CERTIFICATE PINNING
// Prevents MITM even with compromised CAs
// ============================================

// PinnedCerts stores SHA-256 hashes of pinned certificate public keys
type PinnedCerts struct {
	mu           sync.RWMutex
	pins         map[string]bool
	backupPins   map[string]bool
	rotationTime *time.Time
}

// NewPinnedCerts creates a new certificate pinning validator
func NewPinnedCerts(pins []string) *PinnedCerts {
	pc := &PinnedCerts{
		pins:       make(map[string]bool),
		backupPins: make(map[string]bool),
	}
	for _, pin := range pins {
		pc.pins[pin] = true
	}
	return pc
}

// NewPinnedCertsWithBackup creates a validator with primary and backup pins for rotation
func NewPinnedCertsWithBackup(pins, backupPins []string) *PinnedCerts {
	pc := NewPinnedCerts(pins)
	for _, pin := range backupPins {
		pc.backupPins[pin] = true
	}
	return pc
}

// AddPin adds a new pin to the primary pin set
func (pc *PinnedCerts) AddPin(pin string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.pins[pin] = true
}

// AddBackupPin adds a new pin to the backup pin set
func (pc *PinnedCerts) AddBackupPin(pin string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.backupPins[pin] = true
}

// RemovePin removes a pin from the primary pin set
func (pc *PinnedCerts) RemovePin(pin string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	delete(pc.pins, pin)
}

// RotatePins promotes backup pins to primary and clears backup
func (pc *PinnedCerts) RotatePins() {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	// Move backup pins to primary
	pc.pins = pc.backupPins
	pc.backupPins = make(map[string]bool)

	// Record rotation time
	now := time.Now()
	pc.rotationTime = &now
}

// ScheduleRotation schedules a pin rotation at the specified time
func (pc *PinnedCerts) ScheduleRotation(rotationTime time.Time) {
	go func() {
		time.Sleep(time.Until(rotationTime))
		pc.RotatePins()
	}()
}

// GetRotationTime returns the last rotation time
func (pc *PinnedCerts) GetRotationTime() *time.Time {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.rotationTime
}

// VerifyCertificate verifies the certificate against pinned hashes
func (pc *PinnedCerts) VerifyCertificate(rawCerts [][]byte, _ [][]*x509.Certificate) error {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	if len(pc.pins) == 0 && len(pc.backupPins) == 0 {
		return nil // No pins configured
	}

	for _, rawCert := range rawCerts {
		cert, err := x509.ParseCertificate(rawCert)
		if err != nil {
			continue
		}

		hash := GetCertificatePin(cert)

		// Check primary pins
		if pc.pins[hash] {
			return nil // Found matching pin
		}

		// Check backup pins (for rotation transitions)
		if pc.backupPins[hash] {
			return nil // Found matching backup pin
		}
	}

	return fmt.Errorf("certificate pinning validation failed: no matching pin")
}

// ValidatePinFormat checks if a pin string is properly formatted
func ValidatePinFormat(pin string) error {
	// Pin should be base64-encoded SHA-256 hash (44 characters with padding)
	if len(pin) != 44 {
		return fmt.Errorf("invalid pin length: expected 44 characters, got %d", len(pin))
	}

	// Try to decode to verify it's valid base64
	decoded, err := base64.StdEncoding.DecodeString(pin)
	if err != nil {
		return fmt.Errorf("invalid base64 encoding: %w", err)
	}

	// SHA-256 produces 32 bytes
	if len(decoded) != 32 {
		return fmt.Errorf("invalid pin hash length: expected 32 bytes, got %d", len(decoded))
	}

	return nil
}

// ComputePinFromPEM computes a pin from a PEM-encoded certificate
func ComputePinFromPEM(pemData []byte) (string, error) {
	block, rest := pem.Decode(pemData)
	if block == nil {
		return "", fmt.Errorf("failed to decode PEM block")
	}

	// Check if there are multiple blocks or unexpected data
	if len(rest) > 0 {
		return "", fmt.Errorf("unexpected data after PEM block")
	}

	// Only process CERTIFICATE blocks
	if block.Type != "CERTIFICATE" {
		return "", fmt.Errorf("expected CERTIFICATE block, got %s", block.Type)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse certificate: %w", err)
	}

	return GetCertificatePin(cert), nil
}

// GetCertificatePin computes the SHA-256 hash of a certificate's SPKI
func GetCertificatePin(cert *x509.Certificate) string {
	hash := sha256.Sum256(cert.RawSubjectPublicKeyInfo)
	return base64.StdEncoding.EncodeToString(hash[:])
}

// GetAllPins returns all configured pins (primary and backup)
func (pc *PinnedCerts) GetAllPins() (primary []string, backup []string) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	for pin := range pc.pins {
		primary = append(primary, pin)
	}
	for pin := range pc.backupPins {
		backup = append(backup, pin)
	}
	return
}

// CreatePinningHTTPClient creates an HTTP client with certificate pinning
func CreatePinningHTTPClient(pins []string) *http.Client {
	pc := NewPinnedCerts(pins)

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				VerifyPeerCertificate: pc.VerifyCertificate,
				MinVersion:            tls.VersionTLS13, // Require TLS 1.3
			},
		},
	}
}

// CreatePinningHTTPClientWithBackup creates an HTTP client with primary and backup pins
func CreatePinningHTTPClientWithBackup(pins, backupPins []string) *http.Client {
	pc := NewPinnedCertsWithBackup(pins, backupPins)

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				VerifyPeerCertificate: pc.VerifyCertificate,
				MinVersion:            tls.VersionTLS13,
			},
		},
	}
}

// ============================================
// CERTIFICATE PINNING CONFIG FOR CLIENTS
// Mobile/desktop apps should implement this
// ============================================

// CertPinConfig for client applications
type CertPinConfig struct {
	Domain            string   `json:"domain"`
	Pins              []string `json:"pins"`        // Primary pins
	BackupPins        []string `json:"backup_pins"` // Backup pins for rotation
	IncludeSubdomains bool     `json:"include_subdomains"`
	ExpiresAt         int64    `json:"expires_at"` // Unix timestamp
	ReportURI         string   `json:"report_uri"` // Where to report failures
}

// GetDefaultPinConfig returns the pinning configuration for clients
func GetDefaultPinConfig() []CertPinConfig {
	return []CertPinConfig{
		{
			Domain: "api.securemessenger.local",
			Pins: []string{
				// SHA-256 hash of your certificate's SPKI
				// Generate with: openssl x509 -in cert.pem -pubkey -noout | openssl pkey -pubin -outform der | openssl dgst -sha256 -binary | base64
				"BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB=", // Placeholder - replace with real pin
			},
			BackupPins: []string{
				"CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC=", // Backup certificate pin
			},
			IncludeSubdomains: true,
			ExpiresAt:         0, // Never expires
			ReportURI:         "/api/v1/security/pin-failure",
		},
	}
}

// ============================================
// TLS CONFIGURATION
// Secure TLS settings for the server
// ============================================

// GetSecureTLSConfig returns a hardened TLS configuration
func GetSecureTLSConfig() *tls.Config {
	return &tls.Config{
		// Only TLS 1.3 - it fixes many security issues in 1.2
		MinVersion: tls.VersionTLS13,
		MaxVersion: tls.VersionTLS13,

		// Prefer server cipher suites
		PreferServerCipherSuites: true,

		// Only use strong cipher suites (TLS 1.3 only has strong ones)
		CipherSuites: []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_AES_128_GCM_SHA256,
		},

		// Curve preferences
		CurvePreferences: []tls.CurveID{
			tls.X25519, // Most secure and fast
			tls.CurveP384,
		},

		// Session tickets for TLS 1.3
		SessionTicketsDisabled: false,

		// Client authentication (optional, for mTLS)
		// ClientAuth: tls.RequireAndVerifyClientCert,
	}
}

// ============================================
// HPKP HEADER (Deprecated but educational)
// Modern approach: Expect-CT + Certificate Transparency
// ============================================

// GetExpectCTHeader returns the Expect-CT header value
func GetExpectCTHeader() string {
	// max-age: 1 day, enforce, report to endpoint
	return `max-age=86400, enforce, report-uri="/api/v1/security/ct-failure"`
}

// Certificate Transparency verification should be done at the TLS level
// Most modern browsers do this automatically
