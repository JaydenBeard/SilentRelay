package security

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jaydenbeard/messaging-app/internal/metrics"
)

// ComprehensiveAuditValidator provides comprehensive validation and error handling for audit logging
type ComprehensiveAuditValidator struct {
	auditLogger       *AuditLogger
	errorLogger       *log.Logger
	validationMetrics *ComprehensiveValidationMetrics
}

// ComprehensiveValidationMetrics tracks validation performance and failures
type ComprehensiveValidationMetrics struct {
	TotalValidations         int
	ValidationFailures       int
	CriticalEventBypasses    int
	ConfigurationValidations int
	EventValidations         int
	DatabaseValidations      int
	PathValidations          int
	SeverityValidations      int
	EventTypeValidations     int
}

// NewComprehensiveAuditValidator creates a new comprehensive validator
func NewComprehensiveAuditValidator(auditLogger *AuditLogger) *ComprehensiveAuditValidator {
	return &ComprehensiveAuditValidator{
		auditLogger:       auditLogger,
		errorLogger:       log.New(os.Stderr, "[AUDIT_VALIDATION] ", log.LstdFlags|log.LUTC),
		validationMetrics: &ComprehensiveValidationMetrics{},
	}
}

// ValidateAuditEventWithContext validates an audit event with comprehensive checks
func (v *ComprehensiveAuditValidator) ValidateAuditEventWithContext(ctx context.Context, event *AuditEvent) error {
	v.validationMetrics.TotalValidations++
	v.validationMetrics.EventValidations++

	// Log validation start
	log.Printf("[AUDIT_EVENT_VALIDATION] Starting comprehensive validation for event: ID=%s, Type=%s", event.ID, event.EventType)

	// 1. Validate event ID with edge case handling
	if err := v.validateEventIDWithEdgeCases(event); err != nil {
		v.logValidationFailure("event_id_validation", event, err)
		return err
	}

	// 2. Validate event type with edge case handling
	if err := v.validateEventTypeWithEdgeCases(event); err != nil {
		v.logValidationFailure("event_type_validation", event, err)
		return err
	}

	// 3. Validate severity with edge case handling
	if err := v.validateSeverityWithEdgeCases(event); err != nil {
		v.logValidationFailure("severity_validation", event, err)
		return err
	}

	// 4. Validate timestamp with edge case handling
	if err := v.validateTimestampWithEdgeCases(event); err != nil {
		v.logValidationFailure("timestamp_validation", event, err)
		return err
	}

	// 5. Validate critical event bypass logic with edge case handling
	if err := v.validateCriticalEventBypassWithEdgeCases(event); err != nil {
		v.logValidationFailure("critical_bypass_validation", event, err)
		return err
	}

	// 6. Validate against audit configuration with edge case handling
	if err := v.validateEventAgainstConfigWithEdgeCases(event); err != nil {
		v.logValidationFailure("config_compliance_validation", event, err)
		return err
	}

	// 7. Validate event data size limits with edge case handling
	if err := v.validateEventDataSizeWithEdgeCases(event); err != nil {
		v.logValidationFailure("event_data_size_validation", event, err)
		return err
	}

	// 8. Validate IP address format with edge case handling
	if err := v.validateIPAddressWithEdgeCases(event.IPAddress); err != nil {
		v.logValidationFailure("ip_address_validation", event, err)
		return err
	}

	// 9. Validate user agent length with edge case handling
	if err := v.validateUserAgentWithEdgeCases(event.UserAgent); err != nil {
		v.logValidationFailure("user_agent_validation", event, err)
		return err
	}

	// 10. Validate resource fields with edge case handling
	if err := v.validateResourceFieldsWithEdgeCases(event); err != nil {
		v.logValidationFailure("resource_fields_validation", event, err)
		return err
	}

	// 11. Validate compliance metadata with edge case handling
	if err := v.validateComplianceMetadataWithEdgeCases(event); err != nil {
		v.logValidationFailure("compliance_metadata_validation", event, err)
		return err
	}

	log.Printf("[AUDIT_EVENT_VALIDATION] Event validation successful: ID=%s, Type=%s", event.ID, event.EventType)
	return nil
}

// validateEventIDWithEdgeCases validates event ID with comprehensive edge case handling
func (v *ComprehensiveAuditValidator) validateEventIDWithEdgeCases(event *AuditEvent) error {
	if event.ID == uuid.Nil {
		return fmt.Errorf("audit event ID cannot be nil")
	}

	// Check for version information in UUID
	uuidVersion := event.ID.Version()
	if uuidVersion != 4 { // UUID v4 is expected
		log.Printf("[AUDIT_EVENT_WARNING] Non-standard UUID version detected: %d for event ID: %s", uuidVersion, event.ID)
	}

	return nil
}

// validateEventTypeWithEdgeCases validates event type with comprehensive edge case handling
func (v *ComprehensiveAuditValidator) validateEventTypeWithEdgeCases(event *AuditEvent) error {
	if event.EventType == "" {
		return fmt.Errorf("audit event type cannot be empty")
	}

	// Check for extremely long event type names
	if len(string(event.EventType)) > 100 {
		return fmt.Errorf("event type name too long: %d characters (max: 100)", len(string(event.EventType)))
	}

	// Check for invalid characters in event type
	if strings.ContainsAny(string(event.EventType), " \t\n\r<>\"'\\") {
		return fmt.Errorf("event type contains invalid characters")
	}

	// Check if event type is one of the known critical types that should never be filtered
	if isKnownCriticalEventType(event.EventType) {
		// This is informational - critical events will bypass filtering anyway
		log.Printf("[AUDIT_EVENT_INFO] Known critical event type detected: %s", event.EventType)
	}

	return nil
}

// validateSeverityWithEdgeCases validates severity with comprehensive edge case handling
func (v *ComprehensiveAuditValidator) validateSeverityWithEdgeCases(event *AuditEvent) error {
	if event.Severity == "" {
		return fmt.Errorf("audit event severity cannot be empty")
	}

	// Validate that severity is one of the known values
	knownSeverities := []AuditSeverity{
		AuditSeverityCritical,
		AuditSeverityHigh,
		AuditSeverityMedium,
		AuditSeverityLow,
		AuditSeverityInfo,
	}

	validSeverity := false
	for _, knownSeverity := range knownSeverities {
		if event.Severity == knownSeverity {
			validSeverity = true
			break
		}
	}

	if !validSeverity {
		return fmt.Errorf("unknown severity level: %s", event.Severity)
	}

	// Check for severity mismatch with event type
	expectedSeverity := getSeverityForEventType(event.EventType)
	if event.Severity != expectedSeverity && event.Severity != AuditSeverityCritical {
		log.Printf("[AUDIT_EVENT_WARNING] Severity mismatch: EventType=%s expects %s but got %s",
			event.EventType, expectedSeverity, event.Severity)
	}

	return nil
}

