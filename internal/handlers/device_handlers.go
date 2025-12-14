package handlers

// Device management handlers including device linking, PIN management, and device approval.

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jaydenbeard/messaging-app/internal/db"
	"github.com/jaydenbeard/messaging-app/internal/middleware"
	"github.com/jaydenbeard/messaging-app/internal/models"
	"github.com/jaydenbeard/messaging-app/internal/websocket"
)

// ================== Device Management ==================

// GetDevices returns all devices for the current user
func GetDevices(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		devices, err := database.GetUserDevices(userID)
		if err != nil {
			http.Error(w, "Failed to get devices", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, devices)
	}
}

// RemoveDevice removes a device from the user's account
func RemoveDevice(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		deviceIDStr := vars["deviceId"]
		deviceID, err := uuid.Parse(deviceIDStr)
		if err != nil {
			http.Error(w, "Invalid device ID", http.StatusBadRequest)
			return
		}

		if err := database.RemoveDevice(userID, deviceID); err != nil {
			http.Error(w, "Failed to remove device", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]string{"status": "removed"})
	}
}

// ================== PIN Management ==================

// GetPIN retrieves the user's PIN settings
func GetPIN(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		pinHash, pinLength, err := database.GetUserPIN(userID)
		if err != nil {
			http.Error(w, "Failed to get PIN", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]interface{}{
			"has_pin":    pinHash != "",
			"pin_hash":   pinHash,
			"pin_length": pinLength,
		})
	}
}

