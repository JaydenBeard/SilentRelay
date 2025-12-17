/**
 * EmptyState Component
 * 
 * Animated empty state placeholders for lists
 * Features subtle floating animation and optional call-to-action
 */

import { forwardRef } from 'react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import type { LucideIcon } from 'lucide-react';

interface EmptyStateProps {
    icon: LucideIcon;
    title: string;
    description?: string;
    actionLabel?: string;
    onAction?: () => void;
    className?: string;
    animate?: boolean;
}

export const EmptyState = forwardRef<HTMLDivElement, EmptyStateProps>(
    ({ icon: Icon, title, description, actionLabel, onAction, className, animate = true }, ref) => {
        return (
            <div
                ref={ref}
                className={cn(
                    'flex flex-col items-center justify-center p-8 text-center',
                    className
                )}
            >
                {/* Icon with glow and animation */}
                <div className={cn('relative mb-6', animate && 'empty-state-float')}>
                    {/* Glow effect */}
                    <div className="absolute inset-0 bg-primary/20 rounded-2xl blur-2xl scale-150 empty-state-pulse" />

                    {/* Icon container */}
                    <div className="relative w-20 h-20 rounded-2xl bg-gradient-to-br from-background-secondary to-background-tertiary flex items-center justify-center border border-border/50 shadow-lg">
                        <Icon className="h-9 w-9 text-foreground-muted" strokeWidth={1.5} />
                    </div>
                </div>

                {/* Title */}
                <h3 className="text-lg font-semibold mb-2">{title}</h3>

                {/* Description */}
                {description && (
                    <p className="text-foreground-muted text-sm max-w-xs mb-6">
                        {description}
                    </p>
                )}

                {/* Action button */}
                {actionLabel && onAction && (
                    <Button
                        onClick={onAction}
                        className="rounded-xl shadow-lg shadow-primary/20 px-6"
                    >
                        {actionLabel}
                    </Button>
                )}
            </div>
        );
    }
);

EmptyState.displayName = 'EmptyState';

export default EmptyState;
