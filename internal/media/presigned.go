package media

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MediaService handles presigned URL generation for media uploads/downloads
// Media is encrypted client-side before upload - server never sees plaintext
type MediaService struct {
	client     *minio.Client
	bucket     string
	cdnBaseURL string // Optional CDN URL for downloads
}

// UploadURLResult contains the presigned upload URL and metadata
type UploadURLResult struct {
	MediaID   uuid.UUID `json:"media_id"`
	UploadURL string    `json:"upload_url"`
	ExpiresIn int       `json:"expires_in"` // seconds
	MaxSize   int64     `json:"max_size"`   // bytes
}

// DownloadURLResult contains the presigned download URL
type DownloadURLResult struct {
	MediaID     uuid.UUID `json:"media_id"`
	DownloadURL string    `json:"download_url"`
	ExpiresIn   int       `json:"expires_in"`
	CacheHit    bool      `json:"cache_hit"` // true if served from CDN edge
}

// NewMediaService creates a new media service
func NewMediaService(endpoint, accessKey, secretKey, bucket string, useSSL bool, cdnBaseURL string) (*MediaService, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	// Ensure bucket exists
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
	}

	return &MediaService{
		client:     client,
		bucket:     bucket,
		cdnBaseURL: cdnBaseURL,
	}, nil
}

// GenerateUploadURL creates a presigned PUT URL for direct client upload
// The client uploads encrypted bytes directly to blob storage
func (m *MediaService) GenerateUploadURL(contentType string, maxSize int64) (*UploadURLResult, error) {
	mediaID := uuid.New()
	objectName := fmt.Sprintf("media/%s", mediaID.String())

	// Generate presigned PUT URL (15 minutes validity)
	expiry := 15 * time.Minute

	presignedURL, err := m.client.PresignedPutObject(
		context.Background(),
		m.bucket,
		objectName,
		expiry,
	)
	if err != nil {
		return nil, err
	}

	return &UploadURLResult{
		MediaID:   mediaID,
		UploadURL: presignedURL.String(),
		ExpiresIn: int(expiry.Seconds()),
		MaxSize:   maxSize,
	}, nil
}

// GenerateDownloadURL creates a presigned GET URL for client download
// If CDN is configured, returns CDN URL for edge caching
func (m *MediaService) GenerateDownloadURL(mediaID uuid.UUID) (*DownloadURLResult, error) {
	objectName := fmt.Sprintf("media/%s", mediaID.String())

	// If CDN is configured, use CDN URL with signed query params
	if m.cdnBaseURL != "" {
		cdnURL, err := m.generateCDNURL(mediaID)
		if err != nil {
			return nil, err
		}
		return &DownloadURLResult{
			MediaID:     mediaID,
			DownloadURL: cdnURL,
			ExpiresIn:   3600, // 1 hour for CDN
			CacheHit:    true,
		}, nil
	}

	// Direct presigned URL from MinIO (1 hour validity)
	expiry := 1 * time.Hour

	presignedURL, err := m.client.PresignedGetObject(
		context.Background(),
		m.bucket,
		objectName,
		expiry,
		url.Values{},
	)
	if err != nil {
		return nil, err
	}

	return &DownloadURLResult{
		MediaID:     mediaID,
		DownloadURL: presignedURL.String(),
		ExpiresIn:   int(expiry.Seconds()),
		CacheHit:    false,
	}, nil
}

// generateCDNURL creates a signed CDN URL
func (m *MediaService) generateCDNURL(mediaID uuid.UUID) (string, error) {
	// CDN URL format: https://cdn.example.com/media/{mediaID}?token={signed_token}&expires={timestamp}
	expiry := time.Now().Add(1 * time.Hour).Unix()

	// In production, generate HMAC signature for CDN authentication
	// For now, return simple URL
	return fmt.Sprintf("%s/media/%s?expires=%d", m.cdnBaseURL, mediaID.String(), expiry), nil
}

// GenerateThumbnailUploadURL creates upload URL for encrypted thumbnail
func (m *MediaService) GenerateThumbnailUploadURL(mediaID uuid.UUID) (*UploadURLResult, error) {
	objectName := fmt.Sprintf("thumbnails/%s", mediaID.String())

	expiry := 15 * time.Minute

	presignedURL, err := m.client.PresignedPutObject(
		context.Background(),
		m.bucket,
		objectName,
		expiry,
	)
	if err != nil {
		return nil, err
	}

	return &UploadURLResult{
		MediaID:   mediaID,
		UploadURL: presignedURL.String(),
		ExpiresIn: int(expiry.Seconds()),
		MaxSize:   500 * 1024, // 500KB max for thumbnails
	}, nil
}

// GenerateThumbnailDownloadURL creates download URL for thumbnail
func (m *MediaService) GenerateThumbnailDownloadURL(mediaID uuid.UUID) (*DownloadURLResult, error) {
	objectName := fmt.Sprintf("thumbnails/%s", mediaID.String())

	expiry := 1 * time.Hour

	presignedURL, err := m.client.PresignedGetObject(
		context.Background(),
		m.bucket,
		objectName,
		expiry,
		url.Values{},
	)
	if err != nil {
		return nil, err
	}

	return &DownloadURLResult{
		MediaID:     mediaID,
		DownloadURL: presignedURL.String(),
		ExpiresIn:   int(expiry.Seconds()),
	}, nil
}

// DeleteMedia removes media from storage
func (m *MediaService) DeleteMedia(mediaID uuid.UUID) error {
	objectName := fmt.Sprintf("media/%s", mediaID.String())
	return m.client.RemoveObject(
		context.Background(),
		m.bucket,
		objectName,
		minio.RemoveObjectOptions{},
	)
}

// GetMediaInfo returns metadata about stored media (size, last modified)
func (m *MediaService) GetMediaInfo(mediaID uuid.UUID) (map[string]interface{}, error) {
	objectName := fmt.Sprintf("media/%s", mediaID.String())

	info, err := m.client.StatObject(
		context.Background(),
		m.bucket,
		objectName,
		minio.StatObjectOptions{},
	)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"media_id":      mediaID,
		"size":          info.Size,
		"content_type":  info.ContentType,
		"last_modified": info.LastModified,
		"etag":          info.ETag,
	}, nil
}
