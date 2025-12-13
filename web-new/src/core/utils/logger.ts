/**
 * Security-focused logger for sanitization and security operations
 */
export const logger = {
  /**
   * Log security warnings
   * @param message - Warning message
   * @param data - Additional data
   */
  warn: (message: string, data?: any) => {
    console.warn(`[SECURITY] ${message}`, data);
    // In production, this would also send to security monitoring
  },

  /**
   * Log security errors
   * @param message - Error message
   * @param error - Error object
   */
  error: (message: string, error?: any) => {
    console.error(`[SECURITY] ${message}`, error);
    // In production, this would also send to security monitoring
  },

  /**
   * Log security information
   * @param message - Information message
   * @param data - Additional data
   */
  info: (message: string, data?: any) => {
    console.info(`[SECURITY] ${message}`, data);
  }
};