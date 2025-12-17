package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jaydenbeard/messaging-app/internal/db"
	"github.com/jaydenbeard/messaging-app/internal/middleware"
)

// SendFriendRequest sends a friend request to another user
func SendFriendRequest(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req struct {
			UserID string `json:"user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		addresseeID, err := uuid.Parse(req.UserID)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		// Can't send friend request to yourself
		if userID == addresseeID {
			http.Error(w, "Cannot send friend request to yourself", http.StatusBadRequest)
			return
		}

		// Check if blocked
		isBlocked, err := database.IsBlocked(addresseeID, userID)
		if err == nil && isBlocked {
			http.Error(w, "Cannot send friend request", http.StatusForbidden)
			return
		}

		if err := database.SendFriendRequest(userID, addresseeID); err != nil {
			if err.Error() == "already friends" || err.Error() == "friend request already pending" {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			http.Error(w, "Failed to send friend request", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]bool{"success": true})
	}
}

// AcceptFriendRequest accepts a pending friend request
func AcceptFriendRequest(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req struct {
			UserID string `json:"user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		requesterID, err := uuid.Parse(req.UserID)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		if err := database.AcceptFriendRequest(userID, requesterID); err != nil {
			if err.Error() == "no pending friend request found" {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, "Failed to accept friend request", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]bool{"success": true})
	}
}

// DeclineFriendRequest declines a pending friend request
func DeclineFriendRequest(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req struct {
			UserID string `json:"user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		requesterID, err := uuid.Parse(req.UserID)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		if err := database.DeclineFriendRequest(userID, requesterID); err != nil {
			if err.Error() == "no pending friend request found" {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, "Failed to decline friend request", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]bool{"success": true})
	}
}

// RemoveFriend removes an existing friendship
func RemoveFriend(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		friendIDStr := vars["userId"]

		friendID, err := uuid.Parse(friendIDStr)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		if err := database.RemoveFriend(userID, friendID); err != nil {
			http.Error(w, "Failed to remove friend", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]bool{"success": true})
	}
}

// GetFriends returns all friends of the current user
func GetFriends(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		friends, err := database.GetFriends(userID)
		if err != nil {
			http.Error(w, "Failed to get friends", http.StatusInternalServerError)
			return
		}

		// Return empty array instead of null
		if friends == nil {
			friends = []db.FriendInfo{}
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, friends)
	}
}

// GetFriendRequests returns pending friend requests
func GetFriendRequests(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get direction from query param (incoming, outgoing, or all)
		direction := r.URL.Query().Get("direction")

		var incoming, outgoing []db.FriendRequest
		var err error

		if direction == "" || direction == "all" || direction == "incoming" {
			incoming, err = database.GetPendingFriendRequests(userID)
			if err != nil {
				http.Error(w, "Failed to get friend requests", http.StatusInternalServerError)
				return
			}
		}

		if direction == "" || direction == "all" || direction == "outgoing" {
			outgoing, err = database.GetSentFriendRequests(userID)
			if err != nil {
				http.Error(w, "Failed to get friend requests", http.StatusInternalServerError)
				return
			}
		}

		// Return empty arrays instead of null
		if incoming == nil {
			incoming = []db.FriendRequest{}
		}
		if outgoing == nil {
			outgoing = []db.FriendRequest{}
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]interface{}{
			"incoming": incoming,
			"outgoing": outgoing,
		})
	}
}

// GetFriendshipStatus returns the friendship status with a specific user
func GetFriendshipStatus(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		otherIDStr := vars["userId"]

		otherID, err := uuid.Parse(otherIDStr)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		status, err := database.GetFriendshipStatus(userID, otherID)
		if err != nil {
			http.Error(w, "Failed to get friendship status", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]string{"status": status})
	}
}

// CancelFriendRequest cancels an outgoing friend request
func CancelFriendRequest(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req struct {
			UserID string `json:"user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		addresseeID, err := uuid.Parse(req.UserID)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		// Cancel by removing the friendship record
		if err := database.RemoveFriend(userID, addresseeID); err != nil {
			http.Error(w, "Failed to cancel friend request", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]bool{"success": true})
	}
}
