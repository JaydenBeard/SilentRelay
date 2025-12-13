/**
 * Onboarding Wizard
 *
 * Guides new users through:
 * 1. Setting up their unique username
 * 2. Saving their recovery key
 */

import { useState, useCallback, useEffect } from 'react';
import { useAuthStore } from '@/core/store/authStore';
import { users } from '@/core/api/client';
import { signalProtocol } from '@/core/crypto/signal';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Shield,
  User,
  Key,
  Copy,
  Check,
  Loader2,
  AlertTriangle,
  CheckCircle2,
  Lock,
} from 'lucide-react';

// Generate a 24-word recovery key (BIP39-style mnemonic)
const WORD_LIST = [
  'abandon', 'ability', 'able', 'about', 'above', 'absent', 'absorb', 'abstract',
  'absurd', 'abuse', 'access', 'accident', 'account', 'accuse', 'achieve', 'acid',
  'acoustic', 'acquire', 'across', 'act', 'action', 'actor', 'actress', 'actual',
  'adapt', 'add', 'addict', 'address', 'adjust', 'admit', 'adult', 'advance',
  'advice', 'aerobic', 'affair', 'afford', 'afraid', 'again', 'age', 'agent',
  'agree', 'ahead', 'aim', 'air', 'airport', 'aisle', 'alarm', 'album',
  'alcohol', 'alert', 'alien', 'all', 'alley', 'allow', 'almost', 'alone',
  'alpha', 'already', 'also', 'alter', 'always', 'amateur', 'amazing', 'among',
  'amount', 'amused', 'analyst', 'anchor', 'ancient', 'anger', 'angle', 'angry',
  'animal', 'ankle', 'announce', 'annual', 'another', 'answer', 'antenna', 'antique',
  'anxiety', 'any', 'apart', 'apology', 'appear', 'apple', 'approve', 'april',
  'arch', 'arctic', 'area', 'arena', 'argue', 'arm', 'armed', 'armor',
  'army', 'around', 'arrange', 'arrest', 'arrive', 'arrow', 'art', 'artefact',
  'artist', 'artwork', 'ask', 'aspect', 'assault', 'asset', 'assist', 'assume',
  'asthma', 'athlete', 'atom', 'attack', 'attend', 'attitude', 'attract', 'auction',
  'audit', 'august', 'aunt', 'author', 'auto', 'autumn', 'average', 'avocado',
];

function generateRecoveryKey(): string {
  const words: string[] = [];
  const array = new Uint32Array(24);
  crypto.getRandomValues(array);

  for (let i = 0; i < 24; i++) {
    words.push(WORD_LIST[array[i] % WORD_LIST.length]);
  }

  return words.join(' ');
}

// Username validation
function validateUsername(username: string): { valid: boolean; error?: string } {
  if (username.length < 3) {
    return { valid: false, error: 'Username must be at least 3 characters' };
  }
  if (username.length > 30) {
    return { valid: false, error: 'Username must be 30 characters or less' };
  }
  if (!/^[a-zA-Z0-9_]+$/.test(username)) {
    return { valid: false, error: 'Only letters, numbers, and underscores allowed' };
  }
  if (/^[0-9]/.test(username)) {
    return { valid: false, error: 'Username cannot start with a number' };
  }
  return { valid: true };
}

interface UsernameStepProps {
  onNext: () => void;
}

