# SilentRelay API Documentation

> **Base URL:** `https://silentrelay.com.au/api/v1`  
> **WebSocket:** `wss://silentrelay.com.au/ws`

## Overview

SilentRelay is an end-to-end encrypted messaging platform. All message content is encrypted client-side using the Signal Protocol before being transmitted.

### Authentication

All protected endpoints require a JWT token in the `Authorization` header:

```
Authorization: Bearer <jwt_token>
```

Tokens are obtained via the authentication flow and expire after 24 hours. Use the refresh token to obtain new access tokens.

---

## Endpoints

### Health & Monitoring

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| GET | `/health` | No | Health check for load balancer |
| GET | `/metrics` | No | Prometheus metrics |

---

## Authentication

### Request Verification Code

Send an SMS verification code to a phone number.

```http
POST /api/v1/auth/request-code
```

**Request Body:**

```json
{
  "phone": "+61412345678"
}
```

**Response (200 OK):**

```json
{
  "success": true,
  "message": "Verification code sent"
}
```

**Rate Limit:** 3 requests per hour per phone number

---

### Verify Code

Validate the SMS verification code.

```http
POST /api/v1/auth/verify
```

**Request Body:**

```json
{
  "phone": "+61412345678",
  "code": "123456"
}
```

**Response (200 OK) - New User:**

```json
{
  "valid": true,
  "user_exists": false
}
```

**Response (200 OK) - Existing User:**

```json
{
  "valid": true,
  "user_exists": true,
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "user": {
    "id": "uuid",
    "phone": "+61412345678",
    "username": "johndoe",
    "display_name": "John Doe"
  }
}
```

---

### Register

Create a new user account after phone verification.

```http
POST /api/v1/auth/register
```

**Request Body:**

```json
{
  "phone": "+61412345678",
  "code": "123456",
  "username": "johndoe",
  "display_name": "John Doe",
  "device_id": "device-uuid",
  "identity_key": "base64-encoded-key",
  "signed_prekey": "base64-encoded-key",
  "signed_prekey_signature": "base64-signature",
  "prekeys": ["base64-key-1", "base64-key-2", "..."]
}
```

**Response (201 Created):**

```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "user": {
    "id": "uuid",
    "phone": "+61412345678",
    "username": "johndoe",
    "display_name": "John Doe"
  }
}
```

---

### Login (Existing User)

Login with existing account credentials.

```http
POST /api/v1/auth/login
```

**Request Body:**

```json
{
  "access_token": "eyJhbGc...",
  "device_id": "new-device-uuid",
  "identity_key": "base64-encoded-key",
  "signed_prekey": "base64-encoded-key",
  "prekeys": ["..."]
}
```

---

### Refresh Token

Obtain new access token using refresh token.

```http
POST /api/v1/auth/refresh
```

**Request Body:**

```json
{
  "refresh_token": "eyJhbGc..."
}
```

**Response (200 OK):**

```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc..."
}
```

---

## User Management

### Get Current User

```http
GET /api/v1/users/me
Authorization: Bearer <token>
```

**Response (200 OK):**

```json
{
  "id": "uuid",
  "phone": "+61412345678",
  "username": "johndoe",
  "display_name": "John Doe",
  "avatar_url": "https://...",
  "created_at": "2024-01-15T10:30:00Z"
}
```

---

### Update User

```http
PUT /api/v1/users/me
Authorization: Bearer <token>
```

**Request Body:**

```json
{
  "display_name": "John D.",
  "avatar_url": "https://..."
}
```

---

### Delete User

Permanently delete user account.

```http
DELETE /api/v1/users/me
Authorization: Bearer <token>
```

---

### Search Users

Search for users by username or phone.

```http
GET /api/v1/users/search?q=john
Authorization: Bearer <token>
```

**Response (200 OK):**

```json
{
  "users": [
    {
      "id": "uuid",
      "username": "johndoe",
      "display_name": "John Doe",
      "avatar_url": "https://..."
    }
  ]
}
```

**Rate Limit:** 10 requests per minute

---

### Get User Profile

```http
GET /api/v1/users/{userId}/profile
Authorization: Bearer <token>
```

---

### Check Username Availability

```http
GET /api/v1/users/check-username/{username}
Authorization: Bearer <token>
```

**Response (200 OK):**

```json
{
  "available": true
}
```

---

### Get User Keys (for E2E Encryption)

Get a user's public keys for establishing encrypted session.

```http
GET /api/v1/users/{userId}/keys
Authorization: Bearer <token>
```

