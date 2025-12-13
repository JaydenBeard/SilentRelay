package security

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// ============================================
// ZERO TRUST ARCHITECTURE
// Never trust, always verify
// "Assume breach" mentality
// ============================================

// ZeroTrustPolicy defines access control policies
type ZeroTrustPolicy struct {
	// Identity verification
	RequireRecentAuth      bool          // Must have authenticated recently
	MaxAuthAge             time.Duration // How recent is "recent"
	RequireDeviceBinding   bool          // Session must be bound to device
	RequirePINVerification bool          // PIN must be verified this session

	// Context verification
	AllowedIPRanges      []string     // Restrict to specific IPs
	AllowedCountries     []string     // GeoIP restriction
	AllowedTimeWindows   []TimeWindow // Time-based access
	RequireSecureContext bool         // Must be HTTPS

	// Behavioral verification
	MaxRiskScore  int  // Maximum acceptable risk score
	RequireStepUp bool // Require additional verification
}

// TimeWindow defines allowed access times
type TimeWindow struct {
	DaysOfWeek []time.Weekday
	StartHour  int
	EndHour    int
	Timezone   string
}

// ZeroTrustContext contains request context for verification
type ZeroTrustContext struct {
	UserID            uuid.UUID
	SessionID         uuid.UUID
	DeviceID          uuid.UUID
	IPAddress         string
	UserAgent         string
	Timestamp         time.Time
	AuthenticatedAt   time.Time
	PINVerifiedAt     *time.Time
	DeviceFingerprint string
	RiskScore         int
	Country           string
	IsSecure          bool
}

// ZeroTrustEnforcer enforces zero trust policies
type ZeroTrustEnforcer struct {
	defaultPolicy    *ZeroTrustPolicy
	resourcePolicies map[string]*ZeroTrustPolicy
	riskScorer       *RiskScorer
}

// NewZeroTrustEnforcer creates a new zero trust enforcer
func NewZeroTrustEnforcer() *ZeroTrustEnforcer {
	return &ZeroTrustEnforcer{
		defaultPolicy: &ZeroTrustPolicy{
			RequireRecentAuth:      true,
			MaxAuthAge:             24 * time.Hour,
			RequireDeviceBinding:   true,
			RequirePINVerification: true,
			RequireSecureContext:   true,
			MaxRiskScore:           50,
		},
		resourcePolicies: make(map[string]*ZeroTrustPolicy),
		riskScorer:       NewRiskScorer(),
	}
}

// SetResourcePolicy sets a policy for a specific resource
func (zte *ZeroTrustEnforcer) SetResourcePolicy(resource string, policy *ZeroTrustPolicy) {
	zte.resourcePolicies[resource] = policy
}

// Verify checks if access should be allowed
func (zte *ZeroTrustEnforcer) Verify(ctx *ZeroTrustContext, resource string) *ZeroTrustDecision {
	policy := zte.defaultPolicy
	if p, ok := zte.resourcePolicies[resource]; ok {
		policy = p
	}

	decision := &ZeroTrustDecision{
		Allowed: true,
		Reasons: []string{},
	}

	// Check authentication age
	if policy.RequireRecentAuth {
		authAge := time.Since(ctx.AuthenticatedAt)
		if authAge > policy.MaxAuthAge {
			decision.Allowed = false
			decision.Reasons = append(decision.Reasons, "Authentication too old")
			decision.RequiredAction = "reauthenticate"
		}
	}

	// Check PIN verification
	if policy.RequirePINVerification && decision.Allowed {
		if ctx.PINVerifiedAt == nil {
			decision.Allowed = false
			decision.Reasons = append(decision.Reasons, "PIN verification required")
			decision.RequiredAction = "verify_pin"
		}
	}

	// Check secure context
	if policy.RequireSecureContext && !ctx.IsSecure {
		decision.Allowed = false
		decision.Reasons = append(decision.Reasons, "Secure connection required")
		decision.RequiredAction = "use_https"
	}

	// Check risk score
	if ctx.RiskScore > policy.MaxRiskScore {
		decision.Allowed = false
		decision.Reasons = append(decision.Reasons, "Risk score too high")
		decision.RequiredAction = "step_up_auth"
	}

	return decision
}

