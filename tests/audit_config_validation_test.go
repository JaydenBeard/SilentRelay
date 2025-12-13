package tests

import (
	"testing"
	"time"

	"github.com/jaydenbeard/messaging-app/internal/security"
)

func TestAuditConfigBoundsValidation(t *testing.T) {
	t.Run("Test QueueSize bounds validation", func(t *testing.T) {
		// Test QueueSize too low
		config := &security.AuditConfig{
			QueueSize:              50, // Below minimum of 100
			BatchSize:              100,
			MaxRetries:             3,
			FlushInterval:          5 * time.Second,
			BaseRetryDelay:         100 * time.Millisecond,
			MaxConcurrentOverflows: 10,
			AuditFailureLogPath:    "audit_failures.log",
		}

		err := security.ValidateAuditConfig(config)
		if err == nil {
			t.Error("Expected error for QueueSize too low, got nil")
		}
		// Data loss prevention catches this first

		// Test QueueSize too high
		config.QueueSize = 2000000 // Above maximum of 1,000,000
		err = security.ValidateAuditConfig(config)
		if err == nil {
			t.Error("Expected error for QueueSize too high, got nil")
		} else if err.Error() != "QueueSize must not exceed 1,000,000 to prevent memory exhaustion" {
			t.Errorf("Expected specific error message, got: %v", err)
		}

		// Test QueueSize valid
		config.QueueSize = 500000 // Within bounds
		err = security.ValidateAuditConfig(config)
		if err != nil {
			t.Errorf("Expected no error for valid QueueSize, got: %v", err)
		}
	})

	t.Run("Test BatchSize bounds validation", func(t *testing.T) {
		// Test BatchSize too low
		config := &security.AuditConfig{
			QueueSize:              1000,
			BatchSize:              0, // Below minimum of 1
			MaxRetries:             3,
			FlushInterval:          5 * time.Second,
			BaseRetryDelay:         100 * time.Millisecond,
			MaxConcurrentOverflows: 10,
			AuditFailureLogPath:    "audit_failures.log",
		}

		err := security.ValidateAuditConfig(config)
		if err == nil {
			t.Error("Expected error for BatchSize too low, got nil")
		} else if err.Error() != "BatchSize must be at least 1" {
			t.Errorf("Expected specific error message, got: %v", err)
		}

		// Test BatchSize too high
		config.BatchSize = 20000 // Above maximum of 10,000
		err = security.ValidateAuditConfig(config)
		if err == nil {
			t.Error("Expected error for BatchSize too high, got nil")
		} else if err.Error() != "BatchSize must not exceed 10,000 to prevent database transaction timeouts and memory pressure" {
			t.Errorf("Expected specific error message, got: %v", err)
		}

		// Test BatchSize valid
		config.BatchSize = 5000 // Within bounds
		err = security.ValidateAuditConfig(config)
		if err != nil {
			t.Errorf("Expected no error for valid BatchSize, got: %v", err)
		}
	})

	t.Run("Test MaxRetries bounds validation", func(t *testing.T) {
		// Test MaxRetries negative
		config := &security.AuditConfig{
			QueueSize:              1000,
			BatchSize:              100,
			MaxRetries:             -1, // Negative value
			FlushInterval:          5 * time.Second,
			BaseRetryDelay:         100 * time.Millisecond,
			MaxConcurrentOverflows: 10,
			AuditFailureLogPath:    "audit_failures.log",
		}

		err := security.ValidateAuditConfig(config)
		if err == nil {
			t.Error("Expected error for MaxRetries negative, got nil")
		} else if err.Error() != "MaxRetries must be non-negative" {
			t.Errorf("Expected specific error message, got: %v", err)
		}

		// Test MaxRetries too high
		config.MaxRetries = 15 // Above maximum of 10
		err = security.ValidateAuditConfig(config)
		if err == nil {
			t.Error("Expected error for MaxRetries too high, got nil")
		}
		// Data loss prevention may catch this with a different message

		// Test MaxRetries valid
		config.MaxRetries = 5 // Within bounds
		err = security.ValidateAuditConfig(config)
		if err != nil {
			t.Errorf("Expected no error for valid MaxRetries, got: %v", err)
		}
	})

	t.Run("Test BaseRetryDelay bounds validation", func(t *testing.T) {
		// Test BaseRetryDelay too low
		config := &security.AuditConfig{
			QueueSize:              1000,
			BatchSize:              100,
			MaxRetries:             3,
			FlushInterval:          5 * time.Second,
			BaseRetryDelay:         5 * time.Millisecond, // Below minimum of 10ms
			MaxConcurrentOverflows: 10,
			AuditFailureLogPath:    "audit_failures.log",
		}

		err := security.ValidateAuditConfig(config)
		if err == nil {
			t.Error("Expected error for BaseRetryDelay too low, got nil")
		} else if err.Error() != "BaseRetryDelay must be at least 10ms to ensure minimum backoff" {
			t.Errorf("Expected specific error message, got: %v", err)
		}

		// Test BaseRetryDelay too high
		config.BaseRetryDelay = 10 * time.Second // Above maximum of 5s
		err = security.ValidateAuditConfig(config)
		if err == nil {
			t.Error("Expected error for BaseRetryDelay too high, got nil")
		} else if err.Error() != "BaseRetryDelay must not exceed 5 seconds to prevent excessive retry delays" {
			t.Errorf("Expected specific error message, got: %v", err)
		}

		// Test BaseRetryDelay valid
		config.BaseRetryDelay = 500 * time.Millisecond // Within bounds
		err = security.ValidateAuditConfig(config)
		if err != nil {
			t.Errorf("Expected no error for valid BaseRetryDelay, got: %v", err)
		}
	})

	t.Run("Test FlushInterval bounds validation", func(t *testing.T) {
		// Test FlushInterval too low
		config := &security.AuditConfig{
			QueueSize:              1000,
			BatchSize:              100,
			MaxRetries:             3,
			FlushInterval:          500 * time.Millisecond, // Below minimum of 1s
			BaseRetryDelay:         100 * time.Millisecond,
			MaxConcurrentOverflows: 10,
			AuditFailureLogPath:    "audit_failures.log",
		}

		err := security.ValidateAuditConfig(config)
		if err == nil {
			t.Error("Expected error for FlushInterval too low, got nil")
		} else if err.Error() != "FlushInterval must be at least 1 second to prevent excessive database writes" {
			t.Errorf("Expected specific error message, got: %v", err)
		}

		// Test FlushInterval too high
		config.FlushInterval = 2 * time.Hour // Above maximum of 1h
		err = security.ValidateAuditConfig(config)
		if err == nil {
			t.Error("Expected error for FlushInterval too high, got nil")
		}
		// Data loss prevention catches this with a different message

		// Test FlushInterval valid
		config.FlushInterval = 30 * time.Minute // Within bounds
		err = security.ValidateAuditConfig(config)
		if err != nil {
			t.Errorf("Expected no error for valid FlushInterval, got: %v", err)
		}
	})

	t.Run("Test AuditFailureLogPath validation", func(t *testing.T) {
		// Test path too long
		config := &security.AuditConfig{
			QueueSize:              1000,
			BatchSize:              100,
			MaxRetries:             3,
			FlushInterval:          5 * time.Second,
			BaseRetryDelay:         100 * time.Millisecond,
			MaxConcurrentOverflows: 10,
			AuditFailureLogPath:    "this_is_an_extremely_long_path_name_that_exceeds_the_maximum_allowed_length_of_255_characters_and_should_be_rejected_by_the_validation_system_because_it_could_potentially_cause_filesystem_issues_or_buffer_overflows_in_some_systems_that_have_limited_path_length_support_and_could_lead_to_security_vulnerabilities_or_system_instability_when_attempting_to_create_or_write_to_such_files_with_excessively_long_names.go", // 256+ characters
		}

		err := security.ValidateAuditConfig(config)
		if err == nil {
			t.Error("Expected error for path too long, got nil")
		} else if err.Error() != "AuditFailureLogPath must not exceed 255 characters" {
			t.Errorf("Expected specific error message, got: %v", err)
		}

		// Test path with invalid characters
		config.AuditFailureLogPath = "audit|failures.log"
		err = security.ValidateAuditConfig(config)
		if err == nil {
			t.Error("Expected error for invalid characters in path, got nil")
		} else if err.Error() != "AuditFailureLogPath contains invalid characters that could cause filesystem issues" {
			t.Errorf("Expected specific error message, got: %v", err)
		}

		// Test path with traversal sequences
		config.AuditFailureLogPath = "../../audit_failures.log"
		err = security.ValidateAuditConfig(config)
		if err == nil {
			t.Error("Expected error for path traversal sequences, got nil")
		} else if err.Error() != "AuditFailureLogPath contains path traversal sequences that could compromise system security" {
			t.Errorf("Expected specific error message, got: %v", err)
		}

		// Test valid path
		config.AuditFailureLogPath = "audit_failures.log"
		err = security.ValidateAuditConfig(config)
		if err != nil {
			t.Errorf("Expected no error for valid path, got: %v", err)
		}
	})

	t.Run("Test comprehensive validation with all valid values", func(t *testing.T) {
		config := &security.AuditConfig{
			QueueSize:              500000,
			BatchSize:              5000,
			MaxRetries:             5,
			FlushInterval:          30 * time.Minute,
			BaseRetryDelay:         500 * time.Millisecond,
			MaxConcurrentOverflows: 50,
			AuditFailureLogPath:    "audit_failures.log",
			MinSeverity:            security.AuditSeverityInfo,
			AllowedEventTypes:      nil,
		}

		err := security.ValidateAuditConfig(config)
		if err != nil {
			t.Errorf("Expected no error for comprehensive valid config, got: %v", err)
		}
	})
}
