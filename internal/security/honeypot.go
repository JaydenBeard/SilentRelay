package security

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ============================================
// HONEYPOTS & DECEPTION TECHNOLOGY
// Detect attackers by luring them to fake resources
// ============================================

// HoneypotType defines types of honeypot traps
type HoneypotType string

const (
	HoneypotTypeEndpoint   HoneypotType = "endpoint"   // Fake API endpoints
	HoneypotTypeHeader     HoneypotType = "header"     // Fake headers that look exploitable
	HoneypotTypeCredential HoneypotType = "credential" // Fake leaked credentials
	HoneypotTypeFile       HoneypotType = "file"       // Fake sensitive files
	HoneypotTypeDatabase   HoneypotType = "database"   // Fake database entries
)

// HoneypotHit records when an attacker triggers a honeypot
type HoneypotHit struct {
	ID        uuid.UUID              `json:"id"`
	Type      HoneypotType           `json:"type"`
	Name      string                 `json:"name"`
	SourceIP  string                 `json:"source_ip"`
	UserAgent string                 `json:"user_agent"`
	Method    string                 `json:"method"`
	Path      string                 `json:"path"`
	Headers   map[string]string      `json:"headers"`
	Body      string                 `json:"body,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

// HoneypotManager manages all honeypot traps
type HoneypotManager struct {
	db        *sql.DB
	ids       *IntrusionDetector
	endpoints map[string]string       // path -> honeypot name
	headers   map[string]string       // header -> honeypot name
	canaries  map[string]*CanaryToken // token -> canary
}

// NewHoneypotManager creates a new honeypot manager
func NewHoneypotManager(db *sql.DB, ids *IntrusionDetector) *HoneypotManager {
	hm := &HoneypotManager{
		db:        db,
		ids:       ids,
		endpoints: make(map[string]string),
		headers:   make(map[string]string),
		canaries:  make(map[string]*CanaryToken),
	}

	// Register default honeypot endpoints
	// These look like real targets but are traps
	hm.RegisterEndpoint("/admin", "admin_panel")
	hm.RegisterEndpoint("/administrator", "admin_panel_alt")
	hm.RegisterEndpoint("/wp-admin", "wordpress_admin")
	hm.RegisterEndpoint("/phpmyadmin", "phpmyadmin")
	hm.RegisterEndpoint("/backup", "backup_dir")
	hm.RegisterEndpoint("/backup.sql", "backup_file")
	hm.RegisterEndpoint("/dump.sql", "db_dump")
	hm.RegisterEndpoint("/.git/config", "git_config")
	hm.RegisterEndpoint("/.env", "env_file")
	hm.RegisterEndpoint("/api/v1/internal/debug", "debug_endpoint")
	hm.RegisterEndpoint("/api/v1/admin/users/export", "user_export")
	hm.RegisterEndpoint("/api/v1/admin/keys/master", "master_key")
	hm.RegisterEndpoint("/actuator/env", "spring_actuator")
	hm.RegisterEndpoint("/graphql", "graphql_endpoint")
	hm.RegisterEndpoint("/.aws/credentials", "aws_creds")
	hm.RegisterEndpoint("/config/database.yml", "db_config")

	return hm
}

// RegisterEndpoint registers a honeypot endpoint
func (hm *HoneypotManager) RegisterEndpoint(path, name string) {
	hm.endpoints[path] = name
}

// HoneypotMiddleware intercepts requests to honeypot endpoints
func (hm *HoneypotManager) HoneypotMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.ToLower(r.URL.Path)

		// Check if this is a honeypot endpoint
		if name, isHoneypot := hm.endpoints[path]; isHoneypot {
			hm.recordHit(&HoneypotHit{
				ID:        uuid.New(),
				Type:      HoneypotTypeEndpoint,
				Name:      name,
				SourceIP:  GetRealIP(r),
				UserAgent: r.UserAgent(),
				Method:    r.Method,
				Path:      r.URL.Path,
				Headers:   extractHeaders(r),
				Timestamp: time.Now(),
			})

			// Return a believable but useless response
			hm.serveHoneypotResponse(w, r, name)
			return
		}

		// Check for suspicious header access
		hm.checkSuspiciousHeaders(r)

		next.ServeHTTP(w, r)
	})
}

// recordHit records a honeypot trigger
func (hm *HoneypotManager) recordHit(hit *HoneypotHit) {
	// Log to database
	ctx := context.Background()
	headersJSON, err := json.Marshal(hit.Headers)
	if err != nil {
		log.Printf("Failed to marshal headers for honeypot hit: %v", err)
		return
	}
	extraJSON, err := json.Marshal(hit.Extra)
	if err != nil {
		log.Printf("Failed to marshal extra data for honeypot hit: %v", err)
		return
	}

	if _, err := hm.db.ExecContext(ctx, `
		INSERT INTO honeypot_hits (id, type, name, source_ip, user_agent, method, path, headers, body, extra, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, hit.ID, hit.Type, hit.Name, hit.SourceIP, hit.UserAgent, hit.Method, hit.Path,
		headersJSON, hit.Body, extraJSON, hit.Timestamp); err != nil {
		log.Printf("Warning: failed to insert honeypot hit: %v", err)
	}

	// Alert IDS
	if hm.ids != nil {
		hm.ids.RecordSuspiciousPayload(hit.SourceIP, hit.Path, "honeypot_triggered")
	}
}