function UsernameStep({ onNext }: UsernameStepProps) {
  const [username, setUsername] = useState('');
  const [isChecking, setIsChecking] = useState(false);
  const [isAvailable, setIsAvailable] = useState<boolean | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { updateUser } = useAuthStore();

  // Debounced username availability check
  useEffect(() => {
    const validation = validateUsername(username);
    if (!validation.valid) {
      setError(validation.error || null);
      setIsAvailable(null);
      return;
    }

    setError(null);
    setIsChecking(true);
    setIsAvailable(null);

    const timeoutId = setTimeout(async () => {
      try {
        const result = await users.checkUsername(username);
        setIsAvailable(result.available);
        if (!result.available) {
          setError(result.message || 'Username is taken');
        }
      } catch {
        // If check fails, don't block - will be validated on submit
        setIsAvailable(null);
      } finally {
        setIsChecking(false);
      }
    }, 500);

    return () => clearTimeout(timeoutId);
  }, [username]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const validation = validateUsername(username);
    if (!validation.valid) {
      setError(validation.error || 'Invalid username');
      return;
    }

    setIsSubmitting(true);
    setError(null);

    try {
      await users.setUsername(username);
      updateUser({ username });
      onNext();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to set username');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleUsernameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    // Remove @ if user types it (we show it as prefix)
    const value = e.target.value.replace(/^@/, '').toLowerCase();
    setUsername(value);
  };

  return (
    <div className="space-y-6">
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-primary/10 mb-4">
          <User className="h-8 w-8 text-primary" />
        </div>
        <h2 className="text-xl font-bold">Choose Your Username</h2>
        <p className="text-foreground-secondary mt-2 text-sm">
          This is how others will find and message you.
        </p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="text-sm font-medium mb-2 block">Username</label>
          <div className="relative">
            <span className="absolute left-3 top-1/2 -translate-y-1/2 text-foreground-muted font-mono">
              @
            </span>
            <Input
              type="text"
              placeholder="your_username"
              value={username}
              onChange={handleUsernameChange}
              className="pl-8 font-mono"
              autoFocus
              autoComplete="off"
              spellCheck={false}
            />
            {username && (
              <div className="absolute right-3 top-1/2 -translate-y-1/2">
                {isChecking ? (
                  <Loader2 className="h-4 w-4 animate-spin text-foreground-muted" />
                ) : isAvailable === true ? (
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                ) : isAvailable === false ? (
                  <AlertTriangle className="h-4 w-4 text-red-500" />
                ) : null}
              </div>
            )}
          </div>
          {username && (
            <p className={`text-xs mt-1 ${error ? 'text-red-500' : isAvailable ? 'text-green-500' : 'text-foreground-muted'}`}>
              {error || (isAvailable ? 'Username is available!' : isChecking ? 'Checking availability...' : `${username.length}/30 characters`)}
            </p>
          )}
        </div>

        <div className="bg-background-tertiary rounded-lg p-4 text-sm">
          <p className="text-foreground-secondary">
            <strong>Why a username?</strong>
          </p>
          <ul className="mt-2 space-y-1 text-foreground-muted text-xs">
            <li>• Your phone number stays private</li>
            <li>• Share your @username to let people message you</li>
            <li>• Only your username is visible to other users</li>
          </ul>
        </div>

        <Button
          type="submit"
          className="w-full"
          disabled={isSubmitting || !username || isChecking || isAvailable === false || !!error}
        >
          {isSubmitting ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            'Continue'
          )}
        </Button>
      </form>
    </div>
  );
}

interface PinStepProps {
  onNext: () => void;
}

