package security

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// IdentityKeyRotationManager handles automatic rotation of Signal Protocol identity keys
// This provides forward secrecy and prevents long-term key compromise
type IdentityKeyRotationManager struct {
	ctx                 context.Context
	cancelFunc          context.CancelFunc
	rotationTicker      *time.Ticker
	rotationLock        sync.RWMutex
	logger              *log.Logger
	enabled             bool
	rotationInterval    time.Duration
	lastRotationTime    time.Time
	identityKeyStore    IdentityKeyStore
	compromiseDetection CompromiseDetector
}

// IdentityKeyStore interface for storing and retrieving identity keys
type IdentityKeyStore interface {
	GetIdentityKey(userID string) (*IdentityKeyPair, error)
	StoreIdentityKey(userID string, keyPair *IdentityKeyPair) error
	InvalidateOldKeys(userID string, keepCurrent bool) error
}

// CompromiseDetector interface for detecting key compromise
type CompromiseDetector interface {
	IsKeyCompromised(userID string, keyPair *IdentityKeyPair) (bool, error)
	ReportCompromise(userID string, keyPair *IdentityKeyPair) error
}

// NewIdentityKeyRotationManager creates a new identity key rotation manager
func NewIdentityKeyRotationManager(store IdentityKeyStore, detector CompromiseDetector) *IdentityKeyRotationManager {
	return &IdentityKeyRotationManager{
		logger:              log.New(os.Stdout, "[IDENTITY-KEY-ROTATION] ", log.Ldate|log.Ltime|log.LUTC),
		enabled:             true,
		rotationInterval:    30 * 24 * time.Hour, // Default: 30 days
		identityKeyStore:    store,
		compromiseDetection: detector,
	}
}

// Start begins the automatic identity key rotation scheduler
func (ikrm *IdentityKeyRotationManager) Start() {
	ikrm.rotationLock.Lock()
	defer ikrm.rotationLock.Unlock()

	if ikrm.enabled {
		ikrm.logger.Println("Starting identity key rotation scheduler")

		// Create context for scheduler
		ikrm.ctx, ikrm.cancelFunc = context.WithCancel(context.Background())

		// Start with initial rotation check
		go ikrm.runRotationScheduler()
	} else {
		ikrm.logger.Println("Identity key rotation scheduler is disabled")
	}
}

// Stop stops the automatic identity key rotation scheduler
func (ikrm *IdentityKeyRotationManager) Stop() {
	ikrm.rotationLock.Lock()
	defer ikrm.rotationLock.Unlock()

	if ikrm.cancelFunc != nil {
		ikrm.cancelFunc()
		ikrm.logger.Println("Identity key rotation scheduler stopped")
	}

	if ikrm.rotationTicker != nil {
		ikrm.rotationTicker.Stop()
	}
}

// Enable enables the identity key rotation scheduler
func (ikrm *IdentityKeyRotationManager) Enable() {
	ikrm.rotationLock.Lock()
	defer ikrm.rotationLock.Unlock()

	ikrm.enabled = true
	ikrm.logger.Println("Identity key rotation scheduler enabled")
}

// Disable disables the identity key rotation scheduler
func (ikrm *IdentityKeyRotationManager) Disable() {
	ikrm.rotationLock.Lock()
	defer ikrm.rotationLock.Unlock()

	ikrm.enabled = false
	ikrm.logger.Println("Identity key rotation scheduler disabled")

	// Stop any running scheduler
	if ikrm.cancelFunc != nil {
		ikrm.cancelFunc()
	}
	if ikrm.rotationTicker != nil {
		ikrm.rotationTicker.Stop()
	}
}

// SetRotationInterval sets the rotation interval for identity keys
func (ikrm *IdentityKeyRotationManager) SetRotationInterval(interval time.Duration) {
	ikrm.rotationLock.Lock()
	defer ikrm.rotationLock.Unlock()

	if interval < 24*time.Hour { // Minimum 24 hours for identity keys
		ikrm.logger.Printf("Warning: Identity key rotation interval %v is too short, using minimum 24 hours", interval)
		interval = 24 * time.Hour
	}

	ikrm.rotationInterval = interval
	ikrm.logger.Printf("Identity key rotation interval set to: %v", interval)

	// Restart scheduler with new interval if running
	if ikrm.enabled && ikrm.ctx != nil {
		ikrm.restartScheduler()
	}
}

// runRotationScheduler runs the main rotation scheduling loop
func (ikrm *IdentityKeyRotationManager) runRotationScheduler() {
	// Initial check for immediate rotation if needed
	ikrm.checkAndRotateIfNeeded()

	// Set up ticker for periodic checks (daily checks)
	checkInterval := 24 * time.Hour
	if checkInterval < 1*time.Hour {
		checkInterval = 1 * time.Hour // Minimum check interval
	}

	ikrm.rotationTicker = time.NewTicker(checkInterval)
	ikrm.logger.Printf("Identity key rotation scheduler running with check interval: %v", checkInterval)

	for {
		select {
		case <-ikrm.rotationTicker.C:
			ikrm.checkAndRotateIfNeeded()
		case <-ikrm.ctx.Done():
			ikrm.logger.Println("Identity key rotation scheduler stopped")
			return
		}
	}
}

