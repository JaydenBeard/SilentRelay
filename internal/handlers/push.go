package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/jaydenbeard/messaging-app/internal/push"
)

// PushHandler handles push notification related endpoints
type PushHandler struct {
	deviceStore *push.DeviceStore
	pushService *push.PushService
}

// NewPushHandler creates a new push handler
func NewPushHandler(deviceStore *push.DeviceStore, pushService *push.PushService) *PushHandler {
	return &PushHandler{
		deviceStore: deviceStore,
		pushService: pushService,
	}
}

// RegisterTokenRequest is the request body for registering a device token
type RegisterTokenRequest struct {
	Token    string `json:"token"`
	Platform string `json:"platform"` // "ios" or "android"
	BundleID string `json:"bundle_id"`
}

// RegisterToken handles POST /api/v1/devices/push-token
func (h *PushHandler) RegisterToken(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req RegisterTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Token == "" {
		http.Error(w, "Token is required", http.StatusBadRequest)
		return
	}

	if req.Platform == "" {
		req.Platform = "ios"
	}

	err := h.deviceStore.RegisterToken(r.Context(), userID, req.Token, req.Platform, req.BundleID)
	if err != nil {
		log.Printf("[Push] Failed to register token for user %s: %v", userID, err)
		http.Error(w, "Failed to register token", http.StatusInternalServerError)
		return
	}

	log.Printf("[Push] Registered %s token for user %s", req.Platform, userID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// UnregisterToken handles DELETE /api/v1/devices/push-token
func (h *PushHandler) UnregisterToken(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req RegisterTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Token == "" {
		// Remove all tokens for user
		err := h.deviceStore.RemoveAllTokensForUser(r.Context(), userID)
		if err != nil {
			http.Error(w, "Failed to remove tokens", http.StatusInternalServerError)
			return
		}
	} else {
		// Remove specific token
		err := h.deviceStore.RemoveToken(r.Context(), req.Token)
		if err != nil {
			http.Error(w, "Failed to remove token", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// TestPush handles POST /api/v1/devices/test-push (for testing)
func (h *PushHandler) TestPush(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err := h.pushService.SendToUser(r.Context(), userID, &push.Notification{
		Title:    "Test Notification",
		Body:     "Push notifications are working!",
		Sound:    "default",
		Priority: 10,
		PushType: "alert",
	})

	if err != nil {
		log.Printf("[Push] Test push failed for user %s: %v", userID, err)
		http.Error(w, "Failed to send test notification", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
}
