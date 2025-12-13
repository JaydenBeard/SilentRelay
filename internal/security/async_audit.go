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
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jaydenbeard/messaging-app/internal/metrics"
	"github.com/lib/pq"
)

// AsyncAuditLogger provides high-performance, non-blocking audit logging
// with buffered channels, batch processing, and comprehensive error handling
type AsyncAuditLogger struct {
	db                *sql.DB
	config            *AuditConfig
	inputChan         chan *AuditEvent   // Buffered input channel for non-blocking writes
	processingChan    chan []*AuditEvent // Channel for batch processing
	shutdown          chan struct{}      // Shutdown signal
	wg                sync.WaitGroup     // Wait group for goroutine management
	bufferPool        sync.Pool          // Buffer pool for performance optimization
	deadLetterChan    chan *AuditEvent   // Dead letter queue for failed events
	failureLogger     *log.Logger        // Failure logger
	failureFile       *os.File           // Failure log file
	overflowSemaphore chan struct{}      // Semaphore to limit concurrent overflow writes
	metrics           *AsyncAuditMetrics // Performance metrics
	processingWorkers int                // Number of processing workers
	flushTicker       *time.Ticker       // Ticker for periodic flushes
	processingActive  int32              // Atomic flag for processing state
}

// AsyncAuditMetrics tracks performance metrics for async audit logging
type AsyncAuditMetrics struct {
	EventsReceived       int64
	EventsProcessed      int64
	BatchesProcessed     int64
	ProcessingLatency    int64
	QueueDepth           int64
	OverflowEvents       int64
	DeadLetterEvents     int64
	DroppedEvents        int64
	ProcessingErrors     int64
	LastBatchSize        int64
	LastProcessingTime   int64
	ConcurrentProcessors int64
}

// AsyncAuditConfig extends AuditConfig with async-specific settings
type AsyncAuditConfig struct {
	*AuditConfig
	InputBufferSize       int           `json:"input_buffer_size"`       // Size of input buffer channel
	ProcessingBufferSize  int           `json:"processing_buffer_size"`  // Size of processing buffer channel
	NumProcessingWorkers  int           `json:"num_processing_workers"`  // Number of concurrent processing workers
	MaxQueueDepth         int           `json:"max_queue_depth"`         // Maximum queue depth before backpressure
	MetricsEnabled        bool          `json:"metrics_enabled"`         // Enable performance metrics
	MetricsReportInterval time.Duration `json:"metrics_report_interval"` // Metrics reporting interval
}

// NewAsyncAuditLogger creates a new async audit logger with default settings
func NewAsyncAuditLogger(db *sql.DB) *AsyncAuditLogger {
	return NewAsyncAuditLoggerWithConfig(db, DefaultAsyncAuditConfig())
}