// serveHoneypotResponse returns a believable fake response
func (hm *HoneypotManager) serveHoneypotResponse(w http.ResponseWriter, _ *http.Request, name string) {
	// Add delay to slow down automated scanners
	time.Sleep(500 * time.Millisecond)

	switch name {
	case "admin_panel", "admin_panel_alt":
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte(`<!DOCTYPE html><html><head><title>Admin Login</title></head>
			<body><form action="/admin/login" method="POST">
			<input name="username" placeholder="Username"/>
			<input name="password" type="password" placeholder="Password"/>
			<button>Login</button></form></body></html>`)); err != nil {
			log.Printf("Warning: failed to write honeypot response: %v", err)
		}

	case "env_file", "aws_creds", "db_config":
		// Return fake credentials that will trigger alerts when used
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`# FAKE CREDENTIALS - DO NOT USE
DB_PASSWORD=honeypot_trap_12345
AWS_ACCESS_KEY_ID=AKIAHONEYPOT123456789
AWS_SECRET_ACCESS_KEY=honeypot/trap/key/DoNotUseThisWillTriggerAlert
ADMIN_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.honeypot`)); err != nil {
			log.Printf("Warning: failed to write honeypot response: %v", err)
		}

	case "git_config":
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`[core]
	repositoryformatversion = 0
	filemode = true
[remote "origin"]
	url = https://honeypot:trap@github.com/fake/repo.git`)); err != nil {
			log.Printf("Warning: failed to write honeypot response: %v", err)
		}

	case "backup_file", "db_dump":
		w.Header().Set("Content-Type", "application/sql")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`-- Honeypot trap database dump
-- This file is monitored. Access has been logged.
CREATE TABLE users (id INT, username VARCHAR(255), password_hash VARCHAR(255));
INSERT INTO users VALUES (1, 'admin', 'honeypot_hash_do_not_use');`)); err != nil {
			log.Printf("Warning: failed to write honeypot response: %v", err)
		}

	case "graphql_endpoint":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"data":null,"errors":[{"message":"Honeypot introspection disabled"}]}`)); err != nil {
			log.Printf("Warning: failed to write honeypot response: %v", err)
		}

	default:
		w.WriteHeader(http.StatusForbidden)
		if _, err := w.Write([]byte("Access Denied")); err != nil {
			log.Printf("Warning: failed to write honeypot response: %v", err)
		}
	}
}

// checkSuspiciousHeaders looks for attempts to exploit via headers
func (hm *HoneypotManager) checkSuspiciousHeaders(r *http.Request) {
	suspiciousHeaders := []string{
		"X-Forwarded-For", // IP spoofing attempts
		"X-Originating-IP",
		"X-Remote-IP",
		"X-Debug", // Debug mode attempts
		"X-Debug-Token",
		"X-Custom-IP-Authorization",
		"Admin",
		"X-Requested-With", // CSRF bypass attempts
	}

	for _, header := range suspiciousHeaders {
		value := r.Header.Get(header)
		if value != "" && hm.isSuspiciousValue(value) {
			hm.recordHit(&HoneypotHit{
				ID:        uuid.New(),
				Type:      HoneypotTypeHeader,
				Name:      header,
				SourceIP:  GetRealIP(r),
				UserAgent: r.UserAgent(),
				Method:    r.Method,
				Path:      r.URL.Path,
				Headers:   map[string]string{header: value},
				Timestamp: time.Now(),
			})
		}
	}
}

func (hm *HoneypotManager) isSuspiciousValue(value string) bool {
	suspicious := []string{
		"127.0.0.1", "localhost", "0.0.0.0",
		"admin", "root", "debug", "true",
		"../", "..\\", "%00", "<script",
	}

	lower := strings.ToLower(value)
	for _, s := range suspicious {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}

func extractHeaders(r *http.Request) map[string]string {
	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	return headers
}

// ============================================
// CANARY TOKENS
// Unique tokens that trigger alerts when accessed
// ============================================

// CanaryToken is a unique tracking token
type CanaryToken struct {
	ID           uuid.UUID
	Token        string
	Type         string // url, email, file, db_record
	Description  string
	CreatedAt    time.Time
	TriggeredAt  *time.Time
	TriggerCount int
}

// GenerateCanaryToken creates a new canary token
func (hm *HoneypotManager) GenerateCanaryToken(tokenType, description string) *CanaryToken {
	token, err := SecureRandomHex(32)
	if err != nil {
		log.Printf("Warning: Failed to generate canary token: %v", err)
		token = uuid.New().String() // Fallback to UUID
	}
	canary := &CanaryToken{
		ID:          uuid.New(),
		Token:       token,
		Type:        tokenType,
		Description: description,
		CreatedAt:   time.Now(),
	}

	hm.canaries[token] = canary

	// Store in database
	ctx := context.Background()
	if _, err := hm.db.ExecContext(ctx, `
		INSERT INTO canary_tokens (id, token, type, description, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, canary.ID, canary.Token, canary.Type, canary.Description, canary.CreatedAt); err != nil {
		log.Printf("Warning: Failed to store canary token: %v", err)
	}

	return canary
}

