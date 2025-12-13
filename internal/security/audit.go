package security

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jaydenbeard/messaging-app/internal/metrics"
	"github.com/lib/pq"
)

// AuditEventType defines the type of security event
type AuditEventType string

// AuditSeverity defines the severity level of an audit event
type AuditSeverity string

// AuditResult defines the outcome of an audited action
type AuditResult string

const (
	// Authentication events
	AuditEventLoginAttempt   AuditEventType = "login_attempt"
	AuditEventLoginSuccess   AuditEventType = "login_success"
	AuditEventLoginFailed    AuditEventType = "login_failed"
	AuditEventLogout         AuditEventType = "logout"
	AuditEventPINVerified    AuditEventType = "pin_verified"
	AuditEventPINFailed      AuditEventType = "pin_failed"
	AuditEventPINLocked      AuditEventType = "pin_locked"
	AuditEventSessionCreated AuditEventType = "session_created"
	AuditEventSessionRevoked AuditEventType = "session_revoked"
	AuditEventMFAEnabled     AuditEventType = "mfa_enabled"
	AuditEventMFADisabled    AuditEventType = "mfa_disabled"
	AuditEventMFAVerified    AuditEventType = "mfa_verified"

	// Key events
	AuditEventKeyGenerated      AuditEventType = "key_generated"
	AuditEventKeyRotated        AuditEventType = "key_rotated"
	AuditEventKeyRevoked        AuditEventType = "key_revoked"
	AuditEventPrekeysUploaded   AuditEventType = "prekeys_uploaded"
	AuditEventPrekeysLow        AuditEventType = "prekeys_low"
	AuditEventRecoveryKeyViewed AuditEventType = "recovery_key_viewed"
	AuditEventRecoveryKeyUsed   AuditEventType = "recovery_key_used"

	// Device events
	AuditEventDeviceAdded      AuditEventType = "device_added"
	AuditEventDeviceRemoved    AuditEventType = "device_removed"
	AuditEventDeviceSuspicious AuditEventType = "device_suspicious"
	AuditEventDeviceApproved   AuditEventType = "device_approved"
	AuditEventDeviceRejected   AuditEventType = "device_rejected"

	// Security events
	AuditEventSafetyVerified    AuditEventType = "safety_verified"
	AuditEventBruteForceBlocked AuditEventType = "brute_force_blocked"
	AuditEventSuspiciousIP      AuditEventType = "suspicious_ip"
	AuditEventReplayAttempt     AuditEventType = "replay_attempt"
	AuditEventInvalidRequest    AuditEventType = "invalid_request"
	AuditEventRateLimited       AuditEventType = "rate_limited"
	AuditEventHoneypotTriggered AuditEventType = "honeypot_triggered"
	AuditEventIntrusionDetected AuditEventType = "intrusion_detected"

	// Account events
	AuditEventProfileUpdated  AuditEventType = "profile_updated"
	AuditEventPrivacyChanged  AuditEventType = "privacy_changed"
	AuditEventAccountBlocked  AuditEventType = "account_blocked"
	AuditEventAccountDeleted  AuditEventType = "account_deleted"
	AuditEventAccountCreated  AuditEventType = "account_created"
	AuditEventAccountRecovery AuditEventType = "account_recovery"

	// Admin events
	AuditEventAdminAction      AuditEventType = "admin_action"
	AuditEventConfigChanged    AuditEventType = "config_changed"
	AuditEventPermissionGrant  AuditEventType = "permission_grant"
	AuditEventPermissionRevoke AuditEventType = "permission_revoke"

	// Data access events
	AuditEventDataExport   AuditEventType = "data_export"
	AuditEventDataAccess   AuditEventType = "data_access"
	AuditEventDataModified AuditEventType = "data_modified"
	AuditEventDataDeleted  AuditEventType = "data_deleted"
)

const (
	// Severity levels for compliance
	AuditSeverityCritical AuditSeverity = "critical"
	AuditSeverityHigh     AuditSeverity = "high"
	AuditSeverityMedium   AuditSeverity = "medium"
	AuditSeverityLow      AuditSeverity = "low"
	AuditSeverityInfo     AuditSeverity = "info"
)

const (
	// Result outcomes
	AuditResultSuccess AuditResult = "success"
	AuditResultFailure AuditResult = "failure"
	AuditResultDenied  AuditResult = "denied"
	AuditResultError   AuditResult = "error"
	AuditResultPending AuditResult = "pending"
)

