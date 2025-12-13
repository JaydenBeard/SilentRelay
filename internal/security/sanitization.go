// Package security provides security utilities for the messaging application.
// This file contains input sanitization functions to protect against various attacks.
package security

import (
	"regexp"
	"strings"
	"unicode"
)

// ===========================================================================
// Input Sanitization Constants
// ===========================================================================

// Maximum lengths for various input types
const (
	MaxUsernameLength     = 50
	MaxDisplayNameLength  = 100
	MaxEmailLength        = 254
	MaxPhoneLength        = 20
	MaxMessageLength      = 10000
	MaxGroupNameLength    = 100
	MaxDeviceNameLength   = 100
	MaxSearchQueryLength  = 100
	MaxURLLength          = 2000
	MaxGenericInputLength = 500
)

// ===========================================================================
// SQL Injection Prevention
// ===========================================================================

// sanitizationSQLPatterns contains common SQL injection patterns for sanitization
// NOTE: Uses different name to avoid conflict with hardening.go patterns
var sanitizationSQLPatterns = []string{
	"--",
	";--",
	"/*",
	"*/",
	"@@",
	"@",
	"char(",
	"nchar(",
	"varchar(",
	"nvarchar(",
	"alter ",
	"begin ",
	"cast(",
	"create ",
	"cursor ",
	"declare ",
	"delete ",
	"drop ",
	"exec(",
	"execute(",
	"fetch ",
	"insert ",
	"kill ",
	"select ",
	"sys.",
	"sysobjects",
	"syscolumns",
	"shutdown ",
	"update ",
	"union ",
	"xp_",
}

// sanitizationSQLRegex matches common SQL injection attempts
var sanitizationSQLRegex = regexp.MustCompile(`(?i)(union\s+select|select\s+\*|drop\s+table|delete\s+from|insert\s+into|update\s+.+\s+set|or\s+1\s*=\s*1|or\s+'1'\s*=\s*'1'|or\s+"1"\s*=\s*"1"|and\s+1\s*=\s*1|'\s*or\s*'|"\s*or\s*"|;\s*drop|;\s*delete|;\s*update|;\s*insert|;\s*select)`)

// ContainsSQLInjection checks if the input contains potential SQL injection patterns
func ContainsSQLInjection(input string) bool {
	lower := strings.ToLower(input)

	// Check regex patterns
	if sanitizationSQLRegex.MatchString(lower) {
		return true
	}

	// Check string patterns
	for _, pattern := range sanitizationSQLPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	return false
}

// SanitizeSQLLikePattern sanitizes input for use in SQL LIKE clauses
// Escapes special LIKE characters: %, _, [, ]
func SanitizeSQLLikePattern(input string) string {
	input = strings.ReplaceAll(input, "\\", "\\\\")
	input = strings.ReplaceAll(input, "%", "\\%")
	input = strings.ReplaceAll(input, "_", "\\_")
	input = strings.ReplaceAll(input, "[", "\\[")
	input = strings.ReplaceAll(input, "]", "\\]")
	return input
}

// ===========================================================================
// XSS Prevention
// ===========================================================================

// sanitizationXSSPatterns contains common XSS patterns for sanitization
// NOTE: Uses different name to avoid conflict with hardening.go patterns
var sanitizationXSSPatterns = []string{
	"<script",
	"</script>",
	"javascript:",
	"vbscript:",
	"onload=",
	"onerror=",
	"onclick=",
	"onmouseover=",
	"onfocus=",
	"onblur=",
	"onchange=",
	"onsubmit=",
	"<iframe",
	"<frame",
	"<embed",
	"<object",
	"<svg",
	"<img",
	"expression(",
	"data:",
}

// htmlEntities for basic HTML escaping
var htmlReplacer = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	"\"", "&quot;",
	"'", "&#39;",
	"`", "&#96;",
)

// ContainsXSS checks if the input contains potential XSS patterns
func ContainsXSS(input string) bool {
	lower := strings.ToLower(input)

	for _, pattern := range sanitizationXSSPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	return false
}

// SanitizeHTML escapes HTML entities to prevent XSS
func SanitizeHTML(input string) string {
	return htmlReplacer.Replace(input)
}

// StripHTML removes all HTML tags from input
func StripHTML(input string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(input, "")
}

// ===========================================================================
// Path Traversal Prevention
// ===========================================================================

// pathTraversalPatterns contains common path traversal patterns
var pathTraversalPatterns = []string{
	"..",
	"../",
	"..\\",
	"%2e%2e",
	"%2e%2e%2f",
	"%2e%2e/",
	"..%2f",
	"%2e%2e\\",
	"....//",
	"....\\\\",
}

// ContainsPathTraversal checks if the input contains path traversal attempts
func ContainsPathTraversal(input string) bool {
	lower := strings.ToLower(input)

	for _, pattern := range pathTraversalPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	return false
}

// SanitizePath removes path traversal sequences and normalizes the path
func SanitizePath(input string) string {
	// Remove any path traversal patterns
	result := input
	for _, pattern := range pathTraversalPatterns {
		result = strings.ReplaceAll(strings.ToLower(result), pattern, "")
	}

	// Remove any remaining double slashes
	for strings.Contains(result, "//") {
		result = strings.ReplaceAll(result, "//", "/")
	}
	for strings.Contains(result, "\\\\") {
		result = strings.ReplaceAll(result, "\\\\", "\\")
	}

	return result
}

// ===========================================================================
// General Input Sanitization
// ===========================================================================

