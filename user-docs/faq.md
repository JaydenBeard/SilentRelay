# Frequently Asked Questions

Find answers to common questions about SilentRelay.

## Getting Started

### What is SilentRelay?

SilentRelay is a private messaging service that uses end-to-end encryption to protect your conversations. Unlike other messaging apps, SilentRelay is designed so that even the service operators cannot read your messages.

### How is SilentRelay different from Signal/WhatsApp?

- **Self-hostable**: You can run your own SilentRelay server
- **Open source server**: Both client and server code are auditable
- **No phone number required**: Username-based system
- **Advanced privacy features**: Sealed sender, key transparency
- **Enterprise features**: Audit logging, device management

### Is SilentRelay free?

Yes, SilentRelay is free for personal use. Enterprise deployments may have licensing fees for support and advanced features.

### Do I need a phone number?

No, SilentRelay uses usernames instead of phone numbers. This provides better privacy and works internationally without SMS requirements.

## Security & Privacy

### Can SilentRelay read my messages?

No. Messages are encrypted on your device using the Signal Protocol. The encryption keys never leave your device, so SilentRelay cannot decrypt your messages even if compelled by law.

### What does the server see?

The server sees:
- Encrypted message content (ciphertext)
- Message timestamps
- Your public encryption keys
- Usernames and device information

The server does NOT see:
- Message content
- Who you're messaging (with sealed sender)
- Your contact list
- File contents

### How do I verify a contact is real?

Use safety numbers:
1. Open a chat
2. Tap the contact's name
3. View "Safety Number"
4. Compare numbers in person or via another secure channel

If numbers don't match, someone may be intercepting your communication.

### What happens if my device is stolen?

1. Change your PIN immediately from another device
2. Remove the stolen device in Settings → Devices
3. Re-verify safety numbers with all contacts
4. Consider enabling disappearing messages

### Are calls encrypted?

Yes, voice and video calls use WebRTC with end-to-end encryption. Call content cannot be intercepted by the server or network observers.

## Features & Functionality

### How do I start a new conversation?

1. Click "Start New Chat" or the "+" button
2. Type a username (minimum 3 characters)
3. Select the user from search results
4. Start messaging

### Can I create groups?

Yes, SilentRelay supports group messaging with the same encryption as 1-on-1 chats. Groups can have up to 100 members.

### How do disappearing messages work?

- Set a timer (5 seconds to 1 week) in chat settings
- Messages automatically delete after the timer expires
- Timer starts when the message is read
- Works for text, images, and files

### Can I send files?

Yes, you can send files up to 100MB. Supported formats include images, documents, audio, and video files. All files are encrypted before upload.

### What about notifications?

SilentRelay supports push notifications that can be customized:
- Enable/disable all notifications
- Mute specific chats
- Set quiet hours
- Control notification previews

## Account & Settings

### How do I change my username?

Usernames cannot be changed for security reasons. If you need a different username, you'll need to create a new account.

### What is a recovery key?

A 24-word mnemonic phrase (BIP39 standard) that can restore your account on new devices. Store it securely offline - never share it.

### How do I link multiple devices?

1. Go to Settings → Devices
2. Click "Link New Device"
3. Scan the QR code with your new device
4. The device will sync your chats and encryption keys

### Can I export my data?

Yes, go to Settings → Privacy → Export Data. This downloads all your messages, media, and account information in an encrypted format.

## Technical Questions

### What browsers are supported?

SilentRelay works on:
- Chrome/Chromium 90+
- Firefox 88+
- Safari 14+
- Edge 90+

Mobile browsers are supported but the mobile app provides a better experience.

### Does SilentRelay work offline?

- You can read existing messages offline
- New messages will be sent when you reconnect
- File uploads resume automatically
- Some features require internet connection

### How much storage do I get?

Storage limits depend on your server configuration. Contact your administrator for specific limits. Files are stored encrypted on the server.

### Can I use SilentRelay on multiple devices simultaneously?

Yes, you can link up to 5 devices. All devices share the same encryption keys and message history.

## Troubleshooting

### Messages aren't sending

**Check**:
- Internet connection
- Server status (check with administrator)
- Device storage (clear cache if full)
- Try refreshing the page

### Can't find a contact

**Possible issues**:
- Username spelling (case-sensitive)
- User doesn't exist
- User has blocked you
- Search requires minimum 3 characters

### Notifications not working

**Check**:
- Browser permissions for notifications
- SilentRelay notification settings
- Do Not Disturb mode
- Chat-specific mute settings

### App is slow or laggy

**Try**:
- Clear browser cache
- Close other tabs/applications
- Check internet speed
- Restart browser
- Update browser to latest version

### Safety numbers changed unexpectedly

**This is serious** - it may indicate:
- Someone intercepted your communication
- One of you reinstalled the app
- Device compromise

Re-verify safety numbers immediately through another secure channel.

## Advanced Features

### What is sealed sender?

A privacy feature that hides metadata about who is messaging whom from the server. This prevents social graph analysis and traffic correlation attacks.

### How does key transparency work?

SilentRelay maintains a public log of all encryption key changes. This prevents "key substitution" attacks where someone tries to impersonate a user by changing their keys.

### What is perfect forward secrecy?

Each message uses a unique encryption key. If any key is compromised, it cannot decrypt past or future messages. SilentRelay achieves this through the Double Ratchet algorithm.

### Can I self-host SilentRelay?

Yes, SilentRelay is designed for self-hosting. See the [deployment guide](../DEPLOY.md) for instructions.

## Support & Contact

### How do I get help?

1. Check this FAQ
2. Review the [Troubleshooting Guide](troubleshooting.md)
3. Search existing [GitHub Issues](https://github.com/JaydenBeard/end2endsecure.com/issues)
4. Contact your server administrator
5. Open a new issue if you found a bug

### Where can I report security issues?

Use our [Bug Bounty Program](../docs/BUG_BOUNTY.md) for security research. Do not report security vulnerabilities publicly.

### How do I contribute?

SilentRelay is open source! See our [Contributing Guide](../CONTRIBUTING.md) to get started.

---

*This FAQ is regularly updated. Last updated: December 2025*