// AuditConfig holds configuration for audit logging
type AuditConfig struct {
	MinSeverity            AuditSeverity    `json:"min_severity"`
	AllowedEventTypes      []AuditEventType `json:"allowed_event_types"`
	QueueSize              int              `json:"queue_size"`
	BatchSize              int              `json:"batch_size"`
	FlushInterval          time.Duration    `json:"flush_interval"`
	MaxRetries             int              `json:"max_retries"`
	BaseRetryDelay         time.Duration    `json:"base_retry_delay"`
	MaxConcurrentOverflows int              `json:"max_concurrent_overflows"`
	AuditFailureLogPath    string           `json:"audit_failure_log_path"`
}

// DefaultAuditConfig returns default audit configuration
func DefaultAuditConfig() *AuditConfig {
	return &AuditConfig{
		MinSeverity:            AuditSeverityInfo,
		AllowedEventTypes:      nil, // nil means all allowed
		QueueSize:              100000,
		BatchSize:              100,
		FlushInterval:          5 * time.Second,
		MaxRetries:             3,
		BaseRetryDelay:         100 * time.Millisecond,
		MaxConcurrentOverflows: 10,
		AuditFailureLogPath:    "/tmp/audit_failures.log",
	}
}

// validateAuditConfigWithLogging validates the audit configuration with detailed logging
func validateAuditConfigWithLogging(config *AuditConfig) error {
	log.Printf("[AUDIT_CONFIG] Starting audit configuration validation")

	// Create comprehensive validator for configuration validation
	validator := NewComprehensiveAuditValidator(nil) // nil auditLogger for config-only validation

	// Use comprehensive validation
	err := validator.ValidateAuditConfigurationWithComprehensiveChecks(config)
	if err != nil {
		return err
	}

	log.Printf("[AUDIT_CONFIG] All audit configuration validations passed successfully")
	return nil
}

// ValidateAuditConfig validates the audit configuration
func ValidateAuditConfig(config *AuditConfig) error {
	return validateAuditConfigWithLogging(config)
}

// AuditEvent represents a security audit log entry with compliance-ready fields
type AuditEvent struct {
	// Core identification
	ID        uuid.UUID  `json:"id"`
	UserID    *uuid.UUID `json:"user_id,omitempty"`
	SessionID *uuid.UUID `json:"session_id,omitempty"`
	DeviceID  *uuid.UUID `json:"device_id,omitempty"`

	// Event classification
	EventType AuditEventType `json:"event_type"`
	Severity  AuditSeverity  `json:"severity"`
	Result    AuditResult    `json:"result"`

	// Resource information
	Resource     string `json:"resource,omitempty"`
	ResourceID   string `json:"resource_id,omitempty"`
	ResourceType string `json:"resource_type,omitempty"`

	// Action details
	Action                string         `json:"action"`
	EventData             map[string]any `json:"event_data,omitempty"`
	PreMarshaledEventData []byte         `json:"-"` // Pre-marshaled JSON for performance
	Description           string         `json:"description,omitempty"`

	// Request context
	IPAddress     string `json:"ip_address"`
	UserAgent     string `json:"user_agent"`
	RequestID     string `json:"request_id"`
	RequestPath   string `json:"request_path,omitempty"`
	RequestMethod string `json:"request_method,omitempty"`

	// Geographic context (for compliance)
	Country string `json:"country,omitempty"`
	Region  string `json:"region,omitempty"`
	City    string `json:"city,omitempty"`

	// Timing
	Timestamp time.Time `json:"timestamp"`
	Duration  int64     `json:"duration_ms,omitempty"`

	// Compliance metadata
	ComplianceFlags []string `json:"compliance_flags,omitempty"` // e.g., ["GDPR", "HIPAA", "SOC2"]
	DataCategory    string   `json:"data_category,omitempty"`    // e.g., "PII", "PHI", "financial"
	RetentionDays   int      `json:"retention_days,omitempty"`
}

// AuditLogger handles security audit logging
type AuditLogger struct {
	db                *sql.DB
	config            *AuditConfig
	queue             chan *AuditEvent
	wg                sync.WaitGroup
	shutdown          chan struct{}
	bufferPool        sync.Pool
	deadLetterChan    chan *AuditEvent
	failureLogger     *log.Logger
	failureFile       *os.File
	overflowSemaphore chan struct{} // Semaphore to limit concurrent overflow writes
}

// NewAuditLogger creates a new audit logger with default settings
func NewAuditLogger(db *sql.DB) *AuditLogger {
	return NewAuditLoggerWithConfig(db, DefaultAuditConfig())
}

