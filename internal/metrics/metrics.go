package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// WebSocket metrics
	WebSocketConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "messenger_websocket_connections",
			Help: "Number of active WebSocket connections",
		},
		[]string{"server_id"},
	)

	WebSocketMessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messenger_websocket_messages_total",
			Help: "Total number of WebSocket messages processed",
		},
		[]string{"server_id", "message_type", "direction"},
	)

	// Message metrics
	MessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messenger_messages_total",
			Help: "Total number of messages sent",
		},
		[]string{"type"}, // direct, group
	)

	MessageDeliveryLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "messenger_message_delivery_latency_seconds",
			Help:    "Message delivery latency in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to 16s
		},
		[]string{"delivery_type"}, // immediate, offline
	)

	// Authentication metrics
	AuthAttemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messenger_auth_attempts_total",
			Help: "Total number of authentication attempts",
		},
		[]string{"type", "result"}, // login/register, success/failure
	)

	PINAttemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messenger_pin_attempts_total",
			Help: "Total number of PIN verification attempts",
		},
		[]string{"result"}, // success, failure, locked
	)

	// API metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messenger_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "messenger_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// Pre-key metrics
	PreKeysRemaining = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "messenger_prekeys_remaining",
			Help: "Number of unused pre-keys remaining per user",
		},
		[]string{"user_id"},
	)

	PreKeysReplenished = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "messenger_prekeys_replenished_total",
			Help: "Total number of pre-key batches replenished",
		},
	)

	// Rate limiting metrics
	RateLimitHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messenger_rate_limit_hits_total",
			Help: "Total number of rate limit hits",
		},
		[]string{"endpoint", "tier"},
	)

	RateLimitRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messenger_rate_limit_requests_total",
			Help: "Total number of rate limited requests",
		},
		[]string{"endpoint", "tier", "result"}, // result: allowed, denied
	)

	AbuseDetectionEvents = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messenger_abuse_detection_events_total",
			Help: "Total number of abuse detection events",
		},
		[]string{"type", "action"}, // type: ip/user, action: penalty/strict
	)

	StrictModeActivations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messenger_strict_mode_activations_total",
			Help: "Total number of strict mode activations",
		},
		[]string{"entity_type"}, // ip, user, global
	)

	RateLimitGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "messenger_rate_limit_current_requests",
			Help: "Current number of requests in rate limit windows",
		},
		[]string{"tier", "mode"}, // tier: ip/user/endpoint/global, mode: normal/strict
	)

	// Media metrics
	MediaUploadsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messenger_media_uploads_total",
			Help: "Total number of media uploads",
		},
		[]string{"type"}, // image, video, audio, document
	)

	MediaUploadSize = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "messenger_media_upload_size_bytes",
			Help:    "Size of uploaded media files in bytes",
			Buckets: prometheus.ExponentialBuckets(1024, 4, 10), // 1KB to 1GB
		},
	)

	// Group metrics
	GroupMessagesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "messenger_group_messages_total",
			Help: "Total number of group messages sent",
		},
	)

	GroupFanOutLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "messenger_group_fanout_latency_seconds",
			Help:    "Time to fan out group message to all members",
			Buckets: prometheus.ExponentialBuckets(0.01, 2, 10), // 10ms to 10s
		},
	)

	// Inbox metrics
	OfflineMessagesQueued = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "messenger_offline_messages_queued_total",
			Help: "Total number of messages queued for offline users",
		},
	)

	OfflineMessagesDelivered = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "messenger_offline_messages_delivered_total",
			Help: "Total number of offline messages delivered on reconnect",
		},
	)

	// Cleanup metrics
	ExpiredMessagesCleanedUp = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "messenger_expired_messages_cleaned_up_total",
			Help: "Total number of expired messages cleaned up",
		},
	)

	// Audit logging metrics
	AuditQueueDepth = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "messenger_audit_queue_depth",
			Help: "Current depth of the audit logging queue",
		},
	)

	AuditOverflowEventsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "messenger_audit_overflow_events_total",
			Help: "Total number of audit events that overflowed the queue",
		},
	)

	AuditBatchWriteLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "messenger_audit_batch_write_latency_seconds",
			Help:    "Latency of audit batch writes in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to 1s
		},
	)

	AuditEventsProcessedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "messenger_audit_events_processed_total",
			Help: "Total number of audit events processed",
		},
	)

	AuditBatchSize = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "messenger_audit_batch_size",
			Help:    "Size of audit event batches written",
			Buckets: prometheus.LinearBuckets(1, 10, 20), // 1 to 200
		},
	)

	AuditDeadLetterEventsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "messenger_audit_dead_letter_events_total",
			Help: "Total number of audit events sent to dead letter queue",
		},
	)

	AuditDroppedEventsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "messenger_audit_dropped_events_total",
			Help: "Total number of audit events dropped due to system failures",
		},
	)

	AuditValidationFailuresTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messenger_audit_validation_failures_total",
			Help: "Total number of audit validation failures by type",
		},
		[]string{"validation_type"},
	)

	AuditCriticalEventBypassesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "messenger_audit_critical_event_bypasses_total",
			Help: "Total number of critical events that bypassed filtering",
		},
	)

	// Security metrics
	SecurityEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messenger_security_events_total",
			Help: "Total number of security events detected",
		},
		[]string{"event_type", "severity", "action"},
	)

	TokenBlacklistEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messenger_token_blacklist_events_total",
			Help: "Total number of token blacklist events",
		},
		[]string{"operation", "reason"},
	)

	TokenBlacklistGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "messenger_token_blacklist_current_count",
			Help: "Current number of blacklisted tokens",
		},
	)

	SSLErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messenger_ssl_errors_total",
			Help: "Total number of SSL/TLS errors",
		},
		[]string{"error_type", "tls_version"},
	)

	SSLConnectionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messenger_ssl_connections_total",
			Help: "Total number of SSL/TLS connections",
		},
		[]string{"tls_version", "cipher_suite"},
	)

	SecurityHeaderEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messenger_security_header_events_total",
			Help: "Total number of security header events",
		},
		[]string{"header_type", "action"},
	)

	RateLimitSecurityEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messenger_rate_limit_security_events_total",
			Help: "Total number of security-related rate limit events",
		},
		[]string{"endpoint", "tier", "action"},
	)

	SecurityValidationEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messenger_security_validation_events_total",
			Help: "Total number of security validation events",
		},
		[]string{"validation_type", "result"},
	)

	SecurityBypassAttemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messenger_security_bypass_attempts_total",
			Help: "Total number of security bypass attempts detected",
		},
		[]string{"bypass_type", "source"},
	)

	SecurityConfigurationChangesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messenger_security_configuration_changes_total",
			Help: "Total number of security configuration changes",
		},
		[]string{"configuration_type", "change_type"},
	)
)

