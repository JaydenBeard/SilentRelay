/**
 * ConversationItem Component - Premium Design
 *
 * A touch-optimized conversation list item with:
 * - Avatar with online indicator
 * - Truncated message preview (max 2 lines)
 * - Relative timestamps
 * - Unread badge
 * - Premium hover effects
 */

import { memo, useMemo, useCallback } from 'react';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { cn } from '@/lib/utils';
import { Paperclip, Pin, Trash2 } from 'lucide-react';
import { useChatStore } from '@/core/store/chatStore';
import type { Conversation, Message } from '@/core/types';

// Stable empty array to prevent infinite re-renders in zustand selectors
const EMPTY_MESSAGES: Message[] = [];

interface ConversationItemProps {
  conversation: Conversation;
  isActive: boolean;
  onClick: () => void;
  onDelete?: (conversationId: string) => void;
}

// Relative time formatter
function formatRelativeTime(timestamp: number): string {
  const now = Date.now();
  const diff = now - timestamp;
  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (seconds < 60) return 'now';
  if (minutes < 60) return `${minutes}m`;
  if (hours < 24) return `${hours}h`;
  if (days === 1) return 'Yesterday';
  if (days < 7) {
    const dayNames = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
    return dayNames[new Date(timestamp).getDay()];
  }

  return new Date(timestamp).toLocaleDateString(undefined, {
    month: 'short',
    day: 'numeric',
  });
}

// Truncate text with ellipsis
function truncateText(text: string, maxLength: number = 50): string {
  if (text.length <= maxLength) return text;
  return text.slice(0, maxLength).trim() + '…';
}

// Check if content is a file message
function isFileMessage(content: string): boolean {
  return content.startsWith('[FILE:') && content.endsWith(']');
}

// Extract file name from file message
function extractFileName(content: string): string {
  try {
    const jsonStr = content.slice(6, -1);
    const metadata = JSON.parse(jsonStr);
    return metadata.fileName || 'File';
  } catch {
    return 'File';
  }
}

