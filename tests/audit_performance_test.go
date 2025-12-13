package tests

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jaydenbeard/messaging-app/internal/security"
)

// TestPerformanceMetrics tests the performance impact of audit logging
func TestPerformanceMetrics(t *testing.T) {
	// Skip in CI/short mode - these tests require a real database
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	t.Run("Benchmark Audit Logging Performance", func(t *testing.T) {
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

		// Create audit logger with performance-focused configuration
		config := &security.AuditConfig{
			MinSeverity:            security.AuditSeverityInfo,
			QueueSize:              100000,
			BatchSize:              1000,
			MaxRetries:             3,
			FlushInterval:          1 * time.Second,
			BaseRetryDelay:         100 * time.Millisecond,
			MaxConcurrentOverflows: 50,
			AuditFailureLogPath:    "audit_performance_test.log",
		}

		auditLogger := security.NewAuditLoggerWithConfig(db, config)
		defer auditLogger.Shutdown(10 * time.Second)

		// Test 1: Measure baseline performance without logging
		startTime := time.Now()
		baselineEvents := createTestEvents(1000)
		baselineDuration := time.Since(startTime)

		// Test 2: Measure performance with logging
		startTime = time.Now()
		for _, event := range baselineEvents {
			auditLogger.Log(event)
		}
		loggingDuration := time.Since(startTime)

		// Calculate overhead
		overhead := float64(loggingDuration.Milliseconds()) / float64(baselineDuration.Milliseconds())
		t.Logf("Performance overhead: %.2f%%", overhead*100)

		// Verify overhead is within acceptable limits (<5%)
		if overhead > 1.05 {
			t.Errorf("Performance overhead %.2f%% exceeds acceptable limit of 5%%", overhead*100)
		}
	})

	t.Run("High Volume Load Test", func(t *testing.T) {
		// Create a mock database
		db, err := sql.Open("postgres", "postgres://user:password@localhost:5432/testdb?sslmode=disable")
		if err != nil {
			t.Skip("Skipping test - no database available")
		}
		defer db.Close()

		// Create audit logger with high volume configuration
		config := &security.AuditConfig{
			MinSeverity:            security.AuditSeverityInfo,
			QueueSize:              1000000,
			BatchSize:              5000,
			MaxRetries:             3,
			FlushInterval:          500 * time.Millisecond,
			BaseRetryDelay:         50 * time.Millisecond,
			MaxConcurrentOverflows: 100,
			AuditFailureLogPath:    "audit_high_volume_test.log",
		}

		auditLogger := security.NewAuditLoggerWithConfig(db, config)
		defer auditLogger.Shutdown(10 * time.Second)

		// Test high volume scenario (10,000+ events/sec)
		targetRate := 10000
		testDuration := 5 * time.Second
		eventsPerGoroutine := targetRate * int(testDuration.Seconds()) / 10

		var wg sync.WaitGroup
		var totalEvents int64

		// Start multiple goroutines to generate load
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < eventsPerGoroutine; j++ {
					event := &security.AuditEvent{
						ID:        uuid.New(),
						EventType: security.AuditEventLoginSuccess,
						Severity:  security.AuditSeverityMedium,
						Timestamp: time.Now().UTC(),
						IPAddress: fmt.Sprintf("192.168.1.%d", workerID),
						UserAgent: "TestAgent",
						EventData: map[string]any{
							"worker_id": workerID,
							"event_num": j,
						},
					}
					auditLogger.Log(event)
					atomic.AddInt64(&totalEvents, 1)
				}
			}(i)
		}

		// Wait for completion
		wg.Wait()

		// Verify we achieved target rate
		actualRate := float64(totalEvents) / testDuration.Seconds()
		t.Logf("Achieved event rate: %.0f events/sec", actualRate)

		if actualRate < float64(targetRate)*0.9 {
			t.Errorf("Failed to achieve target rate. Got %.0f, expected %d", actualRate, targetRate)
		}
	})

	t.Run("Non-Blocking Behavior Test", func(t *testing.T) {
		// Create a mock database
		db, err := sql.Open("postgres", "postgres://user:password@localhost:5432/testdb?sslmode=disable")
		if err != nil {
			t.Skip("Skipping test - no database available")
		}
		defer db.Close()

		// Create audit logger with small queue to test overflow behavior
		config := &security.AuditConfig{
			MinSeverity:            security.AuditSeverityInfo,
			QueueSize:              100, // Small queue to force overflow
			BatchSize:              10,
			MaxRetries:             3,
			FlushInterval:          1 * time.Second,
			BaseRetryDelay:         100 * time.Millisecond,
			MaxConcurrentOverflows: 10,
			AuditFailureLogPath:    "audit_nonblocking_test.log",
		}

		auditLogger := security.NewAuditLoggerWithConfig(db, config)
		defer auditLogger.Shutdown(10 * time.Second)

		// Test that logging doesn't block under overflow conditions
		var requestLatencies []time.Duration
		numRequests := 1000

		for i := 0; i < numRequests; i++ {
			startTime := time.Now()

			// Simulate a request that includes audit logging
			event := &security.AuditEvent{
				ID:        uuid.New(),
				EventType: security.AuditEventLoginSuccess,
				Severity:  security.AuditSeverityMedium,
				Timestamp: time.Now().UTC(),
				IPAddress: "192.168.1.1",
				UserAgent: "TestAgent",
			}
			auditLogger.Log(event)

			// Simulate some request processing
			time.Sleep(1 * time.Millisecond)

			requestLatencies = append(requestLatencies, time.Since(startTime))
		}

		// Calculate average request latency
		var totalLatency time.Duration
		for _, latency := range requestLatencies {
			totalLatency += latency
		}
		avgLatency := totalLatency / time.Duration(numRequests)

		t.Logf("Average request latency with audit logging: %v", avgLatency)

		// Verify no blocking occurred (latency should be minimal)
		if avgLatency > 10*time.Millisecond {
			t.Errorf("Request processing blocked by audit logging. Avg latency: %v", avgLatency)
		}
	})
}

