# Group API

The Group API provides endpoints for creating and managing group chats, including member management and group metadata.

## Overview

**Base Path**: `/api/v1/groups`
**Authentication**: Requires valid JWT authentication
**Rate Limiting**: Normal rate limits apply

## Group Management

### 1. Create Group

**Endpoint**: `POST /api/v1/groups`
**Description**: Creates a new group chat.

**Headers**:
- `Authorization: Bearer <access_token>`
- `Content-Type: application/json`

**Request Body**:
```json
{
  "name": "Project Team",
  "description": "Discussion about the upcoming project",
  "members": [
    "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
    "7ba7b810-9dad-11d1-80b4-00c04fd430c8"
  ],
  "is_public": false,
  "avatar_url": "https://example.com/group_avatar.jpg"
}
```

**Response**:
```json
{
  "group_id": "group-550e8400-e29b-41d4-a716-446655440000",
  "name": "Project Team",
  "created_at": "2025-12-04T07:00:00Z",
  "created_by": "550e8400-e29b-41d4-a716-446655440000",
  "member_count": 3,
  "is_admin": true
}
```

**Status Codes**:
- `200 OK`: Group created successfully
- `400 Bad Request`: Invalid request format or missing fields
- `401 Unauthorized`: Invalid or missing authentication token
- `404 Not Found`: One or more member users not found
- `429 Too Many Requests`: Rate limit exceeded

**Rate Limiting**: 30 requests per minute per user

**Group Creation Notes**:
- Creator automatically becomes group admin
- Maximum 100 members per group
- Group names limited to 50 characters
- Descriptions limited to 500 characters

---

### 2. Get Group Details

**Endpoint**: `GET /api/v1/groups/{groupId}`
**Description**: Retrieves detailed information about a specific group.

**Headers**:
- `Authorization: Bearer <access_token>`

**Path Parameters**:
- `groupId`: UUID of the group to retrieve

**Response**:
```json
{
  "group_id": "group-550e8400-e29b-41d4-a716-446655440000",
  "name": "Project Team",
  "description": "Discussion about the upcoming project",
  "created_at": "2025-12-04T07:00:00Z",
  "created_by": "550e8400-e29b-41d4-a716-446655440000",
  "updated_at": "2025-12-04T07:30:00Z",
  "avatar_url": "https://example.com/group_avatar.jpg",
  "is_public": false,
  "member_count": 3,
  "is_admin": true,
  "current_user_role": "admin",
  "members": [
    {
      "user_id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "john_doe",
      "display_name": "John Doe",
      "avatar_url": "https://example.com/avatar1.jpg",
      "role": "admin",
      "joined_at": "2025-12-04T07:00:00Z",
      "is_online": true
    },
    {
      "user_id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
      "username": "jane_smith",
      "display_name": "Jane Smith",
      "avatar_url": "https://example.com/avatar2.jpg",
      "role": "member",
      "joined_at": "2025-12-04T07:05:00Z",
      "is_online": false
    }
  ]
}
```

**Status Codes**:
- `200 OK`: Group details retrieved successfully
- `400 Bad Request`: Invalid group ID format
- `401 Unauthorized`: Invalid or missing authentication token
- `403 Forbidden`: User not a member of this group
- `404 Not Found`: Group not found
- `429 Too Many Requests`: Rate limit exceeded

**Rate Limiting**: 60 requests per minute per user

---

## Group Membership Management

### 3. Add Group Member

**Endpoint**: `POST /api/v1/groups/{groupId}/members`
**Description**: Adds a new member to a group (admin only).

**Headers**:
- `Authorization: Bearer <access_token>`
- `Content-Type: application/json`

**Path Parameters**:
- `groupId`: UUID of the group

**Request Body**:
```json
{
  "user_id": "7ba7b810-9dad-11d1-80b4-00c04fd430c8",
  "encrypted_key": "base64_encrypted_group_key",
  "role": "member"
}
```

**Response**:
```json
{
  "status": "added",
  "group_id": "group-550e8400-e29b-41d4-a716-446655440000",
  "user_id": "7ba7b810-9dad-11d1-80b4-00c04fd430c8",
  "member_count": 4
}
```