// NewAuditLoggerWithConfig creates a new audit logger with custom configuration
func NewAuditLoggerWithConfig(db *sql.DB, config *AuditConfig) *AuditLogger {
	// Validate configuration first
	if err := ValidateAuditConfig(config); err != nil {
		log.Printf("Invalid audit configuration: %v", err)
		// Fall back to default configuration
		config = DefaultAuditConfig()
		log.Printf("Falling back to default audit configuration")
	}

	// Create failure logger for audit system failures
	var failureFile *os.File
	var err error
	if config.AuditFailureLogPath != "" {
		failureFile, err = os.OpenFile(config.AuditFailureLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			log.Printf("Failed to open audit failure log file at %s: %v", config.AuditFailureLogPath, err)
			failureFile = os.Stderr
		}
	} else {
		// Default to /tmp/audit_failures.log if no path is configured (always writable)
		failureFile, err = os.OpenFile("/tmp/audit_failures.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			log.Printf("Failed to open default audit failure log file: %v", err)
			failureFile = os.Stderr
		}
	}
	failureLogger := log.New(failureFile, "[AUDIT_FAILURE] ", log.LstdFlags|log.LUTC)

	al := &AuditLogger{
		db:                db,
		config:            config,
		queue:             make(chan *AuditEvent, config.QueueSize),
		shutdown:          make(chan struct{}),
		deadLetterChan:    make(chan *AuditEvent, 1000), // buffer for failed events
		failureLogger:     failureLogger,
		failureFile:       failureFile,
		overflowSemaphore: make(chan struct{}, config.MaxConcurrentOverflows),
		bufferPool: sync.Pool{
			New: func() any {
				return &bytes.Buffer{}
			},
		},
	}

	// Start background writer
	al.wg.Add(1)
	go al.batchWriter()

	// Start dead letter handler
	al.wg.Add(1)
	go al.deadLetterHandler()

	return al
}

// Shutdown gracefully shuts down the audit logger
func (al *AuditLogger) Shutdown(timeout time.Duration) error {
	// Step 1: Close the main queue channel to stop accepting new events
	close(al.queue)

	// Step 2: Signal shutdown to processing goroutines
	close(al.shutdown)

	// Step 3: Wait for all pending events to be processed with timeout
	done := make(chan struct{})
	go func() {
		// Wait for batch writer and dead letter handler to complete
		al.wg.Wait()

		// Close the failure log file
		if al.failureFile != nil && al.failureFile != os.Stderr {
			if err := al.failureFile.Close(); err != nil {
				log.Printf("Warning: failed to close failure file: %v", err)
			}
		}
		close(done)
	}()

	// Step 4: Handle timeout to prevent indefinite blocking
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		// Timeout occurred, but we still need to ensure resources are cleaned up
		// The goroutines should have completed their work, but we'll force cleanup
		if al.failureFile != nil && al.failureFile != os.Stderr {
			if err := al.failureFile.Close(); err != nil {
				log.Printf("Warning: failed to close failure file: %v", err)
			}
		}
		return fmt.Errorf("audit logger shutdown timed out after %v", timeout)
	}
}

// Log records an audit event
func (al *AuditLogger) Log(event *AuditEvent) {
	// Set defaults
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.Severity == "" {
		event.Severity = getSeverityForEventType(event.EventType)
	}
	if event.Result == "" {
		event.Result = AuditResultSuccess
	}

	// Check filters
	if !al.shouldLog(event) {
		return
	}

	// Pre-marshal EventData for performance
	if event.EventData != nil {
		buf := al.bufferPool.Get().(*bytes.Buffer)
		buf.Reset()
		if err := json.NewEncoder(buf).Encode(event.EventData); err == nil {
			event.PreMarshaledEventData = make([]byte, buf.Len())
			copy(event.PreMarshaledEventData, buf.Bytes())
		}
		al.bufferPool.Put(buf)
	}

	select {
	case al.queue <- event:
		metrics.AuditQueueDepth.Set(float64(len(al.queue)))
	default:
		// Queue full, spawn goroutine for overflow
		metrics.AuditOverflowEventsTotal.Inc()
		go func() {
			// Acquire semaphore slot for concurrent overflow control
			al.overflowSemaphore <- struct{}{}
			defer func() {
				<-al.overflowSemaphore
			}()

			if err := al.write(event); err != nil {
				al.failureLogger.Printf("Failed to write overflow audit event: %v", err)
			}
			metrics.AuditEventsProcessedTotal.Inc()
		}()
	}
}

// shouldLog checks if an event should be logged based on configuration filters
func (al *AuditLogger) shouldLog(event *AuditEvent) bool {
	// Create comprehensive validator for event validation
	validator := NewComprehensiveAuditValidator(al)

	// Use comprehensive validation
	err := validator.ValidateAuditEventWithContext(context.Background(), event)
	if err != nil {
		log.Printf("[AUDIT_EVENT_FILTERED] Event failed validation: %v", err)
		return false
	}

	// Critical events always pass validation and get logged
	if event.Severity == AuditSeverityCritical {
		log.Printf("[AUDIT_CRITICAL_BYPASS] Critical event bypassed filtering: EventType=%s, EventID=%s",
			event.EventType, event.ID)
		return true
	}

	return true
}