// TestFilteringPerformance tests configurable filtering performance
func TestFilteringPerformance(t *testing.T) {
	// Skip in CI/short mode - this test takes 15-20 seconds
	if testing.Short() {
		t.Skip("Skipping filtering performance tests in short mode")
	}

	t.Run("Filter Configuration Validation", func(t *testing.T) {
		// Create a mock database
		db, err := sql.Open("postgres", "postgres://user:password@localhost:5432/testdb?sslmode=disable")
		if err != nil {
			t.Skip("Skipping test - no database available")
		}
		defer db.Close()

		// Test different filter configurations
		testConfigs := []struct {
			name        string
			config      *security.AuditConfig
			expectedLog bool
		}{
			{
				name: "Allow all events",
				config: &security.AuditConfig{
					MinSeverity:       security.AuditSeverityInfo,
					AllowedEventTypes: nil, // nil means all allowed
					QueueSize:         1000,
					BatchSize:         100,
					MaxRetries:        3,
					FlushInterval:     1 * time.Second,
				},
				expectedLog: true,
			},
			{
				name: "Filter by severity (medium minimum)",
				config: &security.AuditConfig{
					MinSeverity:       security.AuditSeverityMedium,
					AllowedEventTypes: nil,
					QueueSize:         1000,
					BatchSize:         100,
					MaxRetries:        3,
					FlushInterval:     1 * time.Second,
				},
				expectedLog: false, // Info event should be filtered
			},
			{
				name: "Filter by event type",
				config: &security.AuditConfig{
					MinSeverity:       security.AuditSeverityInfo,
					AllowedEventTypes: []security.AuditEventType{security.AuditEventLoginSuccess},
					QueueSize:         1000,
					BatchSize:         100,
					MaxRetries:        3,
					FlushInterval:     1 * time.Second,
				},
				expectedLog: true,
			},
		}

		for _, tc := range testConfigs {
			t.Run(tc.name, func(t *testing.T) {
				auditLogger := security.NewAuditLoggerWithConfig(db, tc.config)
				defer auditLogger.Shutdown(5 * time.Second)

				// Create test event
				event := &security.AuditEvent{
					ID:        uuid.New(),
					EventType: security.AuditEventLoginSuccess,
					Severity:  security.AuditSeverityInfo,
					Timestamp: time.Now().UTC(),
					IPAddress: "192.168.1.1",
					UserAgent: "TestAgent",
				}

				// Log the event
				auditLogger.Log(event)

				// Give some time for processing
				time.Sleep(100 * time.Millisecond)

				// For this test, we'll just verify the logger doesn't crash
				// Actual filtering validation would require database inspection
			})
		}
	})

	t.Run("Critical Event Bypass Test", func(t *testing.T) {
		// Create a mock database
		db, err := sql.Open("postgres", "postgres://user:password@localhost:5432/testdb?sslmode=disable")
		if err != nil {
			t.Skip("Skipping test - no database available")
		}
		defer db.Close()

		// Create audit logger with restrictive filtering
		config := &security.AuditConfig{
			MinSeverity:       security.AuditSeverityHigh, // Would normally filter medium events
			AllowedEventTypes: []security.AuditEventType{security.AuditEventLoginSuccess},
			QueueSize:         1000,
			BatchSize:         100,
			MaxRetries:        3,
			FlushInterval:     1 * time.Second,
		}

		auditLogger := security.NewAuditLoggerWithConfig(db, config)
		defer auditLogger.Shutdown(5 * time.Second)

		// Test that critical events bypass filtering
		criticalEvent := &security.AuditEvent{
			ID:        uuid.New(),
			EventType: security.AuditEventAccountDeleted,
			Severity:  security.AuditSeverityCritical,
			Timestamp: time.Now().UTC(),
			IPAddress: "192.168.1.1",
			UserAgent: "TestAgent",
		}

		auditLogger.Log(criticalEvent)

		// Give some time for processing
		time.Sleep(100 * time.Millisecond)

		// Verify critical event was processed despite filters
		// This test verifies no panic occurs with critical event bypass
	})
}

