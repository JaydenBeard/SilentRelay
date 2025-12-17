/**
 * PulseDot Component
 * 
 * Online status indicator with optional pulse animation
 * States: online (green pulse), away (yellow pulse), offline (gray, no pulse)
 */

import { forwardRef } from 'react';
import { cn } from '@/lib/utils';

type StatusType = 'online' | 'away' | 'offline';

interface PulseDotProps extends React.HTMLAttributes<HTMLSpanElement> {
    status?: StatusType;
    size?: 'sm' | 'md' | 'lg';
    pulse?: boolean;
}

const sizeClasses = {
    sm: 'w-2 h-2',
    md: 'w-3 h-3',
    lg: 'w-4 h-4',
};

export const PulseDot = forwardRef<HTMLSpanElement, PulseDotProps>(
    ({ status = 'online', size = 'md', pulse = true, className, ...props }, ref) => {
        return (
            <span
                ref={ref}
                className={cn(
                    'pulse-dot',
                    sizeClasses[size],
                    status === 'offline' && 'pulse-dot-offline',
                    status === 'away' && 'pulse-dot-away',
                    !pulse && '[&::before]:hidden',
                    className
                )}
                aria-label={`Status: ${status}`}
                {...props}
            />
        );
    }
);

PulseDot.displayName = 'PulseDot';

export default PulseDot;
