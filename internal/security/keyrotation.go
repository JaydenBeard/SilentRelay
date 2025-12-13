package security

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"sync"
	"time"

	"github.com/jaydenbeard/messaging-app/internal/config"
)

// KeyRotationScheduler handles automatic JWT key rotation
type KeyRotationScheduler struct {
	ctx            context.Context
	cancelFunc     context.CancelFunc
	rotationTicker *time.Ticker
	rotationLock   sync.Mutex
	logger         *log.Logger
	enabled        bool
}

// NewKeyRotationScheduler creates a new key rotation scheduler
func NewKeyRotationScheduler() *KeyRotationScheduler {
	return &KeyRotationScheduler{
		logger:  log.New(os.Stdout, "[KEY-ROTATION] ", log.Ldate|log.Ltime|log.LUTC),
		enabled: true,
	}
}

// Start begins the automatic key rotation scheduler
func (krs *KeyRotationScheduler) Start() {
	krs.rotationLock.Lock()
	defer krs.rotationLock.Unlock()

	if krs.enabled {
		krs.logger.Println("Starting key rotation scheduler")

		// Create context for scheduler
		krs.ctx, krs.cancelFunc = context.WithCancel(context.Background())

		// Start with initial rotation check
		go krs.runRotationScheduler()
	} else {
		krs.logger.Println("Key rotation scheduler is disabled")
	}
}

// Stop stops the automatic key rotation scheduler
func (krs *KeyRotationScheduler) Stop() {
	krs.rotationLock.Lock()
	defer krs.rotationLock.Unlock()

	if krs.cancelFunc != nil {
		krs.cancelFunc()
		krs.logger.Println("Key rotation scheduler stopped")
	}

	if krs.rotationTicker != nil {
		krs.rotationTicker.Stop()
	}
}

// Enable enables the key rotation scheduler
func (krs *KeyRotationScheduler) Enable() {
	krs.rotationLock.Lock()
	defer krs.rotationLock.Unlock()

	krs.enabled = true
	krs.logger.Println("Key rotation scheduler enabled")
}

// Disable disables the key rotation scheduler
func (krs *KeyRotationScheduler) Disable() {
	krs.rotationLock.Lock()
	defer krs.rotationLock.Unlock()

	krs.enabled = false
	krs.logger.Println("Key rotation scheduler disabled")

	// Stop any running scheduler
	if krs.cancelFunc != nil {
		krs.cancelFunc()
	}
	if krs.rotationTicker != nil {
		krs.rotationTicker.Stop()
	}
}

// SetRotationInterval sets the rotation interval
func (krs *KeyRotationScheduler) SetRotationInterval(interval time.Duration) {
	krs.rotationLock.Lock()
	defer krs.rotationLock.Unlock()

	config.SetRotationInterval(interval)
	krs.logger.Printf("Rotation interval set to: %v", interval)

	// Restart scheduler with new interval if running
	if krs.enabled && krs.ctx != nil {
		krs.restartScheduler()
	}
}

// runRotationScheduler runs the main rotation scheduling loop
func (krs *KeyRotationScheduler) runRotationScheduler() {
	// Initial check for immediate rotation if needed
	krs.checkAndRotateIfNeeded()

	// Set up ticker for periodic checks
	_, rotationInterval := config.GetRotationInfo()
	checkInterval := rotationInterval / 4 // Check 4 times per rotation interval
	if checkInterval < 1*time.Hour {
		checkInterval = 1 * time.Hour // Minimum check interval
	}

	krs.rotationTicker = time.NewTicker(checkInterval)
	krs.logger.Printf("Key rotation scheduler running with check interval: %v", checkInterval)

	for {
		select {
		case <-krs.rotationTicker.C:
			krs.checkAndRotateIfNeeded()
		case <-krs.ctx.Done():
			krs.logger.Println("Key rotation scheduler stopped")
			return
		}
	}
}

// checkAndRotateIfNeeded checks if rotation should occur and performs it if needed
func (krs *KeyRotationScheduler) checkAndRotateIfNeeded() {
	if !krs.enabled {
		return
	}

	// Check if rotation should occur
	if config.ShouldRotate() {
		krs.logger.Println("Automatic rotation condition met - initiating key rotation")

		// Generate new secure secret
		newSecret, err := generateSecureJWTSecret()
		if err != nil {
			krs.logger.Printf("ERROR: Failed to generate new JWT secret: %v", err)
			return
		}

		// Perform rotation
		err = config.RotateSecret(newSecret)
		if err != nil {
			krs.logger.Printf("ERROR: Failed to rotate JWT secret: %v", err)
			return
		}

		krs.logger.Println("Automatic key rotation completed successfully")
	} else {
		// Log rotation status
		lastRotation, interval := config.GetRotationInfo()
		timeSinceLast := time.Since(lastRotation)
		krs.logger.Printf("Rotation check: %v since last rotation, next rotation in %v",
			timeSinceLast, interval-timeSinceLast)
	}
}

// restartScheduler restarts the scheduler with updated interval
func (krs *KeyRotationScheduler) restartScheduler() {
	krs.rotationLock.Lock()
	defer krs.rotationLock.Unlock()

	if krs.rotationTicker != nil {
		krs.rotationTicker.Stop()
	}

	// Restart the scheduler
	go krs.runRotationScheduler()
}

// generateSecureJWTSecret generates a cryptographically secure JWT secret
func generateSecureJWTSecret() (string, error) {
	// Generate 64 random bytes (512 bits) for high security
	randomBytes := make([]byte, 64)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	// Encode as hex string
	secret := hex.EncodeToString(randomBytes)

	// Validate the generated secret
	err = config.ValidateJWTSecret(secret)
	if err != nil {
		return "", err
	}

	return secret, nil
}

// GenerateSecureJWTSecret exports the secure secret generation for external use
func GenerateSecureJWTSecret() (string, error) {
	return generateSecureJWTSecret()
}

// ForceImmediateRotation forces an immediate key rotation
func (krs *KeyRotationScheduler) ForceImmediateRotation() error {
	if !krs.enabled {
		return nil
	}

	krs.logger.Println("Forcing immediate key rotation")

	// Generate new secure secret
	newSecret, err := generateSecureJWTSecret()
	if err != nil {
		krs.logger.Printf("ERROR: Failed to generate new JWT secret: %v", err)
		return err
	}

	// Perform rotation
	err = config.RotateSecret(newSecret)
	if err != nil {
		krs.logger.Printf("ERROR: Failed to rotate JWT secret: %v", err)
		return err
	}

	krs.logger.Println("Immediate key rotation completed successfully")
	return nil
}

// GetRotationStatus returns the current rotation status
func (krs *KeyRotationScheduler) GetRotationStatus() (enabled bool, lastRotation time.Time, interval time.Duration) {
	lastRotation, interval = config.GetRotationInfo()
	return krs.enabled, lastRotation, interval
}
