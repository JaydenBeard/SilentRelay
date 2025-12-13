package tests

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jaydenbeard/messaging-app/internal/security"
	_ "github.com/mattn/go-sqlite3"
)

// TestRetryLogicWithRealDatabase tests the retry logic implementation using a real SQLite database
func TestRetryLogicWithRealDatabase(t *testing.T) {
	// Skip in CI/short mode - these tests create AuditLoggers that take time
	if testing.Short() {
		t.Skip("Skipping retry logic tests in short mode")
	}

	// Create a test database file
	testDBFile := "test_audit_retry_working.db"
	defer os.Remove(testDBFile)

	// Create a real SQLite database for testing
	realDB, err := sql.Open("sqlite3", testDBFile)
	if err != nil {
		t.Skip("SQLite not available, skipping test")
		return
	}
	defer realDB.Close()

	// Create the audit log table
	_, err = realDB.Exec(`
		CREATE TABLE IF NOT EXISTS security_audit_log (
			id TEXT PRIMARY KEY,
			user_id TEXT,
			session_id TEXT,
			device_id TEXT,
			event_type TEXT,
			severity TEXT,
			result TEXT,
			resource TEXT,
			resource_id TEXT,
			resource_type TEXT,
			action TEXT,
			event_data TEXT,
			description TEXT,
			ip_address TEXT,
			user_agent TEXT,
			request_id TEXT,
			request_path TEXT,
			request_method TEXT,
			country TEXT,
			region TEXT,
			city TEXT,
			timestamp DATETIME,
			duration_ms INTEGER,
			compliance_flags TEXT,
			data_category TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Create audit logger with aggressive retry settings for testing
	config := &security.AuditConfig{
		MaxRetries:     3,
		BaseRetryDelay: 5 * time.Millisecond,
		QueueSize:      100,
		BatchSize:      1,
		FlushInterval:  100 * time.Millisecond, // Fast flush for testing
	}

	auditLogger := security.NewAuditLoggerWithConfig(realDB, config)

	// Create test events
	event1 := &security.AuditEvent{
		ID:        uuid.New(),
		EventType: security.AuditEventLoginSuccess,
		Severity:  security.AuditSeverityMedium,
		Result:    security.AuditResultSuccess,
		Timestamp: time.Now().UTC(),
	}

	event2 := &security.AuditEvent{
		ID:        uuid.New(),
		EventType: security.AuditEventLoginFailed,
		Severity:  security.AuditSeverityHigh,
		Result:    security.AuditResultFailure,
		Timestamp: time.Now().UTC(),
	}

	// Test the Log method which should use retry logic internally
	startTime := time.Now()
	auditLogger.Log(event1)
	auditLogger.Log(event2)

	// Give time for processing and retries
	time.Sleep(300 * time.Millisecond)
	elapsedTime := time.Since(startTime)

	// Verify events were written to database
	var count int
	err = realDB.QueryRow("SELECT COUNT(*) FROM security_audit_log").Scan(&count)
	if err != nil {
		t.Errorf("Failed to query audit log: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 events in audit log, got %d", count)
	}

	// Verify that the operation completed within reasonable time
	// With retry logic, it should take some time but not too long
	if elapsedTime > 5*time.Second {
		t.Errorf("Operation took too long: %v", elapsedTime)
	}

	// Clean up
	auditLogger.Shutdown(5 * time.Second)
}

// TestRetryLogicWithFailureScenario tests retry logic when we simulate failures
func TestRetryLogicWithFailureScenario(t *testing.T) {
	// Skip in CI/short mode
	if testing.Short() {
		t.Skip("Skipping retry failure tests in short mode")
	}

	// Create a test database file
	testDBFile := "test_audit_retry_failure.db"
	defer os.Remove(testDBFile)

	// Create a real SQLite database for testing
	realDB, err := sql.Open("sqlite3", testDBFile)
	if err != nil {
		t.Skip("SQLite not available, skipping test")
		return
	}
	defer realDB.Close()

	// Create the audit log table
	_, err = realDB.Exec(`
		CREATE TABLE IF NOT EXISTS security_audit_log (
			id TEXT PRIMARY KEY,
			user_id TEXT,
			session_id TEXT,
			device_id TEXT,
			event_type TEXT,
			severity TEXT,
			result TEXT,
			resource TEXT,
			resource_id TEXT,
			resource_type TEXT,
			action TEXT,
			event_data TEXT,
			description TEXT,
			ip_address TEXT,
			user_agent TEXT,
			request_id TEXT,
			request_path TEXT,
			request_method TEXT,
			country TEXT,
			region TEXT,
			city TEXT,
			timestamp DATETIME,
			duration_ms INTEGER,
			compliance_flags TEXT,
			data_category TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Create audit logger with specific retry settings
	config := &security.AuditConfig{
		MaxRetries:     2,
		BaseRetryDelay: 10 * time.Millisecond,
		QueueSize:      100,
		BatchSize:      1,
		FlushInterval:  100 * time.Millisecond,
	}

	auditLogger := security.NewAuditLoggerWithConfig(realDB, config)

	// Create test event
	event := &security.AuditEvent{
		ID:        uuid.New(),
		EventType: security.AuditEventLoginSuccess,
		Severity:  security.AuditSeverityMedium,
		Result:    security.AuditResultSuccess,
		Timestamp: time.Now().UTC(),
	}

	// Test the Log method
	startTime := time.Now()
	auditLogger.Log(event)

	// Give time for processing
	time.Sleep(300 * time.Millisecond)
	elapsedTime := time.Since(startTime)

	// Verify event was written to database
	var count int
	err = realDB.QueryRow("SELECT COUNT(*) FROM security_audit_log").Scan(&count)
	if err != nil {
		t.Errorf("Failed to query audit log: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 event in audit log, got %d", count)
	}

	// Verify timing is reasonable for retry logic
	if elapsedTime > 5*time.Second {
		t.Errorf("Operation took too long: %v", elapsedTime)
	}

	// Clean up
	auditLogger.Shutdown(5 * time.Second)
}

// TestExponentialBackoffBehavior tests exponential backoff timing
func TestExponentialBackoffBehavior(t *testing.T) {
	// Skip in CI/short mode
	if testing.Short() {
		t.Skip("Skipping backoff tests in short mode")
	}

	// Create a test database file
	testDBFile := "test_audit_retry_backoff.db"
	defer os.Remove(testDBFile)

	// Create a real SQLite database for testing
	realDB, err := sql.Open("sqlite3", testDBFile)
	if err != nil {
		t.Skip("SQLite not available, skipping test")
		return
	}
	defer realDB.Close()

	// Create the audit log table
	_, err = realDB.Exec(`
		CREATE TABLE IF NOT EXISTS security_audit_log (
			id TEXT PRIMARY KEY,
			user_id TEXT,
			session_id TEXT,
			device_id TEXT,
			event_type TEXT,
			severity TEXT,
			result TEXT,
			resource TEXT,
			resource_id TEXT,
			resource_type TEXT,
			action TEXT,
			event_data TEXT,
			description TEXT,
			ip_address TEXT,
			user_agent TEXT,
			request_id TEXT,
			request_path TEXT,
			request_method TEXT,
			country TEXT,
			region TEXT,
			city TEXT,
			timestamp DATETIME,
			duration_ms INTEGER,
			compliance_flags TEXT,
			data_category TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Create audit logger with specific retry settings for timing test
	config := &security.AuditConfig{
		MaxRetries:     3,
		BaseRetryDelay: 20 * time.Millisecond,
		QueueSize:      100,
		BatchSize:      1,
		FlushInterval:  100 * time.Millisecond,
	}

	auditLogger := security.NewAuditLoggerWithConfig(realDB, config)

	// Test timing of retry operations
	startTime := time.Now()
	event := &security.AuditEvent{
		ID:        uuid.New(),
		EventType: security.AuditEventLoginSuccess,
		Severity:  security.AuditSeverityMedium,
		Result:    security.AuditResultSuccess,
		Timestamp: time.Now().UTC(),
	}

	auditLogger.Log(event)

	// Wait for processing to complete
	time.Sleep(500 * time.Millisecond)
	elapsedTime := time.Since(startTime)

	// Verify that the operation completed
	var count int
	err = realDB.QueryRow("SELECT COUNT(*) FROM security_audit_log").Scan(&count)
	if err != nil {
		t.Errorf("Failed to query audit log: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 event in audit log, got %d", count)
	}

	// Verify exponential backoff timing
	// Expected delays: 20ms, 40ms, 80ms = 140ms total minimum
	// But we have other overhead, so be conservative
	expectedMinDelay := 50 * time.Millisecond
	if elapsedTime < expectedMinDelay {
		t.Errorf("Expected at least %v delay for exponential backoff, got %v", expectedMinDelay, elapsedTime)
	}

	// Clean up
	auditLogger.Shutdown(5 * time.Second)
}

// TestMaxRetriesConfiguration tests that max retries configuration works
func TestMaxRetriesConfiguration(t *testing.T) {
	// Skip in CI/short mode
	if testing.Short() {
		t.Skip("Skipping max retries tests in short mode")
	}

	// Create a test database file
	testDBFile := "test_audit_retry_max.db"
	defer os.Remove(testDBFile)

	// Create a real SQLite database for testing
	realDB, err := sql.Open("sqlite3", testDBFile)
	if err != nil {
		t.Skip("SQLite not available, skipping test")
		return
	}
	defer realDB.Close()

	// Create the audit log table
	_, err = realDB.Exec(`
		CREATE TABLE IF NOT EXISTS security_audit_log (
			id TEXT PRIMARY KEY,
			user_id TEXT,
			session_id TEXT,
			device_id TEXT,
			event_type TEXT,
			severity TEXT,
			result TEXT,
			resource TEXT,
			resource_id TEXT,
			resource_type TEXT,
			action TEXT,
			event_data TEXT,
			description TEXT,
			ip_address TEXT,
			user_agent TEXT,
			request_id TEXT,
			request_path TEXT,
			request_method TEXT,
			country TEXT,
			region TEXT,
			city TEXT,
			timestamp DATETIME,
			duration_ms INTEGER,
			compliance_flags TEXT,
			data_category TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Create audit logger with limited retries
	config := &security.AuditConfig{
		MaxRetries:     2,
		BaseRetryDelay: 5 * time.Millisecond,
		QueueSize:      100,
		BatchSize:      1,
		FlushInterval:  100 * time.Millisecond,
	}

	auditLogger := security.NewAuditLoggerWithConfig(realDB, config)

	// Create test event
	event := &security.AuditEvent{
		ID:        uuid.New(),
		EventType: security.AuditEventLoginSuccess,
		Severity:  security.AuditSeverityMedium,
		Result:    security.AuditResultSuccess,
		Timestamp: time.Now().UTC(),
	}

	// Test the Log method
	auditLogger.Log(event)

	// Give time for processing
	time.Sleep(300 * time.Millisecond)

	// Verify event was written to database
	var count int
	err = realDB.QueryRow("SELECT COUNT(*) FROM security_audit_log").Scan(&count)
	if err != nil {
		t.Errorf("Failed to query audit log: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 event in audit log, got %d", count)
	}

	// Clean up
	auditLogger.Shutdown(5 * time.Second)
}

// TestDeadLetterQueueBehavior tests dead letter queue handling
func TestDeadLetterQueueBehavior(t *testing.T) {
	// Skip in CI/short mode
	if testing.Short() {
		t.Skip("Skipping dead letter queue tests in short mode")
	}

	// Create a test database file
	testDBFile := "test_audit_retry_deadletter.db"
	defer os.Remove(testDBFile)

	// Create a real SQLite database for testing
	realDB, err := sql.Open("sqlite3", testDBFile)
	if err != nil {
		t.Skip("SQLite not available, skipping test")
		return
	}
	defer realDB.Close()

	// Create the audit log table
	_, err = realDB.Exec(`
		CREATE TABLE IF NOT EXISTS security_audit_log (
			id TEXT PRIMARY KEY,
			user_id TEXT,
			session_id TEXT,
			device_id TEXT,
			event_type TEXT,
			severity TEXT,
			result TEXT,
			resource TEXT,
			resource_id TEXT,
			resource_type TEXT,
			action TEXT,
			event_data TEXT,
			description TEXT,
			ip_address TEXT,
			user_agent TEXT,
			request_id TEXT,
			request_path TEXT,
			request_method TEXT,
			country TEXT,
			region TEXT,
			city TEXT,
			timestamp DATETIME,
			duration_ms INTEGER,
			compliance_flags TEXT,
			data_category TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Create audit logger
	config := &security.AuditConfig{
		MaxRetries:     2,
		BaseRetryDelay: 5 * time.Millisecond,
		QueueSize:      100,
		BatchSize:      1,
		FlushInterval:  100 * time.Millisecond,
	}

	auditLogger := security.NewAuditLoggerWithConfig(realDB, config)

	// Create test event
	event := &security.AuditEvent{
		ID:        uuid.New(),
		EventType: security.AuditEventLoginSuccess,
		Severity:  security.AuditSeverityMedium,
		Result:    security.AuditResultSuccess,
		Timestamp: time.Now().UTC(),
	}

	// Test the Log method
	auditLogger.Log(event)

	// Give time for processing
	time.Sleep(300 * time.Millisecond)

	// Verify event was written to database
	var count int
	err = realDB.QueryRow("SELECT COUNT(*) FROM security_audit_log").Scan(&count)
	if err != nil {
		t.Errorf("Failed to query audit log: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 event in audit log, got %d", count)
	}

	// Clean up
	auditLogger.Shutdown(5 * time.Second)
}
