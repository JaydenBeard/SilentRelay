# Device Management API

This API provides endpoints for managing user devices in the SilentRelay messaging system. Device management is crucial for end-to-end encryption as it ensures proper key distribution and session management across multiple devices.

## Authentication

All device management endpoints require authentication via JWT token in the Authorization header:

```
Authorization: Bearer <jwt_token>
```

## Endpoints

### List User Devices

Get all devices associated with the authenticated user.

**GET** `/api/v1/devices`

**Response:**
```json
{
  "devices": [
    {
      "device_id": "uuid",
      "device_name": "iPhone 13",
      "device_type": "mobile",
      "is_primary": true,
      "registered_at": "2024-01-01T00:00:00Z",
      "last_seen": "2024-01-01T12:00:00Z",
      "is_active": true
    }
  ]
}
```

### Remove Device

Remove a device from the user's account. This will revoke all sessions for that device and trigger key rotation.

**DELETE** `/api/v1/devices/{deviceId}`

**Parameters:**
- `deviceId` (path): UUID of the device to remove

**Response:** `204 No Content`

**Security Notes:**
- Users cannot remove their primary device
- Removing a device triggers automatic key rotation for security
- All active sessions for the device are immediately revoked

### Set Primary Device

Change which device is considered the primary device for key provisioning.

**PUT** `/api/v1/devices/{deviceId}/primary`

**Parameters:**
- `deviceId` (path): UUID of the device to set as primary

**Response:** `200 OK`

**Security Notes:**
- Only one device can be primary at a time
- Primary device receives all new key material first
- Changing primary device triggers key synchronization

## Device Approval Flow

### Request Device Approval

When a new device attempts to login, it must be approved by an existing device.

**POST** `/api/v1/device-approval/request`

**Request Body:**
```json
{
  "device_name": "New iPad",
  "device_type": "tablet"
}
```

**Response:**
```json
{
  "request_id": "uuid",
  "approval_code": "123456",
  "expires_at": "2024-01-01T00:05:00Z"
}
```

### Verify Approval Code

Check if an approval code is valid and not expired.

**POST** `/api/v1/device-approval/verify`

**Request Body:**
```json
{
  "request_id": "uuid",
  "approval_code": "123456"
}
```

**Response:** `200 OK` or `400 Bad Request`

### Get Pending Approvals

Get all pending device approval requests for the user.

**GET** `/api/v1/device-approval/pending`

**Response:**
```json
{
  "requests": [
    {
      "request_id": "uuid",
      "device_name": "New iPad",
      "device_type": "tablet",
      "created_at": "2024-01-01T00:00:00Z",
      "expires_at": "2024-01-01T00:05:00Z"
    }
  ]
}
```

### Check Approval Status

Check the status of a specific approval request.

**GET** `/api/v1/device-approval/{requestId}/status`

**Response:**
```json
{
  "status": "pending|approved|denied|expired",
  "approved_by_device": "uuid",
  "responded_at": "2024-01-01T00:01:00Z"
}
```

### Approve Device

Approve a pending device approval request.

**POST** `/api/v1/device-approval/{requestId}/approve`

**Response:** `200 OK`

### Deny Device

Deny a pending device approval request.

**POST** `/api/v1/device-approval/{requestId}/deny`

**Response:** `200 OK`

## Error Responses

All endpoints may return the following error responses:

**401 Unauthorized:**
```json
{
  "error": "authentication_required",
  "message": "Valid authentication token required"
}
```

**403 Forbidden:**
```json
{
  "error": "insufficient_permissions",
  "message": "You do not have permission to perform this action"
}
```

**404 Not Found:**
```json
{
  "error": "device_not_found",
  "message": "The specified device was not found"
}
```

**429 Too Many Requests:**
```json
{
  "error": "rate_limit_exceeded",
  "message": "Too many requests. Please try again later."
}
```

## Security Considerations

1. **Device Removal**: Always triggers key rotation to prevent access via stolen keys
2. **Primary Device**: Only primary devices can approve new devices
3. **Approval Timeout**: Approval codes expire after 5 minutes for security
4. **Rate Limiting**: Device management operations are rate limited
5. **Audit Logging**: All device operations are logged for security monitoring

## Rate Limits

- Device listing: 30 requests per minute
- Device removal: 5 requests per minute
- Device approval operations: 10 requests per minute
- Primary device changes: 3 requests per hour