function PinStep({ onNext }: PinStepProps) {
  const [pin, setPin] = useState('');
  const [confirmPin, setConfirmPin] = useState('');
  const [isSettingUp, setIsSettingUp] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (pin.length < 6) {
      setError('PIN must be at least 6 characters');
      return;
    }

    if (pin !== confirmPin) {
      setError('PINs do not match');
      return;
    }

    setIsSettingUp(true);

    try {
      await signalProtocol.setupEncryption(pin);
      onNext();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to set up encryption');
    } finally {
      setIsSettingUp(false);
    }
  };

  const isValid = pin.length >= 6 && pin === confirmPin;

  return (
    <div className="space-y-6">
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-primary/10 mb-4">
          <Lock className="h-8 w-8 text-primary" />
        </div>
        <h2 className="text-xl font-bold">Set Up PIN Protection</h2>
        <p className="text-foreground-secondary mt-2 text-sm">
          Create a PIN to protect your encryption keys. This PIN will be required to access your messages.
        </p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="text-sm font-medium mb-2 block">PIN (6+ characters)</label>
          <Input
            type="password"
            placeholder="Enter your PIN"
            value={pin}
            onChange={(e) => setPin(e.target.value)}
            required
            autoFocus
            className="text-center text-lg font-mono tracking-widest"
          />
        </div>

        <div>
          <label className="text-sm font-medium mb-2 block">Confirm PIN</label>
          <Input
            type="password"
            placeholder="Confirm your PIN"
            value={confirmPin}
            onChange={(e) => setConfirmPin(e.target.value)}
            required
            className="text-center text-lg font-mono tracking-widest"
          />
        </div>

        {error && (
          <div className="text-center">
            <p className="text-sm text-red-500">{error}</p>
          </div>
        )}

        <div className="bg-background-tertiary rounded-lg p-4 text-sm">
          <p className="text-foreground-secondary">
            <strong>Why a PIN?</strong>
          </p>
          <ul className="mt-2 space-y-1 text-foreground-muted text-xs">
            <li>• Protects your encryption keys if your device is compromised</li>
            <li>• Required to unlock your messages on new devices</li>
            <li>• Keep it secure - don't share it with anyone</li>
          </ul>
        </div>

        <Button
          type="submit"
          className="w-full"
          disabled={isSettingUp || !isValid}
        >
          {isSettingUp ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            'Set PIN & Continue'
          )}
        </Button>
      </form>
    </div>
  );
}

interface RecoveryKeyStepProps {
  onComplete: () => void;
}

function RecoveryKeyStep({ onComplete }: RecoveryKeyStepProps) {
  const [recoveryKey, setRecoveryKey] = useState('');
  const [copied, setCopied] = useState(false);
  const [confirmed, setConfirmed] = useState(false);
  const { setRecoveryKey: storeRecoveryKey } = useAuthStore();

  useEffect(() => {
    const key = generateRecoveryKey();
    setRecoveryKey(key);
    storeRecoveryKey(key);
  }, [storeRecoveryKey]);

  const handleCopy = useCallback(() => {
    navigator.clipboard.writeText(recoveryKey);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }, [recoveryKey]);

  const handleComplete = () => {
    if (confirmed) {
      onComplete();
    }
  };

  return (
    <div className="space-y-4 sm:space-y-6">
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-12 h-12 sm:w-16 sm:h-16 rounded-2xl bg-yellow-500/10 mb-3 sm:mb-4">
          <Key className="h-6 w-6 sm:h-8 sm:w-8 text-yellow-500" />
        </div>
        <h2 className="text-lg sm:text-xl font-bold">Save Your Recovery Key</h2>
        <p className="text-foreground-secondary mt-2 text-xs sm:text-sm">
          Write down these 24 words in order. This is the ONLY way to recover your account and messages.
        </p>
      </div>

      <div className="space-y-3">
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-2">
          {recoveryKey.split(' ').map((word, index) => (
            <div
              key={index}
              className="bg-zinc-900 border border-zinc-700 rounded px-2 sm:px-3 py-1.5 sm:py-2 font-mono"
            >
              <span className="text-[10px] text-zinc-500 block mb-0.5">{index + 1}</span>
              <span className="text-xs sm:text-sm text-emerald-400 break-all">{word}</span>
            </div>
          ))}
        </div>

        <Button
          variant="outline"
          size="sm"
          className="w-full"
          onClick={handleCopy}
        >
          {copied ? (
            <>
              <Check className="h-4 w-4 mr-2" />
              Copied!
            </>
          ) : (
            <>
              <Copy className="h-4 w-4 mr-2" />
              Copy to Clipboard
            </>
          )}
        </Button>
      </div>

      <div className="bg-red-500/10 border border-red-500/30 rounded-lg p-3 sm:p-4">
        <div className="flex items-start gap-2 sm:gap-3">
          <AlertTriangle className="h-4 w-4 sm:h-5 sm:w-5 text-red-500 flex-shrink-0 mt-0.5" />
          <div className="text-xs sm:text-sm">
            <p className="font-medium text-red-400">Important Warning</p>
            <ul className="mt-1 sm:mt-2 space-y-1 text-red-300 text-[11px] sm:text-xs">
              <li>• Never share this recovery key with anyone</li>
              <li>• SilentRelay cannot recover your account without this key</li>
              <li>• If you lose this key, your messages are gone forever</li>
              <li>• Store it safely - use a password manager or write on paper</li>
            </ul>
          </div>
        </div>
      </div>

      <div className="space-y-3 sm:space-y-4">
        <label className="flex items-start gap-2 sm:gap-3 cursor-pointer">
          <input
            type="checkbox"
            checked={confirmed}
            onChange={(e) => setConfirmed(e.target.checked)}
            className="mt-0.5 sm:mt-1 rounded border-border w-4 h-4 flex-shrink-0"
          />
          <span className="text-xs sm:text-sm text-foreground-secondary">
            I have written down my recovery key and stored it safely. I understand losing this key means losing access to my encrypted messages.
          </span>
        </label>

        <Button
          className="w-full"
          disabled={!confirmed}
          onClick={handleComplete}
        >
          I've Saved My Recovery Key
        </Button>
      </div>
    </div>
  );
}

