package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/joho/godotenv"
)

// JWTKeyManager provides secure JWT secret management with rotation support
type JWTKeyManager struct {
	currentSecret    string
	previousSecret   string
	rotationTime     time.Time
	rotationInterval time.Duration
	lock             sync.RWMutex
	logger           *log.Logger
}

// VaultClient provides secure secret management via HashiCorp Vault
type VaultClient struct {
	client     *api.Client
	mountPath  string
	secretPath string
	logger     *log.Logger
}

// Global instances
var (
	keyManager = &JWTKeyManager{
		logger: log.New(os.Stdout, "[JWT-ROTATION] ", log.Ldate|log.Ltime|log.LUTC),
	}
	vaultClient *VaultClient
)

// InitializeKeyManager sets up the JWT key manager with current secret
func InitializeKeyManager(secret string) {
	keyManager.lock.Lock()
	defer keyManager.lock.Unlock()

	keyManager.currentSecret = secret
	keyManager.previousSecret = "" // No previous secret initially
	keyManager.rotationTime = time.Now()
	keyManager.rotationInterval = 24 * time.Hour // Default rotation interval
	keyManager.logger.Printf("JWT Key Manager initialized with rotation interval: %v", keyManager.rotationInterval)
}

// InitializeVaultClient sets up HashiCorp Vault client for secret management
func InitializeVaultClient(vaultAddr, token, mountPath, secretPath string) error {
	config := &api.Config{
		Address: vaultAddr,
	}

	client, err := api.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to create Vault client: %w", err)
	}

	client.SetToken(token)

	// Test connection
	_, err = client.Sys().Health()
	if err != nil {
		return fmt.Errorf("failed to connect to Vault: %w", err)
	}

	vaultClient = &VaultClient{
		client:     client,
		mountPath:  mountPath,
		secretPath: secretPath,
		logger:     log.New(os.Stdout, "[VAULT] ", log.Ldate|log.Ltime|log.LUTC),
	}

	vaultClient.logger.Printf("Vault client initialized - Address: %s, Mount: %s, Path: %s",
		vaultAddr, mountPath, secretPath)

	return nil
}

// GetSecretFromVault retrieves a secret from HashiCorp Vault
func GetSecretFromVault(key string) (string, error) {
	if vaultClient == nil {
		return "", fmt.Errorf("vault client not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	secret, err := vaultClient.client.KVv2(vaultClient.mountPath).Get(ctx, vaultClient.secretPath)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve secret from Vault: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("secret not found in Vault path: %s/%s", vaultClient.mountPath, vaultClient.secretPath)
	}

	value, ok := secret.Data[key].(string)
	if !ok {
		return "", fmt.Errorf("secret key '%s' not found or not a string", key)
	}

	return value, nil
}

// GetJWTSecretFromVault retrieves JWT secret from Vault with fallback to env
func GetJWTSecretFromVault() (string, error) {
	// Try Vault first
	if vaultClient != nil {
		secret, err := GetSecretFromVault("jwt_secret")
		if err == nil && secret != "" {
			vaultClient.logger.Printf("JWT secret retrieved from Vault")
			return secret, nil
		}
		vaultClient.logger.Printf("Failed to get JWT secret from Vault, falling back to environment: %v", err)
	}

	// Fallback to environment variable
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", fmt.Errorf("JWT_SECRET not found in Vault or environment")
	}

	return secret, nil
}

// GetCurrentSecret provides thread-safe access to current JWT secret
func GetCurrentSecret() string {
	keyManager.lock.RLock()
	defer keyManager.lock.RUnlock()
	return keyManager.currentSecret
}

// GetPreviousSecret provides thread-safe access to previous JWT secret (for rotation)
func GetPreviousSecret() string {
	keyManager.lock.RLock()
	defer keyManager.lock.RUnlock()
	return keyManager.previousSecret
}