// getSeverityLevel returns a numeric level for severity comparison
func getSeverityLevel(severity AuditSeverity) int {
	switch severity {
	case AuditSeverityCritical:
		return 5
	case AuditSeverityHigh:
		return 4
	case AuditSeverityMedium:
		return 3
	case AuditSeverityLow:
		return 2
	case AuditSeverityInfo:
		return 1
	default:
		return 0
	}
}

// LogFromRequest creates and logs an event from HTTP request
func (al *AuditLogger) LogFromRequest(r *http.Request, userID *uuid.UUID, eventType AuditEventType, data map[string]any) {
	event := &AuditEvent{
		ID:            uuid.New(),
		UserID:        userID,
		EventType:     eventType,
		Severity:      getSeverityForEventType(eventType),
		Result:        AuditResultSuccess,
		EventData:     data,
		IPAddress:     GetRealIP(r),
		UserAgent:     r.UserAgent(),
		RequestID:     r.Header.Get("X-Request-ID"),
		RequestPath:   r.URL.Path,
		RequestMethod: r.Method,
		Timestamp:     time.Now().UTC(),
	}
	al.Log(event)
}

// LogSecurityEvent logs a security-related event with appropriate severity
func (al *AuditLogger) LogSecurityEvent(ctx context.Context, eventType AuditEventType, result AuditResult, userID *uuid.UUID, description string, data map[string]any) {
	event := &AuditEvent{
		ID:          uuid.New(),
		UserID:      userID,
		EventType:   eventType,
		Severity:    getSeverityForEventType(eventType),
		Result:      result,
		Description: description,
		EventData:   data,
		Timestamp:   time.Now().UTC(),
	}
	al.Log(event)
}

// LogAdminAction logs an administrative action for compliance
func (al *AuditLogger) LogAdminAction(adminID uuid.UUID, action string, resource string, resourceID string, data map[string]any) {
	event := &AuditEvent{
		ID:              uuid.New(),
		UserID:          &adminID,
		EventType:       AuditEventAdminAction,
		Severity:        AuditSeverityHigh,
		Result:          AuditResultSuccess,
		Action:          action,
		Resource:        resource,
		ResourceID:      resourceID,
		ResourceType:    "admin",
		EventData:       data,
		Timestamp:       time.Now().UTC(),
		ComplianceFlags: []string{"SOC2", "audit_trail"},
	}
	al.Log(event)
}

// LogDataAccess logs data access for compliance (GDPR, HIPAA)
func (al *AuditLogger) LogDataAccess(userID uuid.UUID, resource string, resourceID string, dataCategory string, data map[string]any) {
	event := &AuditEvent{
		ID:              uuid.New(),
		UserID:          &userID,
		EventType:       AuditEventDataAccess,
		Severity:        AuditSeverityMedium,
		Result:          AuditResultSuccess,
		Resource:        resource,
		ResourceID:      resourceID,
		ResourceType:    "data",
		DataCategory:    dataCategory,
		EventData:       data,
		Timestamp:       time.Now().UTC(),
		ComplianceFlags: []string{"GDPR", "data_access"},
	}
	al.Log(event)
}

// batchWriter processes queued events in batches
func (al *AuditLogger) batchWriter() {
	defer al.wg.Done()

	batch := make([]*AuditEvent, 0, al.config.BatchSize)
	ticker := time.NewTicker(al.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case event := <-al.queue:
			batch = append(batch, event)
			if len(batch) >= al.config.BatchSize {
				al.writeBatch(batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				al.writeBatch(batch)
				batch = batch[:0]
			}

		case <-al.shutdown:
			// Drain remaining events
			for {
				select {
				case event := <-al.queue:
					batch = append(batch, event)
				default:
					if len(batch) > 0 {
						al.writeBatch(batch)
					}
					return
				}
			}
		}
	}
}

// deadLetterHandler processes permanently failed audit events
func (al *AuditLogger) deadLetterHandler() {
	defer al.wg.Done()

	for {
		select {
		case event := <-al.deadLetterChan:
			// Log the failed event to the failure log
			al.failureLogger.Printf("Permanently failed audit event: ID=%s, Type=%s, UserID=%v, Error=Max retries exceeded",
				event.ID, event.EventType, event.UserID)
		case <-al.shutdown:
			// Drain remaining failed events
			for {
				select {
				case event := <-al.deadLetterChan:
					al.failureLogger.Printf("Permanently failed audit event on shutdown: ID=%s, Type=%s, UserID=%v",
						event.ID, event.EventType, event.UserID)
				default:
					return
				}
			}
		}
	}
}

