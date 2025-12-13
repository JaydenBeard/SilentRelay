# API Best Practices

This guide provides recommendations and best practices for implementing and using the SilentRelay API effectively.

## Getting Started

### API Client Setup

**Recommended Libraries**:
- **JavaScript/TypeScript**: Axios, Fetch API
- **Python**: requests, httpx
- **Java**: OkHttp, Retrofit
- **Swift**: URLSession, Alamofire
- **Kotlin**: Ktor, Retrofit

**Base Configuration**:
```javascript
// JavaScript example
const apiClient = axios.create({
  baseURL: 'https://api.yourdomain.com/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
    'Accept': 'application/json'
  }
});

// Add request interceptor for auth
apiClient.interceptors.request.use((config) => {
  const token = getAccessToken(); // Your token storage
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Add response interceptor for error handling
apiClient.interceptors.response.use(
  (response) => response,
  (error) => handleApiError(error)
);
```

---

## Authentication Best Practices

### Token Management

**Storage Recommendations**:
- **Web**: HTTP-only, Secure, SameSite cookies
- **Mobile**: Platform keychain/keystore
- **Desktop**: Encrypted local storage

**Token Rotation**:
```javascript
// Token refresh logic
async function ensureValidToken() {
  const token = getAccessToken();
  const expiresAt = getTokenExpiration();

  if (!token || (expiresAt && new Date(expiresAt) < new Date())) {
    try {
      const refreshToken = getRefreshToken();
      const response = await apiClient.post('/auth/refresh', {
        refresh_token: refreshToken
      });
      storeAccessToken(response.data.access_token);
      storeTokenExpiration(response.data.expires_at);
      return response.data.access_token;
    } catch (error) {
      // Full re-authentication required
      redirectToLogin();
    }
  }
  return token;
}
```

---

### Device Management

**Device Approval Flow**:
1. **Request Approval**: New device initiates approval
2. **Notify Primary**: Primary device receives WebSocket notification
3. **User Decision**: Primary device user approves/denies
4. **Complete Login**: New device can authenticate

**Best Practices**:
- Implement device approval UI with security warnings
- Show device information (name, type, location, IP)
- Provide clear approval/denial options
- Implement timeout handling (15 minute expiration)

---

## Messaging Best Practices

### Message Handling

**Encryption Workflow**:
1. **Retrieve Keys**: Get recipient's public keys
2. **Establish Session**: Perform X3DH handshake
3. **Encrypt Message**: Use Double Ratchet for encryption
4. **Send Message**: Transmit via WebSocket
5. **Store Locally**: Save encrypted message locally
6. **Update Status**: Send delivery/read receipts

**Message Storage**:
```javascript
// Example message storage structure
const messageStore = {
  conversations: {
    'conv-uuid': {
      id: 'conv-uuid',
      with: 'user-uuid',
      messages: [
        {
          id: 'msg-uuid',
          sender: 'user-uuid',
          timestamp: '2025-12-04T07:00:00Z',
          content: 'encrypted-base64',
          status: 'delivered',
          type: 'text',
          attachments: []
        }
      ],
      unreadCount: 0,
      lastMessage: 'msg-uuid',
      lastActivity: '2025-12-04T07:00:00Z'
    }
  }
};
```

---

### WebSocket Implementation