export function Onboarding() {
  const { onboardingStep, setOnboardingStep, completeOnboarding } = useAuthStore();

  const handleUsernameComplete = () => {
    setOnboardingStep('pin');
  };

  const handlePinComplete = () => {
    setOnboardingStep('recovery');
  };

  const handleRecoveryComplete = () => {
    completeOnboarding();
  };

  return (
    <div className="fixed inset-0 z-50 bg-background/95 backdrop-blur-sm overflow-y-auto">
      <div className="min-h-full flex flex-col items-center justify-start sm:justify-center px-4 py-6 sm:py-8">
        <div className="w-full max-w-md">
          {/* Progress indicator */}
          <div className="flex items-center justify-center gap-2 mb-6 sm:mb-8">
            <div className={`w-8 h-1 rounded-full ${onboardingStep === 'username' ? 'bg-primary' : 'bg-primary/30'}`} />
            <div className={`w-8 h-1 rounded-full ${onboardingStep === 'pin' ? 'bg-primary' : 'bg-primary/30'}`} />
            <div className={`w-8 h-1 rounded-full ${onboardingStep === 'recovery' ? 'bg-primary' : 'bg-primary/30'}`} />
          </div>

          {/* Logo */}
          <div className="text-center mb-4 sm:mb-6">
            <div className="inline-flex items-center gap-2">
              <Shield className="h-6 w-6 text-primary" />
              <span className="text-lg font-bold">SilentRelay</span>
            </div>
          </div>

          {/* Content Card */}
          <div className="bg-background-secondary border border-border rounded-xl p-4 sm:p-6">
            {onboardingStep === 'username' && (
              <UsernameStep onNext={handleUsernameComplete} />
            )}
            {onboardingStep === 'pin' && (
              <PinStep onNext={handlePinComplete} />
            )}
            {onboardingStep === 'recovery' && (
              <RecoveryKeyStep onComplete={handleRecoveryComplete} />
            )}
          </div>

          {/* Step indicator text */}
          <p className="text-center text-xs text-foreground-muted mt-4 pb-4">
            Step {
              onboardingStep === 'username' ? '1' :
              onboardingStep === 'pin' ? '2' :
              onboardingStep === 'recovery' ? '3' : '1'
            } of 3
          </p>
        </div>
      </div>
    </div>
  );
}
