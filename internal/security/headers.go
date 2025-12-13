package security

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SecurityHeadersMiddleware adds essential security headers
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// XSS protection (legacy browsers)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Referrer policy - don't leak URLs
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions policy - disable dangerous features
		w.Header().Set("Permissions-Policy",
			"accelerometer=(), camera=(), geolocation=(), gyroscope=(), "+
				"magnetometer=(), microphone=(), payment=(), usb=()")

		// Content Security Policy - modern secure implementation with enhanced security
		csp := []string{
			"default-src 'self'",
			"script-src 'self' 'nonce-{nonce-value}' 'strict-dynamic' 'unsafe-eval'",
			"style-src 'self' 'nonce-{nonce-value}' https://fonts.googleapis.com https://fonts.gstatic.com",
			"font-src 'self' data: https://fonts.gstatic.com",
			"img-src 'self' data: blob: https:",
			"connect-src 'self' wss: https:",
			"frame-src 'none'",
			"object-src 'none'",
			"base-uri 'self'",
			"form-action 'self'",
			"worker-src 'none'",
			"manifest-src 'self'",
			"prefetch-src 'self'",
			"media-src 'self'",
			"sandbox allow-same-origin allow-scripts allow-forms",
			"require-trusted-types-for 'script'",
			"upgrade-insecure-requests",
			"block-all-mixed-content",
			"report-uri /csp-report",
			"report-to csp-endpoint",
		}
		w.Header().Set("Content-Security-Policy", strings.Join(csp, "; "))

		// HSTS - force HTTPS (1 year, include subdomains, preload)
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

		// Report-To header for CSP violation reporting
		w.Header().Set("Report-To", "{\"group\":\"csp-endpoint\",\"max_age\":10886400,\"endpoints\":[{\"url\":\"/csp-report\"}],\"include_subdomains\":true}")

		// Prevent caching of sensitive data
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
		}

		// Remove server identification
		w.Header().Del("Server")
		w.Header().Del("X-Powered-By")

		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware handles CORS with strict origin validation
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	originSet := make(map[string]bool)
	for _, origin := range allowedOrigins {
		originSet[origin] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Only allow configured origins
			if origin != "" && originSet[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID, X-Device-ID")
				w.Header().Set("Access-Control-Max-Age", "86400")
				w.Header().Set("Vary", "Origin")
			}

			// Handle preflight
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequestIDMiddleware adds a unique request ID for tracing
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID, _ = SecureRandomHex(16)
		}

		w.Header().Set("X-Request-ID", requestID)
		r.Header.Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r)
	})
}

// MaxBodySizeMiddleware limits request body size
func MaxBodySizeMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// WAFMiddleware provides basic Web Application Firewall functionality
func WAFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block suspicious user agents
		ua := strings.ToLower(r.UserAgent())
		blockedAgents := []string{
			"sqlmap", "nikto", "nessus", "nmap", "masscan",
			"burp", "dirbuster", "gobuster", "wfuzz", "ffuf",
		}
		for _, blocked := range blockedAgents {
			if strings.Contains(ua, blocked) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		}

		// Block requests with suspicious paths
		path := strings.ToLower(r.URL.Path)
		blockedPaths := []string{
			".git", ".env", ".svn", ".htaccess",
			"wp-admin", "phpmyadmin", "adminer",
			"..%2f", "%2e%2e", "etc/passwd",
		}
		for _, blocked := range blockedPaths {
			if strings.Contains(path, blocked) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		}

		// Check query params for injection
		for key, values := range r.URL.Query() {
			if SanitizeString(key) == "" {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			for _, v := range values {
				if SanitizeString(v) == "" {
					http.Error(w, "Bad Request", http.StatusBadRequest)
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// RateLimiter provides per-IP rate limiting
type RateLimiter struct {
	requests map[string][]time.Time
	mu       sync.RWMutex
	limit    int
	window   time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
	go rl.cleanup()
	return rl
}

// RateLimitMiddleware creates rate limiting middleware
func RateLimitMiddleware(limit int, window time.Duration) func(http.Handler) http.Handler {
	limiter := NewRateLimiter(limit, window)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := GetRealIP(r)

			if !limiter.Allow(ip) {
				w.Header().Set("Retry-After", strconv.Itoa(int(window.Seconds())))
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Allow checks if a request is allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Filter old requests
	filtered := make([]time.Time, 0)
	for _, t := range rl.requests[ip] {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}

	if len(filtered) >= rl.limit {
		rl.requests[ip] = filtered
		return false
	}

	rl.requests[ip] = append(filtered, now)
	return true
}

// cleanup removes expired entries
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-rl.window)
		for ip, times := range rl.requests {
			filtered := make([]time.Time, 0)
			for _, t := range times {
				if t.After(cutoff) {
					filtered = append(filtered, t)
				}
			}
			if len(filtered) == 0 {
				delete(rl.requests, ip)
			} else {
				rl.requests[ip] = filtered
			}
		}
		rl.mu.Unlock()
	}
}
