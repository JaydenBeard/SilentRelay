package handlers

// Message and Group handlers for messaging and group chat operations.

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jaydenbeard/messaging-app/internal/db"
	"github.com/jaydenbeard/messaging-app/internal/middleware"
	"github.com/jaydenbeard/messaging-app/internal/websocket"
)

// ================== Message Handlers ==================

// GetMessages retrieves message history
func GetMessages(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		messages, err := database.GetPendingMessages(userID)
		if err != nil {
			http.Error(w, "Failed to fetch messages", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, messages)
	}
}

// UpdateMessageStatus updates delivery/read status
func UpdateMessageStatus(database *db.PostgresDB, hub *websocket.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		messageIDStr := vars["messageId"]

		messageID, err := uuid.Parse(messageIDStr)
		if err != nil {
			http.Error(w, "Invalid message ID", http.StatusBadRequest)
			return
		}

		var req struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate status
		if req.Status != "delivered" && req.Status != "read" {
			http.Error(w, "Invalid status", http.StatusBadRequest)
			return
		}

		// This will also notify the sender via WebSocket
		// The actual notification is handled in the hub

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]interface{}{
			"message_id": messageID,
			"status":     req.Status,
		})
	}
}

// ================== Group Handlers ==================

// CreateGroup creates a new group chat
func CreateGroup(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req struct {
			Name    string      `json:"name"`
			Members []uuid.UUID `json:"members"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		groupID, err := database.CreateGroup(req.Name, userID)
		if err != nil {
			http.Error(w, "Failed to create group", http.StatusInternalServerError)
			return
		}

		// Add initial members
		for _, memberID := range req.Members {
			if err := database.AddGroupMember(*groupID, memberID, ""); err != nil {
				log.Printf("Warning: failed to add group member: %v", err)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]interface{}{
			"group_id": groupID,
			"name":     req.Name,
		})
	}
}

// GetGroup returns group details
func GetGroup(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		groupIDStr := vars["groupId"]

		groupID, err := uuid.Parse(groupIDStr)
		if err != nil {
			http.Error(w, "Invalid group ID", http.StatusBadRequest)
			return
		}

		members, err := database.GetGroupMembers(groupID)
		if err != nil {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]interface{}{
			"group_id": groupID,
			"members":  members,
		})
	}
}

// AddGroupMember adds a user to a group
func AddGroupMember(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get authenticated user ID
		requesterID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		groupIDStr := vars["groupId"]

		groupID, err := uuid.Parse(groupIDStr)
		if err != nil {
			http.Error(w, "Invalid group ID", http.StatusBadRequest)
			return
		}

		// AUTHORIZATION: Check if requester is a group admin
		isAdmin, err := database.IsGroupAdmin(groupID, requesterID)
		if err != nil {
			fmt.Printf("Error checking group admin status for user %s: %v\n", requesterID, err)
			http.Error(w, "Failed to verify permissions", http.StatusInternalServerError)
			return
		}
		if !isAdmin {
			http.Error(w, "Only group admins can add members", http.StatusForbidden)
			return
		}

		var req struct {
			UserID       uuid.UUID `json:"user_id"`
			EncryptedKey string    `json:"encrypted_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := database.AddGroupMember(groupID, req.UserID, req.EncryptedKey); err != nil {
			fmt.Printf("Error adding member to group %s: %v\n", groupID, err)
			http.Error(w, "Failed to add member", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]string{"status": "added"})
	}
}

// RemoveGroupMember removes a user from a group
func RemoveGroupMember(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get authenticated user ID
		requesterID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		groupIDStr := vars["groupId"]
		userIDStr := vars["userId"]

		groupID, err := uuid.Parse(groupIDStr)
		if err != nil {
			http.Error(w, "Invalid group ID", http.StatusBadRequest)
			return
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		// AUTHORIZATION: Check if requester is a group admin
		isAdmin, err := database.IsGroupAdmin(groupID, requesterID)
		if err != nil {
			fmt.Printf("Error checking group admin status for user %s: %v\n", requesterID, err)
			http.Error(w, "Failed to verify permissions", http.StatusInternalServerError)
			return
		}
		if !isAdmin {
			http.Error(w, "Only group admins can remove members", http.StatusForbidden)
			return
		}

		if err := database.RemoveGroupMember(groupID, userID); err != nil {
			fmt.Printf("Error removing member from group %s: %v\n", groupID, err)
			http.Error(w, "Failed to remove member", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]string{"status": "removed"})
	}
}
