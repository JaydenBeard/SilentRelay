package handlers

// Authentication handlers for user registration, login, and token management.

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jaydenbeard/messaging-app/internal/auth"
	"github.com/jaydenbeard/messaging-app/internal/db"
	"github.com/jaydenbeard/messaging-app/internal/models"
	"github.com/jaydenbeard/messaging-app/internal/security"
)

// RequestVerificationCode godoc
// @Summary Request SMS verification code
// @Description Sends a 6-digit verification code via SMS to the provided phone number
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.AuthRequest true "Phone number in E.164 format"
// @Success 200 {object} map[string]interface{} "Verification code sent"
// @Failure 400 {object} map[string]string "Invalid request body or phone number"
// @Failure 429 {object} map[string]string "Account locked or rate limited"
// @Failure 500 {object} map[string]string "Failed to send verification code"
// @Router /auth/request-code [post]
func RequestVerificationCode(authService *auth.AuthService, auditLogger *security.AuditLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.AuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate phone number format
		if err := validatePhoneNumber(req.PhoneNumber); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Check if account is locked
		if lockoutTracker.isLocked(req.PhoneNumber) {
			locked, until := lockoutTracker.getLockoutInfo(req.PhoneNumber)
			if locked && until != nil {
				auditLogger.LogSecurityEvent(r.Context(), security.AuditEventBruteForceBlocked, security.AuditResultDenied, nil, "Account locked due to too many failed attempts", map[string]any{"phone_number": req.PhoneNumber})
				http.Error(w, fmt.Sprintf("Account locked until %s", until.Format(time.RFC3339)), http.StatusTooManyRequests)
				return
			}
		}

		code, err := authService.RequestVerificationCode(req.PhoneNumber)
		if err != nil {
			auditLogger.LogSecurityEvent(r.Context(), security.AuditEventInvalidRequest, security.AuditResultError, nil, "Failed to send verification code", map[string]any{"phone_number": req.PhoneNumber, "error": err.Error()})
			http.Error(w, "Failed to send verification code", http.StatusInternalServerError)
			return
		}

		auditLogger.LogSecurityEvent(r.Context(), security.AuditEventLoginAttempt, security.AuditResultSuccess, nil, "Verification code requested", map[string]any{"phone_number": req.PhoneNumber})

		w.Header().Set("Content-Type", "application/json")

		// SECURITY: Only return code in development mode
		// Check DEV_MODE environment variable (defaults to false/production)
		devMode := os.Getenv("DEV_MODE") == "true"
		if devMode {
			// DEV MODE ONLY - Never enable in production
			writeJSON(w, map[string]interface{}{
				"message": "Verification code sent (DEV MODE)",
				"code":    code,
			})
		} else {
			writeJSON(w, map[string]interface{}{
				"message": "Verification code sent",
			})
		}
	}
}

