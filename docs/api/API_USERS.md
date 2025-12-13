# User Management API

The User Management API provides endpoints for managing user profiles, retrieving cryptographic keys, and handling user-related operations.

## Overview

**Base Path**: `/api/v1/users`
**Authentication**: Requires valid JWT authentication
**Rate Limiting**: Normal rate limits apply

## Endpoints

### 1. Get Current User

**Endpoint**: `GET /api/v1/users/me`
**Description**: Retrieves the authenticated user's profile information.

**Headers**:
- `Authorization: Bearer <access_token>`

**Response**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "phone_number": "+1234567890",
  "username": "john_doe",
  "display_name": "John Doe",
  "avatar_url": "https://example.com/avatar.jpg",
  "created_at": "2025-01-01T00:00:00Z",
  "updated_at": "2025-01-01T00:00:00Z",
  "last_seen": "2025-12-04T07:00:00Z",
  "privacy_settings": {
    "show_last_seen": true,
    "show_read_receipts": true,
    "allow_chat_requests": true
  }
}
```

**Status Codes**:
- `200 OK`: User profile retrieved successfully
- `401 Unauthorized`: Invalid or missing authentication token
- `404 Not Found`: User not found
- `429 Too Many Requests`: Rate limit exceeded

**Rate Limiting**: 60 requests per minute per user

---

### 2. Update User Profile

**Endpoint**: `PUT|PATCH /api/v1/users/me`
**Description**: Updates the authenticated user's profile information.

**Headers**:
- `Authorization: Bearer <access_token>`
- `Content-Type: application/json`

**Request Body**:
```json
{
  "display_name": "John Doe Updated",
  "username": "john_doe_updated",
  "avatar_url": "https://example.com/new_avatar.jpg",
  "bio": "Software Developer",
  "privacy_settings": {
    "show_last_seen": false,
    "show_read_receipts": true
  }
}
```

**Response**:
```json
{
  "status": "updated"
}
```

**Status Codes**:
- `200 OK`: Profile updated successfully
- `400 Bad Request`: Invalid request format or data
- `401 Unauthorized`: Invalid or missing authentication token
- `404 Not Found`: User not found
- `409 Conflict`: Username already taken
- `429 Too Many Requests`: Rate limit exceeded

**Rate Limiting**: 30 requests per minute per user

**Field Validation**:
- `username`: 3-30 characters, alphanumeric + underscore
- `display_name`: 1-50 characters
- `avatar_url`: Must be valid HTTPS URL
- `bio`: 0-500 characters

---

### 3. Delete User Account

**Endpoint**: `DELETE /api/v1/users/me`
**Description**: Permanently deletes the authenticated user's account and all associated data.

**Headers**:
- `Authorization: Bearer <access_token>`

**Response**:
```json
{
  "status": "deleted"
}
```

**Status Codes**:
- `200 OK`: Account deleted successfully
- `401 Unauthorized`: Invalid or missing authentication token
- `404 Not Found`: User not found
- `429 Too Many Requests`: Rate limit exceeded

**Important Notes**:
- **Irreversible**: This action cannot be undone
- **Data Removal**: All messages, contacts, and media are permanently deleted
- **Compliance**: Satisfies GDPR "Right to Deletion" requirements
- **Rate Limiting**: 5 requests per hour per user to prevent accidental deletion

---

### 4. Upload Pre-Keys

**Endpoint**: `POST /api/v1/users/me/prekeys`
**Description**: Uploads a batch of one-time pre-keys for end-to-end encryption session establishment.

**Headers**:
- `Authorization: Bearer <access_token>`
- `Content-Type: application/json`

**Request Body**:
```json
{
  "prekeys": [
    {
      "prekey_id": 1001,
      "public_key": "base64_encoded_ec_public_key"
    },
    {
      "prekey_id": 1002,
      "public_key": "base64_encoded_ec_public_key"
    }
  ]
}
```

**Response**:
```json
{
  "status": "uploaded"
}
```

**Status Codes**:
- `200 OK`: Pre-keys uploaded successfully
- `400 Bad Request`: Invalid pre-key format or missing fields
- `401 Unauthorized`: Invalid or missing authentication token
- `413 Payload Too Large`: Too many pre-keys (max 100)
- `429 Too Many Requests`: Rate limit exceeded

**Security Notes**:
- Pre-keys are used for initial X3DH handshake
- Each pre-key can only be used once
- Automatically rotates old pre-keys
- **Rate Limiting**: 10 requests per minute per user

---

### 5. Get User Public Keys

**Endpoint**: `GET /api/v1/users/{userId}/keys`
**Description**: Retrieves a user's public cryptographic keys for establishing end-to-end encrypted sessions.

**Headers**:
- `Authorization: Bearer <access_token>`

**Path Parameters**:
- `userId`: UUID of the target user

**Response**:
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "identity_key": "base64_encoded_identity_public_key",
  "signed_prekey": "base64_encoded_signed_prekey",
  "signed_prekey_signature": "base64_encoded_signature",
  "prekeys": [
    {
      "prekey_id": 1001,
      "public_key": "base64_encoded_ec_public_key"
    }
  ],
  "last_key_update": "2025-12-04T07:00:00Z",
  "key_expiration": "2025-12-11T07:00:00Z"
}
```

