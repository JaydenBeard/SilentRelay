# Cryptographic Implementation Details

This document provides comprehensive details about the cryptographic algorithms, protocols, and implementations used in the SilentRelay application.

## Overview

The SilentRelay implements multiple layers of cryptography to ensure end-to-end security, authentication, and data protection. All implementations follow industry best practices and are designed for production security.

## End-to-End Encryption (E2EE)

### Signal Protocol Implementation

**Protocol**: Custom Signal Protocol implementation
**Location**: `internal/security/signal.go`
**Status**: Production Ready

#### Key Exchange (X3DH)
```go
// X25519 key exchange with HKDF key derivation
type X3DHKeyBundle struct {
    IdentityKey     [32]byte  // Long-term identity key
    SignedPreKey    [32]byte  // Medium-term signed pre-key
    SignedPreKeyID  uint32    // Pre-key identifier
    SignedPreKeySig []byte    // Ed25519 signature
    OneTimePreKey   *[32]byte // Optional one-time pre-key
    OneTimePreKeyID *uint32   // One-time pre-key identifier
}
```

**Algorithm Details**:
- **Key Exchange**: X25519 (Curve25519)
- **Signature**: Ed25519
- **Hash Function**: SHA-256
- **Key Derivation**: HKDF-SHA256

#### Double Ratchet Algorithm
```go
type DoubleRatchetState struct {
    RootKey        [32]byte  // Root key for key derivation
    ChainKeySend   [32]byte  // Sending chain key
    ChainKeyRecv   [32]byte  // Receiving chain key
    MessageKeySend [32]byte  // Current sending message key
    MessageKeyRecv [32]byte  // Current receiving message key
    SendRatchet    KeyPair   // Current sending ratchet key
    RecvRatchet    [32]byte  // Current receiving ratchet public key
    PrevChainLen   uint32    // Previous chain length
    SendCount      uint32    // Messages sent in current chain
    RecvCount      uint32    // Messages received in current chain
}
```

**Security Properties**:
- **Forward Secrecy**: Compromised keys don't expose past messages
- **Post-Compromise Security**: New keys generated after compromise
- **Replay Protection**: Message counters prevent replay attacks

#### Symmetric Encryption
```go
// AES-256-GCM for message encryption
func (sp *SignalProtocol) EncryptAESGCM(plaintext, key []byte) ([]byte, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }

    ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
    return ciphertext, nil
}
```

**Parameters**:
- **Algorithm**: AES-256-GCM
- **Key Size**: 256 bits
- **Nonce Size**: 96 bits (GCM standard)
- **Authentication Tag**: 128 bits

## Authentication & Authorization

### JWT Token Security

**Implementation**: `internal/auth/auth.go`
**Algorithm**: HMAC-SHA256
**Key Size**: 256 bits minimum

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
- **Short Expiration**: Access tokens expire in 1 hour
- **Secure Claims**: Includes device fingerprinting
- **Replay Protection**: Unique JTI per token

### PIN-Based Security

**Implementation**: `internal/security/pin.go`
**Algorithm**: Argon2id

**Parameters**:
```go
type Argon2Params struct {
    Time    uint32 // 3 iterations
    Memory  uint32 // 65536 KiB
    Threads uint32 // 4 parallel threads
    KeyLen  uint32 // 32 bytes
}
```

**Security Properties**:
- **Memory-Hard**: Resistant to GPU/ASIC attacks
- **Configurable**: Parameters adjustable for security/performance balance
- **Salt Usage**: Unique salt per PIN hash

## Server-Side Security

### Database Encryption

**Implementation**: AES-256-GCM for stored messages
**Location**: `internal/db/postgres.go`

**Encryption Flow**:
```go
// Messages encrypted before database storage
encryptedMessage := encryptMessage(message, encryptionKey)
db.SaveMessage(encryptedMessage)

// Messages decrypted after retrieval
decryptedMessage := decryptMessage(encryptedMessage, encryptionKey)
```

### Key Management

**Implementation**: `internal/security/keyrotation.go`
**Features**:
- Automatic JWT secret rotation
- Configurable rotation intervals
- Zero-downtime key transitions
- Secure key distribution

## Post-Quantum Preparation

**Implementation**: `internal/security/postquantum.go`
**Status**: Research/Prototype

**Supported Algorithms**:
- **KEM**: Kyber512, Kyber768, Kyber1024
- **Signatures**: Dilithium2, Dilithium3, Dilithium5
- **Hybrid Mode**: Classical + Post-Quantum combinations

