# Recovery Key + Server Backup Feature

> **Status**: Future Enhancement  
> **Priority**: Medium  
> **Created**: 2025-12-16

## Problem Statement

Currently, the recovery key shown during onboarding is **generated but never used**. Users write it down but it serves no actual purpose. Additionally, when logging into a new device, users must "Start Fresh" and lose access to all previous messages.

## Proposed Solution

Use the recovery key to enable **encrypted cross-device message recovery**:

1. **Encrypt the master key** with the recovery key
2. **Store encrypted blob on server** (SilentRelay can't decrypt it)
3. **New device recovery**: Enter recovery key → decrypt master key → access messages

## Key Hierarchy

```
Recovery Key (24 words - user writes down)
    │
    ▼ PBKDF2 derivation
Recovery Master Key
    │
    ▼ AES-256-GCM encrypt
┌─────────────────────────────┐
│ Encrypted Key Blob          │ ──► Stored on server
│ - Master key (encrypted)    │
│ - Salt                      │
│ - IV                        │
└─────────────────────────────┘
```

## User Flows

### New User Onboarding

1. Set PIN → creates master key
2. Generate recovery key (24 words)
3. **Encrypt master key with recovery key**
4. **Upload encrypted blob to server**
5. User writes down recovery key

### Existing User - New Device (with recovery key)

1. Login with phone
2. See "New Device" screen
3. Choose "Recover with Recovery Key" (new option)
4. Enter 24 words
5. **Download encrypted blob from server**
6. **Decrypt master key with recovery key**
7. Set new PIN (re-encrypt master key locally)
8. Access all messages

### Existing User - New Device (without recovery key)

1. Login with phone
2. See "New Device" screen  
3. Choose "Start Fresh" (existing flow)
4. Create new encryption keys
5. No access to old messages

## Technical Components

### Backend API

- `POST /api/v1/keys/backup` - Upload encrypted key blob
- `GET /api/v1/keys/backup` - Download encrypted key blob
- `DELETE /api/v1/keys/backup` - Delete backup (when user creates new keys)

### Crypto Functions

- `deriveKeyFromRecoveryWords(words: string[]): CryptoKey`
- `encryptMasterKey(masterKey, recoveryKey): EncryptedBlob`
- `decryptMasterKey(blob, recoveryKey): MasterKey`

### Message Backup (Optional - Phase 2)

- Encrypt message history with master key
- Store encrypted messages on server
- Sync to new device after recovery

## Security Considerations

1. **Server never sees plaintext keys** - Only encrypted blob stored
2. **Recovery key is the root of trust** - If lost, no recovery possible
3. **Rate limiting** on recovery attempts - Prevent brute force
4. **Blob versioning** - Handle key rotation gracefully
5. **Optional cloud backup toggle** - User choice

## Implementation Phases

### Phase 1: Recovery Key Infrastructure

- Generate recovery key from BIP39 wordlist ✅ (already done)
- Derive encryption key from recovery words
- Encrypt/upload master key to server
- Download/decrypt on new device

### Phase 2: Message Backup

- Encrypt local message DB with master key
- Upload encrypted backup periodically
- Download + decrypt on new device

### Phase 3: iOS Implementation

- Port recovery key flow to iOS
- iCloud Keychain integration (optional)

## Effort Estimate

- Phase 1: 2-3 days
- Phase 2: 3-4 days  
- Phase 3: 2-3 days

## References

- Signal's SVR (Secure Value Recovery)
- Apple's iCloud Keychain backup
- WhatsApp's encrypted backup (uses 64-digit key)
