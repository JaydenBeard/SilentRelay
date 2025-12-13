package handlers

// User profile handlers for user management operations.

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jaydenbeard/messaging-app/internal/db"
	"github.com/jaydenbeard/messaging-app/internal/middleware"
	"github.com/jaydenbeard/messaging-app/internal/pubsub"
)

// GetCurrentUser returns the authenticated user's profile
func GetCurrentUser(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		user, err := database.GetUserByID(userID)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, user)
	}
}

// GetUserProfile returns a user's public profile by ID
func GetUserProfile(database *db.PostgresDB, redis *pubsub.RedisClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verify caller is authenticated
		_, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get target user ID from URL
		vars := mux.Vars(r)
		targetUserIDStr := vars["userId"]
		targetUserID, err := uuid.Parse(targetUserIDStr)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		user, err := database.GetUserByID(targetUserID)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Get target user's privacy settings
		privacySettings, err := database.GetPrivacySettings(targetUserID)
		if err != nil {
			// Log but continue with default settings (show online/last seen)
			fmt.Printf("Warning: Failed to get privacy settings for user %s: %v\n", targetUserID, err)
		}
		showOnlineStatus := true
		showLastSeen := true
		if privacySettings != nil {
			if v, ok := privacySettings["show_online_status"].(bool); ok {
				showOnlineStatus = v
			}
			if v, ok := privacySettings["show_last_seen"].(bool); ok {
				showLastSeen = v
			}
		}

		// Return only public info (no phone number, no keys)
		// Respect user's privacy settings for online status and last seen
		publicProfile := map[string]interface{}{
			"user_id":      user["user_id"],
			"username":     user["username"],
			"display_name": user["display_name"],
			"avatar_url":   user["avatar_url"],
		}

		// Only include online status if user allows it AND redis is available
		if showOnlineStatus && redis != nil {
			isOnline, lastSeen := redis.GetUserPresence(targetUserID)
			publicProfile["is_online"] = isOnline
			// Use Redis last_seen if available and user allows it
			if showLastSeen && !lastSeen.IsZero() {
				publicProfile["last_seen"] = lastSeen
			}
		} else if showOnlineStatus {
			// Fallback to database is_active if redis not available
			publicProfile["is_online"] = user["is_active"]
		} else {
			publicProfile["is_online"] = false // Always appear offline
		}

		// Only include last seen from DB if we haven't set it from Redis
		if showLastSeen {
			if _, ok := publicProfile["last_seen"]; !ok {
				publicProfile["last_seen"] = user["last_seen"]
			}
		}
		// If showLastSeen is false, don't include last_seen at all

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, publicProfile)
	}
}

// UpdateUser updates user profile
func UpdateUser(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var updates map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := database.UpdateUser(userID, updates); err != nil {
			http.Error(w, "Failed to update user", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]string{"status": "updated"})
	}
}

// DeleteUser permanently deletes a user account
func DeleteUser(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Delete user and all associated data
		if err := database.DeleteUser(userID); err != nil {
			// Log the actual error for debugging
			fmt.Printf("Error deleting user %s: %v\n", userID, err)
			// Return generic error to client
			http.Error(w, "Failed to delete account", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]string{"status": "deleted"})
	}
}