// validateTimestampWithEdgeCases validates timestamp with comprehensive edge case handling
func (v *ComprehensiveAuditValidator) validateTimestampWithEdgeCases(event *AuditEvent) error {
	if event.Timestamp.IsZero() {
		return fmt.Errorf("audit event timestamp cannot be zero")
	}

	// Check for future timestamps (allow small clock skew)
	now := time.Now().UTC()
	if event.Timestamp.After(now.Add(5 * time.Minute)) {
		return fmt.Errorf("event timestamp is too far in the future: %v (current: %v)",
			event.Timestamp, now)
	}

	// Check for extremely old timestamps
	if event.Timestamp.Before(now.AddDate(-1, 0, 0)) { // Older than 1 year
		log.Printf("[AUDIT_EVENT_WARNING] Very old event timestamp: %v (current: %v)",
			event.Timestamp, now)
	}

	// Check for reasonable time precision
	if event.Timestamp.Nanosecond() == 0 {
		log.Printf("[AUDIT_EVENT_WARNING] Event timestamp has no nanosecond precision")
	}

	return nil
}

// validateCriticalEventBypassWithEdgeCases validates critical event bypass logic
func (v *ComprehensiveAuditValidator) validateCriticalEventBypassWithEdgeCases(event *AuditEvent) error {
	if event.Severity == AuditSeverityCritical {
		v.validationMetrics.CriticalEventBypasses++
		metrics.AuditCriticalEventBypassesTotal.Inc()

		// Log comprehensive bypass details
		v.logCriticalEventBypass(event)

		// Critical events always bypass filtering for compliance
		return nil
	}

	// For non-critical events, ensure they don't incorrectly claim to be critical
	if strings.Contains(string(event.EventType), "critical") ||
		strings.Contains(string(event.EventType), "emergency") {
		log.Printf("[AUDIT_EVENT_WARNING] Non-critical event has critical-sounding name: %s", event.EventType)
	}

	return nil
}

// validateEventAgainstConfigWithEdgeCases validates event against audit configuration with edge cases
func (v *ComprehensiveAuditValidator) validateEventAgainstConfigWithEdgeCases(event *AuditEvent) error {
	// If no audit logger is configured, skip config validation
	if v.auditLogger == nil || v.auditLogger.config == nil {
		return nil
	}

	// Check minimum severity with edge case handling
	if getSeverityLevel(event.Severity) < getSeverityLevel(v.auditLogger.config.MinSeverity) {
		// For critical events, this should never happen due to bypass logic
		if event.Severity == AuditSeverityCritical {
			return fmt.Errorf("critical event incorrectly filtered by severity configuration")
		}
		return fmt.Errorf("event severity %s is below minimum configured severity %s",
			event.Severity, v.auditLogger.config.MinSeverity)
	}

	// Check allowed event types (nil means all allowed) with edge case handling
	if v.auditLogger.config.AllowedEventTypes != nil {
		// Check for empty allowed event types list (should be nil, not empty slice)
		if len(v.auditLogger.config.AllowedEventTypes) == 0 {
			return fmt.Errorf("allowed event types list is empty (should be nil for 'all allowed')")
		}

		allowed := false
		for _, allowedType := range v.auditLogger.config.AllowedEventTypes {
			if event.EventType == allowedType {
				allowed = true
				break
			}
		}
		if !allowed {
			// Special handling for critical events that should never be filtered
			if isKnownCriticalEventType(event.EventType) {
				return fmt.Errorf("critical event type %s incorrectly excluded from allowed event types", event.EventType)
			}
			return fmt.Errorf("event type %s is not in allowed event types", event.EventType)
		}
	}

	return nil
}

// validateEventDataSizeWithEdgeCases validates event data doesn't exceed reasonable limits with edge cases
func (v *ComprehensiveAuditValidator) validateEventDataSizeWithEdgeCases(event *AuditEvent) error {
	// Check EventData size with edge case handling
	if event.EventData != nil {
		dataSize := estimateMapSize(event.EventData)
		if dataSize > 10000 { // 10KB limit
			return fmt.Errorf("event data exceeds maximum size limit of 10KB (actual: %d bytes)", dataSize)
		}

		// Check for deeply nested structures that could cause performance issues
		if isDeeplyNested(event.EventData, 5) {
			log.Printf("[AUDIT_EVENT_WARNING] Deeply nested event data detected for EventID: %s", event.ID)
		}
	}

	// Check PreMarshaledEventData size with edge case handling
	if len(event.PreMarshaledEventData) > 10000 { // 10KB limit
		return fmt.Errorf("pre-marshaled event data exceeds maximum size limit of 10KB")
	}

	// Check Description length with edge case handling
	if len(event.Description) > 4096 { // 4KB limit
		return fmt.Errorf("event description exceeds maximum length of 4096 characters")
	}

	// Check for unreasonable description content
	if containsSuspiciousContent(event.Description) {
		log.Printf("[AUDIT_EVENT_WARNING] Suspicious content detected in event description for EventID: %s", event.ID)
	}

	return nil
}

// validateIPAddressWithEdgeCases validates IP address format with edge cases
func (v *ComprehensiveAuditValidator) validateIPAddressWithEdgeCases(ipAddress string) error {
	if ipAddress == "" {
		return nil // IP address can be empty
	}

	// Basic IP validation - check for reasonable format
	if len(ipAddress) > 45 { // IPv6 max length
		return fmt.Errorf("IP address too long: %d characters", len(ipAddress))
	}

	// Check for invalid characters in path
	if strings.ContainsAny(ipAddress, " \t\n\r<>\"'\\") {
		return fmt.Errorf("IP address contains invalid characters")
	}

	// Check for common placeholder/private IP patterns that might indicate testing
	if isPlaceholderIP(ipAddress) {
		log.Printf("[AUDIT_EVENT_WARNING] Placeholder/test IP address detected: %s", ipAddress)
	}

	// Check for localhost/loopback addresses
	if isLoopbackIP(ipAddress) {
		log.Printf("[AUDIT_EVENT_INFO] Loopback IP address detected: %s", ipAddress)
	}

	return nil
}