**Status Codes**:
- `200 OK`: Keys retrieved successfully
- `400 Bad Request`: Invalid user ID format
- `401 Unauthorized`: Invalid or missing authentication token
- `403 Forbidden`: Not authorized to access this user's keys
- `404 Not Found`: User not found
- `429 Too Many Requests`: Rate limit exceeded

**Security Notes**:
- **Key Transparency**: All key changes are logged in the transparency log
- **Key Rotation**: Signed pre-keys rotate automatically
- **Access Control**: Only authenticated users can retrieve keys
- **Rate Limiting**: 120 requests per minute per user (allows for multi-device sync)

---

### 6. Check Username Availability

**Endpoint**: `GET /api/v1/users/check-username/{username}`
**Description**: Checks if a username is available for registration.

**Headers**:
- `Authorization: Bearer <access_token>`

**Path Parameters**:
- `username`: Desired username to check

**Response (Available)**:
```json
{
  "available": true
}
```

**Response (Taken)**:
```json
{
  "available": false,
  "message": "Username is already taken"
}
```

**Response (Invalid)**:
```json
{
  "available": false,
  "message": "Username must be at least 3 characters"
}
```

**Status Codes**:
- `200 OK`: Availability check completed
- `400 Bad Request`: Invalid username format
- `401 Unauthorized`: Invalid or missing authentication token
- `429 Too Many Requests`: Rate limit exceeded

**Username Requirements**:
- Minimum 3 characters, maximum 30 characters
- Alphanumeric characters and underscores only
- No consecutive underscores
- Cannot start or end with underscore

**Rate Limiting**: 20 requests per minute per user

---

## User Search

### Search Users

**Endpoint**: `GET /api/v1/users/search?q={query}`
**Description**: Searches for users by username or display name.

**Headers**:
- `Authorization: Bearer <access_token>`

**Query Parameters**:
- `q`: Search query (minimum 3 characters)
- `limit`: Results limit (default: 20, max: 20)

**Response**:
```json
{
  "results": [
    {
      "user_id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "john_doe",
      "display_name": "John Doe",
      "avatar_url": "https://example.com/avatar.jpg",
      "has_common_contacts": true
    }
  ],
  "total": 1,
  "limit": 20,
  "offset": 0
}
```

**Status Codes**:
- `200 OK`: Search completed successfully
- `400 Bad Request`: Invalid query parameters
- `401 Unauthorized`: Invalid or missing authentication token
- `429 Too Many Requests`: Rate limit exceeded

**Security Notes**:
- **Privacy Protection**: Only returns users who allow discovery
- **Anti-Enumeration**: Rate limited and requires minimum query length
- **Result Limiting**: Maximum 20 results to prevent scraping
- **Rate Limiting**: 10 requests per minute per user (strict to prevent abuse)

---

## Security Considerations

### Data Privacy

- **Minimal Exposure**: Only essential profile data is returned
- **Privacy Settings**: Respects user privacy preferences
- **No Email/Phone Exposure**: Phone numbers never returned in search results

### Rate Limiting Strategy

- **Profile Updates**: 30/minute - allows frequent updates but prevents spam
- **Key Retrieval**: 120/minute - supports multi-device synchronization
- **Search**: 10/minute - prevents user enumeration attacks
- **Deletion**: 5/hour - prevents accidental account loss

### End-to-End Encryption Context

- **Key Management**: This API provides keys needed for Signal Protocol
- **Session Establishment**: Keys retrieved here enable X3DH handshake
- **Forward Secrecy**: Regular key rotation maintained server-side

---

## Examples

### Complete User Profile Management

```bash
# Get current user profile
curl -X GET https://api.yourdomain.com/v1/users/me \
  -H "Authorization: Bearer $ACCESS_TOKEN"

# Update profile
curl -X PUT https://api.yourdomain.com/v1/users/me \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "display_name": "John Doe Updated",
    "username": "john_doe_updated",
    "bio": "Senior Software Engineer"
  }'

# Upload pre-keys for E2EE
curl -X POST https://api.yourdomain.com/v1/users/me/prekeys \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "prekeys": [
      {"prekey_id": 1001, "public_key": "base64_key_1"},
      {"prekey_id": 1002, "public_key": "base64_key_2"}
    ]
  }'

# Get another user's keys for session establishment
curl -X GET https://api.yourdomain.com/v1/users/550e8400-e29b-41d4-a716-446655440000/keys \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

---

## Related APIs

- **[Authentication API](API_AUTHENTICATION.md)** - User registration and login
- **[Device Management API](API_DEVICES.md)** - Manage user devices
- **[WebSocket API](API_WEBSOCKET.md)** - Real-time messaging using retrieved keys

---

*© 2025 SilentRelay. All rights reserved.*