import { useState, useMemo } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { useAuth } from '@/hooks/useAuth';
import { Shield, ArrowLeft, Loader2, ChevronDown } from 'lucide-react';
import {
  countryConfigs,
  getCountryConfig,
  getE164PhoneNumber,
  validatePhoneNumber,
} from '@/lib/phoneNumber';
import { ErrorMessage } from '@/components/error';
import { ErrorCodes, isAppError } from '@/lib/errors';

type AuthStep = 'phone' | 'verify' | 'setup';

export default function AuthPage() {
  const [step, setStep] = useState<AuthStep>('phone');
  const [countryCode, setCountryCode] = useState('+1');
  const [phoneDigits, setPhoneDigits] = useState('');
  const [verificationCode, setVerificationCode] = useState('');
  const [validationError, setValidationError] = useState<string | null>(null);

  const { isLoading, error, devCode, sendCode, verifyCode, register, clearError } = useAuth();

  // Get current country config
  const countryConfig = useMemo(() => getCountryConfig(countryCode), [countryCode]);

  // Format phone for display
  const displayPhone = useMemo(() => {
    if (!countryConfig) return phoneDigits;
    return countryConfig.format(phoneDigits);
  }, [phoneDigits, countryConfig]);

  // Get E.164 formatted number for API
  const e164Phone = useMemo(
    () => getE164PhoneNumber(phoneDigits, countryCode),
    [phoneDigits, countryCode]
  );

  // Handle backspace on space characters - delete the preceding digit
  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Backspace') {
      const input = e.currentTarget;
      const cursorPos = input.selectionStart || 0;

      // Check if the character before cursor is a space
      if (cursorPos > 0 && displayPhone[cursorPos - 1] === ' ') {
        e.preventDefault();
        // Count how many digits are before this cursor position
        const digitsBeforeCursor = displayPhone.slice(0, cursorPos).replace(/\D/g, '').length;
        // Remove that digit
        const newDigits = phoneDigits.slice(0, digitsBeforeCursor - 1) + phoneDigits.slice(digitsBeforeCursor);
        setPhoneDigits(newDigits);
      }
    }
  };

  // Handle phone input - extract digits from formatted input
  const handlePhoneChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const rawInput = e.target.value;
    // Extract only digits from the input
    let newDigits = rawInput.replace(/\D/g, '');

    // Strip leading zero if required by the country
    if (countryConfig?.stripLeadingZero && newDigits.startsWith('0')) {
      newDigits = newDigits.slice(1);
    }

    // Limit to expected length
    if (countryConfig) {
      newDigits = newDigits.slice(0, countryConfig.length);
    }

    setPhoneDigits(newDigits);
    setValidationError(null);
  };

  // Handle country code change - reset phone if switching
  const handleCountryChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const newCode = e.target.value;
    setCountryCode(newCode);

    // Re-process the phone digits for the new country
    const newConfig = getCountryConfig(newCode);
    if (newConfig && phoneDigits) {
      // Strip leading zero if new country requires it
      let processed = phoneDigits;
      if (newConfig.stripLeadingZero && processed.startsWith('0')) {
        processed = processed.slice(1);
      }
      setPhoneDigits(processed.slice(0, newConfig.length));
    }

    setValidationError(null);
  };

  const handleSendCode = async (e: React.FormEvent) => {
    e.preventDefault();
    setValidationError(null);

    // Validate phone number
    const validation = validatePhoneNumber(phoneDigits, countryCode);
    if (!validation.isValid) {
      setValidationError(validation.error || 'Invalid phone number');
      return;
    }

    const success = await sendCode(e164Phone);
    if (success) {
      setStep('verify');
    }
  };

  const handleVerifyCode = async (e: React.FormEvent) => {
    e.preventDefault();
    const result = await verifyCode(e164Phone, verificationCode);
    if (result?.isNewUser) {
      setStep('setup');
      // Auto-start registration for new users
      await register(e164Phone);
    }
  };

  const handleBack = () => {
    clearError();
    setStep('phone');
    setVerificationCode('');
    setValidationError(null);
  };

  // Get the error to display (validation error takes precedence)
  const displayError = validationError
    ? { message: validationError, code: ErrorCodes.AUTH_INVALID_PHONE }
    : error
      ? { message: error.message, code: isAppError(error) ? error.code : undefined }
      : null;

  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-4 sm:p-6 overflow-y-auto">
      <div className="w-full max-w-md my-auto">
        {/* Logo */}
        <div className="text-center mb-6 sm:mb-8">
          <div className="inline-flex items-center justify-center w-14 h-14 sm:w-16 sm:h-16 rounded-2xl bg-primary/10 mb-3 sm:mb-4">
            <Shield className="h-7 w-7 sm:h-8 sm:w-8 text-primary" />
          </div>
          <h1 className="text-xl sm:text-2xl font-bold">SilentRelay</h1>
          <p className="text-foreground-secondary mt-2 text-sm sm:text-base">
            {step === 'phone' && 'Enter your phone number to get started'}
            {step === 'verify' && 'Enter the verification code'}
            {step === 'setup' && 'Setting up your secure account'}
          </p>
        </div>

        {/* Form Card */}
        <div className="bg-background-secondary border border-border rounded-xl p-4 sm:p-6">
          {step === 'phone' && (
            <form onSubmit={handleSendCode} className="space-y-4">
              <div>
                <label className="text-sm font-medium mb-2 block">
                  Phone Number
                </label>
                <div className="flex gap-2">
                  {/* Country Code Dropdown */}
                  <div className="relative">
                    <select
                      value={countryCode}
                      onChange={handleCountryChange}
                      className="h-10 pl-3 pr-8 rounded-md border border-input bg-background text-sm appearance-none cursor-pointer focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
                    >
                      {countryConfigs.map((cc) => (
                        <option key={cc.code} value={cc.code}>
                          {cc.country} {cc.code}
                        </option>
                      ))}
                    </select>
                    <ChevronDown className="absolute right-2 top-1/2 -translate-y-1/2 h-4 w-4 text-foreground-secondary pointer-events-none" />
                  </div>
                  {/* Phone Number Input */}
                  <Input
                    type="tel"
                    placeholder={countryConfig?.placeholder || '(555) 123-4567'}
                    value={displayPhone}
                    onChange={handlePhoneChange}
                    onKeyDown={handleKeyDown}
                    required
                    autoFocus
                    className="flex-1"
                  />
                </div>
              </div>
              {displayError && (
                <ErrorMessage message={displayError.message} code={displayError.code} />
              )}
              <Button
                type="submit"
                className="w-full"
                disabled={isLoading || (countryConfig && phoneDigits.length !== countryConfig.length)}
              >
                {isLoading ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  'Continue'
                )}
              </Button>
            </form>
          )}

          {step === 'verify' && (
            <form onSubmit={handleVerifyCode} className="space-y-4">
              <button
                type="button"
                onClick={handleBack}
                className="flex items-center gap-1 text-sm text-foreground-secondary hover:text-foreground mb-4"
              >
                <ArrowLeft className="h-4 w-4" />
                Change number
              </button>
              <div>
                <label className="text-sm font-medium mb-2 block">
                  Verification Code
                </label>
                <Input
                  type="text"
                  placeholder="000000"
                  value={verificationCode}
                  onChange={(e) => setVerificationCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                  maxLength={6}
                  required
                  autoFocus
                  className="text-center text-2xl tracking-widest font-mono"
                />
                <p className="text-xs text-foreground-muted mt-2">
                  Sent to {e164Phone}
                </p>
                {/* Dev mode code display */}
                {devCode && (
                  <div className="mt-3 p-3 rounded-lg bg-yellow-500/10 border border-yellow-500/30">
                    <p className="text-xs text-yellow-500 font-medium">DEV MODE</p>
                    <p className="text-lg font-mono font-bold text-yellow-400 tracking-widest">
                      {devCode}
                    </p>
                  </div>
                )}
              </div>
              {error && (
                <ErrorMessage
                  message={error.message}
                  code={isAppError(error) ? error.code : undefined}
                />
              )}
              <Button
                type="submit"
                className="w-full"
                disabled={isLoading || verificationCode.length !== 6}
              >
                {isLoading ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  'Verify'
                )}
              </Button>
            </form>
          )}

          {step === 'setup' && (
            <div className="space-y-4">
              <div className="text-center py-8">
                <Loader2 className="h-8 w-8 animate-spin mx-auto text-primary mb-4" />
                <p className="text-foreground-secondary">
                  Generating encryption keys...
                </p>
                <p className="text-xs text-foreground-muted mt-2">
                  This ensures only you can read your messages
                </p>
              </div>
              {error && (
                <div className="text-center">
                  <ErrorMessage
                    message={error.message}
                    code={isAppError(error) ? error.code : undefined}
                  />
                </div>
              )}
            </div>
          )}
        </div>

        {/* Footer */}
        <p className="text-center text-xs text-foreground-muted mt-4 sm:mt-6 pb-4">
          By continuing, you agree to our Terms of Service and Privacy Policy.
        </p>
      </div>
    </div>
  );
}