// validateUserAgentWithEdgeCases validates user agent length with edge cases
func (v *ComprehensiveAuditValidator) validateUserAgentWithEdgeCases(userAgent string) error {
	if len(userAgent) > 1024 { // 1KB limit
		return fmt.Errorf("user agent exceeds maximum length of 1024 characters")
	}

	// Check for empty user agent (could indicate bot or script)
	if userAgent == "" {
		log.Printf("[AUDIT_EVENT_INFO] Empty user agent detected")
	}

	// Check for suspicious user agent patterns
	if containsSuspiciousUserAgentPatterns(userAgent) {
		log.Printf("[AUDIT_EVENT_WARNING] Suspicious user agent pattern detected: %s", userAgent)
	}

	return nil
}

// validateResourceFieldsWithEdgeCases validates resource fields with edge cases
func (v *ComprehensiveAuditValidator) validateResourceFieldsWithEdgeCases(event *AuditEvent) error {
	// Validate resource field lengths
	if len(event.Resource) > 255 {
		return fmt.Errorf("resource name exceeds maximum length of 255 characters")
	}

	if len(event.ResourceID) > 255 {
		return fmt.Errorf("resource ID exceeds maximum length of 255 characters")
	}

	if len(event.ResourceType) > 100 {
		return fmt.Errorf("resource type exceeds maximum length of 100 characters")
	}

	// Check for suspicious resource patterns
	if containsSuspiciousResourcePatterns(event.Resource) {
		log.Printf("[AUDIT_EVENT_WARNING] Suspicious resource pattern detected: %s", event.Resource)
	}

	return nil
}

// validateComplianceMetadataWithEdgeCases validates compliance metadata with edge cases
func (v *ComprehensiveAuditValidator) validateComplianceMetadataWithEdgeCases(event *AuditEvent) error {
	// Validate compliance flags
	for _, flag := range event.ComplianceFlags {
		if len(flag) > 50 {
			return fmt.Errorf("compliance flag exceeds maximum length of 50 characters: %s", flag)
		}

		if strings.ContainsAny(flag, " \t\n\r<>\"'\\") {
			return fmt.Errorf("compliance flag contains invalid characters: %s", flag)
		}
	}

	// Validate data category
	if event.DataCategory != "" {
		if len(event.DataCategory) > 50 {
			return fmt.Errorf("data category exceeds maximum length of 50 characters")
		}

		// Check for valid data categories
		validCategories := []string{"PII", "PHI", "financial", "personal", "sensitive", "public"}
		validCategory := false
		for _, category := range validCategories {
			if strings.EqualFold(event.DataCategory, category) {
				validCategory = true
				break
			}
		}

		if !validCategory {
			log.Printf("[AUDIT_EVENT_WARNING] Unknown data category: %s", event.DataCategory)
		}
	}

	// Validate retention days
	if event.RetentionDays < 0 {
		return fmt.Errorf("retention days cannot be negative")
	}

	if event.RetentionDays > 3650 { // ~10 years max
		log.Printf("[AUDIT_EVENT_WARNING] Very long retention period: %d days", event.RetentionDays)
	}

	return nil
}

// Helper functions for edge case detection

// isKnownCriticalEventType checks if an event type is in the critical list
func isKnownCriticalEventType(eventType AuditEventType) bool {
	criticalTypes := getCriticalEventTypes()
	for _, criticalType := range criticalTypes {
		if eventType == criticalType {
			return true
		}
	}
	return false
}

// estimateMapSize estimates the size of a map in bytes
func estimateMapSize(data map[string]any) int {
	size := 0
	for key, value := range data {
		size += len(key)
		switch v := value.(type) {
		case string:
			size += len(v)
		case []byte:
			size += len(v)
		case map[string]any:
			size += estimateMapSize(v)
		case []any:
			for _, item := range v {
				if itemMap, ok := item.(map[string]any); ok {
					size += estimateMapSize(itemMap)
				} else if itemStr, ok := item.(string); ok {
					size += len(itemStr)
				}
			}
		}
	}
	return size
}

// isDeeplyNested checks if a map is deeply nested
func isDeeplyNested(data map[string]any, maxDepth int) bool {
	if maxDepth <= 0 {
		return true
	}

	for _, value := range data {
		switch v := value.(type) {
		case map[string]any:
			if isDeeplyNested(v, maxDepth-1) {
				return true
			}
		case []any:
			for _, item := range v {
				if itemMap, ok := item.(map[string]any); ok {
					if isDeeplyNested(itemMap, maxDepth-1) {
						return true
					}
				}
			}
		}
	}
	return false
}

// containsSuspiciousContent checks for suspicious content patterns
func containsSuspiciousContent(text string) bool {
	suspiciousPatterns := []string{
		"eval(",
		"script>",
		"javascript:",
		"onerror=",
		"onload=",
		"document.cookie",
		"window.location",
		"<?php",
		"<%=",
		"1=1",
		"OR 1=1",
		"UNION SELECT",
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(strings.ToLower(text), pattern) {
			return true
		}
	}
	return false
}

// isPlaceholderIP checks for placeholder/test IP addresses
func isPlaceholderIP(ip string) bool {
	placeholderIPs := []string{
		"0.0.0.0",
		"127.0.0.1",
		"localhost",
		"::1",
		"192.0.2.",    // TEST-NET-1
		"198.51.100.", // TEST-NET-2
		"203.0.113.",  // TEST-NET-3
	}

	for _, placeholder := range placeholderIPs {
		if strings.HasPrefix(ip, placeholder) {
			return true
		}
	}
	return false
}

// isLoopbackIP checks for loopback IP addresses
func isLoopbackIP(ip string) bool {
	return ip == "127.0.0.1" || ip == "::1" || strings.HasPrefix(ip, "127.")
}

