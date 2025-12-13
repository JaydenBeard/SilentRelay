# SilentRelay User Documentation

Welcome to SilentRelay, a private messaging service built on military-grade encryption. This documentation is designed for end users who want to understand how to use SilentRelay effectively and securely.

## Documentation Overview

### For New Users
- **[Getting Started](getting-started.md)** - Create your account and send your first message
- **[User Manual](user-manual.md)** - Complete guide to all features and settings

### Security & Privacy
- **[Privacy & Security Guide](privacy-security.md)** - Understand how your data is protected
- **[Device Management](device-management.md)** - Manage your linked devices and security

### Troubleshooting
- **[FAQ](faq.md)** - Answers to common questions
- **[Troubleshooting](troubleshooting.md)** - Solutions to common issues

### Advanced Topics
- **[Groups](groups.md)** - Create and manage group conversations
- **[File Sharing](file-sharing.md)** - Send and receive encrypted files
- **[Voice & Video Calls](calls.md)** - Make secure calls

## Key Features

SilentRelay provides end-to-end encrypted messaging with these key features:

- **End-to-End Encryption**: Messages encrypted on your device—servers can't read them
- **Signal Protocol**: Industry-standard encryption used by Signal and WhatsApp
- **Perfect Forward Secrecy**: Compromised keys don't expose past messages
- **Sealed Sender**: Server can't see who's messaging whom
- **Self-Destructing Messages**: Auto-delete messages after a set time
- **Device Verification**: QR codes and safety numbers to verify contacts
- **Open Source**: Both client and server code are publicly auditable

### How Encryption Works

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ Your Device │────▶│   Server    │────▶│Recipient   │
│             │     │(Ciphertext  │     │ Device     │
│  Alice's    │     │ Only)       │     │            │
│  Private    │     │             │     │  Bob's     │
│  Key        │     │             │     │  Private   │
└─────────────┘     └─────────────┘     │  Key       │
                                        └─────────────┘
```

*Figure 1: End-to-end encryption prevents server access to message content*

## Getting Help

If you can't find what you're looking for:

1. Check the [FAQ](faq.md) for common questions
2. Review the [Troubleshooting](troubleshooting.md) guide
3. Contact support through the app (Settings → Help & Support)

## Security Notice

Your privacy is our priority. SilentRelay is designed so that even we cannot access your messages. For security best practices, see our [Privacy & Security Guide](privacy-security.md).

---

*This documentation is for SilentRelay users. For developer documentation, see [docs/README.md](../docs/README.md).*