// SetPIN sets or updates the user's PIN
func SetPIN(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req struct {
			PinHash   string `json:"pin_hash"`
			PinLength int    `json:"pin_length"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.PinLength != 4 && req.PinLength != 6 {
			http.Error(w, "PIN must be 4 or 6 digits", http.StatusBadRequest)
			return
		}

		if err := database.SaveUserPIN(userID, req.PinHash, req.PinLength); err != nil {
			http.Error(w, "Failed to save PIN", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]string{"status": "saved"})
	}
}

// DeletePIN removes the user's PIN
func DeletePIN(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if err := database.DeleteUserPIN(userID); err != nil {
			http.Error(w, "Failed to delete PIN", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]string{"status": "deleted"})
	}
}

// ================== Device Approval (Secure Linking) ==================

// generateApprovalCode generates a 6-digit approval code
func generateApprovalCode() string {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		log.Printf("Warning: failed to generate random bytes: %v", err)
		return "000000"
	}
	code := int(b[0])<<16 | int(b[1])<<8 | int(b[2])
	return fmt.Sprintf("%06d", code%1000000)
}

// RequestDeviceApproval initiates a device linking request
func RequestDeviceApproval(database *db.PostgresDB, hub *websocket.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			PhoneNumber string `json:"phone_number"`
			DeviceID    string `json:"device_id"`
			DeviceName  string `json:"device_name"`
			DeviceType  string `json:"device_type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Get user by phone
		userID, err := database.GetUserByPhone(req.PhoneNumber)
		if err != nil || userID == nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		deviceID, err := uuid.Parse(req.DeviceID)
		if err != nil {
			http.Error(w, "Invalid device ID", http.StatusBadRequest)
			return
		}

		// Check if device is already linked (active)
		isLinked, err := database.IsDeviceLinked(*userID, deviceID)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if isLinked {
			// Device already linked and active - check if it's the primary device
			isPrimary, err := database.IsPrimaryDevice(*userID, deviceID)
			if err != nil {
				// Log but continue - isPrimary defaults to false
				fmt.Printf("Warning: Failed to check primary device status: %v\n", err)
			}
			w.Header().Set("Content-Type", "application/json")
			writeJSON(w, map[string]interface{}{
				"status":        "already_linked",
				"requires_code": false,
				"is_primary":    isPrimary,
			})
			return
		}

		// Check if device was previously linked but is now inactive
		wasPreviouslyLinked, err := database.WasDeviceEverLinked(*userID, deviceID)
		if err == nil && wasPreviouslyLinked {
			isPrimary, err := database.IsPrimaryDevice(*userID, deviceID)
			if err != nil {
				// Log but continue - isPrimary defaults to false
				fmt.Printf("Warning: Failed to check primary device status: %v\n", err)
			}
			w.Header().Set("Content-Type", "application/json")
			writeJSON(w, map[string]interface{}{
				"status":        "previously_linked",
				"requires_code": false,
				"is_primary":    isPrimary,
				"message":       "Device recognized, will be reactivated on login",
			})
			return
		}

		// Check if user has ANY devices at all
		hasDevices, err := database.HasLinkedDevices(*userID)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if !hasDevices {
			w.Header().Set("Content-Type", "application/json")
			writeJSON(w, map[string]interface{}{
				"status":          "first_device",
				"requires_code":   false,
				"will_be_primary": true,
			})
			return
		}

		// User has devices, but this one isn't linked - check if there's a primary device
		primaryDevice, err := database.GetPrimaryDevice(*userID)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if primaryDevice == nil {
			w.Header().Set("Content-Type", "application/json")
			writeJSON(w, map[string]interface{}{
				"status":          "first_device",
				"requires_code":   false,
				"will_be_primary": true,
			})
			return
		}

		// Generate approval code and create request
		code := generateApprovalCode()
		approvalReq, err := database.CreateDeviceApprovalRequest(*userID, deviceID, req.DeviceName, req.DeviceType, code)
		if err != nil {
			http.Error(w, "Failed to create approval request", http.StatusInternalServerError)
			return
		}

		// Notify ONLY the primary device via WebSocket
		hub.SendToDevice(primaryDevice.DeviceID, map[string]interface{}{
			"type": "device_approval_request",
			"payload": map[string]interface{}{
				"request_id":  approvalReq.RequestID,
				"device_name": req.DeviceName,
				"device_type": req.DeviceType,
				"code":        code,
				"expires_at":  approvalReq.ExpiresAt,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]interface{}{
			"status":              "pending",
			"requires_code":       true,
			"request_id":          approvalReq.RequestID,
			"expires_at":          approvalReq.ExpiresAt,
			"primary_device_name": primaryDevice.DeviceName,
		})
	}
}

// SetPrimaryDevice changes which device is the primary device
func SetPrimaryDevice(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		deviceIDStr := vars["deviceId"]

		deviceID, err := uuid.Parse(deviceIDStr)
		if err != nil {
			http.Error(w, "Invalid device ID", http.StatusBadRequest)
			return
		}

		// Verify the device belongs to this user
		isLinked, err := database.IsDeviceLinked(userID, deviceID)
		if err != nil || !isLinked {
			http.Error(w, "Device not found", http.StatusNotFound)
			return
		}

		// Set as primary
		if err := database.SetPrimaryDevice(userID, deviceID); err != nil {
			http.Error(w, "Failed to set primary device", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]string{"status": "primary_updated"})
	}
}

// GetPendingApprovals returns pending device approval requests for the authenticated user
func GetPendingApprovals(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		requests, err := database.GetPendingApprovalRequests(userID)
		if err != nil {
			http.Error(w, "Failed to get requests", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, requests)
	}
}

