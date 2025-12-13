import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type { PrivacySettings, NotificationSettings, AppSettings } from '../types';

interface SettingsState {
  privacy: PrivacySettings;
  notifications: NotificationSettings;
  app: AppSettings;
}

interface SettingsActions {
  updatePrivacy: (settings: Partial<PrivacySettings>) => void;
  updateNotifications: (settings: Partial<NotificationSettings>) => void;
  updateApp: (settings: Partial<AppSettings>) => void;
  resetToDefaults: () => void;
  loadPrivacyFromServer: () => Promise<void>;
}

type SettingsStore = SettingsState & SettingsActions;

const defaultSettings: SettingsState = {
  privacy: {
    readReceipts: true,
    onlineStatus: true,
    lastSeen: true,
    typingIndicators: true,
  },
  notifications: {
    enabled: true,
    sound: true,
    preview: true,
  },
  app: {
    theme: 'dark',
    fontSize: 'medium',
    language: 'en',
  },
};

// Map frontend setting names to backend setting names
const privacySettingMap: Record<string, string> = {
  readReceipts: 'show_read_receipts',
  onlineStatus: 'show_online_status',
  lastSeen: 'show_last_seen',
  typingIndicators: 'show_typing_indicator',
};

// Sync a privacy setting to the backend
async function syncPrivacyToBackend(setting: string, value: boolean): Promise<void> {
  const backendSetting = privacySettingMap[setting];
  if (!backendSetting) return;

  try {
    const { useAuthStore } = await import('./authStore');
    const token = useAuthStore.getState().token;
    if (!token) return;

    await fetch('/api/v1/privacy', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ setting: backendSetting, value }),
    });
  } catch (error) {
    console.error('Failed to sync privacy setting to server:', error);
  }
}

export const useSettingsStore = create<SettingsStore>()(
  persist(
    (set) => ({
      ...defaultSettings,

      updatePrivacy: (settings) => {
        // Update local state
        set((state) => ({
          privacy: { ...state.privacy, ...settings },
        }));

        // Sync to backend for privacy-critical settings
        Object.entries(settings).forEach(([key, value]) => {
          if (typeof value === 'boolean') {
            syncPrivacyToBackend(key, value);
          }
        });
      },

      updateNotifications: (settings) =>
        set((state) => ({
          notifications: { ...state.notifications, ...settings },
        })),

      updateApp: (settings) =>
        set((state) => ({
          app: { ...state.app, ...settings },
        })),

      resetToDefaults: () => set(defaultSettings),

      loadPrivacyFromServer: async () => {
        try {
          const { useAuthStore } = await import('./authStore');
          const token = useAuthStore.getState().token;
          if (!token) return;

          const response = await fetch('/api/v1/privacy', {
            headers: { 'Authorization': `Bearer ${token}` },
          });

          if (response.ok) {
            const serverSettings = await response.json();
            set((state) => ({
              privacy: {
                readReceipts: serverSettings.show_read_receipts ?? state.privacy.readReceipts,
                onlineStatus: serverSettings.show_online_status ?? state.privacy.onlineStatus,
                lastSeen: serverSettings.show_last_seen ?? state.privacy.lastSeen,
                typingIndicators: serverSettings.show_typing_indicator ?? state.privacy.typingIndicators,
              },
            }));
          }
        } catch (error) {
          console.error('Failed to load privacy settings from server:', error);
        }
      },
    }),
    {
      name: 'settings-storage',
      storage: createJSONStorage(() => localStorage),
    }
  )
);
