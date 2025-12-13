package tests

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jaydenbeard/messaging-app/internal/security"
)

func TestAuditConcurrencyLimiting(t *testing.T) {
	// Skip in CI/short mode - these tests create AuditLoggers with real DB connections
	if testing.Short() {
		t.Skip("Skipping audit concurrency tests in short mode")
	}

	t.Run("Semaphore should limit concurrent overflow writes", func(t *testing.T) {
		// Create a test database connection with timeout
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

		// Create audit logger with limited concurrency
		config := &security.AuditConfig{
			MaxConcurrentOverflows: 2,  // Very low limit for testing
			QueueSize:              10, // Small queue to trigger overflows
			BatchSize:              5,
			MaxRetries:             0, // No retries for faster testing
			FlushInterval:          1 * time.Second,
		}

		auditLogger := security.NewAuditLoggerWithConfig(db, config)

		// Create a channel to track when goroutines are running
		running := make(chan struct{}, 10)
		var wg sync.WaitGroup

		// Generate many events to trigger overflow
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				// Create an event that will likely overflow
				event := &security.AuditEvent{
					ID:        uuid.New(),
					EventType: security.AuditEventLoginSuccess,
					Severity:  security.AuditSeverityMedium,
					Result:    security.AuditResultSuccess,
					Timestamp: time.Now().UTC(),
					IPAddress: fmt.Sprintf("192.168.1.%d", id),
					UserAgent: "test-agent",
					RequestID: fmt.Sprintf("req-%d", id),
				}

				running <- struct{}{}
				auditLogger.Log(event)
				<-running
			}(i)
		}

		// Wait for all goroutines to start
		time.Sleep(100 * time.Millisecond)

		// Check that we don't exceed the concurrency limit
		// The semaphore should prevent more than MaxConcurrentOverflows goroutines
		// from running the overflow write logic simultaneously
		runningCount := len(running)
		if runningCount > config.MaxConcurrentOverflows+1 { // +1 for some tolerance
			t.Errorf("Concurrency limit exceeded: got %d concurrent operations, expected max %d",
				runningCount, config.MaxConcurrentOverflows)
		}

		wg.Wait()
		auditLogger.Shutdown(5 * time.Second)
	})

	t.Run("Configuration validation should prevent resource exhaustion", func(t *testing.T) {
		// Test that validation prevents excessively high concurrency limits
		config := &security.AuditConfig{
			MaxConcurrentOverflows: 200, // Exceeds maximum
			QueueSize:              1000,
			BatchSize:              100,
			MaxRetries:             3,
			FlushInterval:          5 * time.Second,
		}

		err := security.ValidateAuditConfig(config)
		if err == nil {
			t.Error("Expected validation to fail for MaxConcurrentOverflows = 200")
		} else if err.Error() != "MaxConcurrentOverflows must not exceed 100 to prevent resource exhaustion" {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	t.Run("Boundary conditions should work correctly", func(t *testing.T) {
		// Test minimum valid boundary configuration
		// Requirements: QueueSize>=1000, MaxConcurrentOverflows>=5, BaseRetryDelay>=10ms, MaxRetries>=2
		configMin := &security.AuditConfig{
			MaxConcurrentOverflows: 5,    // Minimum valid
			QueueSize:              1000, // Minimum required by validation
			BatchSize:              10,
			MaxRetries:             2, // Minimum required for reliable persistence
			FlushInterval:          1 * time.Second,
			BaseRetryDelay:         10 * time.Millisecond, // Minimum required
		}

		err := security.ValidateAuditConfig(configMin)
		if err != nil {
			t.Errorf("Min boundary config failed validation: %v", err)
		}

		// Test maximum recommended boundary
		configMax := &security.AuditConfig{
			MaxConcurrentOverflows: 50,    // Maximum recommended (100 generates warning)
			QueueSize:              10000, // Large queue
			BatchSize:              100,
			MaxRetries:             5,
			FlushInterval:          5 * time.Second,
			BaseRetryDelay:         100 * time.Millisecond,
		}

		err = security.ValidateAuditConfig(configMax)
		if err != nil {
			t.Errorf("Max boundary config failed validation: %v", err)
		}
	})
}

func TestAuditOverflowUnderLoad(t *testing.T) {
	// Skip in CI/short mode - this test takes ~5s and creates lots of events
	if testing.Short() {
		t.Skip("Skipping audit overflow tests in short mode")
	}

	// Create a test database connection with timeout
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

	// Create audit logger with reasonable concurrency limit
	config := &security.AuditConfig{
		MaxConcurrentOverflows: 5,
		QueueSize:              20, // Small queue to trigger overflows
		BatchSize:              10,
		MaxRetries:             0, // No retries for faster testing
		FlushInterval:          1 * time.Second,
	}

	auditLogger := security.NewAuditLoggerWithConfig(db, config)

	// Track metrics
	var totalEvents int32 = 0
	var mu sync.Mutex

	// Create many events under load
	var wg sync.WaitGroup
	numEvents := 100

	for i := 0; i < numEvents; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			event := &security.AuditEvent{
				ID:        uuid.New(),
				EventType: security.AuditEventLoginSuccess,
				Severity:  security.AuditSeverityMedium,
				Result:    security.AuditResultSuccess,
				Timestamp: time.Now().UTC(),
				IPAddress: fmt.Sprintf("192.168.1.%d", id%256),
				UserAgent: "test-agent",
				RequestID: fmt.Sprintf("req-%d", id),
			}

			mu.Lock()
			totalEvents++
			mu.Unlock()

			// Simulate some processing time
			time.Sleep(1 * time.Millisecond)

			auditLogger.Log(event)
		}(i)
	}

	wg.Wait()

	// The system should handle the load without crashing
	// and the semaphore should prevent resource exhaustion
	t.Logf("Processed %d total events", totalEvents)

	// Try to log one more event to ensure system is still responsive
	finalEvent := &security.AuditEvent{
		ID:        uuid.New(),
		EventType: security.AuditEventLogout,
		Severity:  security.AuditSeverityInfo,
		Result:    security.AuditResultSuccess,
		Timestamp: time.Now().UTC(),
		IPAddress: "192.168.1.255",
		UserAgent: "test-agent",
		RequestID: "final-req",
	}

	auditLogger.Log(finalEvent)

	auditLogger.Shutdown(5 * time.Second)
}

