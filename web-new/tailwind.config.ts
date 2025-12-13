import type { Config } from 'tailwindcss';

const config: Config = {
  darkMode: 'class',
  content: [
    './index.html',
    './src/**/*.{js,ts,jsx,tsx}',
  ],
  theme: {
    extend: {
      colors: {
        // Primary backgrounds - deep navy tones
        background: {
          DEFAULT: 'hsl(222, 47%, 6%)',
          secondary: 'hsl(222, 35%, 10%)',
          tertiary: 'hsl(222, 30%, 14%)',
        },
        // Foreground text - improved contrast for WCAG AA compliance
        foreground: {
          DEFAULT: 'hsl(210, 40%, 98%)',
          secondary: 'hsl(215, 20%, 70%)',  // Improved from 65%
          muted: 'hsl(215, 15%, 55%)',      // Improved from 45% for 4.5:1 contrast
        },
        // Primary accent - teal
        primary: {
          DEFAULT: 'hsl(168, 84%, 51%)',
          hover: 'hsl(168, 84%, 45%)',
          foreground: 'hsl(222, 47%, 6%)',
        },
        // Secondary accent - purple
        accent: {
          DEFAULT: 'hsl(262, 83%, 58%)',
          hover: 'hsl(262, 83%, 52%)',
        },
        // Status colors
        success: {
          DEFAULT: 'hsl(142, 71%, 45%)',
        },
        warning: {
          DEFAULT: 'hsl(38, 92%, 50%)',
        },
        destructive: {
          DEFAULT: 'hsl(0, 84%, 60%)',
          foreground: 'hsl(210, 40%, 98%)',
        },
        // UI elements
        border: 'hsl(222, 20%, 18%)',
        input: 'hsl(222, 20%, 18%)',
        ring: 'hsl(168, 84%, 51%)',
        // Chat-specific
        message: {
          outgoing: 'hsl(168, 84%, 35%)',
          incoming: 'hsl(222, 30%, 18%)',
        },
      },
      borderRadius: {
        lg: '12px',
        md: '8px',
        sm: '4px',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
        mono: ['JetBrains Mono', 'Menlo', 'monospace'],
      },
      fontSize: {
        '2xs': ['0.625rem', { lineHeight: '0.875rem' }],
      },
      animation: {
        'fade-in': 'fadeIn 0.2s ease-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'slide-down': 'slideDown 0.3s ease-out',
        'pulse-soft': 'pulseSoft 2s ease-in-out infinite',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        slideDown: {
          '0%': { opacity: '0', transform: 'translateY(-10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        pulseSoft: {
          '0%, 100%': { opacity: '1' },
          '50%': { opacity: '0.7' },
        },
      },
    },
  },
  plugins: [],
};

export default config;