// CheckCanaryToken checks if a value is a canary token
func (hm *HoneypotManager) CheckCanaryToken(value string) *CanaryToken {
	if canary, ok := hm.canaries[value]; ok {
		now := time.Now()
		canary.TriggeredAt = &now
		canary.TriggerCount++

		// Update database
		ctx := context.Background()
		if _, err := hm.db.ExecContext(ctx, `
			UPDATE canary_tokens SET triggered_at = $1, trigger_count = trigger_count + 1
			WHERE token = $2
		`, now, value); err != nil {
			log.Printf("Warning: failed to update canary token: %v", err)
		}

		return canary
	}
	return nil
}

// ============================================
// FAKE DATA HONEYPOTS
// Planted in database to detect unauthorized access
// ============================================

// FakeUser represents a honeypot user account
type FakeUser struct {
	UserID      uuid.UUID
	PhoneNumber string
	Username    string
	IsHoneypot  bool
}

// CreateHoneypotUsers creates fake user accounts for detection
func (hm *HoneypotManager) CreateHoneypotUsers(ctx context.Context) error {
	fakeUsers := []FakeUser{
		{
			UserID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			PhoneNumber: "+10000000001",
			Username:    "admin_backup",
		},
		{
			UserID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			PhoneNumber: "+10000000002",
			Username:    "system_test",
		},
		{
			UserID:      uuid.MustParse("00000000-0000-0000-0000-000000000003"),
			PhoneNumber: "+10000000003",
			Username:    "security_audit",
		},
	}

	for _, user := range fakeUsers {
		if _, err := hm.db.ExecContext(ctx, `
			INSERT INTO users (user_id, phone_number, username, public_identity_key, public_signed_prekey, signed_prekey_signature, is_honeypot)
			VALUES ($1, $2, $3, 'honeypot_key', 'honeypot_prekey', 'honeypot_sig', true)
			ON CONFLICT (user_id) DO NOTHING
		`, user.UserID, user.PhoneNumber, user.Username); err != nil {
			log.Printf("Warning: failed to create honeypot user: %v", err)
		}
	}

	return nil
}

// IsHoneypotUser checks if a user is a honeypot
func (hm *HoneypotManager) IsHoneypotUser(ctx context.Context, userID uuid.UUID) bool {
	var isHoneypot bool
	err := hm.db.QueryRowContext(ctx, `
		SELECT COALESCE(is_honeypot, false) FROM users WHERE user_id = $1
	`, userID).Scan(&isHoneypot)

	return err == nil && isHoneypot
}

// ============================================
// DECOY DOCUMENTS
// Fake sensitive files with tracking
// ============================================

// GenerateDecoyDocument creates a trackable fake document
func (hm *HoneypotManager) GenerateDecoyDocument(docType string) ([]byte, string) {
	canary := hm.GenerateCanaryToken("document", docType)

	var content []byte
	switch docType {
	case "credentials":
		content = []byte(`# INTERNAL CREDENTIALS - DO NOT SHARE
# Last Updated: 2024-01-01
# Tracking ID: ` + canary.Token + `

ADMIN_PASSWORD=SuperSecretPassword123!
DATABASE_URL=postgres://admin:` + canary.Token + `@db.internal:5432/prod
AWS_SECRET=` + canary.Token + `
`)
	case "api_keys":
		content = []byte(`{
  "tracking_id": "` + canary.Token + `",
  "api_keys": {
    "stripe": "sk_live_` + canary.Token[:24] + `",
    "twilio": "SK` + canary.Token[:32] + `",
    "sendgrid": "SG.` + canary.Token + `"
  }
}`)
	default:
		content = []byte("Canary: " + canary.Token)
	}

	return content, canary.Token
}
