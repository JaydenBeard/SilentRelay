/**
 * AvatarRing Component
 * 
 * Premium avatar wrapper with gradient accent ring
 * Features:
 * - Static gradient ring
 * - Animated story ring (for users with active stories)
 * - Online status indicator integration
 */

import { forwardRef } from 'react';
import { cn } from '@/lib/utils';
import { Avatar, AvatarImage, AvatarFallback } from '@/components/ui/avatar';

interface AvatarRingProps {
    src?: string;
    alt?: string;
    fallback?: string;
    size?: 'sm' | 'md' | 'lg' | 'xl';
    hasStory?: boolean;
    storyViewed?: boolean;
    isOnline?: boolean;
    className?: string;
}

const sizeClasses = {
    sm: 'h-8 w-8',
    md: 'h-10 w-10',
    lg: 'h-12 w-12',
    xl: 'h-16 w-16',
};

const ringPadding = {
    sm: 'p-[2px]',
    md: 'p-[2px]',
    lg: 'p-[3px]',
    xl: 'p-[3px]',
};

const dotSize = {
    sm: 'w-2.5 h-2.5 border',
    md: 'w-3 h-3 border-2',
    lg: 'w-3.5 h-3.5 border-2',
    xl: 'w-4 h-4 border-2',
};

const dotPosition = {
    sm: 'bottom-0 right-0',
    md: 'bottom-0 right-0',
    lg: '-bottom-0.5 -right-0.5',
    xl: '-bottom-0.5 -right-0.5',
};

export const AvatarRing = forwardRef<HTMLDivElement, AvatarRingProps>(
    ({
        src,
        alt = '',
        fallback,
        size = 'md',
        hasStory = false,
        storyViewed = false,
        isOnline,
        className
    }, ref) => {
        // Determine ring style based on story status
        const showRing = hasStory;
        const ringClass = hasStory
            ? (storyViewed ? 'story-ring-viewed' : 'avatar-ring-story')
            : '';

        return (
            <div ref={ref} className={cn('relative inline-block', className)}>
                {/* Outer ring for stories */}
                <div className={cn(
                    showRing && 'avatar-ring',
                    showRing && ringClass,
                    showRing && ringPadding[size]
                )}>
                    {/* Inner background spacer */}
                    <div className={cn(showRing && 'avatar-ring-inner')}>
                        <Avatar className={cn(sizeClasses[size])}>
                            <AvatarImage src={src} alt={alt} />
                            <AvatarFallback className="text-xs font-medium">
                                {fallback || alt?.charAt(0).toUpperCase() || '?'}
                            </AvatarFallback>
                        </Avatar>
                    </div>
                </div>

                {/* Online status indicator */}
                {isOnline !== undefined && (
                    <span
                        className={cn(
                            'absolute rounded-full border-background',
                            dotSize[size],
                            dotPosition[size],
                            isOnline ? 'pulse-dot' : 'pulse-dot pulse-dot-offline'
                        )}
                    />
                )}
            </div>
        );
    }
);

AvatarRing.displayName = 'AvatarRing';

export default AvatarRing;
