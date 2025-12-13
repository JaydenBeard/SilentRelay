package tests

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jaydenbeard/messaging-app/internal/security"
)

func TestComprehensiveAuditValidation(t *testing.T) {
	// Skip in CI/short mode - this test creates AuditLogger that requires a real database
	if testing.Short() {
		t.Skip("Skipping audit validation tests in short mode")
	}

	t.Run("Test Comprehensive Event Validation", func(t *testing.T) {
		// Create a mock database with timeout
		db, err := sql.Open("postgres", "postgres://messaging:testpassword@localhost:5432/messaging?sslmode=disable&connect_timeout=5")
		if err != nil {
			t.Skip("Skipping test - no database available")
		}
		defer db.Close()

		// Test connection with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			t.Skip("Skipping test - database not responding: ", err)
		}

		// Create audit logger with comprehensive validator
		config := security.DefaultAuditConfig()
		auditLogger := security.NewAuditLoggerWithConfig(db, config)
		validator := security.NewComprehensiveAuditValidator(auditLogger)

		// Test 1: Valid event should pass validation
		validEvent := &security.AuditEvent{
			ID:        uuid.New(),
			EventType: security.AuditEventLoginSuccess,
			Severity:  security.AuditSeverityMedium,
			Timestamp: time.Now().UTC(),
			IPAddress: "192.168.1.1",
			UserAgent: "Mozilla/5.0",
		}

		err = validator.ValidateAuditEventWithContext(context.Background(), validEvent)
		if err != nil {
			t.Errorf("Valid event failed validation: %v", err)
		}

		// Test 2: Event with nil ID should fail
		invalidEvent := &security.AuditEvent{
			ID:        uuid.Nil,
			EventType: security.AuditEventLoginSuccess,
			Severity:  security.AuditSeverityMedium,
			Timestamp: time.Now().UTC(),
		}

		err = validator.ValidateAuditEventWithContext(context.Background(), invalidEvent)
		if err == nil {
			t.Error("Event with nil ID should fail validation")
		}

		// Test 3: Critical event should bypass validation
		criticalEvent := &security.AuditEvent{
			ID:        uuid.New(),
			EventType: security.AuditEventAccountDeleted,
			Severity:  security.AuditSeverityCritical,
			Timestamp: time.Now().UTC(),
		}

		err = validator.ValidateAuditEventWithContext(context.Background(), criticalEvent)
		if err != nil {
			t.Errorf("Critical event should bypass validation: %v", err)
		}

		// Test 4: Event with invalid severity should fail
		invalidSeverityEvent := &security.AuditEvent{
			ID:        uuid.New(),
			EventType: security.AuditEventLoginSuccess,
			Severity:  "invalid_severity",
			Timestamp: time.Now().UTC(),
		}

		err = validator.ValidateAuditEventWithContext(context.Background(), invalidSeverityEvent)
		if err == nil {
			t.Error("Event with invalid severity should fail validation")
		}
	})

	t.Run("Test Comprehensive Configuration Validation", func(t *testing.T) {
		validator := security.NewComprehensiveAuditValidator(nil)

		// Test 1: Valid configuration should pass
		validConfig := security.DefaultAuditConfig()
		err := validator.ValidateAuditConfigurationWithComprehensiveChecks(validConfig)
		if err != nil {
			t.Errorf("Valid configuration failed validation: %v", err)
		}

		// Test 2: Configuration with an empty AllowedEventTypes list (not nil) should fail
		// Note: MinSeverity=High does NOT filter Critical events (Critical has higher severity level)
		invalidConfig := &security.AuditConfig{
			MinSeverity:            security.AuditSeverityHigh,
			QueueSize:              1000,
			BatchSize:              100,
			MaxRetries:             3,
			FlushInterval:          5 * time.Second,
			BaseRetryDelay:         100 * time.Millisecond,
			MaxConcurrentOverflows: 10,
			AuditFailureLogPath:    "audit_failures.log",
			AllowedEventTypes:      []security.AuditEventType{}, // Empty slice should fail
		}

		err = validator.ValidateAuditConfigurationWithComprehensiveChecks(invalidConfig)
		if err == nil {
			t.Error("Configuration with empty AllowedEventTypes list should fail validation")
		}

		// Test 3: Configuration with too long flush interval should fail
		invalidFlushConfig := &security.AuditConfig{
			MinSeverity:            security.AuditSeverityInfo,
			QueueSize:              1000,
			BatchSize:              100,
			MaxRetries:             3,
			FlushInterval:          2 * time.Hour, // Too long
			BaseRetryDelay:         100 * time.Millisecond,
			MaxConcurrentOverflows: 10,
			AuditFailureLogPath:    "audit_failures.log",
		}

		err = validator.ValidateAuditConfigurationWithComprehensiveChecks(invalidFlushConfig)
		if err == nil {
			t.Error("Configuration with too long flush interval should fail validation")
		}
	})

	t.Run("Test Data Loss Prevention Validation", func(t *testing.T) {
		validator := security.NewComprehensiveAuditValidator(nil)

		// Test 1: Configuration with data loss risks should fail
		riskyConfig := &security.AuditConfig{
			MinSeverity:            security.AuditSeverityHigh, // Would filter critical
			QueueSize:              100,                        // Too small
			BatchSize:              1000,                       // Too large relative to queue
			MaxRetries:             0,                          // No retries
			FlushInterval:          2 * time.Hour,              // Too long
			BaseRetryDelay:         10 * time.Second,           // Too long
			MaxConcurrentOverflows: 1,                          // Too small
			AuditFailureLogPath:    "",                         // No failure logging
			AllowedEventTypes: []security.AuditEventType{
				security.AuditEventLoginSuccess, // Missing critical types
			},
		}

		err := validator.ValidateAuditConfigurationWithComprehensiveChecks(riskyConfig)
		if err == nil {
			t.Error("Configuration with data loss risks should fail validation")
		}

		// Test 2: Safe configuration should pass
		safeConfig := &security.AuditConfig{
			MinSeverity:            security.AuditSeverityInfo,
			QueueSize:              10000,
			BatchSize:              100,
			MaxRetries:             3,
			FlushInterval:          30 * time.Second,
			BaseRetryDelay:         100 * time.Millisecond,
			MaxConcurrentOverflows: 10,
			AuditFailureLogPath:    "audit_failures.log",
			AllowedEventTypes:      nil, // All allowed
		}

		err = validator.ValidateAuditConfigurationWithComprehensiveChecks(safeConfig)
		if err != nil {
			t.Errorf("Safe configuration should pass validation: %v", err)
		}
	})

	t.Run("Test Edge Case Validation", func(t *testing.T) {
		validator := security.NewComprehensiveAuditValidator(nil)

		// Test 1: Event with extremely long description
		longDescriptionEvent := &security.AuditEvent{
			ID:          uuid.New(),
			EventType:   security.AuditEventLoginSuccess,
			Severity:    security.AuditSeverityMedium,
			Timestamp:   time.Now().UTC(),
			Description: makeString(5000), // Too long
		}

		err := validator.ValidateAuditEventWithContext(context.Background(), longDescriptionEvent)
		if err == nil {
			t.Error("Event with extremely long description should fail validation")
		}

		// Test 2: Event with suspicious IP address
		suspiciousEvent := &security.AuditEvent{
			ID:        uuid.New(),
			EventType: security.AuditEventLoginSuccess,
			Severity:  security.AuditSeverityMedium,
			Timestamp: time.Now().UTC(),
			IPAddress: "0.0.0.0", // Placeholder IP
		}

		err = validator.ValidateAuditEventWithContext(context.Background(), suspiciousEvent)
		if err != nil {
			t.Errorf("Event with placeholder IP should pass validation (with warning): %v", err)
		}

		// Test 3: Event with future timestamp
		futureEvent := &security.AuditEvent{
			ID:        uuid.New(),
			EventType: security.AuditEventLoginSuccess,
			Severity:  security.AuditSeverityMedium,
			Timestamp: time.Now().UTC().Add(1 * time.Hour), // Future timestamp
		}

		err = validator.ValidateAuditEventWithContext(context.Background(), futureEvent)
		if err == nil {
			t.Error("Event with future timestamp should fail validation")
		}
	})
}