export const ConversationItem = memo(function ConversationItem({
  conversation,
  isActive,
  onClick,
  onDelete,
}: ConversationItemProps) {
  const { lastMessage, recipientName, recipientAvatar, unreadCount, isOnline, isPinned, recipientId } = conversation;

  // IMPORTANT: Use a stable empty array reference to prevent infinite re-renders
  // The || [] pattern creates a new array reference each render, breaking React's memoization
  const conversationMessages = useChatStore((state) => state.messages[conversation.id]);
  const messages = conversationMessages ?? EMPTY_MESSAGES;

  // Only show online status if we've received a reply (meaning they accepted our message)
  // This prevents seeing online status before they've engaged with us
  const hasReceivedReply = useMemo(() =>
    messages.some(m => m.senderId === recipientId),
    [messages, recipientId]
  );
  const showOnline = hasReceivedReply && isOnline;

  // Handle delete click - prevent propagation to avoid selecting the conversation
  const handleDelete = useCallback((e: React.MouseEvent) => {
    e.stopPropagation();
    onDelete?.(conversation.recipientId);
  }, [onDelete, conversation.recipientId]);

  // Format the last message preview
  const messagePreview = useMemo(() => {
    if (!lastMessage) return 'No messages yet';

    if (isFileMessage(lastMessage.content)) {
      return extractFileName(lastMessage.content);
    }

    return truncateText(lastMessage.content);
  }, [lastMessage]);

  // Format timestamp
  const timestamp = useMemo(() => {
    if (!lastMessage) return '';
    return formatRelativeTime(lastMessage.timestamp);
  }, [lastMessage]);

  // Get initials for avatar fallback
  const initials = useMemo(() => {
    const name = recipientName.replace(/^@/, '');
    return name.charAt(0).toUpperCase();
  }, [recipientName]);

  return (
    <button
      onClick={onClick}
      aria-label={`Chat with ${recipientName}${unreadCount > 0 ? `, ${unreadCount} unread messages` : ''}`}
      aria-current={isActive ? 'true' : undefined}
      className={cn(
        // Base styles
        'conversation-item w-full relative',
        'flex items-center gap-3',
        'px-3 py-2.5',
        'text-left',
        'rounded-xl',
        'transition-all duration-200',
        'focus-ring',
        'group',
        // Touch target
        'min-h-[68px]',
        // Hover state - subtle glow
        'hover:bg-background-secondary/80',
        // Active state - more prominent
        isActive && 'bg-primary/10 hover:bg-primary/15 shadow-sm',
        // Unread state
        unreadCount > 0 && 'font-medium'
      )}
    >
      {/* Avatar with online indicator */}
      <div className="relative flex-shrink-0">
        <Avatar className={cn(
          'h-12 w-12 transition-transform duration-200',
          'ring-2 ring-transparent',
          isActive && 'ring-primary/20',
          'group-hover:scale-[1.02]'
        )}>
          <AvatarImage src={recipientAvatar} alt="" />
          <AvatarFallback className={cn(
            'text-foreground-secondary font-medium',
            isActive
              ? 'bg-primary/10 text-primary'
              : 'bg-gradient-to-br from-background-tertiary to-background-secondary'
          )}>
            {initials}
          </AvatarFallback>
        </Avatar>
        {showOnline && (
          <span
            className="absolute -bottom-0.5 -right-0.5 w-3.5 h-3.5 rounded-full bg-emerald-500 border-2 border-background shadow-sm shadow-emerald-500/30"
            aria-label="Online"
          />
        )}
      </div>

      {/* Content area */}
      <div className="flex-1 min-w-0">
        {/* Top row: Name and timestamp/delete */}
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-1.5 min-w-0">
            <span
              className={cn(
                'truncate font-medium',
                isActive ? 'text-primary' : 'text-foreground'
              )}
            >
              {recipientName}
            </span>
            {isPinned && (
              <Pin className="h-3 w-3 text-foreground-muted flex-shrink-0 fill-current" />
            )}
          </div>
          <div className="flex items-center gap-1.5 flex-shrink-0">
            {lastMessage && (
              <span
                className={cn(
                  'text-xs tabular-nums transition-opacity',
                  unreadCount > 0 ? 'text-primary font-medium' : 'text-foreground-muted',
                  // Hide timestamp on hover when delete button shows
                  'group-hover:opacity-0'
                )}
              >
                {timestamp}
              </span>
            )}
            {/* Delete button - shows on hover */}
            {onDelete && (
              <button
                onClick={handleDelete}
                className={cn(
                  'p-1 rounded-md transition-all duration-200',
                  'text-foreground-muted hover:text-destructive hover:bg-destructive/10',
                  // Hidden by default, visible on hover
                  'opacity-0 group-hover:opacity-100',
                  'absolute right-3 top-1/2 -translate-y-1/2'
                )}
                aria-label={`Delete conversation with ${recipientName}`}
              >
                <Trash2 className="h-4 w-4" />
              </button>
            )}
          </div>
        </div>

        {/* Bottom row: Message preview and badge */}
        <div className="flex items-center justify-between gap-2 mt-0.5">
          <span
            className={cn(
              'text-[13px] truncate flex items-center gap-1.5',
              unreadCount > 0 ? 'text-foreground-secondary' : 'text-foreground-muted'
            )}
          >
            {lastMessage && isFileMessage(lastMessage.content) && (
              <Paperclip className="h-3.5 w-3.5 flex-shrink-0" />
            )}
            {messagePreview}
          </span>

          {/* Unread badge - Premium pill style */}
          {unreadCount > 0 && (
            <span className="flex-shrink-0 min-w-[20px] h-5 px-1.5 rounded-full bg-primary text-primary-foreground text-xs font-medium flex items-center justify-center shadow-sm shadow-primary/30">
              {unreadCount > 99 ? '99+' : unreadCount}
            </span>
          )}
        </div>
      </div>
    </button>
  );
});

/**
 * Skeleton loading state for conversation item
 */
export function ConversationItemSkeleton() {
  return (
    <div className="flex items-center gap-3 px-3 py-2.5 min-h-[68px]">
      {/* Avatar skeleton */}
      <div className="skeleton h-12 w-12 rounded-full flex-shrink-0" />

      {/* Content skeleton */}
      <div className="flex-1 min-w-0 space-y-2.5">
        <div className="flex justify-between gap-2">
          <div className="skeleton h-4 w-28 rounded-md" />
          <div className="skeleton h-3 w-10 rounded-md" />
        </div>
        <div className="skeleton h-3.5 w-36 rounded-md" />
      </div>
    </div>
  );
}
