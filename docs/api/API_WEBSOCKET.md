# WebSocket API

The WebSocket API provides real-time messaging capabilities, enabling instant message delivery, presence updates, and multi-device synchronization.

## Overview

**Endpoint**: `wss://api.yourdomain.com/ws?token=<jwt_token>`
**Protocol**: WebSocket with JWT authentication
**Message Format**: JSON-encoded messages with type and payload

## Connection Establishment

### Authentication

**Connection URL**:
```
wss://api.yourdomain.com/ws?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Alternative Headers**:
```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Sec-WebSocket-Protocol: Bearer, eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Connection Flow**:
1. Client establishes WebSocket connection with JWT token
2. Server validates token and extracts user/device information
3. Connection registered with user's WebSocket hub
4. Client can send/receive messages in real-time

---

## Message Types

### 1. Message Delivery

**Type**: `message`
**Direction**: Server → Client
**Description**: Delivers a new message to the client.

**Payload**:
```json
{
  "type": "message",
  "payload": {
    "message_id": "550e8400-e29b-41d4-a716-446655440000",
    "sender_id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
    "conversation_id": "conv-550e8400-e29b-41d4-a716-446655440000",
    "encrypted_content": "base64_encrypted_message",
    "timestamp": "2025-12-04T07:00:00Z",
    "message_type": "text",
    "metadata": {
      "content_type": "text/plain",
      "priority": "normal"
    },
    "attachments": [
      {
        "media_id": "media-550e8400-e29b-41d4-a716-446655440000",
        "content_type": "image/jpeg",
        "size": 2048,
        "encrypted": true
      }
    ]
  }
}
```

---

### 2. Message Status Update

**Type**: `message_status`
**Direction**: Server → Client
**Description**: Notifies about message delivery/read status changes.

**Payload**:
```json
{
  "type": "message_status",
  "payload": {
    "message_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "read",
    "timestamp": "2025-12-04T07:05:00Z",
    "recipient_id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
  }
}
```

**Status Values**: `sent`, `delivered`, `read`, `failed`

---

### 3. Presence Update

**Type**: `presence`
**Direction**: Server → Client
**Description**: Notifies about user presence status changes.

**Payload**:
```json
{
  "type": "presence",
  "payload": {
    "user_id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
    "status": "online",
    "last_seen": "2025-12-04T07:00:00Z",
    "devices": [
      {
        "device_id": "550e8400-e29b-41d4-a716-446655440000",
        "device_type": "web",
        "status": "active"
      }
    ]
  }
}
```

**Status Values**: `online`, `offline`, `away`, `busy`

---

### 4. Device Approval Request

**Type**: `device_approval_request`
**Direction**: Server → Client (Primary Device)
**Description**: Notifies primary device about new device approval request.

**Payload**:
```json
{
  "type": "device_approval_request",
  "payload": {
    "request_id": "req-550e8400-e29b-41d4-a716-446655440000",
    "device_name": "New Laptop",
    "device_type": "web",
    "code": "123456",
    "expires_at": "2025-12-04T07:15:00Z",
    "ip_address": "192.168.1.102",
    "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
    "location": "United States"
  }
}
```

---

### 5. Device Approval Response

**Type**: `device_approved` / `device_denied`
**Direction**: Server → Client (New Device)
**Description**: Notifies new device about approval status.

**Approved Payload**:
```json
{
  "type": "device_approved",
  "payload": {
    "request_id": "req-550e8400-e29b-41d4-a716-446655440000",
    "message": "Device approved. You can now complete login."
  }
}
```

**Denied Payload**:
```json
{
  "type": "device_denied",
  "payload": {
    "request_id": "req-550e8400-e29b-41d4-a716-446655440000",
    "message": "Device access denied by primary device."
  }
}
```

---

### 6. Group Message

**Type**: `group_message`
**Direction**: Server → Client
**Description**: Delivers a group message to all members.

**Payload**:
```json
{
  "type": "group_message",
  "payload": {
    "message_id": "550e8400-e29b-41d4-a716-446655440000",
    "group_id": "group-550e8400-e29b-41d4-a716-446655440000",
    "sender_id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
    "encrypted_content": "base64_encrypted_message",
    "timestamp": "2025-12-04T07:00:00Z",
    "message_type": "text"
  }
}
```

---

### 7. Typing Indicator

**Type**: `typing`
**Direction**: Client → Server → Other Clients
**Description**: Notifies when a user is typing in a conversation.

**Payload**:
```json
{
  "type": "typing",
  "payload": {
    "conversation_id": "conv-550e8400-e29b-41d4-a716-446655440000",
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "is_typing": true,
    "timestamp": "2025-12-04T07:00:00Z"
  }
}
```

---

### 8. Media Key Exchange

