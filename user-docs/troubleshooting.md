# Troubleshooting Guide

Solutions to common issues with SilentRelay.

## Connection Issues

### Can't connect to SilentRelay

**Symptoms**: "Connection failed" or "Server unreachable"

**Solutions**:
1. **Check internet connection**
   - Try loading other websites
   - Switch between WiFi and mobile data

2. **Server status**
   - Check if the server is down (ask administrator)
   - Try accessing from a different network

3. **Browser issues**
   - Clear browser cache and cookies
   - Try a different browser
   - Disable browser extensions temporarily

4. **Firewall/VPN**
   - Disable VPN if using one
   - Check corporate firewall settings
   - Try accessing from a different network

### WebSocket connection fails

**Symptoms**: Messages not sending, no real-time updates

**Solutions**:
1. **Check browser console** (F12 → Console)
   - Look for WebSocket errors
   - Note any error messages

2. **Network restrictions**
   - WebSockets may be blocked by corporate firewalls
   - Try HTTPS-only mode if available

3. **Browser compatibility**
   - Ensure you're using a supported browser
   - Update browser to latest version

## Message Issues

### Messages not sending

**Symptoms**: Messages stuck on "sending" or fail to send

**Solutions**:
1. **Check connection**
   - Ensure stable internet connection
   - Try refreshing the page

2. **Message size**
   - Large messages may fail - try breaking into smaller parts
   - Check file size limits (100MB max)

3. **Server issues**
   - Check server status with administrator
   - Try again later if server is overloaded

4. **Clear cache**
   - Settings → Privacy → Clear Cache
   - Restart browser completely

### Messages not received

**Symptoms**: Others say they sent messages but you didn't receive them

**Solutions**:
1. **Check spam/blocked**
   - Verify the contact isn't blocked
   - Check if you blocked them accidentally

2. **Device offline**
   - Messages are delivered when you come online
   - Check message status indicators

3. **Server issues**
   - Contact administrator if widespread issue
   - Check server maintenance schedule

### Duplicate messages

**Symptoms**: Receiving the same message multiple times

**Causes**:
- Network connectivity issues
- Browser tab refresh
- Server synchronization problems

**Solutions**:
- Refresh the page to sync
- Clear browser cache
- Contact support if persistent

## File Upload/Download Issues

### Files won't upload

**Symptoms**: File upload fails or gets stuck

**Solutions**:
1. **File size**
   - Check file size (100MB limit)
   - Compress large files

2. **File type**
   - Verify supported format
   - Try converting to different format

3. **Connection**
   - Ensure stable upload connection
   - Try smaller files first

4. **Storage quota**
   - Check if you've exceeded storage limits
   - Contact administrator

### Files won't download

**Symptoms**: Download fails or corrupted files

**Solutions**:
1. **Try again**
   - Downloads can resume automatically
   - Clear download cache

2. **Browser settings**
   - Check download folder permissions
   - Disable download managers/extensions

3. **File corruption**
   - Request sender to re-upload
   - Check file integrity if possible

## Authentication Issues

### Can't log in

**Symptoms**: Login fails with error message

**Solutions**:
1. **Check credentials**
   - Verify username spelling (case-sensitive)
   - Reset PIN if forgotten

2. **Account status**
   - Check if account is locked due to failed attempts
   - Contact administrator if suspended

3. **Device issues**
   - Try logging in from different device
   - Clear browser data

4. **Two-factor authentication**
   - Check if 2FA code is required
   - Verify authenticator app time sync

### PIN forgotten

**Solutions**:
1. **Recovery key**
   - Use 24-word recovery phrase
   - Settings → Security → Recover Account

2. **Device reset**
   - Reset device to regain access
   - Will need recovery key

3. **Contact administrator**
   - For enterprise deployments
   - May require identity verification

### Session expired

**Symptoms**: Forced logout, need to log in again

**Solutions**:
1. **Normal behavior**
   - Sessions expire for security (hourly rotation)
   - Re-enter PIN to continue

2. **Security issue**
   - If unexpected, change PIN immediately
   - Check linked devices for suspicious activity

## Contact Issues

### Can't find user

**Symptoms**: User search returns no results

**Solutions**:
1. **Username spelling**
   - Check for typos (case-sensitive)
   - Minimum 3 characters required

2. **User existence**
   - Verify user has an account
   - Check if user blocked you

3. **Server issues**
   - Try search again later
   - Contact administrator

### Contact verification fails

**Symptoms**: Safety numbers don't match

**This is a security issue**:

1. **Don't proceed**
   - Stop messaging immediately
   - Alert the contact through another channel

2. **Verify identity**
   - Meet in person if possible
   - Use phone call or video call
   - Compare safety numbers verbally

3. **If compromised**
   - Delete the conversation
   - Block the contact
   - Change your PIN
   - Re-verify all important contacts