**Response (200 OK):**

```json
{
  "identity_key": "base64-encoded-key",
  "signed_prekey": "base64-encoded-key",
  "signed_prekey_signature": "base64-signature",
  "one_time_prekey": "base64-encoded-key"
}
```

---

## Device Management

### Get Devices

List all registered devices for current user.

```http
GET /api/v1/devices
Authorization: Bearer <token>
```

**Response (200 OK):**

```json
{
  "devices": [
    {
      "id": "device-uuid",
      "name": "iPhone 15",
      "platform": "ios",
      "is_primary": true,
      "last_seen": "2024-01-15T10:30:00Z"
    }
  ]
}
```

---

### Remove Device

```http
DELETE /api/v1/devices/{deviceId}
Authorization: Bearer <token>
```

---

### Set Primary Device

```http
PUT /api/v1/devices/{deviceId}/primary
Authorization: Bearer <token>
```

---

## Device Approval (Multi-Device)

Secure flow for linking new devices to existing account.

### Request Device Approval

Called from new device requesting to link.

```http
POST /api/v1/device-approval/request
```

**Request Body:**

```json
{
  "phone": "+61412345678",
  "device_id": "new-device-uuid",
  "device_name": "New iPhone"
}
```

**Response (200 OK):**

```json
{
  "request_id": "uuid",
  "code": "ABC123",
  "expires_at": "2024-01-15T10:35:00Z"
}
```

---

### Verify Approval Code

```http
POST /api/v1/device-approval/verify
```

**Request Body:**

```json
{
  "request_id": "uuid",
  "code": "ABC123"
}
```

---

### Check Approval Status

```http
GET /api/v1/device-approval/{requestId}/status
```

**Response (200 OK):**

```json
{
  "status": "pending|approved|denied|expired"
}
```

---

### Approve Device (from primary device)

```http
POST /api/v1/device-approval/{requestId}/approve
Authorization: Bearer <token>
```

---

### Deny Device (from primary device)

```http
POST /api/v1/device-approval/{requestId}/deny
Authorization: Bearer <token>
```

---

## Privacy & Blocking

### Get Privacy Settings

```http
GET /api/v1/privacy
Authorization: Bearer <token>
```

**Response (200 OK):**

```json
{
  "last_seen": "everyone|contacts|nobody",
  "read_receipts": true,
  "typing_indicators": true
}
```

---

### Update Privacy Setting

```http
POST /api/v1/privacy
Authorization: Bearer <token>
```

**Request Body:**

```json
{
  "setting": "last_seen",
  "value": "contacts"
}
```

---

### Block User

```http
POST /api/v1/users/block
Authorization: Bearer <token>
```

**Request Body:**

```json
{
  "user_id": "uuid-to-block"
}
```

---

### Unblock User

```http
POST /api/v1/users/unblock
Authorization: Bearer <token>
```

**Request Body:**

```json
{
  "user_id": "uuid-to-unblock"
}
```

---

### Get Blocked Users

```http
GET /api/v1/users/blocked
Authorization: Bearer <token>
```

---

## Messages

### Get Message History

```http
GET /api/v1/messages?conversation_id=uuid&limit=50&before=timestamp
Authorization: Bearer <token>
```

---

### Update Message Status

```http
PUT /api/v1/messages/{messageId}/status
Authorization: Bearer <token>
```

**Request Body:**

```json
{
  "status": "delivered|read"
}
```

---

## Groups

### Create Group

```http
POST /api/v1/groups
Authorization: Bearer <token>
```

**Request Body:**

```json
{
  "name": "Family Chat",
  "member_ids": ["uuid1", "uuid2"]
}
```

---

### Get Group

```http
GET /api/v1/groups/{groupId}
Authorization: Bearer <token>
```

---

### Add Group Member

```http
POST /api/v1/groups/{groupId}/members
Authorization: Bearer <token>
```

**Request Body:**

```json
{
  "user_id": "uuid"
}
```

---

### Remove Group Member

```http
DELETE /api/v1/groups/{groupId}/members/{userId}
Authorization: Bearer <token>
```

---

## Media

### Get Upload URL (Presigned)

Get a presigned URL to upload media to object storage.

```http
POST /api/v1/media/upload-url
Authorization: Bearer <token>
```

**Request Body:**

```json
{
  "filename": "image.jpg",
  "content_type": "image/jpeg",
  "size": 1024000
}
```

**Response (200 OK):**

```json
{
  "upload_url": "https://minio.../presigned-url",
  "media_id": "uuid",
  "expires_at": "2024-01-15T10:35:00Z"
}
```