func TestValidationMetrics(t *testing.T) {
	validator := security.NewComprehensiveAuditValidator(nil)

	// Reset metrics
	validator.ResetValidationMetrics()

	// Perform some validations
	validConfig := security.DefaultAuditConfig()
	validator.ValidateAuditConfigurationWithComprehensiveChecks(validConfig)

	// Check metrics
	metrics := validator.GetValidationMetrics()
	if metrics.TotalValidations == 0 {
		t.Error("Expected validation metrics to be recorded")
	}

	if metrics.ConfigurationValidations == 0 {
		t.Error("Expected configuration validation metrics to be recorded")
	}
}

func TestCriticalEventBypass(t *testing.T) {
	validator := security.NewComprehensiveAuditValidator(nil)

	// Reset metrics
	validator.ResetValidationMetrics()

	// Test critical event bypass
	criticalEvent := &security.AuditEvent{
		ID:        uuid.New(),
		EventType: security.AuditEventAccountDeleted,
		Severity:  security.AuditSeverityCritical,
		Timestamp: time.Now().UTC(),
	}

	err := validator.ValidateAuditEventWithContext(context.Background(), criticalEvent)
	if err != nil {
		t.Errorf("Critical event should bypass validation: %v", err)
	}

	// Check that bypass was recorded
	metrics := validator.GetValidationMetrics()
	if metrics.CriticalEventBypasses == 0 {
		t.Error("Expected critical event bypass to be recorded")
	}
}

// Helper function to create long strings for testing
func makeString(length int) string {
	result := make([]rune, length)
	for i := range result {
		result[i] = 'a'
	}
	return string(result)
}