// containsSuspiciousUserAgentPatterns checks for suspicious user agent patterns
func containsSuspiciousUserAgentPatterns(userAgent string) bool {
	if userAgent == "" {
		return false
	}

	suspiciousPatterns := []string{
		"sqlmap",
		"nikto",
		"dirb",
		"wget",
		"curl",
		"python-requests",
		"java/",
		"go-http-client",
		"scrapy",
		"bot",
		"spider",
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(strings.ToLower(userAgent), pattern) {
			return true
		}
	}

	return false
}

// containsSuspiciousResourcePatterns checks for suspicious resource patterns
func containsSuspiciousResourcePatterns(resource string) bool {
	suspiciousPatterns := []string{
		"../",
		"../../",
		"../",
		"..\\",
		"passwd",
		"shadow",
		".env",
		"config",
		"admin",
		"backup",
		"dump",
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(strings.ToLower(resource), pattern) {
			return true
		}
	}

	return false
}

// validateEventAgainstConfig validates event against audit configuration
func (v *ComprehensiveAuditValidator) validateEventAgainstConfig(event *AuditEvent) error {
	// If no audit logger is configured, skip config validation
	if v.auditLogger == nil || v.auditLogger.config == nil {
		return nil
	}

	// Check minimum severity
	if getSeverityLevel(event.Severity) < getSeverityLevel(v.auditLogger.config.MinSeverity) {
		return fmt.Errorf("event severity %s is below minimum configured severity %s",
			event.Severity, v.auditLogger.config.MinSeverity)
	}

	// Check allowed event types (nil means all allowed)
	if v.auditLogger.config.AllowedEventTypes != nil {
		allowed := false
		for _, allowedType := range v.auditLogger.config.AllowedEventTypes {
			if event.EventType == allowedType {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("event type %s is not in allowed event types", event.EventType)
		}
	}

	return nil
}

// validateEventDataSize validates event data doesn't exceed reasonable limits
func (v *ComprehensiveAuditValidator) validateEventDataSize(event *AuditEvent) error {
	// Check EventData size
	if event.EventData != nil {
		if len(event.EventData) > 10000 { // 10KB limit
			return fmt.Errorf("event data exceeds maximum size limit of 10KB")
		}
	}

	// Check PreMarshaledEventData size
	if len(event.PreMarshaledEventData) > 10000 { // 10KB limit
		return fmt.Errorf("pre-marshaled event data exceeds maximum size limit of 10KB")
	}

	// Check Description length
	if len(event.Description) > 4096 { // 4KB limit
		return fmt.Errorf("event description exceeds maximum length of 4096 characters")
	}

	return nil
}

// validateIPAddress validates IP address format
func (v *ComprehensiveAuditValidator) validateIPAddress(ipAddress string) error {
	if ipAddress == "" {
		return nil // IP address can be empty
	}

	// Basic IP validation - check for reasonable format
	if len(ipAddress) > 45 { // IPv6 max length
		return fmt.Errorf("IP address too long: %d characters", len(ipAddress))
	}

	// Check for invalid characters
	if strings.ContainsAny(ipAddress, " \t\n\r<>\"'\\") {
		return fmt.Errorf("IP address contains invalid characters")
	}

	return nil
}

// validateUserAgent validates user agent length
func (v *ComprehensiveAuditValidator) validateUserAgent(userAgent string) error {
	if len(userAgent) > 1024 { // 1KB limit
		return fmt.Errorf("user agent exceeds maximum length of 1024 characters")
	}
	return nil
}

// ValidateAuditConfigurationWithComprehensiveChecks validates audit configuration with comprehensive checks
func (v *ComprehensiveAuditValidator) ValidateAuditConfigurationWithComprehensiveChecks(config *AuditConfig) error {
	v.validationMetrics.TotalValidations++
	v.validationMetrics.ConfigurationValidations++

	log.Printf("[AUDIT_CONFIG_VALIDATION] Starting comprehensive configuration validation")

	// 1. Validate data loss prevention configuration
	if err := v.validateDataLossPreventionConfiguration(config); err != nil {
		v.logValidationFailure("data_loss_prevention_validation", nil, err)
		return err
	}

	// 2. Validate MaxConcurrentOverflows with comprehensive checks
	if err := v.validateMaxConcurrentOverflows(config.MaxConcurrentOverflows); err != nil {
		v.logValidationFailure("max_concurrent_overflows_validation", nil, err)
		return err
	}

	// 3. Validate QueueSize with comprehensive checks
	if err := v.validateQueueSize(config.QueueSize); err != nil {
		v.logValidationFailure("queue_size_validation", nil, err)
		return err
	}

	// 4. Validate BatchSize with comprehensive checks
	if err := v.validateBatchSize(config.BatchSize); err != nil {
		v.logValidationFailure("batch_size_validation", nil, err)
		return err
	}

	// 5. Validate MaxRetries with comprehensive checks
	if err := v.validateMaxRetries(config.MaxRetries); err != nil {
		v.logValidationFailure("max_retries_validation", nil, err)
		return err
	}

	// 6. Validate BaseRetryDelay with comprehensive checks
	if err := v.validateBaseRetryDelay(config.BaseRetryDelay); err != nil {
		v.logValidationFailure("base_retry_delay_validation", nil, err)
		return err
	}

	// 7. Validate FlushInterval with comprehensive checks
	if err := v.validateFlushInterval(config.FlushInterval); err != nil {
		v.logValidationFailure("flush_interval_validation", nil, err)
		return err
	}

	// 8. Validate AuditFailureLogPath with comprehensive checks
	if err := v.validateAuditFailureLogPath(config.AuditFailureLogPath); err != nil {
		v.logValidationFailure("audit_failure_log_path_validation", nil, err)
		return err
	}

	// 9. Validate MinSeverity with comprehensive checks
	if err := v.validateMinSeverity(config.MinSeverity); err != nil {
		v.logValidationFailure("min_severity_validation", nil, err)
		return err
	}

	// 10. Validate AllowedEventTypes with comprehensive checks
	if err := v.validateAllowedEventTypes(config.AllowedEventTypes); err != nil {
		v.logValidationFailure("allowed_event_types_validation", nil, err)
		return err
	}

	// 11. Validate configuration consistency
	if err := v.validateConfigurationConsistency(config); err != nil {
		v.logValidationFailure("configuration_consistency_validation", nil, err)
		return err
	}

	// 12. Validate data retention and compliance configuration
	if err := v.validateDataRetentionConfiguration(config); err != nil {
		v.logValidationFailure("data_retention_validation", nil, err)
		return err
	}

	log.Printf("[AUDIT_CONFIG_VALIDATION] Configuration validation successful")
	return nil
}

// validateDataLossPreventionConfiguration validates configuration prevents data loss
func (v *ComprehensiveAuditValidator) validateDataLossPreventionConfiguration(config *AuditConfig) error {
	// 1. Ensure queue size is sufficient to handle expected load
	if config.QueueSize < 1000 {
		return fmt.Errorf("QueueSize too small for data loss prevention: %d (minimum recommended: 1000)", config.QueueSize)
	}

	// 2. Ensure batch size is reasonable for database performance
	if config.BatchSize > config.QueueSize/10 {
		log.Printf("[AUDIT_CONFIG_WARNING] Large batch size relative to queue: %d vs %d",
			config.BatchSize, config.QueueSize)
	}

	// 3. Ensure flush interval is frequent enough to prevent data loss
	if config.FlushInterval > 30*time.Minute {
		return fmt.Errorf("FlushInterval too long for data loss prevention: %v (maximum recommended: 30m)",
			config.FlushInterval)
	}

	// 4. Ensure retry configuration doesn't cause excessive delays
	totalMaxDelay := config.BaseRetryDelay * (1 << uint(config.MaxRetries))
	if totalMaxDelay > 2*time.Minute {
		return fmt.Errorf("Total retry delay too long for data loss prevention: %v (maximum recommended: 2m)",
			totalMaxDelay)
	}

	// 5. Ensure critical events are never filtered out
	if getSeverityLevel(config.MinSeverity) > getSeverityLevel(AuditSeverityCritical) {
		return fmt.Errorf("Configuration would filter out critical events, causing data loss")
	}

	// 6. Ensure critical event types are always allowed
	if config.AllowedEventTypes != nil {
		criticalEventTypes := getCriticalEventTypes()
		for _, criticalType := range criticalEventTypes {
			if !containsEventType(config.AllowedEventTypes, criticalType) {
				return fmt.Errorf("Configuration excludes critical event type %s, causing data loss", criticalType)
			}
		}
	}

	// 7. Ensure failure logging is configured to prevent silent failures
	if config.AuditFailureLogPath == "" {
		log.Printf("[AUDIT_CONFIG_WARNING] No audit failure log path configured - failures may be lost")
	}

	// 8. Validate queue overflow protection
	if config.MaxConcurrentOverflows < 5 {
		return fmt.Errorf("MaxConcurrentOverflows too low for overflow protection: %d (minimum recommended: 5)",
			config.MaxConcurrentOverflows)
	}

	log.Printf("[AUDIT_DATA_LOSS_PREVENTION] Configuration validated for data loss prevention")
	return nil
}

// validateDataRetentionConfiguration validates data retention settings
func (v *ComprehensiveAuditValidator) validateDataRetentionConfiguration(config *AuditConfig) error {
	// This would validate against organizational retention policies
	// For now, we ensure the configuration supports reasonable retention

	// Validate that the configuration supports timely persistence
	if config.FlushInterval > 1*time.Hour {
		return fmt.Errorf("FlushInterval too long for compliance: %v (maximum: 1h)",
			config.FlushInterval)
	}

	// Validate that retry configuration supports reliable persistence
	if config.MaxRetries < 2 {
		return fmt.Errorf("MaxRetries too low for reliable persistence: %d (minimum: 2)",
			config.MaxRetries)
	}

	log.Printf("[AUDIT_DATA_RETENTION] Data retention configuration validated")
	return nil
}

// ValidateAuditEventForDataLossPrevention validates event to prevent data loss
func (v *ComprehensiveAuditValidator) ValidateAuditEventForDataLossPrevention(event *AuditEvent) error {
	// 1. Ensure critical events are never lost
	if event.Severity == AuditSeverityCritical {
		// Critical events should always be processed
		return nil
	}

	// 2. Validate event has sufficient identifying information
	if event.UserID == nil && event.SessionID == nil && event.DeviceID == nil && event.IPAddress == "" {
		return fmt.Errorf("event lacks sufficient identifying information, risk of data loss")
	}

	// 3. Validate event has reasonable timestamp (not too old or future)
	now := time.Now().UTC()
	if event.Timestamp.Before(now.AddDate(-1, 0, 0)) || event.Timestamp.After(now.Add(1*time.Hour)) {
		log.Printf("[AUDIT_DATA_LOSS_RISK] Event timestamp outside reasonable range: %v", event.Timestamp)
	}

	// 4. Validate event data is not empty for important events
	importantEventTypes := []AuditEventType{
		AuditEventAccountDeleted,
		AuditEventAccountBlocked,
		AuditEventAdminAction,
		AuditEventConfigChanged,
		AuditEventDataAccess,
		AuditEventDataModified,
		AuditEventDataDeleted,
	}

	for _, importantType := range importantEventTypes {
		if event.EventType == importantType && len(event.EventData) == 0 {
			return fmt.Errorf("important event %s has no event data, risk of data loss", event.EventType)
		}
	}

	return nil
}

// ValidateSystemResilience validates that the audit system can handle failure scenarios
func (v *ComprehensiveAuditValidator) ValidateSystemResilience() error {
	// 1. Check that dead letter queue has capacity
	if cap(v.auditLogger.deadLetterChan) < 100 {
		return fmt.Errorf("dead letter queue capacity too small: %d (minimum: 100)",
			cap(v.auditLogger.deadLetterChan))
	}

	// 2. Check that overflow semaphore has reasonable capacity
	if cap(v.auditLogger.overflowSemaphore) < 5 {
		return fmt.Errorf("overflow semaphore capacity too small: %d (minimum: 5)",
			cap(v.auditLogger.overflowSemaphore))
	}

	// 3. Validate that failure logging is working
	if v.auditLogger.failureLogger == nil {
		return fmt.Errorf("failure logger not initialized")
	}

	// 4. Check that database connection is healthy
	if v.auditLogger.db != nil {
		if err := v.ValidateDatabaseConnection(v.auditLogger.db); err != nil {
			return fmt.Errorf("database connection validation failed: %w", err)
		}
	}

	log.Printf("[AUDIT_SYSTEM_RESILIENCE] System resilience validation successful")
	return nil
}

// validateMaxConcurrentOverflows validates MaxConcurrentOverflows with comprehensive checks
func (v *ComprehensiveAuditValidator) validateMaxConcurrentOverflows(value int) error {
	v.validationMetrics.PathValidations++

	if value < 1 {
		return fmt.Errorf("MaxConcurrentOverflows must be at least 1 to ensure basic functionality")
	}
	if value > 100 {
		return fmt.Errorf("MaxConcurrentOverflows must not exceed 100 to prevent resource exhaustion")
	}
	if value > 50 {
		log.Printf("[AUDIT_CONFIG_WARNING] High MaxConcurrentOverflows value: %d (recommended max: 50)", value)
	}

	return nil
}

// validateQueueSize validates QueueSize with comprehensive checks
func (v *ComprehensiveAuditValidator) validateQueueSize(value int) error {
	v.validationMetrics.PathValidations++

	if value < 100 {
		return fmt.Errorf("QueueSize must be at least 100 to ensure basic functionality")
	}
	if value > 1000000 {
		return fmt.Errorf("QueueSize must not exceed 1,000,000 to prevent memory exhaustion")
	}
	if value > 500000 {
		log.Printf("[AUDIT_CONFIG_WARNING] High QueueSize value: %d (recommended max: 500,000)", value)
	}

	return nil
}

// validateBatchSize validates BatchSize with comprehensive checks
func (v *ComprehensiveAuditValidator) validateBatchSize(value int) error {
	v.validationMetrics.PathValidations++

	if value < 1 {
		return fmt.Errorf("BatchSize must be at least 1")
	}
	if value > 10000 {
		return fmt.Errorf("BatchSize must not exceed 10,000 to prevent database transaction timeouts and memory pressure")
	}
	if value > 5000 {
		log.Printf("[AUDIT_CONFIG_WARNING] High BatchSize value: %d (recommended max: 5,000)", value)
	}

	return nil
}

// validateMaxRetries validates MaxRetries with comprehensive checks
func (v *ComprehensiveAuditValidator) validateMaxRetries(value int) error {
	v.validationMetrics.PathValidations++

	if value < 0 {
		return fmt.Errorf("MaxRetries must be non-negative")
	}
	if value > 10 {
		return fmt.Errorf("MaxRetries must not exceed 10 to prevent excessive retry delays and resource consumption")
	}
	if value > 5 {
		log.Printf("[AUDIT_CONFIG_WARNING] High MaxRetries value: %d (recommended max: 5)", value)
	}

	return nil
}

// validateBaseRetryDelay validates BaseRetryDelay with comprehensive checks
func (v *ComprehensiveAuditValidator) validateBaseRetryDelay(value time.Duration) error {
	v.validationMetrics.PathValidations++

	if value < 10*time.Millisecond {
		return fmt.Errorf("BaseRetryDelay must be at least 10ms to ensure minimum backoff")
	}
	if value > 5*time.Second {
		return fmt.Errorf("BaseRetryDelay must not exceed 5 seconds to prevent excessive retry delays")
	}
	if value > 1*time.Second {
		log.Printf("[AUDIT_CONFIG_WARNING] High BaseRetryDelay value: %v (recommended max: 1s)", value)
	}

	return nil
}

// validateFlushInterval validates FlushInterval with comprehensive checks
func (v *ComprehensiveAuditValidator) validateFlushInterval(value time.Duration) error {
	v.validationMetrics.PathValidations++

	if value < time.Second {
		return fmt.Errorf("FlushInterval must be at least 1 second to prevent excessive database writes")
	}
	if value > 1*time.Hour {
		return fmt.Errorf("FlushInterval must not exceed 1 hour to ensure timely audit log persistence")
	}
	if value > 30*time.Minute {
		log.Printf("[AUDIT_CONFIG_WARNING] High FlushInterval value: %v (recommended max: 30m)", value)
	}

	return nil
}

// validateAuditFailureLogPath validates AuditFailureLogPath with comprehensive checks
func (v *ComprehensiveAuditValidator) validateAuditFailureLogPath(value string) error {
	v.validationMetrics.PathValidations++

	if value == "" {
		return nil // Empty path is allowed (will use default)
	}

	if len(value) > 255 {
		return fmt.Errorf("AuditFailureLogPath must not exceed 255 characters")
	}

	// Check for invalid characters in path
	if strings.ContainsAny(value, `\:*?"<>|`) {
		return fmt.Errorf("AuditFailureLogPath contains invalid characters that could cause filesystem issues")
	}

	// Check for path traversal attempts
	if strings.Contains(value, "..") {
		return fmt.Errorf("AuditFailureLogPath contains path traversal sequences that could compromise system security")
	}

	// Check if path starts with valid prefix
	if !strings.HasPrefix(value, "/") && !strings.Contains(value, ":\\") {
		// Allow relative paths but ensure they don't contain suspicious patterns
		if strings.HasPrefix(value, ".") && value != "." {
			return fmt.Errorf("AuditFailureLogPath contains suspicious relative path patterns")
		}
	}

	return nil
}

// validateMinSeverity validates MinSeverity with comprehensive checks
func (v *ComprehensiveAuditValidator) validateMinSeverity(value AuditSeverity) error {
	v.validationMetrics.SeverityValidations++

	// Prevent critical events from being filtered out
	if getSeverityLevel(value) > getSeverityLevel(AuditSeverityCritical) {
		return fmt.Errorf("MinSeverity cannot exclude critical events: %s would filter out critical severity events", value)
	}

	// Warn if filtering out high severity events
	if getSeverityLevel(value) > getSeverityLevel(AuditSeverityHigh) {
		log.Printf("[AUDIT_CONFIG_WARNING] MinSeverity would filter out high severity events: %s", value)
	}

	return nil
}

// validateAllowedEventTypes validates AllowedEventTypes with comprehensive checks
func (v *ComprehensiveAuditValidator) validateAllowedEventTypes(value []AuditEventType) error {
	v.validationMetrics.EventTypeValidations++

	if value == nil {
		return nil // nil means all event types are allowed
	}

	// Prevent critical event types from being excluded
	criticalEventTypes := getCriticalEventTypes()
	for _, criticalType := range criticalEventTypes {
		if !containsEventType(value, criticalType) {
			return fmt.Errorf("AllowedEventTypes cannot exclude critical event types: %s is missing", criticalType)
		}
	}

	// Check for duplicate event types
	seen := make(map[AuditEventType]bool)
	for _, eventType := range value {
		if seen[eventType] {
			log.Printf("[AUDIT_CONFIG_WARNING] Duplicate event type in AllowedEventTypes: %s", eventType)
		}
		seen[eventType] = true
	}

	return nil
}

// validateConfigurationConsistency validates overall configuration consistency
func (v *ComprehensiveAuditValidator) validateConfigurationConsistency(config *AuditConfig) error {
	// Check that batch size is reasonable relative to queue size
	if config.BatchSize > config.QueueSize/10 {
		log.Printf("[AUDIT_CONFIG_WARNING] BatchSize is large relative to QueueSize: %d vs %d (recommended ratio: 1:10)",
			config.BatchSize, config.QueueSize)
	}

	// Check that retry configuration is reasonable
	if config.MaxRetries > 0 && config.BaseRetryDelay > 0 {
		totalMaxDelay := config.BaseRetryDelay * (1 << uint(config.MaxRetries)) // 2^MaxRetries
		if totalMaxDelay > 1*time.Minute {
			log.Printf("[AUDIT_CONFIG_WARNING] Total maximum retry delay could be high: %v (recommended max: 1m)",
				totalMaxDelay)
		}
	}

	// Check that flush interval is reasonable relative to batch size
	if config.FlushInterval > 0 && config.BatchSize > 0 {
		maxEventsPerSecond := float64(config.BatchSize) / config.FlushInterval.Seconds()
		if maxEventsPerSecond < 10 {
			log.Printf("[AUDIT_CONFIG_WARNING] Low maximum events per second: %.1f (recommended min: 10)",
				maxEventsPerSecond)
		}
	}

	return nil
}

// ValidateDatabaseConnection validates database connection for audit logging
func (v *ComprehensiveAuditValidator) ValidateDatabaseConnection(db *sql.DB) error {
	v.validationMetrics.TotalValidations++
	v.validationMetrics.DatabaseValidations++

	// Test database connection
	err := db.PingContext(context.Background())
	if err != nil {
		v.logValidationFailure("database_connection_validation", nil, err)
		return fmt.Errorf("database connection validation failed: %w", err)
	}

	// Test audit table existence
	var tableExists bool
	err = db.QueryRowContext(context.Background(), `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_name = 'security_audit_log'
		)
	`).Scan(&tableExists)

	if err != nil {
		v.logValidationFailure("audit_table_validation", nil, err)
		return fmt.Errorf("audit table validation failed: %w", err)
	}

	if !tableExists {
		v.logValidationFailure("audit_table_validation", nil,
			fmt.Errorf("security_audit_log table does not exist"))
		return fmt.Errorf("security_audit_log table does not exist")
	}

	log.Printf("[AUDIT_DATABASE_VALIDATION] Database validation successful")
	return nil
}

// logValidationFailure logs validation failures with comprehensive details
func (v *ComprehensiveAuditValidator) logValidationFailure(validationType string, event *AuditEvent, err error) {
	v.validationMetrics.ValidationFailures++

	// Record validation failure metric
	metrics.AuditValidationFailuresTotal.WithLabelValues(validationType).Inc()

	errorDetails := map[string]interface{}{
		"validation_type":  validationType,
		"error":            err.Error(),
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
		"validation_count": v.validationMetrics.TotalValidations,
		"failure_count":    v.validationMetrics.ValidationFailures,
	}

	if event != nil {
		errorDetails["event_id"] = event.ID
		errorDetails["event_type"] = event.EventType
		errorDetails["event_severity"] = event.Severity
		if event.UserID != nil {
			errorDetails["user_id"] = event.UserID.String()
		}
		errorDetails["ip_address"] = event.IPAddress
		errorDetails["user_agent"] = event.UserAgent
	}

	errorJSON, err := json.Marshal(errorDetails)
	if err != nil {
		v.errorLogger.Printf("Warning: Failed to marshal error details: %v", err)
		errorJSON = []byte("{}")
	}

	// Log to error logger with comprehensive details
	v.errorLogger.Printf("[AUDIT_VALIDATION_FAILURE] %s: %s", validationType, errorJSON)

	// Log to audit failure logger if available
	if v.auditLogger != nil && v.auditLogger.failureLogger != nil {
		v.auditLogger.failureLogger.Printf("[AUDIT_VALIDATION_FAILURE] %s: %s", validationType, errorJSON)
	}

	// Log to standard log with different severity based on context
	if event != nil && event.Severity == AuditSeverityCritical {
		log.Printf("[AUDIT_CRITICAL_VALIDATION_FAILURE] Critical event validation failed: %s, EventID=%s, Error=%v",
			validationType, event.ID, err)
	} else {
		log.Printf("[AUDIT_VALIDATION_FAILURE] %s: %v", validationType, err)
	}

	// Write to validation failure log file for forensic analysis
	v.writeValidationFailureToLogFile(validationType, errorDetails)
}

// writeValidationFailureToLogFile writes validation failures to dedicated log file
func (v *ComprehensiveAuditValidator) writeValidationFailureToLogFile(validationType string, details map[string]interface{}) {
	validationLogFile := "audit_validation_failures.log"
	file, err := os.OpenFile(validationLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		v.errorLogger.Printf("Failed to open validation failure log file: %v", err)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			v.errorLogger.Printf("Warning: failed to close file: %v", err)
		}
	}()

	logEntry, err := json.Marshal(details)
	if err != nil {
		v.errorLogger.Printf("Warning: Failed to marshal log entry: %v", err)
		return
	}
	logEntry = append(logEntry, '\n')

	if _, writeErr := file.Write(logEntry); writeErr != nil {
		v.errorLogger.Printf("Failed to write to validation failure log: %v", writeErr)
	}
}