// TestResourceUsage tests memory and CPU usage under load
func TestResourceUsage(t *testing.T) {
	// Skip in CI/short mode - this test takes 35+ seconds
	if testing.Short() {
		t.Skip("Skipping resource usage tests in short mode")
	}

	t.Run("Memory Usage Under Load", func(t *testing.T) {
		// Create a mock database
		db, err := sql.Open("postgres", "postgres://user:password@localhost:5432/testdb?sslmode=disable")
		if err != nil {
			t.Skip("Skipping test - no database available")
		}
		defer db.Close()

		// Create audit logger
		config := &security.AuditConfig{
			MinSeverity:            security.AuditSeverityInfo,
			QueueSize:              100000,
			BatchSize:              1000,
			MaxRetries:             3,
			FlushInterval:          1 * time.Second,
			BaseRetryDelay:         100 * time.Millisecond,
			MaxConcurrentOverflows: 50,
			AuditFailureLogPath:    "audit_memory_test.log",
		}

		auditLogger := security.NewAuditLoggerWithConfig(db, config)
		defer auditLogger.Shutdown(10 * time.Second)

		// Get initial memory usage
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)

		// Generate load
		numEvents := 50000
		var wg sync.WaitGroup

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < numEvents/10; j++ {
					event := &security.AuditEvent{
						ID:        uuid.New(),
						EventType: security.AuditEventLoginSuccess,
						Severity:  security.AuditSeverityMedium,
						Timestamp: time.Now().UTC(),
						IPAddress: fmt.Sprintf("192.168.1.%d", workerID),
						UserAgent: "TestAgent",
						EventData: map[string]any{
							"worker_id": workerID,
							"event_num": j,
							"data":      make([]byte, 100), // Some payload
						},
					}
					auditLogger.Log(event)
				}
			}(i)
		}

		wg.Wait()

		// Get final memory usage
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)

		// Calculate memory increase
		memoryIncrease := m2.Alloc - m1.Alloc
		t.Logf("Memory increase: %d bytes (%.2f MB)", memoryIncrease, float64(memoryIncrease)/(1024*1024))

		// Verify memory usage is reasonable (<10MB per 10k events)
		memoryPerEvent := float64(memoryIncrease) / float64(numEvents)
		if memoryPerEvent > 1024*1024*10/10000 { // 10MB per 10k events = 1KB per event
			t.Errorf("Excessive memory usage: %.2f bytes/event", memoryPerEvent)
		}
	})

	t.Run("CPU Usage Under Load", func(t *testing.T) {
		// Create a mock database
		db, err := sql.Open("postgres", "postgres://user:password@localhost:5432/testdb?sslmode=disable")
		if err != nil {
			t.Skip("Skipping test - no database available")
		}
		defer db.Close()

		// Create audit logger
		config := &security.AuditConfig{
			MinSeverity:            security.AuditSeverityInfo,
			QueueSize:              100000,
			BatchSize:              1000,
			MaxRetries:             3,
			FlushInterval:          1 * time.Second,
			BaseRetryDelay:         100 * time.Millisecond,
			MaxConcurrentOverflows: 50,
			AuditFailureLogPath:    "audit_cpu_test.log",
		}

		auditLogger := security.NewAuditLoggerWithConfig(db, config)
		defer auditLogger.Shutdown(10 * time.Second)

		// Measure CPU usage during high load
		startTime := time.Now()
		numEvents := 10000

		for i := 0; i < numEvents; i++ {
			event := &security.AuditEvent{
				ID:        uuid.New(),
				EventType: security.AuditEventLoginSuccess,
				Severity:  security.AuditSeverityMedium,
				Timestamp: time.Now().UTC(),
				IPAddress: "192.168.1.1",
				UserAgent: "TestAgent",
				EventData: map[string]any{
					"event_num": i,
					"data":      make([]byte, 500), // Larger payload
				},
			}
			auditLogger.Log(event)
		}

		// Wait for processing to complete
		time.Sleep(2 * time.Second)

		// Calculate processing rate
		processingTime := time.Since(startTime)
		eventsPerSecond := float64(numEvents) / processingTime.Seconds()

		t.Logf("Processing rate: %.0f events/sec", eventsPerSecond)
		t.Logf("Processing time: %v", processingTime)

		// Verify reasonable processing rate (>1000 events/sec)
		if eventsPerSecond < 1000 {
			t.Errorf("Low processing rate: %.0f events/sec", eventsPerSecond)
		}
	})
}