**Type**: `media_key`
**Direction**: Client → Server → Client
**Description**: Exchanges media encryption keys directly between clients for end-to-end encrypted media.

**Payload**:
```json
{
  "type": "media_key",
  "payload": {
    "media_id": "media-550e8400-e29b-41d4-a716-446655440000",
    "recipient_id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
    "encrypted_key": "base64_encrypted_media_key",
    "algorithm": "AES-256-GCM",
    "timestamp": "2025-12-04T07:00:00Z"
  }
}
```

**Notes**:
- The `encrypted_key` is encrypted using the recipient's public key for E2EE
- Server forwards the message to the specified recipient without reading the content
- Clients must have the media_id from a message attachment to decrypt the key

---

### 9. Error Message

**Type**: `error`
**Direction**: Server → Client
**Description**: Notifies client about WebSocket errors.

**Payload**:
```json
{
  "type": "error",
  "payload": {
    "error_code": "auth_failed",
    "message": "Invalid authentication token",
    "timestamp": "2025-12-04T07:00:00Z",
    "reconnect": true,
    "retry_after": 5
  }
}
```

**Common Error Codes**:
- `auth_failed`: Authentication failure
- `rate_limited`: Too many requests
- `invalid_message`: Malformed message format
- `server_error`: Internal server error
- `connection_timeout`: Inactivity timeout

---

### 10. Message Delivery (Updated)

**Type**: `deliver`
**Direction**: Server → Client
**Description**: Delivers encrypted messages to recipients (updated from generic "message").

**Payload**:
```json
{
  "type": "deliver",
  "message_id": "550e8400-e29b-41d4-a716-446655440000",
  "sender_id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
  "timestamp": "2025-12-04T07:00:00Z",
  "payload": {
    "receiver_id": "550e8400-e29b-41d4-a716-446655440001",
    "group_id": null,
    "ciphertext": "base64_encrypted_message",
    "message_type": "text",
    "media_id": null,
    "media_type": null
  }
}
```

---

### 11. Message Sent Acknowledgment

**Type**: `sent_ack`
**Direction**: Server → Client
**Description**: Confirms that a sent message was successfully stored and queued for delivery.

**Payload**:
```json
{
  "type": "sent_ack",
  "message_id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-12-04T07:00:00Z",
  "payload": {
    "status": "sent"
  }
}
```

---

### 12. Status Update

**Type**: `status_update`
**Direction**: Server → Client
**Description**: Notifies about message delivery and read status changes.

**Payload**:
```json
{
  "type": "status_update",
  "message_id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-12-04T07:05:00Z",
  "payload": {
    "status": "delivered"
  }
}
```

---

### 13. User Presence

**Type**: `user_online` / `user_offline`
**Direction**: Server → Client
**Description**: Broadcasts user online/offline status to contacts.

**Online Payload**:
```json
{
  "type": "user_online",
  "sender_id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
  "server_id": "server-1",
  "timestamp": "2025-12-04T07:00:00Z",
  "payload": {
    "user_id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
  }
}
```

---

### 14. Heartbeat Acknowledgment

**Type**: `heartbeat_ack`
**Direction**: Server → Client
**Description**: Acknowledges client heartbeat to maintain connection.

**Payload**:
```json
{
  "type": "heartbeat_ack",
  "timestamp": "2025-12-04T07:00:00Z"
}
```

---

### 15. WebRTC Call Signaling

**Types**: `call_offer`, `call_answer`, `call_reject`, `call_end`, `call_busy`, `ice_candidate`
**Direction**: Client ↔ Server ↔ Client
**Description**: Signaling messages for WebRTC voice/video calls.

**Call Offer Example**:
```json
{
  "type": "call_offer",
  "sender_id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
  "timestamp": "2025-12-04T07:00:00Z",
  "payload": {
    "target_id": "550e8400-e29b-41d4-a716-446655440000",
    "call_type": "video",
    "sdp": "v=0\r\no=- 123456789 0 IN IP4 127.0.0.1\r\n..."
  }
}
```

---

### 16. Device Synchronization

**Types**: `sync_request`, `sync_data`, `sync_ack`
**Direction**: Client → Server → Client
**Description**: Encrypted device-to-device synchronization.

**Sync Request Example**:
```json
{
  "type": "sync_request",
  "sender_id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
  "device_id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-12-04T07:00:00Z",
  "payload": {
    "target_device_id": "550e8400-e29b-41d4-a716-446655440001"
  }
}
```

---

## Security Considerations

### Message Authentication

- **JWT Validation**: All connections require valid JWT tokens
- **Device Binding**: Tokens bound to specific devices
- **Rate Limiting**: WebSocket message rate limiting
- **Connection Limits**: Maximum concurrent connections per user

### Encryption Requirements

