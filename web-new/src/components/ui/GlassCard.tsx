/**
 * GlassCard Component
 * 
 * Premium glassmorphism card with blur backdrop
 * Available in three intensity levels: subtle, default, premium
 */

import { forwardRef } from 'react';
import { cn } from '@/lib/utils';

interface GlassCardProps extends React.HTMLAttributes<HTMLDivElement> {
    variant?: 'subtle' | 'default' | 'premium';
    hover?: boolean;
    press?: boolean;
}

export const GlassCard = forwardRef<HTMLDivElement, GlassCardProps>(
    ({ className, variant = 'default', hover = false, press = false, children, ...props }, ref) => {
        return (
            <div
                ref={ref}
                className={cn(
                    // Base glass styles
                    variant === 'subtle' && 'glass-subtle',
                    variant === 'default' && 'glass-card',
                    variant === 'premium' && 'glass-premium',
                    // Interactive states
                    hover && 'micro-lift',
                    press && 'micro-press',
                    className
                )}
                {...props}
            >
                {children}
            </div>
        );
    }
);

GlassCard.displayName = 'GlassCard';

export default GlassCard;
