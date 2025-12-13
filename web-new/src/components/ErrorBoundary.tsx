/**
 * Comprehensive Error Boundary Component
 *
 * Provides graceful error handling with:
 * - Fallback UI for different error types
 * - Error reporting and logging
 * - Recovery options
 * - Development vs production modes
 */

import React, { Component, ErrorInfo, ReactNode } from 'react';
import { AlertTriangle, RefreshCw, Home, Bug } from 'lucide-react';
import { Button } from './ui/button';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
  showReportButton?: boolean;
}

interface State {
  hasError: boolean;
  error: Error | null;
  errorInfo: ErrorInfo | null;
  errorId: string | null;
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null,
      errorId: null,
    };
  }

  static getDerivedStateFromError(error: Error): Partial<State> {
    return {
      hasError: true,
      error,
      errorId: `error-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
    };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    this.setState({
      error,
      errorInfo,
    });

    // Log error
    console.error('ErrorBoundary caught an error:', error, errorInfo);

    // Call custom error handler if provided
    if (this.props.onError) {
      this.props.onError(error, errorInfo);
    }

    // Report to error tracking service (if available)
    this.reportError(error, errorInfo);
  }

  private reportError = (error: Error, errorInfo: ErrorInfo) => {
    // In a real app, you would send this to Sentry, LogRocket, etc.
    const errorReport = {
      message: error.message,
      stack: error.stack,
      componentStack: errorInfo.componentStack,
      timestamp: new Date().toISOString(),
      userAgent: navigator.userAgent,
      url: window.location.href,
      errorId: this.state.errorId,
    };

    // For now, just log to console in development
    if (process.env.NODE_ENV === 'development') {
      console.error('Error Report:', errorReport);
    }

    // TODO: Send to error tracking service
    // errorTrackingService.report(errorReport);
  };

  private handleRetry = () => {
    this.setState({
      hasError: false,
      error: null,
      errorInfo: null,
      errorId: null,
    });
  };

  private handleReload = () => {
    window.location.reload();
  };

  private handleGoHome = () => {
    window.location.href = '/';
  };

  private handleReportBug = () => {
    const errorId = this.state.errorId;
    const subject = `Bug Report: Application Error ${errorId}`;
    const body = `Error ID: ${errorId}\n\nPlease describe what you were doing when this error occurred:\n\n`;

    // Open email client or bug reporting system
    window.open(`mailto:support@silentrelay.com?subject=${encodeURIComponent(subject)}&body=${encodeURIComponent(body)}`);
  };

  render() {
    if (this.state.hasError) {
      // Custom fallback UI
      if (this.props.fallback) {
        return this.props.fallback;
      }

      // Default error UI
      return <ErrorFallback
        error={this.state.error}
        errorId={this.state.errorId}
        onRetry={this.handleRetry}
        onReload={this.handleReload}
        onGoHome={this.handleGoHome}
        onReportBug={this.props.showReportButton ? this.handleReportBug : undefined}
      />;
    }

    return this.props.children;
  }
}

/**
 * Error Fallback UI Component
 */
interface ErrorFallbackProps {
  error: Error | null;
  errorId: string | null;
  onRetry?: () => void;
  onReload?: () => void;
  onGoHome?: () => void;
  onReportBug?: () => void;
}

function ErrorFallback({
  error,
  errorId,
  onRetry,
  onReload,
  onGoHome,
  onReportBug,
}: ErrorFallbackProps) {
  const isDevelopment = process.env.NODE_ENV === 'development';

  return (
    <div className="min-h-screen flex items-center justify-center p-4 bg-background">
      <div className="max-w-md w-full space-y-6 text-center">
        {/* Error Icon */}
        <div className="w-16 h-16 rounded-full bg-destructive/10 flex items-center justify-center mx-auto">
          <AlertTriangle className="h-8 w-8 text-destructive" />
        </div>

        {/* Error Title */}
        <div>
          <h1 className="text-2xl font-bold text-foreground mb-2">
            Something went wrong
          </h1>
          <p className="text-muted-foreground">
            We encountered an unexpected error. Please try again or contact support if the problem persists.
          </p>
        </div>

        {/* Error Details (Development Only) */}
        {isDevelopment && error && (
          <div className="bg-muted p-4 rounded-lg text-left">
            <h3 className="font-semibold text-sm mb-2">Error Details:</h3>
            <p className="text-sm font-mono text-destructive break-all">
              {error.message}
            </p>
            {errorId && (
              <p className="text-xs text-muted-foreground mt-2">
                Error ID: {errorId}
              </p>
            )}
          </div>
        )}

        {/* Action Buttons */}
        <div className="flex flex-col gap-3">
          {onRetry && (
            <Button onClick={onRetry} className="w-full">
              <RefreshCw className="h-4 w-4 mr-2" />
              Try Again
            </Button>
          )}

          <div className="flex gap-2">
            {onReload && (
              <Button variant="outline" onClick={onReload} className="flex-1">
                <RefreshCw className="h-4 w-4 mr-2" />
                Reload Page
              </Button>
            )}

            {onGoHome && (
              <Button variant="outline" onClick={onGoHome} className="flex-1">
                <Home className="h-4 w-4 mr-2" />
                Go Home
              </Button>
            )}
          </div>

          {onReportBug && (
            <Button variant="ghost" onClick={onReportBug} className="w-full">
              <Bug className="h-4 w-4 mr-2" />
              Report Bug
            </Button>
          )}
        </div>

        {/* Support Information */}
        <div className="text-xs text-muted-foreground">
          <p>
            If this error persists, please contact our support team with the error details above.
          </p>
        </div>
      </div>
    </div>
  );
}

/**
 * Hook for handling async errors in functional components
 */
export function useErrorHandler() {
  return (error: Error, errorInfo?: { componentStack?: string }) => {
    // Log the error
    console.error('Async error caught:', error, errorInfo);

    // In a real app, you might want to show a toast notification
    // or send to error tracking service
  };
}

/**
 * Higher-order component for adding error boundaries
 */
export function withErrorBoundary<P extends object>(
  Component: React.ComponentType<P>,
  errorBoundaryProps?: Omit<Props, 'children'>
) {
  const WrappedComponent = (props: P) => (
    <ErrorBoundary {...errorBoundaryProps}>
      <Component {...props} />
    </ErrorBoundary>
  );

  WrappedComponent.displayName = `withErrorBoundary(${Component.displayName || Component.name})`;

  return WrappedComponent;
}

export default ErrorBoundary;