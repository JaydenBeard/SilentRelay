/**
 * Secure Cookie Utilities
 *
 * Provides secure cookie management for authentication tokens
 * with proper security flags and expiration handling.
 */

export interface CookieOptions {
  expires?: Date;
  path?: string;
  domain?: string;
  secure?: boolean;
  httpOnly?: boolean;
  sameSite?: 'Strict' | 'Lax' | 'None';
  maxAge?: number;
  partitioned?: boolean;
  priority?: 'High' | 'Medium' | 'Low';
  useHostPrefix?: boolean;
}

export class SecureCookieManager {
  private static readonly TOKEN_COOKIE = 'auth_token';
  private static readonly REFRESH_TOKEN_COOKIE = 'refresh_token';
  private static readonly DEVICE_ID_COOKIE = 'device_id';

  /**
   * Set authentication cookies with secure defaults
   */
  static setAuthCookies(
    token: string,
    refreshToken: string,
    deviceId: string,
    expiresIn: number = 24 * 60 * 60 * 1000 // 24 hours
  ): void {
    const expires = new Date(Date.now() + expiresIn);

    // Set auth token (short-lived, accessible to JavaScript)
    this.setCookie(this.TOKEN_COOKIE, token, {
      expires,
      secure: true,
      sameSite: 'Strict',
      path: '/',
    });

    // Set refresh token (longer-lived, HTTP-only)
    this.setCookie(this.REFRESH_TOKEN_COOKIE, refreshToken, {
      expires: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000), // 7 days
      secure: true,
      httpOnly: true,
      sameSite: 'Strict',
      path: '/',
    });

    // Set device ID
    this.setCookie(this.DEVICE_ID_COOKIE, deviceId, {
      expires: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000), // 1 year
      secure: true,
      sameSite: 'Strict',
      path: '/',
    });
  }

  /**
   * Get authentication token from cookies
   */
  static getAuthToken(): string | null {
    return this.getCookie(this.TOKEN_COOKIE);
  }

  /**
   * Get refresh token from cookies (will be null due to httpOnly)
   */
  static getRefreshToken(): string | null {
    return this.getCookie(this.REFRESH_TOKEN_COOKIE);
  }

  /**
   * Get device ID from cookies
   */
  static getDeviceId(): string | null {
    return this.getCookie(this.DEVICE_ID_COOKIE);
  }

  /**
   * Clear all authentication cookies
   */
  static clearAuthCookies(): void {
    this.deleteCookie(this.TOKEN_COOKIE);
    this.deleteCookie(this.REFRESH_TOKEN_COOKIE);
    this.deleteCookie(this.DEVICE_ID_COOKIE);
  }

  /**
   * Check if user is authenticated (token exists and not expired)
   */
  static isAuthenticated(): boolean {
    const token = this.getAuthToken();
    if (!token) return false;

    try {
      // Parse JWT to check expiration
      const payload = JSON.parse(atob(token.split('.')[1]));
      const now = Math.floor(Date.now() / 1000);
      return payload.exp > now;
    } catch {
      return false;
    }
  }

  /**
   * Refresh authentication token
   */
  static async refreshAuthToken(): Promise<boolean> {
    try {
      const refreshToken = this.getRefreshToken();
      if (!refreshToken) return false;

      const response = await fetch('/api/v1/auth/refresh', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ refresh_token: refreshToken }),
        credentials: 'include', // Include cookies
      });

      if (!response.ok) return false;

      const data = await response.json();
      const deviceId = this.getDeviceId();

      if (data.token && data.refresh_token && deviceId) {
        this.setAuthCookies(data.token, data.refresh_token, deviceId);
        return true;
      }

      return false;
    } catch {
      return false;
    }
  }

  /**
   * Set a cookie with options
   */
  private static setCookie(name: string, value: string, options: CookieOptions = {}): void {
    const cookieParts: string[] = [`${name}=${encodeURIComponent(value)}`];

    if (options.expires) {
      cookieParts.push(`expires=${options.expires.toUTCString()}`);
    }

    if (options.maxAge) {
      cookieParts.push(`max-age=${options.maxAge}`);
    }

    if (options.path) {
      cookieParts.push(`path=${options.path}`);
    }

    if (options.domain) {
      cookieParts.push(`domain=${options.domain}`);
    }

    if (options.secure) {
      cookieParts.push('secure');
    }

    if (options.httpOnly) {
      cookieParts.push('httponly');
    }

    if (options.sameSite) {
      cookieParts.push(`samesite=${options.sameSite}`);
    }

    document.cookie = cookieParts.join('; ');
  }

  /**
   * Get a cookie value
   */
  private static getCookie(name: string): string | null {
    const nameEQ = name + '=';
    const ca = document.cookie.split(';');

    for (let i = 0; i < ca.length; i++) {
      let c = ca[i];
      while (c.charAt(0) === ' ') c = c.substring(1, c.length);
      if (c.indexOf(nameEQ) === 0) {
        return decodeURIComponent(c.substring(nameEQ.length, c.length));
      }
    }

    return null;
  }

  /**
   * Delete a cookie
   */
  private static deleteCookie(name: string): void {
    this.setCookie(name, '', {
      expires: new Date(0),
      path: '/',
      secure: true,
      sameSite: 'Strict',
    });
  }
}

export default SecureCookieManager;