// RotateSecret performs secure JWT secret rotation with dual-key support
func RotateSecret(newSecret string) error {
	if len(newSecret) < 32 {
		return fmt.Errorf("new JWT secret must be at least 32 characters long")
	}

	keyManager.lock.Lock()
	defer keyManager.lock.Unlock()

	// Validate new secret has sufficient entropy
	if err := ValidateJWTSecret(newSecret); err != nil {
		return fmt.Errorf("new JWT secret validation failed: %w", err)
	}

	// Log rotation event
	keyManager.logger.Printf("Starting JWT secret rotation - current: %s, new: %s",
		getSecretPreview(keyManager.currentSecret),
		getSecretPreview(newSecret))

	// Store current secret as previous for transition period
	keyManager.previousSecret = keyManager.currentSecret
	keyManager.currentSecret = newSecret
	keyManager.rotationTime = time.Now()

	// Log successful rotation
	keyManager.logger.Printf("JWT secret rotation completed successfully")
	keyManager.logger.Printf("Transition period started - both old and new keys will be accepted")

	return nil
}

// loadEnvFiles loads environment files in the correct order
func loadEnvFiles() {
	// Load base .env file (ignore error - file may not exist)
	_ = godotenv.Load()

	// Load environment-specific file (e.g., .env.development, .env.production)
	if env := os.Getenv("NODE_ENV"); env != "" {
		_ = godotenv.Load(".env." + env)
	}

	// Load local overrides (.env.local)
	_ = godotenv.Load(".env.local")
}

// Config holds all configuration for the chat server
type Config struct {
	ServerID    string
	ServerPort  string
	RedisURL    string
	PostgresURL string
	ConsulURL   string
	JWTSecret   string
	MinioURL    string
	MinioKey    string
	MinioSecret string
	MinioBucket string
	RateLimits  *RateLimitConfig
	MediaLimits *MediaLimitConfig
}

// Load reads configuration from Vault or environment variables
func Load() *Config {
	// Load environment files in order: .env -> .env.{NODE_ENV} -> .env.local
	loadEnvFiles()

	// Try to initialize Vault client if Vault environment variables are set
	vaultAddr := os.Getenv("VAULT_ADDR")
	vaultToken := os.Getenv("VAULT_TOKEN")
	mountPath := getEnv("VAULT_MOUNT_PATH", "secret")
	secretPath := getEnv("VAULT_SECRET_PATH", "messaging")

	if vaultAddr != "" && vaultToken != "" {
		if err := InitializeVaultClient(vaultAddr, vaultToken, mountPath, secretPath); err != nil {
			log.Printf("Warning: Failed to initialize Vault client: %v", err)
			log.Printf("Falling back to environment variables for secrets")
		}
	}

	// Get JWT secret from Vault or environment
	jwtSecret, err := GetJWTSecretFromVault()
	if err != nil {
		log.Fatalf("FATAL: JWT_SECRET not found in Vault or environment: %v", err)
	}
	if len(jwtSecret) < 32 {
		log.Fatal("FATAL: JWT_SECRET must be at least 32 characters long for security.")
	}

	// Initialize the key manager with the retrieved secret
	InitializeKeyManager(jwtSecret)

	config := &Config{
		ServerID:    getEnv("SERVER_ID", "chat-server-1"),
		ServerPort:  getEnv("SERVER_PORT", "8080"),
		RedisURL:    getEnv("REDIS_URL", "localhost:6379"),
		PostgresURL: getEnv("POSTGRES_URL", "postgres://messaging:messaging@localhost:5432/messaging?sslmode=disable"),
		ConsulURL:   getEnv("CONSUL_URL", "localhost:8500"),
		JWTSecret:   jwtSecret,
		MinioURL:    getEnv("MINIO_URL", "localhost:9000"),
		MinioKey:    getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecret: getEnv("MINIO_SECRET_KEY", "minioadmin123"),
		MinioBucket: getEnv("MINIO_BUCKET", "encrypted-media"),
		RateLimits: &RateLimitConfig{
			IPLimits:       make(map[string]*TieredLimitConfig),
			UserLimits:     make(map[string]*TieredLimitConfig),
			EndpointLimits: make(map[string]*TieredLimitConfig),
			GlobalLimits: &TieredLimitConfig{
				Normal: &LimitConfig{
					MaxRequests: 1000,
					Window:      1 * time.Minute,
				},
				Strict: &LimitConfig{
					MaxRequests: 500,
					Window:      1 * time.Minute,
				},
			},
			AbuseDetection: &AbuseDetectionConfig{
				Threshold:          100,
				Window:             5 * time.Minute,
				PenaltyDuration:    15 * time.Minute,
				StrictModeDuration: 30 * time.Minute,
			},
		},
		MediaLimits: &MediaLimitConfig{
			MaxImageSize: getEnvInt64("MAX_IMAGE_SIZE_MB", 100) * 1024 * 1024, // 100MB default
			MaxVideoSize: getEnvInt64("MAX_VIDEO_SIZE_MB", 500) * 1024 * 1024, // 500MB default
			MaxAudioSize: getEnvInt64("MAX_AUDIO_SIZE_MB", 50) * 1024 * 1024,  // 50MB default
			MaxFileSize:  getEnvInt64("MAX_FILE_SIZE_MB", 50) * 1024 * 1024,   // 50MB default
		},
	}

	// Validate configuration for production
	if err := validateProductionSecrets(config); err != nil {
		log.Fatalf("FATAL: Production secret validation failed: %v", err)
	}

	return config
}

