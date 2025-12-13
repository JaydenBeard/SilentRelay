# Media API

The Media API provides endpoints for secure media upload and download operations, supporting end-to-end encrypted attachments in messages.

## Overview

**Base Path**: `/api/v1/media`
**Authentication**: Requires valid JWT authentication
**Rate Limiting**: Normal rate limits with strict limits on uploads

## Media Upload Workflow

1. **Request Upload URL**: Get presigned URL for secure upload
2. **Encrypt Media**: Client encrypts media content
3. **Upload to Storage**: Direct upload to MinIO/S3
4. **Attach to Message**: Reference media ID in message
5. **Download Media**: Retrieve via presigned download URL

---

### 1. Get Upload URL

**Endpoint**: `POST /api/v1/media/upload-url`
**Description**: Generates a presigned URL for media upload with encryption requirements.

**Headers**:
- `Authorization: Bearer <access_token>`
- `Content-Type: application/json`

**Request Body**:
```json
{
  "file_name": "photo.jpg",
  "content_type": "image/jpeg",
  "file_size": 2048576,
  "encryption_method": "AES-256-GCM",
  "media_purpose": "message_attachment"
}
```

**Response**:
```json
{
  "fileId": "media-550e8400-e29b-41d4-a716-446655440000",
  "uploadUrl": "https://api.yourdomain.com/v1/media/upload-proxy/media-550e8400-e29b-41d4-a716-446655440000",
  "downloadUrl": "https://api.yourdomain.com/v1/media/download-proxy/media-550e8400-e29b-41d4-a716-446655440000",
  "expiresIn": 3600,
  "maxFileSize": 52428800,
  "encryptionRequirements": {
    "algorithm": "AES-256-GCM",
    "keyLength": 256,
    "ivLength": 12,
    "authTagLength": 16
  }
}
```

**Status Codes**:
- `200 OK`: Upload URL generated successfully
- `400 Bad Request`: Invalid request format or parameters
- `401 Unauthorized`: Invalid or missing authentication token
- `413 Payload Too Large`: File size exceeds maximum (50MB)
- `429 Too Many Requests`: Rate limit exceeded

**Rate Limiting**: 10 requests per minute per user

**Media Requirements**:
- **Maximum Size**: 50MB per file
- **Supported Types**: Images, videos, audio, documents
- **Encryption**: Mandatory client-side encryption
- **Content Types**: `image/*`, `video/*`, `audio/*`, `application/*`

---

### 2. Upload Media (Proxy)

**Endpoint**: `PUT|POST /api/v1/media/upload-proxy/{mediaId}`
**Description**: Proxy endpoint for media upload to avoid mixed content issues.

**Headers**:
- `Authorization: Bearer <access_token>`
- `Content-Type`: Actual media content type
- `X-Encryption-Info`: Encryption metadata

**Path Parameters**:
- `mediaId`: Media ID obtained from upload URL request

**Request Headers**:
```http
X-Encryption-Info: {
  "algorithm": "AES-256-GCM",
  "key_id": "encryption-key-uuid",
  "iv": "base64_initialization_vector"
}
```

**Response**:
```json
{
  "fileId": "media-550e8400-e29b-41d4-a716-446655440000",
  "status": "uploaded",
  "contentType": "image/jpeg",
  "size": 2048576,
  "encrypted": true,
  "checksum": "sha256-hash-of-encrypted-content"
}
```

**Status Codes**:
- `200 OK`: Media uploaded successfully
- `400 Bad Request`: Invalid media ID or missing encryption info
- `401 Unauthorized`: Invalid or missing authentication token
- `413 Payload Too Large`: File exceeds size limit
- `429 Too Many Requests`: Rate limit exceeded

**Rate Limiting**: 5 requests per minute per user

---

### 3. Get Download URL

**Endpoint**: `GET /api/v1/media/{mediaId}`
**Description**: Generates a presigned URL for media download.

**Headers**:
- `Authorization: Bearer <access_token>`

**Path Parameters**:
- `mediaId`: UUID of the media to download

**Response**:
```json
{
  "url": "https://api.yourdomain.com/v1/media/download-proxy/media-550e8400-e29b-41d4-a716-446655440000",
  "expiresIn": 3600
}
```

**Status Codes**:
- `200 OK`: Download URL generated successfully
- `400 Bad Request`: Invalid media ID format
- `401 Unauthorized`: Invalid or missing authentication token
- `403 Forbidden`: Not authorized to access this media
- `404 Not Found`: Media not found
- `429 Too Many Requests`: Rate limit exceeded

