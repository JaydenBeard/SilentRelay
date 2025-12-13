/**
 * Auth Hook
 *
 * Provides authentication methods with Signal Protocol
 * key generation during registration.
 */

import { useState, useCallback, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '@/core/store/authStore';
// Chat data persists across logout (encrypted with PIN)
import { signalProtocol } from '@/core/crypto/signal';
import { auth } from '@/core/api/client';

interface AuthError {
  message: string;
  code?: string;
}

// Helper to convert Uint8Array to base64 string
function uint8ArrayToBase64(bytes: Uint8Array): string {
  let binary = '';
  for (let i = 0; i < bytes.length; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary);
}

// Generate a UUID v4
function generateUUID(): string {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
    const r = Math.random() * 16 | 0;
    const v = c === 'x' ? r : (r & 0x3 | 0x8);
    return v.toString(16);
  });
}

export function useAuth() {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<AuthError | null>(null);
  const navigate = useNavigate();

  const { setAuth, logout: storeLogout, isAuthenticated } = useAuthStore();
  // Chat data persists across logout - encrypted with PIN

  // Dev mode code (only populated in development)
  const [devCode, setDevCode] = useState<string | null>(null);

  // Store the verification code for use during registration
  const verificationCodeRef = useRef<string>('');

  // Send verification code
  const sendCode = useCallback(async (phoneNumber: string): Promise<boolean> => {
    setIsLoading(true);
    setError(null);
    setDevCode(null);

    try {
      const result = await auth.sendCode(phoneNumber);
      // In dev mode, the server returns the code
      if (result.code) {
        setDevCode(result.code);
      }
      return true;
    } catch (err) {
      setError({
        message: err instanceof Error ? err.message : 'Failed to send verification code',
      });
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Verify code
  const verifyCode = useCallback(
    async (phoneNumber: string, code: string): Promise<{ isNewUser: boolean } | null> => {
      setIsLoading(true);
      setError(null);

      try {
        const result = await auth.verifyCode(phoneNumber, code);

        // user_exists: true means existing user, false means new user
        const isNewUser = !result.user_exists;

        if (result.user_exists && result.token && result.user && result.device_id) {
          // Existing user - log them in with full user data
          const user = {
            id: result.user.id,
            phoneNumber: result.user.phone_number,
            username: result.user.username,
            displayName: result.user.display_name,
            avatar: result.user.avatar_url,
          };
          setAuth(result.token, result.refresh_token!, user as any, result.device_id);
          navigate('/chat');
        } else {
          // Store the code for registration
          verificationCodeRef.current = code;
        }

        return { isNewUser };
      } catch (err) {
        setError({
          message: err instanceof Error ? err.message : 'Invalid verification code',
        });
        return null;
      } finally {
        setIsLoading(false);
      }
    },
    [setAuth, navigate]
  );

  // Complete registration with Signal Protocol keys
  const register = useCallback(
    async (phoneNumber: string): Promise<boolean> => {
      setIsLoading(true);
      setError(null);

      try {
        // Initialize Signal Protocol
        await signalProtocol.initialize();

        // Generate identity keys
        const identityKeyPair = await signalProtocol.generateIdentityKeyPair();
        const signedPreKey = await signalProtocol.generateSignedPreKey(1);
        const preKeys = await signalProtocol.generatePreKeys(1, 100);

        // Generate device ID - device key IS the identity key (same as Signal protocol)
        const deviceId = generateUUID();

        // Convert keys to base64 strings
        const publicIdentityKey = uint8ArrayToBase64(identityKeyPair.publicKey);
        const publicSignedPrekey = uint8ArrayToBase64(signedPreKey.publicKey);
        const signedPrekeySignature = uint8ArrayToBase64(signedPreKey.signature);
        const publicDeviceKey = publicIdentityKey; // Device key = Identity key for this device

        // Register with server
        const result = await auth.register({
          phoneNumber,
          code: verificationCodeRef.current,
          publicIdentityKey,
          publicSignedPrekey,
          signedPrekeySignature,
          preKeys: preKeys.map((pk) => ({
            prekeyId: pk.keyId,
            publicKey: uint8ArrayToBase64(pk.publicKey),
          })),
          deviceId,
          deviceType: 'web',
          publicDeviceKey,
        });

        // Set auth state - convert snake_case response to camelCase
        const user = {
          id: result.user.user_id,
          phoneNumber: result.user.phone_number,
        };
        // Pass true for isNewUser to trigger onboarding
        // Note: deviceId is generated locally, not returned from server
        setAuth(result.access_token, result.refresh_token, user as any, deviceId, true);
        navigate('/chat');

        return true;
      } catch (err) {
        setError({
          message: err instanceof Error ? err.message : 'Failed to complete registration',
        });
        return false;
      } finally {
        setIsLoading(false);
      }
    },
    [setAuth, navigate]
  );

  // Logout
  const logout = useCallback(async () => {
    try {
      await auth.logout();
    } catch {
      // Ignore logout errors
    } finally {
      storeLogout();
      // Don't clear chat - data is encrypted with PIN and will persist
      // When user logs back in with same PIN, they'll get their messages back
      navigate('/');
    }
  }, [storeLogout, navigate]);

  return {
    isLoading,
    error,
    isAuthenticated,
    devCode,
    sendCode,
    verifyCode,
    register,
    logout,
    clearError: () => setError(null),
  };
}
