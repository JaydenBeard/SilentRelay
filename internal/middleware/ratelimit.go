package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jaydenbeard/messaging-app/internal/metrics"
	"github.com/redis/go-redis/v9"
)

// EnhancedRateLimiter implements sophisticated multi-tier rate limiting with DDoS protection
type EnhancedRateLimiter struct {
	// Redis client for distributed rate limiting
	redisClient *redis.Client
	ctx         context.Context

	// Abuse detection (still in-memory for performance)
	abuseDetector *AbuseDetector

	// Configuration
	config *RateLimitConfig

	// Logging
	logger *log.Logger
}

// TieredLimit represents rate limits at different tiers
type TieredLimit struct {
	NormalLimit *LimitConfig
	StrictLimit *LimitConfig
	CurrentMode string // "normal" or "strict"
	LastUpdated time.Time
}

// LimitConfig defines rate limit parameters
type LimitConfig struct {
	MaxRequests int
	Window      time.Duration
	Requests    []time.Time
}

// RateLimitConfig holds all rate limiting configuration
type RateLimitConfig struct {
	IPLimits       map[string]*TieredLimitConfig
	UserLimits     map[string]*TieredLimitConfig
	EndpointLimits map[string]*TieredLimitConfig
	GlobalLimits   *TieredLimitConfig
	AbuseDetection *AbuseDetectionConfig
}

// TieredLimitConfig defines tiered limit configuration
type TieredLimitConfig struct {
	Normal *LimitConfig
	Strict *LimitConfig
}

// AbuseDetectionConfig defines abuse detection parameters
type AbuseDetectionConfig struct {
	Threshold          int
	Window             time.Duration
	PenaltyDuration    time.Duration
	StrictModeDuration time.Duration
}

// AbuseDetector implements abuse detection algorithms
type AbuseDetector struct {
	ipAttempts    map[string][]time.Time
	userAttempts  map[string][]time.Time
	penaltyBox    map[string]time.Time // IP/User -> penalty end time
	strictModeEnd map[string]time.Time // IP/User -> strict mode end time
	mu            sync.RWMutex
	config        *AbuseDetectionConfig
}

// NewEnhancedRateLimiter creates a new enhanced rate limiter with Redis support
func NewEnhancedRateLimiter(config *RateLimitConfig, redisClient *redis.Client) *EnhancedRateLimiter {
	rl := &EnhancedRateLimiter{
		redisClient:   redisClient,
		ctx:           context.Background(),
		abuseDetector: NewAbuseDetector(config.AbuseDetection),
		config:        config,
		logger:        log.New(log.Writer(), "[RATE-LIMIT] ", log.Ldate|log.Ltime|log.LUTC),
	}

	// Initialize cleanup goroutines (only for abuse detector now)
	go rl.abuseDetector.cleanup()

	return rl
}

// NewAbuseDetector creates a new abuse detector
func NewAbuseDetector(config *AbuseDetectionConfig) *AbuseDetector {
	if config == nil {
		config = &AbuseDetectionConfig{
			Threshold:          100,
			Window:             5 * time.Minute,
			PenaltyDuration:    15 * time.Minute,
			StrictModeDuration: 30 * time.Minute,
		}
	}

	return &AbuseDetector{
		ipAttempts:    make(map[string][]time.Time),
		userAttempts:  make(map[string][]time.Time),
		penaltyBox:    make(map[string]time.Time),
		strictModeEnd: make(map[string]time.Time),
		config:        config,
	}
}

// cleanup is not needed with Redis-based rate limiting - Redis handles TTL expiration

// cleanup removes old abuse detection data
func (ad *AbuseDetector) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ad.mu.Lock()

		now := time.Now()

		// Cleanup IP attempts
		for ip, times := range ad.ipAttempts {
			ad.ipAttempts[ip] = ad.filterOldAttempts(times, ad.config.Window, now)
			if len(ad.ipAttempts[ip]) == 0 {
				delete(ad.ipAttempts, ip)
			}
		}

		// Cleanup user attempts
		for user, times := range ad.userAttempts {
			ad.userAttempts[user] = ad.filterOldAttempts(times, ad.config.Window, now)
			if len(ad.userAttempts[user]) == 0 {
				delete(ad.userAttempts, user)
			}
		}

		// Cleanup penalty box
		for key, endTime := range ad.penaltyBox {
			if now.After(endTime) {
				delete(ad.penaltyBox, key)
			}
		}

		// Cleanup strict mode
		for key, endTime := range ad.strictModeEnd {
			if now.After(endTime) {
				delete(ad.strictModeEnd, key)
			}
		}

		ad.mu.Unlock()
	}
}

