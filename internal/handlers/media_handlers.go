package handlers

// Media handlers for file upload, download, and privacy settings.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jaydenbeard/messaging-app/internal/config"
	"github.com/jaydenbeard/messaging-app/internal/db"
	"github.com/jaydenbeard/messaging-app/internal/middleware"
	"github.com/jaydenbeard/messaging-app/internal/websocket"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ================== Media Handlers ==================

// GetUploadURL returns a presigned URL for media upload
func GetUploadURL(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			FileName    string `json:"file_name"`
			ContentType string `json:"content_type"`
			FileSize    int64  `json:"file_size"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate file upload parameters
		if err := validateFileUpload(req.FileName, req.ContentType, req.FileSize, cfg.MediaLimits); err != nil {
			log.Printf("SECURITY: File upload validation failed: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Generate media ID
		mediaID := uuid.New()

		// Generate presigned URL from MinIO/S3
		useHTTPS := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
		scheme := "http"
		if useHTTPS {
			scheme = "https"
		}

		// For now, proxy through backend to avoid mixed content
		baseURL := scheme + "://" + r.Host
		uploadURL := baseURL + "/api/v1/media/upload-proxy/" + mediaID.String()
		downloadURL := baseURL + "/api/v1/media/download-proxy/" + mediaID.String()

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]interface{}{
			"fileId":      mediaID.String(),
			"uploadUrl":   uploadURL,
			"downloadUrl": downloadURL,
			"expiresIn":   3600,
		})
	}
}

// GetPrivacySettings returns the user's privacy settings
func GetPrivacySettings(database *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		settings, err := database.GetPrivacySettings(userID)
		if err != nil {
			http.Error(w, "Failed to get privacy settings", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, settings)
	}
}

// UpdatePrivacySetting updates a specific privacy setting
func UpdatePrivacySetting(database *db.PostgresDB, hub *websocket.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req struct {
			Setting string `json:"setting"`
			Value   bool   `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := database.UpdatePrivacySetting(userID, req.Setting, req.Value); err != nil {
			fmt.Printf("Error updating privacy setting for user %s: %v\n", userID, err)
			http.Error(w, "Failed to update setting", http.StatusInternalServerError)
			return
		}

		// If online status was changed, broadcast presence update immediately
		if req.Setting == "show_online_status" && hub != nil {
			hub.BroadcastPresenceUpdate(userID, true)
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]string{"status": "updated"})
	}
}

// GetMediaURL returns a presigned URL for media download
func GetMediaURL(database *db.PostgresDB, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		mediaIDStr := vars["mediaId"]

		mediaID, err := uuid.Parse(mediaIDStr)
		if err != nil {
			http.Error(w, "Invalid media ID", http.StatusBadRequest)
			return
		}

		// Generate presigned download URL
		useHTTPS := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
		scheme := "http"
		if useHTTPS {
			scheme = "https"
		}

		// Proxy through backend to avoid mixed content
		baseURL := scheme + "://" + r.Host
		downloadURL := baseURL + "/api/v1/media/download-proxy/" + mediaID.String()

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]interface{}{
			"url":       downloadURL,
			"expiresIn": 3600,
		})
	}
}