// retryDBOperation retries a database operation with exponential backoff and comprehensive error handling
func (al *AuditLogger) retryDBOperation(events []*AuditEvent, operation func() error) error {
	var lastErr error
	delay := al.config.BaseRetryDelay

	// Create comprehensive validator for error handling
	validator := NewComprehensiveAuditValidator(al)

	for attempt := 0; attempt <= al.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(delay)
			delay *= 2 // exponential backoff
		}

		err := operation()
		if err == nil {
			// Log successful operation after retries
			if attempt > 0 {
				log.Printf("[AUDIT_RETRY_SUCCESS] Operation succeeded after %d retries", attempt)
			}
			return nil
		}

		lastErr = err

		// Comprehensive error classification and handling
		errorType := classifyDatabaseError(err)
		log.Printf("[AUDIT_DB_ERROR] %s error (attempt %d/%d): %v",
			errorType, attempt+1, al.config.MaxRetries+1, err)

		// Log all retry attempts with comprehensive details
		for _, event := range events {
			al.failureLogger.Printf("Audit DB operation failed (%s, attempt %d/%d): %v, EventID=%s, EventType=%s",
				errorType, attempt+1, al.config.MaxRetries+1, err, event.ID, event.EventType)

			// Add error context to event data for forensic analysis
			if event.EventData == nil {
				event.EventData = make(map[string]any)
			}
			event.EventData["audit_error_type"] = errorType
			event.EventData["audit_error_message"] = err.Error()
			event.EventData["audit_retry_attempt"] = attempt + 1
		}

		// Check if this is a critical error that should fail fast
		if isCriticalDatabaseError(err) {
			log.Printf("[AUDIT_CRITICAL_ERROR] Critical database error detected, failing fast: %v", err)
			break
		}
	}

	// All retries exhausted, send failed events to dead letter queue with comprehensive handling
	for _, event := range events {
		// Validate event before sending to dead letter queue
		if err := validator.ValidateAuditEventBeforeLogging(event); err != nil {
			al.failureLogger.Printf("Failed event validation before dead letter queue: ID=%s, Error=%v", event.ID, err)
			continue
		}

		select {
		case al.deadLetterChan <- event:
			al.failureLogger.Printf("Sent failed event to dead letter queue: ID=%s, Type=%s, Error=%v",
				event.ID, event.EventType, lastErr)
			metrics.AuditDeadLetterEventsTotal.Inc()
		default:
			// Dead letter queue full, log to failure logger with comprehensive details
			al.failureLogger.Printf("Dead letter queue full, dropping failed event: ID=%s, Type=%s, Error=%v",
				event.ID, event.EventType, lastErr)
			metrics.AuditDroppedEventsTotal.Inc()

			// If this is a critical event, we need to ensure it's not lost
			if event.Severity == AuditSeverityCritical {
				// Write critical events to emergency log file
				al.writeCriticalEventToEmergencyLog(event, lastErr)
			}
		}
	}

	return lastErr
}

// classifyDatabaseError classifies database errors for appropriate handling
func classifyDatabaseError(err error) string {
	if err == nil {
		return "unknown"
	}

	errorStr := err.Error()

	// Check for common database error patterns
	if strings.Contains(errorStr, "connection refused") ||
		strings.Contains(errorStr, "network error") ||
		strings.Contains(errorStr, "dial") {
		return "connection_error"
	}

	if strings.Contains(errorStr, "timeout") ||
		strings.Contains(errorStr, "deadline exceeded") {
		return "timeout_error"
	}

	if strings.Contains(errorStr, "deadlock") ||
		strings.Contains(errorStr, "lock") {
		return "deadlock_error"
	}

	if strings.Contains(errorStr, "disk full") ||
		strings.Contains(errorStr, "storage") {
		return "storage_error"
	}

	if strings.Contains(errorStr, "syntax") ||
		strings.Contains(errorStr, "SQL") {
		return "syntax_error"
	}

	if strings.Contains(errorStr, "constraint") ||
		strings.Contains(errorStr, "duplicate") {
		return "constraint_error"
	}

	return "general_error"
}