// ApproveDevice approves a device linking request
func ApproveDevice(database *db.PostgresDB, hub *websocket.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		requestIDStr := vars["requestId"]
		requestID, err := uuid.Parse(requestIDStr)
		if err != nil {
			http.Error(w, "Invalid request ID", http.StatusBadRequest)
			return
		}

		// SECURITY: Get and validate approver's device ID
		deviceIDStr := r.Header.Get("X-Device-ID")
		if deviceIDStr == "" {
			http.Error(w, "Device ID required in X-Device-ID header", http.StatusBadRequest)
			return
		}

		approverDeviceID, err := uuid.Parse(deviceIDStr)
		if err != nil {
			http.Error(w, "Invalid device ID format", http.StatusBadRequest)
			return
		}

		// SECURITY: Verify device belongs to this user
		isLinked, err := database.IsDeviceLinked(userID, approverDeviceID)
		if err != nil || !isLinked {
			http.Error(w, "Device not found or not linked to your account", http.StatusForbidden)
			return
		}

		// SECURITY: Verify device is primary (only primary device can approve)
		isPrimary, err := database.IsPrimaryDevice(userID, approverDeviceID)
		if err != nil || !isPrimary {
			http.Error(w, "Only primary device can approve new devices", http.StatusForbidden)
			return
		}

		if err := database.ApproveDeviceRequest(requestID, approverDeviceID); err != nil {
			fmt.Printf("Error approving device request %s: %v\n", requestID, err)
			http.Error(w, "Failed to approve device", http.StatusBadRequest)
			return
		}

		// Notify the new device that it's been approved
		hub.BroadcastToUser(userID, map[string]interface{}{
			"type": "device_approved",
			"payload": map[string]interface{}{
				"request_id": requestID,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]string{"status": "approved"})
	}
}

// DenyDevice denies a device linking request
func DenyDevice(database *db.PostgresDB, hub *websocket.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		requestIDStr := vars["requestId"]
		requestID, err := uuid.Parse(requestIDStr)
		if err != nil {
			http.Error(w, "Invalid request ID", http.StatusBadRequest)
			return
		}

		if err := database.DenyDeviceRequest(requestID); err != nil {
			http.Error(w, "Failed to deny request", http.StatusInternalServerError)
			return
		}

		// Notify the new device that it's been denied
		hub.BroadcastToUser(userID, map[string]interface{}{
			"type": "device_denied",
			"payload": map[string]interface{}{
				"request_id": requestID,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]string{"status": "denied"})
	}
}

// CheckApprovalStatus checks the status of an approval request
func CheckApprovalStatus(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		requestIDStr := vars["requestId"]
		requestID, err := uuid.Parse(requestIDStr)
		if err != nil {
			http.Error(w, "Invalid request ID", http.StatusBadRequest)
			return
		}

		status, err := database.CheckApprovalStatus(requestID)
		if err != nil {
			http.Error(w, "Request not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]string{"status": status})
	}
}

// VerifyApprovalCode verifies the code entered on the new device
func VerifyApprovalCode(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			PhoneNumber string `json:"phone_number"`
			Code        string `json:"code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Get user by phone
		userID, err := database.GetUserByPhone(req.PhoneNumber)
		if err != nil || userID == nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Verify the code
		approvalReq, err := database.VerifyApprovalCode(*userID, req.Code)
		if err != nil {
			http.Error(w, "Invalid or expired code", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]interface{}{
			"status":     "valid",
			"request_id": approvalReq.RequestID,
		})
	}
}

// ================== Key Management ==================

// UploadPrekeys stores new one-time pre-keys
func UploadPrekeys(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req struct {
			Prekeys []models.PreKey `json:"prekeys"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		prekeys := make([]struct {
			ID        int
			PublicKey string
		}, len(req.Prekeys))

		for i, pk := range req.Prekeys {
			prekeys[i] = struct {
				ID        int
				PublicKey string
			}{
				ID:        int(pk.PreKeyID),
				PublicKey: pk.PublicKey,
			}
		}

		if err := database.SavePreKeys(userID, prekeys); err != nil {
			http.Error(w, "Failed to save prekeys", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]string{"status": "uploaded"})
	}
}

// GetUserKeys returns a user's public keys for E2EE session
func GetUserKeys(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		userIDStr := vars["userId"]

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		keys, err := database.GetUserKeys(userID)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, keys)
	}
}