// MetricsMiddleware wraps HTTP handlers with metrics
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()
		path := r.URL.Path

		HTTPRequestsTotal.WithLabelValues(r.Method, path, strconv.Itoa(wrapped.statusCode)).Inc()
		HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Handler returns the Prometheus metrics handler
func Handler() http.Handler {
	return promhttp.Handler()
}

// RecordMessageSent records a sent message metric
func RecordMessageSent(messageType string) {
	MessagesTotal.WithLabelValues(messageType).Inc()
}

// RecordDeliveryLatency records message delivery latency
func RecordDeliveryLatency(deliveryType string, latency time.Duration) {
	MessageDeliveryLatency.WithLabelValues(deliveryType).Observe(latency.Seconds())
}

// RecordAuthAttempt records an authentication attempt
func RecordAuthAttempt(authType string, success bool) {
	result := "failure"
	if success {
		result = "success"
	}
	AuthAttemptsTotal.WithLabelValues(authType, result).Inc()
}

// RecordPINAttempt records a PIN verification attempt
func RecordPINAttempt(result string) {
	PINAttemptsTotal.WithLabelValues(result).Inc()
}

// RecordMediaUpload records a media upload
func RecordMediaUpload(mediaType string, sizeBytes int64) {
	MediaUploadsTotal.WithLabelValues(mediaType).Inc()
	MediaUploadSize.Observe(float64(sizeBytes))
}

// RecordRateLimitHit records a rate limit hit
func RecordRateLimitHit(endpoint string, tier string) {
	RateLimitHits.WithLabelValues(endpoint, tier).Inc()
}

// RecordRateLimitRequest records a rate limit request
func RecordRateLimitRequest(endpoint string, tier string, result string) {
	RateLimitRequests.WithLabelValues(endpoint, tier, result).Inc()
}

// RecordAbuseDetectionEvent records an abuse detection event
func RecordAbuseDetectionEvent(entityType string, action string) {
	AbuseDetectionEvents.WithLabelValues(entityType, action).Inc()
}

// RecordStrictModeActivation records a strict mode activation
func RecordStrictModeActivation(entityType string) {
	StrictModeActivations.WithLabelValues(entityType).Inc()
}

// UpdateRateLimitGauge updates the current rate limit gauge
func UpdateRateLimitGauge(tier string, mode string, value float64) {
	RateLimitGauge.WithLabelValues(tier, mode).Set(value)
}

// Security Metrics Functions
func RecordSecurityEvent(eventType string, severity string, action string) {
	SecurityEventsTotal.WithLabelValues(eventType, severity, action).Inc()
}

func RecordTokenBlacklistEvent(operation string, reason string) {
	TokenBlacklistEventsTotal.WithLabelValues(operation, reason).Inc()
}

func UpdateTokenBlacklistCount(count int) {
	TokenBlacklistGauge.Set(float64(count))
}

func RecordSSLError(errorType string, tlsVersion string) {
	SSLErrorsTotal.WithLabelValues(errorType, tlsVersion).Inc()
}

func RecordSSLConnection(tlsVersion string, cipherSuite string) {
	SSLConnectionsTotal.WithLabelValues(tlsVersion, cipherSuite).Inc()
}

func RecordSecurityHeaderEvent(headerType string, action string) {
	SecurityHeaderEventsTotal.WithLabelValues(headerType, action).Inc()
}

func RecordRateLimitSecurityEvent(endpoint string, tier string, action string) {
	RateLimitSecurityEventsTotal.WithLabelValues(endpoint, tier, action).Inc()
}

func RecordSecurityValidationEvent(validationType string, result string) {
	SecurityValidationEventsTotal.WithLabelValues(validationType, result).Inc()
}

func RecordSecurityBypassAttempt(bypassType string, source string) {
	SecurityBypassAttemptsTotal.WithLabelValues(bypassType, source).Inc()
}

func RecordSecurityConfigurationChange(configurationType string, changeType string) {
	SecurityConfigurationChangesTotal.WithLabelValues(configurationType, changeType).Inc()
}
