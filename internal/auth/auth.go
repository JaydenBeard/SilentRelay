package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"

	// crypto/sha1 removed - using SHA-256 for TOTP (RFC 6238 compliant)
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jaydenbeard/messaging-app/internal/config"
	"github.com/jaydenbeard/messaging-app/internal/db"
	"github.com/jaydenbeard/messaging-app/internal/security"
	"github.com/jaydenbeard/messaging-app/internal/sms"
	"github.com/redis/go-redis/v9"
)

// Security errors
var (
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
	ErrInvalidCode        = errors.New("invalid verification code")
	ErrUserNotFound       = errors.New("user not found")
	ErrRateLimited        = errors.New("too many requests")
	ErrJWTSecretEmpty     = errors.New("JWT secret is empty or invalid")
	ErrJWTSecretWeak      = errors.New("JWT secret is too weak for security requirements")
	ErrTokenBlacklisted   = errors.New("token has been blacklisted due to security concerns")
	ErrSessionFixation    = errors.New("session fixation attempt detected")
	ErrTokenCompromised   = errors.New("token appears to be compromised")
	ErrBlacklistOperation = errors.New("failed to update token blacklist")
)

// AuthService handles authentication with secure JWT secret management
type AuthService struct {
	db                *db.PostgresDB
	smsService        *sms.ClickSendService
	jwtSecret         []byte
	previousJWTSecret []byte
	secretLock        sync.RWMutex // Thread-safe access to JWT secret
	rotationLogger    *log.Logger
	redisClient       *redis.Client
	blacklistLock     sync.RWMutex // Thread-safe access to blacklist operations
	securityLogger    *log.Logger
}

// Claims represents JWT claims
type Claims struct {
	UserID   uuid.UUID `json:"user_id"`
	DeviceID uuid.UUID `json:"device_id"`
	jwt.RegisteredClaims
}

// NewAuthService creates a new auth service with secure JWT secret validation
func NewAuthService(database *db.PostgresDB, jwtSecret string) (*AuthService, error) {
	// Validate JWT secret security requirements
	if jwtSecret == "" {
		return nil, ErrJWTSecretEmpty
	}

	if len(jwtSecret) < 32 {
		return nil, ErrJWTSecretWeak
	}

	// Validate cryptographic strength
	if !validateJWTSecretStrength(jwtSecret) {
		return nil, ErrJWTSecretWeak
	}

	// Check environment mode for fail-fast behavior
	nodeEnv := os.Getenv("NODE_ENV")

	// Initialize SMS service
	smsService, err := sms.NewClickSendService()
	if err != nil {
		if nodeEnv == "production" {
			return nil, fmt.Errorf("failed to initialize SMS service in production: %w", err)
		}
		log.Printf("Warning: Failed to initialize SMS service: %v", err)
		log.Printf("SMS verification codes will not be sent - check ClickSend configuration")
		// Don't fail auth service creation, just log the warning
	} else {
		log.Printf("SMS service initialized successfully with ClickSend")
		// Perform health check to verify service is operational
		if err := smsService.HealthCheck(); err != nil {
			if nodeEnv == "production" {
				return nil, fmt.Errorf("SMS service health check failed in production: %w", err)
			}
			log.Printf("Warning: SMS service health check failed: %v", err)
			log.Printf("SMS service may not be fully operational - check ClickSend account status")
		} else {
			log.Printf("SMS service health check passed - service is operational")
		}
	}

	// Initialize Redis client for token blacklisting
	// Use REDIS_URL for consistency with docker-compose, fallback to REDIS_ADDR for legacy support
	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = os.Getenv("REDIS_ADDR")
	}
	if redisAddr == "" {
		redisAddr = "localhost:6379" // Default fallback
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0, // Use default DB
	})

	// Test Redis connection
	_, err = redisClient.Ping(context.Background()).Result()
	if err != nil {
		if nodeEnv == "production" {
			return nil, fmt.Errorf("failed to connect to Redis in production: %w", err)
		}
		log.Printf("Warning: Failed to connect to Redis: %v", err)
		log.Printf("Token blacklisting will use fallback in-memory cache")
	}

	// Get previous secret if available for dual-key support
	currentSecret, previousSecret, hasPrevious := config.GetAllActiveSecrets()
	if !hasPrevious {
		previousSecret = ""
	}

	return &AuthService{
		db:                database,
		smsService:        smsService,
		jwtSecret:         []byte(currentSecret),
		previousJWTSecret: []byte(previousSecret),
		rotationLogger:    log.New(os.Stdout, "[AUTH-ROTATION] ", log.Ldate|log.Ltime|log.LUTC),
		redisClient:       redisClient,
		securityLogger:    log.New(os.Stdout, "[AUTH-SECURITY] ", log.Ldate|log.Ltime|log.LUTC),
	}, nil
}