// isCriticalDatabaseError checks if an error is critical and should fail fast
func isCriticalDatabaseError(err error) bool {
	if err == nil {
		return false
	}

	errorStr := err.Error()

	// These errors indicate fundamental database problems that won't resolve with retries
	criticalPatterns := []string{
		"database does not exist",
		"table does not exist",
		"permission denied",
		"authentication failed",
		"role does not exist",
		"fatal",
		"panic",
	}

	for _, pattern := range criticalPatterns {
		if strings.Contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

// writeCriticalEventToEmergencyLog writes critical events to emergency log when normal logging fails
func (al *AuditLogger) writeCriticalEventToEmergencyLog(event *AuditEvent, err error) {
	// Create emergency log file
	emergencyLogFile := "/tmp/audit_emergency_critical.log"
	file, fileErr := os.OpenFile(emergencyLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if fileErr != nil {
		al.failureLogger.Printf("Failed to open emergency log file: %v, Critical event lost: ID=%s", fileErr, event.ID)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			al.failureLogger.Printf("Warning: failed to close file: %v", err)
		}
	}()

	// Write comprehensive emergency log entry
	emergencyLog := fmt.Sprintf("[EMERGENCY_CRITICAL_EVENT] Time=%s, EventID=%s, EventType=%s, UserID=%v, Error=%v\n",
		time.Now().UTC().Format(time.RFC3339),
		event.ID, event.EventType, event.UserID, err)

	if _, writeErr := file.WriteString(emergencyLog); writeErr != nil {
		al.failureLogger.Printf("Failed to write to emergency log: %v, Critical event: ID=%s", writeErr, event.ID)
	}

	// Also log to stderr for immediate visibility
	log.Printf("[EMERGENCY_CRITICAL_EVENT] %s", emergencyLog)
}

// writeBatch writes a batch of events to the database
func (al *AuditLogger) writeBatch(events []*AuditEvent) {
	start := time.Now()
	defer func() {
		metrics.AuditBatchWriteLatency.Observe(time.Since(start).Seconds())
		metrics.AuditBatchSize.Observe(float64(len(events)))
	}()
	if len(events) == 0 {
		return
	}

	// Use retry logic for the entire batch operation
	err := al.retryDBOperation(events, func() error {
		tx, err := al.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to start transaction for audit batch: %w", err)
		}

		stmt, err := tx.Prepare(`
			INSERT INTO security_audit_log
			(id, user_id, session_id, device_id, event_type, severity, result,
			 resource, resource_id, resource_type, action, event_data, description,
			 ip_address, user_agent, request_id, request_path, request_method,
			 country, region, city, timestamp, duration_ms, compliance_flags, data_category)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)
		`)
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				al.failureLogger.Printf("Warning: rollback failed: %v", rbErr)
			}
			return fmt.Errorf("failed to prepare audit batch statement: %w", err)
		}

		for _, event := range events {
			// Use pre-marshaled event data if available to avoid redundant marshaling
			var eventData []byte
			if len(event.PreMarshaledEventData) > 0 {
				eventData = event.PreMarshaledEventData
			} else {
				eventData, _ = json.Marshal(event.EventData)
			}

			// Use pq.Array for PostgreSQL array type
			complianceFlags := pq.Array(event.ComplianceFlags)

			_, err = stmt.Exec(
				event.ID, event.UserID, event.SessionID, event.DeviceID,
				event.EventType, event.Severity, event.Result,
				event.Resource, event.ResourceID, event.ResourceType,
				event.Action, eventData, event.Description,
				event.IPAddress, event.UserAgent, event.RequestID,
				event.RequestPath, event.RequestMethod,
				event.Country, event.Region, event.City,
				event.Timestamp, event.Duration, complianceFlags, event.DataCategory,
			)
			if err != nil {
				if rbErr := tx.Rollback(); rbErr != nil {
					log.Printf("Warning: tx.Rollback failed: %v", rbErr)
				}
				if clErr := stmt.Close(); clErr != nil {
					log.Printf("Warning: stmt.Close failed: %v", clErr)
				}
				return fmt.Errorf("failed to insert audit event %s: %w", event.ID, err)
			}
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit audit batch: %w", err)
		}

		return nil
	})

	if err != nil {
		al.failureLogger.Printf("Audit batch write failed after retries: %v", err)
	}
}

// write persists a single event to the database
func (al *AuditLogger) write(event *AuditEvent) error {
	// Use retry logic for single event write
	return al.retryDBOperation([]*AuditEvent{event}, func() error {
		// Use pre-marshaled event data if available to avoid redundant marshaling
		var eventData []byte
		if len(event.PreMarshaledEventData) > 0 {
			eventData = event.PreMarshaledEventData
		} else {
			eventData, _ = json.Marshal(event.EventData)
		}

		// Use pq.Array for PostgreSQL array type
		complianceFlags := pq.Array(event.ComplianceFlags)

		_, err := al.db.Exec(`
			INSERT INTO security_audit_log
			(id, user_id, session_id, device_id, event_type, severity, result,
			 resource, resource_id, resource_type, action, event_data, description,
			 ip_address, user_agent, request_id, request_path, request_method,
			 country, region, city, timestamp, duration_ms, compliance_flags, data_category)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)
		`, event.ID, event.UserID, event.SessionID, event.DeviceID,
			event.EventType, event.Severity, event.Result,
			event.Resource, event.ResourceID, event.ResourceType,
			event.Action, eventData, event.Description,
			event.IPAddress, event.UserAgent, event.RequestID,
			event.RequestPath, event.RequestMethod,
			event.Country, event.Region, event.City,
			event.Timestamp, event.Duration, complianceFlags, event.DataCategory,
		)

		if err != nil {
			return fmt.Errorf("failed to write audit log: %w", err)
		}
		return nil
	})
}