// logValidationSuccess logs successful validations with comprehensive details
func (v *ComprehensiveAuditValidator) logValidationSuccess(validationType string, event *AuditEvent) {
	successDetails := map[string]interface{}{
		"validation_type":  validationType,
		"result":           "success",
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
		"validation_count": v.validationMetrics.TotalValidations,
		"success_count":    v.validationMetrics.TotalValidations - v.validationMetrics.ValidationFailures,
	}

	if event != nil {
		successDetails["event_id"] = event.ID
		successDetails["event_type"] = event.EventType
		successDetails["event_severity"] = event.Severity
	}

	successJSON, err := json.Marshal(successDetails)
	if err != nil {
		log.Printf("Warning: Failed to marshal success details: %v", err)
		successJSON = []byte("{}")
	}

	// Log validation success with appropriate detail level
	if event == nil || event.Severity != AuditSeverityCritical {
		log.Printf("[AUDIT_VALIDATION_SUCCESS] %s: EventID=%s, Type=%s",
			validationType, successDetails["event_id"], successDetails["event_type"])
	} else {
		// For critical events, log more comprehensive success details
		log.Printf("[AUDIT_CRITICAL_VALIDATION_SUCCESS] %s: %s", validationType, successJSON)
	}
}

// logCriticalEventBypass logs when critical events bypass normal validation
func (v *ComprehensiveAuditValidator) logCriticalEventBypass(event *AuditEvent) {
	v.validationMetrics.CriticalEventBypasses++
	metrics.AuditCriticalEventBypassesTotal.Inc()

	bypassDetails := map[string]interface{}{
		"event_id":      event.ID,
		"event_type":    event.EventType,
		"severity":      event.Severity,
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
		"bypass_reason": "critical_severity_override",
	}

	bypassJSON, err := json.Marshal(bypassDetails)
	if err != nil {
		log.Printf("Warning: Failed to marshal bypass details: %v", err)
		bypassJSON = []byte("{}")
	}

	log.Printf("[AUDIT_CRITICAL_BYPASS] Critical event bypassed all validation: %s", bypassJSON)

	// Write to bypass log for compliance auditing
	v.writeCriticalBypassToLogFile(bypassDetails)
}

