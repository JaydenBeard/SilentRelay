package handlers

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	ws "github.com/gorilla/websocket"
	"github.com/jaydenbeard/messaging-app/internal/auth"
	"github.com/jaydenbeard/messaging-app/internal/db"
	"github.com/jaydenbeard/messaging-app/internal/middleware"
	"github.com/jaydenbeard/messaging-app/internal/models"
	security "github.com/jaydenbeard/messaging-app/internal/security"
	"github.com/jaydenbeard/messaging-app/internal/websocket"
)

// Note: Common utilities (validation, writeJSON, lockout tracking) moved to common.go

// ===========================================================================
// WebSocket Security Components
// ===========================================================================

// WebSocketConnectionTracker tracks connection attempts for security monitoring
type WebSocketConnectionTracker struct {
	mu                 sync.RWMutex
	connectionAttempts map[string][]time.Time // IP -> timestamps of attempts
	failedAttempts     map[string][]time.Time // IP -> timestamps of failed attempts
	suspiciousIPs      map[string]time.Time   // IP -> when flagged as suspicious
}

// Global tracker instance
var wsTracker = &WebSocketConnectionTracker{
	connectionAttempts: make(map[string][]time.Time),
	failedAttempts:     make(map[string][]time.Time),
	suspiciousIPs:      make(map[string]time.Time),
}

// Constants for rate limiting and security thresholds
const (
	wsMaxConnectionsPerMinute = 30              // Max WebSocket connections per IP per minute
	wsMaxFailedAttemptsPerMin = 10              // Max failed attempts before flagging
	wsSuspiciousCooldown      = 5 * time.Minute // How long to track suspicious IPs
)

// recordConnectionAttempt records a WebSocket connection attempt
func (t *WebSocketConnectionTracker) recordConnectionAttempt(ip string, success bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-1 * time.Minute)

	// Clean old entries and add new one
	if success {
		attempts := t.connectionAttempts[ip]
		filtered := make([]time.Time, 0)
		for _, ts := range attempts {
			if ts.After(cutoff) {
				filtered = append(filtered, ts)
			}
		}
		t.connectionAttempts[ip] = append(filtered, now)
	} else {
		attempts := t.failedAttempts[ip]
		filtered := make([]time.Time, 0)
		for _, ts := range attempts {
			if ts.After(cutoff) {
				filtered = append(filtered, ts)
			}
		}
		t.failedAttempts[ip] = append(filtered, now)

		// Check if should flag as suspicious
		if len(t.failedAttempts[ip]) >= wsMaxFailedAttemptsPerMin {
			t.suspiciousIPs[ip] = now
			log.Printf("SECURITY: IP %s flagged as suspicious due to %d failed WebSocket attempts", ip, len(t.failedAttempts[ip]))
		}
	}
}

// isRateLimited checks if an IP should be rate limited
func (t *WebSocketConnectionTracker) isRateLimited(ip string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	now := time.Now()
	cutoff := now.Add(-1 * time.Minute)

	// Check if IP is flagged as suspicious
	if flagTime, ok := t.suspiciousIPs[ip]; ok {
		if now.Sub(flagTime) < wsSuspiciousCooldown {
			return true
		}
	}

	// Count recent attempts
	count := 0
	for _, ts := range t.connectionAttempts[ip] {
		if ts.After(cutoff) {
			count++
		}
	}

	return count >= wsMaxConnectionsPerMinute
}

// isSuspicious checks if an IP is flagged as suspicious
func (t *WebSocketConnectionTracker) isSuspicious(ip string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if flagTime, ok := t.suspiciousIPs[ip]; ok {
		if time.Since(flagTime) < wsSuspiciousCooldown {
			return true
		}
	}
	return false
}

// Note: getClientIP and generateRequestFingerprint moved to common.go