**Status Codes**:
- `200 OK`: Member added successfully
- `400 Bad Request`: Invalid request format or user ID
- `401 Unauthorized`: Invalid or missing authentication token
- `403 Forbidden`: User not a group admin or group is full
- `404 Not Found`: Group or user not found
- `409 Conflict`: User already a member of this group
- `429 Too Many Requests`: Rate limit exceeded

**Rate Limiting**: 20 requests per minute per user

**Admin Requirements**:
- Only group admins can add members
- Maximum 100 members per group
- Encrypted key required for E2EE group messaging

---

### 4. Remove Group Member

**Endpoint**: `DELETE /api/v1/groups/{groupId}/members/{userId}`
**Description**: Removes a member from a group (admin only).

**Headers**:
- `Authorization: Bearer <access_token>`

**Path Parameters**:
- `groupId`: UUID of the group
- `userId`: UUID of the user to remove

**Response**:
```json
{
  "status": "removed",
  "group_id": "group-550e8400-e29b-41d4-a716-446655440000",
  "user_id": "7ba7b810-9dad-11d1-80b4-00c04fd430c8",
  "member_count": 2
}
```

**Status Codes**:
- `200 OK`: Member removed successfully
- `400 Bad Request`: Invalid group or user ID format
- `401 Unauthorized`: Invalid or missing authentication token
- `403 Forbidden`: User not a group admin or trying to remove self
- `404 Not Found`: Group or user not found
- `429 Too Many Requests`: Rate limit exceeded

**Rate Limiting**: 20 requests per minute per user

**Restrictions**:
- Admins cannot remove themselves (prevents group orphaning)
- Cannot remove the last admin (promote another member first)
- Removed users lose access to group messages and history

---

## Group Security Model

### End-to-End Encryption

- **Group Keys**: Each group has unique encryption keys
- **Key Distribution**: Encrypted keys distributed to members
- **Forward Secrecy**: Group keys rotate when members change
- **Access Control**: Only members can decrypt group messages

### Admin Privileges

- **Member Management**: Add/remove members
- **Group Settings**: Modify group name, description, avatar
- **Message Moderation**: Delete inappropriate messages
- **Role Assignment**: Promote/demote members

### Rate Limiting Strategy

- **Group Creation**: 30/minute - prevents group spam
- **Group Retrieval**: 60/minute - supports active usage
- **Membership Changes**: 20/minute - prevents abuse
- **Anti-Enumeration**: Strict limits on group discovery

---

## Examples

### Complete Group Management Flow

```bash
# Create a new group
curl -X POST https://api.yourdomain.com/v1/groups \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Project Team",
    "description": "Discussion about the upcoming project",
    "members": [
      "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
      "7ba7b810-9dad-11d1-80b4-00c04fd430c8"
    ]
  }'

# Get group details
curl -X GET https://api.yourdomain.com/v1/groups/group-550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer $ACCESS_TOKEN"

# Add a new member (admin only)
curl -X POST https://api.yourdomain.com/v1/groups/group-550e8400-e29b-41d4-a716-446655440000/members \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "8ba7b810-9dad-11d1-80b4-00c04fd430c8",
    "encrypted_key": "base64_encrypted_group_key"
  }'

# Remove a member (admin only)
curl -X DELETE https://api.yourdomain.com/v1/groups/group-550e8400-e29b-41d4-a716-446655440000/members/7ba7b810-9dad-11d1-80b4-00c04fd430c8 \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

---

## Related APIs

- **[Messaging API](API_MESSAGES.md)** - Group message retrieval and status
- **[Media API](API_MEDIA.md)** - Media attachments in group chats
- **[WebSocket API](API_WEBSOCKET.md)** - Real-time group messaging
- **[User Management API](API_USERS.md)** - User information for group members

---

## Group Features Matrix

| Feature | Available | Notes |
|---------|-----------|-------|
| **Group Creation** | [x] | Max 100 members |
| **Group Avatars** | [x] | HTTPS URLs only |
| **Member Roles** | [x] | Admin/member roles |
| **Group Search** | [ ] | Privacy protection |
| **Public Groups** | [ ] | All groups private |
| **Group Links** | [ ] | No invite links |
| **Message History** | [x] | Full E2EE history |
| **Member Online Status** | [x] | Real-time presence |

---

*© 2025 SilentRelay. All rights reserved.*