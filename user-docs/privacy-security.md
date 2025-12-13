# Privacy & Security Guide

SilentRelay is designed with privacy as the foundation. This guide explains how your data is protected and best practices for staying secure.

## How SilentRelay Protects Your Privacy

### End-to-End Encryption

**What it means**: Your messages are encrypted on your device and can only be decrypted by the intended recipient.

**How it works**:
- Messages are encrypted using the Signal Protocol
- Encryption keys never leave your device
- The server only sees encrypted data (ciphertext)
- Even SilentRelay operators cannot read your messages

**What the server sees**:
- Message content
- File contents
- Who you're messaging (with sealed sender)
- Your contact list
- Encrypted blobs
- Message timestamps (for delivery)
- Your public keys

### Server Visibility Comparison

```
Traditional Messaging:     SilentRelay:
┌─────────────────────┐   ┌─────────────────────┐
│ Server Sees:        │   │ Server Sees:       │
│ • Message content   │   │ • Encrypted blobs  │
│ • Who sent to whom  │   │ • Timestamps only  │
│ • Contact lists     │   │ • Public keys      │
│ • File contents     │   │                    │
└─────────────────────┘   └─────────────────────┘
```

*Figure 2: SilentRelay minimizes server visibility compared to traditional messaging*

### Signal Protocol

SilentRelay uses the same encryption protocol as Signal and WhatsApp:

- **X3DH Key Agreement**: Secure key exchange without transmitting secrets
- **Double Ratchet Algorithm**: Unique encryption keys for every message
- **Perfect Forward Secrecy**: Compromised keys don't expose past messages
- **Post-Compromise Security**: Automatic key rotation after breaches

### Sealed Sender

**What it is**: Hides metadata about who is messaging whom from the server.

**Benefits**:
- Server cannot build social graphs
- Protects against traffic analysis attacks
- Maintains privacy even from network observers

## Verifying Your Contacts

### Safety Numbers

Every contact has a unique "safety number" (fingerprint):

1. **View Safety Number**
   - Open a chat
   - Tap contact name → "Safety Number"
   - Or scan QR code for easier verification

2. **Verify In Person**
   - Compare numbers with your contact face-to-face
   - Or use another secure channel (not SilentRelay)

3. **What Changes Mean**
   - If numbers match: Conversation is secure
   - If numbers change: Possible interception attempt
   - Re-verify immediately if numbers don't match

### Key Transparency

SilentRelay uses a key transparency log to prevent key substitution:

- **Public Audit Trail**: All key changes are logged
- **Cryptographic Proofs**: Mathematical proof of key validity
- **Automatic Verification**: Background checks of contact keys

## Privacy Settings

### Read Receipts

**Send Read Receipts**: Let others know when you've read their messages
- More transparent communication
- Reveals when you're active

**Receive Read Receipts**: See when others read your messages
- Know if messages were received
- Others can see your activity patterns

**Recommendation**: Enable for trusted contacts, disable for sensitive conversations

### Last Seen

**Show Last Seen**: Display when you were last active
- Friends know when you're available
- Reveals your activity patterns

**Hide Last Seen**: Keep your activity private
- Maximum privacy
- Others can't see when you're online

### Disappearing Messages

**How it works**:
- Messages automatically delete after a set time
- Timer starts when message is read
- Works for text, media, and files

**Timer Options**: 5 seconds to 1 week

**Security Benefits**:
- Limits data exposure if device is compromised
- Reduces long-term storage risks
- Self-destructing evidence

**Limitations**:
- Screenshots can still be taken
- Recipients can forward messages before deletion
- Deleted messages cannot be recovered

## Device Security

### Linking Devices

**Secure Process**:
1. Generate device-specific keys
2. QR code scanning for secure pairing
3. Encrypted key transfer

**Best Practices**:
- Only link devices you physically control
- Use strong PINs on all devices
- Remove old devices immediately

### Device Management

**Monitor Linked Devices**:
- Settings → Devices → View all linked devices
- See last active time for each device
- Remove suspicious devices immediately