// filterOldAttempts removes attempts outside the time window
func (ad *AbuseDetector) filterOldAttempts(times []time.Time, window time.Duration, now time.Time) []time.Time {
	filtered := []time.Time{}
	for _, t := range times {
		if now.Sub(t) < window {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// Middleware returns an HTTP middleware that enforces enhanced rate limiting
func (rl *EnhancedRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting for WebSocket upgrade requests
		if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") ||
			strings.HasPrefix(r.URL.Path, "/ws") {
			next.ServeHTTP(w, r)
			return
		}

		// Extract identifiers
		ip := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ip = forwarded
		} else if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
			ip = realIP
		}

		userID := ""
		if user := r.Context().Value("userID"); user != nil {
			userID = user.(string)
		}

		endpoint := r.Method + " " + r.URL.Path

		// Check if in penalty box
		if rl.abuseDetector.IsInPenaltyBox(ip) || (userID != "" && rl.abuseDetector.IsInPenaltyBox(userID)) {
			metrics.RecordRateLimitHit(endpoint, "penalty")
			metrics.RecordRateLimitRequest(endpoint, "penalty", "denied")
			rl.logger.Printf("RATE LIMIT DENIED - %s is in penalty box (IP: %s, User: %s)", endpoint, ip, userID)
			http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			return
		}

		// Check global limits first
		if !rl.allowGlobalRequest() {
			metrics.RecordRateLimitHit(endpoint, "global")
			metrics.RecordRateLimitRequest(endpoint, "global", "denied")
			rl.logger.Printf("RATE LIMIT DENIED - global limit reached (IP: %s, User: %s, Endpoint: %s)", ip, userID, endpoint)
			http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			return
		}

		// Check endpoint limits
		if !rl.allowEndpointRequest(endpoint) {
			metrics.RecordRateLimitHit(endpoint, "endpoint")
			metrics.RecordRateLimitRequest(endpoint, "endpoint", "denied")
			rl.logger.Printf("RATE LIMIT DENIED - endpoint limit reached (IP: %s, User: %s, Endpoint: %s)", ip, userID, endpoint)
			http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			return
		}

		// Check IP limits
		if !rl.allowIPRequest(ip) {
			metrics.RecordRateLimitHit(endpoint, "ip")
			metrics.RecordRateLimitRequest(endpoint, "ip", "denied")
			rl.logger.Printf("RATE LIMIT DENIED - IP limit reached (IP: %s, User: %s, Endpoint: %s)", ip, userID, endpoint)
			http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			return
		}

		// Check user limits if authenticated
		if userID != "" && !rl.allowUserRequest(userID) {
			metrics.RecordRateLimitHit(endpoint, "user")
			metrics.RecordRateLimitRequest(endpoint, "user", "denied")
			rl.logger.Printf("RATE LIMIT DENIED - user limit reached (IP: %s, User: %s, Endpoint: %s)", ip, userID, endpoint)
			http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			return
		}

		// Record successful request
		metrics.RecordRateLimitRequest(endpoint, "allowed", "allowed")
		rl.logger.Printf("RATE LIMIT ALLOWED - request permitted (IP: %s, User: %s, Endpoint: %s)", ip, userID, endpoint)

		// Record abuse detection attempt
		rl.abuseDetector.recordAttempt(ip, userID)

		next.ServeHTTP(w, r)
	})
}