**Rate Limiting**: 30 requests per minute per user

---

### 4. Download Media (Proxy)

**Endpoint**: `GET /api/v1/media/download-proxy/{mediaId}`
**Description**: Proxy endpoint for media download to avoid mixed content issues.

**Headers**:
- `Authorization: Bearer <access_token>`

**Path Parameters**:
- `mediaId`: UUID of the media to download

**Response**:
- Binary media content with appropriate Content-Type headers

**Status Codes**:
- `200 OK`: Media downloaded successfully
- `400 Bad Request`: Invalid media ID format
- `401 Unauthorized`: Invalid or missing authentication token
- `403 Forbidden`: Not authorized to access this media
- `404 Not Found`: Media not found
- `429 Too Many Requests`: Rate limit exceeded

**Rate Limiting**: 60 requests per minute per user

---

## Media Security Model

### End-to-End Encryption

- **Client-Side Encryption**: Media encrypted before upload
- **Server Storage**: Only encrypted ciphertext stored
- **Key Management**: Encryption keys managed client-side
- **No Server Access**: Server cannot decrypt media content

### Content Protection

- **Content-Type Validation**: Prevents malicious file uploads
- **Size Limitations**: 50MB maximum per file
- **Rate Limiting**: Prevents storage abuse
- **Access Control**: Only authorized users can access media

### Privacy Features

- **Expiring URLs**: Presigned URLs expire after 1 hour
- **No Direct Access**: Media always proxied through API
- **Encryption Metadata**: Encryption info included in responses
- **Checksum Verification**: Integrity verification available

---

## Supported Media Types

| Category | MIME Types | Max Size | Notes |
|----------|------------|----------|-------|
| **Images** | `image/jpeg`, `image/png`, `image/gif`, `image/webp` | 10MB | Common image formats |
| **Videos** | `video/mp4`, `video/webm`, `video/quicktime` | 50MB | Common video formats |
| **Audio** | `audio/mpeg`, `audio/ogg`, `audio/wav`, `audio/aac` | 20MB | Common audio formats |
| **Documents** | `application/pdf`, `application/msword`, `application/vnd.openxmlformats-officedocument.wordprocessingml.document` | 50MB | Common document formats |
| **Archives** | `application/zip`, `application/x-rar-compressed` | 50MB | Compressed files |

---

## Examples

### Complete Media Upload Flow

```bash
# Step 1: Get upload URL
curl -X POST https://api.yourdomain.com/v1/media/upload-url \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "file_name": "vacation.jpg",
    "content_type": "image/jpeg",
    "file_size": 2048576,
    "encryption_method": "AES-256-GCM"
  }'

# Step 2: Upload encrypted media
curl -X PUT https://api.yourdomain.com/v1/media/upload-proxy/media-550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: image/jpeg" \
  -H "X-Encryption-Info: {\"algorithm\":\"AES-256-GCM\",\"key_id\":\"enc-key-uuid\",\"iv\":\"base64-iv\"}" \
  --data-binary "@encrypted_vacation.jpg"

# Step 3: Get download URL (for recipient)
curl -X GET https://api.yourdomain.com/v1/media/media-550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer $RECIPIENT_TOKEN"

# Step 4: Download media
curl -X GET https://api.yourdomain.com/v1/media/download-proxy/media-550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer $RECIPIENT_TOKEN" \
  --output downloaded_encrypted_vacation.jpg
```

---

## Related APIs

- **[Messaging API](API_MESSAGES.md)** - Attach media to messages
- **[User Management API](API_USERS.md)** - User authentication for media access
- **[WebSocket API](API_WEBSOCKET.md)** - Real-time media delivery notifications

---

## Media Best Practices

### Encryption Guidelines

- **Algorithm**: Use AES-256-GCM for media encryption
- **Key Management**: Store encryption keys securely client-side
- **Key Rotation**: Rotate media encryption keys periodically
- **Integrity**: Include HMAC for tamper detection

### Performance Tips

- **Chunked Uploads**: For large files, consider chunked uploads
- **Compression**: Compress media before encryption
- **Thumbnails**: Generate and upload thumbnails separately
- **Caching**: Cache downloaded media with proper expiration

### Security Recommendations

- **Content Scanning**: Scan media for malware client-side
- **Metadata Stripping**: Remove EXIF data from images
- **Size Validation**: Validate file sizes before upload
- **Type Verification**: Verify MIME types match content

---

*© 2025 SilentRelay. All rights reserved.*