/**
 * Safety Number Banner
 *
 * Shows a warning when a contact's identity key has changed.
 * This alerts users that the contact may be on a new device or
 * potentially indicates a security issue.
 */

import { useState } from 'react';
import { AlertTriangle, X, Shield } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

interface SafetyNumberBannerProps {
    contactName: string;
    onVerify?: () => void;
    onDismiss?: () => void;
    className?: string;
}

export function SafetyNumberBanner({
    contactName,
    onVerify,
    onDismiss,
    className,
}: SafetyNumberBannerProps) {
    const [dismissed, setDismissed] = useState(false);

    if (dismissed) return null;

    const handleDismiss = () => {
        setDismissed(true);
        onDismiss?.();
    };

    return (
        <div
            className={cn(
                'flex items-center gap-3 px-4 py-3 bg-warning/10 border-b border-warning/20',
                className
            )}
            role="alert"
        >
            <AlertTriangle className="h-5 w-5 text-warning flex-shrink-0" />

            <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-warning">
                    Security code changed
                </p>
                <p className="text-xs text-foreground-muted mt-0.5">
                    {contactName}'s security code has changed. Verify it's really them.
                </p>
            </div>

            <div className="flex items-center gap-2 flex-shrink-0">
                {onVerify && (
                    <Button
                        variant="ghost"
                        size="sm"
                        onClick={onVerify}
                        className="text-warning hover:text-warning hover:bg-warning/10"
                    >
                        <Shield className="h-4 w-4 mr-1" />
                        Verify
                    </Button>
                )}

                <button
                    onClick={handleDismiss}
                    className="p-1 rounded-full hover:bg-warning/10 text-foreground-muted"
                    aria-label="Dismiss warning"
                >
                    <X className="h-4 w-4" />
                </button>
            </div>
        </div>
    );
}

/**
 * Safety Number Dialog
 *
 * Shows the full safety number for verification.
 * Users can compare this with their contact in person or via secure channel.
 */
interface SafetyNumberDialogProps {
    isOpen: boolean;
    onClose: () => void;
    contactName: string;
    yourNumber: string;
    theirNumber: string;
    onMarkVerified?: () => void;
}

export function SafetyNumberDialog({
    isOpen,
    onClose,
    contactName,
    yourNumber,
    theirNumber,
    onMarkVerified,
}: SafetyNumberDialogProps) {
    if (!isOpen) return null;

    // Format the safety number for display (groups of 5 digits)
    const formatNumber = (num: string) => {
        const cleaned = num.replace(/\s/g, '');
        return cleaned.match(/.{1,5}/g)?.join(' ') || num;
    };

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
            {/* Backdrop */}
            <div
                className="absolute inset-0 bg-black/50"
                onClick={onClose}
            />

            {/* Dialog */}
            <div className="relative bg-background rounded-2xl shadow-xl max-w-md w-full mx-4 p-6">
                <h2 className="text-xl font-bold mb-4">Verify Safety Number</h2>

                <p className="text-sm text-foreground-muted mb-6">
                    Compare these numbers with {contactName} in person or via another
                    secure channel. If they match, your messages are end-to-end encrypted.
                </p>

                <div className="space-y-4 mb-6">
                    <div className="p-4 bg-background-secondary rounded-xl">
                        <p className="text-xs text-foreground-muted mb-2">Your safety number:</p>
                        <p className="font-mono text-sm tracking-wide">
                            {formatNumber(yourNumber)}
                        </p>
                    </div>

                    <div className="p-4 bg-background-secondary rounded-xl">
                        <p className="text-xs text-foreground-muted mb-2">{contactName}'s safety number:</p>
                        <p className="font-mono text-sm tracking-wide">
                            {formatNumber(theirNumber)}
                        </p>
                    </div>
                </div>

                <div className="flex gap-3">
                    <Button
                        variant="outline"
                        className="flex-1"
                        onClick={onClose}
                    >
                        Close
                    </Button>
                    {onMarkVerified && (
                        <Button
                            className="flex-1"
                            onClick={() => {
                                onMarkVerified();
                                onClose();
                            }}
                        >
                            Mark as Verified
                        </Button>
                    )}
                </div>
            </div>
        </div>
    );
}