// validateProductionSecrets checks for placeholder values in production
func validateProductionSecrets(config *Config) error {
	nodeEnv := getEnv("NODE_ENV", "development")
	if nodeEnv != "production" {
		return nil // Skip validation for non-production
	}

	placeholders := map[string]string{
		"JWT_SECRET":             "YOUR_JWT_SECRET_64_CHARS_HEX_HERE",
		"HMAC_SECRET":            "YOUR_HMAC_SECRET_64_CHARS_HEX_HERE",
		"TURN_SECRET":            "YOUR_TURN_SECRET_64_CHARS_HEX_HERE",
		"POSTGRES_PASSWORD":      "YOUR_POSTGRES_PASSWORD_64_CHARS_HEX_HERE",
		"REDIS_PASSWORD":         "YOUR_REDIS_PASSWORD_32_CHARS_HEX_HERE",
		"MINIO_ROOT_PASSWORD":    "YOUR_MINIO_ROOT_PASSWORD_64_CHARS_HEX_HERE",
		"MINIO_SECRET_KEY":       "YOUR_MINIO_SECRET_KEY_64_CHARS_HEX_HERE",
		"GRAFANA_ADMIN_PASSWORD": "YOUR_GRAFANA_ADMIN_PASSWORD_32_CHARS_HEX_HERE",
		"FCM_SERVER_KEY":         "YOUR_FCM_SERVER_KEY_HERE",
		"FCM_PROJECT_ID":         "YOUR_FIREBASE_PROJECT_ID_HERE",
		"APNS_KEY_ID":            "YOUR_APNS_KEY_ID_HERE",
		"APNS_TEAM_ID":           "YOUR_APNS_TEAM_ID_HERE",
		"SMTP_PASSWORD":          "YOUR_SMTP_PASSWORD_HERE",
		"CLICKSEND_USERNAME":     "YOUR_CLICKSEND_USERNAME_HERE",
		"CLICKSEND_API_KEY":      "YOUR_CLICKSEND_API_KEY_HERE",
	}

	// Check environment variables for placeholders
	for envVar, placeholder := range placeholders {
		if value := os.Getenv(envVar); value == placeholder {
			return fmt.Errorf("production environment detected but %s contains placeholder value '%s'. Replace with real secret", envVar, placeholder)
		}
	}

	// Additional checks for default/weak values
	if config.JWTSecret == "a1b2c3d4e5f6789012345678901234567890123456789012345678901234567890" {
		return fmt.Errorf("production environment detected but JWT_SECRET is using default development value. Generate new secret")
	}

	if config.MinioSecret == "minioadmin123" {
		return fmt.Errorf("production environment detected but MINIO_SECRET_KEY is using default value. Change to strong secret")
	}
	if os.Getenv("CLICKSEND_USERNAME") == "" {
		return fmt.Errorf("production environment detected but CLICKSEND_USERNAME is not set")
	}

	if os.Getenv("CLICKSEND_API_KEY") == "" {
		return fmt.Errorf("production environment detected but CLICKSEND_API_KEY is not set")
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// MustGetEnv retrieves an environment variable or fails if not set
func MustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("FATAL: Required environment variable %s is not set", key)
	}
	return value
}

