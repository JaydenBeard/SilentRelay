# SilentRelay

[![CI/CD Pipeline](https://github.com/JaydenBeard/silentrelay.com.au/actions/workflows/ci-cd.yml/badge.svg)](https://github.com/JaydenBeard/silentrelay.com.au/actions/workflows/ci-cd.yml)

A private, end-to-end encrypted messaging service for family and friends. **No one can read your messages - not even the server operators.**

## Security Features

### Encryption

- **True End-to-End Encryption**: Messages encrypted on device - server only sees ciphertext
- **Signal Protocol**: X3DH + Double Ratchet (same as Signal, WhatsApp)
- **Forward Secrecy**: Compromise of one key does not expose past or future messages
- **Post-Compromise Security**: Automatic key rotation limits damage
- **Sealed Sender**: Metadata protection - server cannot see who is messaging whom

### Key Management

- **Key Transparency Log**: Blockchain-like audit trail prevents key substitution attacks
- **Safety Numbers**: Verify contacts with QR codes or fingerprints
- **24-Word Recovery Key**: BIP39 mnemonic for backup encryption
- **Automatic Key Rotation**: Signed pre-keys rotated weekly

### Access Control

- **PIN Protection**: 4 or 6 digit PIN required on login
- **Device Binding**: Sessions tied to device fingerprints
- **Session Rotation**: Tokens rotated hourly
- **Brute Force Protection**: Progressive lockout after failed attempts

### Server Hardening

- **TLS 1.3 Only**: No fallback to vulnerable versions
- **Certificate Pinning**: Prevents MITM even with compromised CAs
- **WAF**: SQL injection, XSS, path traversal blocked
- **Rate Limiting**: Abuse prevention on all endpoints
- **Intrusion Detection**: Real-time attack detection and blocking

### Compliance

- **Audit Logging**: All security events logged (not content)
- **Disappearing Messages**: Auto-delete after configurable time
- **Right to Deletion**: Full data export and deletion
- **Privacy Settings**: Control read receipts, last seen, etc.

## Architecture

```
           
   Client        Load        Chat Servers  
  (React)     WS      Balancer            (Go x 2)     
  E2EE Keys           (HAProxy)        
                     
                                                    
         
                                                                   
                                                                   
                  
     Redis                        PostgreSQL             MinIO      
  - Presence                     - Users              - Encrypted   
  - Pub/Sub                      - Messages             Media       
  - Connections                  - Groups           
              

         
              Consul     
           - Discovery   
           - Health      
         
```

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Go 1.22+ (for local development)
- Node.js 20+ (for frontend development)

### Start Everything with Docker

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop everything
docker-compose down
```

Services will be available at:

- **Load Balancer**: http://localhost (port 80)
- **HAProxy Stats**: http://localhost:8404/stats
- **Consul UI**: http://localhost:8500
- **MinIO Console**: http://localhost:9001

### Local Development

#### Backend

```bash
# Install Go dependencies
go mod download

# Run a chat server locally
cd cmd/chatserver
go run . 

# Or run with environment variables
SERVER_ID=dev-server-1 SERVER_PORT=8080 go run .
```

#### Frontend

```bash
cd web

# Install dependencies
npm install

# Start development server
npm run dev
```

Frontend runs at http://localhost:3000 with API proxy to backend.

## How E2EE Works

### Key Generation (On Registration)

1. **Identity Key Pair**: Ed25519 key pair for long-term identity
2. **Signed Pre-Key**: Rotated periodically, signed by identity key
3. **One-Time Pre-Keys**: Batch of 100 ephemeral keys for initial handshakes

### Establishing a Session (X3DH)

When Alice wants to message Bob for the first time:

1. Alice fetches Bob's public keys from the server
2. Alice generates an ephemeral key pair
3. Alice performs X3DH to derive a shared secret:
   - DH(Alice's Identity, Bob's Signed PreKey)
   - DH(Alice's Ephemeral, Bob's Identity)
   - DH(Alice's Ephemeral, Bob's Signed PreKey)
   - DH(Alice's Ephemeral, Bob's One-Time PreKey) [optional]
4. Alice derives root key and chain keys from shared secret

### Message Encryption (Double Ratchet)

Each message uses a unique key derived from the chain key:

```
Chain Key -> HKDF -> Message Key (used once)
                  -> Next Chain Key
```

When Bob responds, both parties perform a "ratchet step" to establish new chain keys, ensuring forward secrecy.

## Project Structure

```
SilentRelay/
  cmd/
    chatserver/          # Main server entry point
  internal/
    auth/                # JWT authentication
    config/              # Configuration management
    contacts/            # Privacy-preserving contact discovery
    db/                  # PostgreSQL data access
    handlers/            # HTTP/WebSocket handlers
    inbox/               # Offline message storage
    media/               # Presigned URL generation
    metrics/             # Prometheus metrics
    middleware/          # Auth middleware
    models/              # Data models
    privacy/             # Privacy settings management
    pubsub/              # Redis pub/sub
    queue/               # Async message processing
    reactions/           # Message reactions/replies
    registry/            # Consul service discovery
    secrets/             # Vault integration
    security/            # Security hardening
       hardening.go     # Secure memory, constant-time ops
       headers.go       # Security headers, CORS, WAF
       audit.go         # Security event logging
       session.go       # Device-bound sessions
       keytransparency.go # Key audit trail
       certpinning.go   # TLS certificate pinning
       intrusion.go     # Attack detection (IDS)
       honeypot.go      # Deception technology
       chaos.go         # Chaos engineering
       zerotrust.go     # Zero trust architecture
       hsm.go           # Hardware security module
       postquantum.go   # Post-quantum crypto
       supplychain.go   # SBOM, SLSA, Sigstore
    websocket/           # WebSocket hub and clients
  infrastructure/
    db/                  # Database migrations
    haproxy/             # Load balancer config
  web/                   # React frontend
    src/
        components/      # React components
        core/
            crypto/      # Signal Protocol implementation
            services/    # WebSocket service
            store/       # Zustand state management
  docker-compose.yml     # Container orchestration
  Dockerfile             # Go server container
  go.mod                 # Go dependencies
```

## Security Considerations

### What the Server Can See

- Message content: NO (encrypted ciphertext only)
- Media content: NO (encrypted before upload)
- Who is messaging whom: NO (conversation list is device-to-device synced)
- Chat requests/blocked users: NO (client-side only)
- Message timestamps: YES (for delivery routing only)
- User phone numbers and public keys: YES

### What the Server Cannot Do

- Decrypt any messages
- Read media attachments
- See your contact list or conversations
- Know who you are chatting with
- Forge messages from users
- Perform man-in-the-middle attacks (clients verify keys)

### Enterprise Security Hardening

1. TLS 1.3 with certificate pinning
2. Key Transparency Log (prevents key substitution)
3. Safety numbers (QR code and fingerprint verification)
4. Disappearing messages (auto-cleanup job)
5. Recovery key backup (24-word BIP39 mnemonic)
6. Audit logging (all security events)
7. Intrusion detection (real-time threat blocking)
8. Rate limiting (per-IP and per-user)
9. Session security (device binding, hourly rotation)
10. Input validation (SQL injection, XSS blocked)

### Production Checklist

See [docs/security/SECURITY.md](docs/security/SECURITY.md) for full security documentation.

### Red Team Ready

Additional hardening for enterprise security:

- **MITRE ATT&CK**: Mapped controls to 45+ techniques - see [docs/security/MITRE_ATTCK_MAPPING.md](docs/security/MITRE_ATTCK_MAPPING.md)
- **HSM Integration**: Server keys never leave hardware
- **Post-Quantum Ready**: Hybrid Kyber+X25519 preparation
- **Zero Trust**: Never trust, always verify
- **Honeypots**: Fake admin panels, credentials, endpoints
- **Canary Tokens**: Trackable fake secrets
- **Supply Chain Security**: SBOM, SLSA, Sigstore
- **Chaos Engineering**: Game day scenarios for security
- **Bug Bounty**: Responsible disclosure program

See [docs/security/REDTEAM_CHECKLIST.md](docs/security/REDTEAM_CHECKLIST.md) for the full attack resistance matrix.

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_ID` | Unique server identifier | `chat-server-1` |
| `SERVER_PORT` | HTTP/WebSocket port | `8080` |
| `REDIS_URL` | Redis connection string | `localhost:6379` |
| `POSTGRES_URL` | PostgreSQL connection string | See docker-compose |
| `CONSUL_URL` | Consul agent address | `localhost:8500` |
| `JWT_SECRET` | JWT signing secret | Change in production (minimum 32 characters) |
| `MINIO_URL` | MinIO server address | `localhost:9000` |

## API Documentation

For complete API documentation, see [docs/api/API_DOCUMENTATION_INDEX.md](docs/api/API_DOCUMENTATION_INDEX.md).

### Authentication

```
POST /api/v1/auth/request-code
POST /api/v1/auth/verify
POST /api/v1/auth/register
POST /api/v1/auth/refresh
```

### Users

```
GET  /api/v1/users/me
PUT  /api/v1/users/me
POST /api/v1/users/me/prekeys
GET  /api/v1/users/{userId}/keys
GET  /api/v1/users/search?q=...
```

### Messages

```
GET  /api/v1/messages
PUT  /api/v1/messages/{messageId}/status
```

### Groups

```
POST   /api/v1/groups
GET    /api/v1/groups/{groupId}
POST   /api/v1/groups/{groupId}/members
DELETE /api/v1/groups/{groupId}/members/{userId}
```

### WebSocket

```
GET /ws?token=<jwt>
```

## Key Rotation Feature

**Automatic JWT Secret Rotation with Zero-Downtime**

SilentRelay includes a comprehensive key rotation mechanism that:

- **Automatically rotates JWT secrets** on a configurable schedule (default: 24 hours)
- **Maintains zero-downtime** during rotation with dual-key support
- **Provides comprehensive logging** for all rotation events
- **Generates cryptographically secure secrets** (512-bit entropy)
- **Ensures thread-safe operations** throughout the rotation process

### Key Benefits

- **Enhanced Security**: Regular secret rotation reduces risk of key compromise
- **Compliance Ready**: Meets industry standards for key rotation (NIST, PCI DSS, ISO 27001)
- **Zero Downtime**: No service interruption during rotation
- **Backward Compatible**: Existing tokens continue to work during transition
- **Production Ready**: Comprehensive testing and documentation

### Configuration

```env
# Key Rotation Configuration
JWT_ROTATION_INTERVAL=24h  # Optional: Custom rotation interval (default: 24h)
```

For detailed implementation documentation, see [docs/security/KEY_ROTATION_IMPLEMENTATION.md](docs/security/KEY_ROTATION_IMPLEMENTATION.md).

## Contributing

This is a private project for family and friends, but feel free to fork for your own use.

## License

Private use only. All rights reserved.
