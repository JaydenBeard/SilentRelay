/**
 * Phone Number Formatting Utility
 *
 * Handles country-specific phone number formatting and validation
 * for international 2FA services (e.g., Twilio, Vonage).
 */

export interface CountryPhoneConfig {
  code: string;           // Country dial code (e.g., '+61')
  country: string;        // Country abbreviation
  length: number;         // Expected digit length (after country code)
  stripLeadingZero: boolean; // Whether to strip leading 0 from local numbers
  placeholder: string;    // Example placeholder
  format: (digits: string) => string; // Display formatting function
}

// Country phone configurations
export const countryConfigs: CountryPhoneConfig[] = [
  {
    code: '+1',
    country: 'US/CA',
    length: 10,
    stripLeadingZero: false,
    placeholder: '(555) 123-4567',
    format: (d) => d.length >= 6
      ? `(${d.slice(0, 3)}) ${d.slice(3, 6)}-${d.slice(6, 10)}`
      : d.length >= 3
        ? `(${d.slice(0, 3)}) ${d.slice(3)}`
        : d,
  },
  {
    code: '+61',
    country: 'AU',
    length: 9, // Australian mobile is 9 digits after country code (04XX becomes 4XX)
    stripLeadingZero: true,
    placeholder: '412 345 678',
    format: (d) => d.length >= 6
      ? `${d.slice(0, 3)} ${d.slice(3, 6)} ${d.slice(6, 9)}`
      : d.length >= 3
        ? `${d.slice(0, 3)} ${d.slice(3)}`
        : d,
  },
  {
    code: '+44',
    country: 'UK',
    length: 10, // UK mobile is 10 digits after country code (07XXX becomes 7XXX)
    stripLeadingZero: true,
    placeholder: '7911 123456',
    format: (d) => d.length >= 4
      ? `${d.slice(0, 4)} ${d.slice(4, 10)}`
      : d,
  },
  {
    code: '+49',
    country: 'DE',
    length: 11, // German mobile varies, typically 10-11
    stripLeadingZero: true,
    placeholder: '151 12345678',
    format: (d) => d.length >= 3
      ? `${d.slice(0, 3)} ${d.slice(3)}`
      : d,
  },
  {
    code: '+33',
    country: 'FR',
    length: 9, // French mobile is 9 digits after country code
    stripLeadingZero: true,
    placeholder: '6 12 34 56 78',
    format: (d) => {
      const parts = [];
      for (let i = 0; i < d.length; i += 2) {
        parts.push(d.slice(i, i + 2));
      }
      return parts.join(' ');
    },
  },
  {
    code: '+81',
    country: 'JP',
    length: 10, // Japanese mobile
    stripLeadingZero: true,
    placeholder: '90 1234 5678',
    format: (d) => d.length >= 6
      ? `${d.slice(0, 2)} ${d.slice(2, 6)} ${d.slice(6)}`
      : d.length >= 2
        ? `${d.slice(0, 2)} ${d.slice(2)}`
        : d,
  },
  {
    code: '+86',
    country: 'CN',
    length: 11, // Chinese mobile
    stripLeadingZero: false,
    placeholder: '138 1234 5678',
    format: (d) => d.length >= 7
      ? `${d.slice(0, 3)} ${d.slice(3, 7)} ${d.slice(7)}`
      : d.length >= 3
        ? `${d.slice(0, 3)} ${d.slice(3)}`
        : d,
  },
  {
    code: '+91',
    country: 'IN',
    length: 10, // Indian mobile
    stripLeadingZero: false,
    placeholder: '98765 43210',
    format: (d) => d.length >= 5
      ? `${d.slice(0, 5)} ${d.slice(5)}`
      : d,
  },
  {
    code: '+55',
    country: 'BR',
    length: 11, // Brazilian mobile (with area code)
    stripLeadingZero: false,
    placeholder: '11 91234 5678',
    format: (d) => d.length >= 7
      ? `${d.slice(0, 2)} ${d.slice(2, 7)} ${d.slice(7)}`
      : d.length >= 2
        ? `${d.slice(0, 2)} ${d.slice(2)}`
        : d,
  },
  {
    code: '+52',
    country: 'MX',
    length: 10, // Mexican mobile
    stripLeadingZero: false,
    placeholder: '55 1234 5678',
    format: (d) => d.length >= 6
      ? `${d.slice(0, 2)} ${d.slice(2, 6)} ${d.slice(6)}`
      : d.length >= 2
        ? `${d.slice(0, 2)} ${d.slice(2)}`
        : d,
  },
  {
    code: '+34',
    country: 'ES',
    length: 9, // Spanish mobile
    stripLeadingZero: false,
    placeholder: '612 345 678',
    format: (d) => d.length >= 6
      ? `${d.slice(0, 3)} ${d.slice(3, 6)} ${d.slice(6)}`
      : d.length >= 3
        ? `${d.slice(0, 3)} ${d.slice(3)}`
        : d,
  },
  {
    code: '+39',
    country: 'IT',
    length: 10, // Italian mobile
    stripLeadingZero: false,
    placeholder: '312 345 6789',
    format: (d) => d.length >= 6
      ? `${d.slice(0, 3)} ${d.slice(3, 6)} ${d.slice(6)}`
      : d.length >= 3
        ? `${d.slice(0, 3)} ${d.slice(3)}`
        : d,
  },
  {
    code: '+31',
    country: 'NL',
    length: 9, // Dutch mobile
    stripLeadingZero: true,
    placeholder: '6 12345678',
    format: (d) => d.length >= 1
      ? `${d.slice(0, 1)} ${d.slice(1)}`
      : d,
  },
  {
    code: '+46',
    country: 'SE',
    length: 9, // Swedish mobile
    stripLeadingZero: true,
    placeholder: '70 123 45 67',
    format: (d) => d.length >= 5
      ? `${d.slice(0, 2)} ${d.slice(2, 5)} ${d.slice(5, 7)} ${d.slice(7)}`
      : d.length >= 2
        ? `${d.slice(0, 2)} ${d.slice(2)}`
        : d,
  },
  {
    code: '+47',
    country: 'NO',
    length: 8, // Norwegian mobile
    stripLeadingZero: false,
    placeholder: '412 34 567',
    format: (d) => d.length >= 5
      ? `${d.slice(0, 3)} ${d.slice(3, 5)} ${d.slice(5)}`
      : d.length >= 3
        ? `${d.slice(0, 3)} ${d.slice(3)}`
        : d,
  },
  {
    code: '+64',
    country: 'NZ',
    length: 9, // New Zealand mobile (after stripping 0)
    stripLeadingZero: true,
    placeholder: '21 123 4567',
    format: (d) => d.length >= 5
      ? `${d.slice(0, 2)} ${d.slice(2, 5)} ${d.slice(5)}`
      : d.length >= 2
        ? `${d.slice(0, 2)} ${d.slice(2)}`
        : d,
  },
  {
    code: '+65',
    country: 'SG',
    length: 8, // Singapore mobile
    stripLeadingZero: false,
    placeholder: '9123 4567',
    format: (d) => d.length >= 4
      ? `${d.slice(0, 4)} ${d.slice(4)}`
      : d,
  },
  {
    code: '+82',
    country: 'KR',
    length: 10, // South Korean mobile
    stripLeadingZero: true,
    placeholder: '10 1234 5678',
    format: (d) => d.length >= 6
      ? `${d.slice(0, 2)} ${d.slice(2, 6)} ${d.slice(6)}`
      : d.length >= 2
        ? `${d.slice(0, 2)} ${d.slice(2)}`
        : d,
  },
  {
    code: '+7',
    country: 'RU',
    length: 10, // Russian mobile
    stripLeadingZero: false,
    placeholder: '912 345 67 89',
    format: (d) => d.length >= 7
      ? `${d.slice(0, 3)} ${d.slice(3, 6)} ${d.slice(6, 8)} ${d.slice(8)}`
      : d.length >= 3
        ? `${d.slice(0, 3)} ${d.slice(3)}`
        : d,
  },
  {
    code: '+27',
    country: 'ZA',
    length: 9, // South African mobile
    stripLeadingZero: true,
    placeholder: '82 123 4567',
    format: (d) => d.length >= 5
      ? `${d.slice(0, 2)} ${d.slice(2, 5)} ${d.slice(5)}`
      : d.length >= 2
        ? `${d.slice(0, 2)} ${d.slice(2)}`
        : d,
  },
];

