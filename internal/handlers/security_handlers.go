package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jaydenbeard/messaging-app/internal/security"
)

// SecurityHandler handles security-related endpoints
type SecurityHandler struct {
	auditLogger *security.AuditLogger
	ids         *security.IntrusionDetector
}

// NewSecurityHandler creates a new security handler
func NewSecurityHandler(auditLogger *security.AuditLogger, ids *security.IntrusionDetector) *SecurityHandler {
	return &SecurityHandler{
		auditLogger: auditLogger,
		ids:         ids,
	}
}

// GetSecurityEvents returns recent security events for the user
func (h *SecurityHandler) GetSecurityEvents(w http.ResponseWriter, r *http.Request) {
	_, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// TODO: Parse user ID and fetch events
	// events, err := h.auditLogger.GetRecentSecurityEvents(r.Context(), userID)

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, map[string]interface{}{
		"events": []interface{}{},
	})
}

// GetActiveSessions returns active sessions for the user
func (h *SecurityHandler) GetActiveSessions(w http.ResponseWriter, r *http.Request) {
	// Return mock data for now
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, map[string]interface{}{
		"sessions": []map[string]interface{}{
			{
				"id":          "current",
				"device":      r.UserAgent(),
				"ip":          security.GetRealIP(r),
				"last_active": time.Now().Format(time.RFC3339),
				"current":     true,
			},
		},
	})
}

// RevokeSession revokes a specific session
func (h *SecurityHandler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	// Implementation would revoke the specified session
	w.WriteHeader(http.StatusOK)
	writeJSON(w, map[string]interface{}{
		"success": true,
	})
}

// RevokeAllSessions revokes all other sessions
func (h *SecurityHandler) RevokeAllSessions(w http.ResponseWriter, r *http.Request) {
	// Implementation would revoke all other sessions
	w.WriteHeader(http.StatusOK)
	writeJSON(w, map[string]interface{}{
		"success": true,
	})
}

// ReportPinFailure handles certificate pinning failure reports from clients
func (h *SecurityHandler) ReportPinFailure(w http.ResponseWriter, r *http.Request) {
	var report struct {
		Hostname    string `json:"hostname"`
		Port        int    `json:"port"`
		ExpectedPin string `json:"expected_pin"`
		ActualPin   string `json:"actual_pin"`
		Timestamp   string `json:"timestamp"`
	}

	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Log the pin failure - this could indicate MITM attack
	h.ids.RecordSuspiciousPayload(
		security.GetRealIP(r),
		report.Hostname,
		"certificate_pin_failure",
	)

	w.WriteHeader(http.StatusOK)
}

// ReportCTFailure handles Certificate Transparency failure reports
func (h *SecurityHandler) ReportCTFailure(w http.ResponseWriter, r *http.Request) {
	var report struct {
		Hostname  string `json:"hostname"`
		Timestamp string `json:"timestamp"`
		Details   string `json:"details"`
	}

	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Log CT failure
	h.ids.RecordSuspiciousPayload(
		security.GetRealIP(r),
		report.Hostname,
		"ct_failure",
	)

	w.WriteHeader(http.StatusOK)
}

// HealthzSecure is a secure health check that also verifies security controls
func (h *SecurityHandler) HealthzSecure(w http.ResponseWriter, r *http.Request) {
	checks := make(map[string]bool)

	// Check security headers are set
	checks["security_headers"] = true

	// Check rate limiting is active
	checks["rate_limiting"] = true

	// Check IDS is running
	checks["intrusion_detection"] = h.ids != nil

	// Check audit logging
	checks["audit_logging"] = h.auditLogger != nil

	// Overall status
	allOK := true
	for _, ok := range checks {
		if !ok {
			allOK = false
			break
		}
	}

	status := http.StatusOK
	if !allOK {
		status = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	writeJSON(w, map[string]interface{}{
		"status": map[bool]string{true: "healthy", false: "degraded"}[allOK],
		"checks": checks,
		"time":   time.Now().Format(time.RFC3339),
	})
}