// NewAsyncAuditLoggerWithConfig creates a new async audit logger with custom configuration
func NewAsyncAuditLoggerWithConfig(db *sql.DB, config *AsyncAuditConfig) *AsyncAuditLogger {
	// Validate configuration first
	if err := ValidateAuditConfig(config.AuditConfig); err != nil {
		log.Printf("Invalid audit configuration: %v", err)
		// Fall back to default configuration
		config.AuditConfig = DefaultAuditConfig()
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
	failureLogger := log.New(failureFile, "[ASYNC_AUDIT_FAILURE] ", log.LstdFlags|log.LUTC)

	// Initialize metrics
	metrics := &AsyncAuditMetrics{}
	if config.MetricsEnabled {
		go reportAsyncAuditMetrics(metrics, config.MetricsReportInterval)
	}

	aal := &AsyncAuditLogger{
		db:                db,
		config:            config.AuditConfig,
		inputChan:         make(chan *AuditEvent, config.InputBufferSize),
		processingChan:    make(chan []*AuditEvent, config.ProcessingBufferSize),
		shutdown:          make(chan struct{}),
		deadLetterChan:    make(chan *AuditEvent, 1000),
		failureLogger:     failureLogger,
		failureFile:       failureFile,
		overflowSemaphore: make(chan struct{}, config.MaxConcurrentOverflows),
		metrics:           metrics,
		processingWorkers: config.NumProcessingWorkers,
		bufferPool: sync.Pool{
			New: func() any {
				return &bytes.Buffer{}
			},
		},
	}

	// Start background workers
	aal.startWorkers()

	return aal
}

// DefaultAsyncAuditConfig returns default async audit configuration
func DefaultAsyncAuditConfig() *AsyncAuditConfig {
	return &AsyncAuditConfig{
		AuditConfig:           DefaultAuditConfig(),
		InputBufferSize:       10000,
		ProcessingBufferSize:  100,
		NumProcessingWorkers:  4,
		MaxQueueDepth:         50000,
		MetricsEnabled:        true,
		MetricsReportInterval: 30 * time.Second,
	}
}

// startWorkers starts all background processing workers
func (aal *AsyncAuditLogger) startWorkers() {
	// Start input processor
	aal.wg.Add(1)
	go aal.inputProcessor()

	// Start processing workers
	for i := 0; i < aal.processingWorkers; i++ {
		aal.wg.Add(1)
		go aal.batchProcessor(i)
	}

	// Start dead letter handler
	aal.wg.Add(1)
	go aal.deadLetterHandler()

	// Start flush ticker
	aal.flushTicker = time.NewTicker(aal.config.FlushInterval)
	aal.wg.Add(1)
	go aal.periodicFlusher()

	log.Printf("[ASYNC_AUDIT_INIT] Started async audit logger with %d processing workers", aal.processingWorkers)
}

// inputProcessor handles incoming events and creates batches
func (aal *AsyncAuditLogger) inputProcessor() {
	defer aal.wg.Done()

	batch := make([]*AuditEvent, 0, aal.config.BatchSize)
	batchTimer := time.NewTimer(aal.config.FlushInterval)
	defer batchTimer.Stop()

	for {
		select {
		case event := <-aal.inputChan:
			atomic.AddInt64(&aal.metrics.EventsReceived, 1)
			atomic.AddInt64(&aal.metrics.QueueDepth, 1)

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
			if !aal.shouldLog(event) {
				atomic.AddInt64(&aal.metrics.QueueDepth, -1)
				continue
			}

			// Pre-marshal EventData for performance
			if event.EventData != nil {
				buf := aal.bufferPool.Get().(*bytes.Buffer)
				buf.Reset()
				if err := json.NewEncoder(buf).Encode(event.EventData); err == nil {
					event.PreMarshaledEventData = make([]byte, buf.Len())
					copy(event.PreMarshaledEventData, buf.Bytes())
				}
				aal.bufferPool.Put(buf)
			}

			batch = append(batch, event)

			// Send batch when full
			if len(batch) >= aal.config.BatchSize {
				aal.sendBatch(batch)
				batch = batch[:0]
				batchTimer.Reset(aal.config.FlushInterval)
			}

			atomic.AddInt64(&aal.metrics.QueueDepth, -1)

		case <-batchTimer.C:
			if len(batch) > 0 {
				aal.sendBatch(batch)
				batch = batch[:0]
			}

		case <-aal.shutdown:
			// Drain remaining events
			for {
				select {
				case event := <-aal.inputChan:
					batch = append(batch, event)
				default:
					if len(batch) > 0 {
						aal.sendBatch(batch)
					}
					return
				}
			}
		}
	}
}

// sendBatch sends a batch to processing channel with backpressure handling
func (aal *AsyncAuditLogger) sendBatch(batch []*AuditEvent) {
	select {
	case aal.processingChan <- batch:
		atomic.AddInt64(&aal.metrics.BatchesProcessed, 1)
		atomic.StoreInt64(&aal.metrics.LastBatchSize, int64(len(batch)))
	default:
		// Processing channel full, apply backpressure
		atomic.AddInt64(&aal.metrics.OverflowEvents, 1)

		// Spawn goroutine for overflow processing
		go func() {
			// Acquire semaphore slot for concurrent overflow control
			aal.overflowSemaphore <- struct{}{}
			defer func() {
				<-aal.overflowSemaphore
			}()

			// Process batch directly to avoid blocking
			if err := aal.writeBatch(batch); err != nil {
				aal.failureLogger.Printf("Failed to write overflow audit batch: %v", err)
				atomic.AddInt64(&aal.metrics.ProcessingErrors, 1)
			}
			atomic.AddInt64(&aal.metrics.EventsProcessed, int64(len(batch)))
		}()
	}
}

// batchProcessor processes batches from the processing channel
func (aal *AsyncAuditLogger) batchProcessor(workerID int) {
	defer aal.wg.Done()

	for {
		select {
		case batch := <-aal.processingChan:
			startTime := time.Now()
			atomic.AddInt64(&aal.metrics.ConcurrentProcessors, 1)

			if err := aal.writeBatch(batch); err != nil {
				aal.failureLogger.Printf("Worker %d: Failed to write audit batch: %v", workerID, err)
				atomic.AddInt64(&aal.metrics.ProcessingErrors, 1)
			} else {
				atomic.AddInt64(&aal.metrics.EventsProcessed, int64(len(batch)))
			}

			processingTime := time.Since(startTime)
			atomic.AddInt64(&aal.metrics.LastProcessingTime, int64(processingTime.Milliseconds()))
			atomic.AddInt64(&aal.metrics.ConcurrentProcessors, -1)

		case <-aal.shutdown:
			return
		}
	}
}

// periodicFlusher handles periodic flushing of events
func (aal *AsyncAuditLogger) periodicFlusher() {
	defer aal.wg.Done()

	for {
		select {
		case <-aal.flushTicker.C:
			// Trigger flush by sending empty batch if processing channel has space
			select {
			case aal.processingChan <- []*AuditEvent{}:
			default:
				// Processing channel busy, skip this flush cycle
			}

		case <-aal.shutdown:
			aal.flushTicker.Stop()
			return
		}
	}
}

// Log records an audit event asynchronously
func (aal *AsyncAuditLogger) Log(event *AuditEvent) {
	select {
	case aal.inputChan <- event:
		// Non-blocking write successful
		metrics.AuditQueueDepth.Set(float64(len(aal.inputChan)))
	default:
		// Input channel full, apply backpressure
		metrics.AuditOverflowEventsTotal.Inc()

		// Spawn goroutine for overflow processing
		go func() {
			// Acquire semaphore slot for concurrent overflow control
			aal.overflowSemaphore <- struct{}{}
			defer func() {
				<-aal.overflowSemaphore
			}()

			if err := aal.write(event); err != nil {
				aal.failureLogger.Printf("Failed to write overflow audit event: %v", err)
			}
			metrics.AuditEventsProcessedTotal.Inc()
		}()
	}
}

// shouldLog checks if an event should be logged based on configuration filters
func (aal *AsyncAuditLogger) shouldLog(event *AuditEvent) bool {
	// Create comprehensive validator for event validation
	validator := NewComprehensiveAuditValidator(&AuditLogger{
		db:     aal.db,
		config: aal.config,
	})

	// Use comprehensive validation
	err := validator.ValidateAuditEventWithContext(context.Background(), event)
	if err != nil {
		log.Printf("[ASYNC_AUDIT_EVENT_FILTERED] Event failed validation: %v", err)
		return false
	}

	// Critical events always pass validation and get logged
	if event.Severity == AuditSeverityCritical {
		log.Printf("[ASYNC_AUDIT_CRITICAL_BYPASS] Critical event bypassed filtering: EventType=%s, EventID=%s",
			event.EventType, event.ID)
		return true
	}

	return true
}

// writeBatch writes a batch of events to the database with retry logic
func (aal *AsyncAuditLogger) writeBatch(events []*AuditEvent) error {
	if len(events) == 0 {
		return nil
	}

	start := time.Now()
	defer func() {
		atomic.AddInt64(&aal.metrics.ProcessingLatency, int64(time.Since(start).Milliseconds()))
	}()

	// Use retry logic for the entire batch operation
	return aal.retryDBOperation(events, func() error {
		tx, err := aal.db.Begin()
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
				aal.failureLogger.Printf("Warning: rollback failed: %v", rbErr)
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
}

// write persists a single event to the database with retry logic
func (aal *AsyncAuditLogger) write(event *AuditEvent) error {
	// Use retry logic for single event write
	return aal.retryDBOperation([]*AuditEvent{event}, func() error {
		// Use pre-marshaled event data if available to avoid redundant marshaling
		var eventData []byte
		if len(event.PreMarshaledEventData) > 0 {
			eventData = event.PreMarshaledEventData
		} else {
			eventData, _ = json.Marshal(event.EventData)
		}

		// Use pq.Array for PostgreSQL array type
		complianceFlags := pq.Array(event.ComplianceFlags)

		_, err := aal.db.Exec(`
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

// deadLetterHandler processes permanently failed audit events
func (aal *AsyncAuditLogger) deadLetterHandler() {
	defer aal.wg.Done()

	for {
		select {
		case event := <-aal.deadLetterChan:
			// Log the failed event to the failure log
			aal.failureLogger.Printf("Permanently failed audit event: ID=%s, Type=%s, UserID=%v, Error=Max retries exceeded",
				event.ID, event.EventType, event.UserID)
			atomic.AddInt64(&aal.metrics.DeadLetterEvents, 1)

		case <-aal.shutdown:
			// Drain remaining failed events
			for {
				select {
				case event := <-aal.deadLetterChan:
					aal.failureLogger.Printf("Permanently failed audit event on shutdown: ID=%s, Type=%s, UserID=%v",
						event.ID, event.EventType, event.UserID)
					atomic.AddInt64(&aal.metrics.DeadLetterEvents, 1)
				default:
					return
				}
			}
		}
	}
}

// retryDBOperation retries a database operation with exponential backoff and comprehensive error handling
func (aal *AsyncAuditLogger) retryDBOperation(events []*AuditEvent, operation func() error) error {
	var lastErr error
	delay := aal.config.BaseRetryDelay

	// Create comprehensive validator for error handling
	validator := NewComprehensiveAuditValidator(&AuditLogger{
		db:     aal.db,
		config: aal.config,
	})

	for attempt := 0; attempt <= aal.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(delay)
			delay *= 2 // exponential backoff
		}

		err := operation()
		if err == nil {
			// Log successful operation after retries
			if attempt > 0 {
				log.Printf("[ASYNC_AUDIT_RETRY_SUCCESS] Operation succeeded after %d retries", attempt)
			}
			return nil
		}

		lastErr = err

		// Comprehensive error classification and handling
		errorType := classifyDatabaseError(err)
		log.Printf("[ASYNC_AUDIT_DB_ERROR] %s error (attempt %d/%d): %v",
			errorType, attempt+1, aal.config.MaxRetries+1, err)

		// Log all retry attempts with comprehensive details
		for _, event := range events {
			aal.failureLogger.Printf("Audit DB operation failed (%s, attempt %d/%d): %v, EventID=%s, EventType=%s",
				errorType, attempt+1, aal.config.MaxRetries+1, err, event.ID, event.EventType)

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
			log.Printf("[ASYNC_AUDIT_CRITICAL_ERROR] Critical database error detected, failing fast: %v", err)
			break
		}
	}

	// All retries exhausted, send failed events to dead letter queue with comprehensive handling
	for _, event := range events {
		// Validate event before sending to dead letter queue
		if err := validator.ValidateAuditEventBeforeLogging(event); err != nil {
			aal.failureLogger.Printf("Failed event validation before dead letter queue: ID=%s, Error=%v", event.ID, err)
			continue
		}

		select {
		case aal.deadLetterChan <- event:
			aal.failureLogger.Printf("Sent failed event to dead letter queue: ID=%s, Type=%s, Error=%v",
				event.ID, event.EventType, lastErr)
			atomic.AddInt64(&aal.metrics.DeadLetterEvents, 1)
		default:
			// Dead letter queue full, log to failure logger with comprehensive details
			aal.failureLogger.Printf("Dead letter queue full, dropping failed event: ID=%s, Type=%s, Error=%v",
				event.ID, event.EventType, lastErr)
			atomic.AddInt64(&aal.metrics.DroppedEvents, 1)

			// If this is a critical event, we need to ensure it's not lost
			if event.Severity == AuditSeverityCritical {
				// Write critical events to emergency log file
				aal.writeCriticalEventToEmergencyLog(event, lastErr)
			}
		}
	}

	return lastErr
}

// Shutdown gracefully shuts down the async audit logger
func (aal *AsyncAuditLogger) Shutdown(timeout time.Duration) error {
	// Step 1: Signal shutdown to processing goroutines
	close(aal.shutdown)

	// Step 2: Wait for all pending events to be processed with timeout
	done := make(chan struct{})
	go func() {
		// Wait for all workers to complete
		aal.wg.Wait()

		// Close the failure log file
		if aal.failureFile != nil && aal.failureFile != os.Stderr {
			if err := aal.failureFile.Close(); err != nil {
				log.Printf("Warning: failed to close failure file: %v", err)
			}
		}
		close(done)
	}()

	// Step 3: Handle timeout to prevent indefinite blocking
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		// Timeout occurred, but we still need to ensure resources are cleaned up
		// The goroutines should have completed their work, but we'll force cleanup
		if aal.failureFile != nil && aal.failureFile != os.Stderr {
			if err := aal.failureFile.Close(); err != nil {
				log.Printf("Warning: failed to close failure file: %v", err)
			}
		}
		return fmt.Errorf("async audit logger shutdown timed out after %v", timeout)
	}
}

// reportAsyncAuditMetrics reports performance metrics periodically
func reportAsyncAuditMetrics(metrics *AsyncAuditMetrics, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		eventsReceived := atomic.LoadInt64(&metrics.EventsReceived)
		eventsProcessed := atomic.LoadInt64(&metrics.EventsProcessed)
		batchesProcessed := atomic.LoadInt64(&metrics.BatchesProcessed)
		processingLatency := atomic.LoadInt64(&metrics.ProcessingLatency)
		queueDepth := atomic.LoadInt64(&metrics.QueueDepth)
		overflowEvents := atomic.LoadInt64(&metrics.OverflowEvents)
		deadLetterEvents := atomic.LoadInt64(&metrics.DeadLetterEvents)
		droppedEvents := atomic.LoadInt64(&metrics.DroppedEvents)
		processingErrors := atomic.LoadInt64(&metrics.ProcessingErrors)
		lastBatchSize := atomic.LoadInt64(&metrics.LastBatchSize)
		lastProcessingTime := atomic.LoadInt64(&metrics.LastProcessingTime)
		concurrentProcessors := atomic.LoadInt64(&metrics.ConcurrentProcessors)

		log.Printf("[ASYNC_AUDIT_METRICS] EventsReceived=%d, EventsProcessed=%d, BatchesProcessed=%d, "+
			"ProcessingLatency=%dms, QueueDepth=%d, OverflowEvents=%d, DeadLetterEvents=%d, "+
			"DroppedEvents=%d, ProcessingErrors=%d, LastBatchSize=%d, LastProcessingTime=%dms, "+
			"ConcurrentProcessors=%d",
			eventsReceived, eventsProcessed, batchesProcessed, processingLatency,
			queueDepth, overflowEvents, deadLetterEvents, droppedEvents,
			processingErrors, lastBatchSize, lastProcessingTime, concurrentProcessors)
	}
}

// GetMetrics returns current performance metrics
func (aal *AsyncAuditLogger) GetMetrics() *AsyncAuditMetrics {
	return &AsyncAuditMetrics{
		EventsReceived:       atomic.LoadInt64(&aal.metrics.EventsReceived),
		EventsProcessed:      atomic.LoadInt64(&aal.metrics.EventsProcessed),
		BatchesProcessed:     atomic.LoadInt64(&aal.metrics.BatchesProcessed),
		ProcessingLatency:    atomic.LoadInt64(&aal.metrics.ProcessingLatency),
		QueueDepth:           atomic.LoadInt64(&aal.metrics.QueueDepth),
		OverflowEvents:       atomic.LoadInt64(&aal.metrics.OverflowEvents),
		DeadLetterEvents:     atomic.LoadInt64(&aal.metrics.DeadLetterEvents),
		DroppedEvents:        atomic.LoadInt64(&aal.metrics.DroppedEvents),
		ProcessingErrors:     atomic.LoadInt64(&aal.metrics.ProcessingErrors),
		LastBatchSize:        atomic.LoadInt64(&aal.metrics.LastBatchSize),
		LastProcessingTime:   atomic.LoadInt64(&aal.metrics.LastProcessingTime),
		ConcurrentProcessors: atomic.LoadInt64(&aal.metrics.ConcurrentProcessors),
	}
}

// LogFromRequest creates and logs an event from HTTP request asynchronously
func (aal *AsyncAuditLogger) LogFromRequest(r *http.Request, userID *uuid.UUID, eventType AuditEventType, data map[string]any) {
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
	aal.Log(event)
}

// LogSecurityEvent logs a security-related event with appropriate severity asynchronously
func (aal *AsyncAuditLogger) LogSecurityEvent(ctx context.Context, eventType AuditEventType, result AuditResult, userID *uuid.UUID, description string, data map[string]any) {
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
	aal.Log(event)
}

// LogAdminAction logs an administrative action for compliance asynchronously
func (aal *AsyncAuditLogger) LogAdminAction(adminID uuid.UUID, action string, resource string, resourceID string, data map[string]any) {
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
	aal.Log(event)
}

// LogDataAccess logs data access for compliance (GDPR, HIPAA) asynchronously
func (aal *AsyncAuditLogger) LogDataAccess(userID uuid.UUID, resource string, resourceID string, dataCategory string, data map[string]any) {
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
	aal.Log(event)
}

// writeCriticalEventToEmergencyLog writes critical events to emergency log file
func (aal *AsyncAuditLogger) writeCriticalEventToEmergencyLog(event *AuditEvent, err error) {
	emergencyLogFile := "/tmp/audit_emergency_critical.log"
	file, fileErr := os.OpenFile(emergencyLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if fileErr != nil {
		aal.failureLogger.Printf("Failed to open emergency log file: %v", fileErr)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			aal.failureLogger.Printf("Warning: failed to close file: %v", err)
		}
	}()

	emergencyData := map[string]interface{}{
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
		"event_id":       event.ID,
		"event_type":     event.EventType,
		"severity":       event.Severity,
		"user_id":        event.UserID,
		"error":          err.Error(),
		"retry_attempts": event.EventData["audit_retry_attempt"],
	}

	logEntry, err := json.Marshal(emergencyData)
	if err != nil {
		aal.failureLogger.Printf("Failed to marshal emergency log entry: %v", err)
		return
	}
	logEntry = append(logEntry, '\n')

	if _, writeErr := file.Write(logEntry); writeErr != nil {
		aal.failureLogger.Printf("Failed to write to emergency log: %v", writeErr)
	}
}