// writeCriticalBypassToLogFile writes critical event bypasses to dedicated log file
func (v *ComprehensiveAuditValidator) writeCriticalBypassToLogFile(details map[string]interface{}) {
	bypassLogFile := "audit_critical_bypasses.log"
	file, err := os.OpenFile(bypassLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		v.errorLogger.Printf("Failed to open critical bypass log file: %v", err)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			v.errorLogger.Printf("Warning: failed to close file: %v", err)
		}
	}()

	logEntry, err := json.Marshal(details)
	if err != nil {
		v.errorLogger.Printf("Warning: Failed to marshal bypass log entry: %v", err)
		return
	}
	logEntry = append(logEntry, '\n')

	if _, writeErr := file.Write(logEntry); writeErr != nil {
		v.errorLogger.Printf("Failed to write to critical bypass log: %v", writeErr)
	}
}

// logConfigurationValidationResult logs configuration validation results
func (v *ComprehensiveAuditValidator) logConfigurationValidationResult(config *AuditConfig, success bool, err error) {
	validationResult := map[string]interface{}{
		"validation_type": "configuration",
		"result":          "success",
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
		"config_summary": map[string]interface{}{
			"queue_size":               config.QueueSize,
			"batch_size":               config.BatchSize,
			"max_retries":              config.MaxRetries,
			"flush_interval":           config.FlushInterval.String(),
			"base_retry_delay":         config.BaseRetryDelay.String(),
			"max_concurrent_overflows": config.MaxConcurrentOverflows,
		},
	}

	if !success {
		validationResult["result"] = "failure"
		validationResult["error"] = err.Error()
		log.Printf("[AUDIT_CONFIG_VALIDATION_FAILURE] Configuration validation failed: %v", err)
	} else {
		log.Printf("[AUDIT_CONFIG_VALIDATION_SUCCESS] Configuration validation successful")
	}

	// Write to configuration validation log
	v.writeConfigValidationToLogFile(validationResult)

	// Write to configuration validation log
	v.writeConfigValidationToLogFile(validationResult)
}