**Connection Management**:
```javascript
// WebSocket client with reconnection
class MessengerWebSocket {
  constructor(url, token) {
    this.url = url;
    this.token = token;
    this.socket = null;
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
    this.reconnectDelay = 1000; // 1 second
    this.messageQueue = [];
    this.connect();
  }

  connect() {
    this.socket = new WebSocket(`${this.url}?token=${this.token}`);

    this.socket.onopen = () => {
      this.reconnectAttempts = 0;
      this.reconnectDelay = 1000;
      this.flushQueue();
    };

    this.socket.onmessage = (event) => {
      this.handleMessage(event);
    };

    this.socket.onclose = () => {
      this.scheduleReconnect();
    };

    this.socket.onerror = (error) => {
      console.error('WebSocket error:', error);
    };
  }

  scheduleReconnect() {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      setTimeout(() => {
        this.reconnectAttempts++;
        this.reconnectDelay = Math.min(this.reconnectDelay * 2, 30000); // Max 30s
        this.connect();
      }, this.reconnectDelay);
    }
  }

  send(message) {
    if (this.socket && this.socket.readyState === WebSocket.OPEN) {
      this.socket.send(JSON.stringify(message));
    } else {
      this.messageQueue.push(message);
      if (this.messageQueue.length > 100) {
        this.messageQueue.shift(); // Prevent memory leaks
      }
    }
  }

  flushQueue() {
    while (this.messageQueue.length > 0) {
      const message = this.messageQueue.shift();
      this.send(message);
    }
  }
}
```

---

## Media Handling Best Practices

### Upload Workflow

```javascript
// Media upload with progress tracking
async function uploadMedia(file, encryptionKey) {
  try {
    // Step 1: Encrypt the file
    const encryptedData = await encryptFile(file, encryptionKey);

    // Step 2: Get upload URL
    const uploadResponse = await apiClient.post('/media/upload-url', {
      file_name: file.name,
      content_type: file.type,
      file_size: encryptedData.size,
      encryption_method: 'AES-256-GCM'
    });

    // Step 3: Upload with progress
    const uploadResult = await uploadWithProgress(
      uploadResponse.data.uploadUrl,
      encryptedData,
      (progress) => updateUploadProgress(progress)
    );

    // Step 4: Return media info for message attachment
    return {
      mediaId: uploadResult.fileId,
      downloadUrl: uploadResponse.data.downloadUrl,
      encryptionInfo: {
        algorithm: 'AES-256-GCM',
        keyId: encryptionKey.id,
        iv: encryptionKey.iv
      }
    };
  } catch (error) {
    handleUploadError(error);
    throw error;
  }
}
```

---

### Download Workflow

```javascript
// Media download with decryption
async function downloadMedia(mediaId, encryptionKey) {
  try {
    // Step 1: Get download URL
    const downloadResponse = await apiClient.get(`/media/${mediaId}`);

    // Step 2: Download the file
    const downloadResult = await downloadWithProgress(
      downloadResponse.data.url,
      (progress) => updateDownloadProgress(progress)
    );

    // Step 3: Decrypt the file
    const decryptedData = await decryptFile(
      downloadResult,
      encryptionKey
    );

    return decryptedData;
  } catch (error) {
    handleDownloadError(error);
    throw error;
  }
}
```

---

## Group Management Best Practices

### Group Creation

```javascript
// Create group with proper error handling
async function createGroup(name, members) {
  try {
    // Validate group name
    if (!name || name.length > 50) {
      throw new Error('Invalid group name');
    }

    // Validate members
    if (!members || members.length < 1 || members.length > 100) {
      throw new Error('Invalid number of members');
    }

    // Create group
    const response = await apiClient.post('/groups', {
      name: name,
      description: `${name} group`,
      members: members
    });

    // Generate and distribute group keys
    await distributeGroupKeys(response.data.group_id, members);

    return response.data;
  } catch (error) {
    handleGroupError(error);
    throw error;
  }
}
```

---

## Performance Optimization

### Caching Strategies

**Recommended Caching**:
- **User Keys**: Cache for 1 hour (key rotation period)
- **Media URLs**: Cache for 30 minutes (presigned URL expiration)
- **Message History**: Cache locally with sync
- **Group Members**: Cache with presence updates

### Batch Operations

