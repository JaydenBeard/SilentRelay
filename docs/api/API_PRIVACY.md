# Privacy Settings API

The Privacy Settings API allows users to control their privacy preferences, including read receipts, last seen status, and group membership controls.

## Overview

**Base Path**: `/api/v1/privacy`
**Authentication**: Requires valid JWT authentication
**Rate Limiting**: Normal limits with strict limits on updates

## Privacy Settings

The API manages the following privacy controls:

| Setting | Type | Options | Default | Description |
|---------|------|---------|---------|-------------|
| `show_read_receipts` | boolean | `true`/`false` | `true` | Show when messages are read |
| `show_last_seen` | boolean | `true`/`false` | `true` | Show last seen timestamp |
| `show_typing_indicator` | boolean | `true`/`false` | `true` | Show typing indicators |
| `who_can_see_profile` | string | `everyone`, `contacts`, `nobody` | `everyone` | Who can view profile |
| `who_can_add_to_groups` | string | `everyone`, `contacts`, `nobody` | `everyone` | Who can add to groups |
| `disappearing_messages_default` | integer/null | seconds or `null` | `null` | Default message expiration |

## Privacy Endpoints

### 1. Get Privacy Settings

**Endpoint**: `GET /api/v1/privacy/settings`
**Description**: Retrieves the current privacy settings for the authenticated user.

**Headers**:
- `Authorization: Bearer <access_token>`

**Response**:
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "show_read_receipts": true,
  "show_last_seen": true,
  "show_typing_indicator": true,
  "who_can_see_profile": "everyone",
  "who_can_add_to_groups": "contacts",
  "disappearing_messages_default": 86400,
  "updated_at": "2025-12-04T07:00:00Z"
}
```

**Status Codes**:
- `200 OK`: Settings retrieved successfully
- `401 Unauthorized`: Invalid or missing authentication token
- `429 Too Many Requests`: Rate limit exceeded

**Rate Limiting**: 15 requests per minute per user

---

### 2. Update Privacy Setting

**Endpoint**: `PUT /api/v1/privacy/settings`
**Description**: Updates one or more privacy settings.

**Headers**:
- `Authorization: Bearer <access_token>`
- `Content-Type: application/json`

**Request Body**:
```json
{
  "setting": "show_read_receipts",
  "value": false
}
```

**Response**:
```json
{
  "status": "updated"
}
```

**Status Codes**:
- `200 OK`: Setting updated successfully
- `400 Bad Request`: Invalid setting name or value
- `401 Unauthorized`: Invalid or missing authentication token
- `429 Too Many Requests`: Rate limit exceeded

**Rate Limiting**: 5 requests per minute per user

---

## Privacy Logic

### Profile Visibility

**who_can_see_profile** controls who can view:
- Last seen timestamp
- Profile information
- Online status

| Setting | Can View |
|---------|----------|
| `everyone` | All users |
| `contacts` | Only contacts |
| `nobody` | No one |

### Group Membership

**who_can_add_to_groups** controls who can add the user to groups:

| Setting | Can Add |
|---------|---------|
| `everyone` | Any user |
| `contacts` | Only contacts |
| `nobody` | No one (admin only) |

### Disappearing Messages

**disappearing_messages_default** sets the default expiration for new messages:

- `null`: Messages don't disappear (default)
- `300`: 5 minutes
- `3600`: 1 hour
- `86400`: 24 hours
- `604800`: 7 days

---

## Examples

### Get Current Settings

```bash
curl -X GET https://api.yourdomain.com/v1/privacy/settings \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

### Disable Read Receipts

```bash
curl -X PUT https://api.yourdomain.com/v1/privacy/settings \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "setting": "show_read_receipts",
    "value": false
  }'
```

### Set Profile to Contacts Only

```bash
curl -X PUT https://api.yourdomain.com/v1/privacy/settings \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "setting": "who_can_see_profile",
    "value": "contacts"
  }'
```

### Enable 24-Hour Disappearing Messages

```bash
curl -X PUT https://api.yourdomain.com/v1/privacy/settings \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "setting": "disappearing_messages_default",
    "value": 86400
  }'
```

### Disable Disappearing Messages

```bash
curl -X PUT https://api.yourdomain.com/v1/privacy/settings \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "setting": "disappearing_messages_default",
    "value": null
  }'
```

---

## Privacy Enforcement

### Read Receipts

When `show_read_receipts` is `false`:
- User's read status is not sent to message senders
- Other users see messages as "delivered" but not "read"

### Last Seen

When `show_last_seen` is `false`:
- User's last seen timestamp is hidden
- Other users see "last seen unavailable"

### Typing Indicators

When `show_typing_indicator` is `false`:
- User's typing status is not broadcast
- Other users don't see typing indicators

### Profile Access

When `who_can_see_profile` is restricted:
- Non-contacts cannot view profile information
- API returns limited profile data

### Group Additions

When `who_can_add_to_groups` is restricted:
- Only authorized users can add the user to groups
- API validates permissions before group membership

---

## Related APIs

- **[User Management API](API_USERS.md)** - User profile information
- **[Groups API](API_GROUPS.md)** - Group membership management
- **[Messages API](API_MESSAGES.md)** - Message read receipts and status

---

## Privacy Considerations

### Data Retention

- Privacy settings are stored securely in the database
- Settings changes are logged for audit purposes
- Deleted accounts have privacy settings permanently removed

### Default Settings

- All users start with privacy-friendly defaults
- Settings can be changed at any time
- Changes take effect immediately

### Contact Verification

- "contacts" restrictions require mutual contact relationships
- Contact status is verified in real-time
- Changes to contact relationships may affect privacy settings

---

*© 2025 SilentRelay. All rights reserved.*