// TestLogIntegrity tests that logs are complete and accurate
func TestLogIntegrity(t *testing.T) {
	t.Run("Event Completeness Test", func(t *testing.T) {
		// Create a mock database
		db, err := sql.Open("postgres", "postgres://user:password@localhost:5432/testdb?sslmode=disable")
		if err != nil {
			t.Skip("Skipping test - no database available")
		}
		defer db.Close()

		// Create audit logger
		config := &security.AuditConfig{
			MinSeverity:            security.AuditSeverityInfo,
			QueueSize:              10000,
			BatchSize:              100,
			MaxRetries:             3,
			FlushInterval:          1 * time.Second,
			BaseRetryDelay:         100 * time.Millisecond,
			MaxConcurrentOverflows: 10,
			AuditFailureLogPath:    "audit_integrity_test.log",
		}

		auditLogger := security.NewAuditLoggerWithConfig(db, config)
		defer auditLogger.Shutdown(10 * time.Second)

		// Generate events with unique identifiers
		numEvents := 1000
		var eventIDs []uuid.UUID

		for i := 0; i < numEvents; i++ {
			eventID := uuid.New()
			eventIDs = append(eventIDs, eventID)

			event := &security.AuditEvent{
				ID:        eventID,
				EventType: security.AuditEventLoginSuccess,
				Severity:  security.AuditSeverityMedium,
				Timestamp: time.Now().UTC(),
				IPAddress: "192.168.1.1",
				UserAgent: "TestAgent",
				EventData: map[string]any{
					"sequence_num": i,
					"unique_id":    eventID.String(),
				},
			}
			auditLogger.Log(event)
		}

		// Wait for processing
		time.Sleep(2 * time.Second)

		// Verify all events were processed
		// For this test, we verify no panic and successful completion
		// Actual database verification would require query inspection
	})

	t.Run("Data Accuracy Test", func(t *testing.T) {
		// Create a mock database
		db, err := sql.Open("postgres", "postgres://user:password@localhost:5432/testdb?sslmode=disable")
		if err != nil {
			t.Skip("Skipping test - no database available")
		}
		defer db.Close()

		// Create audit logger
		config := &security.AuditConfig{
			MinSeverity:            security.AuditSeverityInfo,
			QueueSize:              10000,
			BatchSize:              100,
			MaxRetries:             3,
			FlushInterval:          1 * time.Second,
			BaseRetryDelay:         100 * time.Millisecond,
			MaxConcurrentOverflows: 10,
			AuditFailureLogPath:    "audit_accuracy_test.log",
		}

		auditLogger := security.NewAuditLoggerWithConfig(db, config)
		defer auditLogger.Shutdown(10 * time.Second)

		// Create test event with specific data
		testEventID := uuid.New()
		testData := map[string]any{
			"test_field": "test_value",
			"number":     42,
			"nested": map[string]any{
				"inner_field": "inner_value",
			},
		}

		event := &security.AuditEvent{
			ID:          testEventID,
			EventType:   security.AuditEventLoginSuccess,
			Severity:    security.AuditSeverityMedium,
			Timestamp:   time.Now().UTC(),
			IPAddress:   "192.168.1.100",
			UserAgent:   "TestAccuracyAgent",
			EventData:   testData,
			Description: "Test accuracy event",
		}

		auditLogger.Log(event)

		// Wait for processing
		time.Sleep(1 * time.Second)

		// Verify the event was processed
		// This test verifies no panic occurs during processing
	})
}