// GetJWTSecret provides secure access to JWT secret with validation
func GetJWTSecret() (string, error) {
	secret := GetCurrentSecret()
	if secret == "" {
		return "", fmt.Errorf("JWT secret not initialized")
	}

	// Validate minimum length
	if len(secret) < 32 {
		return "", fmt.Errorf("JWT secret is too short (minimum 32 characters)")
	}

	return secret, nil
}

// GetAllActiveSecrets returns both current and previous secrets for dual-key validation
func GetAllActiveSecrets() (current, previous string, hasPrevious bool) {
	keyManager.lock.RLock()
	defer keyManager.lock.RUnlock()

	return keyManager.currentSecret, keyManager.previousSecret, keyManager.previousSecret != ""
}

// GetRotationInfo returns information about the last rotation
func GetRotationInfo() (lastRotation time.Time, interval time.Duration) {
	keyManager.lock.RLock()
	defer keyManager.lock.RUnlock()

	return keyManager.rotationTime, keyManager.rotationInterval
}

// SetRotationInterval sets the automatic rotation interval
func SetRotationInterval(interval time.Duration) {
	keyManager.lock.Lock()
	defer keyManager.lock.Unlock()

	if interval < 1*time.Hour {
		keyManager.logger.Printf("Warning: Rotation interval %v is too short, using minimum 1 hour", interval)
		interval = 1 * time.Hour
	}

	keyManager.rotationInterval = interval
	keyManager.logger.Printf("Rotation interval set to: %v", interval)
}

// ShouldRotate checks if automatic rotation should occur based on interval
func ShouldRotate() bool {
	keyManager.lock.RLock()
	defer keyManager.lock.RUnlock()

	if keyManager.rotationInterval <= 0 {
		return false
	}

	return time.Since(keyManager.rotationTime) >= keyManager.rotationInterval
}

// getSecretPreview returns a safe preview of secret for logging (first 4 + last 4 chars)
func getSecretPreview(secret string) string {
	if len(secret) <= 8 {
		return "****"
	}
	return secret[:4] + "..." + secret[len(secret)-4:]
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	IPLimits       map[string]*TieredLimitConfig
	UserLimits     map[string]*TieredLimitConfig
	EndpointLimits map[string]*TieredLimitConfig
	GlobalLimits   *TieredLimitConfig
	AbuseDetection *AbuseDetectionConfig
}

// TieredLimitConfig defines tiered limit configuration
type TieredLimitConfig struct {
	Normal *LimitConfig
	Strict *LimitConfig
}

// LimitConfig defines rate limit parameters
type LimitConfig struct {
	MaxRequests int
	Window      time.Duration
}

// AbuseDetectionConfig defines abuse detection parameters
type AbuseDetectionConfig struct {
	Threshold          int
	Window             time.Duration
	PenaltyDuration    time.Duration
	StrictModeDuration time.Duration
}

// MediaLimitConfig defines media upload size limits for DoS protection
type MediaLimitConfig struct {
	MaxImageSize int64 // Maximum size for images in bytes (default: 100MB)
	MaxVideoSize int64 // Maximum size for videos in bytes (default: 500MB)
	MaxAudioSize int64 // Maximum size for audio files in bytes (default: 50MB)
	MaxFileSize  int64 // Maximum size for other files in bytes (default: 50MB)
}

// ValidateJWTSecret checks if a JWT secret meets security requirements
func ValidateJWTSecret(secret string) error {
	if secret == "" {
		return fmt.Errorf("JWT secret cannot be empty")
	}

	if len(secret) < 32 {
		return fmt.Errorf("JWT secret must be at least 32 characters long")
	}

	// Check for sufficient character diversity
	uniqueChars := make(map[rune]bool)
	for _, char := range secret {
		uniqueChars[char] = true
	}

	if len(uniqueChars) < 10 {
		return fmt.Errorf("JWT secret must contain at least 10 unique characters")
	}

	return nil
}