// VerifyCode godoc
// @Summary Verify SMS code
// @Description Validates the SMS verification code. Returns tokens for existing users.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.AuthVerifyRequest true "Phone number and verification code"
// @Success 200 {object} map[string]interface{} "Code verified, returns user_exists and tokens if existing user"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Invalid code"
// @Failure 429 {object} map[string]string "Too many attempts"
// @Router /auth/verify [post]
func VerifyCode(authService *auth.AuthService, database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.AuthVerifyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Check if user exists first
		userID, exists, err := authService.GetUserByPhone(req.PhoneNumber)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// For existing users, verify and mark as verified (they'll login next)
		// For new users, only check validity without marking (they'll register next)
		var valid bool
		if exists {
			// Existing user - verify and mark as verified (they'll proceed to login)
			valid, err = authService.VerifyCode(req.PhoneNumber, req.Code)
		} else {
			// New user - only check validity, don't mark as verified yet
			valid, err = authService.CheckCode(req.PhoneNumber, req.Code)
		}

		if err != nil {
			http.Error(w, "Verification failed", http.StatusInternalServerError)
			return
		}

		if !valid {
			// Record failed attempt for lockout protection
			lockoutTracker.recordFailedAttempt(req.PhoneNumber)
			http.Error(w, "Invalid or expired code", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		// For existing users, generate tokens and return user data
		if exists && userID != nil {
			// Get user details
			user, err := database.GetUserByID(*userID)
			if err != nil {
				http.Error(w, "Failed to get user", http.StatusInternalServerError)
				return
			}

			// Generate device ID for this session
			deviceID := uuid.New()

			// Generate tokens
			accessToken, refreshToken, _, err := authService.GenerateTokens(*userID, deviceID)
			if err != nil {
				http.Error(w, "Failed to generate tokens", http.StatusInternalServerError)
				return
			}

			writeJSON(w, map[string]interface{}{
				"verified":      true,
				"user_exists":   true,
				"token":         accessToken,
				"refresh_token": refreshToken,
				"device_id":     deviceID.String(),
				"user": map[string]interface{}{
					"id":           userID.String(),
					"phone_number": user["phone_number"],
					"username":     user["username"],
					"display_name": user["display_name"],
					"avatar_url":   user["avatar_url"],
				},
			})
			return
		}

		// For new users, just return verification status
		writeJSON(w, map[string]interface{}{
			"verified":    true,
			"user_exists": exists,
			"user_id":     userID,
		})
	}
}

// Register creates a new user account
func Register(authService *auth.AuthService, database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("[Register] Invalid request body: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		log.Printf("[Register] Attempting registration for phone: %s", req.PhoneNumber)

		// Validate required fields
		if req.PhoneNumber == "" || req.PublicIdentityKey == "" || req.PublicSignedPrekey == "" {
			log.Printf("[Register] Missing required fields")
			http.Error(w, "Missing required fields", http.StatusBadRequest)
			return
		}

		// SECURITY: Re-verify code to prevent TOCTOU vulnerability
		if req.Code == "" {
			log.Printf("[Register] No verification code provided")
			http.Error(w, "Verification code required", http.StatusBadRequest)
			return
		}

		// SECURITY: Check code validity first WITHOUT marking as verified
		// This prevents code consumption if user creation fails
		log.Printf("[Register] Checking code %s for phone %s", req.Code, req.PhoneNumber)
		valid, err := authService.CheckCode(req.PhoneNumber, req.Code)
		if err != nil {
			log.Printf("[Register] CheckCode error: %v", err)
			http.Error(w, "Invalid or expired verification code", http.StatusUnauthorized)
			return
		}
		if !valid {
			log.Printf("[Register] Code check failed - code invalid, expired, or already used")
			http.Error(w, "Invalid or expired verification code", http.StatusUnauthorized)
			return
		}
		log.Printf("[Register] Code is valid")

		// Get display name (convert pointer to string)
		displayName := ""
		if req.DisplayName != nil {
			displayName = *req.DisplayName
		}

		// Create user
		userID, err := authService.RegisterUser(
			req.PhoneNumber,
			displayName,
			req.PublicIdentityKey,
			req.PublicSignedPrekey,
			req.SignedPrekeySignature,
		)
		if err != nil {
			log.Printf("[Register] Failed to create user: %v", err)
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}

		// Mark code as verified AFTER successful user creation
		if err := authService.MarkCodeVerified(req.PhoneNumber, req.Code); err != nil {
			log.Printf("[Register] Warning: Failed to mark code as verified (non-critical): %v", err)
			// Don't fail registration - user is already created
		}
		log.Printf("[Register] User created successfully: %s", userID)

		// Generate tokens
		accessToken, refreshToken, expiresAt, err := authService.GenerateTokens(*userID, req.DeviceID)
		if err != nil {
			http.Error(w, "Failed to generate tokens", http.StatusInternalServerError)
			return
		}

		// Register this device as the PRIMARY device (first device for new user)
		deviceName := "Primary Device"
		if req.DeviceName != nil && *req.DeviceName != "" {
			deviceName = *req.DeviceName
		}
		deviceType := req.DeviceType
		if deviceType == "" {
			deviceType = "web"
		}
		if err := database.RegisterDevice(*userID, req.DeviceID, deviceName, deviceType, req.PublicDeviceKey, true); err != nil {
			// Log but don't fail - device registration is not critical for signup
			fmt.Printf("Warning: Failed to register primary device: %v\n", err)
		}

		// Save one-time pre-keys if provided
		if len(req.PreKeys) > 0 {
			prekeys := make([]struct {
				ID        int
				PublicKey string
			}, len(req.PreKeys))
			for i, pk := range req.PreKeys {
				prekeys[i] = struct {
					ID        int
					PublicKey string
				}{
					ID:        int(pk.PreKeyID),
					PublicKey: pk.PublicKey,
				}
			}
			if err := database.SavePreKeys(*userID, prekeys); err != nil {
				log.Printf("[Register] Warning: Failed to save prekeys: %v", err)
				// Don't fail registration - prekeys can be uploaded later
			} else {
				log.Printf("[Register] Saved %d prekeys for user %s", len(prekeys), userID)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, models.AuthResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresAt:    expiresAt,
			User: models.User{
				UserID:      *userID,
				PhoneNumber: req.PhoneNumber,
				DisplayName: req.DisplayName,
			},
		})
	}
}

// Login handles returning user authentication on a new device
func Login(authService *auth.AuthService, database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if req.PhoneNumber == "" {
			http.Error(w, "Phone number required", http.StatusBadRequest)
			return
		}

		// Get user by phone
		userID, exists, err := authService.GetUserByPhone(req.PhoneNumber)
		if err != nil || !exists {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Get user details
		user, err := database.GetUserByID(*userID)
		if err != nil {
			http.Error(w, "Failed to get user", http.StatusInternalServerError)
			return
		}

		// Generate tokens for this device
		accessToken, refreshToken, expiresAt, err := authService.GenerateTokens(*userID, req.DeviceID)
		if err != nil {
			http.Error(w, "Failed to generate tokens", http.StatusInternalServerError)
			return
		}

		// Register this device
		deviceID := req.DeviceID
		deviceName := req.DeviceName
		if deviceName == "" {
			deviceName = "Web Browser"
		}
		deviceType := req.DeviceType
		if deviceType == "" {
			deviceType = "web"
		}

		// Check if this device should be primary
		// Note: These checks use sensible defaults if DB fails - log errors for debugging
		// 1. Check if user has any devices (if not, this is the first device)
		hasDevices, err := database.HasLinkedDevices(*userID)
		if err != nil {
			log.Printf("Warning: Failed to check linked devices for user %s: %v", userID, err)
			hasDevices = false // Default to false (treat as first device)
		}
		// 2. Check if user has a primary device (if not, this becomes primary)
		primaryDevice, err := database.GetPrimaryDevice(*userID)
		if err != nil {
			log.Printf("Warning: Failed to get primary device for user %s: %v", userID, err)
			// primaryDevice will be nil, which is handled below
		}
		// 3. Check if this device was previously the primary device
		isPreviouslyPrimary, err := database.IsPrimaryDevice(*userID, deviceID)
		if err != nil {
			log.Printf("Warning: Failed to check primary device status for user %s: %v", userID, err)
			isPreviouslyPrimary = false
		}

		shouldBePrimary := !hasDevices || primaryDevice == nil || isPreviouslyPrimary

		if err := database.RegisterDevice(*userID, deviceID, deviceName, deviceType, req.PublicDeviceKey, shouldBePrimary); err != nil {
			// Log but don't fail - device registration is not critical
			log.Printf("Warning: Failed to register device for user %s: %v", userID, err)
		} else if shouldBePrimary {
			// Ensure this device is set as primary (in case it wasn't set correctly)
			if err := database.SetPrimaryDevice(*userID, deviceID); err != nil {
				log.Printf("Warning: Failed to set primary device for user %s: %v", userID, err)
			}
		}

		// Get display name from user map
		var displayName *string
		if dn, ok := user["display_name"].(string); ok && dn != "" {
			displayName = &dn
		}

		// Get PIN status (only return has_pin flag, not the hash or length)
		pinHash, _, err := database.GetUserPIN(*userID)
		if err != nil {
			log.Printf("Warning: Failed to get PIN status for user %s: %v", userID, err)
		}
		hasPIN := pinHash != ""

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]interface{}{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"expires_at":    expiresAt,
			"user": models.User{
				UserID:      *userID,
				PhoneNumber: req.PhoneNumber,
				DisplayName: displayName,
			},
			"has_pin": hasPIN,
			// SECURITY: Don't expose pin_hash or pin_length to prevent offline attacks
		})
	}
}

// RefreshToken generates a new access token
func RefreshToken(authService *auth.AuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		accessToken, expiresAt, err := authService.RefreshAccessToken(req.RefreshToken)
		if err != nil {
			http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]interface{}{
			"access_token": accessToken,
			"expires_at":   expiresAt,
		})
	}
}