// UploadProxy proxies file uploads to MinIO with streaming and size limits for DoS protection
func UploadProxy(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		mediaIDStr := vars["mediaId"]

		mediaID, err := uuid.Parse(mediaIDStr)
		if err != nil {
			log.Printf("SECURITY: Upload attempt with invalid media ID: %s", mediaIDStr)
			http.Error(w, "Invalid media ID", http.StatusBadRequest)
			return
		}

		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			log.Printf("SECURITY: Upload attempt without Content-Type header for media %s", mediaID)
			http.Error(w, "Content-Type header required", http.StatusBadRequest)
			return
		}

		// Determine size limit based on content type
		var maxSize int64
		switch {
		case strings.HasPrefix(contentType, "image/"):
			maxSize = cfg.MediaLimits.MaxImageSize
		case strings.HasPrefix(contentType, "video/"):
			maxSize = cfg.MediaLimits.MaxVideoSize
		case strings.HasPrefix(contentType, "audio/"):
			maxSize = cfg.MediaLimits.MaxAudioSize
		default:
			maxSize = cfg.MediaLimits.MaxFileSize
		}

		// Log upload attempt
		clientIP := getClientIP(r)
		log.Printf("SECURITY: Media upload attempt - ID: %s, Type: %s, MaxSize: %d bytes, ClientIP: %s",
			mediaID, contentType, maxSize, clientIP)

		// Initialize MinIO client
		useSSL := strings.HasPrefix(cfg.MinioURL, "https://")
		endpoint := strings.TrimPrefix(cfg.MinioURL, "http://")
		endpoint = strings.TrimPrefix(endpoint, "https://")

		minioClient, err := minio.New(endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.MinioKey, cfg.MinioSecret, ""),
			Secure: useSSL,
		})
		if err != nil {
			log.Printf("SECURITY: Failed to create MinIO client for upload %s: %v", mediaID, err)
			http.Error(w, "Failed to connect to storage", http.StatusInternalServerError)
			return
		}

		// Object name in MinIO
		objectName := fmt.Sprintf("media/%s", mediaID.String())

		// Check if bucket exists
		exists, err := minioClient.BucketExists(context.Background(), cfg.MinioBucket)
		if err != nil {
			log.Printf("SECURITY: Error checking bucket %s for upload %s: %v", cfg.MinioBucket, mediaID, err)
			http.Error(w, "Failed to check storage bucket", http.StatusInternalServerError)
			return
		}
		if !exists {
			err = minioClient.MakeBucket(context.Background(), cfg.MinioBucket, minio.MakeBucketOptions{})
			if err != nil {
				log.Printf("SECURITY: Error creating bucket %s for upload %s: %v", cfg.MinioBucket, mediaID, err)
				http.Error(w, "Failed to create storage bucket", http.StatusInternalServerError)
				return
			}
		}

		// Use streaming upload with size limit to prevent memory exhaustion
		limitedReader := io.LimitReader(r.Body, maxSize+1)

		// Create a pipe for streaming
		pipeReader, pipeWriter := io.Pipe()

		// Start goroutine to copy from limited reader to pipe
		go func() {
			defer func() {
				if err := pipeWriter.Close(); err != nil {
					log.Printf("Warning: failed to close pipe writer: %v", err)
				}
			}()
			bytesCopied, err := io.Copy(pipeWriter, limitedReader)
			if err != nil {
				log.Printf("SECURITY: Error copying upload data for %s: %v", mediaID, err)
				return
			}
			if bytesCopied > maxSize {
				log.Printf("SECURITY: Upload size limit exceeded for %s: %d > %d", mediaID, bytesCopied, maxSize)
			}
		}()

		// Stream directly to MinIO without loading into memory
		_, err = minioClient.PutObject(
			context.Background(),
			cfg.MinioBucket,
			objectName,
			pipeReader,
			-1,
			minio.PutObjectOptions{
				ContentType: contentType,
			},
		)

		if err != nil {
			log.Printf("SECURITY: Upload failed for media %s to bucket %s: %v", mediaID, cfg.MinioBucket, err)
			if strings.Contains(err.Error(), "unexpected EOF") || strings.Contains(err.Error(), "size") {
				http.Error(w, fmt.Sprintf("File size exceeds maximum allowed size of %d bytes", maxSize), http.StatusRequestEntityTooLarge)
			} else {
				http.Error(w, "Upload failed", http.StatusInternalServerError)
			}
			return
		}

		log.Printf("SECURITY: Upload successful - ID: %s, Type: %s, ClientIP: %s", mediaID, contentType, clientIP)

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]interface{}{
			"fileId": mediaID.String(),
			"status": "uploaded",
		})
	}
}

// DownloadProxy proxies file downloads from MinIO (to avoid mixed content)
func DownloadProxy(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		mediaIDStr := vars["mediaId"]
		fmt.Printf("[Download] Request for media: %s\n", mediaIDStr)

		mediaID, err := uuid.Parse(mediaIDStr)
		if err != nil {
			fmt.Printf("[Download] Invalid media ID: %s\n", mediaIDStr)
			http.Error(w, "Invalid media ID", http.StatusBadRequest)
			return
		}

		// Initialize MinIO client
		useSSL := strings.HasPrefix(cfg.MinioURL, "https://")
		endpoint := strings.TrimPrefix(cfg.MinioURL, "http://")
		endpoint = strings.TrimPrefix(endpoint, "https://")
		fmt.Printf("[Download] MinIO endpoint=%s, useSSL=%v, bucket=%s\n", endpoint, useSSL, cfg.MinioBucket)

		minioClient, err := minio.New(endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.MinioKey, cfg.MinioSecret, ""),
			Secure: useSSL,
		})
		if err != nil {
			fmt.Printf("[Download] Failed to create MinIO client: %v\n", err)
			http.Error(w, "Failed to connect to storage", http.StatusInternalServerError)
			return
		}

		// Object name in MinIO
		objectName := fmt.Sprintf("media/%s", mediaID.String())
		fmt.Printf("[Download] Looking for object: %s\n", objectName)

		// Get object from MinIO
		obj, err := minioClient.GetObject(
			context.Background(),
			cfg.MinioBucket,
			objectName,
			minio.GetObjectOptions{},
		)
		if err != nil {
			fmt.Printf("[Download] GetObject error: %v\n", err)
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		defer func() {
			if err := obj.Close(); err != nil {
				log.Printf("Warning: failed to close object: %v", err)
			}
		}()

		// Get object info for Content-Type
		objInfo, err := obj.Stat()
		if err != nil {
			fmt.Printf("[Download] Stat error: %v\n", err)
			http.Error(w, "Failed to get file info", http.StatusInternalServerError)
			return
		}

		fmt.Printf("[Download] Serving file: size=%d, contentType=%s\n", objInfo.Size, objInfo.ContentType)

		// Set headers
		w.Header().Set("Content-Type", objInfo.ContentType)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", objInfo.Size))

		// Stream the file to the client
		bytesWritten, err := io.Copy(w, obj)
		if err != nil {
			fmt.Printf("[Download] Error streaming file: %v\n", err)
		} else {
			fmt.Printf("[Download] Streamed %d bytes\n", bytesWritten)
		}
	}
}
