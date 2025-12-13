import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import * as Sentry from '@sentry/react';
import App from './App';
import './index.css';

// Initialize Sentry error tracking
Sentry.init({
  dsn: import.meta.env.VITE_SENTRY_DSN || 'https://976166aae109e88f4f927ff06341c459@o4510516725088256.ingest.de.sentry.io/4510516761133136',
  environment: import.meta.env.MODE,

  // Performance monitoring (optional)
  tracesSampleRate: import.meta.env.PROD ? 0.1 : 1.0, // 10% in prod, 100% in dev

  // Only send errors in production
  enabled: import.meta.env.PROD,

  // Filter out common noise
  ignoreErrors: [
    'ResizeObserver loop limit exceeded',
    'ResizeObserver loop completed with undelivered notifications',
    'Non-Error promise rejection captured',
  ],

  // Attach user info when available
  beforeSend(event) {
    // You can modify or filter events here
    return event;
  },
});

// Expose Sentry globally for debugging/verification
// eslint-disable-next-line @typescript-eslint/no-explicit-any
(window as any).Sentry = Sentry;

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <Sentry.ErrorBoundary fallback={<div>Something went wrong. Please refresh the page.</div>}>
      <BrowserRouter
        future={{
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        }}
      >
        <App />
      </BrowserRouter>
    </Sentry.ErrorBoundary>
  </StrictMode>
);