**Message Status Updates**:
```javascript
// Batch status updates
async function updateMessageStatuses(statusUpdates) {
  try {
    // Group by conversation for efficiency
    const byConversation = groupBy(statusUpdates, 'conversation_id');

    // Process in batches of 20
    for (const [conversationId, updates] of Object.entries(byConversation)) {
      const batches = chunk(updates, 20);
      for (const batch of batches) {
        await Promise.all(batch.map(update =>
          apiClient.put(`/messages/${update.message_id}/status`, {
            status: update.status
          })
        ));
      }
    }
  } catch (error) {
    handleBatchError(error);
  }
}
```

---

## Security Best Practices

### Client-Side Security

**Essential Implementations**:
- **Token Storage**: Use secure storage mechanisms
- **Biometric Protection**: Protect sensitive operations
- **Certificate Pinning**: Implement TLS certificate pinning
- **Jailbreak Detection**: Prevent use on compromised devices

**Data Protection**:
```javascript
// Secure data storage example
import * as Keychain from 'react-native-keychain';

async function storeSensitiveData(key, value) {
  await Keychain.setGenericPassword(key, value, {
    accessControl: Keychain.ACCESS_CONTROL.BIOMETRY_ANY,
    accessible: Keychain.ACCESSIBLE.WHEN_UNLOCKED
  });
}

async function retrieveSensitiveData(key) {
  const credentials = await Keychain.getGenericPassword();
  if (credentials && credentials.username === key) {
    return credentials.password;
  }
  return null;
}
```

---

### Network Security

**Recommended Configurations**:
- **TLS 1.3 Only**: No fallback to older versions
- **Certificate Pinning**: Pin server certificates
- **Network Isolation**: Separate API traffic
- **VPN Detection**: Warn users on VPNs

```javascript
// Certificate pinning example
const pinnedCerts = [
  'sha256/AbCdEfGhIjKlMnOpQrStUvWxYz1234567890',
  'sha256/ZyXwVuTsRqPoNmlKjIhGfEdCbA0987654321'
];

// Configure axios with certificate pinning
const apiClient = axios.create({
  httpsAgent: new https.Agent({
    cert: fs.readFileSync('path-to-cert.pem'),
    rejectUnauthorized: true
  })
});
```

---

## Monitoring and Analytics

### Error Tracking

**Recommended Implementation**:
```javascript
// Error tracking middleware
function setupErrorTracking() {
  apiClient.interceptors.response.use(
    (response) => response,
    (error) => {
      const errorData = {
        timestamp: new Date().toISOString(),
        endpoint: error.config?.url,
        method: error.config?.method,
        status: error.response?.status,
        code: error.response?.data?.error?.code,
        message: error.response?.data?.error?.message,
        userAgent: navigator.userAgent,
        platform: getPlatformInfo()
      };

      // Send to error tracking service
      trackError(errorData);

      // Re-throw for application handling
      return Promise.reject(error);
    }
  );
}
```

---

## Related APIs

- **[API Documentation Index](API_DOCUMENTATION_INDEX.md)** - Main API navigation
- **[API Security](API_SECURITY.md)** - Security requirements and error handling
- **[API Changelog](API_CHANGELOG.md)** - Version history and updates

---

## Implementation Checklist

### Basic Integration
- [ ] Set up API client with proper configuration
- [ ] Implement token management and rotation
- [ ] Handle authentication and error responses
- [ ] Implement basic messaging functionality
- [ ] Set up WebSocket connection with reconnection

### Advanced Features
- [ ] Implement device approval workflow
- [ ] Add group management capabilities
- [ ] Support media upload/download with encryption
- [ ] Implement presence and typing indicators
- [ ] Add message search and history

### Security & Compliance
- [ ] Implement proper token storage
- [ ] Set up certificate pinning
- [ ] Add biometric protection
- [ ] Implement rate limit handling
- [ ] Set up error tracking and monitoring

### Performance Optimization
- [ ] Implement caching strategies
- [ ] Add batch operations
- [ ] Set up connection pooling
- [ ] Implement lazy loading
- [ ] Add background sync

---

*© 2025 SilentRelay. All rights reserved.*