**Migration Phases**:
1. **Classical Only**: Current production state
2. **Hybrid Mode**: Classical + PQ algorithms
3. **PQ Preferred**: PQ primary, classical fallback
4. **PQ Only**: Pure post-quantum cryptography

## Hardware Security Module (HSM) Interface

**Implementation**: `internal/security/hsm.go`
**Status**: Interface Defined, Not Implemented

**Supported Operations**:
- Key generation and storage
- Digital signatures
- Key wrapping/unwrapping
- Cryptographic acceleration

**Provider Support**:
- AWS CloudHSM
- Software HSM (development only)
- PKCS#11 interface

## Cryptographic Security Analysis

### Algorithm Security Levels

| Algorithm | Security Level | Status | Notes |
|-----------|----------------|--------|-------|
| AES-256-GCM | 256-bit | Active | Industry standard |
| X25519 | 128-bit | Active | Curve25519 key exchange |
| Ed25519 | 128-bit | Active | Edwards-curve signatures |
| HMAC-SHA256 | 256-bit | Active | JWT authentication |
| Argon2id | Configurable | Active | Password hashing |
| Kyber768 | 128-bit quantum | Planned | Post-quantum KEM |
| Dilithium3 | 128-bit quantum | Planned | Post-quantum signatures |

### Key Sizes and Parameters

| Component | Key Size | Purpose | Rotation |
|-----------|----------|---------|----------|
| JWT Secret | 256+ bits | Token signing | 90 days |
| Message Keys | 256 bits | Content encryption | Per message |
| Identity Keys | 256 bits | User identification | Never |
| Pre-Keys | 256 bits | Session establishment | Weekly |
| PIN Hashes | 256 bits | Authentication | On change |

### Performance Characteristics

| Operation | Algorithm | Performance | Notes |
|-----------|-----------|-------------|-------|
| Key Exchange | X25519 | ~10μs | Fast, constant time |
| Message Encryption | AES-256-GCM | ~1μs per KB | Hardware accelerated |
| Signature | Ed25519 | ~50μs | Fast verification |
| PIN Hashing | Argon2id | ~100ms | Memory-hard, slow by design |
| JWT Signing | HMAC-SHA256 | ~1μs | Very fast |

## Cryptographic Testing

### Unit Tests

**Coverage Areas**:
- Algorithm correctness
- Key generation and validation
- Encryption/decryption round trips
- Signature verification
- Key derivation functions

**Test Vectors**:
```go
// Example test for X25519 key exchange
func TestX25519KeyExchange(t *testing.T) {
    alicePriv, alicePub := generateKeyPair()
    bobPriv, bobPub := generateKeyPair()

    aliceShared := sharedSecret(alicePriv, bobPub)
    bobShared := sharedSecret(bobPriv, alicePub)

    assert.Equal(t, aliceShared, bobShared)
}
```

### Integration Tests

**End-to-End Encryption**:
- Message encryption/decryption flows
- Key exchange protocols
- Multi-device synchronization
- Group message encryption

### Security Audits

**Regular Assessments**:
- Algorithm implementation review
- Side-channel attack analysis
- Cryptographic parameter validation
- Compliance with standards

## Standards Compliance

### Implemented Standards

| Standard | Components | Status |
|----------|------------|--------|
| RFC 7748 | X25519 key exchange | Compliant |
| RFC 8032 | Ed25519 signatures | Compliant |
| NIST SP 800-38D | AES-GCM | Compliant |
| RFC 5869 | HKDF | Compliant |
| RFC 9106 | Argon2id | Compliant |

### Security Best Practices

- **Key Separation**: Different keys for different purposes
- **Perfect Forward Secrecy**: Ephemeral keys for each session
- **Replay Protection**: Nonces and counters
- **Secure Random**: Cryptographically secure random generation
- **Constant Time**: Timing attack resistance

## Future Cryptographic Enhancements

### Planned Improvements

1. **Post-Quantum Migration**
   - Hybrid cryptography implementation
   - Algorithm negotiation protocols
   - Performance optimization

2. **Hardware Security**
   - HSM integration completion
   - Secure enclave support
   - TPM integration

3. **Advanced Protocols**
   - MLS (Message Layer Security)
   - PAKE (Password Authenticated Key Exchange)
   - Threshold cryptography

## Security Contact

**Cryptography Team**: `crypto@silentrelay.com.au`
**Security Issues**: `security@silentrelay.com.au`
**PGP Key**: Available at `https://silentrelay.com.au/pgp`

---

*© 2025 SilentRelay. All rights reserved.*