## Notification Issues

### No push notifications

**Symptoms**: Not receiving message alerts

**Solutions**:
1. **Browser permissions**
   - Check notification permissions in browser settings
   - Allow notifications for SilentRelay

2. **App settings**
   - Settings → Notifications → Enable push notifications
   - Check "Do Not Disturb" mode

3. **Per-chat settings**
   - Verify chat isn't muted
   - Check notification preferences

4. **Browser issues**
   - Try different browser
   - Clear browser cache

### Notifications not working on mobile

**Symptoms**: Desktop notifications work but mobile doesn't

**Solutions**:
1. **App vs browser**
   - Use the mobile app instead of browser
   - Browser notifications may be limited

2. **Battery optimization**
   - Disable battery optimization for SilentRelay
   - Allow background activity

3. **Focus mode**
   - Check if "Do Not Disturb" is active
   - Verify notification channels

## Performance Issues

### App is slow

**Symptoms**: Lag, slow message loading, delayed responses

**Solutions**:
1. **Browser optimization**
   - Close unnecessary tabs
   - Clear browser cache and cookies
   - Update browser to latest version

2. **Device resources**
   - Close other applications
   - Restart device
   - Check available RAM/storage

3. **Network issues**
   - Test internet speed
   - Try different network
   - Check for VPN interference

4. **Large conversations**
   - Very long chats may load slowly
   - Consider archiving old conversations

### High battery usage

**Symptoms**: Battery drains quickly when using SilentRelay

**Solutions**:
1. **Background activity**
   - Reduce push notification frequency
   - Close app when not in use

2. **Media handling**
   - Disable auto-download of media
   - Reduce image quality if available

3. **Device settings**
   - Check battery optimization settings
   - Allow SilentRelay to run in background

## Security Issues

### Suspicious activity

**Symptoms**: Unexpected login, unknown devices, safety number changes

**Immediate actions**:
1. **Change PIN**
   - Settings → Security → Change PIN
   - Use strong, unique PIN

2. **Review devices**
   - Settings → Devices → Remove unknown devices
   - Check last active times

3. **Re-verify contacts**
   - Check safety numbers with trusted contacts
   - Use alternative communication methods

4. **Security audit**
   - Review recent activity logs
   - Change recovery key if compromised

### Account compromised

**Emergency response**:
1. **Secure account**
   - Change PIN immediately
   - Remove all suspicious devices
   - Enable two-factor if available

2. **Assess damage**
   - Check conversation history for suspicious activity
   - Re-verify all safety numbers

3. **Prevent future issues**
   - Use stronger PIN
   - Enable disappearing messages
   - Be more cautious with links/files

## Device-Specific Issues

### iOS Safari issues

**Common problems**:
- WebSocket connections fail
- Notifications don't work reliably
- Memory issues with large chats

**Solutions**:
- Use the iOS app if available
- Clear Safari cache regularly
- Close other Safari tabs

### Android Chrome issues

**Common problems**:
- Battery optimization kills background tasks
- Notifications delayed or missing
- Memory pressure crashes

**Solutions**:
- Disable battery optimization for Chrome
- Allow background activity
- Use Android app if available

### Desktop browser issues

**Common problems**:
- Extension conflicts
- Cache corruption
- Hardware acceleration issues

**Solutions**:
- Try incognito/private mode
- Disable browser extensions
- Update graphics drivers

## Advanced Troubleshooting

### Check browser console

For technical issues:
1. Press F12 or right-click → Inspect
2. Go to Console tab
3. Look for error messages
4. Note any SilentRelay-related errors

### Network diagnostics

1. **Ping test**: `ping your-server.com`
2. **Traceroute**: `tracert your-server.com`
3. **Port check**: Verify port 443 (HTTPS) is open

### Clear all data

**Last resort**:
1. Settings → Privacy → Clear All Data
2. Log out completely
3. Clear browser cache/cookies
4. Restart browser
5. Log back in

**Warning**: This will remove local message history. Messages will re-sync from server.

## Getting Help

### Before contacting support

1. **Try basic fixes**
   - Refresh page
   - Clear cache
   - Try different browser/device

2. **Gather information**
   - Browser and version
   - Operating system
   - Error messages
   - Steps to reproduce

3. **Check status**
   - Server status page
   - Service announcements
   - Known issues

### Contacting support

1. **In-app support**
   - Settings → Help & Support
   - Include diagnostic information

2. **Community forums**
   - Check GitHub issues
   - Search existing problems

3. **Enterprise support**
   - Contact your administrator
   - Use official support channels

### Reporting bugs

When reporting issues:
- Include browser/OS information
- Describe steps to reproduce
- Include error messages
- Attach screenshots if helpful
- Check if issue is already reported

---

*If you continue experiencing issues after trying these solutions, please contact support with detailed information about your problem.*