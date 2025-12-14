/**
 * PIN Unlock Component
 * 
 * Shown when an existing user logs in and needs to unlock their encrypted data.
 * If this is a new device (no encryption set up), offers to sync from primary device
 * or set up fresh encryption.
 */

import { useState, useCallback, useEffect } from 'react';
import { Shield, Lock, Eye, EyeOff, AlertCircle, Smartphone, RefreshCw, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { useAuthStore } from '@/core/store/authStore';
import { signalProtocol } from '@/core/crypto/signal';
import {
  getSyncState,
  onSyncStateChange,
  type SyncState,
} from '@/core/services/deviceSync';

export function PinUnlock() {
  const [pin, setPin] = useState('');
  const [showPin, setShowPin] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [isNewDevice, setIsNewDevice] = useState(false);
  const [isChecking, setIsChecking] = useState(true);
  const [syncState, setSyncState] = useState<SyncState>(getSyncState());

  const { user, logout } = useAuthStore();

  // Check if encryption is set up on this device
  useEffect(() => {
    async function checkEncryption() {
      try {
        await signalProtocol.initialize();
        const isEnabled = signalProtocol.isEncryptionEnabled();

        if (!isEnabled) {
          // New device - no encryption set up
          setIsNewDevice(true);
        }
      } catch (err) {
        console.error('Failed to check encryption status:', err);
        setIsNewDevice(true);
      } finally {
        setIsChecking(false);
      }
    }
    checkEncryption();
  }, []);

  // Listen for sync state changes
  useEffect(() => {
    const unsubscribe = onSyncStateChange((state) => {
      setSyncState(state);

      // If sync succeeded, complete setup
      if (state.status === 'success') {
        // Reload page to reinitialize with synced data
        window.location.reload();
      }
    });
    return unsubscribe;
  }, []);

  // Handle new device setup - go to PIN setup in onboarding
  const handleNewDeviceSetup = useCallback(() => {
    // Set onboarding to PIN step and skip PIN unlock
    useAuthStore.setState({
      needsOnboarding: true,
      onboardingStep: 'pin',
      needsPinUnlock: false
    });
  }, []);

  const handleUnlock = useCallback(async () => {
    if (pin.length < 6) {
      setError('PIN must be at least 6 characters');
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      // Initialize signal protocol if not already
      await signalProtocol.initialize();

      // Try to unlock with the PIN
      await signalProtocol.unlockWithPin(pin);

      // Success! Mark PIN unlock as complete (use getState to avoid dependency issues)
      useAuthStore.getState().completePinUnlock();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Invalid PIN');
      setPin('');
    } finally {
      setIsLoading(false);
    }
  }, [pin]);

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && pin.length >= 6) {
      handleUnlock();
    }
  }, [pin, handleUnlock]);

  // Show loading while checking encryption status
  if (isChecking) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center p-4">
        <div className="text-center">
          <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin mx-auto mb-4" />
          <p className="text-foreground-muted">Checking device...</p>
        </div>
      </div>
    );
  }

  // New device - show sync/setup options
  if (isNewDevice) {
    const isSyncing = ['requesting', 'waiting', 'receiving'].includes(syncState.status);
    const syncFailed = ['timeout', 'failed', 'no_primary'].includes(syncState.status);

    return (
      <div className="min-h-screen bg-background flex items-center justify-center p-4">
        <div className="w-full max-w-md">
          {/* Logo and Title */}
          <div className="text-center mb-8">
            <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-primary/10 mb-4">
              <Smartphone className="w-8 h-8 text-primary" />
            </div>
            <h1 className="text-2xl font-bold mb-2">New Device</h1>
            <p className="text-foreground-secondary">
              {user?.displayName || user?.username || 'User'}
            </p>
          </div>

          {/* Sync Status Card */}
          {isSyncing && (
            <div className="bg-background-secondary rounded-2xl p-6 shadow-lg border border-border mb-4">
              <div className="flex items-center gap-3 mb-4">
                <Loader2 className="w-5 h-5 text-primary animate-spin" />
                <div>
                  <h2 className="font-semibold">Syncing Sessions</h2>
                  <p className="text-sm text-foreground-secondary">
                    {syncState.message || 'Please wait...'}
                  </p>
                </div>
              </div>
              <p className="text-xs text-foreground-muted text-center">
                Keep your other device open until sync completes.
              </p>
            </div>
          )}

          {/* Sync Failed / New Device Options */}
          {(!isSyncing || syncFailed) && (
            <div className="bg-background-secondary rounded-2xl p-6 shadow-lg border border-border">
              {syncFailed && (
                <div className="mb-4 p-3 bg-warning/10 border border-warning/20 rounded-lg flex items-center gap-2 text-warning">
                  <AlertCircle className="w-4 h-4 flex-shrink-0" />
                  <span className="text-sm">{syncState.message || 'Sync failed'}</span>
                </div>
              )}

              <p className="text-foreground-secondary text-center mb-6">
                {syncFailed
                  ? "Couldn't sync from your other device. You can try again or start fresh."
                  : "Set up encryption on this device to secure your messages."
                }
              </p>

              <div className="space-y-3">
                {/* Retry sync button - only show if we have WebSocket */}
                {syncFailed && (
                  <Button
                    onClick={() => {
                      // Would trigger sync request again
                      // For now, just refresh to retry
                      window.location.reload();
                    }}
                    variant="outline"
                    className="w-full"
                    size="lg"
                  >
                    <RefreshCw className="w-4 h-4 mr-2" />
                    Try Sync Again
                  </Button>
                )}

                <Button
                  onClick={handleNewDeviceSetup}
                  className="w-full"
                  size="lg"
                >
                  Start Fresh
                </Button>
              </div>

              <p className="text-xs text-foreground-muted text-center mt-4">
                Starting fresh creates new encryption keys. Previous messages won't sync.
              </p>
            </div>
          )}

          {/* Logout Option */}
          <div className="text-center mt-6">
            <button
              onClick={logout}
              className="text-sm text-foreground-secondary hover:text-foreground transition-colors"
            >
              Sign in with a different account
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        {/* Logo and Title */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-primary/10 mb-4">
            <Shield className="w-8 h-8 text-primary" />
          </div>
          <h1 className="text-2xl font-bold mb-2">Welcome Back</h1>
          <p className="text-foreground-secondary">
            {user?.displayName || user?.username || 'User'}
          </p>
        </div>

        {/* PIN Entry Card */}
        <div className="bg-background-secondary rounded-2xl p-6 shadow-lg border border-border">
          <div className="flex items-center gap-3 mb-6">
            <div className="p-2 bg-primary/10 rounded-lg">
              <Lock className="w-5 h-5 text-primary" />
            </div>
            <div>
              <h2 className="font-semibold">Enter Your PIN</h2>
              <p className="text-sm text-foreground-secondary">
                Unlock your encrypted messages
              </p>
            </div>
          </div>

          {/* Error Message */}
          {error && (
            <div className="mb-4 p-3 bg-destructive/10 border border-destructive/20 rounded-lg flex items-center gap-2 text-destructive">
              <AlertCircle className="w-4 h-4 flex-shrink-0" />
              <span className="text-sm">{error}</span>
            </div>
          )}

          {/* PIN Input */}
          <div className="relative mb-6">
            <Input
              type={showPin ? 'text' : 'password'}
              placeholder="Enter your PIN"
              value={pin}
              onChange={(e) => setPin(e.target.value)}
              onKeyDown={handleKeyDown}
              className="pr-10 text-center text-lg tracking-widest"
              maxLength={20}
              disabled={isLoading}
              autoFocus
            />
            <button
              type="button"
              onClick={() => setShowPin(!showPin)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-foreground-secondary hover:text-foreground transition-colors"
            >
              {showPin ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
            </button>
          </div>

          {/* Unlock Button */}
          <Button
            onClick={handleUnlock}
            disabled={pin.length < 6 || isLoading}
            className="w-full"
            size="lg"
          >
            {isLoading ? 'Unlocking...' : 'Unlock'}
          </Button>

          {/* Help Text */}
          <p className="text-xs text-foreground-secondary text-center mt-4">
            Your PIN protects your encrypted messages. If you've forgotten it,
            you can use your recovery key in Settings.
          </p>
        </div>

        {/* Logout Option */}
        <div className="text-center mt-6">
          <button
            onClick={logout}
            className="text-sm text-foreground-secondary hover:text-foreground transition-colors"
          >
            Sign in with a different account
          </button>
        </div>
      </div>
    </div>
  );
}

export default PinUnlock;