// ZeroTrustDecision represents an access decision
type ZeroTrustDecision struct {
	Allowed        bool
	Reasons        []string
	RequiredAction string
	StepUpRequired bool
}

// ZeroTrustMiddleware enforces zero trust for HTTP requests
func (zte *ZeroTrustEnforcer) ZeroTrustMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Build context from request
		ctx := zte.buildContext(r)

		// Calculate risk score
		ctx.RiskScore = zte.riskScorer.Score(ctx)

		// Verify access
		decision := zte.Verify(ctx, r.URL.Path)

		if !decision.Allowed {
			w.Header().Set("X-ZeroTrust-Denied", decision.Reasons[0])
			w.Header().Set("X-ZeroTrust-Action", decision.RequiredAction)
			http.Error(w, "Access Denied: "+decision.Reasons[0], http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (zte *ZeroTrustEnforcer) buildContext(r *http.Request) *ZeroTrustContext {
	return &ZeroTrustContext{
		IPAddress: GetRealIP(r),
		UserAgent: r.UserAgent(),
		Timestamp: time.Now(),
		IsSecure:  r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
	}
}

// ============================================
// RISK SCORING
// Continuous risk assessment
// ============================================

// RiskScorer calculates risk scores for requests
type RiskScorer struct {
	weights map[string]int
}

// NewRiskScorer creates a new risk scorer
func NewRiskScorer() *RiskScorer {
	return &RiskScorer{
		weights: map[string]int{
			"new_device":        20,
			"new_ip":            15,
			"unusual_time":      10,
			"rapid_requests":    25,
			"suspicious_agent":  30,
			"failed_auth":       35,
			"tor_exit":          40,
			"vpn_detected":      10,
			"proxy_detected":    15,
			"impossible_travel": 50,
		},
	}
}

// Score calculates a risk score (0-100)
func (rs *RiskScorer) Score(ctx *ZeroTrustContext) int {
	score := 0

	// New device detection
	if rs.isNewDevice(ctx) {
		score += rs.weights["new_device"]
	}

	// New IP detection
	if rs.isNewIP(ctx) {
		score += rs.weights["new_ip"]
	}

	// Unusual time access
	if rs.isUnusualTime(ctx) {
		score += rs.weights["unusual_time"]
	}

	// Suspicious user agent
	if rs.isSuspiciousAgent(ctx.UserAgent) {
		score += rs.weights["suspicious_agent"]
	}

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return score
}

func (rs *RiskScorer) isNewDevice(_ *ZeroTrustContext) bool {
	// Would check device history based on ctx.DeviceFingerprint
	// Implementation would use ctx.DeviceFingerprint
	return false
}

func (rs *RiskScorer) isNewIP(_ *ZeroTrustContext) bool {
	// Would check IP history based on ctx.IPAddress
	// Implementation would use ctx.IPAddress
	return false
}

func (rs *RiskScorer) isUnusualTime(ctx *ZeroTrustContext) bool {
	hour := ctx.Timestamp.Hour()
	return hour < 6 || hour > 23 // Unusual hours
}

func (rs *RiskScorer) isSuspiciousAgent(ua string) bool {
	// Check for automation tools, scanners, etc.
	suspicious := []string{"curl", "wget", "python", "go-http", "scanner"}
	for _, s := range suspicious {
		if contains(ua, s) {
			return true
		}
	}
	return false
}

// ============================================
// MICRO-SEGMENTATION
// Fine-grained network/resource isolation
// ============================================

// Segment represents a security segment
type Segment struct {
	ID          string
	Name        string
	Description string
	Resources   []string
	AllowedFrom []string // Segment IDs that can access
	AllowedTo   []string // Segment IDs this can access
}

// SegmentManager manages micro-segmentation
type SegmentManager struct {
	segments map[string]*Segment
}

// NewSegmentManager creates a segment manager
func NewSegmentManager() *SegmentManager {
	sm := &SegmentManager{
		segments: make(map[string]*Segment),
	}

	// Define default segments
	sm.AddSegment(&Segment{
		ID:          "public",
		Name:        "Public",
		Description: "Public-facing resources",
		Resources:   []string{"/api/v1/auth/*", "/health", "/ws"},
		AllowedFrom: []string{"*"},
	})

	sm.AddSegment(&Segment{
		ID:          "authenticated",
		Name:        "Authenticated",
		Description: "Requires authentication",
		Resources:   []string{"/api/v1/users/*", "/api/v1/messages/*"},
		AllowedFrom: []string{"public"},
	})

	sm.AddSegment(&Segment{
		ID:          "sensitive",
		Name:        "Sensitive",
		Description: "Sensitive operations",
		Resources:   []string{"/api/v1/keys/*", "/api/v1/security/*"},
		AllowedFrom: []string{"authenticated"},
	})

	sm.AddSegment(&Segment{
		ID:          "admin",
		Name:        "Admin",
		Description: "Administrative operations",
		Resources:   []string{"/api/v1/admin/*"},
		AllowedFrom: []string{"sensitive"},
	})

	return sm
}

// AddSegment adds a new segment
func (sm *SegmentManager) AddSegment(segment *Segment) {
	sm.segments[segment.ID] = segment
}

// CanAccess checks if access is allowed between segments
func (sm *SegmentManager) CanAccess(fromSegment, toSegment string) bool {
	to, ok := sm.segments[toSegment]
	if !ok {
		return false
	}

	for _, allowed := range to.AllowedFrom {
		if allowed == "*" || allowed == fromSegment {
			return true
		}
	}

	return false
}

// ============================================
// CONTINUOUS VERIFICATION
// Re-verify throughout the session
// ============================================

// ContinuousVerifier performs ongoing verification
type ContinuousVerifier struct {
	checkInterval time.Duration
	lastCheck     map[uuid.UUID]time.Time
}

// NewContinuousVerifier creates a continuous verifier
func NewContinuousVerifier() *ContinuousVerifier {
	return &ContinuousVerifier{
		checkInterval: 5 * time.Minute,
		lastCheck:     make(map[uuid.UUID]time.Time),
	}
}

// ShouldReVerify checks if re-verification is needed
func (cv *ContinuousVerifier) ShouldReVerify(sessionID uuid.UUID) bool {
	last, ok := cv.lastCheck[sessionID]
	if !ok {
		return true
	}
	return time.Since(last) > cv.checkInterval
}

// RecordVerification records a successful verification
func (cv *ContinuousVerifier) RecordVerification(sessionID uuid.UUID) {
	cv.lastCheck[sessionID] = time.Now()
}

// ============================================
// IDENTITY BINDING
// Cryptographically bind identity to session
// ============================================

// IdentityBinding cryptographically binds session to identity
type IdentityBinding struct {
	SessionID    uuid.UUID
	UserID       uuid.UUID
	DeviceID     uuid.UUID
	BindingProof []byte
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

// CreateBinding creates a cryptographic identity binding
func CreateBinding(sessionID, userID, deviceID uuid.UUID) *IdentityBinding {
	// Create binding proof
	data := sessionID.String() + userID.String() + deviceID.String()
	hash := sha256.Sum256([]byte(data))

	return &IdentityBinding{
		SessionID:    sessionID,
		UserID:       userID,
		DeviceID:     deviceID,
		BindingProof: hash[:],
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}
}

// Verify verifies the binding proof
func (ib *IdentityBinding) Verify() bool {
	if time.Now().After(ib.ExpiresAt) {
		return false
	}

	data := ib.SessionID.String() + ib.UserID.String() + ib.DeviceID.String()
	hash := sha256.Sum256([]byte(data))

	return hex.EncodeToString(hash[:]) == hex.EncodeToString(ib.BindingProof)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