// allowGlobalRequest checks if a request should be allowed at global level
func (rl *EnhancedRateLimiter) allowGlobalRequest() bool {
	key := "ratelimit:global"
	maxRequests := 1000 // Default normal mode limit
	window := time.Minute

	// Check if in strict mode (ignore error - defaults to normal mode)
	strictMode, err := rl.redisClient.Get(rl.ctx, "ratelimit:global:mode").Result()
	if err != nil && err != redis.Nil {
		rl.logger.Printf("Warning: Failed to get global mode: %v", err)
	}
	if strictMode == "strict" {
		maxRequests = 500
	}

	// Use Redis sorted set to track requests with timestamps
	now := time.Now().Unix()
	windowStart := now - int64(window.Seconds())

	// Remove old requests (ignore error - non-critical cleanup)
	if err := rl.redisClient.ZRemRangeByScore(rl.ctx, key, "-inf", fmt.Sprintf("(%d", windowStart)).Err(); err != nil {
		rl.logger.Printf("Warning: Failed to remove old requests: %v", err)
	}

	// Count current requests in window
	count, err := rl.redisClient.ZCard(rl.ctx, key).Result()
	if err != nil && err != redis.Nil {
		rl.logger.Printf("Warning: Failed to count requests: %v", err)
		// On error, allow the request rather than blocking
		return true
	}

	// Check if limit exceeded
	if count >= int64(maxRequests) {
		return false
	}

	// Add current request (ignore error - best effort)
	if err := rl.redisClient.ZAdd(rl.ctx, key, redis.Z{Score: float64(now), Member: fmt.Sprintf("%d", now)}).Err(); err != nil {
		rl.logger.Printf("Warning: Failed to add request: %v", err)
	}
	if err := rl.redisClient.Expire(rl.ctx, key, window).Err(); err != nil {
		rl.logger.Printf("Warning: Failed to set expiry: %v", err)
	}

	return true
}

// allowEndpointRequest checks if a request should be allowed at endpoint level
func (rl *EnhancedRateLimiter) allowEndpointRequest(endpoint string) bool {
	key := fmt.Sprintf("ratelimit:endpoint:%s", endpoint)
	maxRequests := 100 // Default normal mode limit
	window := time.Minute

	// Check if in strict mode for this endpoint
	strictMode, err := rl.redisClient.Get(rl.ctx, fmt.Sprintf("ratelimit:endpoint:%s:mode", endpoint)).Result()
	if err != nil && err != redis.Nil {
		rl.logger.Printf("Warning: Failed to get endpoint mode: %v", err)
	}
	if strictMode == "strict" {
		maxRequests = 50
	}

	now := time.Now().Unix()
	windowStart := now - int64(window.Seconds())

	// Remove old requests
	if err := rl.redisClient.ZRemRangeByScore(rl.ctx, key, "-inf", fmt.Sprintf("(%d", windowStart)).Err(); err != nil {
		rl.logger.Printf("Warning: Failed to remove old requests: %v", err)
	}

	// Count current requests in window
	count, err := rl.redisClient.ZCard(rl.ctx, key).Result()
	if err != nil && err != redis.Nil {
		rl.logger.Printf("Warning: Failed to count requests: %v", err)
		return true
	}

	// Check if limit exceeded
	if count >= int64(maxRequests) {
		return false
	}

	// Add current request
	if err := rl.redisClient.ZAdd(rl.ctx, key, redis.Z{Score: float64(now), Member: fmt.Sprintf("%d", now)}).Err(); err != nil {
		rl.logger.Printf("Warning: Failed to add request: %v", err)
	}
	if err := rl.redisClient.Expire(rl.ctx, key, window).Err(); err != nil {
		rl.logger.Printf("Warning: Failed to set expiry: %v", err)
	}

	return true
}

