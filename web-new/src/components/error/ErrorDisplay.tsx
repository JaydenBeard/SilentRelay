/**
 * Error Display Components
 *
 * Displays errors to users with error codes for support reference.
 */

import { AlertTriangle, XCircle, AlertCircle, Info, Copy, RefreshCw, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import type { AppError, ErrorSeverity } from '@/lib/errors';
import { useState } from 'react';

interface ErrorDisplayProps {
  error: AppError;
  onDismiss?: () => void;
  onRetry?: () => void;
}

/**
 * Get icon for error severity
 */
function SeverityIcon({ severity, className }: { severity: ErrorSeverity; className?: string }) {
  switch (severity) {
    case 'fatal':
      return <XCircle className={className} />;
    case 'error':
      return <AlertCircle className={className} />;
    case 'warning':
      return <AlertTriangle className={className} />;
    case 'info':
      return <Info className={className} />;
  }
}

/**
 * Get color classes for error severity
 */
function getSeverityColors(severity: ErrorSeverity) {
  switch (severity) {
    case 'fatal':
      return {
        bg: 'bg-red-500/10',
        border: 'border-red-500/30',
        icon: 'text-red-500',
        text: 'text-red-100',
      };
    case 'error':
      return {
        bg: 'bg-destructive/10',
        border: 'border-destructive/30',
        icon: 'text-destructive',
        text: 'text-foreground',
      };
    case 'warning':
      return {
        bg: 'bg-yellow-500/10',
        border: 'border-yellow-500/30',
        icon: 'text-yellow-500',
        text: 'text-foreground',
      };
    case 'info':
      return {
        bg: 'bg-blue-500/10',
        border: 'border-blue-500/30',
        icon: 'text-blue-500',
        text: 'text-foreground',
      };
  }
}

/**
 * Inline error banner for non-fatal errors
 */
export function ErrorBanner({ error, onDismiss, onRetry }: ErrorDisplayProps) {
  const [copied, setCopied] = useState(false);
  const colors = getSeverityColors(error.severity);

  const copyToClipboard = async () => {
    const text = `Error Code: ${error.code}\nReference: ${error.referenceId}\nMessage: ${error.message}`;
    await navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className={`${colors.bg} ${colors.border} border rounded-lg p-4`}>
      <div className="flex items-start gap-3">
        <SeverityIcon severity={error.severity} className={`h-5 w-5 mt-0.5 ${colors.icon}`} />

        <div className="flex-1 min-w-0">
          <p className={`text-sm font-medium ${colors.text}`}>{error.message}</p>

          <div className="mt-2 flex items-center gap-2 text-xs text-foreground-muted">
            <span className="font-mono bg-background/50 px-1.5 py-0.5 rounded">
              {error.code}
            </span>
            <span>|</span>
            <span className="font-mono">
              Ref: {error.referenceId}
            </span>
          </div>

          <div className="mt-3 flex items-center gap-2">
            <Button
              variant="ghost"
              size="sm"
              onClick={copyToClipboard}
              className="h-7 text-xs"
            >
              <Copy className="h-3 w-3 mr-1" />
              {copied ? 'Copied!' : 'Copy Details'}
            </Button>
            {onRetry && (
              <Button
                variant="ghost"
                size="sm"
                onClick={onRetry}
                className="h-7 text-xs"
              >
                <RefreshCw className="h-3 w-3 mr-1" />
                Try Again
              </Button>
            )}
          </div>
        </div>

        {onDismiss && (
          <button
            onClick={onDismiss}
            className="text-foreground-muted hover:text-foreground transition-colors"
          >
            <X className="h-4 w-4" />
          </button>
        )}
      </div>
    </div>
  );
}

/**
 * Full-screen error display for fatal errors
 */
export function FatalErrorScreen({ error, onRetry }: ErrorDisplayProps) {
  const [copied, setCopied] = useState(false);

  const copyToClipboard = async () => {
    const text = `Error Code: ${error.code}\nReference: ${error.referenceId}\nMessage: ${error.message}\nTimestamp: ${error.timestamp.toISOString()}`;
    await navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-6">
      <div className="max-w-md w-full text-center">
        {/* Icon */}
        <div className="inline-flex items-center justify-center w-20 h-20 rounded-full bg-red-500/10 mb-6">
          <XCircle className="h-10 w-10 text-red-500" />
        </div>

        {/* Title */}
        <h1 className="text-2xl font-bold text-foreground mb-2">
          Something went wrong
        </h1>

        {/* Message */}
        <p className="text-foreground-secondary mb-6">
          {error.message}
        </p>

        {/* Error Details Card */}
        <div className="bg-background-secondary border border-border rounded-xl p-4 mb-6 text-left">
          <h2 className="text-sm font-medium text-foreground mb-3">Error Details</h2>

          <div className="space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-foreground-muted">Error Code</span>
              <span className="font-mono text-foreground">{error.code}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-foreground-muted">Reference ID</span>
              <span className="font-mono text-foreground">{error.referenceId}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-foreground-muted">Time</span>
              <span className="text-foreground">
                {error.timestamp.toLocaleTimeString()}
              </span>
            </div>
          </div>

          <div className="mt-4 pt-4 border-t border-border">
            <p className="text-xs text-foreground-muted">
              If you need support, please provide the error code and reference ID above.
            </p>
          </div>
        </div>

        {/* Actions */}
        <div className="flex flex-col gap-3">
          {onRetry && (
            <Button onClick={onRetry} className="w-full">
              <RefreshCw className="h-4 w-4 mr-2" />
              Try Again
            </Button>
          )}

          <Button
            variant="outline"
            onClick={copyToClipboard}
            className="w-full"
          >
            <Copy className="h-4 w-4 mr-2" />
            {copied ? 'Copied!' : 'Copy Error Details'}
          </Button>

          <Button
            variant="ghost"
            onClick={() => window.location.reload()}
            className="w-full"
          >
            Refresh Page
          </Button>
        </div>

        {/* Support Link */}
        <p className="mt-6 text-xs text-foreground-muted">
          Need help?{' '}
          <a href="mailto:support@example.com" className="text-primary hover:underline">
            Contact Support
          </a>
        </p>
      </div>
    </div>
  );
}

/**
 * Compact error message for inline display
 */
export function InlineError({ error }: { error: AppError }) {
  return (
    <div className="text-sm text-destructive flex items-center gap-2">
      <AlertCircle className="h-4 w-4 flex-shrink-0" />
      <span>{error.message}</span>
      <span className="text-xs text-foreground-muted font-mono">({error.code})</span>
    </div>
  );
}

/**
 * Simple error message with code (for form fields)
 */
export function ErrorMessage({ message, code }: { message: string; code?: string }) {
  return (
    <p className="text-sm text-destructive">
      {message}
      {code && <span className="text-xs text-foreground-muted ml-1">({code})</span>}
    </p>
  );
}
