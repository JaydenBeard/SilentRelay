/**
 * Error Code Schema
 *
 * Standardized error codes for the application.
 * Format: [CATEGORY]-[NUMBER]
 *
 * Categories:
 * - AUTH: Authentication and authorization errors
 * - NET: Network and connectivity errors
 * - CRYPTO: Encryption and key management errors
 * - MSG: Messaging errors
 * - MEDIA: File upload/download errors
 * - CALL: WebRTC call errors
 * - SYS: System and internal errors
 */

// Error categories
export type ErrorCategory = 'AUTH' | 'NET' | 'CRYPTO' | 'MSG' | 'MEDIA' | 'CALL' | 'SYS';

// Error severity levels
export type ErrorSeverity = 'info' | 'warning' | 'error' | 'fatal';

// Error code definitions
export const ErrorCodes = {
  // Authentication errors (AUTH-001 to AUTH-099)
  AUTH_INVALID_PHONE: 'AUTH-001',
  AUTH_INVALID_CODE: 'AUTH-002',
  AUTH_CODE_EXPIRED: 'AUTH-003',
  AUTH_TOO_MANY_ATTEMPTS: 'AUTH-004',
  AUTH_SESSION_EXPIRED: 'AUTH-005',
  AUTH_UNAUTHORIZED: 'AUTH-006',
  AUTH_FORBIDDEN: 'AUTH-007',
  AUTH_DEVICE_NOT_APPROVED: 'AUTH-008',
  AUTH_REGISTRATION_FAILED: 'AUTH-009',
  AUTH_TOKEN_REFRESH_FAILED: 'AUTH-010',

  // Network errors (NET-001 to NET-099)
  NET_CONNECTION_FAILED: 'NET-001',
  NET_TIMEOUT: 'NET-002',
  NET_SERVER_UNAVAILABLE: 'NET-003',
  NET_WEBSOCKET_FAILED: 'NET-004',
  NET_WEBSOCKET_CLOSED: 'NET-005',
  NET_REQUEST_FAILED: 'NET-006',
  NET_OFFLINE: 'NET-007',
  NET_DNS_FAILED: 'NET-008',
  NET_SSL_ERROR: 'NET-009',

  // Cryptography errors (CRYPTO-001 to CRYPTO-099)
  CRYPTO_KEY_GENERATION_FAILED: 'CRYPTO-001',
  CRYPTO_ENCRYPTION_FAILED: 'CRYPTO-002',
  CRYPTO_DECRYPTION_FAILED: 'CRYPTO-003',
  CRYPTO_SIGNATURE_INVALID: 'CRYPTO-004',
  CRYPTO_SESSION_NOT_FOUND: 'CRYPTO-005',
  CRYPTO_SESSION_CORRUPTED: 'CRYPTO-006',
  CRYPTO_PREKEY_EXHAUSTED: 'CRYPTO-007',
  CRYPTO_KEY_EXCHANGE_FAILED: 'CRYPTO-008',
  CRYPTO_STORAGE_FAILED: 'CRYPTO-009',

  // Messaging errors (MSG-001 to MSG-099)
  MSG_SEND_FAILED: 'MSG-001',
  MSG_DELIVERY_FAILED: 'MSG-002',
  MSG_RECIPIENT_NOT_FOUND: 'MSG-003',
  MSG_CONTENT_TOO_LARGE: 'MSG-004',
  MSG_INVALID_FORMAT: 'MSG-005',
  MSG_RATE_LIMITED: 'MSG-006',

  // Media errors (MEDIA-001 to MEDIA-099)
  MEDIA_UPLOAD_FAILED: 'MEDIA-001',
  MEDIA_DOWNLOAD_FAILED: 'MEDIA-002',
  MEDIA_FILE_TOO_LARGE: 'MEDIA-003',
  MEDIA_UNSUPPORTED_TYPE: 'MEDIA-004',
  MEDIA_ENCRYPTION_FAILED: 'MEDIA-005',
  MEDIA_DECRYPTION_FAILED: 'MEDIA-006',
  MEDIA_NOT_FOUND: 'MEDIA-007',
  MEDIA_CORRUPT: 'MEDIA-008',

  // Call errors (CALL-001 to CALL-099)
  CALL_CONNECTION_FAILED: 'CALL-001',
  CALL_MEDIA_ACCESS_DENIED: 'CALL-002',
  CALL_PEER_UNAVAILABLE: 'CALL-003',
  CALL_ICE_FAILED: 'CALL-004',
  CALL_SIGNALING_FAILED: 'CALL-005',
  CALL_TIMEOUT: 'CALL-006',
  CALL_REJECTED: 'CALL-007',

  // System errors (SYS-001 to SYS-099)
  SYS_INTERNAL_ERROR: 'SYS-001',
  SYS_DATABASE_ERROR: 'SYS-002',
  SYS_STORAGE_FULL: 'SYS-003',
  SYS_BROWSER_NOT_SUPPORTED: 'SYS-004',
  SYS_PERMISSION_DENIED: 'SYS-005',
  SYS_MAINTENANCE: 'SYS-006',
  SYS_UNKNOWN: 'SYS-999',
} as const;

