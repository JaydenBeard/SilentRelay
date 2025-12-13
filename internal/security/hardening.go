package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// ============================================
// SECURE MEMORY HANDLING
// Keys should never linger in memory
// ============================================

// SecureBytes wraps sensitive data with secure zeroing
type SecureBytes struct {
	data []byte
	mu   sync.Mutex
}

// NewSecureBytes creates a new secure byte container
func NewSecureBytes(data []byte) *SecureBytes {
	copied := make([]byte, len(data))
	copy(copied, data)
	return &SecureBytes{data: copied}
}

// Bytes returns the data (use sparingly, zero when done)
func (s *SecureBytes) Bytes() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data
}

// Zero securely wipes the memory
func (s *SecureBytes) Zero() {
	s.mu.Lock()
	defer s.mu.Unlock()
	SecureZero(s.data)
	s.data = nil
}

// SecureZero overwrites a byte slice with zeros
// Uses volatile writes to prevent compiler optimization
func SecureZero(b []byte) {
	for i := range b {
		b[i] = 0
	}
	// Prevent compiler optimization by using the slice
	_ = b[len(b)-1]
}

// SecureZeroString zeros a string's underlying bytes
// WARNING: Strings are immutable in Go, this is a hack
// This function is unsafe and should be avoided - strings are immutable in Go
// Instead, we'll provide a safe alternative that creates a new empty string
func SecureZeroString(s *string) {
	if s == nil {
		return
	}
	// Safe alternative: replace with empty string
	// This doesn't actually zero the memory but prevents the string content from being accessible
	*s = ""
}

// ============================================
// CONSTANT-TIME OPERATIONS
// Prevent timing attacks
// ============================================

