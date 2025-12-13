package security

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// CSPViolation represents a Content Security Policy violation report
type CSPViolation struct {
	DocumentURI        string `json:"document-uri"`
	Referrer           string `json:"referrer"`
	ViolatedDirective  string `json:"violated-directive"`
	EffectiveDirective string `json:"effective-directive"`
	OriginalPolicy     string `json:"original-policy"`
	BlockedURI         string `json:"blocked-uri"`
	StatusCode         int    `json:"status-code"`
	LineNumber         int    `json:"line-number"`
	ColumnNumber       int    `json:"column-number"`
	SourceFile         string `json:"source-file"`
	ScriptSample       string `json:"script-sample"`
}

// CSPReport represents the overall CSP violation report
type CSPReport struct {
	CSPReport CSPViolation `json:"csp-report"`
}

// CSPEnforcementMiddleware enforces CSP compliance and validates requests
func CSPEnforcementMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate Content-Type for POST requests
		if r.Method == http.MethodPost {
			contentType := r.Header.Get("Content-Type")
			if contentType != "" && !strings.Contains(contentType, "application/json") &&
				!strings.Contains(contentType, "multipart/form-data") &&
				!strings.Contains(contentType, "application/x-www-form-urlencoded") {
				http.Error(w, "Invalid Content-Type", http.StatusUnsupportedMediaType)
				return
			}
		}

		// Validate User-Agent to block known malicious bots
		userAgent := strings.ToLower(r.UserAgent())
		blockedAgents := []string{
			"sqlmap", "nikto", "nessus", "nmap", "masscan",
			"burp", "dirbuster", "gobuster", "wfuzz", "ffuf",
			"curl", "wget", "python-requests", "java", "go-http-client",
		}
		for _, blocked := range blockedAgents {
			if strings.Contains(userAgent, blocked) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		}

		// Validate Referer header for sensitive endpoints
		if strings.HasPrefix(r.URL.Path, "/api/") {
			referer := r.Referer()
			if referer != "" && !strings.HasPrefix(referer, "https://") {
				http.Error(w, "Invalid Referer", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// CSPViolationHandler handles CSP violation reports
func CSPViolationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the request body with size limit
	body, err := io.ReadAll(io.LimitReader(r.Body, 10*1024)) // 10KB max
	if err != nil {
		log.Printf("ERROR: Failed to read CSP report: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Parse the CSP violation report
	var report CSPReport
	err = json.Unmarshal(body, &report)
	if err != nil {
		log.Printf("ERROR: Invalid CSP report format: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Log the violation
	logCSPViolation(report.CSPReport)

	// Return 204 No Content as per CSP spec
	w.WriteHeader(http.StatusNoContent)
}

// logCSPViolation logs CSP violations to file and console
func logCSPViolation(violation CSPViolation) {
	// Create log entry
	logEntry := struct {
		Timestamp    string `json:"timestamp"`
		CSPViolation `json:"violation"`
	}{
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		CSPViolation: violation,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(logEntry)
	if err != nil {
		log.Printf("Error marshaling CSP violation: %v", err)
		return
	}

	// Log to console
	log.Printf("CSP Violation: %s", string(jsonData))

	// Log to file
	file, err := os.OpenFile("csp_violations.log",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Error opening CSP violation log file: %v", err)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Warning: failed to close file: %v", err)
		}
	}()

	if _, err := file.Write(append(jsonData, '\n')); err != nil {
		log.Printf("Error writing to CSP violation log file: %v", err)
	}
}

// CSPMiddleware adds CSP headers with nonce support
func CSPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate a unique nonce for this request
		nonce := GenerateSecureNonce(16)

		// Set modern CSP headers with enhanced security
		csp := []string{
			"default-src 'self'",
			"script-src 'self' 'nonce-" + nonce + "' 'strict-dynamic' 'unsafe-eval'",
			"style-src 'self' 'nonce-" + nonce + "' https://fonts.googleapis.com https://fonts.gstatic.com",
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

		// Add CSP bypass mitigation headers
		w.Header().Set("X-Content-Security-Policy", strings.Join(csp, "; ")) // Legacy header
		w.Header().Set("X-WebKit-CSP", strings.Join(csp, "; "))              // WebKit header

		// Set main CSP header
		w.Header().Set("Content-Security-Policy", strings.Join(csp, "; "))

		// Set Report-To header
		w.Header().Set("Report-To", "{\"group\":\"csp-endpoint\",\"max_age\":10886400,\"endpoints\":[{\"url\":\"/csp-report\"}],\"include_subdomains\":true}")

		// Add additional security headers to prevent bypass
		w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")
		w.Header().Set("X-Download-Options", "noopen")
		w.Header().Set("X-DNS-Prefetch-Control", "off")

		next.ServeHTTP(w, r)
	})
}

// CSPBypassMitigation adds additional protection against CSP bypass techniques
func CSPBypassMitigation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mitigate CSP bypass via URL fragments
		if strings.Contains(r.URL.RawQuery, "#") || strings.Contains(r.URL.Fragment, "javascript:") {
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return
		}

		// Mitigate CSP bypass via data URIs in query params
		for _, values := range r.URL.Query() {
			for _, value := range values {
				if strings.Contains(strings.ToLower(value), "data:") ||
					strings.Contains(strings.ToLower(value), "javascript:") ||
					strings.Contains(strings.ToLower(value), "vbscript:") {
					http.Error(w, "Potential CSP bypass attempt detected", http.StatusForbidden)
					return
				}
			}
		}

		// Mitigate CSP bypass via malicious headers
		maliciousHeaders := []string{
			"x-forwarded-host", "x-host", "x-original-url",
			"x-rewrite-url", "x-override-url", "x-http-method-override",
		}
		for _, header := range maliciousHeaders {
			if r.Header.Get(header) != "" {
				http.Error(w, "Forbidden header detected", http.StatusForbidden)
				return
			}
		}

		// Mitigate CSP bypass via HTTP method override
		if strings.ToLower(r.Header.Get("X-HTTP-Method-Override")) != "" {
			http.Error(w, "Method override not allowed", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GenerateSecureNonce generates a cryptographically secure nonce
func GenerateSecureNonce(length int) string {
	// Generate cryptographically secure random bytes
	randomBytes := make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		// Fallback to timestamp-based nonce if crypto fails
		return "nonce-" + time.Now().UTC().Format("20060102150405.999999999")
	}
	// Return base64 encoded nonce
	return "nonce-" + base64.URLEncoding.EncodeToString(randomBytes)
}