// writeConfigValidationToLogFile writes configuration validation results to log file
func (v *ComprehensiveAuditValidator) writeConfigValidationToLogFile(result map[string]interface{}) {
	configLogFile := "audit_config_validation.log"
	file, err := os.OpenFile(configLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		v.errorLogger.Printf("Failed to open config validation log file: %v", err)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			v.errorLogger.Printf("Warning: failed to close file: %v", err)
		}
	}()

	logEntry, err := json.Marshal(result)
	if err != nil {
		v.errorLogger.Printf("Warning: Failed to marshal config validation log entry: %v", err)
		return
	}
	logEntry = append(logEntry, '\n')

	if _, writeErr := file.Write(logEntry); writeErr != nil {
		v.errorLogger.Printf("Failed to write to config validation log: %v", writeErr)
	}
}

// GetValidationMetrics returns current validation metrics
func (v *ComprehensiveAuditValidator) GetValidationMetrics() *ComprehensiveValidationMetrics {
	return v.validationMetrics
}

// ResetValidationMetrics resets validation metrics
func (v *ComprehensiveAuditValidator) ResetValidationMetrics() {
	v.validationMetrics = &ComprehensiveValidationMetrics{}
}

// ValidateAuditEventBeforeLogging validates event before logging to prevent data loss
func (v *ComprehensiveAuditValidator) ValidateAuditEventBeforeLogging(event *AuditEvent) error {
	// This is a lightweight validation to prevent data loss from malformed events
	if event == nil {
		return fmt.Errorf("cannot log nil audit event")
	}

	if event.EventType == "" {
		return fmt.Errorf("cannot log audit event with empty event type")
	}

	// Ensure critical events are never filtered out
	if event.Severity == AuditSeverityCritical {
		return nil // Always allow critical events
	}

	return nil
}

// ValidateAuditSystemHealth performs comprehensive health check of audit system
func (v *ComprehensiveAuditValidator) ValidateAuditSystemHealth() error {
	// Check queue health
	queueLength := len(v.auditLogger.queue)
	if queueLength > v.auditLogger.config.QueueSize/2 {
		log.Printf("[AUDIT_HEALTH_WARNING] High queue length: %d/%d", queueLength, v.auditLogger.config.QueueSize)
	}

	// Check dead letter queue health
	deadLetterLength := len(v.auditLogger.deadLetterChan)
	if deadLetterLength > 500 {
		log.Printf("[AUDIT_HEALTH_WARNING] High dead letter queue length: %d", deadLetterLength)
	}

	// Check metrics for anomalies - note: counters don't have Value() method in this context
	// This would need to be implemented differently for actual monitoring

	return nil
}