**Security Alerts**:
- Get notified of new device links
- Alerts for suspicious login attempts
- Immediate notification of security events

## Account Security

### PIN Protection

**Device PIN**: 4 or 6-digit code for quick access
- Protects your account on this device
- Required for sensitive operations
- Can be changed in Settings → Security

### Recovery Key

**24-Word Mnemonic**:
- BIP39 standard for secure backup
- Can restore your account on new devices
- Never share with anyone
- Store offline in secure location

**When You Need It**:
- Lost all your devices
- Factory reset without backup
- Switching to new phone/computer

### Password Best Practices

- Use unique, strong PINs
- Never reuse PINs from other services
- Change PINs regularly
- Use biometric unlock when available

## Data Management

### Export Your Data

**What You Can Export**:
- All messages and media
- Contact list
- Account settings
- Device information

**How to Export**:
1. Settings → Privacy → Export Data
2. Choose date range
3. Download encrypted archive
4. Decrypt with your recovery key

### Delete Your Account

**Permanent Deletion**:
- Removes all messages and account data
- Cannot be undone
- Requires confirmation with recovery key

**Before Deleting**:
- Export important data
- Inform contacts of account deletion
- Remove linked devices

## Security Best Practices

### General Advice

1. **Verify Contacts**: Always check safety numbers for important conversations
2. **Use Disappearing Messages**: For sensitive communications
3. **Keep Software Updated**: Security improvements are released regularly
4. **Secure Your Devices**: Use strong PINs and biometric locks
5. **Be Suspicious**: If something seems off, verify safety numbers

### Avoiding Common Threats

- **Phishing**: Never click suspicious links in messages
- **Social Engineering**: Verify identities through other channels
- **Malware**: Only download files from trusted sources
- **Shoulder Surfing**: Be aware of people watching your screen
- **Unattended Devices**: Always lock your device when away

### Emergency Situations

**If Your Device is Lost/Stolen**:
1. Change your PIN immediately from another device
2. Remove the lost device from Settings → Devices
3. Enable disappearing messages for all chats
4. Monitor for suspicious activity

**If You Suspect Compromise**:
1. Re-verify safety numbers with all contacts
2. Change PIN and review linked devices
3. Export and backup important data
4. Consider account reset if necessary

## Advanced Security Features

### Key Rotation

- **Automatic**: Keys rotate every 24 hours
- **Manual**: Force rotation in Settings → Security
- **Zero Downtime**: No interruption during rotation

### Audit Logging

- **Security Events**: All security-related actions logged
- **No Content Logging**: Message content never logged
- **Compliance Ready**: Meets regulatory requirements

### Intrusion Detection

- **Real-time Monitoring**: Automatic threat detection
- **Rate Limiting**: Prevents brute force attacks
- **Anomaly Detection**: Unusual activity flagged

## Understanding Risks

### What SilentRelay Cannot Protect Against

- **Physical Access**: If someone has your unlocked device
- **Keylogger Malware**: Captures what you type
- **Screen Capture**: Screenshots bypass encryption
- **Social Engineering**: Tricking you into revealing information
- **Legal Compulsion**: Court orders (but we have nothing to give)

### What SilentRelay Does Protect Against

- **Server Breaches**: Encrypted data is useless to attackers
- **Network Interception**: MITM attacks blocked by certificate pinning
- **Metadata Analysis**: Sealed sender hides communication patterns
- **Key Compromise**: Forward secrecy limits damage
- **Mass Surveillance**: No backdoors or weak encryption

## Staying Informed

### Security Updates

- Follow our security blog for updates
- Enable security notifications in Settings
- Review changelog for new security features

### Community Resources

- Security documentation: [docs/SECURITY.md](../docs/SECURITY.md)
- Threat model: [docs/THREAT_MODEL.md](../docs/THREAT_MODEL.md)
- Bug bounty program: [docs/BUG_BOUNTY.md](../docs/BUG_BOUNTY.md)

Remember: Security is a process, not a product. Stay vigilant and follow best practices to keep your communications private.