// getSeverityForEventType returns the default severity for an event type
func getSeverityForEventType(eventType AuditEventType) AuditSeverity {
	switch eventType {
	case AuditEventLoginFailed, AuditEventPINFailed, AuditEventPINLocked,
		AuditEventBruteForceBlocked, AuditEventSuspiciousIP, AuditEventReplayAttempt,
		AuditEventIntrusionDetected, AuditEventHoneypotTriggered:
		return AuditSeverityHigh

	case AuditEventAccountDeleted, AuditEventAccountBlocked, AuditEventKeyRevoked,
		AuditEventDeviceSuspicious, AuditEventAdminAction, AuditEventConfigChanged:
		return AuditSeverityCritical

	case AuditEventLoginSuccess, AuditEventSessionCreated, AuditEventDeviceAdded,
		AuditEventKeyRotated, AuditEventPermissionGrant, AuditEventPermissionRevoke:
		return AuditSeverityMedium

	case AuditEventProfileUpdated, AuditEventPrivacyChanged, AuditEventDataAccess:
		return AuditSeverityLow

	default:
		return AuditSeverityInfo
	}
}

// getCriticalEventTypes returns a list of event types that are considered critical
func getCriticalEventTypes() []AuditEventType {
	return []AuditEventType{
		AuditEventAccountDeleted,
		AuditEventAccountBlocked,
		AuditEventKeyRevoked,
		AuditEventDeviceSuspicious,
		AuditEventAdminAction,
		AuditEventConfigChanged,
	}
}

// containsEventType checks if an event type is in a slice of event types
func containsEventType(eventTypes []AuditEventType, target AuditEventType) bool {
	for _, et := range eventTypes {
		if et == target {
			return true
		}
	}
	return false
}

// TestValidateAuditConfig demonstrates the validation functionality
func TestValidateAuditConfig() {
	// Test valid configuration
	validConfig := &AuditConfig{
		MaxConcurrentOverflows: 50,
		QueueSize:              1000,
		BatchSize:              100,
		MaxRetries:             3,
		FlushInterval:          5 * time.Second,
	}

	if err := ValidateAuditConfig(validConfig); err != nil {
		log.Printf("Valid config failed validation: %v", err)
	} else {
		log.Printf("Valid config passed validation")
	}

	// Test invalid MaxConcurrentOverflows (too low)
	invalidLowConfig := &AuditConfig{
		MaxConcurrentOverflows: 0,
		QueueSize:              1000,
		BatchSize:              100,
		MaxRetries:             3,
		FlushInterval:          5 * time.Second,
	}

	if err := ValidateAuditConfig(invalidLowConfig); err != nil {
		log.Printf("Invalid low config correctly failed validation: %v", err)
	} else {
		log.Printf("Invalid low config incorrectly passed validation")
	}

	// Test invalid MaxConcurrentOverflows (too high)
	invalidHighConfig := &AuditConfig{
		MaxConcurrentOverflows: 150,
		QueueSize:              1000,
		BatchSize:              100,
		MaxRetries:             3,
		FlushInterval:          5 * time.Second,
	}

	if err := ValidateAuditConfig(invalidHighConfig); err != nil {
		log.Printf("Invalid high config correctly failed validation: %v", err)
	} else {
		log.Printf("Invalid high config incorrectly passed validation")
	}

	// Test MinSeverity that would exclude critical events
	invalidMinSeverityConfig := &AuditConfig{
		MinSeverity:            AuditSeverityHigh, // This would exclude critical events
		MaxConcurrentOverflows: 50,
		QueueSize:              1000,
		BatchSize:              100,
		MaxRetries:             3,
		FlushInterval:          5 * time.Second,
	}

	if err := ValidateAuditConfig(invalidMinSeverityConfig); err != nil {
		log.Printf("Invalid MinSeverity config correctly failed validation: %v", err)
	} else {
		log.Printf("Invalid MinSeverity config incorrectly passed validation")
	}

	// Test AllowedEventTypes that excludes critical event types
	invalidAllowedTypesConfig := &AuditConfig{
		AllowedEventTypes: []AuditEventType{
			AuditEventLoginSuccess,
			AuditEventLoginFailed,
			// Missing critical event types like AuditEventAccountDeleted, AuditEventAdminAction, etc.
		},
		MaxConcurrentOverflows: 50,
		QueueSize:              1000,
		BatchSize:              100,
		MaxRetries:             3,
		FlushInterval:          5 * time.Second,
	}

	if err := ValidateAuditConfig(invalidAllowedTypesConfig); err != nil {
		log.Printf("Invalid AllowedEventTypes config correctly failed validation: %v", err)
	} else {
		log.Printf("Invalid AllowedEventTypes config incorrectly passed validation")
	}
}