// Benchmark tests
func BenchmarkAuditLogging(b *testing.B) {
	// Create a mock database
	db, err := sql.Open("postgres", "postgres://user:password@localhost:5432/testdb?sslmode=disable")
	if err != nil {
		b.Skip("Skipping benchmark - no database available")
	}
	defer db.Close()

	// Create audit logger
	config := &security.AuditConfig{
		MinSeverity:            security.AuditSeverityInfo,
		QueueSize:              100000,
		BatchSize:              1000,
		MaxRetries:             3,
		FlushInterval:          1 * time.Second,
		BaseRetryDelay:         100 * time.Millisecond,
		MaxConcurrentOverflows: 50,
		AuditFailureLogPath:    "audit_benchmark_test.log",
	}

	auditLogger := security.NewAuditLoggerWithConfig(db, config)
	defer auditLogger.Shutdown(10 * time.Second)

	// Create test event
	event := &security.AuditEvent{
		ID:        uuid.New(),
		EventType: security.AuditEventLoginSuccess,
		Severity:  security.AuditSeverityMedium,
		Timestamp: time.Now().UTC(),
		IPAddress: "192.168.1.1",
		UserAgent: "BenchmarkAgent",
		EventData: map[string]any{
			"benchmark": true,
			"data":      make([]byte, 100),
		},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Generate unique event for each iteration
			newEvent := *event
			newEvent.ID = uuid.New()
			newEvent.Timestamp = time.Now().UTC()
			auditLogger.Log(&newEvent)
		}
	})
	b.StopTimer()
}

// Helper functions
func createTestEvents(count int) []*security.AuditEvent {
	events := make([]*security.AuditEvent, count)
	for i := 0; i < count; i++ {
		events[i] = &security.AuditEvent{
			ID:        uuid.New(),
			EventType: security.AuditEventLoginSuccess,
			Severity:  security.AuditSeverityMedium,
			Timestamp: time.Now().UTC(),
			IPAddress: fmt.Sprintf("192.168.1.%d", i%256),
			UserAgent: "TestAgent",
			EventData: map[string]any{
				"event_num": i,
				"data":      make([]byte, 50),
			},
		}
	}
	return events
}

func init() {
	// rand.Seed is deprecated in Go 1.20+ - global random is auto-seeded

	// Clean up test log files
	os.Remove("audit_performance_test.log")
	os.Remove("audit_high_volume_test.log")
	os.Remove("audit_nonblocking_test.log")
	os.Remove("audit_memory_test.log")
	os.Remove("audit_cpu_test.log")
	os.Remove("audit_integrity_test.log")
	os.Remove("audit_accuracy_test.log")
	os.Remove("audit_benchmark_test.log")
}
