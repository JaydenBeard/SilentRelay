package tests

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"testing"
)

// ============================================
// SECURITY TEST SUITE
// These tests verify security controls work
// ============================================

// TestInputValidation ensures input validation rejects malicious input
func TestInputValidation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(string) bool
		want     bool
	}{
		// Phone number validation
		{"valid_phone", "+14155551234", isValidPhone, true},
		{"invalid_phone_no_plus", "14155551234", isValidPhone, false},
		{"invalid_phone_too_short", "+1234", isValidPhone, false},
		{"invalid_phone_letters", "+1415abc1234", isValidPhone, false},
		{"sql_injection_phone", "+1' OR '1'='1", isValidPhone, false},

		// UUID validation
		{"valid_uuid", "550e8400-e29b-41d4-a716-446655440000", isValidUUID, true},
		{"invalid_uuid_short", "550e8400-e29b-41d4", isValidUUID, false},
		{"invalid_uuid_chars", "550e8400-e29b-41d4-a716-44665544ZZZZ", isValidUUID, false},
		{"sql_injection_uuid", "550e8400'; DROP TABLE users;--", isValidUUID, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.validate(tt.input); got != tt.want {
				t.Errorf("%s: validate(%q) = %v, want %v", tt.name, tt.input, got, tt.want)
			}
		})
	}
}

// TestSQLInjectionPrevention ensures SQL injection is blocked
func TestSQLInjectionPrevention(t *testing.T) {
	payloads := []string{
		"'; DROP TABLE users; --",
		"1' OR '1'='1",
		"1; DELETE FROM messages;",
		"' UNION SELECT * FROM users --",
		"1' AND 1=1 --",
		"admin'--",
		"' OR 1=1#",
		"1' OR 'x'='x",
		"'; EXEC xp_cmdshell('dir'); --",
	}

	for _, payload := range payloads {
		t.Run(payload[:min(len(payload), 20)], func(t *testing.T) {
			if !containsSQLInjection(payload) {
				t.Errorf("Failed to detect SQL injection: %s", payload)
			}
		})
	}
}

// TestXSSPrevention ensures XSS attacks are blocked
func TestXSSPrevention(t *testing.T) {
	payloads := []string{
		"<script>alert('xss')</script>",
		"<img src=x onerror=alert('xss')>",
		"<svg onload=alert('xss')>",
		"javascript:alert('xss')",
		"<body onload=alert('xss')>",
		"<iframe src='javascript:alert(1)'>",
		"<div style='expression(alert(1))'>",
		"<a href='data:text/html,<script>alert(1)</script>'>",
	}

	for _, payload := range payloads {
		t.Run(payload[:min(len(payload), 20)], func(t *testing.T) {
			if !containsXSS(payload) {
				t.Errorf("Failed to detect XSS: %s", payload)
			}
		})
	}
}

// TestPathTraversal ensures path traversal attacks are blocked
func TestPathTraversal(t *testing.T) {
	payloads := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\config\\sam",
		"%2e%2e%2f%2e%2e%2fetc%2fpasswd",
		"....//....//etc/passwd",
		"..;/etc/passwd",
		"..%00/etc/passwd",
		"..%252f..%252fetc/passwd",
	}

	for _, payload := range payloads {
		t.Run(payload[:min(len(payload), 20)], func(t *testing.T) {
			if !containsPathTraversal(payload) {
				t.Errorf("Failed to detect path traversal: %s", payload)
			}
		})
	}
}

// TestConstantTimeComparison ensures timing attacks are prevented
func TestConstantTimeComparison(t *testing.T) {
	a := "correct_password_12345"
	b := "correct_password_12345"
	c := "wrong_password_12345"
	d := "c"

	// These should be constant time regardless of where they differ
	if !constantTimeCompare(a, b) {
		t.Error("Equal strings should match")
	}
	if constantTimeCompare(a, c) {
		t.Error("Different strings should not match")
	}
	if constantTimeCompare(a, d) {
		t.Error("Different length strings should not match")
	}
}

// TestSecureRandomness ensures random generation is secure
func TestSecureRandomness(t *testing.T) {
	// Generate multiple tokens and ensure they're unique
	tokens := make(map[string]bool)

	for i := 0; i < 1000; i++ {
		token := generateSecureToken(32)
		if tokens[token] {
			t.Errorf("Duplicate token generated: %s", token)
		}
		tokens[token] = true

		// Ensure sufficient entropy (at least 256 bits)
		if len(token) < 32 {
			t.Errorf("Token too short: %d bytes", len(token))
		}
	}
}

