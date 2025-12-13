package security

import (
	"context"
	"database/sql"
	"log"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================
// INTRUSION DETECTION SYSTEM
// Detect and respond to attacks in real-time
// ============================================

// AlertLevel represents severity of security events
type AlertLevel int

const (
	AlertLow AlertLevel = iota
	AlertMedium
	AlertHigh
	AlertCritical
)

func (a AlertLevel) String() string {
	switch a {
	case AlertLow:
		return "low"
	case AlertMedium:
		return "medium"
	case AlertHigh:
		return "high"
	case AlertCritical:
		return "critical"
	}
	return "unknown"
}

// SecurityAlert represents a detected security event
type SecurityAlert struct {
	ID           uuid.UUID
	Type         string
	Level        AlertLevel
	SourceIP     net.IP
	UserID       *uuid.UUID
	Description  string
	Evidence     map[string]interface{}
	Timestamp    time.Time
	Acknowledged bool
}

// IntrusionDetector monitors for security threats
type IntrusionDetector struct {
	db          *sql.DB
	auditLogger *AuditLogger
	alerts      chan *SecurityAlert

	// Thresholds
	failedLoginThreshold int
	rateLimitThreshold   int
	scanDetectionWindow  time.Duration

	// Tracking
	mu             sync.RWMutex
	failedLogins   map[string][]time.Time    // IP -> timestamps
	endpointAccess map[string]map[string]int // IP -> endpoint -> count
	blockedIPs     map[string]time.Time
}

// NewIntrusionDetector creates a new IDS
func NewIntrusionDetector(db *sql.DB, auditLogger *AuditLogger) *IntrusionDetector {
	ids := &IntrusionDetector{
		db:                   db,
		auditLogger:          auditLogger,
		alerts:               make(chan *SecurityAlert, 100),
		failedLoginThreshold: 5,
		rateLimitThreshold:   100,
		scanDetectionWindow:  time.Minute,
		failedLogins:         make(map[string][]time.Time),
		endpointAccess:       make(map[string]map[string]int),
		blockedIPs:           make(map[string]time.Time),
	}

	go ids.processAlerts()
	go ids.cleanupLoop()

	return ids
}

// RecordFailedLogin records a failed login attempt
func (ids *IntrusionDetector) RecordFailedLogin(ip string, userID *uuid.UUID) {
	ids.mu.Lock()
	defer ids.mu.Unlock()

	now := time.Now()
	ids.failedLogins[ip] = append(ids.failedLogins[ip], now)

	// Filter old entries
	cutoff := now.Add(-ids.scanDetectionWindow)
	filtered := make([]time.Time, 0)
	for _, t := range ids.failedLogins[ip] {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}
	ids.failedLogins[ip] = filtered

	// Check threshold
	if len(filtered) >= ids.failedLoginThreshold {
		ids.raiseAlert(&SecurityAlert{
			ID:          uuid.New(),
			Type:        "brute_force_attempt",
			Level:       AlertHigh,
			SourceIP:    net.ParseIP(ip),
			UserID:      userID,
			Description: "Multiple failed login attempts detected",
			Evidence: map[string]interface{}{
				"attempt_count": len(filtered),
				"window":        ids.scanDetectionWindow.String(),
			},
			Timestamp: now,
		})

		// Auto-block IP temporarily
		ids.blockIP(ip, 15*time.Minute)
	}
}

// RecordEndpointAccess tracks endpoint access for scan detection
func (ids *IntrusionDetector) RecordEndpointAccess(ip, endpoint string) {
	ids.mu.Lock()
	defer ids.mu.Unlock()

	if ids.endpointAccess[ip] == nil {
		ids.endpointAccess[ip] = make(map[string]int)
	}
	ids.endpointAccess[ip][endpoint]++

	// Check for scanning behavior (accessing many different endpoints)
	if len(ids.endpointAccess[ip]) > 50 {
		ids.raiseAlert(&SecurityAlert{
			ID:          uuid.New(),
			Type:        "endpoint_scanning",
			Level:       AlertMedium,
			SourceIP:    net.ParseIP(ip),
			Description: "Possible endpoint scanning detected",
			Evidence: map[string]interface{}{
				"unique_endpoints": len(ids.endpointAccess[ip]),
			},
			Timestamp: time.Now(),
		})
	}
}

// RecordSuspiciousPayload detects attack payloads
func (ids *IntrusionDetector) RecordSuspiciousPayload(ip string, payload string, attackType string) {
	ids.raiseAlert(&SecurityAlert{
		ID:          uuid.New(),
		Type:        attackType,
		Level:       AlertHigh,
		SourceIP:    net.ParseIP(ip),
		Description: "Attack payload detected and blocked",
		Evidence: map[string]interface{}{
			"payload_snippet": truncate(payload, 100),
		},
		Timestamp: time.Now(),
	})

	// Block IP immediately for detected attacks
	ids.blockIP(ip, 1*time.Hour)
}

// RecordKeyCompromise handles potential key compromise
func (ids *IntrusionDetector) RecordKeyCompromise(userID uuid.UUID, reason string) {
	ids.raiseAlert(&SecurityAlert{
		ID:          uuid.New(),
		Type:        "key_compromise",
		Level:       AlertCritical,
		UserID:      &userID,
		Description: "Potential key compromise detected",
		Evidence: map[string]interface{}{
			"reason": reason,
		},
		Timestamp: time.Now(),
	})
}

// IsBlocked checks if an IP is blocked
func (ids *IntrusionDetector) IsBlocked(ip string) bool {
	ids.mu.RLock()
	defer ids.mu.RUnlock()

	if blockedUntil, ok := ids.blockedIPs[ip]; ok {
		if time.Now().Before(blockedUntil) {
			return true
		}
		// Expired, will be cleaned up
	}
	return false
}

// blockIP blocks an IP for a duration
func (ids *IntrusionDetector) blockIP(ip string, duration time.Duration) {
	ids.blockedIPs[ip] = time.Now().Add(duration)

	// Persist to database
	ctx := context.Background()
	if _, err := ids.db.ExecContext(ctx, `
		INSERT INTO ip_reputation (ip_address, blocked_until, threat_score)
		VALUES ($1, $2, $3)
		ON CONFLICT (ip_address) DO UPDATE SET 
			blocked_until = EXCLUDED.blocked_until,
			threat_score = ip_reputation.threat_score + 10
	`, ip, time.Now().Add(duration), 50); err != nil {
		log.Printf("Warning: failed to persist blocked IP: %v", err)
	}
}

// raiseAlert sends an alert for processing
func (ids *IntrusionDetector) raiseAlert(alert *SecurityAlert) {
	select {
	case ids.alerts <- alert:
	default:
		// Channel full, log directly
		ids.logAlert(alert)
	}
}

// processAlerts handles incoming alerts
func (ids *IntrusionDetector) processAlerts() {
	for alert := range ids.alerts {
		ids.logAlert(alert)

		// Create security incident for high/critical
		if alert.Level >= AlertHigh {
			ids.createIncident(alert)
		}
	}
}

// logAlert logs an alert to the database
func (ids *IntrusionDetector) logAlert(alert *SecurityAlert) {
	ctx := context.Background()

	if _, err := ids.db.ExecContext(ctx, `
		INSERT INTO security_incidents 
		(incident_id, incident_type, severity, affected_user_id, source_ip, description, evidence)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, alert.ID, alert.Type, alert.Level.String(), alert.UserID,
		alert.SourceIP.String(), alert.Evidence); err != nil {
		log.Printf("Warning: failed to log security alert: %v", err)
	}
}

// createIncident creates a security incident
func (ids *IntrusionDetector) createIncident(alert *SecurityAlert) {
	// In production, this would:
	// - Send to SIEM
	// - Page on-call
	// - Create Jira ticket
	// - etc.
}

// cleanupLoop removes expired entries
func (ids *IntrusionDetector) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		ids.mu.Lock()

		now := time.Now()

		// Cleanup blocked IPs
		for ip, until := range ids.blockedIPs {
			if now.After(until) {
				delete(ids.blockedIPs, ip)
			}
		}

		// Cleanup endpoint access tracking
		ids.endpointAccess = make(map[string]map[string]int)

		ids.mu.Unlock()
	}
}

// GetActiveAlerts returns recent alerts
func (ids *IntrusionDetector) GetActiveAlerts(ctx context.Context, limit int) ([]*SecurityAlert, error) {
	rows, err := ids.db.QueryContext(ctx, `
		SELECT incident_id, incident_type, severity, affected_user_id, source_ip, description, evidence, created_at
		FROM security_incidents
		WHERE status = 'open'
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var alerts []*SecurityAlert
	for rows.Next() {
		alert := &SecurityAlert{}
		var severity string
		var sourceIP string

		err := rows.Scan(
			&alert.ID, &alert.Type, &severity, &alert.UserID,
			&sourceIP, &alert.Description, &alert.Evidence, &alert.Timestamp,
		)
		if err != nil {
			continue
		}

		alert.SourceIP = net.ParseIP(sourceIP)
		switch severity {
		case "low":
			alert.Level = AlertLow
		case "medium":
			alert.Level = AlertMedium
		case "high":
			alert.Level = AlertHigh
		case "critical":
			alert.Level = AlertCritical
		}

		alerts = append(alerts, alert)
	}

	return alerts, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
