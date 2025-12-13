# Messaging API

The Messaging API provides endpoints for retrieving message history and updating message statuses in the SilentRelay platform.

## Overview

**Base Path**: `/api/v1/messages`
**Authentication**: Requires valid JWT authentication
**Rate Limiting**: Normal rate limits apply

## Message Retrieval

### 1. Get Messages

**Endpoint**: `GET /api/v1/messages`
**Description**: Retrieves pending messages for the authenticated user.

**Headers**:
- `Authorization: Bearer <access_token>`

**Query Parameters**:
- `limit`: Maximum number of messages to retrieve (default: 50, max: 100)
- `offset`: Pagination offset
- `before`: Retrieve messages before this timestamp
- `after`: Retrieve messages after this timestamp
- `with`: Filter messages with specific user (UUID)

**Response**:
```json
{
  "messages": [
    {
      "message_id": "550e8400-e29b-41d4-a716-446655440000",
      "sender_id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
      "recipient_id": "550e8400-e29b-41d4-a716-446655440000",
      "conversation_id": "conv-550e8400-e29b-41d4-a716-446655440000",
      "encrypted_content": "base64_encrypted_message",
      "timestamp": "2025-12-04T07:00:00Z",
      "status": "sent",
      "message_type": "text",
      "metadata": {
        "content_type": "text/plain",
        "content_length": 128,
        "priority": "normal"
      },
      "attachments": [
        {
          "attachment_id": "attach-550e8400-e29b-41d4-a716-446655440000",
          "media_id": "media-550e8400-e29b-41d4-a716-446655440000",
          "content_type": "image/jpeg",
          "size": 2048,
          "encrypted": true
        }
      ]
    }
  ],
  "total": 1,
  "limit": 50,
  "offset": 0,
  "has_more": false
}
```

**Status Codes**:
- `200 OK`: Messages retrieved successfully
- `400 Bad Request`: Invalid query parameters
- `401 Unauthorized`: Invalid or missing authentication token
- `429 Too Many Requests`: Rate limit exceeded

**Rate Limiting**: 60 requests per minute per user

**Important Notes**:
- **End-to-End Encryption**: All message content is encrypted client-side
- **Server Storage**: Only stores encrypted ciphertext and metadata
- **No Content Access**: Server cannot decrypt or read message content
- **Pagination**: Use `limit` and `offset` for large message histories

---

### 2. Update Message Status

**Endpoint**: `PUT /api/v1/messages/{messageId}/status`
**Description**: Updates the delivery or read status of a message.

**Headers**:
- `Authorization: Bearer <access_token>`
- `Content-Type: application/json`

**Path Parameters**:
- `messageId`: UUID of the message to update

**Request Body**:
```json
{
  "status": "delivered"
}
```

**Valid Status Values**:
- `sent`: Message sent from device
- `delivered`: Message delivered to recipient device
- `read`: Message read by recipient
- `failed`: Message delivery failed

**Response**:
```json
{
  "message_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "delivered"
}
```

**Status Codes**:
- `200 OK`: Status updated successfully
- `400 Bad Request`: Invalid message ID or status value
- `401 Unauthorized`: Invalid or missing authentication token
- `403 Forbidden`: Not authorized to update this message status
- `404 Not Found`: Message not found
- `429 Too Many Requests`: Rate limit exceeded

**Rate Limiting**: 120 requests per minute per user

**WebSocket Integration**:
- Status updates trigger real-time notifications via WebSocket
- Recipient devices receive `message_status_update` events

---

## Message Attachments

Message attachments are handled through the **[Media API](API_MEDIA.md)** with the following workflow:

1. **Upload Media**: Use `/media/upload-url` to get presigned URL
2. **Encrypt Content**: Client encrypts media before upload
3. **Attach to Message**: Include media ID in message metadata
4. **Download Media**: Use `/media/{mediaId}` with proper authentication

---

## Security Considerations

### Message Privacy

- **No Server Access**: Server cannot read message content (E2EE)
- **Metadata Protection**: Minimal metadata stored (timestamps, IDs only)
- **Sealed Sender**: Recipient identity protected in transit
- **Forward Secrecy**: Each message uses unique encryption keys

### Rate Limiting Strategy

- **Message Retrieval**: 60/minute - supports active messaging
- **Status Updates**: 120/minute - allows for multi-device sync
- **Anti-Spam**: Additional limits on message sending via WebSocket

### Data Retention

- **Default Retention**: Messages stored until deleted by user
- **Disappearing Messages**: Auto-delete after configurable time
- **GDPR Compliance**: Full message deletion on user request

---

## Examples

### Message Retrieval Examples

```bash
# Get recent messages (default limit: 50)
curl -X GET https://api.yourdomain.com/v1/messages \
  -H "Authorization: Bearer $ACCESS_TOKEN"

# Get messages with specific user
curl -X GET "https://api.yourdomain.com/v1/messages?with=6ba7b810-9dad-11d1-80b4-00c04fd430c8" \
  -H "Authorization: Bearer $ACCESS_TOKEN"

# Get messages with pagination
curl -X GET "https://api.yourdomain.com/v1/messages?limit=20&offset=40" \
  -H "Authorization: Bearer $ACCESS_TOKEN"

# Get messages from last 24 hours
curl -X GET "https://api.yourdomain.com/v1/messages?after=$(date -d '24 hours ago' -u +'%Y-%m-%dT%H:%M:%SZ')" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

### Message Status Updates

```bash
# Mark message as delivered
curl -X PUT https://api.yourdomain.com/v1/messages/$MESSAGE_ID/status \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "delivered"}'

# Mark message as read
curl -X PUT https://api.yourdomain.com/v1/messages/$MESSAGE_ID/status \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "read"}'
```

---

## Related APIs

- **[Group API](API_GROUPS.md)** - Group message management
- **[Media API](API_MEDIA.md)** - Media attachment handling
- **[WebSocket API](API_WEBSOCKET.md)** - Real-time message delivery
- **[User Management API](API_USERS.md)** - User key retrieval for encryption

---

## Message Types Reference

| Type | Description | Encryption |
|------|-------------|------------|
| `text` | Plain text messages | E2EE |
| `image` | Image attachments | E2EE |
| `video` | Video attachments | E2EE |
| `audio` | Audio messages | E2EE |
| `file` | Document attachments | E2EE |
| `location` | Geolocation sharing | E2EE |
| `contact` | Contact card sharing | E2EE |
| `system` | System notifications | Server-side |

---

*© 2025 SilentRelay. All rights reserved.*