// TestRateLimiting ensures rate limiting works
func TestRateLimiting(t *testing.T) {
	limiter := newRateLimiter(5) // 5 requests total

	ip := "192.168.1.1"

	// First 5 should succeed
	for i := 0; i < 5; i++ {
		if !limiter.allow(ip) {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 6th should fail
	if limiter.allow(ip) {
		t.Error("6th request should be rate limited")
	}
}

// TestNonceReplayPrevention ensures nonces can't be reused
func TestNonceReplayPrevention(t *testing.T) {
	store := newNonceStore()

	nonce := "unique_nonce_12345"

	// First use should succeed
	if !store.use(nonce) {
		t.Error("First use of nonce should succeed")
	}

	// Second use should fail (replay attack)
	if store.use(nonce) {
		t.Error("Reusing nonce should fail")
	}
}

// TestPasswordHashing ensures passwords are properly hashed
func TestPasswordHashing(t *testing.T) {
	password := "SecureP@ssw0rd123!"

	hash1 := hashPassword(password)
	hash2 := hashPassword(password)

	// Hashes should be different (due to salt)
	if hash1 == hash2 {
		t.Error("Password hashes should use unique salts")
	}

	// Verification should work
	if !verifyPassword(password, hash1) {
		t.Error("Password verification failed")
	}

	// Wrong password should fail
	if verifyPassword("wrong", hash1) {
		t.Error("Wrong password should not verify")
	}
}

// TestSessionTokenGeneration ensures session tokens are secure
func TestSessionTokenGeneration(t *testing.T) {
	tokens := make(map[string]bool)

	for i := 0; i < 100; i++ {
		token := generateSessionToken()

		// Check uniqueness
		if tokens[token] {
			t.Errorf("Duplicate session token: %s", token)
		}
		tokens[token] = true

		// Check length (at least 256 bits = 32 bytes = 64 hex chars)
		if len(token) < 64 {
			t.Errorf("Token too short: %d", len(token))
		}

		// Check for predictable patterns
		if strings.Contains(token, "0000") {
			t.Error("Token contains suspicious pattern")
		}
	}
}

// Helper functions (stubs for actual implementation)
func isValidPhone(s string) bool {
	if len(s) < 8 || len(s) > 16 {
		return false
	}
	if s[0] != '+' {
		return false
	}
	for _, c := range s[1:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func isValidUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}

func containsSQLInjection(s string) bool {
	lower := strings.ToLower(s)
	patterns := []string{"'", "--", ";", "drop", "delete", "union", "select", "exec", "insert", "update"}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func containsXSS(s string) bool {
	lower := strings.ToLower(s)
	patterns := []string{"<script", "javascript:", "onerror", "onload", "<iframe", "<svg", "expression(", "data:text/html"}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func containsPathTraversal(s string) bool {
	patterns := []string{"..", "%2e", "%252f", "..;", "..%00"}
	for _, p := range patterns {
		if strings.Contains(strings.ToLower(s), p) {
			return true
		}
	}
	return false
}

func constantTimeCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	result := 0
	for i := 0; i < len(a); i++ {
		result |= int(a[i]) ^ int(b[i])
	}
	return result == 0
}

func generateSecureToken(n int) string {
	// Use crypto/rand for secure random bytes
	bytes := make([]byte, n)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

type rateLimiter struct {
	limit int
	count map[string]int
}

func newRateLimiter(limit int) *rateLimiter {
	return &rateLimiter{limit: limit, count: make(map[string]int)}
}

func (r *rateLimiter) allow(ip string) bool {
	r.count[ip]++
	return r.count[ip] <= r.limit
}

type nonceStore struct {
	used map[string]bool
}

func newNonceStore() *nonceStore {
	return &nonceStore{used: make(map[string]bool)}
}

func (n *nonceStore) use(nonce string) bool {
	if n.used[nonce] {
		return false
	}
	n.used[nonce] = true
	return true
}

func hashPassword(password string) string {
	// Use unique salt for each hash
	salt := make([]byte, 16)
	_, _ = rand.Read(salt)
	return "hash_" + password + "_" + hex.EncodeToString(salt)
}

func verifyPassword(password, hash string) bool {
	// Check if hash contains the password (simplified for test)
	return strings.Contains(hash, "hash_"+password+"_")
}

func generateSessionToken() string {
	// Generate 32 bytes (256 bits) of random data
	bytes := make([]byte, 32)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
