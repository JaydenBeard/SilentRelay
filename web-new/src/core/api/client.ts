/**
 * API Client
 *
 * Provides typed HTTP methods for communicating with the backend.
 * Automatically handles authentication tokens and error responses.
 * Uses standardized error codes for all errors.
 */

import { useAuthStore } from '../store/authStore';
import { AppError, ErrorCodes, httpStatusToErrorCode, type ErrorCode } from '@/lib/errors';

const API_BASE = '/api/v1';

interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';
  body?: unknown;
  headers?: Record<string, string>;
  skipAuth?: boolean;
  context?: string; // For error context (e.g., 'user', 'media')
}

async function request<T>(endpoint: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, headers = {}, skipAuth = false, context } = options;

  // Get auth token
  const token = useAuthStore.getState().token;

  const requestHeaders: Record<string, string> = {
    'Content-Type': 'application/json',
    ...headers,
  };

  if (!skipAuth && token) {
    requestHeaders['Authorization'] = `Bearer ${token}`;
  }

  let response: Response;

  try {
    response = await fetch(`${API_BASE}${endpoint}`, {
      method,
      headers: requestHeaders,
      body: body ? JSON.stringify(body) : undefined,
    });
  } catch (error) {
    // Network error - no response received
    if (error instanceof TypeError && error.message.includes('fetch')) {
      // Check if offline
      if (!navigator.onLine) {
        throw new AppError(ErrorCodes.NET_OFFLINE);
      }
      throw new AppError(ErrorCodes.NET_CONNECTION_FAILED);
    }
    throw new AppError(ErrorCodes.NET_REQUEST_FAILED, undefined, { originalError: String(error) });
  }

  // Handle non-JSON responses
  const contentType = response.headers.get('content-type');
  if (!contentType?.includes('application/json')) {
    if (!response.ok) {
      const errorCode = httpStatusToErrorCode(response.status, context);
      throw new AppError(errorCode, undefined, { status: response.status });
    }
    return {} as T;
  }

  let data: Record<string, unknown>;
  try {
    data = await response.json();
  } catch {
    throw new AppError(ErrorCodes.NET_REQUEST_FAILED, 'Invalid response format');
  }

  if (!response.ok) {
    // Map server error code if provided
    const serverCode = data.code as string | undefined;
    const serverMessage = (data.error || data.message) as string | undefined;

    // Check if server returned a known error code
    let errorCode: ErrorCode;
    if (serverCode && Object.values(ErrorCodes).includes(serverCode as ErrorCode)) {
      errorCode = serverCode as ErrorCode;
    } else {
      errorCode = httpStatusToErrorCode(response.status, context);
    }

    throw new AppError(errorCode, serverMessage, {
      status: response.status,
      serverCode,
      endpoint,
    });
  }

  return data as T;
}

// Auth endpoints
export const auth = {
  sendCode: (phoneNumber: string) =>
    request<{ message: string; code?: string }>('/auth/request-code', {
      method: 'POST',
      body: { phone_number: phoneNumber },
      skipAuth: true,
    }),

  verifyCode: (phoneNumber: string, code: string) =>
    request<{
      verified: boolean;
      user_exists: boolean;
      user_id?: string;
      // These are only returned for existing users who are logging in
      token?: string;
      refresh_token?: string;
      user?: {
        id: string;
        phone_number: string;
        username?: string;
        display_name?: string;
        avatar_url?: string;
      };
      device_id?: string;
    }>('/auth/verify', {
      method: 'POST',
      body: { phone_number: phoneNumber, code },
      skipAuth: true,
    }),

  register: (data: {
    phoneNumber: string;
    code: string;
    publicIdentityKey: string;
    publicSignedPrekey: string;
    signedPrekeySignature: string;
    preKeys: Array<{ prekeyId: number; publicKey: string }>;
    deviceId: string;
    deviceType: string;
    publicDeviceKey: string;
  }) =>
    request<{
      access_token: string;
      refresh_token: string;
      user: { user_id: string; phone_number: string };
    }>('/auth/register', {
      method: 'POST',
      body: {
        phone_number: data.phoneNumber,
        code: data.code,
        public_identity_key: data.publicIdentityKey,
        public_signed_prekey: data.publicSignedPrekey,
        signed_prekey_signature: data.signedPrekeySignature,
        prekeys: data.preKeys.map(k => ({
          prekey_id: k.prekeyId,
          public_key: k.publicKey,
        })),
        device_id: data.deviceId,
        device_type: data.deviceType,
        public_device_key: data.publicDeviceKey,
      },
      skipAuth: true,
    }),

  refreshToken: (refreshToken: string) =>
    request<{ token: string; refresh_token: string }>('/auth/refresh', {
      method: 'POST',
      body: { refresh_token: refreshToken },
      skipAuth: true,
    }),

  logout: () =>
    request<void>('/auth/logout', {
      method: 'POST',
    }),
};