- **Transport Security**: TLS 1.3 required for all connections
- **Message Encryption**: All message content E2EE
- **Payload Validation**: Strict JSON schema validation
- **Size Limits**: Maximum message size enforcement

### Connection Management

- **Heartbeat**: Regular ping/pong messages
- **Reconnection**: Automatic reconnection logic
- **Session Recovery**: State restoration after reconnect
- **Error Handling**: Comprehensive error recovery

---

## WebSocket Protocol

### Message Format

```json
{
  "type": "message_type",
  "payload": {
    // Type-specific payload
  },
  "timestamp": "2025-12-04T07:00:00Z",
  "message_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### Supported Message Types

| Type | Direction | Description |
|------|-----------|-------------|
| `deliver` | S→C | Message delivery to recipient |
| `sent_ack` | S→C | Acknowledge message was sent |
| `status_update` | S→C | Message status changed (delivered/read) |
| `user_online` | S→C | User came online |
| `user_offline` | S→C | User went offline |
| `device_approval_request` | S→C | Device approval request |
| `device_approved` | S→C | Device approval successful |
| `device_denied` | S→C | Device approval denied |
| `group_message` | S→C | Group message delivery |
| `media_key` | C→S→C | Media encryption key exchange |
| `typing` | C↔S↔C | Typing indicator |
| `heartbeat_ack` | S→C | Heartbeat acknowledgment |
| `error` | S→C | Error notification |
| `call_offer` | C↔S↔C | WebRTC call offer |
| `call_answer` | C↔S↔C | WebRTC call answer |
| `call_reject` | C↔S↔C | Call rejection |
| `call_end` | C↔S↔C | Call termination |
| `call_busy` | S→C | User is busy |
| `ice_candidate` | C↔S↔C | WebRTC ICE candidate |
| `sync_request` | C→S→C | Device sync request |
| `sync_data` | C→S→C | Device sync data |
| `sync_ack` | C→S→C | Device sync acknowledgment |
| `send` | C→S | Send encrypted message |
| `delivery_ack` | C→S | Acknowledge message delivery |
| `read_receipt` | C→S | Mark messages as read |
| `heartbeat` | C→S | Keep-alive ping |
| `ping` | S→C | Keepalive ping |
| `pong` | C→S | Keepalive response |

---

## Examples

### WebSocket Connection Example

```javascript
// JavaScript WebSocket client example
const socket = new WebSocket(
  `wss://api.yourdomain.com/ws?token=${accessToken}`
);

// Connection opened
socket.addEventListener('open', (event) => {
  console.log('WebSocket connected');
  socket.send(JSON.stringify({
    type: 'presence',
    payload: { status: 'online' }
  }));
});

// Message received
socket.addEventListener('message', (event) => {
  const message = JSON.parse(event.data);
  switch (message.type) {
    case 'message':
      handleNewMessage(message.payload);
      break;
    case 'presence':
      updatePresence(message.payload);
      break;
    case 'error':
      handleError(message.payload);
      break;
  }
});

// Error handling
socket.addEventListener('error', (error) => {
  console.error('WebSocket error:', error);
});

// Connection closed
socket.addEventListener('close', (event) => {
  console.log('WebSocket disconnected');
  // Implement reconnection logic
});
```

### Message Sending Example

```json
// Send a text message
{
  "type": "message",
  "payload": {
    "recipient_id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
    "encrypted_content": "base64_encrypted_message",
    "message_type": "text",
    "timestamp": "2025-12-04T07:00:00Z"
  }
}

// Send a typing indicator
{
  "type": "typing",
  "payload": {
    "conversation_id": "conv-550e8400-e29b-41d4-a716-446655440000",
    "is_typing": true
  }
}
```

---

## Related APIs

- **[Authentication API](API_AUTHENTICATION.md)** - JWT token management for WebSocket auth
- **[Messaging API](API_MESSAGES.md)** - Message retrieval and status updates
- **[User Management API](API_USERS.md)** - User information for message context
- **[Media API](API_MEDIA.md)** - Media attachments referenced in messages

---

## WebSocket Best Practices

### Connection Management

- **Reconnection Strategy**: Exponential backoff for reconnection
- **Heartbeat**: Implement 30-second ping/pong
- **State Sync**: Request full state after reconnection
- **Error Recovery**: Handle errors gracefully

### Performance Optimization

- **Message Batching**: Batch multiple messages when possible
- **Compression**: Use message compression for large payloads
- **Throttling**: Implement client-side rate limiting
- **Prioritization**: Prioritize important messages

### Security Recommendations

- **Token Rotation**: Rotate JWT tokens periodically
- **Origin Validation**: Verify WebSocket origin
- **Payload Validation**: Validate all incoming messages
- **Encryption**: Ensure all sensitive data is encrypted

---

*© 2025 SilentRelay. All rights reserved.*