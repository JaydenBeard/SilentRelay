package security

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"sync"
	"time"
)

// ============================================
// CHAOS ENGINEERING FOR SECURITY
// Intentionally break things to verify defenses work
// ============================================

// ChaosExperiment defines a chaos engineering test
type ChaosExperiment struct {
	ID          string
	Name        string
	Description string
	Type        ChaosType
	Duration    time.Duration
	Probability float64 // 0-1, chance of triggering
	Active      bool
	StartedAt   *time.Time
	EndsAt      *time.Time
}

// ChaosType defines types of chaos experiments
type ChaosType string

const (
	// Network chaos
	ChaosTypeLatency      ChaosType = "latency"       // Add random latency
	ChaosTypePacketLoss   ChaosType = "packet_loss"   // Drop requests randomly
	ChaosTypeNetworkSplit ChaosType = "network_split" // Simulate network partition

	// Service chaos
	ChaosTypeServiceDown  ChaosType = "service_down"  // Simulate service failure
	ChaosTypeResourceHog  ChaosType = "resource_hog"  // Consume resources
	ChaosTypeSlowResponse ChaosType = "slow_response" // Slow down responses

	// Security chaos
	ChaosTypeFakeAttack  ChaosType = "fake_attack"  // Simulate attack traffic
	ChaosTypeKeyRotation ChaosType = "key_rotation" // Force key rotation
	ChaosTypeSessionKill ChaosType = "session_kill" // Randomly invalidate sessions
	ChaosTypeRateLimit   ChaosType = "rate_limit"   // Hit rate limits
)

// ChaosMonkey manages chaos experiments
type ChaosMonkey struct {
	mu          sync.RWMutex
	experiments map[string]*ChaosExperiment
	enabled     bool
	db          *sql.DB
	// Using crypto/rand instead of math/rand for security-conscious randomness
}

// NewChaosMonkey creates a new chaos engineering manager
func NewChaosMonkey(db *sql.DB, enabled bool) *ChaosMonkey {
	return &ChaosMonkey{
		experiments: make(map[string]*ChaosExperiment),
		enabled:     enabled,
		db:          db,
	}
}

// RegisterExperiment adds a new experiment
func (cm *ChaosMonkey) RegisterExperiment(exp *ChaosExperiment) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.experiments[exp.ID] = exp
}

// StartExperiment begins a chaos experiment
func (cm *ChaosMonkey) StartExperiment(experimentID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	exp, ok := cm.experiments[experimentID]
	if !ok {
		return fmt.Errorf("experiment not found: %s", experimentID)
	}

	now := time.Now()
	exp.Active = true
	exp.StartedAt = &now
	endsAt := now.Add(exp.Duration)
	exp.EndsAt = &endsAt

	// Schedule auto-stop
	go func() {
		time.Sleep(exp.Duration)
		if err := cm.StopExperiment(experimentID); err != nil {
			// Log error but don't fail - experiment may have been manually stopped
			log.Printf("Warning: failed to auto-stop experiment %s: %v", experimentID, err)
		}
	}()

	return nil
}

// StopExperiment stops an active experiment
func (cm *ChaosMonkey) StopExperiment(experimentID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if exp, ok := cm.experiments[experimentID]; ok {
		exp.Active = false
		exp.StartedAt = nil
		exp.EndsAt = nil
	}

	return nil
}

// ShouldTrigger checks if chaos should be injected
func (cm *ChaosMonkey) ShouldTrigger(experimentID string) bool {
	if !cm.enabled {
		return false
	}

	cm.mu.RLock()
	defer cm.mu.RUnlock()

	exp, ok := cm.experiments[experimentID]
	if !ok || !exp.Active {
		return false
	}

	// Use crypto/rand for secure randomness
	n, err := rand.Int(rand.Reader, big.NewInt(1000))
	if err != nil {
		return false
	}
	return float64(n.Int64())/1000.0 < exp.Probability
}

