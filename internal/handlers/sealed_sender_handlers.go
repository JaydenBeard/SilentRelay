package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jaydenbeard/messaging-app/internal/db"
	"github.com/jaydenbeard/messaging-app/internal/middleware"
	security "github.com/jaydenbeard/messaging-app/internal/security"
)

// IssueSealedSenderCertificate issues a new sealed sender certificate for the authenticated user
func IssueSealedSenderCertificate(sealedSenderManager *security.SealedSenderIdentityCertificateManager, database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Parse request body
		var req security.SealedSenderIdentityCertificateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate that the user ID matches the authenticated user
		if req.UserID != userID {
			http.Error(w, "User ID mismatch", http.StatusForbidden)
			return
		}

		// Issue the certificate
		cert, err := sealedSenderManager.IssueCertificateWithPersistence(userID, req.PublicKey)
		if err != nil {
			http.Error(w, "Failed to issue certificate: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, cert)
	}
}

// GetSealedSenderCertificates retrieves all valid sealed sender certificates for the authenticated user
func GetSealedSenderCertificates(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		certificates, err := database.GetUserSealedSenderCertificates(userID)
		if err != nil {
			http.Error(w, "Failed to retrieve certificates", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, certificates)
	}
}

// GetSealedSenderCertificate retrieves a specific sealed sender certificate by ID
func GetSealedSenderCertificate(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		certificateIDStr := vars["certificateId"]

		certificateID, err := uuid.Parse(certificateIDStr)
		if err != nil {
			http.Error(w, "Invalid certificate ID", http.StatusBadRequest)
			return
		}

		cert, err := database.GetSealedSenderCertificate(certificateID)
		if err != nil {
			http.Error(w, "Certificate not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, cert)
	}
}

// RevokeSealedSenderCertificate revokes a sealed sender certificate
func RevokeSealedSenderCertificate(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		certificateIDStr := vars["certificateId"]

		certificateID, err := uuid.Parse(certificateIDStr)
		if err != nil {
			http.Error(w, "Invalid certificate ID", http.StatusBadRequest)
			return
		}

		// Verify the certificate belongs to this user before revoking
		cert, err := database.GetSealedSenderCertificate(certificateID)
		if err != nil {
			http.Error(w, "Certificate not found", http.StatusNotFound)
			return
		}

		if cert.UserID != userID {
			http.Error(w, "Certificate does not belong to this user", http.StatusForbidden)
			return
		}

		if err := database.RevokeSealedSenderCertificate(certificateID); err != nil {
			http.Error(w, "Failed to revoke certificate", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]string{
			"status":         "revoked",
			"certificate_id": certificateID.String(),
		})
	}
}

// VerifySealedSenderCertificate verifies the authenticity of a sealed sender certificate
func VerifySealedSenderCertificate(sealedSenderManager *security.SealedSenderIdentityCertificateManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req security.SealedSenderIdentityCertificate
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		valid, err := sealedSenderManager.VerifyCertificate(&req)
		if err != nil {
			http.Error(w, "Verification failed: "+err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]interface{}{
			"valid":          valid,
			"certificate_id": req.CertificateID.String(),
			"user_id":        req.UserID.String(),
			"expires_at":     req.Expiration,
		})
	}
}

// GetCAPublicKey returns the CA public key for client-side certificate verification
func GetCAPublicKey(sealedSenderManager *security.SealedSenderIdentityCertificateManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caPublicKey, err := sealedSenderManager.GetCAPublicKey()
		if err != nil {
			http.Error(w, "Failed to get CA public key", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]interface{}{
			"ca_public_key": caPublicKey,
			"algorithm":     "ECDSA-P256",
			"issued_at":     time.Now().UTC(),
		})
	}
}

// CleanupExpiredSealedSenderCertificates cleans up expired certificates (admin endpoint)
func CleanupExpiredSealedSenderCertificates(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Admin-only endpoint - check for admin privileges
		// In a production system, this would require proper admin authentication
		// For now, we'll allow it for demonstration purposes

		err := database.CleanupExpiredSealedSenderCertificates()
		if err != nil {
			http.Error(w, "Failed to cleanup expired certificates", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]interface{}{
			"status":    "success",
			"timestamp": time.Now().UTC(),
		})
	}
}