// User endpoints
export const users = {
  getProfile: () =>
    request<{
      id: string;
      phoneNumber: string;
      username?: string;
      displayName?: string;
      avatar?: string;
    }>('/users/me'),

  updateProfile: (data: { username?: string; displayName?: string; avatar?: string }) =>
    request<void>('/users/me', {
      method: 'PATCH',
      body: {
        ...(data.username && { username: data.username }),
        ...(data.displayName && { display_name: data.displayName }),
        ...(data.avatar && { avatar_url: data.avatar }),
      },
    }),

  checkUsername: (username: string) =>
    request<{ available: boolean; message?: string }>(`/users/check-username/${encodeURIComponent(username)}`, {
      method: 'GET',
    }),

  setUsername: (username: string) =>
    request<{ status: string }>('/users/me', {
      method: 'PATCH',
      body: { username },
    }),

  getKeys: (userId: string) =>
    request<{
      identityKey: number[];
      signedPreKey: {
        keyId: number;
        publicKey: number[];
        signature: number[];
      };
      preKey?: {
        keyId: number;
        publicKey: number[];
      };
      registrationId: number;
    }>(`/users/${userId}/keys`, { context: 'user' }),

  uploadPreKeys: (preKeys: Array<{ keyId: number; publicKey: number[] }>) =>
    request<void>('/users/me/prekeys', {
      method: 'POST',
      body: { preKeys },
    }),

  /**
   * Update encryption keys on the server
   * Used when setting up fresh encryption on a new device
   * This will trigger identity_key_changed notifications to all contacts
   */
  updateKeys: (data: {
    publicIdentityKey: string;
    publicSignedPrekey: string;
    signedPrekeySignature: string;
  }) =>
    request<{ status: string; identity_key_changed: boolean }>('/users/keys', {
      method: 'POST',
      body: {
        public_identity_key: data.publicIdentityKey,
        public_signed_prekey: data.publicSignedPrekey,
        signed_prekey_signature: data.signedPrekeySignature,
      },
    }),

  searchByUsername: (username: string) =>
    request<Array<{
      user_id: string;
      username?: string;
      display_name?: string;
      avatar_url?: string;
    }>>(`/users/search?q=${encodeURIComponent(username)}&limit=10`, {
      method: 'GET',
      context: 'user',
    }),

  getUserProfile: (userId: string) =>
    request<{
      user_id: string;
      username?: string;
      display_name?: string;
      avatar_url?: string;
      last_seen?: string;
      is_online?: boolean;
    }>(`/users/${userId}/profile`, {
      method: 'GET',
      context: 'user',
    }),
};

// Messages endpoints
export const messages = {
  getHistory: (conversationId: string, limit = 50, before?: string) =>
    request<Array<{
      id: string;
      senderId: string;
      ciphertext: string;
      messageType: 'prekey' | 'whisper';
      timestamp: string;
    }>>(`/messages/${conversationId}?limit=${limit}${before ? `&before=${before}` : ''}`),
};

// Media endpoints
export const media = {
  getUploadUrl: (fileName: string, mimeType: string) =>
    request<{
      uploadUrl: string;
      mediaId: string;
    }>('/media/upload-url', {
      method: 'POST',
      body: { fileName, mimeType },
      context: 'media',
    }),

  getDownloadUrl: (mediaId: string) =>
    request<{
      downloadUrl: string;
    }>(`/media/${mediaId}/download-url`, { context: 'media' }),
};

// Settings endpoints
export const settings = {
  getPrivacy: () =>
    request<{
      readReceipts: boolean;
      onlineStatus: boolean;
      lastSeen: boolean;
      typingIndicators: boolean;
    }>('/settings/privacy'),

  updatePrivacy: (data: Partial<{
    readReceipts: boolean;
    onlineStatus: boolean;
    lastSeen: boolean;
    typingIndicators: boolean;
  }>) =>
    request<void>('/settings/privacy', {
      method: 'PATCH',
      body: data,
    }),
};

export { AppError };