// ChaosMiddleware injects chaos into HTTP requests
func (cm *ChaosMonkey) ChaosMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !cm.enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Latency injection
		if cm.ShouldTrigger("latency") {
			n, err := rand.Int(rand.Reader, big.NewInt(2000))
			if err != nil {
				log.Printf("Warning: Failed to generate random latency: %v", err)
				n = big.NewInt(1000) // Default to 1 second
			}
			delay := time.Duration(n.Int64()) * time.Millisecond
			time.Sleep(delay)
		}

		// Packet loss simulation
		if cm.ShouldTrigger("packet_loss") {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
			return
		}

		// Slow response
		if cm.ShouldTrigger("slow_response") {
			// Wrap response writer to add delays
			w = &slowResponseWriter{ResponseWriter: w, delay: 100 * time.Millisecond}
		}

		next.ServeHTTP(w, r)
	})
}

// slowResponseWriter adds delays to response writing
type slowResponseWriter struct {
	http.ResponseWriter
	delay time.Duration
}

func (s *slowResponseWriter) Write(data []byte) (int, error) {
	time.Sleep(s.delay)
	return s.ResponseWriter.Write(data)
}

// ============================================
// SECURITY CHAOS EXPERIMENTS
// ============================================

// SimulateAttack generates fake attack traffic
func (cm *ChaosMonkey) SimulateAttack(ctx context.Context, attackType string) error {
	// This generates realistic-looking attack traffic
	// to test our detection systems

	switch attackType {
	case "brute_force":
		// Simulate brute force attempts
		for i := 0; i < 100; i++ {
			// Generate fake login attempts
			time.Sleep(10 * time.Millisecond)
		}

	case "sql_injection":
		// Generate SQL injection patterns
		// Would be sent to test endpoint
		// payloads would be used in actual implementation

	case "scan":
		// Simulate port/endpoint scanning
		// Would be requested to test detection
		// endpoints would be used in actual implementation

	default:
		return fmt.Errorf("unknown attack type: %s", attackType)
	}

	return nil
}

// ForceKeyRotation triggers emergency key rotation
func (cm *ChaosMonkey) ForceKeyRotation(ctx context.Context, userID string) error {
	// Force key rotation to test the process works correctly
	_, err := cm.db.ExecContext(ctx, `
		UPDATE users SET signed_prekey_updated_at = NOW() - INTERVAL '30 days'
		WHERE user_id = $1
	`, userID)
	return err
}

// SimulateSessionCompromise marks sessions as compromised
func (cm *ChaosMonkey) SimulateSessionCompromise(ctx context.Context) error {
	// Randomly invalidate some sessions to test recovery
	_, err := cm.db.ExecContext(ctx, `
		UPDATE sessions SET is_revoked = true
		WHERE random() < 0.01 AND is_revoked = false
	`)
	return err
}

// ============================================
// GAME DAY SCENARIOS
// Full-scale security exercises
// ============================================

// GameDayScenario defines a security exercise
type GameDayScenario struct {
	ID          string
	Name        string
	Description string
	Steps       []GameDayStep
	Duration    time.Duration
}

// GameDayStep is one step in a game day
type GameDayStep struct {
	Order       int
	Action      string
	Description string
	Delay       time.Duration
}