export type ErrorCode = typeof ErrorCodes[keyof typeof ErrorCodes];

// Error message mappings
const errorMessages: Record<ErrorCode, string> = {
  // Auth
  [ErrorCodes.AUTH_INVALID_PHONE]: 'Invalid phone number format',
  [ErrorCodes.AUTH_INVALID_CODE]: 'Invalid verification code',
  [ErrorCodes.AUTH_CODE_EXPIRED]: 'Verification code has expired. Please request a new one',
  [ErrorCodes.AUTH_TOO_MANY_ATTEMPTS]: 'Too many attempts. Please try again later',
  [ErrorCodes.AUTH_SESSION_EXPIRED]: 'Your session has expired. Please sign in again',
  [ErrorCodes.AUTH_UNAUTHORIZED]: 'You need to sign in to continue',
  [ErrorCodes.AUTH_FORBIDDEN]: 'You do not have permission to perform this action',
  [ErrorCodes.AUTH_DEVICE_NOT_APPROVED]: 'This device has not been approved',
  [ErrorCodes.AUTH_REGISTRATION_FAILED]: 'Registration failed. Please try again',
  [ErrorCodes.AUTH_TOKEN_REFRESH_FAILED]: 'Failed to refresh session. Please sign in again',

  // Network
  [ErrorCodes.NET_CONNECTION_FAILED]: 'Unable to connect to the server',
  [ErrorCodes.NET_TIMEOUT]: 'Request timed out. Please check your connection',
  [ErrorCodes.NET_SERVER_UNAVAILABLE]: 'Server is temporarily unavailable',
  [ErrorCodes.NET_WEBSOCKET_FAILED]: 'Real-time connection failed',
  [ErrorCodes.NET_WEBSOCKET_CLOSED]: 'Real-time connection was closed',
  [ErrorCodes.NET_REQUEST_FAILED]: 'Request failed. Please try again',
  [ErrorCodes.NET_OFFLINE]: 'You appear to be offline',
  [ErrorCodes.NET_DNS_FAILED]: 'Unable to resolve server address',
  [ErrorCodes.NET_SSL_ERROR]: 'Secure connection failed',

  // Crypto
  [ErrorCodes.CRYPTO_KEY_GENERATION_FAILED]: 'Failed to generate encryption keys',
  [ErrorCodes.CRYPTO_ENCRYPTION_FAILED]: 'Failed to encrypt message',
  [ErrorCodes.CRYPTO_DECRYPTION_FAILED]: 'Failed to decrypt message',
  [ErrorCodes.CRYPTO_SIGNATURE_INVALID]: 'Message signature verification failed',
  [ErrorCodes.CRYPTO_SESSION_NOT_FOUND]: 'Encryption session not found',
  [ErrorCodes.CRYPTO_SESSION_CORRUPTED]: 'Encryption session is corrupted',
  [ErrorCodes.CRYPTO_PREKEY_EXHAUSTED]: 'Pre-keys exhausted. Unable to establish session',
  [ErrorCodes.CRYPTO_KEY_EXCHANGE_FAILED]: 'Key exchange failed',
  [ErrorCodes.CRYPTO_STORAGE_FAILED]: 'Failed to store encryption keys',

  // Messaging
  [ErrorCodes.MSG_SEND_FAILED]: 'Failed to send message',
  [ErrorCodes.MSG_DELIVERY_FAILED]: 'Message delivery failed',
  [ErrorCodes.MSG_RECIPIENT_NOT_FOUND]: 'Recipient not found',
  [ErrorCodes.MSG_CONTENT_TOO_LARGE]: 'Message content is too large',
  [ErrorCodes.MSG_INVALID_FORMAT]: 'Invalid message format',
  [ErrorCodes.MSG_RATE_LIMITED]: 'Sending too fast. Please slow down',

  // Media
  [ErrorCodes.MEDIA_UPLOAD_FAILED]: 'Failed to upload file',
  [ErrorCodes.MEDIA_DOWNLOAD_FAILED]: 'Failed to download file',
  [ErrorCodes.MEDIA_FILE_TOO_LARGE]: 'File is too large. Maximum size is 100MB',
  [ErrorCodes.MEDIA_UNSUPPORTED_TYPE]: 'File type is not supported',
  [ErrorCodes.MEDIA_ENCRYPTION_FAILED]: 'Failed to encrypt file',
  [ErrorCodes.MEDIA_DECRYPTION_FAILED]: 'Failed to decrypt file',
  [ErrorCodes.MEDIA_NOT_FOUND]: 'File not found',
  [ErrorCodes.MEDIA_CORRUPT]: 'File is corrupted',

  // Call
  [ErrorCodes.CALL_CONNECTION_FAILED]: 'Call connection failed',
  [ErrorCodes.CALL_MEDIA_ACCESS_DENIED]: 'Camera or microphone access was denied',
  [ErrorCodes.CALL_PEER_UNAVAILABLE]: 'The person you are calling is unavailable',
  [ErrorCodes.CALL_ICE_FAILED]: 'Failed to establish peer connection',
  [ErrorCodes.CALL_SIGNALING_FAILED]: 'Call signaling failed',
  [ErrorCodes.CALL_TIMEOUT]: 'Call timed out',
  [ErrorCodes.CALL_REJECTED]: 'Call was rejected',

  // System
  [ErrorCodes.SYS_INTERNAL_ERROR]: 'An internal error occurred',
  [ErrorCodes.SYS_DATABASE_ERROR]: 'Database error occurred',
  [ErrorCodes.SYS_STORAGE_FULL]: 'Storage is full',
  [ErrorCodes.SYS_BROWSER_NOT_SUPPORTED]: 'Your browser is not supported',
  [ErrorCodes.SYS_PERMISSION_DENIED]: 'Permission denied',
  [ErrorCodes.SYS_MAINTENANCE]: 'System is under maintenance',
  [ErrorCodes.SYS_UNKNOWN]: 'An unknown error occurred',
};

