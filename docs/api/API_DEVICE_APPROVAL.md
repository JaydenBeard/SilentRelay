# Device Approval API

The Device Approval API handles the secure device linking workflow, requiring explicit approval from primary devices before new devices can access user accounts.

## Overview

**Base Path**: `/api/v1/device-approval`
**Authentication**: Mixed - some endpoints require auth, others are public
**Rate Limiting**: Strict limits to prevent abuse

## Device Approval Workflow

### Workflow Steps

1. **New Device Request**: Unauthenticated device requests approval
2. **Code Generation**: Server generates 6-digit approval code
3. **Primary Notification**: Primary device receives WebSocket notification
4. **Code Verification**: New device verifies code (proves possession)
5. **Approval Decision**: Primary device approves or denies request
6. **Device Activation**: Approved device can complete login

### Security Features

- **Time-limited Codes**: 15-minute expiration
- **Single-use Codes**: Each code works only once
- **Primary Device Required**: Only primary device can approve
- **WebSocket Notifications**: Real-time approval requests
- **IP Tracking**: Security monitoring of approval attempts

## Device Approval Endpoints

### 1. Request Device Approval

**Endpoint**: `POST /api/v1/device-approval/request`
**Description**: Initiates device linking request (public endpoint).

**Headers**:
- `Content-Type: application/json`

**Request Body**:
```json
{
  "phone_number": "+1234567890",
  "device_id": "550e8400-e29b-41d4-a716-446655440000",
  "device_name": "Work Laptop",
  "device_type": "web"
}
```

**Response Scenarios**:

**Requires Approval**:
```json
{
  "status": "pending",
  "requires_code": true,
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "expires_at": "2025-12-04T07:15:00Z",
  "primary_device_name": "John's iPhone"
}
```

**Already Linked**:
```json
{
  "status": "already_linked",
  "requires_code": false,
  "is_primary": true
}
```

**First Device**:
```json
{
  "status": "first_device",
  "requires_code": false,
  "will_be_primary": true
}
```

**Status Codes**:
- `200 OK`: Request processed
- `400 Bad Request`: Invalid request format
- `404 Not Found`: User not found
- `429 Too Many Requests`: Rate limit exceeded

**Rate Limiting**: 5 requests per minute per IP

---

### 2. Verify Approval Code

**Endpoint**: `POST /api/v1/device-approval/verify`
**Description**: Verifies the 6-digit code entered on new device.

**Headers**:
- `Content-Type: application/json`

**Request Body**:
```json
{
  "phone_number": "+1234567890",
  "code": "123456"
}
```

**Response**:
```json
{
  "status": "valid",
  "request_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Status Codes**:
- `200 OK`: Code verified successfully
- `400 Bad Request`: Invalid request format
- `401 Unauthorized`: Invalid or expired code
- `404 Not Found`: User not found
- `429 Too Many Requests`: Rate limit exceeded

**Rate Limiting**: 10 requests per minute per IP

---

### 3. Check Approval Status

**Endpoint**: `GET /api/v1/device-approval/{requestId}/status`
**Description**: Checks current status of approval request.

**Path Parameters**:
- `requestId`: UUID of the approval request

**Response**:
```json
{
  "status": "pending"
}
```

**Possible Status Values**:
- `pending`: Awaiting primary device approval
- `approved`: Request approved
- `denied`: Request denied
- `expired`: Request expired

**Status Codes**:
- `200 OK`: Status retrieved
- `400 Bad Request`: Invalid request ID
- `404 Not Found`: Request not found
- `429 Too Many Requests`: Rate limit exceeded

**Rate Limiting**: 20 requests per minute per IP

---

### 4. Get Pending Approvals

**Endpoint**: `GET /api/v1/device-approval/pending`
**Description**: Retrieves pending approval requests for authenticated user.

**Headers**:
- `Authorization: Bearer <access_token>`

**Response**:
```json
{
  "requests": [
    {
      "request_id": "550e8400-e29b-41d4-a716-446655440000",
      "device_name": "New Laptop",
      "device_type": "web",
      "requested_at": "2025-12-04T07:30:00Z",
      "expires_at": "2025-12-04T07:45:00Z",
      "ip_address": "192.168.1.102",
      "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
    }
  ]
}
```

**Status Codes**:
- `200 OK`: Requests retrieved
- `401 Unauthorized`: Invalid authentication
- `429 Too Many Requests`: Rate limit exceeded

**Rate Limiting**: 15 requests per minute per user

---

### 5. Approve Device Request

**Endpoint**: `POST /api/v1/device-approval/{requestId}/approve`
**Description**: Approves device linking request.

**Headers**:
- `Authorization: Bearer <access_token>`
- `X-Device-ID: <device_id>`

**Path Parameters**:
- `requestId`: UUID of the approval request

**Response**:
```json
{
  "status": "approved"
}
```

**Status Codes**:
- `200 OK`: Device approved
- `400 Bad Request`: Invalid request ID or missing device ID
- `401 Unauthorized`: Invalid authentication
- `403 Forbidden`: Not primary device
- `404 Not Found`: Request not found
- `429 Too Many Requests`: Rate limit exceeded

**Security Requirements**:
- Requester must be authenticated
- `X-Device-ID` header must match primary device
- Only primary device can approve requests

**Rate Limiting**: 10 requests per minute per user

---

### 6. Deny Device Request

**Endpoint**: `POST /api/v1/device-approval/{requestId}/deny`
**Description**: Denies device linking request.

**Headers**:
- `Authorization: Bearer <access_token>`

**Path Parameters**:
- `requestId`: UUID of the approval request

**Response**:
```json
{
  "status": "denied"
}
```

**Status Codes**:
- `200 OK`: Device denied
- `400 Bad Request`: Invalid request ID
- `401 Unauthorized`: Invalid authentication
- `403 Forbidden`: Not primary device
- `404 Not Found`: Request not found
- `429 Too Many Requests`: Rate limit exceeded

**Rate Limiting**: 10 requests per minute per user

---

## Security Considerations

### Code Security

- **Cryptographically Secure**: Codes generated with crypto/rand
- **Time-Limited**: 15-minute expiration window
- **Single-Use**: Codes invalidated after use
- **Rate Limited**: Strict limits prevent brute force

### Device Validation

- **Primary Device Required**: Only primary device can approve
- **Device ID Verification**: X-Device-ID header validation
- **User Ownership**: Device must belong to authenticated user

### Audit Trail

- **Request Logging**: All approval requests logged
- **IP Tracking**: Source IP addresses recorded
- **User Agent Logging**: Device information tracked
- **Approval Records**: Who approved/denied and when

---

## Examples

### Complete Approval Flow

```bash
# Step 1: New device requests approval
curl -X POST https://api.yourdomain.com/v1/device-approval/request \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number": "+1234567890",
    "device_id": "new-device-uuid",
    "device_name": "Work Laptop",
    "device_type": "web"
  }'