var upgrader = ws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// SECURITY: Handle CORS preflight requests
		if r.Method == http.MethodOptions {
			// Allow preflight requests to proceed
			return true
		}

		origin := r.Header.Get("Origin")

		// SECURITY: Validate origin header format
		if origin == "" {
			// In production, reject empty origin to prevent CSRF
			// Only allow in development mode with explicit flag
			if os.Getenv("DEV_MODE") == "true" {
				// Log warning for development mode
				log.Printf("SECURITY WARNING: Empty origin allowed in DEV_MODE for WebSocket connection from IP=%s", getClientIP(r))
				return true
			}
			log.Printf("SECURITY: WebSocket connection rejected - empty origin header from IP=%s", getClientIP(r))
			return false
		}

		// SECURITY: Validate origin format (must be valid URL)
		parsedOrigin, err := url.Parse(origin)
		if err != nil || parsedOrigin.Host == "" {
			log.Printf("SECURITY: WebSocket connection rejected - invalid origin format: %s from IP=%s", origin, getClientIP(r))
			return false
		}

		// SECURITY: Validate origin scheme (must be http or https)
		if parsedOrigin.Scheme != "http" && parsedOrigin.Scheme != "https" {
			log.Printf("SECURITY: WebSocket connection rejected - invalid origin scheme: %s from IP=%s", parsedOrigin.Scheme, getClientIP(r))
			return false
		}

		// Read allowed origins from environment, with fallback defaults
		allowedOriginsEnv := os.Getenv("ALLOWED_ORIGINS")
		if allowedOriginsEnv == "" {
			// Default allowed origins for development
			allowedOriginsEnv = "http://localhost:3000,http://localhost:5173,https://localhost,https://silentrelay.com.au,https://www.silentrelay.com.au"
		}

		allowedOrigins := strings.Split(allowedOriginsEnv, ",")
		for _, allowed := range allowedOrigins {
			allowed = strings.TrimSpace(allowed)
			if allowed == "" {
				continue
			}

			// SECURITY: Exact match required for security
			if origin == allowed {
				return true
			}

			// SECURITY: Allow subdomains for main domains (e.g., app.silentrelay.com.au)
			// Only if the allowed origin ends with a domain (not localhost)
			if !strings.Contains(allowed, "localhost") {
				// Parse allowed origin
				parsedAllowed, err := url.Parse(allowed)
				if err == nil && parsedAllowed.Host != "" {
					// Check if current origin is a subdomain of allowed origin
					if strings.HasSuffix(parsedOrigin.Host, "."+parsedAllowed.Host) ||
						parsedOrigin.Host == parsedAllowed.Host {
						return true
					}
				}
			}
		}

		log.Printf("SECURITY: WebSocket connection rejected - origin %s not in allowed list from IP=%s", origin, getClientIP(r))
		return false
	},
}

// Note: HealthCheck moved to common.go
// Note: Auth handlers (RequestVerificationCode, VerifyCode, Register, Login, RefreshToken) moved to auth_handlers.go
// Note: User handlers (GetCurrentUser, GetUserProfile, UpdateUser, DeleteUser) moved to user_handlers.go
// Note: Device handlers (GetDevices, RemoveDevice, PIN, Device Approval, Keys, Search) moved to device_handlers.go
// Note: Message handlers (GetMessages, UpdateMessageStatus) moved to message_handlers.go
// Note: Group handlers (CreateGroup, GetGroup, AddGroupMember, RemoveGroupMember) moved to message_handlers.go
// Note: Media handlers (GetUploadURL, GetPrivacySettings, UpdatePrivacySetting, GetMediaURL, UploadProxy, DownloadProxy) moved to media_handlers.go

// ================== WebSocket Handler ==================

// handleWebSocketPreflight handles CORS preflight requests for WebSocket connections
func handleWebSocketPreflight(w http.ResponseWriter, r *http.Request) {
	// SECURITY: Set CORS headers for preflight requests
	origin := r.Header.Get("Origin")

	// Validate origin using the same logic as the WebSocket upgrader
	// Read allowed origins from environment, with fallback defaults
	allowedOriginsEnv := os.Getenv("ALLOWED_ORIGINS")
	if allowedOriginsEnv == "" {
		// Default allowed origins for development
		allowedOriginsEnv = "http://localhost:3000,http://localhost:5173,https://localhost,https://silentrelay.com.au,https://www.silentrelay.com.au"
	}

	allowedOrigins := strings.Split(allowedOriginsEnv, ",")
	validOrigin := false

	for _, allowed := range allowedOrigins {
		allowed = strings.TrimSpace(allowed)
		if allowed == "" {
			continue
		}

		// Check for exact match
		if origin == allowed {
			validOrigin = true
			break
		}

		// Check for subdomain match (only for non-localhost domains)
		if !strings.Contains(allowed, "localhost") {
			parsedAllowed, err := url.Parse(allowed)
			if err == nil && parsedAllowed.Host != "" {
				parsedOrigin, err := url.Parse(origin)
				if err == nil && parsedOrigin.Host != "" {
					// Check if current origin is a subdomain of allowed origin
					if strings.HasSuffix(parsedOrigin.Host, "."+parsedAllowed.Host) ||
						parsedOrigin.Host == parsedAllowed.Host {
						validOrigin = true
						break
					}
				}
			}
		}
	}

	if validOrigin {
		// Set CORS headers for valid origins
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Sec-WebSocket-Protocol")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
		w.WriteHeader(http.StatusOK)
	} else {
		// Reject invalid origins
		log.Printf("SECURITY: WebSocket preflight rejected - invalid origin: %s", origin)
		http.Error(w, "Invalid origin", http.StatusForbidden)
	}
}