// Error severity mappings
const errorSeverities: Record<ErrorCode, ErrorSeverity> = {
  // Auth - mostly errors, some fatal
  [ErrorCodes.AUTH_INVALID_PHONE]: 'error',
  [ErrorCodes.AUTH_INVALID_CODE]: 'error',
  [ErrorCodes.AUTH_CODE_EXPIRED]: 'warning',
  [ErrorCodes.AUTH_TOO_MANY_ATTEMPTS]: 'error',
  [ErrorCodes.AUTH_SESSION_EXPIRED]: 'warning',
  [ErrorCodes.AUTH_UNAUTHORIZED]: 'error',
  [ErrorCodes.AUTH_FORBIDDEN]: 'error',
  [ErrorCodes.AUTH_DEVICE_NOT_APPROVED]: 'error',
  [ErrorCodes.AUTH_REGISTRATION_FAILED]: 'fatal',
  [ErrorCodes.AUTH_TOKEN_REFRESH_FAILED]: 'error',

  // Network - warnings for recoverable, errors for critical
  [ErrorCodes.NET_CONNECTION_FAILED]: 'error',
  [ErrorCodes.NET_TIMEOUT]: 'warning',
  [ErrorCodes.NET_SERVER_UNAVAILABLE]: 'fatal',
  [ErrorCodes.NET_WEBSOCKET_FAILED]: 'error',
  [ErrorCodes.NET_WEBSOCKET_CLOSED]: 'warning',
  [ErrorCodes.NET_REQUEST_FAILED]: 'error',
  [ErrorCodes.NET_OFFLINE]: 'warning',
  [ErrorCodes.NET_DNS_FAILED]: 'fatal',
  [ErrorCodes.NET_SSL_ERROR]: 'fatal',

  // Crypto - mostly fatal as they compromise security
  [ErrorCodes.CRYPTO_KEY_GENERATION_FAILED]: 'fatal',
  [ErrorCodes.CRYPTO_ENCRYPTION_FAILED]: 'fatal',
  [ErrorCodes.CRYPTO_DECRYPTION_FAILED]: 'error',
  [ErrorCodes.CRYPTO_SIGNATURE_INVALID]: 'fatal',
  [ErrorCodes.CRYPTO_SESSION_NOT_FOUND]: 'error',
  [ErrorCodes.CRYPTO_SESSION_CORRUPTED]: 'fatal',
  [ErrorCodes.CRYPTO_PREKEY_EXHAUSTED]: 'error',
  [ErrorCodes.CRYPTO_KEY_EXCHANGE_FAILED]: 'fatal',
  [ErrorCodes.CRYPTO_STORAGE_FAILED]: 'fatal',

  // Messaging - mostly errors
  [ErrorCodes.MSG_SEND_FAILED]: 'error',
  [ErrorCodes.MSG_DELIVERY_FAILED]: 'warning',
  [ErrorCodes.MSG_RECIPIENT_NOT_FOUND]: 'error',
  [ErrorCodes.MSG_CONTENT_TOO_LARGE]: 'error',
  [ErrorCodes.MSG_INVALID_FORMAT]: 'error',
  [ErrorCodes.MSG_RATE_LIMITED]: 'warning',

  // Media - mostly errors
  [ErrorCodes.MEDIA_UPLOAD_FAILED]: 'error',
  [ErrorCodes.MEDIA_DOWNLOAD_FAILED]: 'error',
  [ErrorCodes.MEDIA_FILE_TOO_LARGE]: 'error',
  [ErrorCodes.MEDIA_UNSUPPORTED_TYPE]: 'error',
  [ErrorCodes.MEDIA_ENCRYPTION_FAILED]: 'error',
  [ErrorCodes.MEDIA_DECRYPTION_FAILED]: 'error',
  [ErrorCodes.MEDIA_NOT_FOUND]: 'error',
  [ErrorCodes.MEDIA_CORRUPT]: 'error',

  // Call - errors
  [ErrorCodes.CALL_CONNECTION_FAILED]: 'error',
  [ErrorCodes.CALL_MEDIA_ACCESS_DENIED]: 'error',
  [ErrorCodes.CALL_PEER_UNAVAILABLE]: 'warning',
  [ErrorCodes.CALL_ICE_FAILED]: 'error',
  [ErrorCodes.CALL_SIGNALING_FAILED]: 'error',
  [ErrorCodes.CALL_TIMEOUT]: 'warning',
  [ErrorCodes.CALL_REJECTED]: 'info',

  // System - mostly fatal
  [ErrorCodes.SYS_INTERNAL_ERROR]: 'fatal',
  [ErrorCodes.SYS_DATABASE_ERROR]: 'fatal',
  [ErrorCodes.SYS_STORAGE_FULL]: 'error',
  [ErrorCodes.SYS_BROWSER_NOT_SUPPORTED]: 'fatal',
  [ErrorCodes.SYS_PERMISSION_DENIED]: 'error',
  [ErrorCodes.SYS_MAINTENANCE]: 'info',
  [ErrorCodes.SYS_UNKNOWN]: 'fatal',
};