// checkAndRotateIfNeeded checks if identity key rotation should occur and performs it if needed
func (ikrm *IdentityKeyRotationManager) checkAndRotateIfNeeded() {
	if !ikrm.enabled {
		return
	}

	ikrm.rotationLock.RLock()
	shouldRotate := time.Since(ikrm.lastRotationTime) >= ikrm.rotationInterval
	ikrm.rotationLock.RUnlock()

	if shouldRotate {
		ikrm.logger.Println("Automatic identity key rotation condition met - initiating rotation")

		// Perform rotation for all users
		err := ikrm.RotateAllIdentityKeys()
		if err != nil {
			ikrm.logger.Printf("ERROR: Failed to rotate identity keys: %v", err)
			return
		}

		ikrm.rotationLock.Lock()
		ikrm.lastRotationTime = time.Now()
		ikrm.rotationLock.Unlock()

		ikrm.logger.Println("Automatic identity key rotation completed successfully")
	} else {
		// Log rotation status
		timeSinceLast := time.Since(ikrm.lastRotationTime)
		ikrm.logger.Printf("Identity key rotation check: %v since last rotation, next rotation in %v",
			timeSinceLast, ikrm.rotationInterval-timeSinceLast)
	}
}

// restartScheduler restarts the scheduler with updated interval
func (ikrm *IdentityKeyRotationManager) restartScheduler() {
	ikrm.rotationLock.Lock()
	defer ikrm.rotationLock.Unlock()

	if ikrm.rotationTicker != nil {
		ikrm.rotationTicker.Stop()
	}

	// Restart the scheduler
	go ikrm.runRotationScheduler()
}

// RotateAllIdentityKeys rotates identity keys for all users
func (ikrm *IdentityKeyRotationManager) RotateAllIdentityKeys() error {
	ikrm.logger.Println("Starting identity key rotation for all users")

	// In a real implementation, this would iterate through all users
	// For now, we'll implement the core rotation logic

	// Get all user IDs from database (placeholder)
	userIDs := []string{"user1", "user2", "user3"} // This would come from actual user database

	for _, userID := range userIDs {
		err := ikrm.RotateUserIdentityKey(userID)
		if err != nil {
			ikrm.logger.Printf("ERROR: Failed to rotate identity key for user %s: %v", userID, err)
			// Continue with other users even if one fails
			continue
		}
	}

	ikrm.logger.Println("Identity key rotation completed for all users")
	return nil
}

// RotateUserIdentityKey rotates the identity key for a specific user
func (ikrm *IdentityKeyRotationManager) RotateUserIdentityKey(userID string) error {
	ikrm.logger.Printf("Rotating identity key for user: %s", userID)

	// Check if current key is compromised
	currentKeyPair, err := ikrm.identityKeyStore.GetIdentityKey(userID)
	if err != nil {
		// If key doesn't exist, this might be a new user - proceed with rotation
		if err.Error() != "identity key not found" {
			return fmt.Errorf("failed to get current identity key: %w", err)
		}
		ikrm.logger.Printf("No existing identity key found for user %s - creating new one", userID)
	}

	// Check for compromise
	if currentKeyPair != nil {
		compromised, err := ikrm.compromiseDetection.IsKeyCompromised(userID, currentKeyPair)
		if err != nil {
			ikrm.logger.Printf("WARNING: Failed to check for key compromise for user %s: %v", userID, err)
			// Continue with rotation even if compromise check fails
		} else if compromised {
			ikrm.logger.Printf("SECURITY ALERT: Identity key for user %s is compromised - forcing immediate rotation", userID)
			// Report the compromise
			reportErr := ikrm.compromiseDetection.ReportCompromise(userID, currentKeyPair)
			if reportErr != nil {
				ikrm.logger.Printf("WARNING: Failed to report compromise for user %s: %v", userID, reportErr)
			}
		}
	}

	// Generate new identity key pair with retry logic
	var newKeyPair *KeyPair
	var generateErr error
	for i := 0; i < 3; i++ {
		newKeyPair, generateErr = NewSignalProtocol().GenerateKeyPair()
		if generateErr == nil {
			break
		}
		ikrm.logger.Printf("WARNING: Attempt %d failed to generate identity key: %v", i+1, generateErr)
		time.Sleep(100 * time.Millisecond) // Small delay before retry
	}
	if generateErr != nil {
		return fmt.Errorf("failed to generate new identity key after 3 attempts: %w", generateErr)
	}

	// Validate the new key
	if newKeyPair.PrivateKey == [32]byte{} || newKeyPair.PublicKey == [32]byte{} {
		return errors.New("generated identity key is invalid (empty)")
	}

	// Create identity key pair
	newIdentityKeyPair := &IdentityKeyPair{
		KeyPair: *newKeyPair,
	}

	// Store the new identity key with retry logic
	var storeErr error
	for i := 0; i < 3; i++ {
		storeErr = ikrm.identityKeyStore.StoreIdentityKey(userID, newIdentityKeyPair)
		if storeErr == nil {
			break
		}
		ikrm.logger.Printf("WARNING: Attempt %d failed to store identity key: %v", i+1, storeErr)
		time.Sleep(100 * time.Millisecond) // Small delay before retry
	}
	if storeErr != nil {
		return fmt.Errorf("failed to store new identity key after 3 attempts: %w", storeErr)
	}

	// Invalidate old keys (keep current for transition period)
	invalidErr := ikrm.identityKeyStore.InvalidateOldKeys(userID, true)
	if invalidErr != nil {
		ikrm.logger.Printf("WARNING: Failed to invalidate old keys for user %s: %v", userID, invalidErr)
		// This is not a critical failure - new key is still stored
	}

	ikrm.logger.Printf("Successfully rotated identity key for user: %s", userID)
	return nil
}

