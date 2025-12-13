# Authentication API

The Authentication API provides endpoints for user registration, login, token management, and secure access to the SilentRelay platform.

## Overview

**Base Path**: `/api/v1/auth`
**Authentication**: Public endpoints (no auth required)
**Rate Limiting**: Strict rate limits apply to prevent abuse

## Endpoints

### 1. Request Verification Code

**Endpoint**: `POST /api/v1/auth/request-code`
**Description**: Initiates user registration or login by sending a verification code to the user's phone number.

**Request Body**:
```json
{
  "phone_number": "+1234567890"
}
```

**Response (Development Mode)**:
```json
{
  "message": "Verification code sent (DEV MODE)",
  "code": "123456"
}
```

**Response (Production Mode)**:
```json
{
  "message": "Verification code sent"
}
```

**Status Codes**:
- `200 OK`: Code sent successfully
- `400 Bad Request`: Invalid phone number format
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Failed to send code

**Rate Limiting**: 5 requests per minute per IP address

---

### 2. Verify Code

**Endpoint**: `POST /api/v1/auth/verify`
**Description**: Validates the verification code and returns authentication tokens for existing users.

**Request Body**:
```json
{
  "phone_number": "+1234567890",
  "code": "123456"
}
```

**Response (Existing User)**:
```json
{
  "verified": true,
  "user_exists": true,
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "device_id": "550e8400-e29b-41d4-a716-446655440000",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "phone_number": "+1234567890",
    "username": "john_doe",
    "display_name": "John Doe",
    "avatar_url": "https://example.com/avatar.jpg"
  }
}
```

**Response (New User)**:
```json
{
  "verified": true,
  "user_exists": false,
  "user_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Status Codes**:
- `200 OK`: Code verified successfully
- `400 Bad Request`: Invalid request format
- `401 Unauthorized`: Invalid or expired code
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Verification failed

**Rate Limiting**: 10 requests per minute per IP address

---

### 3. Register User

**Endpoint**: `POST /api/v1/auth/register`
**Description**: Creates a new user account with the provided verification code and cryptographic keys.

**Request Body**:
```json
{
  "phone_number": "+1234567890",
  "code": "123456",
  "public_identity_key": "base64_encoded_public_key",
  "public_signed_prekey": "base64_encoded_signed_prekey",
  "signed_prekey_signature": "base64_encoded_signature",
  "display_name": "John Doe",
  "device_id": "550e8400-e29b-41d4-a716-446655440000",
  "device_name": "Primary Device",
  "device_type": "web"
}
```

**Response**:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2025-12-04T08:18:00Z",
  "user": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "phone_number": "+1234567890",
    "display_name": "John Doe"
  }
}
```

**Status Codes**:
- `200 OK`: User registered successfully
- `400 Bad Request`: Missing required fields or invalid format
- `401 Unauthorized`: Invalid verification code
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Registration failed

**Security Notes**:
- The verification code is re-validated to prevent TOCTOU vulnerabilities
- User creation and code marking as verified are atomic operations
- Device registration happens automatically for the primary device

---

### 4. Login

**Endpoint**: `POST /api/v1/auth/login`
**Description**: Authenticates an existing user on a new device and returns access tokens.

**Request Body**:
```json
{
  "phone_number": "+1234567890",
  "device_id": "550e8400-e29b-41d4-a716-446655440000",
  "device_name": "Web Browser",
  "device_type": "web",
  "public_device_key": "base64_encoded_device_key"
}
```

**Response**:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2025-12-04T08:18:00Z",
  "user": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "phone_number": "+1234567890",
    "display_name": "John Doe"
  },
  "has_pin": true
}
```

**Status Codes**:
- `200 OK`: Login successful
- `400 Bad Request`: Invalid request format
- `404 Not Found`: User not found
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Login failed

**Device Management**:
- Automatically registers the device
- Sets as primary device if no other devices exist
- Returns PIN status (has_pin) for client-side PIN verification

---

### 5. Refresh Token

**Endpoint**: `POST /api/v1/auth/refresh`
**Description**: Generates a new access token using a valid refresh token.

**Request Body**:
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response**:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2025-12-04T08:18:00Z"
}
```

**Status Codes**:
- `200 OK`: Token refreshed successfully
- `400 Bad Request`: Invalid request format
- `401 Unauthorized`: Invalid refresh token
- `429 Too Many Requests`: Rate limit exceeded

**Security Notes**:
- Refresh tokens have longer expiration than access tokens
- Invalidates old access token upon successful refresh
- Rate limited to prevent token exhaustion attacks

---

## Security Considerations

### Authentication Flow

1. **New User**: `request-code` → `verify` → `register` → `login`
2. **Existing User**: `request-code` → `verify` (returns tokens) or `login` (with device info)

### Token Management

- **Access Token**: Short-lived (1 hour), used for API requests
- **Refresh Token**: Long-lived (30 days), used to obtain new access tokens
- **Device Binding**: Tokens are tied to specific devices

### Rate Limiting

- **SMS Endpoints**: 5 requests/minute/IP to prevent SMS spam
- **Auth Endpoints**: 10 requests/minute/IP to prevent brute force
- **Global Limits**: 1000 requests/minute across all endpoints

---

## Examples

### Complete Registration Flow

```bash
# Step 1: Request verification code
curl -X POST https://api.yourdomain.com/v1/auth/request-code \
  -H "Content-Type: application/json" \
  -d '{"phone_number": "+1234567890"}'

# Step 2: Verify code (new user)
curl -X POST https://api.yourdomain.com/v1/auth/verify \
  -H "Content-Type: application/json" \
  -d '{"phone_number": "+1234567890", "code": "123456"}'

# Step 3: Register with cryptographic keys
curl -X POST https://api.yourdomain.com/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number": "+1234567890",
    "code": "123456",
    "public_identity_key": "base64_identity_key",
    "public_signed_prekey": "base64_signed_prekey",
    "signed_prekey_signature": "base64_signature",
    "display_name": "John Doe",
    "device_id": "550e8400-e29b-41d4-a716-446655440000",
    "device_name": "Primary Device",
    "device_type": "web"
  }'
```

### Existing User Login

```bash
curl -X POST https://api.yourdomain.com/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number": "+1234567890",
    "device_id": "new-device-id",
    "device_name": "Secondary Browser",
    "device_type": "web",
    "public_device_key": "base64_device_key"
  }'
```

---

## Related APIs

- **[User Management API](API_USERS.md)** - Manage user profiles and retrieve cryptographic keys
- **[Device Management API](API_DEVICES.md)** - Manage linked devices
- **[WebSocket API](API_WEBSOCKET.md)** - Real-time messaging after authentication

---

*© 2025 SilentRelay. All rights reserved.*