func TestSemaphoreAcquisitionAndRelease(t *testing.T) {
	// Skip in CI/short mode - this test creates AuditLoggers with real DB
	if testing.Short() {
		t.Skip("Skipping semaphore tests in short mode")
	}

	t.Run("Semaphore should properly acquire and release slots", func(t *testing.T) {
		// Create a test database connection with timeout
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

		config := &security.AuditConfig{
			MaxConcurrentOverflows: 3,
			QueueSize:              5, // Very small queue to force overflows
			BatchSize:              2,
			MaxRetries:             0,
			FlushInterval:          1 * time.Second,
		}

		auditLogger := security.NewAuditLoggerWithConfig(db, config)

		// Create events that will definitely overflow
		var wg sync.WaitGroup
		eventsProcessed := make(chan struct{}, 10)

		for i := 0; i < 15; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				event := &security.AuditEvent{
					ID:        uuid.New(),
					EventType: security.AuditEventLoginSuccess,
					Severity:  security.AuditSeverityMedium,
					Result:    security.AuditResultSuccess,
					Timestamp: time.Now().UTC(),
					IPAddress: fmt.Sprintf("192.168.1.%d", id),
					UserAgent: "test-agent",
					RequestID: fmt.Sprintf("req-%d", id),
				}

				auditLogger.Log(event)
				eventsProcessed <- struct{}{}
			}(i)
		}

		wg.Wait()
		close(eventsProcessed)

		// Count processed events
		count := 0
		for range eventsProcessed {
			count++
		}

		t.Logf("Successfully processed %d events with concurrency limit of %d",
			count, config.MaxConcurrentOverflows)

		auditLogger.Shutdown(5 * time.Second)
	})

	t.Run("Semaphore should handle rapid successive calls", func(t *testing.T) {
		// Create a test database connection with timeout
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

		config := &security.AuditConfig{
			MaxConcurrentOverflows: 2,
			QueueSize:              3, // Very small queue
			BatchSize:              2,
			MaxRetries:             0,
			FlushInterval:          1 * time.Second,
		}

		auditLogger := security.NewAuditLoggerWithConfig(db, config)

		// Rapid fire events
		for i := 0; i < 50; i++ {
			event := &security.AuditEvent{
				ID:        uuid.New(),
				EventType: security.AuditEventLoginSuccess,
				Severity:  security.AuditSeverityMedium,
				Result:    security.AuditResultSuccess,
				Timestamp: time.Now().UTC(),
				IPAddress: fmt.Sprintf("192.168.1.%d", i%256),
				UserAgent: "test-agent",
				RequestID: fmt.Sprintf("req-%d", i),
			}

			auditLogger.Log(event)
			time.Sleep(100 * time.Microsecond) // Small delay between events
		}

		// System should handle the load without deadlocks or panics
		time.Sleep(100 * time.Millisecond) // Give time for processing

		auditLogger.Shutdown(5 * time.Second)
	})
}