---

### Get Media Download URL

```http
GET /api/v1/media/download-url/{mediaId}
Authorization: Bearer <token>
```

**Response (200 OK):**

```json
{
  "download_url": "https://minio.../presigned-url",
  "expires_at": "2024-01-15T11:30:00Z"
}
```

---

### Upload Proxy

Direct upload through the server (for HTTPS compliance).

```http
PUT /api/v1/media/upload-proxy/{mediaId}
Authorization: Bearer <token>
Content-Type: <media-type>
```

---

### Download Proxy

```http
GET /api/v1/media/download-proxy/{mediaId}
Authorization: Bearer <token>
```

---

## WebRTC

### Get TURN Credentials

Get time-limited TURN server credentials for peer-to-peer calls.

```http
GET /api/v1/rtc/turn-credentials
Authorization: Bearer <token>
```

**Response (200 OK):**

```json
{
  "urls": ["turn:turn.silentrelay.com.au:3478"],
  "username": "timestamp:username",
  "credential": "hmac-signature",
  "ttl": 86400
}
```

---

## PIN Management

Server-synced PIN for client-side encryption.

### Get PIN

```http
GET /api/v1/pin
Authorization: Bearer <token>
```

**Response (200 OK):**

```json
{
  "pin_hash": "hashed-pin",
  "salt": "random-salt"
}
```

---

### Set PIN

```http
POST /api/v1/pin
Authorization: Bearer <token>
```

**Request Body:**

```json
{
  "pin_hash": "hashed-pin",
  "salt": "random-salt"
}
```

---

### Delete PIN

```http
DELETE /api/v1/pin
Authorization: Bearer <token>
```

---

## WebSocket Protocol

Connect to the WebSocket for real-time messaging:

```
wss://silentrelay.com.au/ws?token=<jwt_token>
```

Or via Sec-WebSocket-Protocol header:

```
Sec-WebSocket-Protocol: Bearer, <jwt_token>
```

### Message Types

| Type | Direction | Description |
|------|-----------|-------------|
| `send` | Client → Server | Send encrypted message |
| `deliver` | Server → Client | Receive encrypted message |
| `read_receipt` | Bidirectional | Mark message as read |
| `typing` | Bidirectional | Typing indicator |
| `heartbeat` | Bidirectional | Keep connection alive |
| `status_update` | Server → Client | Message status change |
| `presence` | Server → Client | User online/offline |
| `sync_request` | Client → Server | Request data sync |
| `sync_data` | Server → Client | Sync response |

### Message Format

```json
{
  "type": "send",
  "messageId": "uuid",
  "timestamp": 1705312200000,
  "payload": {
    "recipientId": "user-uuid",
    "ciphertext": "base64-encrypted-content",
    "messageType": "whisper"
  },
  "signature": "hmac-signature",
  "nonce": [1, 2, 3, ...]
}
```

### HMAC Signing

All WebSocket messages must be signed with HMAC-SHA256 using the JWT token as the key:

```javascript
const signature = HMAC_SHA256(
  JSON.stringify({ type, timestamp, messageId, payload }),
  token.slice(0, 32)
);
```

---

## Error Responses

All errors follow this format:

```json
{
  "error": "Error message",
  "code": "ERROR_CODE"
}
```

### Common Error Codes

| HTTP Code | Error Code | Description |
|-----------|------------|-------------|
| 400 | `INVALID_REQUEST` | Malformed request body |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 403 | `FORBIDDEN` | Not permitted for this resource |
| 404 | `NOT_FOUND` | Resource not found |
| 409 | `CONFLICT` | Resource already exists |
| 429 | `RATE_LIMITED` | Too many requests |
| 500 | `INTERNAL_ERROR` | Server error |

---

## Rate Limits

| Endpoint Category | Limit |
|-------------------|-------|
| Auth (SMS) | 3/hour |
| Auth (verify/login) | 10/minute |
| Search | 10/minute |
| Media upload | 20/hour |
| General API | 100/minute |

---

## Security Notes

1. **End-to-End Encryption**: All message content is encrypted client-side. The server never sees plaintext messages.

2. **Zero-Knowledge**: Conversation metadata is stored client-side only. Server doesn't know who talks to whom.

3. **Perfect Forward Secrecy**: Uses Signal Protocol with rotating prekeys.

4. **HMAC Message Signing**: All WebSocket messages are cryptographically signed.

5. **TLS 1.3**: All connections use modern TLS encryption.
