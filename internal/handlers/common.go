package handlers

// Common utilities, validation functions, and shared types for handlers.
// This file contains foundational components used across multiple handler files.

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"crypto/sha256"
	"encoding/hex"

	"github.com/jaydenbeard/messaging-app/internal/config"
)

// ============================================
// VALIDATION CONSTANTS AND PATTERNS
// ============================================

var (
	phoneRegex       = regexp.MustCompile(`^\+[1-9]\d{1,14}$`) // E.164 format
	usernameRegex    = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)  // Alphanumeric, underscore, hyphen
	allowedMimeTypes = map[string]bool{
		"image/jpeg":      true,
		"image/png":       true,
		"image/gif":       true,
		"image/webp":      true,
		"audio/mpeg":      true,
		"audio/wav":       true,
		"audio/ogg":       true,
		"video/mp4":       true,
		"video/webm":      true,
		"application/pdf": true,
		"text/plain":      true,
	}
)

// ============================================
// JSON RESPONSE HELPER
// ============================================

// writeJSON encodes and writes a JSON response with proper error handling.
// This is a helper to address errcheck lint warnings on json.NewEncoder().Encode().
// If encoding fails, it logs the error - the response is already partially written
// so we can't change the status code at this point.
func writeJSON(w http.ResponseWriter, data interface{}) {
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("ERROR: Failed to encode JSON response: %v", err)
	}
}

// ============================================
// VALIDATION FUNCTIONS
// ============================================

// validatePhoneNumber validates phone number format (E.164)
func validatePhoneNumber(phone string) error {
	if phone == "" {
		return fmt.Errorf("phone number is required")
	}
	if !phoneRegex.MatchString(phone) {
		return fmt.Errorf("phone number must be in E.164 format (e.g., +1234567890)")
	}
	return nil
}

// validateUsername validates username format and constraints
func validateUsername(username string) error {
	if len(username) < 3 {
		return fmt.Errorf("username must be at least 3 characters")
	}
	if len(username) > 30 {
		return fmt.Errorf("username must be 30 characters or less")
	}
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("username can only contain letters, numbers, underscores, and hyphens")
	}
	return nil
}

// validateFileUpload validates file upload parameters with configurable limits
func validateFileUpload(fileName, contentType string, fileSize int64, mediaLimits *config.MediaLimitConfig) error {
	if fileName == "" {
		return fmt.Errorf("file name is required")
	}
	if len(fileName) > 255 {
		return fmt.Errorf("file name too long")
	}
	if fileSize <= 0 {
		return fmt.Errorf("file size must be positive")
	}

	// Check size limits based on content type
	var maxSize int64
	switch {
	case strings.HasPrefix(contentType, "image/"):
		maxSize = mediaLimits.MaxImageSize
	case strings.HasPrefix(contentType, "video/"):
		maxSize = mediaLimits.MaxVideoSize
	case strings.HasPrefix(contentType, "audio/"):
		maxSize = mediaLimits.MaxAudioSize
	default:
		maxSize = mediaLimits.MaxFileSize
	}

	if fileSize > maxSize {
		return fmt.Errorf("file size exceeds maximum allowed size of %d bytes for %s", maxSize, contentType)
	}
	if !allowedMimeTypes[contentType] {
		return fmt.Errorf("file type not allowed: %s", contentType)
	}
	return nil
}

// ============================================
// ACCOUNT LOCKOUT TRACKING
// ============================================

// AccountLockoutTracker tracks failed verification attempts for brute force protection
type AccountLockoutTracker struct {
	mu             sync.RWMutex
	failedAttempts map[string]*LockoutInfo
}

// LockoutInfo stores lockout state for an account
type LockoutInfo struct {
	Count       int
	LastAttempt time.Time
	LockedUntil *time.Time
}

var lockoutTracker = &AccountLockoutTracker{
	failedAttempts: make(map[string]*LockoutInfo),
}

// recordFailedAttempt records a failed verification attempt
func (t *AccountLockoutTracker) recordFailedAttempt(phone string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	info, exists := t.failedAttempts[phone]
	if !exists {
		info = &LockoutInfo{Count: 0}
		t.failedAttempts[phone] = info
	}

	info.Count++
	info.LastAttempt = now

	// Lock account after 5 failed attempts
	if info.Count >= 5 {
		lockoutTime := now.Add(1 * time.Hour)
		info.LockedUntil = &lockoutTime
	}
}

// isLocked checks if an account is currently locked
func (t *AccountLockoutTracker) isLocked(phone string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	info, exists := t.failedAttempts[phone]
	if !exists {
		return false
	}

	if info.LockedUntil == nil {
		return false
	}

	return time.Now().Before(*info.LockedUntil)
}

// getLockoutInfo returns lockout information for display
func (t *AccountLockoutTracker) getLockoutInfo(phone string) (bool, *time.Time) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	info, exists := t.failedAttempts[phone]
	if !exists {
		return false, nil
	}

	return info.LockedUntil != nil && time.Now().Before(*info.LockedUntil), info.LockedUntil
}

// ============================================
// IP AND REQUEST UTILITIES
// ============================================

// getClientIP extracts the real client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (from load balancers/proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP (original client)
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			// Validate it's a real IP
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	// Check X-Real-IP header
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		if net.ParseIP(xrip) != nil {
			return xrip
		}
	}

	// Fall back to remote address
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// generateRequestFingerprint creates a fingerprint of the request for tracking
func generateRequestFingerprint(r *http.Request) string {
	// Collect request characteristics for fingerprinting
	parts := []string{
		r.Header.Get("User-Agent"),
		r.Header.Get("Accept-Language"),
		r.Header.Get("Accept-Encoding"),
		// Don't include IP as it may change for legitimate users
	}

	data := strings.Join(parts, "|")
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8]) // Use first 8 bytes for brevity
}

// ============================================
// HEALTH CHECK
// ============================================

// HealthCheck returns server health status
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{
		"status": "healthy",
	})
}