// validateJWTSecretStrength checks if JWT secret meets cryptographic requirements
func validateJWTSecretStrength(secret string) bool {
	// Check for sufficient entropy
	entropy := 0.0
	charCount := make(map[rune]int)

	for _, char := range secret {
		charCount[char]++
	}

	// Calculate Shannon entropy
	for _, count := range charCount {
		probability := float64(count) / float64(len(secret))
		entropy -= probability * math.Log2(probability)
	}

	// Require minimum entropy of 3.5 bits per character
	return entropy >= 3.5
}

// GetJWTSecret provides thread-safe access to JWT secret (exported for testing)
func (a *AuthService) GetJWTSecret() []byte {
	a.secretLock.RLock()
	defer a.secretLock.RUnlock()
	return a.jwtSecret
}

// GetPreviousJWTSecret provides thread-safe access to previous JWT secret for dual-key validation
func (a *AuthService) GetPreviousJWTSecret() []byte {
	a.secretLock.RLock()
	defer a.secretLock.RUnlock()
	return a.previousJWTSecret
}

// GetAllJWTSecrets returns both current and previous secrets for comprehensive validation
func (a *AuthService) GetAllJWTSecrets() (current, previous []byte) {
	a.secretLock.RLock()
	defer a.secretLock.RUnlock()
	return a.jwtSecret, a.previousJWTSecret
}

// RotateJWTSecret securely rotates the JWT secret with zero-downtime transition
func (a *AuthService) RotateJWTSecret(newSecret string) error {
	// Validate new secret
	if newSecret == "" {
		return ErrJWTSecretEmpty
	}

	if len(newSecret) < 32 {
		return ErrJWTSecretWeak
	}

	if !validateJWTSecretStrength(newSecret) {
		return ErrJWTSecretWeak
	}

	// Thread-safe rotation
	a.secretLock.Lock()
	defer a.secretLock.Unlock()

	// Log rotation event
	a.rotationLogger.Printf("Starting JWT secret rotation in AuthService")

	// Store current secret as previous for transition period
	a.previousJWTSecret = a.jwtSecret
	a.jwtSecret = []byte(newSecret)

	// Update global key manager
	if err := config.RotateSecret(newSecret); err != nil {
		a.rotationLogger.Printf("Warning: Failed to update global key manager: %v", err)
		// Don't fail the rotation, just log the warning
	}

	a.rotationLogger.Printf("JWT secret rotation completed - dual-key validation enabled")
	a.rotationLogger.Printf("Transition period: both old and new keys will be accepted for validation")

	return nil
}