// Query retrieves audit events for a user
func (al *AuditLogger) Query(ctx context.Context, userID uuid.UUID, eventType *AuditEventType, limit int) ([]*AuditEvent, error) {
	var query string
	var args []any

	if eventType != nil {
		query = `
			SELECT user_id, event_type, event_data, ip_address, user_agent, created_at
			FROM security_audit_log
			WHERE user_id = $1 AND event_type = $2
			ORDER BY created_at DESC
			LIMIT $3
		`
		args = []any{userID, *eventType, limit}
	} else {
		query = `
			SELECT user_id, event_type, event_data, ip_address, user_agent, created_at
			FROM security_audit_log
			WHERE user_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		`
		args = []any{userID, limit}
	}

	rows, err := al.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var events []*AuditEvent
	for rows.Next() {
		event := &AuditEvent{}
		var eventData []byte

		err := rows.Scan(
			&event.UserID,
			&event.EventType,
			&eventData,
			&event.IPAddress,
			&event.UserAgent,
			&event.Timestamp,
		)
		if err != nil {
			return nil, err
		}

		if len(eventData) > 0 {
			if err := json.Unmarshal(eventData, &event.EventData); err != nil {
				log.Printf("Warning: failed to unmarshal event data: %v", err)
			}
		}

		events = append(events, event)
	}

	return events, nil
}

// GetRecentSecurityEvents gets recent security-related events
func (al *AuditLogger) GetRecentSecurityEvents(ctx context.Context, userID uuid.UUID) ([]*AuditEvent, error) {
	query := `
		SELECT user_id, event_type, event_data, ip_address, user_agent, created_at
		FROM security_audit_log
		WHERE user_id = $1 
		AND event_type IN ('login_failed', 'pin_failed', 'pin_locked', 'device_suspicious', 'brute_force_blocked', 'suspicious_ip', 'replay_attempt')
		AND created_at > NOW() - INTERVAL '24 hours'
		ORDER BY created_at DESC
	`

	rows, err := al.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var events []*AuditEvent
	for rows.Next() {
		event := &AuditEvent{}
		var eventData []byte

		err := rows.Scan(
			&event.UserID,
			&event.EventType,
			&eventData,
			&event.IPAddress,
			&event.UserAgent,
			&event.Timestamp,
		)
		if err != nil {
			return nil, err
		}

		if len(eventData) > 0 {
			if err := json.Unmarshal(eventData, &event.EventData); err != nil {
				log.Printf("Warning: failed to unmarshal event data: %v", err)
			}
		}

		events = append(events, event)
	}

	return events, nil
}

// CheckSuspiciousActivity checks for suspicious patterns
func (al *AuditLogger) CheckSuspiciousActivity(ctx context.Context, userID uuid.UUID) (bool, string) {
	// Check for multiple failed logins
	var failedCount int
	if err := al.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM security_audit_log
		WHERE user_id = $1 
		AND event_type = 'login_failed'
		AND created_at > NOW() - INTERVAL '1 hour'
	`, userID).Scan(&failedCount); err != nil {
		log.Printf("Warning: failed to check failed login count: %v", err)
	}

	if failedCount >= 5 {
		return true, "Multiple failed login attempts"
	}

	// Check for logins from new IPs
	var newIPCount int
	if err := al.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT ip_address) FROM security_audit_log
		WHERE user_id = $1
		AND event_type = 'login_success'
		AND created_at > NOW() - INTERVAL '24 hours'
		AND ip_address NOT IN (
			SELECT DISTINCT ip_address FROM security_audit_log
			WHERE user_id = $1
			AND event_type = 'login_success'
			AND created_at < NOW() - INTERVAL '24 hours'
		)
	`, userID).Scan(&newIPCount); err != nil {
		log.Printf("Warning: failed to check new IP count: %v", err)
	}

	if newIPCount >= 3 {
		return true, "Logins from multiple new locations"
	}

	return false, ""
}