// SanitizeInput performs comprehensive input sanitization
func SanitizeInput(input string, maxLength int) string {
	if maxLength <= 0 {
		maxLength = MaxGenericInputLength
	}

	// Trim whitespace
	result := strings.TrimSpace(input)

	// Enforce max length
	if len(result) > maxLength {
		result = result[:maxLength]
	}

	// Remove null bytes
	result = strings.ReplaceAll(result, "\x00", "")

	// Normalize unicode
	result = normalizeUnicode(result)

	return result
}

// normalizeUnicode removes potentially dangerous unicode characters
func normalizeUnicode(input string) string {
	var builder strings.Builder
	for _, r := range input {
		// Allow printable characters and common whitespace
		if unicode.IsPrint(r) || r == '\n' || r == '\r' || r == '\t' {
			// Skip control characters
			if !unicode.IsControl(r) || r == '\n' || r == '\r' || r == '\t' {
				builder.WriteRune(r)
			}
		}
	}
	return builder.String()
}

// SanitizeUsername sanitizes and validates usernames
func SanitizeUsername(input string) (string, bool) {
	result := SanitizeInput(input, MaxUsernameLength)

	// Usernames should be alphanumeric with underscores/hyphens only
	validUsername := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validUsername.MatchString(result) {
		return "", false
	}

	// Minimum length check
	if len(result) < 3 {
		return "", false
	}

	return result, true
}

// SanitizeDisplayName sanitizes display names
func SanitizeDisplayName(input string) string {
	result := SanitizeInput(input, MaxDisplayNameLength)

	// Strip any HTML tags
	result = StripHTML(result)

	// Remove any SQL injection patterns (just strip them)
	for _, pattern := range sanitizationSQLPatterns {
		result = strings.ReplaceAll(strings.ToLower(result), pattern, "")
	}

	return result
}

// SanitizePhoneNumber sanitizes phone numbers
func SanitizePhoneNumber(input string) (string, bool) {
	// Remove spaces, dashes, parentheses
	result := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) || r == '+' {
			return r
		}
		return -1
	}, input)

	// Validate length
	if len(result) < 8 || len(result) > MaxPhoneLength {
		return "", false
	}

	// Should start with + for international format
	if !strings.HasPrefix(result, "+") {
		return "", false
	}

	return result, true
}

// SanitizeSearchQuery sanitizes search queries
func SanitizeSearchQuery(input string) string {
	result := SanitizeInput(input, MaxSearchQueryLength)

	// Escape SQL LIKE special characters
	result = SanitizeSQLLikePattern(result)

	return result
}

// SanitizeGroupName sanitizes group names
func SanitizeGroupName(input string) string {
	result := SanitizeInput(input, MaxGroupNameLength)
	result = StripHTML(result)
	return result
}

// SanitizeDeviceName sanitizes device names
func SanitizeDeviceName(input string) string {
	result := SanitizeInput(input, MaxDeviceNameLength)
	result = StripHTML(result)
	return result
}

// ===========================================================================
// Validation Helpers
// ===========================================================================

// IsValidUUID checks if a string is a valid UUID format
func IsValidUUID(input string) bool {
	uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	return uuidRegex.MatchString(input)
}

// IsValidEmail checks if a string appears to be a valid email
func IsValidEmail(input string) bool {
	if len(input) > MaxEmailLength {
		return false
	}
	// RFC 5322 compliant email regex (simplified)
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(input)
}

// IsAlphanumeric checks if input contains only alphanumeric characters
func IsAlphanumeric(input string) bool {
	for _, r := range input {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// ContainsOnlyAllowedChars checks if input contains only characters from allowed set
func ContainsOnlyAllowedChars(input string, allowed string) bool {
	allowedSet := make(map[rune]bool)
	for _, r := range allowed {
		allowedSet[r] = true
	}

	for _, r := range input {
		if !allowedSet[r] {
			return false
		}
	}
	return true
}

// ===========================================================================
// Security Analysis
// ===========================================================================

// InputSecurityReport contains the results of security analysis
type InputSecurityReport struct {
	HasSQLInjection   bool
	HasXSS            bool
	HasPathTraversal  bool
	ExceedsMaxLength  bool
	ContainsNullBytes bool
	SanitizedValue    string
	SecurityScore     int // 0-100, higher is safer
}

// AnalyzeInput performs comprehensive security analysis on input
func AnalyzeInput(input string, maxLength int) InputSecurityReport {
	report := InputSecurityReport{
		HasSQLInjection:   ContainsSQLInjection(input),
		HasXSS:            ContainsXSS(input),
		HasPathTraversal:  ContainsPathTraversal(input),
		ExceedsMaxLength:  len(input) > maxLength,
		ContainsNullBytes: strings.Contains(input, "\x00"),
	}

	// Calculate security score
	score := 100
	if report.HasSQLInjection {
		score -= 40
	}
	if report.HasXSS {
		score -= 30
	}
	if report.HasPathTraversal {
		score -= 20
	}
	if report.ExceedsMaxLength {
		score -= 5
	}
	if report.ContainsNullBytes {
		score -= 5
	}

	if score < 0 {
		score = 0
	}
	report.SecurityScore = score

	// Provide sanitized version
	report.SanitizedValue = SanitizeInput(input, maxLength)

	return report
}

// IsSafe returns true if the input passes all security checks
func (r *InputSecurityReport) IsSafe() bool {
	return !r.HasSQLInjection && !r.HasXSS && !r.HasPathTraversal &&
		!r.ContainsNullBytes && !r.ExceedsMaxLength
}