// RequestVerificationCode generates and stores a verification code
func (a *AuthService) RequestVerificationCode(phoneNumber string) (string, error) {
	// Validate phone number format
	if !security.ValidatePhoneNumber(phoneNumber) {
		return "", fmt.Errorf("invalid phone number format")
	}

	// Generate 6-digit code
	code, err := generateCode(6)
	if err != nil {
		return "", err
	}

	// Store code with 5 minute expiry
	expiresAt := time.Now().Add(5 * time.Minute)
	if err := a.db.SaveVerificationCode(phoneNumber, code, expiresAt); err != nil {
		return "", err
	}

	// Send SMS via ClickSend if service is available AND not in DEV_MODE
	devMode := os.Getenv("DEV_MODE") == "true"
	if devMode {
		log.Printf("DEV_MODE: Skipping SMS send to %s - use code returned in API response", phoneNumber)
	} else if a.smsService != nil {
		if err := a.smsService.SendVerificationCode(phoneNumber, code); err != nil {
			log.Printf("Failed to send SMS verification code to %s: %v", phoneNumber, err)
			// Don't fail the request, just log the error
			// User can still get the code via other means if needed
		} else {
			log.Printf("SMS verification code sent successfully to %s", phoneNumber)
		}
	} else {
		log.Printf("SMS service not configured - verification code not sent to %s", phoneNumber)
	}

	// Return the code (for development/testing purposes)
	// In production, you might want to remove this and only send via SMS
	return code, nil
}

// CheckCode validates a verification code without marking it as verified (for pre-checking)
func (a *AuthService) CheckCode(phoneNumber, code string) (bool, error) {
	return a.db.CheckCode(phoneNumber, code)
}

// VerifyCode validates a verification code and marks it as verified
func (a *AuthService) VerifyCode(phoneNumber, code string) (bool, error) {
	return a.db.VerifyCode(phoneNumber, code)
}

// MarkCodeVerified marks a code as verified after successful user creation
func (a *AuthService) MarkCodeVerified(phoneNumber, code string) error {
	return a.db.MarkCodeVerified(phoneNumber, code)
}

// GenerateTokens creates JWT access and refresh tokens
func (a *AuthService) GenerateTokens(userID, deviceID uuid.UUID) (accessToken, refreshToken string, expiresAt time.Time, err error) {
	// Access token - 1 hour
	accessExpiry := time.Now().Add(1 * time.Hour)
	accessClaims := &Claims{
		UserID:   userID,
		DeviceID: deviceID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID.String(),
		},
	}

	accessTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err = accessTokenObj.SignedString(a.GetJWTSecret())
	if err != nil {
		return "", "", time.Time{}, err
	}

	// Refresh token - 30 days
	refreshExpiry := time.Now().Add(30 * 24 * time.Hour)
	refreshClaims := &Claims{
		UserID:   userID,
		DeviceID: deviceID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID.String(),
		},
	}

	refreshTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err = refreshTokenObj.SignedString(a.GetJWTSecret())
	if err != nil {
		return "", "", time.Time{}, err
	}

	// Store session
	tokenHash := hashToken(accessToken)
	if _, err := a.db.CreateSession(userID, tokenHash, accessExpiry); err != nil {
		log.Printf("Warning: Failed to create session: %v", err)
	}

	return accessToken, refreshToken, accessExpiry, nil
}

// ValidateToken validates a JWT token and returns claims with dual-key support
func (a *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	// Try validation with current secret first
	token, err := a.validateTokenWithSecret(tokenString, a.GetJWTSecret())
	if err == nil {
		return token, nil
	}

	// If current secret fails and we have a previous secret, try with previous secret
	if a.hasPreviousSecret() {
		// Log with hash fingerprint instead of actual token content for security
		tokenFingerprint := hashTokenForBlacklist(tokenString)[:8]
		a.rotationLogger.Printf("Attempting validation with previous JWT secret for token fingerprint: %s...", tokenFingerprint)
		token, err = a.validateTokenWithSecret(tokenString, a.GetPreviousJWTSecret())
		if err == nil {
			a.rotationLogger.Printf("Token validated successfully with previous secret - transition period active")
			return token, nil
		}
	}

	// If both fail, return the appropriate error
	if errors.Is(err, jwt.ErrTokenExpired) {
		return nil, ErrTokenExpired
	}
	return nil, ErrInvalidToken
}