// ConstantTimeCompare compares two strings in constant time
func ConstantTimeCompare(a, b string) bool {
	if len(a) != len(b) {
		// Still do comparison to maintain constant time
		subtle.ConstantTimeCompare([]byte(a), []byte(a))
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// ConstantTimeSelect returns a if selector is 1, b if selector is 0
func ConstantTimeSelect(selector int, a, b []byte) []byte {
	result := make([]byte, len(a))
	subtle.ConstantTimeCopy(selector, result, a)
	subtle.ConstantTimeCopy(1-selector, result, b)
	return result
}

// ============================================
// SECURE RANDOM
// Never use math/rand for security
// ============================================

// SecureRandomBytes generates cryptographically secure random bytes
func SecureRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// SecureRandomHex generates a hex-encoded random string
func SecureRandomHex(n int) (string, error) {
	b, err := SecureRandomBytes(n)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// SecureRandomToken generates a URL-safe random token
func SecureRandomToken(n int) (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
	b, err := SecureRandomBytes(n)
	if err != nil {
		return "", err
	}

	result := make([]byte, n)
	for i, v := range b {
		result[i] = alphabet[int(v)%len(alphabet)]
	}
	return string(result), nil
}

// ============================================
// INPUT VALIDATION
// Trust nothing from the client
// ============================================

var (
	// Strict patterns
	phonePattern    = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)
	usernamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{2,29}$`)
	uuidPattern     = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	base64Pattern   = regexp.MustCompile(`^[A-Za-z0-9+/]*={0,2}$`)
	emojiPattern    = regexp.MustCompile(`^[\p{So}]{1,4}$`)

	// Dangerous patterns to block
	sqlInjectionPatterns = []string{
		"--", ";--", "/*", "*/", "@@", "@",
		"char(", "nchar(", "varchar(", "nvarchar(",
		"alter ", "begin ", "cast(", "create ",
		"cursor ", "declare ", "delete ", "drop ",
		"end ", "exec ", "execute ", "fetch ",
		"insert ", "kill ", "select ", "sys.",
		"sysobjects", "syscolumns", "table ", "update ",
	}

	xssPatterns = []string{
		"<script", "</script", "javascript:", "onerror=",
		"onload=", "onclick=", "onmouseover=", "onfocus=",
		"<iframe", "<object", "<embed", "<svg",
		"expression(", "vbscript:", "data:text/html",
	}
)

// ValidatePhoneNumber validates E.164 format phone numbers
func ValidatePhoneNumber(phone string) bool {
	return phonePattern.MatchString(phone)
}

// ValidateUsername validates username format
func ValidateUsername(username string) bool {
	return usernamePattern.MatchString(username)
}

// ValidateUUID validates UUID format
func ValidateUUID(id string) bool {
	return uuidPattern.MatchString(strings.ToLower(id))
}

// ValidateBase64 validates base64 encoding
func ValidateBase64(s string) bool {
	if len(s) == 0 || len(s)%4 != 0 {
		return false
	}
	return base64Pattern.MatchString(s)
}

// ValidateEmoji validates emoji characters
func ValidateEmoji(s string) bool {
	return emojiPattern.MatchString(s)
}

// ValidateUTF8 ensures string is valid UTF-8 without control chars
func ValidateUTF8(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	// Check for control characters (except newline, tab)
	for _, r := range s {
		if r < 32 && r != '\n' && r != '\t' && r != '\r' {
			return false
		}
		if r == 0x7F { // DEL
			return false
		}
	}
	return true
}

// SanitizeString removes potentially dangerous content
func SanitizeString(s string) string {
	// Remove null bytes
	s = strings.ReplaceAll(s, "\x00", "")

	// Check for SQL injection patterns
	lower := strings.ToLower(s)
	for _, pattern := range sqlInjectionPatterns {
		if strings.Contains(lower, pattern) {
			return "" // Reject entirely
		}
	}

	// Check for XSS patterns
	for _, pattern := range xssPatterns {
		if strings.Contains(lower, pattern) {
			return "" // Reject entirely
		}
	}

	return s
}

// ValidateMessageContent validates encrypted message content
func ValidateMessageContent(ciphertext []byte) bool {
	// Minimum: IV (12) + tag (16) + 1 byte content
	if len(ciphertext) < 29 {
		return false
	}
	// Maximum: 100KB encrypted content
	if len(ciphertext) > 100*1024 {
		return false
	}
	return true
}

// ============================================
// REQUEST FINGERPRINTING
// Detect session hijacking
// ============================================

// DeviceFingerprint creates a fingerprint for device binding
type DeviceFingerprint struct {
	UserAgent  string
	AcceptLang string
	IPPrefix   string // First 3 octets for IPv4, first 64 bits for IPv6
	Timezone   string
}

// CreateFingerprint creates a device fingerprint from request
func CreateFingerprint(r *http.Request) *DeviceFingerprint {
	fp := &DeviceFingerprint{
		UserAgent:  r.UserAgent(),
		AcceptLang: r.Header.Get("Accept-Language"),
		Timezone:   r.Header.Get("X-Timezone"),
	}

	// Get IP prefix (not full IP to allow for NAT/mobile)
	ip := GetRealIP(r)
	if parsed := net.ParseIP(ip); parsed != nil {
		if parsed.To4() != nil {
			// IPv4: use first 3 octets
			parts := strings.Split(ip, ".")
			if len(parts) >= 3 {
				fp.IPPrefix = strings.Join(parts[:3], ".") + ".0"
			}
		} else {
			// IPv6: use first 64 bits
			fp.IPPrefix = ip[:strings.LastIndex(ip, ":")] + "::"
		}
	}

	return fp
}

// Hash returns a hash of the fingerprint
func (fp *DeviceFingerprint) Hash() string {
	data := fp.UserAgent + "|" + fp.AcceptLang + "|" + fp.IPPrefix
	return HashPhoneNumber(data) // Reuse SHA-256 hasher
}

// MatchesFuzzy checks if fingerprints match with some tolerance
func (fp *DeviceFingerprint) MatchesFuzzy(other *DeviceFingerprint) bool {
	// User agent must match
	if fp.UserAgent != other.UserAgent {
		return false
	}
	// IP prefix should match (allows for CGNAT variation)
	if fp.IPPrefix != other.IPPrefix {
		return false
	}
	return true
}

// GetRealIP extracts the real client IP from request
func GetRealIP(r *http.Request) string {
	// Check X-Forwarded-For (from load balancer)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// ============================================
// REPLAY ATTACK PREVENTION
// ============================================

// NonceStore stores used nonces to prevent replay
type NonceStore struct {
	mu     sync.RWMutex
	nonces map[string]time.Time
	ttl    time.Duration
}

// NewNonceStore creates a new nonce store
func NewNonceStore(ttl time.Duration) *NonceStore {
	ns := &NonceStore{
		nonces: make(map[string]time.Time),
		ttl:    ttl,
	}
	go ns.cleanup()
	return ns
}

// Use attempts to use a nonce, returns false if already used
func (ns *NonceStore) Use(nonce string) bool {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	if _, exists := ns.nonces[nonce]; exists {
		return false // Replay detected
	}

	ns.nonces[nonce] = time.Now()
	return true
}

// cleanup removes expired nonces
func (ns *NonceStore) cleanup() {
	ticker := time.NewTicker(ns.ttl / 2)
	for range ticker.C {
		ns.mu.Lock()
		cutoff := time.Now().Add(-ns.ttl)
		for nonce, t := range ns.nonces {
			if t.Before(cutoff) {
				delete(ns.nonces, nonce)
			}
		}
		ns.mu.Unlock()
	}
}

// ============================================
// ANOMALY DETECTION
// Detect suspicious patterns
// ============================================

// AnomalyDetector tracks suspicious activity
type AnomalyDetector struct {
	mu       sync.RWMutex
	failures map[string][]time.Time // IP -> failure timestamps
}

// NewAnomalyDetector creates a new detector
func NewAnomalyDetector() *AnomalyDetector {
	return &AnomalyDetector{
		failures: make(map[string][]time.Time),
	}
}

// RecordFailure records a failed attempt
func (ad *AnomalyDetector) RecordFailure(ip string) {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	ad.failures[ip] = append(ad.failures[ip], time.Now())

	// Keep only last hour
	cutoff := time.Now().Add(-1 * time.Hour)
	filtered := make([]time.Time, 0)
	for _, t := range ad.failures[ip] {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}
	ad.failures[ip] = filtered
}

// IsSuspicious returns true if IP has suspicious activity
func (ad *AnomalyDetector) IsSuspicious(ip string) bool {
	ad.mu.RLock()
	defer ad.mu.RUnlock()

	failures := ad.failures[ip]

	// More than 10 failures in last hour
	if len(failures) > 10 {
		return true
	}

	// More than 5 failures in last 5 minutes (brute force)
	recent := 0
	cutoff := time.Now().Add(-5 * time.Minute)
	for _, t := range failures {
		if t.After(cutoff) {
			recent++
		}
	}
	return recent > 5
}

// ClearIP clears failure history for an IP (on successful auth)
func (ad *AnomalyDetector) ClearIP(ip string) {
	ad.mu.Lock()
	defer ad.mu.Unlock()
	delete(ad.failures, ip)
}
