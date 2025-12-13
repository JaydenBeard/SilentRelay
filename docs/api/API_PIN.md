# PIN Management API

The PIN Management API provides endpoints for setting, updating, and managing user PINs for additional security layers.

## Overview

**Base Path**: `/api/v1/pin`
**Authentication**: Requires valid JWT authentication
**Rate Limiting**: Strict limits to prevent brute force attacks

## PIN Security Features

- **Argon2id Hashing**: Industry-standard password hashing with OWASP-recommended parameters
- **PIN Length**: 4 or 6 digits only
- **Lockout Protection**: Progressive lockout after failed attempts
- **Secure Storage**: Hashed PINs stored in database (never plaintext)

## PIN Endpoints

### 1. Get PIN Status

**Endpoint**: `GET /api/v1/pin`
**Description**: Retrieves the current PIN status for the authenticated user.

**Headers**:
- `Authorization: Bearer <access_token>`

**Response**:
```json
{
  "has_pin": true,
  "pin_hash": "$argon2id$v=19$m=65536,t=3,p=4$...",
  "pin_length": 4
}
```

**Status Codes**:
- `200 OK`: PIN status retrieved successfully
- `401 Unauthorized`: Invalid or missing authentication token
- `429 Too Many Requests`: Rate limit exceeded

**Rate Limiting**: 10 requests per minute per user

---

### 2. Set PIN

**Endpoint**: `POST /api/v1/pin`
**Description**: Sets or updates the user's PIN.

**Headers**:
- `Authorization: Bearer <access_token>`
- `Content-Type: application/json`

**Request Body**:
```json
{
  "pin_hash": "$argon2id$v=19$m=65536,t=3,p=4$...",
  "pin_length": 4
}
```

**Response**:
```json
{
  "status": "saved"
}
```

**Status Codes**:
- `200 OK`: PIN saved successfully
- `400 Bad Request`: Invalid PIN length (must be 4 or 6)
- `401 Unauthorized`: Invalid or missing authentication token
- `429 Too Many Requests`: Rate limit exceeded

**Security Notes**:
- PIN must be hashed client-side using Argon2id before sending
- Server validates PIN length but does not validate hash format
- **Rate Limiting**: 5 requests per minute per user

---

### 3. Delete PIN

**Endpoint**: `DELETE /api/v1/pin`
**Description**: Removes the user's PIN.

**Headers**:
- `Authorization: Bearer <access_token>`

**Response**:
```json
{
  "status": "deleted"
}
```

**Status Codes**:
- `200 OK`: PIN deleted successfully
- `401 Unauthorized`: Invalid or missing authentication token
- `429 Too Many Requests`: Rate limit exceeded

**Rate Limiting**: 5 requests per minute per user

---

## PIN Security Implementation

### Hashing Parameters

The PIN uses Argon2id with OWASP-recommended parameters:

```javascript
// Client-side PIN hashing example
const hashPIN = async (pin) => {
  const encoder = new TextEncoder();
  const salt = crypto.getRandomValues(new Uint8Array(16));

  const key = await crypto.subtle.importKey(
    'raw',
    encoder.encode(pin),
    'PBKDF2',
    false,
    ['deriveBits']
  );

  // Note: In production, use WebAssembly Argon2 implementation
  // This is a simplified example
  const hash = await crypto.subtle.deriveBits(
    {
      name: 'PBKDF2',
      salt: salt,
      iterations: 100000,
      hash: 'SHA-256'
    },
    key,
    256
  );

  return btoa(String.fromCharCode(...new Uint8Array(hash)));
};
```

### Server-Side Validation

```go
// Server validates PIN format
func ValidatePIN(pin string) error {
  if len(pin) != 4 && len(pin) != 6 {
    return ErrPINTooShort
  }

  for _, c := range pin {
    if c < '0' || c > '9' {
      return ErrPINNotNumeric
    }
  }

  return nil
}
```

### Lockout Protection

PIN verification includes progressive lockout:

| Failed Attempts | Lockout Duration |
|-----------------|------------------|
| 3-4 | 5 minutes |
| 5+ | 1 hour |

---

## Examples

### Setting a PIN

```bash
# Hash PIN client-side first
PIN_HASH=$(echo -n "1234" | argon2 somesalt -id -t 3 -m 16 -p 4 -l 32 | grep -o '[$].*')

# Set PIN
curl -X POST https://api.yourdomain.com/v1/pin \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"pin_hash\": \"$PIN_HASH\",
    \"pin_length\": 4
  }"
```

### Checking PIN Status

```bash
curl -X GET https://api.yourdomain.com/v1/pin \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

### Removing PIN

```bash
curl -X DELETE https://api.yourdomain.com/v1/pin \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

---

## Related APIs

- **[Authentication API](API_AUTHENTICATION.md)** - User authentication and session management
- **[Device Management API](API_DEVICES.md)** - Device-specific security settings

---

## Security Considerations

### PIN Best Practices

- **Client-Side Hashing**: Always hash PINs client-side before transmission
- **Secure Storage**: Never store plaintext PINs
- **Rate Limiting**: Strict rate limits prevent brute force attacks
- **Lockout Protection**: Progressive lockout after failed attempts

### Implementation Notes

- PINs are optional but recommended for enhanced security
- PIN verification can be used for sensitive operations
- Failed attempts are tracked per user account
- Lockout duration increases with repeated failures

---

*© 2025 SilentRelay. All rights reserved.*