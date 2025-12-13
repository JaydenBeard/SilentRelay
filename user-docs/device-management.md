# Device Management

Learn how to manage your linked devices and maintain security across all your SilentRelay installations.

## Understanding Device Linking

### What is Device Linking?

Device linking allows you to use SilentRelay on multiple devices simultaneously. All devices share:

- Your message history
- Encryption keys
- Contact list
- Account settings

### Security Implications

**Benefits**:
- Access messages from any device
- Seamless switching between devices
- Automatic synchronization

**Risks**:
- Lost/stolen device can access your messages
- All devices must be secured individually
- Compromise of one device affects all

## Linking a New Device

### Step-by-Step Process

1. **On your current device**:
   - Go to Settings → Devices
   - Click "Link New Device"
   - A QR code will be displayed

2. **On the new device**:
   - Install SilentRelay (web or app)
   - During setup, choose "Link Existing Device"
   - Scan the QR code from your current device
   - Enter your PIN to confirm

3. **Verification**:
   - The new device will appear in your device list
   - Safety numbers remain the same across devices
   - Message history will sync automatically

### Supported Device Types

- **Desktop browsers**: Chrome, Firefox, Safari, Edge
- **Mobile browsers**: iOS Safari, Android Chrome
- **Mobile apps**: iOS and Android (when available)
- **Tablets**: iPad, Android tablets

### Device Limits

- Maximum 5 devices per account
- Remove old devices to add new ones
- No limit on device types (mix desktop/mobile)

## Managing Linked Devices

### Viewing Your Devices

**Device List Shows**:
- Device name (customizable)
- Device type (desktop/mobile)
- Last active time
- Location (approximate, if enabled)

### Device Names

**Setting Names**:
1. Go to Settings → Devices
2. Click on a device
3. Enter a descriptive name (e.g., "John's iPhone", "Work Laptop")

**Best Practices**:
- Use clear, identifiable names
- Include location or purpose
- Update when device ownership changes

### Device Activity

**Monitoring Activity**:
- See when each device was last used
- Check for suspicious login times/locations
- Get alerts for new device links

## Removing Devices

### When to Remove Devices

- **Lost/Stolen**: Immediately remove to prevent access
- **Sold/Given Away**: Remove before transferring ownership
- **No Longer Used**: Clean up unused devices
- **Suspicious Activity**: Remove potentially compromised devices

### How to Remove a Device

1. **From any linked device**:
   - Settings → Devices
   - Find the device to remove
   - Click "Remove Device"
   - Confirm the action

2. **From the device itself** (if accessible):
   - Settings → Devices
   - Select "Unlink This Device"
   - Confirm logout

### Effects of Removal

**What happens**:
- Device can no longer access your account
- Messages stop syncing to that device
- Device data remains until cache is cleared
- Safety numbers unchanged for other devices

**What doesn't happen**:
- Message history isn't deleted from server
- Other devices unaffected
- Contact list preserved

## Security Best Practices

### Device Security

**For Each Device**:
- Use strong, unique PINs
- Enable biometric authentication when available
- Keep devices updated
- Use antivirus/anti-malware software
- Be cautious with public WiFi

### Account Security

**Additional Measures**:
- Enable two-factor authentication (if available)
- Regularly review linked devices
- Change PIN periodically
- Use recovery key for backup access

### Physical Security

**Protect Your Devices**:
- Never leave devices unattended
- Use screen locks with short timeouts
- Encrypt device storage
- Be aware of shoulder surfing

## Troubleshooting Device Issues

### Device Won't Link

**Common Issues**:
- **QR Code Expired**: Codes expire after 5 minutes, generate new one
- **Network Issues**: Ensure both devices have internet
- **Browser Issues**: Try different browser or incognito mode
- **PIN Incorrect**: Verify PIN is correct

**Solutions**:
1. Check internet connection on both devices
2. Try refreshing the page
3. Clear browser cache
4. Contact support if persistent

### Device Shows as Offline

**Possible Causes**:
- Device actually offline
- Network connectivity issues
- App/browser closed
- Battery optimization (mobile)

**Verification**:
- Check device status in device list
- Last seen time indicates last activity
- May take a few minutes to update

### Duplicate Messages

**After Device Linking**:
- Messages may appear duplicated during sync
- This is normal and resolves automatically
- Refresh the page to force sync

### Messages Not Syncing

**Troubleshooting**:
1. Check device is online
2. Verify device is still linked
3. Try unlinking and re-linking
4. Clear app/browser cache
5. Check server status

## Advanced Device Features

### Device-Specific Settings

**Per-Device Preferences**:
- Notification settings
- Theme preferences
- Language settings
- Keyboard shortcuts (desktop)

### Device Fingerprinting

**Security Feature**:
- Each device has unique fingerprint
- Prevents unauthorized access
- Tracks device characteristics
- Alerts on fingerprint changes

### Session Management

**Active Sessions**:
- View all active login sessions
- Force logout from specific sessions
- See session start times
- Monitor for suspicious activity

## Migration and Backup

### Switching Devices

**Seamless Migration**:
1. Link new device
2. Verify sync completion
3. Test functionality on new device
4. Remove old device when satisfied

### Account Recovery

**Using Recovery Key**:
- 24-word mnemonic phrase
- Restores access on new device
- Bypasses device linking process
- Requires secure storage

### Data Export

**Device-Specific Data**:
- Export includes device information
- Shows linking history
- Includes security events
- Useful for audits

## Enterprise Device Management

### For Organizations

**Admin Controls** (if enabled):
- Centralized device management
- Mandatory device naming
- Automatic device removal policies
- Security policy enforcement

### Compliance Features

**Audit Capabilities**:
- Device linking logs
- Security event tracking
- Compliance reporting
- Data retention policies

## Security Alerts

### Types of Alerts

**Device-Related Alerts**:
- New device linked
- Device removed
- Suspicious login attempts
- Multiple failed PIN attempts

### Responding to Alerts

**Immediate Actions**:
1. Verify the activity is legitimate
2. Change PIN if suspicious
3. Remove unknown devices
4. Review recent activity

**Investigation**:
- Check device locations
- Verify login times
- Contact affected users
- Document incident

## Best Practices Summary

### Daily Habits
- Review device list weekly
- Remove unused devices promptly
- Keep recovery key secure and accessible

### Security Checklist
- [ ] Strong PINs on all devices
- [ ] Biometric authentication enabled
- [ ] Devices kept updated
- [ ] Suspicious activity monitored
- [ ] Recovery key backed up securely

### Emergency Procedures
- **Lost Device**: Remove immediately, change PIN
- **Suspicious Activity**: Review all devices, re-verify contacts
- **Account Compromise**: Use recovery key to regain control

Remember: Your devices are the gateway to your encrypted communications. Keep them secure to maintain your privacy.