/**
 * ChatHeader Component - Premium Design
 *
 * Adaptive chat header with:
 * - Glass morphism effect
 * - Compact mobile design
 * - Back navigation button on mobile
 * - Contact avatar and name
 * - Online status / typing indicator
 * - Premium action buttons
 */

import { memo, useMemo } from 'react';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Button } from '@/components/ui/button';
import {
  ChevronLeft,
  Phone,
  Video,
  MoreVertical,
  Shield,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { useIsMobile } from '@/hooks/useViewport';

interface ChatHeaderProps {
  /** Recipient name */
  recipientName: string;
  /** Recipient avatar URL */
  recipientAvatar?: string;
  /** Whether recipient is online */
  isOnline: boolean;
  /** Last seen timestamp */
  lastSeen?: number;
  /** Whether recipient is typing */
  isTyping?: boolean;
  /** Whether this is an encrypted conversation */
  isEncrypted?: boolean;
  /** Callback for back button (mobile only) */
  onBack?: () => void;
  /** Callback for voice call */
  onVoiceCall?: () => void;
  /** Callback for video call */
  onVideoCall?: () => void;
  /** Callback for more options menu */
  onMoreOptions?: () => void;
  /** Additional className */
  className?: string;
}

export const ChatHeader = memo(function ChatHeader({
  recipientName,
  recipientAvatar,
  isOnline,
  lastSeen,
  isTyping,
  isEncrypted = true,
  onBack,
  onVoiceCall,
  onVideoCall,
  onMoreOptions,
  className,
}: ChatHeaderProps) {
  const isMobile = useIsMobile();

  // Format status text
  // If we have no presence data (privacy hidden), show nothing
  const statusText = useMemo(() => {
    if (isTyping) return 'typing...';
    if (isOnline) return 'online';
    if (lastSeen) return `last seen ${formatLastSeen(lastSeen)}`;
    return ''; // No data = privacy is hidden, show nothing
  }, [isTyping, isOnline, lastSeen]);

  // Get initials for avatar
  const initials = useMemo(() => {
    const name = recipientName.replace(/^@/, '');
    return name.charAt(0).toUpperCase();
  }, [recipientName]);

  return (
    <header
      className={cn(
        // Base styles
        'flex items-center justify-between relative',
        // Glass effect background
        'bg-background/80 backdrop-blur-xl',
        // Border - subtle gradient
        'border-b border-border/50',
        // Height and padding
        isMobile ? 'h-14 px-2 pt-safe' : 'h-16 px-4',
        className
      )}
    >
      {/* Subtle gradient overlay */}
      <div className="absolute inset-0 bg-gradient-to-r from-primary/[0.02] via-transparent to-primary/[0.02] pointer-events-none" />

      {/* Left section: back button + avatar + info */}
      <div className="flex items-center gap-2 min-w-0 flex-1 relative">
        {/* Back button (mobile only) */}
        {isMobile && onBack && (
          <Button
            variant="ghost"
            size="icon"
            className="touch-target flex-shrink-0 -ml-1 rounded-xl hover:bg-foreground/5"
            onClick={onBack}
            aria-label="Go back to conversations"
          >
            <ChevronLeft className="h-6 w-6" />
          </Button>
        )}

        {/* Avatar with online indicator */}
        <div className="relative flex-shrink-0">
          <Avatar className={cn(
            'ring-2 ring-background shadow-sm',
            isMobile ? 'h-9 w-9' : 'h-10 w-10'
          )}>
            <AvatarImage src={recipientAvatar} alt="" />
            <AvatarFallback className="bg-gradient-to-br from-background-tertiary to-background-secondary text-foreground-secondary font-medium">
              {initials}
            </AvatarFallback>
          </Avatar>
          {isOnline && (
            <span
              className={cn(
                'absolute bottom-0 right-0 rounded-full bg-emerald-500 border-2 border-background',
                'shadow-sm shadow-emerald-500/30',
                isMobile ? 'w-2.5 h-2.5' : 'w-3 h-3'
              )}
              aria-label="Online"
            />
          )}
        </div>

        {/* Name and status */}
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <h2
              className={cn(
                'font-semibold truncate',
                isMobile ? 'text-sm' : 'text-base'
              )}
            >
              {recipientName}
            </h2>
            {isEncrypted && (
              <div className="flex-shrink-0 p-0.5 rounded bg-primary/10">
                <Shield
                  className="h-3 w-3 text-primary"
                  aria-label="End-to-end encrypted"
                />
              </div>
            )}
          </div>
          <p
            className={cn(
              'truncate',
              isMobile ? 'text-xs' : 'text-sm',
              isTyping
                ? 'text-primary font-medium'
                : isOnline
                  ? 'text-emerald-500'
                  : 'text-foreground-muted'
            )}
          >
            {statusText}
          </p>
        </div>
      </div>

      {/* Right section: action buttons */}
      <div className="flex items-center gap-1 flex-shrink-0 relative">
        <Button
          variant="ghost"
          size="icon"
          className="h-9 w-9 rounded-xl hover:bg-foreground/5 transition-colors"
          onClick={onVoiceCall}
          aria-label="Voice call"
        >
          <Phone className={cn(
            'text-foreground-muted',
            isMobile ? 'h-5 w-5' : 'h-4 w-4'
          )} />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="h-9 w-9 rounded-xl hover:bg-foreground/5 transition-colors"
          onClick={onVideoCall}
          aria-label="Video call"
        >
          <Video className={cn(
            'text-foreground-muted',
            isMobile ? 'h-5 w-5' : 'h-4 w-4'
          )} />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="h-9 w-9 rounded-xl hover:bg-foreground/5 transition-colors"
          onClick={onMoreOptions}
          aria-label="More options"
        >
          <MoreVertical className={cn(
            'text-foreground-muted',
            isMobile ? 'h-5 w-5' : 'h-4 w-4'
          )} />
        </Button>
      </div>
    </header>
  );
});

/**
 * Format last seen timestamp
 */
function formatLastSeen(timestamp: number): string {
  const now = Date.now();
  const diff = now - timestamp;
  const minutes = Math.floor(diff / 60000);
  const hours = Math.floor(diff / 3600000);
  const days = Math.floor(diff / 86400000);

  if (minutes < 1) return 'just now';
  if (minutes < 60) return `${minutes}m ago`;
  if (hours < 24) return `${hours}h ago`;
  if (days < 7) return `${days}d ago`;

  return new Date(timestamp).toLocaleDateString(undefined, {
    month: 'short',
    day: 'numeric',
  });
}

/**
 * Skeleton loading state for chat header
 */
export function ChatHeaderSkeleton({ isMobile }: { isMobile?: boolean }) {
  return (
    <div
      className={cn(
        'flex items-center gap-3 bg-background/80 backdrop-blur-xl border-b border-border/50',
        isMobile ? 'h-14 px-2 pt-safe' : 'h-16 px-4'
      )}
    >
      {isMobile && <div className="skeleton h-9 w-9 rounded-xl" />}
      <div className="skeleton h-10 w-10 rounded-full" />
      <div className="flex-1 space-y-2">
        <div className="skeleton h-4 w-32 rounded-md" />
        <div className="skeleton h-3 w-16 rounded-md" />
      </div>
    </div>
  );
}