/**
 * Get the configuration for a country code
 */
export function getCountryConfig(code: string): CountryPhoneConfig | undefined {
  return countryConfigs.find(c => c.code === code);
}

/**
 * Process raw phone input to clean digits
 * Handles stripping of leading zeros for countries that require it
 */
export function processPhoneInput(rawInput: string, countryCode: string): string {
  // Extract only digits
  let digits = rawInput.replace(/\D/g, '');

  const config = getCountryConfig(countryCode);
  if (!config) return digits;

  // Strip leading zero if required by the country
  if (config.stripLeadingZero && digits.startsWith('0')) {
    digits = digits.slice(1);
  }

  // Limit to expected length
  return digits.slice(0, config.length);
}

/**
 * Format phone digits for display
 */
export function formatPhoneDisplay(digits: string, countryCode: string): string {
  const config = getCountryConfig(countryCode);
  if (!config) return digits;

  return config.format(digits);
}

/**
 * Get the full E.164 format phone number for 2FA services
 * E.164: +[country code][subscriber number]
 */
export function getE164PhoneNumber(digits: string, countryCode: string): string {
  const config = getCountryConfig(countryCode);
  if (!config) return `${countryCode}${digits}`;

  // Process digits (strip leading zero if needed)
  let processed = digits.replace(/\D/g, '');
  if (config.stripLeadingZero && processed.startsWith('0')) {
    processed = processed.slice(1);
  }

  return `${countryCode}${processed}`;
}

/**
 * Validate phone number length for a country
 */
export function validatePhoneNumber(digits: string, countryCode: string): {
  isValid: boolean;
  error?: string;
  expectedLength?: number;
} {
  const config = getCountryConfig(countryCode);
  if (!config) {
    return { isValid: false, error: 'Unknown country code' };
  }

  // Process digits
  let processed = digits.replace(/\D/g, '');
  if (config.stripLeadingZero && processed.startsWith('0')) {
    processed = processed.slice(1);
  }

  if (processed.length === 0) {
    return { isValid: false, error: 'Phone number is required', expectedLength: config.length };
  }

  if (processed.length < config.length) {
    return {
      isValid: false,
      error: `Phone number too short (${processed.length}/${config.length} digits)`,
      expectedLength: config.length,
    };
  }

  if (processed.length > config.length) {
    return {
      isValid: false,
      error: `Phone number too long (${processed.length}/${config.length} digits)`,
      expectedLength: config.length,
    };
  }

  return { isValid: true };
}

/**
 * Get hint text for phone input based on country
 */
export function getPhoneHint(countryCode: string): string {
  const config = getCountryConfig(countryCode);
  if (!config) return '';

  if (config.stripLeadingZero) {
    return `Enter without leading 0 (e.g., ${config.placeholder})`;
  }

  return `e.g., ${config.placeholder}`;
}