// allowIPRequest checks if a request should be allowed at IP level
func (rl *EnhancedRateLimiter) allowIPRequest(ip string) bool {
	key := fmt.Sprintf("ratelimit:ip:%s", ip)
	maxRequests := 60 // Default normal mode limit
	window := time.Minute

	// Check if in strict mode for this IP
	strictMode, err := rl.redisClient.Get(rl.ctx, fmt.Sprintf("ratelimit:ip:%s:mode", ip)).Result()
	if err != nil && err != redis.Nil {
		rl.logger.Printf("Warning: Failed to get IP mode: %v", err)
	}
	if strictMode == "strict" {
		maxRequests = 30
	}

	now := time.Now().Unix()
	windowStart := now - int64(window.Seconds())

	// Remove old requests
	if err := rl.redisClient.ZRemRangeByScore(rl.ctx, key, "-inf", fmt.Sprintf("(%d", windowStart)).Err(); err != nil {
		rl.logger.Printf("Warning: Failed to remove old requests: %v", err)
	}

	// Count current requests in window
	count, err := rl.redisClient.ZCard(rl.ctx, key).Result()
	if err != nil && err != redis.Nil {
		rl.logger.Printf("Warning: Failed to count requests: %v", err)
		return true
	}

	// Check if limit exceeded
	if count >= int64(maxRequests) {
		return false
	}

	// Add current request
	if err := rl.redisClient.ZAdd(rl.ctx, key, redis.Z{Score: float64(now), Member: fmt.Sprintf("%d", now)}).Err(); err != nil {
		rl.logger.Printf("Warning: Failed to add request: %v", err)
	}
	if err := rl.redisClient.Expire(rl.ctx, key, window).Err(); err != nil {
		rl.logger.Printf("Warning: Failed to set expiry: %v", err)
	}

	return true
}

// allowUserRequest checks if a request should be allowed at user level
func (rl *EnhancedRateLimiter) allowUserRequest(userID string) bool {
	key := fmt.Sprintf("ratelimit:user:%s", userID)
	maxRequests := 120 // Default normal mode limit
	window := time.Minute

	// Check if in strict mode for this user
	strictMode, err := rl.redisClient.Get(rl.ctx, fmt.Sprintf("ratelimit:user:%s:mode", userID)).Result()
	if err != nil && err != redis.Nil {
		rl.logger.Printf("Warning: Failed to get user mode: %v", err)
	}
	if strictMode == "strict" {
		maxRequests = 60
	}

	now := time.Now().Unix()
	windowStart := now - int64(window.Seconds())

	// Remove old requests
	if err := rl.redisClient.ZRemRangeByScore(rl.ctx, key, "-inf", fmt.Sprintf("(%d", windowStart)).Err(); err != nil {
		rl.logger.Printf("Warning: Failed to remove old requests: %v", err)
	}

	// Count current requests in window
	count, err := rl.redisClient.ZCard(rl.ctx, key).Result()
	if err != nil && err != redis.Nil {
		rl.logger.Printf("Warning: Failed to count requests: %v", err)
		return true
	}

	// Check if limit exceeded
	if count >= int64(maxRequests) {
		return false
	}

	// Add current request
	if err := rl.redisClient.ZAdd(rl.ctx, key, redis.Z{Score: float64(now), Member: fmt.Sprintf("%d", now)}).Err(); err != nil {
		rl.logger.Printf("Warning: Failed to add request: %v", err)
	}
	if err := rl.redisClient.Expire(rl.ctx, key, window).Err(); err != nil {
		rl.logger.Printf("Warning: Failed to set expiry: %v", err)
	}

	return true
}

// recordAttempt records an attempt for abuse detection
func (ad *AbuseDetector) recordAttempt(ip string, userID string) {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	now := time.Now()

	// Record IP attempt
	if attempts, exists := ad.ipAttempts[ip]; exists {
		ad.ipAttempts[ip] = append(attempts, now)
	} else {
		ad.ipAttempts[ip] = []time.Time{now}
	}

	// Record user attempt if available
	if userID != "" {
		if attempts, exists := ad.userAttempts[userID]; exists {
			ad.userAttempts[userID] = append(attempts, now)
		} else {
			ad.userAttempts[userID] = []time.Time{now}
		}
	}

	// Check for abuse patterns
	ad.checkForAbuse(ip, userID)
}