// ============================================
// BLOCK USER HANDLERS
// ============================================

// BlockUser blocks a user
func BlockUser(database *db.PostgresDB, hub *websocket.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		blockerID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req struct {
			UserID string `json:"user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		blockedID, err := uuid.Parse(req.UserID)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		// Can't block yourself
		if blockerID == blockedID {
			http.Error(w, "Cannot block yourself", http.StatusBadRequest)
			return
		}

		// Insert into blocked_users table
		if err := database.BlockUser(blockerID, blockedID); err != nil {
			log.Printf("Error blocking user: %v", err)
			http.Error(w, "Failed to block user", http.StatusInternalServerError)
			return
		}

		// Send WebSocket notification to the blocked user to remove the conversation
		if hub != nil {
			payload, err := json.Marshal(map[string]string{"blocker_id": blockerID.String()})
			if err != nil {
				log.Printf("Error marshaling block notification: %v", err)
			} else {
				blockNotification := &models.WebSocketMessage{
					Type:    "user_blocked",
					Payload: payload,
				}
				hub.SendToUser(blockedID.String(), blockNotification)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]bool{"success": true})
	}
}

// UnblockUser unblocks a user
func UnblockUser(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		blockerID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req struct {
			UserID string `json:"user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		blockedID, err := uuid.Parse(req.UserID)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		if err := database.UnblockUser(blockerID, blockedID); err != nil {
			log.Printf("Error unblocking user: %v", err)
			http.Error(w, "Failed to unblock user", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]bool{"success": true})
	}
}

// GetBlockedUsers returns list of users blocked by the current user
func GetBlockedUsers(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		blockerID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		blockedUsers, err := database.GetBlockedUsers(blockerID)
		if err != nil {
			log.Printf("Error getting blocked users: %v", err)
			http.Error(w, "Failed to get blocked users", http.StatusInternalServerError)
			return
		}

		// Ensure we return an empty array, not null
		if blockedUsers == nil {
			blockedUsers = []map[string]interface{}{}
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, blockedUsers)
	}
}

// IsBlocked checks if current user is blocked by another user
func IsBlocked(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		currentUserID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		otherUserIDStr := vars["userId"]
		otherUserID, err := uuid.Parse(otherUserIDStr)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		// Check if current user is blocked by the other user
		isBlocked, err := database.IsBlocked(otherUserID, currentUserID)
		if err != nil {
			log.Printf("Error checking block status: %v", err)
			http.Error(w, "Failed to check block status", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]bool{"is_blocked": isBlocked})
	}
}

