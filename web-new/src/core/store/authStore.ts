import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import SecureCookieManager from '../utils/cookies';
import type { User, AuthState } from '../types';

interface AuthStore extends AuthState {
  isAuthenticated: boolean;
  needsOnboarding: boolean;
  needsPinUnlock: boolean; // For existing users logging in
  onboardingStep: 'username' | 'pin' | 'recovery' | 'complete';
  recoveryKey: string | null;

  // Actions
  setAuth: (token: string, refreshToken: string, user: User, deviceId: string, isNewUser?: boolean) => void;
  updateUser: (user: Partial<User>) => void;
  setOnboardingStep: (step: 'username' | 'pin' | 'recovery' | 'complete') => void;
  setRecoveryKey: (key: string) => void;
  completeOnboarding: () => void;
  setNeedsPinUnlock: (needs: boolean) => void;
  completePinUnlock: () => void;
  logout: () => void;
}

export const useAuthStore = create<AuthStore>()(
  persist(
    (set) => ({
      token: SecureCookieManager.getAuthToken(),
      refreshToken: SecureCookieManager.getRefreshToken(),
      user: null,
      deviceId: SecureCookieManager.getDeviceId(),
      isAuthenticated: SecureCookieManager.isAuthenticated(),
      needsOnboarding: false,
      needsPinUnlock: false,
      onboardingStep: 'username',
      recoveryKey: null,

      setAuth: (token, refreshToken, user, deviceId, isNewUser = false) => {
        // Set secure cookies
        SecureCookieManager.setAuthCookies(token, refreshToken, deviceId);

        // Check if master key is already available in session (e.g., from page refresh)
        const hasMasterKey = !!sessionStorage.getItem('signal_master_key');

        set({
          token,
          refreshToken,
          user,
          deviceId,
          isAuthenticated: true,
          needsOnboarding: isNewUser,
          // Existing users need PIN unlock UNLESS master key is already in session
          needsPinUnlock: !isNewUser && !hasMasterKey,
          onboardingStep: isNewUser ? 'username' : 'complete',
        });
      },

      updateUser: (updates) =>
        set((state) => ({
          user: state.user ? { ...state.user, ...updates } : null,
        })),

      setOnboardingStep: (step) =>
        set({ onboardingStep: step }),

      setRecoveryKey: (key) =>
        set({ recoveryKey: key }),

      completeOnboarding: () =>
        set({
          needsOnboarding: false,
          onboardingStep: 'complete',
          recoveryKey: null, // Clear from memory after user confirms
        }),

      setNeedsPinUnlock: (needs) =>
        set({ needsPinUnlock: needs }),

      completePinUnlock: () =>
        set({ needsPinUnlock: false }),

      logout: () => {
        // Clear secure cookies
        SecureCookieManager.clearAuthCookies();

        set({
          token: null,
          refreshToken: null,
          user: null,
          deviceId: null,
          isAuthenticated: false,
          needsOnboarding: false,
          needsPinUnlock: false,
          onboardingStep: 'username',
          recoveryKey: null,
        });
      },
    }),
    {
      name: 'auth-storage',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        // Only persist non-sensitive data, tokens are in secure cookies
        user: state.user,
        isAuthenticated: state.isAuthenticated,
        needsOnboarding: state.needsOnboarding,
        onboardingStep: state.onboardingStep,
        // Note: recoveryKey is NOT persisted for security
        // Note: tokens and deviceId are stored in secure HTTP-only cookies
      }),
      // Custom rehydration to load tokens from cookies
      onRehydrateStorage: () => (state) => {
        if (state) {
          // Load tokens from secure cookies
          const token = SecureCookieManager.getAuthToken();
          const refreshToken = SecureCookieManager.getRefreshToken();
          const deviceId = SecureCookieManager.getDeviceId();
          const isAuthenticated = SecureCookieManager.isAuthenticated();

          state.token = token;
          state.refreshToken = refreshToken;
          state.deviceId = deviceId;
          state.isAuthenticated = isAuthenticated;
        }
      },
    }
  )
);