// checkForAbuse checks if IP or user is exhibiting abusive behavior
func (ad *AbuseDetector) checkForAbuse(ip string, userID string) {
	now := time.Now()

	// Check IP abuse
	if attempts, exists := ad.ipAttempts[ip]; exists {
		recentAttempts := ad.filterOldAttempts(attempts, ad.config.Window, now)
		if len(recentAttempts) >= ad.config.Threshold {
			ad.penaltyBox[ip] = now.Add(ad.config.PenaltyDuration)
			ad.strictModeEnd[ip] = now.Add(ad.config.StrictModeDuration)
			metrics.RecordAbuseDetectionEvent("ip", "penalty")
			metrics.RecordStrictModeActivation("ip")
			log.Printf("ABUSE DETECTED: IP %s placed in penalty box for %v", ip, ad.config.PenaltyDuration)
		}
	}

	// Check user abuse if available
	if userID != "" {
		if attempts, exists := ad.userAttempts[userID]; exists {
			recentAttempts := ad.filterOldAttempts(attempts, ad.config.Window, now)
			if len(recentAttempts) >= ad.config.Threshold {
				ad.penaltyBox[userID] = now.Add(ad.config.PenaltyDuration)
				ad.strictModeEnd[userID] = now.Add(ad.config.StrictModeDuration)
				metrics.RecordAbuseDetectionEvent("user", "penalty")
				metrics.RecordStrictModeActivation("user")
				log.Printf("ABUSE DETECTED: User %s placed in penalty box for %v", userID, ad.config.PenaltyDuration)
			}
		}
	}
}

// IsInPenaltyBox checks if IP or user is in penalty box
func (ad *AbuseDetector) IsInPenaltyBox(key string) bool {
	ad.mu.RLock()
	defer ad.mu.RUnlock()

	if endTime, exists := ad.penaltyBox[key]; exists {
		return time.Now().Before(endTime)
	}
	return false
}

// RecordAttempt records an attempt for abuse detection (public for testing)
func (ad *AbuseDetector) RecordAttempt(ip string, userID string) {
	ad.recordAttempt(ip, userID)
}

// SetGlobalStrictMode enables strict mode globally
func (rl *EnhancedRateLimiter) SetGlobalStrictMode(enable bool) {
	mode := "normal"
	if enable {
		mode = "strict"
	}
	rl.redisClient.Set(rl.ctx, "ratelimit:global:mode", mode, 0)
	rl.logger.Printf("Global strict mode %s", strings.ToUpper(mode))
}

// SetEndpointStrictMode enables strict mode for specific endpoint
func (rl *EnhancedRateLimiter) SetEndpointStrictMode(endpoint string, enable bool) {
	mode := "normal"
	if enable {
		mode = "strict"
	}
	key := fmt.Sprintf("ratelimit:endpoint:%s:mode", endpoint)
	rl.redisClient.Set(rl.ctx, key, mode, 0)
	rl.logger.Printf("Strict mode %s for endpoint: %s", strings.ToUpper(mode), endpoint)
}

// GetRateLimitStatus returns current rate limit status
func (rl *EnhancedRateLimiter) GetRateLimitStatus() map[string]interface{} {
	// Get global mode
	globalMode, err := rl.redisClient.Get(rl.ctx, "ratelimit:global:mode").Result()
	if err != nil && err != redis.Nil {
		rl.logger.Printf("Warning: Failed to get global mode: %v", err)
	}
	if globalMode == "" {
		globalMode = "normal"
	}

	// Get global request count
	globalCount, err := rl.redisClient.ZCard(rl.ctx, "ratelimit:global").Result()
	if err != nil && err != redis.Nil {
		rl.logger.Printf("Warning: Failed to get global count: %v", err)
	}

	// Get approximate counts (Redis doesn't have efficient count operations for patterns)
	// In production, you might want to maintain separate counters
	ipKeys, err := rl.redisClient.Keys(rl.ctx, "ratelimit:ip:*").Result()
	if err != nil && err != redis.Nil {
		rl.logger.Printf("Warning: Failed to get IP keys: %v", err)
	}
	userKeys, err := rl.redisClient.Keys(rl.ctx, "ratelimit:user:*").Result()
	if err != nil && err != redis.Nil {
		rl.logger.Printf("Warning: Failed to get user keys: %v", err)
	}
	endpointKeys, err := rl.redisClient.Keys(rl.ctx, "ratelimit:endpoint:*").Result()
	if err != nil && err != redis.Nil {
		rl.logger.Printf("Warning: Failed to get endpoint keys: %v", err)
	}

	status := map[string]interface{}{
		"global_mode":     globalMode,
		"global_requests": globalCount,
		"ip_counts":       len(ipKeys),
		"user_counts":     len(userKeys),
		"endpoint_counts": len(endpointKeys),
	}

	return status
}
