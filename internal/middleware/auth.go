package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jaydenbeard/messaging-app/internal/auth"
)

type contextKey string

const (
	UserIDKey   contextKey = "user_id"
	DeviceIDKey contextKey = "device_id"
)

// AuthMiddleware validates JWT tokens
func AuthMiddleware(authService *auth.AuthService, skipAuth func(*http.Request) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authentication for public paths
			if skipAuth != nil && skipAuth(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Get token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// Expect "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			token := parts[1]

			// Validate token
			claims, err := authService.ValidateToken(token)
			if err != nil {
				if err == auth.ErrTokenExpired {
					http.Error(w, "Token expired", http.StatusUnauthorized)
				} else {
					http.Error(w, "Invalid token", http.StatusUnauthorized)
				}
				return
			}

			// Add user info to context
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, DeviceIDKey, claims.DeviceID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
	return userID, ok
}

// GetDeviceID extracts device ID from context
func GetDeviceID(ctx context.Context) (uuid.UUID, bool) {
	deviceID, ok := ctx.Value(DeviceIDKey).(uuid.UUID)
	return deviceID, ok
}