// WebSocketHandler handles WebSocket upgrade and connection
func WebSocketHandler(hub *websocket.Hub, authService *auth.AuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// SECURITY: Handle CORS preflight requests
		if r.Method == http.MethodOptions {
			handleWebSocketPreflight(w, r)
			return
		}

		// SECURITY: Extract client IP for logging and rate limiting
		clientIP := getClientIP(r)
		requestFingerprint := generateRequestFingerprint(r)

		// SECURITY: Check if IP is rate limited or suspicious
		if wsTracker.isRateLimited(clientIP) {
			log.Printf("SECURITY: WebSocket rate limit exceeded for IP=%s fingerprint=%s", clientIP, requestFingerprint)
			http.Error(w, "Too many connection attempts", http.StatusTooManyRequests)
			return
		}

		if wsTracker.isSuspicious(clientIP) {
			log.Printf("SECURITY: WebSocket connection blocked for suspicious IP=%s fingerprint=%s", clientIP, requestFingerprint)
			http.Error(w, "Connection temporarily blocked", http.StatusForbidden)
			return
		}

		// Get token from multiple sources (in order of preference)
		token := ""

		// 1. Try Authorization header first (preferred for apps)
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			} else {
				token = authHeader
			}
		}

		// 2. Try Sec-WebSocket-Protocol header (browser alternative)
		// Format is "Bearer, <token>" when using WebSocket subprotocols
		if token == "" {
			wsProtocol := r.Header.Get("Sec-WebSocket-Protocol")
			if wsProtocol != "" {
				// Parse "Bearer, <token>" format
				parts := strings.Split(wsProtocol, ", ")
				if len(parts) == 2 && parts[0] == "Bearer" {
					token = parts[1]
				} else if !strings.Contains(wsProtocol, ",") {
					// Single value - might be just the token
					token = wsProtocol
				}
			}
		}

		// 3. Fallback to query param (required for browser WebSocket API)
		// Note: WebSocket API in browsers doesn't support custom headers during handshake
		// so query params are the standard approach for browser-based WebSocket auth
		if token == "" {
			token = r.URL.Query().Get("token")
		}

		if token == "" {
			log.Printf("SECURITY: WebSocket connection without token from IP=%s fingerprint=%s", clientIP, requestFingerprint)
			wsTracker.recordConnectionAttempt(clientIP, false)
			http.Error(w, "Authorization required", http.StatusUnauthorized)
			return
		}

		// Validate token
		claims, err := authService.ValidateToken(token)
		if err != nil {
			log.Printf("SECURITY: Invalid WebSocket token from IP=%s fingerprint=%s error=%v", clientIP, requestFingerprint, err)
			wsTracker.recordConnectionAttempt(clientIP, false)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// SECURITY: Log successful WebSocket authentication
		log.Printf("SECURITY: WebSocket authenticated user=%s device=%s IP=%s fingerprint=%s",
			claims.UserID, claims.DeviceID, clientIP, requestFingerprint)

		// Record successful connection
		wsTracker.recordConnectionAttempt(clientIP, true)

		// Upgrade connection with response headers
		// If using Sec-WebSocket-Protocol, respond with the accepted protocol
		var responseHeader http.Header
		if r.Header.Get("Sec-WebSocket-Protocol") != "" {
			responseHeader = http.Header{
				"Sec-WebSocket-Protocol": []string{"Bearer"},
			}
		}
		conn, err := upgrader.Upgrade(w, r, responseHeader)
		if err != nil {
			log.Printf("SECURITY: WebSocket upgrade failed for user=%s IP=%s error=%v", claims.UserID, clientIP, err)
			return
		}

		// Create client with connection metadata for security tracking
		client := websocket.NewClient(hub, conn, claims.UserID, claims.DeviceID, token)

		// Register with hub
		hub.Register(client)

		// Start read/write pumps
		go client.WritePump()
		go client.ReadPump()
	}
}

// CSPReportHandler handles Content Security Policy violation reports
func CSPReportHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		security.CSPViolationHandler(w, r)
	}
}

// GetTurnCredentials generates time-limited TURN credentials for WebRTC
// Uses HMAC-SHA1 as per coturn's use-auth-secret mechanism
func GetTurnCredentials() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get TURN secret from environment
		turnSecret := os.Getenv("TURN_SECRET")
		if turnSecret == "" {
			log.Printf("[TURN] TURN_SECRET not set, returning STUN only")
			// Return STUN-only configuration
			writeJSON(w, map[string]interface{}{
				"iceServers": []map[string]interface{}{
					{"urls": "stun:stun.l.google.com:19302"},
					{"urls": "stun:stun1.l.google.com:19302"},
				},
			})
			return
		}

		// Get TURN server URL from environment (defaults to localhost for development)
		turnURL := os.Getenv("TURN_SERVER_URL")
		if turnURL == "" {
			turnURL = "turn:localhost:3478"
		}

		// Get the user ID from context (set by AuthMiddleware)
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Generate time-limited credentials (valid for 24 hours)
		// Username format: timestamp:userID (coturn standard)
		ttl := int64(24 * 60 * 60) // 24 hours in seconds
		timestamp := time.Now().Unix() + ttl
		username := fmt.Sprintf("%d:%s", timestamp, userID.String())

		// Generate credential using HMAC-SHA1 (coturn standard)
		// Note: crypto/hmac, crypto/sha1, and encoding/base64 are imported at the top
		mac := hmac.New(sha1.New, []byte(turnSecret))
		mac.Write([]byte(username))
		credential := base64.StdEncoding.EncodeToString(mac.Sum(nil))

		// Build ICE servers array
		iceServers := []map[string]interface{}{
			{"urls": "stun:stun.l.google.com:19302"},
			{"urls": "stun:stun1.l.google.com:19302"},
			{
				"urls":       turnURL,
				"username":   username,
				"credential": credential,
			},
		}

		// Also add TURNS (TLS) if available
		turnsURL := os.Getenv("TURNS_SERVER_URL")
		if turnsURL != "" {
			iceServers = append(iceServers, map[string]interface{}{
				"urls":       turnsURL,
				"username":   username,
				"credential": credential,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]interface{}{
			"iceServers": iceServers,
			"ttl":        ttl,
		})
	}
}