// UpdateKeys allows a user to update their encryption keys (e.g., when setting up a new device)
// If the identity key changes, all contacts are notified via WebSocket
// POST /api/v1/users/keys
func UpdateKeys(database *db.PostgresDB, hub *websocket.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req struct {
			PublicIdentityKey     string `json:"public_identity_key"`
			PublicSignedPrekey    string `json:"public_signed_prekey"`
			SignedPrekeySignature string `json:"signed_prekey_signature"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if req.PublicIdentityKey == "" || req.PublicSignedPrekey == "" {
			http.Error(w, "Missing required key fields", http.StatusBadRequest)
			return
		}

		// Update keys in database
		identityKeyChanged, err := database.UpdateUserKeys(userID, req.PublicIdentityKey, req.PublicSignedPrekey, req.SignedPrekeySignature)
		if err != nil {
			log.Printf("[Keys] Failed to update keys for user %s: %v", userID, err)
			http.Error(w, "Failed to update keys", http.StatusInternalServerError)
			return
		}

		// If identity key changed, notify all contacts
		if identityKeyChanged {
			log.Printf("[Security] Broadcasting identity_key_changed for user %s", userID)

			// Get all users who have exchanged messages with this user
			contacts, err := database.GetMessagedUsers(userID)
			if err != nil {
				log.Printf("[Security] Warning: failed to get contacts for identity notification: %v", err)
			} else {
				// Build the payload
				payload, _ := json.Marshal(map[string]interface{}{
					"user_id":          userID.String(),
					"new_identity_key": req.PublicIdentityKey,
				})

				// Send notification to each contact
				for _, contactID := range contacts {
					msg := &models.WebSocketMessage{
						Type:      "identity_key_changed",
						SenderID:  userID,
						Timestamp: time.Now().UTC(),
						Payload:   payload,
					}
					hub.SendToUser(contactID.String(), msg)
				}
				log.Printf("[Security] Notified %d contacts about identity key change for user %s", len(contacts), userID)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]interface{}{
			"status":               "updated",
			"identity_key_changed": identityKeyChanged,
		})
	}
}

// ================== User Discovery ==================

// CheckUsername checks if a username is available
func CheckUsername(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		username := vars["username"]

		if username == "" {
			http.Error(w, "Username required", http.StatusBadRequest)
			return
		}

		// Validate username format
		if err := validateUsername(username); err != nil {
			w.Header().Set("Content-Type", "application/json")
			writeJSON(w, map[string]interface{}{
				"available": false,
				"message":   err.Error(),
			})
			return
		}

		// Check availability
		available, err := database.CheckUsernameAvailable(username)
		if err != nil {
			http.Error(w, "Failed to check username", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if available {
			writeJSON(w, map[string]interface{}{
				"available": true,
			})
		} else {
			writeJSON(w, map[string]interface{}{
				"available": false,
				"message":   "Username is already taken",
			})
		}
	}
}

// SearchUsers finds users by username
func SearchUsers(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get current user ID to filter out users who blocked them
		currentUserID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		query := r.URL.Query().Get("q")
		if query == "" {
			http.Error(w, "Search query required", http.StatusBadRequest)
			return
		}

		// SECURITY: Validate query length to prevent enumeration and abuse
		if len(query) < 3 {
			http.Error(w, "Search query must be at least 3 characters", http.StatusBadRequest)
			return
		}

		if len(query) > 50 {
			http.Error(w, "Search query too long (max 50 characters)", http.StatusBadRequest)
			return
		}

		limit := 20
		if l := r.URL.Query().Get("limit"); l != "" {
			// Reduced max limit from 100 to 20 to prevent enumeration
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 20 {
				limit = parsed
			}
		}

		// Use SearchUsersExcludingBlockers to filter out users who blocked the searcher
		users, err := database.SearchUsersExcludingBlockers(query, currentUserID, limit)
		if err != nil {
			fmt.Printf("Error searching users: %v\n", err)
			http.Error(w, "Search failed", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, users)
	}
}
