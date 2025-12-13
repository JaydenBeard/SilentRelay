package tests

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jaydenbeard/messaging-app/internal/security"
)

func TestAuditLoggerShutdown(t *testing.T) {
	// Skip in CI/short mode - this test creates AuditLoggers that may hang
	if testing.Short() {
		t.Skip("Skipping audit shutdown tests in short mode")
	}

	// Create a test database with timeout
	db, err := sql.Open("postgres", "postgres://messaging:testpassword@localhost:5432/messaging?sslmode=disable&connect_timeout=5")
	if err != nil {
		t.Skip("Skipping test: could not open database connection")
	}
	defer db.Close()

	// Test connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Skip("Skipping test: database not available - ", err)
	}

	// Create audit logger
	config := security.DefaultAuditConfig()
	config.QueueSize = 100
	config.BatchSize = 10
	config.FlushInterval = 1 * time.Second

	auditLogger := security.NewAuditLoggerWithConfig(db, config)

	// Add some test events to the queue
	for i := 0; i < 50; i++ {
		event := &security.AuditEvent{
			ID:        uuid.New(),
			EventType: security.AuditEventLoginSuccess,
			Severity:  security.AuditSeverityMedium,
			Result:    security.AuditResultSuccess,
			Timestamp: time.Now().UTC(),
		}
		auditLogger.Log(event)
	}

	// Test shutdown with sufficient timeout
	err = auditLogger.Shutdown(10 * time.Second)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	// Verify that shutdown completed successfully
	// The queue is now closed internally, preventing new events

	// Test shutdown timeout scenario
	// This is harder to test reliably, but we can verify the timeout mechanism exists
	// by checking that shutdown returns an error when timeout occurs
}

func TestAuditLoggerShutdownWithPendingEvents(t *testing.T) {
	// Create a test database with timeout
	db, err := sql.Open("postgres", "postgres://messaging:testpassword@localhost:5432/messaging?sslmode=disable&connect_timeout=5")
	if err != nil {
		t.Skip("Skipping test: could not open database connection")
	}
	defer db.Close()

	// Test connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Skip("Skipping test: database not available - ", err)
	}

	// Create audit logger with slower flush interval
	config := security.DefaultAuditConfig()
	config.QueueSize = 100
	config.BatchSize = 5
	config.FlushInterval = 5 * time.Second // Slower flush to test pending events

	auditLogger := security.NewAuditLoggerWithConfig(db, config)

	// Add events that will take time to process
	for i := 0; i < 25; i++ {
		event := &security.AuditEvent{
			ID:        uuid.New(),
			EventType: security.AuditEventLoginSuccess,
			Severity:  security.AuditSeverityMedium,
			Result:    security.AuditResultSuccess,
			Timestamp: time.Now().UTC(),
		}
		auditLogger.Log(event)
	}

	// Shutdown should wait for all pending events to be processed
	err = auditLogger.Shutdown(15 * time.Second)
	if err != nil {
		t.Errorf("Shutdown with pending events failed: %v", err)
	}
}

func TestAuditLoggerShutdownTimeout(t *testing.T) {
	// This test verifies that shutdown respects timeout
	// Note: This is a more complex test that might need mocking

	// Create a test database with timeout
	db, err := sql.Open("postgres", "postgres://messaging:testpassword@localhost:5432/messaging?sslmode=disable&connect_timeout=5")
	if err != nil {
		t.Skip("Skipping test: could not open database connection")
	}
	defer db.Close()

	// Test connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Skip("Skipping test: database not available - ", err)
	}

	config := security.DefaultAuditConfig()
	config.QueueSize = 100
	config.BatchSize = 10
	config.FlushInterval = 1 * time.Second

	auditLogger := security.NewAuditLoggerWithConfig(db, config)

	// Add some events
	for i := 0; i < 30; i++ {
		event := &security.AuditEvent{
			ID:        uuid.New(),
			EventType: security.AuditEventLoginSuccess,
			Severity:  security.AuditSeverityMedium,
			Result:    security.AuditResultSuccess,
			Timestamp: time.Now().UTC(),
		}
		auditLogger.Log(event)
	}

	// Test with very short timeout - should timeout
	err = auditLogger.Shutdown(1 * time.Millisecond)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	} else if err.Error() != "audit logger shutdown timed out after 1ms" {
		t.Errorf("Unexpected error message: %v", err)
	}
}
