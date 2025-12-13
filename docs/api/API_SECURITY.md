# API Security & Error Handling

This document covers security considerations, error codes, rate limiting, and best practices for the SilentRelay API.

## Overview

**Security Model**: Zero Trust Architecture
**Authentication**: JWT with device binding
**Encryption**: TLS 1.3 + End-to-End Encryption
**Rate Limiting**: Multi-tier protection

---

## Authentication & Authorization

### JWT Token Security

**Token Types**:
- **Access Token**: Short-lived (1 hour), used for API requests
- **Refresh Token**: Long-lived (30 days), used to obtain new access tokens

**Token Structure**:
```json
{
  "sub": "user-uuid",
  "device_id": "device-uuid",
  "exp": 1733328400,
  "iat": 1733324800,
  "iss": "secure-messenger",
  "jti": "token-uuid"
}
```

**Security Features**:
- **Device Binding**: Tokens tied to specific devices
- **Short Expiration**: Access tokens expire quickly
- **Secure Storage**: Tokens stored in secure HTTP-only cookies
- **CSRF Protection**: Anti-CSRF tokens for web clients

---

### Authorization Headers

**Standard Format**:
```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Alternative Methods**:
- **Cookie**: `access_token` cookie (HTTP-only, Secure, SameSite=Lax)
- **Query Param**: `?token=...` (WebSocket only, not recommended)
- **Custom Header**: `X-Access-Token: ...`

---

## Error Codes & Responses

### Standard Error Format

```json
{
  "error": {
    "code": "invalid_token",
    "message": "Invalid or expired authentication token",
    "details": {
      "token_type": "access_token",
      "expiration_time": "2025-12-04T06:00:00Z",
      "current_time": "2025-12-04T07:00:00Z"
    },
    "timestamp": "2025-12-04T07:00:00Z",
    "request_id": "req-550e8400-e29b-41d4-a716-446655440000"
  }
}
```

---

### Common Error Codes

| Code | HTTP Status | Description | Recovery |
|------|-------------|-------------|----------|
| `invalid_token` | 401 | Invalid or expired JWT token | Re-authenticate |
| `missing_token` | 401 | No authentication token provided | Provide token |
| `device_mismatch` | 403 | Token not valid for this device | Use correct device |
| `rate_limited` | 429 | Too many requests | Wait and retry |
| `invalid_request` | 400 | Malformed request format | Fix request |
| `not_found` | 404 | Resource not found | Verify resource ID |
| `forbidden` | 403 | Insufficient permissions | Check authorization |
| `server_error` | 500 | Internal server error | Retry later |
| `maintenance` | 503 | Service unavailable | Check status page |

---

### Authentication Errors

| Code | Scenario | Solution |
|------|----------|----------|
| `token_expired` | Access token expired | Use refresh token |
| `token_revoked` | Token manually revoked | Full re-authentication |
| `device_changed` | Device information changed | Update device binding |
| `ip_changed` | Significant IP change detected | Multi-factor verification |
| `session_hijack` | Suspicious session activity | Force re-authentication |

---

## Enhanced Rate Limiting

The API implements sophisticated multi-tier rate limiting with Redis-backed distributed enforcement, abuse detection, and automatic strict mode activation.

### Rate Limit Response Format

```json
{
  "error": {
    "code": "rate_limited",
    "message": "Too many requests",
    "details": {
      "limit": 60,
      "remaining": 0,
      "reset_after": 30,
      "retry_after": "2025-12-04T07:01:00Z",
      "mode": "strict",
      "penalty_until": "2025-12-04T07:30:00Z"
    },
    "timestamp": "2025-12-04T07:00:00Z"
  }
}
```

**Headers**:
```http
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 30
X-RateLimit-Mode: strict
Retry-After: 30
```

---

### Multi-Tier Rate Limiting

The system enforces limits at four levels with automatic escalation:

| Level | Scope | Enforcement | Storage |
|-------|-------|-------------|---------|
| **Global** | All requests | Distributed | Redis |
| **Endpoint** | Specific endpoints | Per-endpoint | Redis |
| **IP Address** | Client IP | Per-IP | Redis |
| **User** | Authenticated users | Per-user | Redis |

---

### Rate Limit Tiers

| Endpoint Category | Normal Mode | Strict Mode | Window | Abuse Detection |
|-------------------|-------------|-------------|--------|-----------------|
| **Authentication** | 10/min | 5/min | 60s | Enhanced |
| **User Search** | 10/min | 5/min | 60s | Enhanced |
| **Device Approval** | 5/min | 3/min | 60s | Enhanced |
| **Message Retrieval** | 60/min | 30/min | 60s | Standard |
| **Message Status** | 120/min | 60/min | 60s | Standard |
| **Media Upload** | 5/min | 3/min | 60s | Enhanced |
| **Media Download** | 60/min | 30/min | 60s | Standard |
| **General API** | 1000/min | 500/min | 60s | Standard |

---

### Automatic Strict Mode

**Activation Triggers**:
- **Global Strict Mode**: When abuse detection activates penalty mode
- **Endpoint Strict Mode**: When endpoint-specific abuse detected
- **IP Strict Mode**: When IP exhibits abusive patterns
- **User Strict Mode**: When user account shows suspicious activity

**Escalation Levels**:
- **Normal Mode**: Standard limits apply
- **Strict Mode**: 50% limit reduction, enhanced monitoring
- **Penalty Box**: Temporary blocking (15-60 minutes)
- **Block Mode**: Extended blocking with manual review

---

### Abuse Detection Engine

**Detection Algorithms**:
- **Request Velocity**: Tracks requests per time window
- **Pattern Analysis**: Identifies suspicious request patterns
- **IP Reputation**: Maintains IP address reputation scores
- **User Behavior**: Monitors authenticated user activity

**Trigger Thresholds**:
- **Warning Level**: 50 requests in 5 minutes
- **Penalty Level**: 100 requests in 5 minutes → 15-minute penalty
- **Strict Level**: 200 requests in 5 minutes → 30-minute strict mode
- **Block Level**: 300 requests in 5 minutes → 1-hour block

**Penalty Box Features**:
- **Automatic Cleanup**: Expired penalties removed automatically
- **Progressive Duration**: Longer penalties for repeated offenses
- **IP/User Tracking**: Separate tracking for IPs and user accounts
- **Redis Persistence**: Distributed penalty state across servers

---

### Distributed Architecture

**Redis-Based Storage**:
- **Sorted Sets**: Efficient sliding window rate limiting
- **TTL Expiration**: Automatic cleanup of old request data
- **Atomic Operations**: Thread-safe limit enforcement
- **Cross-Server Sync**: Consistent limits across server instances

**Scalability Features**:
- **Horizontal Scaling**: Works across multiple server instances
- **Memory Efficient**: Automatic cleanup prevents memory leaks
- **High Performance**: Redis operations are sub-millisecond
- **Fault Tolerant**: Graceful degradation if Redis unavailable

---

## Security Headers

**Standard Security Headers**:
```http
Strict-Transport-Security: max-age=63072000; includeSubDomains; preload
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-inline' https://cdn.example.com; style-src 'self' 'unsafe-inline'; img-src 'self' data: https://media.example.com; font-src 'self'; connect-src 'self' wss://api.example.com; frame-src 'none'; object-src 'none'
Referrer-Policy: strict-origin-when-cross-origin
Feature-Policy: geolocation 'none'; microphone 'none'; camera 'none'; payment 'none'; usb 'none'
```

---

## API Security Best Practices

### Client-Side Security

- **Token Storage**: Use secure storage (Keychain, Keystore)
- **Token Rotation**: Rotate tokens before expiration
- **Request Signing**: Sign critical requests
- **Certificate Pinning**: Implement TLS certificate pinning

### Request Security

- **Input Validation**: Validate all API inputs
- **Content-Type**: Always specify `application/json`
- **CSRF Protection**: Include anti-CSRF tokens
- **Idempotency Keys**: Use for critical operations

### Error Handling

- **Graceful Degradation**: Handle errors without crashing
- **Retry Logic**: Implement exponential backoff
- **Fallback Mechanisms**: Provide offline capabilities
- **User Notification**: Inform users of critical errors

---

## Security Compliance

| Standard | Compliance | Notes |
|----------|------------|-------|
| **OWASP Top 10** | Fully Compliant | All critical risks addressed |
| **NIST SP 800-63** | Fully Compliant | Digital identity guidelines |
| **PCI DSS** | Level 1 | Payment data protection |
| **GDPR** | Fully Compliant | Data privacy and deletion |
| **ISO 27001** | Certified | Information security management |
| **HIPAA** | Compliant | Healthcare data protection |

---

## Related APIs

- **[Authentication API](API_AUTHENTICATION.md)** - Token management and security
- **[Device Management API](API_DEVICES.md)** - Device security and approval
- **[WebSocket API](API_WEBSOCKET.md)** - Real-time security considerations

---

## Security Checklist for API Consumers

### Implementation Requirements

- [ ] Implement proper token storage and management
- [ ] Use HTTPS for all API communications
- [ ] Validate all API responses
- [ ] Implement rate limit handling
- [ ] Secure local data storage
- [ ] Implement proper error recovery

### Recommended Practices

- [ ] Use hardware-backed keystores when available
- [ ] Implement certificate pinning
- [ ] Use WebSocket for real-time features
- [ ] Implement background sync for offline use
- [ ] Provide user education on security features

### Advanced Security

- [ ] Implement biometric authentication
- [ ] Use hardware security modules (HSM)
- [ ] Implement device attestation
- [ ] Use secure enclaves for key storage
- [ ] Implement anomaly detection

---

*© 2025 SilentRelay. All rights reserved.*