// Predefined game day scenarios
var GameDayScenarios = []GameDayScenario{
	{
		ID:          "key_compromise",
		Name:        "Key Compromise Response",
		Description: "Simulate detection and response to a key compromise",
		Duration:    1 * time.Hour,
		Steps: []GameDayStep{
			{Order: 1, Action: "detect_anomaly", Description: "IDS detects unusual key access patterns"},
			{Order: 2, Action: "alert_security", Description: "Security team notified", Delay: 1 * time.Minute},
			{Order: 3, Action: "assess_impact", Description: "Determine scope of compromise", Delay: 5 * time.Minute},
			{Order: 4, Action: "rotate_keys", Description: "Force rotation of affected keys", Delay: 10 * time.Minute},
			{Order: 5, Action: "notify_users", Description: "Alert affected users", Delay: 15 * time.Minute},
			{Order: 6, Action: "post_mortem", Description: "Document lessons learned", Delay: 30 * time.Minute},
		},
	},
	{
		ID:          "mass_brute_force",
		Name:        "Coordinated Brute Force Attack",
		Description: "Respond to distributed brute force attack",
		Duration:    2 * time.Hour,
		Steps: []GameDayStep{
			{Order: 1, Action: "detect_attack", Description: "IDS detects elevated failed logins"},
			{Order: 2, Action: "activate_defenses", Description: "Increase rate limiting", Delay: 30 * time.Second},
			{Order: 3, Action: "block_ips", Description: "Block attacking IP ranges", Delay: 2 * time.Minute},
			{Order: 4, Action: "enable_captcha", Description: "Enable CAPTCHA for logins", Delay: 5 * time.Minute},
			{Order: 5, Action: "notify_affected", Description: "Warn users whose accounts were targeted", Delay: 30 * time.Minute},
		},
	},
	{
		ID:          "data_exfiltration",
		Name:        "Data Exfiltration Attempt",
		Description: "Detect and respond to data theft attempt",
		Duration:    1 * time.Hour,
		Steps: []GameDayStep{
			{Order: 1, Action: "detect_bulk_access", Description: "Anomaly detector flags bulk data access"},
			{Order: 2, Action: "isolate_session", Description: "Kill suspicious sessions", Delay: 1 * time.Minute},
			{Order: 3, Action: "forensic_capture", Description: "Capture session data for analysis", Delay: 2 * time.Minute},
			{Order: 4, Action: "assess_data_accessed", Description: "Determine what was accessed", Delay: 10 * time.Minute},
			{Order: 5, Action: "notify_if_required", Description: "Breach notification if PII exposed", Delay: 30 * time.Minute},
		},
	},
}

// RunGameDay executes a game day scenario
func (cm *ChaosMonkey) RunGameDay(ctx context.Context, scenarioID string) error {
	var scenario *GameDayScenario
	for _, s := range GameDayScenarios {
		if s.ID == scenarioID {
			scenario = &s
			break
		}
	}

	if scenario == nil {
		return fmt.Errorf("scenario not found: %s", scenarioID)
	}

	fmt.Printf("🎮 Starting Game Day: %s\n", scenario.Name)
	fmt.Printf("Duration: %s\n", scenario.Duration)

	for _, step := range scenario.Steps {
		if step.Delay > 0 {
			fmt.Printf("⏳ Waiting %s before next step...\n", step.Delay)
			time.Sleep(step.Delay)
		}

		fmt.Printf("📋 Step %d: %s\n", step.Order, step.Description)

		// Execute step action (would integrate with real systems)
		cm.executeGameDayStep(ctx, step)
	}

	fmt.Println("✅ Game Day complete!")
	return nil
}

func (cm *ChaosMonkey) executeGameDayStep(_ context.Context, step GameDayStep) {
	// In production, this would trigger actual runbooks/playbooks
	switch step.Action {
	case "detect_anomaly", "detect_attack", "detect_bulk_access":
		// Trigger IDS alert
	case "rotate_keys":
		// Force key rotation
	case "block_ips":
		// Update firewall rules
	case "notify_users", "notify_affected":
		// Send notifications
	}
}

// ============================================
// DEFAULT EXPERIMENTS
// ============================================

// GetDefaultExperiments returns pre-configured experiments
func GetDefaultExperiments() []*ChaosExperiment {
	return []*ChaosExperiment{
		{
			ID:          "latency",
			Name:        "Network Latency",
			Description: "Add random latency to requests",
			Type:        ChaosTypeLatency,
			Duration:    5 * time.Minute,
			Probability: 0.1,
		},
		{
			ID:          "packet_loss",
			Name:        "Packet Loss",
			Description: "Randomly drop requests",
			Type:        ChaosTypePacketLoss,
			Duration:    2 * time.Minute,
			Probability: 0.05,
		},
		{
			ID:          "slow_response",
			Name:        "Slow Responses",
			Description: "Slow down response writing",
			Type:        ChaosTypeSlowResponse,
			Duration:    5 * time.Minute,
			Probability: 0.1,
		},
		{
			ID:          "fake_attack",
			Name:        "Simulated Attack",
			Description: "Generate attack-like traffic",
			Type:        ChaosTypeFakeAttack,
			Duration:    10 * time.Minute,
			Probability: 1.0,
		},
	}
}