/**
 * Application Error class with error codes
 */
export class AppError extends Error {
  readonly code: ErrorCode;
  readonly severity: ErrorSeverity;
  readonly timestamp: Date;
  readonly referenceId: string;
  readonly context?: Record<string, unknown>;

  constructor(
    code: ErrorCode,
    customMessage?: string,
    context?: Record<string, unknown>
  ) {
    const message = customMessage || errorMessages[code] || 'An error occurred';
    super(message);

    this.name = 'AppError';
    this.code = code;
    this.severity = errorSeverities[code] || 'error';
    this.timestamp = new Date();
    this.referenceId = generateReferenceId();
    this.context = context;

    // Maintains proper stack trace for where error was thrown
    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, AppError);
    }
  }

  /**
   * Get formatted error for display to user
   */
  toDisplayString(): string {
    return `${this.message}\n\nError Code: ${this.code}\nReference: ${this.referenceId}`;
  }

  /**
   * Get error details for logging/support
   */
  toLogObject(): Record<string, unknown> {
    return {
      code: this.code,
      message: this.message,
      severity: this.severity,
      referenceId: this.referenceId,
      timestamp: this.timestamp.toISOString(),
      context: this.context,
      stack: this.stack,
    };
  }

  /**
   * Check if error is fatal (should show full-screen error)
   */
  isFatal(): boolean {
    return this.severity === 'fatal';
  }
}

