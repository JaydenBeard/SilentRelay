import DOMPurify from 'dompurify';
import { logger } from './logger';

/**
 * Enhanced message content sanitization with comprehensive XSS protection
 * @param content - Raw message content to sanitize
 * @returns Sanitized content safe for rendering
 */
export function sanitizeMessageContent(content: string): string {
  try {
    // Validate input
    if (content == null || typeof content !== 'string') {
      logger.warn('sanitizeMessageContent: Invalid input type, returning empty string');
      return '';
    }

    // Enhanced DOMPurify configuration with comprehensive XSS protection
    const sanitized = DOMPurify.sanitize(content, {
      ALLOWED_TAGS: ['b', 'i', 'strong', 'em', 'u', 's', 'br', 'p', 'span'],
      ALLOWED_ATTR: ['class', 'style'],
      FORBID_TAGS: ['script', 'iframe', 'frame', 'object', 'embed', 'applet', 'link', 'meta'],
      FORBID_ATTR: ['onerror', 'onclick', 'onload', 'onmouseover', 'style', 'href', 'src'],
      ADD_URI_SAFE_ATTR: ['href', 'src'],
      SANITIZE_DOM: true,
      WHOLE_DOCUMENT: false,
      RETURN_DOM: false,
      RETURN_DOM_FRAGMENT: false,
      RETURN_TRUSTED_TYPE: false,
      ALLOW_UNKNOWN_PROTOCOLS: false,
      ALLOW_DATA_ATTR: false,
      ALLOW_ARIA_ATTR: false,
      ALLOWED_URI_REGEXP: /^(?:(?:https?|mailto|ftp|tel|data):|[^a-z]|[a-z+.\-]+(?:[^a-z+.\-:]|$))/i
    });

    // Additional security checks
    if (containsMaliciousPatterns(sanitized)) {
      logger.warn('sanitizeMessageContent: Detected malicious patterns after sanitization, returning empty string');
      return '';
    }

    return sanitized;
  } catch (error) {
    logger.error('sanitizeMessageContent: Sanitization failed', error);
    // Fallback to strict sanitization
    return DOMPurify.sanitize(content, {
      ALLOWED_TAGS: [],
      ALLOWED_ATTR: []
    });
  }
}

/**
 * Check for malicious patterns that might bypass DOMPurify
 * @param content - Content to check
 * @returns True if malicious patterns detected
 */
function containsMaliciousPatterns(content: string): boolean {
  const maliciousPatterns = [
    // Script injection patterns
    /<script[\s\S]*?>/i,
    /javascript:/i,
    /on\w+\s*=/i,
    /expression\s*\(/i,
    /vbscript:/i,
    /data:text\/html/i,

    // Common XSS patterns
    /&#x?[0-9a-f]+;/i,
    /document\.cookie/i,
    /window\.location/i,
    /eval\s*\(/i,
    /setTimeout\s*\(/i,
    /setInterval\s*\(/i,

    // Malicious URL patterns
    /https?:\/\/(?:[^\/]+\.)?[^\/]+\.(?:js|php|asp|aspx|jsp|cfm|pl|py|rb|sh|exe|dll|bat|cmd|vbs|ps1|msi)(?:\?|#|$)/i,
    /(?:[^\/]+\.)?[^\/]+\.(?:js|php|asp|aspx|jsp|cfm|pl|py|rb|sh|exe|dll|bat|cmd|vbs|ps1|msi)(?:\?|#|$)/i
  ];

  return maliciousPatterns.some(pattern => pattern.test(content));
}

/**
 * Enhanced URL validation for thumbnail URLs with security checks
 * @param url - URL to validate
 * @returns True if URL is safe for rendering
 */
export function isSafeThumbnailUrl(url: string): boolean {
  if (typeof url !== 'string') {
    logger.warn('isSafeThumbnailUrl: Invalid URL type');
    return false;
  }

  try {
    // Check for data URIs with image content
    if (url.startsWith('data:image/')) {
      return true;
    }

    // Check for HTTPS URLs with proper domain validation
    if (url.startsWith('https://')) {
      const parsedUrl = new URL(url);

      // Validate domain and path
      const allowedDomains = ['cdn.silentrelay.com', 'media.silentrelay.com', 'assets.silentrelay.com'];
      const isAllowedDomain = allowedDomains.includes(parsedUrl.hostname);

      // Check for malicious file extensions
      const maliciousExtensions = ['.js', '.php', '.asp', '.aspx', '.jsp', '.cfm', '.pl', '.py', '.rb', '.sh', '.exe', '.dll', '.bat', '.cmd', '.vbs', '.ps1', '.msi'];
      const hasMaliciousExtension = maliciousExtensions.some(ext =>
        parsedUrl.pathname.toLowerCase().endsWith(ext)
      );

      return isAllowedDomain && !hasMaliciousExtension;
    }

    return false;
  } catch (error) {
    logger.error('isSafeThumbnailUrl: URL validation failed', error);
    return false;
  }
}

/**
 * Sanitize file names to prevent path traversal and XSS
 * @param fileName - File name to sanitize
 * @returns Sanitized file name
 */
export function sanitizeFileName(fileName: string): string {
  if (typeof fileName !== 'string') {
    logger.warn('sanitizeFileName: Invalid file name type, returning empty string');
    return '';
  }

  // Remove path traversal characters
  const sanitized = fileName
    .replace(/[\\/:*?"<>|]/g, '_')
    .replace(/^\.+/, '')
    .replace(/\.+$/, '')
    .trim();

  // Additional security check
  if (containsMaliciousPatterns(sanitized)) {
    logger.warn('sanitizeFileName: Detected malicious patterns in file name');
    return 'sanitized_file';
  }

  return sanitized;
}

/**
 * Sanitize user-generated content for display in UI elements
 * @param content - Content to sanitize
 * @returns Sanitized content safe for UI display
 */
export function sanitizeUserContent(content: string): string {
  try {
    if (content == null || typeof content !== 'string') {
      logger.warn('sanitizeUserContent: Invalid content type, returning empty string');
      return '';
    }

    // Strict sanitization for user-generated content
    return DOMPurify.sanitize(content, {
      ALLOWED_TAGS: ['b', 'i', 'strong', 'em'],
      ALLOWED_ATTR: [],
      FORBID_TAGS: ['script', 'iframe', 'frame', 'object', 'embed', 'applet', 'link', 'meta'],
      FORBID_ATTR: ['onerror', 'onclick', 'onload', 'onmouseover', 'style', 'href', 'src'],
      SANITIZE_DOM: true,
      WHOLE_DOCUMENT: false,
      RETURN_TRUSTED_TYPE: false
    });
  } catch (error) {
    logger.error('sanitizeUserContent: Sanitization failed', error);
    return '';
  }
}