// validateTokenWithSecret validates a JWT token using a specific secret
func (a *AuthService) validateTokenWithSecret(tokenString string, secret []byte) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// hasPreviousSecret checks if a previous secret is available for dual-key validation
func (a *AuthService) hasPreviousSecret() bool {
	a.secretLock.RLock()
	defer a.secretLock.RUnlock()
	return len(a.previousJWTSecret) > 0
}

// RefreshAccessToken generates a new access token from a refresh token
func (a *AuthService) RefreshAccessToken(refreshTokenString string) (accessToken string, expiresAt time.Time, err error) {
	claims, err := a.ValidateToken(refreshTokenString)
	if err != nil {
		return "", time.Time{}, err
	}

	// SECURITY: Verify device is still active and belongs to user
	isActive, err := a.db.IsDeviceActive(claims.UserID, claims.DeviceID)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to verify device status: %w", err)
	}
	if !isActive {
		return "", time.Time{}, fmt.Errorf("device is not active or not found")
	}

	// Generate new access token - always use current secret for new tokens
	accessExpiry := time.Now().Add(1 * time.Hour)
	accessClaims := &Claims{
		UserID:   claims.UserID,
		DeviceID: claims.DeviceID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   claims.UserID.String(),
		},
	}

	accessTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err = accessTokenObj.SignedString(a.GetJWTSecret())
	if err != nil {
		return "", time.Time{}, err
	}

	// Store new session
	tokenHash := hashToken(accessToken)
	if _, err := a.db.CreateSession(claims.UserID, tokenHash, accessExpiry); err != nil {
		log.Printf("Warning: Failed to create session: %v", err)
	}

	// Log token refresh event
	a.rotationLogger.Printf("Access token refreshed for user %s, device %s - using current JWT secret",
		claims.UserID, claims.DeviceID)

	return accessToken, accessExpiry, nil
}

// GetUserByPhone finds or creates a user by phone number
func (a *AuthService) GetUserByPhone(phoneNumber string) (*uuid.UUID, bool, error) {
	userID, err := a.db.GetUserByPhone(phoneNumber)
	if err != nil {
		return nil, false, nil // New user
	}
	return userID, true, nil // Existing user
}

// RegisterUser creates a new user with their cryptographic keys
func (a *AuthService) RegisterUser(phoneNumber, displayName, identityKey, signedPrekey, prekeySignature string) (*uuid.UUID, error) {
	return a.db.CreateUser(phoneNumber, displayName, identityKey, signedPrekey, prekeySignature)
}

// RevokeAllUserTokens revokes all active sessions for a user (used when password/PIN changes)
func (a *AuthService) RevokeAllUserTokens(userID uuid.UUID) error {
	return a.db.RevokeAllUserSessions(userID)
}

// ============================================
// TOKEN BLACKLISTING (Session Security)
// ============================================

// BlacklistToken adds a token to the global blacklist to prevent reuse
func (a *AuthService) BlacklistToken(tokenString string, reason string) error {
	a.blacklistLock.Lock()
	defer a.blacklistLock.Unlock()

	// Hash the token for storage (security best practice)
	tokenHash := hashTokenForBlacklist(tokenString)

	// Store in Redis with expiration (7 days for compromised tokens)
	ctx := context.Background()
	err := a.redisClient.Set(ctx, fmt.Sprintf("blacklist:%s", tokenHash), reason, 7*24*time.Hour).Err()
	if err != nil {
		a.securityLogger.Printf("Failed to blacklist token %s: %v", tokenHash[:8], err)
		return fmt.Errorf("failed to blacklist token: %w", err)
	}

	a.securityLogger.Printf("Token blacklisted: %s (reason: %s)", tokenHash[:8], reason)
	return nil
}