# Step 2: New device verifies code
curl -X POST https://api.yourdomain.com/v1/device-approval/verify \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number": "+1234567890",
    "code": "123456"
  }'

# Step 3: Primary device checks pending requests
curl -X GET https://api.yourdomain.com/v1/device-approval/pending \
  -H "Authorization: Bearer $PRIMARY_TOKEN"

# Step 4: Primary device approves
curl -X POST https://api.yourdomain.com/v1/device-approval/$REQUEST_ID/approve \
  -H "Authorization: Bearer $PRIMARY_TOKEN" \
  -H "X-Device-ID: $PRIMARY_DEVICE_ID"

# Step 5: New device can now login
curl -X POST https://api.yourdomain.com/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number": "+1234567890",
    "device_id": "new-device-uuid",
    "device_name": "Work Laptop",
    "device_type": "web",
    "public_device_key": "base64_device_key"
  }'
```

### Check Status

```bash
curl -X GET https://api.yourdomain.com/v1/device-approval/$REQUEST_ID/status
```

### Deny Request

```bash
curl -X POST https://api.yourdomain.com/v1/device-approval/$REQUEST_ID/deny \
  -H "Authorization: Bearer $PRIMARY_TOKEN"
```

---

## WebSocket Integration

### Approval Notifications

Primary devices receive real-time notifications via WebSocket:

```json
{
  "type": "device_approval_request",
  "payload": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "device_name": "New Laptop",
    "device_type": "web",
    "code": "123456",
    "expires_at": "2025-12-04T07:15:00Z"
  }
}
```

### Approval Results

New devices receive approval status:

```json
{
  "type": "device_approved",
  "payload": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "message": "Device approved. You can now complete login."
  }
}
```

---

## Related APIs

- **[Authentication API](API_AUTHENTICATION.md)** - Device login after approval
- **[Device Management API](API_DEVICES.md)** - Device management after linking
- **[WebSocket API](API_WEBSOCKET.md)** - Real-time approval notifications

---

## Security Best Practices

### For Primary Devices

- **Verify Device Details**: Check device name, type, and location
- **Trust Decisions**: Only approve devices you recognize
- **Regular Review**: Periodically review linked devices
- **Immediate Denial**: Deny suspicious approval requests

### For New Devices

- **Secure Code Entry**: Enter codes in private
- **Code Expiration**: Use codes before they expire
- **Device Security**: Ensure device is secure before linking

### For Developers

- **Rate Limiting**: Implement client-side rate limiting
- **Error Handling**: Handle approval failures gracefully
- **User Education**: Explain approval process to users
- **Audit Logging**: Log all approval activities

---

*© 2025 SilentRelay. All rights reserved.*