/**
 * Generate a unique reference ID for support tickets
 * Format: [DATE]-[RANDOM] e.g., "20241204-A1B2C3"
 */
function generateReferenceId(): string {
  const date = new Date();
  const dateStr = date.toISOString().slice(0, 10).replace(/-/g, '');
  const randomStr = Math.random().toString(36).substring(2, 8).toUpperCase();
  return `${dateStr}-${randomStr}`;
}

/**
 * Convert HTTP status codes to error codes
 */
export function httpStatusToErrorCode(status: number, context?: string): ErrorCode {
  switch (status) {
    case 400:
      return ErrorCodes.NET_REQUEST_FAILED;
    case 401:
      return ErrorCodes.AUTH_UNAUTHORIZED;
    case 403:
      return ErrorCodes.AUTH_FORBIDDEN;
    case 404:
      if (context === 'user') return ErrorCodes.MSG_RECIPIENT_NOT_FOUND;
      if (context === 'media') return ErrorCodes.MEDIA_NOT_FOUND;
      return ErrorCodes.NET_REQUEST_FAILED;
    case 408:
      return ErrorCodes.NET_TIMEOUT;
    case 429:
      return ErrorCodes.AUTH_TOO_MANY_ATTEMPTS;
    case 500:
      return ErrorCodes.SYS_INTERNAL_ERROR;
    case 502:
    case 503:
    case 504:
      return ErrorCodes.NET_SERVER_UNAVAILABLE;
    default:
      return ErrorCodes.SYS_UNKNOWN;
  }
}

/**
 * Create an AppError from an API error response
 */
export function createApiError(
  status: number,
  serverCode?: string,
  serverMessage?: string,
  context?: string
): AppError {
  // Map server error codes if provided
  const errorCode = serverCode && serverCode in ErrorCodes
    ? serverCode as ErrorCode
    : httpStatusToErrorCode(status, context);

  return new AppError(errorCode, serverMessage, { status, serverCode });
}

/**
 * Check if an error is an AppError
 */
export function isAppError(error: unknown): error is AppError {
  return error instanceof AppError;
}

/**
 * Get error message for display
 */
export function getErrorMessage(code: ErrorCode): string {
  return errorMessages[code] || errorMessages[ErrorCodes.SYS_UNKNOWN];
}

/**
 * Get error severity
 */
export function getErrorSeverity(code: ErrorCode): ErrorSeverity {
  return errorSeverities[code] || 'error';
}