// ForceImmediateRotation forces an immediate identity key rotation for a specific user
func (ikrm *IdentityKeyRotationManager) ForceImmediateRotation(userID string) error {
	if !ikrm.enabled {
		return nil
	}

	ikrm.logger.Printf("Forcing immediate identity key rotation for user: %s", userID)

	// Perform rotation for the specific user
	err := ikrm.RotateUserIdentityKey(userID)
	if err != nil {
		ikrm.logger.Printf("ERROR: Failed to force rotate identity key for user %s: %v", userID, err)
		return err
	}

	ikrm.logger.Printf("Immediate identity key rotation completed successfully for user: %s", userID)
	return nil
}

// ForceEmergencyRotation forces emergency rotation for all users due to compromise
func (ikrm *IdentityKeyRotationManager) ForceEmergencyRotation() error {
	if !ikrm.enabled {
		return nil
	}

	ikrm.logger.Println("FORCING EMERGENCY IDENTITY KEY ROTATION FOR ALL USERS")

	// Perform rotation for all users
	err := ikrm.RotateAllIdentityKeys()
	if err != nil {
		ikrm.logger.Printf("ERROR: Failed to perform emergency rotation: %v", err)
		return err
	}

	ikrm.rotationLock.Lock()
	ikrm.lastRotationTime = time.Now()
	ikrm.rotationLock.Unlock()

	ikrm.logger.Println("EMERGENCY IDENTITY KEY ROTATION COMPLETED")
	return nil
}

// GetRotationStatus returns the current rotation status
func (ikrm *IdentityKeyRotationManager) GetRotationStatus() (enabled bool, lastRotation time.Time, interval time.Duration) {
	ikrm.rotationLock.RLock()
	defer ikrm.rotationLock.RUnlock()

	return ikrm.enabled, ikrm.lastRotationTime, ikrm.rotationInterval
}

// GenerateSecureIdentityKey generates a cryptographically secure identity key
func GenerateSecureIdentityKey() (*IdentityKeyPair, error) {
	// Generate new key pair
	keyPair, err := NewSignalProtocol().GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate identity key: %w", err)
	}

	return &IdentityKeyPair{
		KeyPair: *keyPair,
	}, nil
}

// IdentityKeyRotationConfig holds configuration for identity key rotation
type IdentityKeyRotationConfig struct {
	Enabled           bool
	RotationInterval  time.Duration
	EmergencyRotation bool
}

// SimpleIdentityKeyStore is a simple in-memory implementation for testing
type SimpleIdentityKeyStore struct {
	store map[string]*IdentityKeyPair
	lock  sync.RWMutex
}

func NewSimpleIdentityKeyStore() *SimpleIdentityKeyStore {
	return &SimpleIdentityKeyStore{
		store: make(map[string]*IdentityKeyPair),
	}
}

func (s *SimpleIdentityKeyStore) GetIdentityKey(userID string) (*IdentityKeyPair, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	keyPair, exists := s.store[userID]
	if !exists {
		return nil, fmt.Errorf("identity key not found for user %s", userID)
	}

	return keyPair, nil
}

func (s *SimpleIdentityKeyStore) StoreIdentityKey(userID string, keyPair *IdentityKeyPair) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.store[userID] = keyPair
	return nil
}

func (s *SimpleIdentityKeyStore) InvalidateOldKeys(userID string, keepCurrent bool) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if keepCurrent {
		// In a real implementation, we would keep the current key and invalidate previous ones
		// For this simple store, we just keep the current key
	} else {
		delete(s.store, userID)
	}

	return nil
}

// SimpleCompromiseDetector is a simple implementation for testing
type SimpleCompromiseDetector struct{}

func (s *SimpleCompromiseDetector) IsKeyCompromised(userID string, keyPair *IdentityKeyPair) (bool, error) {
	// In a real implementation, this would check against known compromised keys
	// For now, we return false (not compromised)
	return false, nil
}

func (s *SimpleCompromiseDetector) ReportCompromise(userID string, keyPair *IdentityKeyPair) error {
	// In a real implementation, this would report to security systems
	return nil
}

// GetRotationInterval returns the rotation interval for testing
func (ikrm *IdentityKeyRotationManager) GetRotationInterval() time.Duration {
	ikrm.rotationLock.RLock()
	defer ikrm.rotationLock.RUnlock()
	return ikrm.rotationInterval
}
