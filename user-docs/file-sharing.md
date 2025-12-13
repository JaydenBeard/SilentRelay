# File Sharing

Send and receive encrypted files securely in SilentRelay.

## Supported File Types

### Documents
- **PDF**: Portable Document Format
- **Office Documents**: DOC, DOCX, XLS, XLSX, PPT, PPTX
- **Text Files**: TXT, RTF, MD
- **Archives**: ZIP, RAR, 7Z

### Images
- **Formats**: JPG, JPEG, PNG, GIF, WebP, BMP
- **Maximum Resolution**: No strict limit (server-dependent)
- **Compression**: Automatic optimization for faster sending

### Audio
- **Formats**: MP3, WAV, M4A, AAC, OGG
- **Quality**: Original quality preserved
- **Playback**: In-app audio player

### Video
- **Formats**: MP4, MOV, AVI, MKV, WebM
- **Codecs**: H.264, H.265, VP8, VP9
- **Maximum Length**: No limit (file size dependent)

### Other
- **Executables**: Blocked for security
- **Scripts**: Blocked for security
- **System Files**: Blocked for security

## File Size Limits

### Individual Files
- **Maximum Size**: 100MB per file
- **Recommended**: Under 50MB for reliable delivery
- **Large Files**: May take longer to upload/download

### Storage Quotas
- **Per User**: Varies by server configuration
- **Group Files**: Count toward group storage
- **Cleanup**: Old files may be automatically deleted

## Sending Files

### From Chat
1. **Open Conversation**
   - Start a 1-on-1 or group chat
   - Click the attachment icon (paperclip)

2. **Select Files**
   - Browse your device
   - Select multiple files (up to 10 at once)
   - Preview selection before sending

3. **Upload Process**
   - Files encrypted on your device
   - Progress indicator shows upload status
   - Cancel upload if needed

4. **Send Confirmation**
   - Files appear in message input
   - Add caption or description
   - Send like a regular message

### Drag and Drop
- **Desktop**: Drag files directly into chat
- **Mobile**: Use file picker
- **Multiple Files**: Drop multiple files at once

## Receiving Files

### Download Process
1. **File Message**
   - Received files appear as message attachments
   - Click to start download
   - Progress indicator shows download status

2. **Automatic Decryption**
   - Files decrypted on your device only
   - Original filename and type preserved
   - Safe to open after download

3. **File Location**
   - Downloads to browser's default download folder
   - Mobile: Downloads folder
   - Desktop: Configurable in browser settings

## File Security

### End-to-End Encryption
- **Client-Side Encryption**: Files encrypted before upload
- **Unique Keys**: Each file uses different encryption key
- **Server Storage**: Only encrypted blobs stored
- **Access Control**: Only recipients can decrypt

### Metadata Protection
- **Filename**: Encrypted (server sees generic name)
- **File Type**: Hidden from server
- **File Size**: Rounded to prevent analysis
- **Timestamps**: Only delivery time visible

### Virus Scanning
- **Automatic Checks**: Files scanned before decryption
- **Malware Detection**: Suspicious files blocked
- **Safe Downloads**: Only clean files delivered

## File Management

### Viewing Files
- **In-App Preview**: Images, documents, audio
- **External Opening**: Click to open in default app
- **Download Required**: Some file types require download

### File History
- **Chat History**: Files remain in conversation
- **Search**: Find files by name or type
- **Delete**: Remove individual files from chat

### Storage Management
- **View Usage**: Settings → Storage
- **Cleanup**: Delete old/unneeded files
- **Auto-Cleanup**: Server may delete old files

## Advanced Features

### File Captions
- **Add Description**: Text description with file
- **Searchable**: Captions included in search
- **Formatting**: Basic text formatting supported

### File Previews
- **Image Thumbnails**: Small previews in chat
- **Document Previews**: First page preview
- **Audio Waveforms**: Visual audio representation

### Batch Operations
- **Multiple Selection**: Select multiple files
- **Bulk Download**: Download several files at once
- **Bulk Delete**: Remove multiple files

## Troubleshooting

### Upload Failures

**Common Issues**:
- **File Too Large**: Reduce size or split file
- **Unsupported Format**: Convert to supported format
- **Network Issues**: Check connection and retry

**Solutions**:
1. Compress large files
2. Convert unsupported formats
3. Try different network
4. Contact support for persistent issues

### Download Problems

**Can't Download**:
- **Storage Full**: Clear space on device
- **Permissions**: Check download permissions
- **Corrupted**: Request sender to re-upload

**Slow Downloads**:
- **Large Files**: May take time on slow connections
- **Server Load**: Try during off-peak hours
- **Resume**: Downloads resume automatically

### File Corruption

**Symptoms**:
- File won't open
- Unexpected file size
- Error messages when opening

**Recovery**:
1. Try downloading again
2. Check file integrity (if possible)
3. Request new upload from sender
4. Contact support with error details

### Preview Issues

**Won't Preview**:
- **Unsupported Format**: Download to view
- **Large File**: May not preview for performance
- **Security**: Some files blocked for safety

## Security Best Practices

### File Handling
- **Verify Sources**: Only accept files from trusted contacts
- **Scan Externally**: Use antivirus on downloaded files
- **Be Cautious**: Don't open unexpected attachments

### Sharing Guidelines
- **Compress Sensitive Files**: Use password-protected archives
- **Use Captions**: Describe file contents
- **Consider Recipients**: Ensure they can open file types

### Privacy Considerations
- **File Metadata**: May contain location, device info
- **EXIF Data**: Photos may include GPS coordinates
- **Strip Metadata**: Use tools to clean files before sharing

## Enterprise Features

### File Policies
- **Type Restrictions**: Block certain file types
- **Size Limits**: Custom per-organization limits
- **Retention**: Automatic file deletion policies

### Audit Logging
- **File Access**: Track who downloads files
- **Upload Logs**: Monitor file sharing activity
- **Compliance**: Meet regulatory requirements

### Integration
- **Cloud Storage**: Link to enterprise file storage
- **Document Management**: Integration with DMS systems
- **Backup**: Automatic file backup solutions

## Performance Tips

### Upload Optimization
- **Compress Files**: Reduce size before sending
- **Batch Upload**: Send multiple files together
- **Stable Connection**: Use reliable internet

### Download Management
- **Selective Download**: Only download needed files
- **Auto-Download**: Configure for trusted senders
- **Storage Monitoring**: Regularly clean old files

### Network Considerations
- **Mobile Data**: Be aware of data usage
- **WiFi Preferred**: Use WiFi for large files
- **Background**: Uploads continue in background

Remember: All files are encrypted and can only be accessed by intended recipients. SilentRelay never has access to your file contents.