// IsTokenBlacklisted checks if a token is in the global blacklist
func (a *AuthService) IsTokenBlacklisted(tokenString string) (bool, string, error) {
	a.blacklistLock.RLock()
	defer a.blacklistLock.RUnlock()

	// Hash the token for lookup
	tokenHash := hashTokenForBlacklist(tokenString)

	// Check Redis blacklist
	ctx := context.Background()
	reason, err := a.redisClient.Get(ctx, fmt.Sprintf("blacklist:%s", tokenHash)).Result()
	if err == redis.Nil {
		// Not blacklisted
		return false, "", nil
	} else if err != nil {
		a.securityLogger.Printf("Error checking token blacklist: %v", err)
		return false, "", fmt.Errorf("failed to check token blacklist: %w", err)
	}

	a.securityLogger.Printf("Blacklisted token detected: %s (reason: %s)", tokenHash[:8], reason)
	return true, reason, nil
}

// BlacklistUserTokens adds all active tokens for a user to the blacklist
func (a *AuthService) BlacklistUserTokens(userID uuid.UUID, reason string) error {
	a.blacklistLock.Lock()
	defer a.blacklistLock.Unlock()

	// First revoke all sessions in database
	if err := a.RevokeAllUserTokens(userID); err != nil {
		a.securityLogger.Printf("Failed to revoke user sessions before blacklisting: %v", err)
	}

	// Get all active sessions for the user
	rows, err := a.db.GetDB().Query(`
		SELECT token_hash FROM sessions
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to retrieve user sessions: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	// Blacklist each token
	ctx := context.Background()
	for rows.Next() {
		var tokenHash string
		if err := rows.Scan(&tokenHash); err != nil {
			continue
		}

		// Blacklist the token
		err := a.redisClient.Set(ctx, fmt.Sprintf("blacklist:%s", tokenHash), reason, 7*24*time.Hour).Err()
		if err != nil {
			a.securityLogger.Printf("Failed to blacklist user token %s: %v", tokenHash[:8], err)
		} else {
			a.securityLogger.Printf("User token blacklisted: %s (reason: %s)", tokenHash[:8], reason)
		}
	}

	return nil
}

// CheckTokenSecurity performs comprehensive security checks on a token
func (a *AuthService) CheckTokenSecurity(tokenString string) error {
	// Check if token is blacklisted
	isBlacklisted, reason, err := a.IsTokenBlacklisted(tokenString)
	if err != nil {
		a.securityLogger.Printf("Token security check failed: %v", err)
		return fmt.Errorf("token security check failed: %w", err)
	}

	if isBlacklisted {
		a.securityLogger.Printf("Security violation: Blacklisted token used (reason: %s)", reason)
		return ErrTokenBlacklisted
	}

	// Check for suspicious patterns (e.g., token reuse, rapid token generation)
	// This would be enhanced with additional security monitoring

	return nil
}

// hashTokenForBlacklist creates a secure hash of a token for blacklist storage
func hashTokenForBlacklist(token string) string {
	// Use SHA-256 for secure hashing
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// GetBlacklistedTokenCount returns the number of currently blacklisted tokens
func (a *AuthService) GetBlacklistedTokenCount() (int64, error) {
	ctx := context.Background()

	// Use Redis KEYS command to count blacklist entries
	// Note: In production, consider using SCAN for large datasets
	keys, err := a.redisClient.Keys(ctx, "blacklist:*").Result()
	if err != nil {
		return 0, fmt.Errorf("failed to count blacklisted tokens: %w", err)
	}

	return int64(len(keys)), nil
}

// ClearExpiredBlacklistEntries removes expired blacklist entries
func (a *AuthService) ClearExpiredBlacklistEntries() error {
	// Redis automatically handles TTL, but we can add manual cleanup if needed
	// This is a placeholder for potential future enhancements
	return nil
}

// TOTP 2FA Methods

// GenerateTOTPSecret generates a new TOTP secret for a user
func (a *AuthService) GenerateTOTPSecret(userID uuid.UUID) (string, string, error) {
	// Generate 32-byte random secret
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	// Encode as base32 for TOTP
	secret := base32.StdEncoding.EncodeToString(secretBytes)

	// Encrypt the secret using AES-256-GCM with JWT secret as master key
	masterKey := a.GetJWTSecret()
	if len(masterKey) < 32 {
		// Pad or hash the key if needed, but JWT secret should be 32+ bytes
		masterKey = append(masterKey, make([]byte, 32-len(masterKey))...)
		masterKey = masterKey[:32]
	} else {
		masterKey = masterKey[:32]
	}

	encryptedSecret, err := security.EncryptAESGCM(secretBytes, masterKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to encrypt TOTP secret: %w", err)
	}

	// Encode encrypted secret as base64 for storage
	encryptedSecretB64 := base64.StdEncoding.EncodeToString(encryptedSecret)

	// Store encrypted secret in database
	if err := a.db.SaveTOTPSecret(userID, encryptedSecretB64); err != nil {
		return "", "", fmt.Errorf("failed to store TOTP secret: %w", err)
	}

	// Generate provisioning URI for QR code
	accountName := fmt.Sprintf("SecureMessenger:%s", userID.String())
	issuer := "Secure Messenger"
	uri := fmt.Sprintf("otpauth://totp/%s?secret=%s&issuer=%s",
		url.QueryEscape(accountName), secret, url.QueryEscape(issuer))

	return secret, uri, nil
}

// ValidateTOTPCode validates a TOTP code against the stored secret for a user
func (a *AuthService) ValidateTOTPCode(userID uuid.UUID, code string) bool {
	// Retrieve encrypted secret from database
	encryptedSecretB64, err := a.db.GetTOTPSecret(userID)
	if err != nil {
		log.Printf("Failed to retrieve TOTP secret for user %s: %v", userID, err)
		return false
	}
	if encryptedSecretB64 == "" {
		// No TOTP secret set for this user
		return false
	}

	// Decode from base64
	encryptedSecret, err := base64.StdEncoding.DecodeString(encryptedSecretB64)
	if err != nil {
		log.Printf("Failed to decode TOTP secret for user %s: %v", userID, err)
		return false
	}

	// Decrypt the secret using AES-256-GCM with JWT secret as master key
	masterKey := a.GetJWTSecret()
	if len(masterKey) < 32 {
		masterKey = append(masterKey, make([]byte, 32-len(masterKey))...)
		masterKey = masterKey[:32]
	} else {
		masterKey = masterKey[:32]
	}

	secretBytes, err := security.DecryptAESGCM(encryptedSecret, masterKey)
	if err != nil {
		log.Printf("Failed to decrypt TOTP secret for user %s: %v", userID, err)
		return false
	}

	// Get current time window (30-second intervals)
	now := time.Now().Unix()
	timeWindow := now / 30

	// Check current window and adjacent windows (±1) for clock skew tolerance
	for offset := int64(-1); offset <= 1; offset++ {
		counter := uint64(timeWindow + offset)

		// Generate HMAC-SHA256 (more secure than SHA-1, RFC 6238 compliant)
		h := hmac.New(sha256.New, secretBytes)
		if err := binary.Write(h, binary.BigEndian, counter); err != nil {
			log.Printf("Warning: binary.Write failed: %v", err)
			continue
		}
		hash := h.Sum(nil)

		// Dynamic truncation (SHA-256 produces 32 bytes, so use last byte for offset)
		offset := hash[31] & 0x0F
		truncatedHash := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7FFFFFFF

		// Generate 6-digit code
		generatedCode := fmt.Sprintf("%06d", truncatedHash%1000000)

		if generatedCode == code {
			return true
		}
	}

	return false
}

func generateCode(length int) (string, error) {
	const digits = "0123456789"
	code := make([]byte, length)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		code[i] = digits[n.Int64()]
	}
	return string(code), nil
}

func hashToken(token string) string {
	// Generate a random salt for each token hash
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		// Fallback to timestamp-based salt if random fails
		salt = []byte(fmt.Sprintf("%d", time.Now().UnixNano()))
	}

	// Combine token with salt and hash
	saltedToken := append([]byte(token), salt...)
	hash := sha256.Sum256(saltedToken)

	// Include salt in the final hash for verification
	finalHash := append(hash[:], salt...)
	return hex.EncodeToString(finalHash)
}
