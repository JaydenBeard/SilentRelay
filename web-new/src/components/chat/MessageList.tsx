/**
 * MessageList Component
 *
 * Message list with:
 * - Automatic scroll to bottom on new messages
 * - Scroll preservation on older messages load
 * - Date separators
 * - Empty state with encryption notice
 * - Loading skeletons
 */

import { useRef, useEffect, useCallback, useMemo } from 'react';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Shield } from 'lucide-react';
import { cn } from '@/lib/utils';
import { MessageBubble, MessageBubbleSkeleton, DateSeparator } from './MessageBubble';
import type { Message, FileMetadata } from '@/core/types';

interface MessageListProps {
  /** List of messages */
  messages: Message[];
  /** Current user ID */
  currentUserId: string;
  /** Whether messages are loading */
  isLoading?: boolean;
  /** Callback when reply is triggered on a message */
  onReply?: (message: Message) => void;
  /** Callback when delete is triggered on a message */
  onDelete?: (message: Message) => void;
  /** Callback for file download - accepts FileMetadata from MessageBubble */
  onFileDownload?: (metadata: FileMetadata) => void;
  /** Set of currently downloading file IDs */
  downloadingFiles?: Set<string>;
  /** Callback when a message becomes visible (for read receipts) */
  onMessageVisible?: (message: Message) => void;
  /** Whether read receipts are enabled (used to reset observer when toggled) */
  readReceiptsEnabled?: boolean;
  /** Additional className */
  className?: string;
}

export function MessageList({
  messages,
  currentUserId,
  isLoading,
  onReply,
  onDelete,
  onFileDownload,
  downloadingFiles = new Set(),
  onMessageVisible,
  readReceiptsEnabled = true,
  className,
}: MessageListProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const bottomRef = useRef<HTMLDivElement>(null);
  const prevMessageCountRef = useRef(messages.length);
  const observerRef = useRef<IntersectionObserver | null>(null);
  const observedMessagesRef = useRef<Set<string>>(new Set());
  const pendingVisibleRef = useRef<Map<string, NodeJS.Timeout>>(new Map());

  // Set up IntersectionObserver for message visibility tracking
  // Read receipts are only sent after a message is visible for 500ms
  useEffect(() => {
    if (!onMessageVisible) return;

    // Create a map of message ID to message object for quick lookup
    const messageMap = new Map<string, Message>();
    messages.forEach(m => messageMap.set(m.id, m));

    // Delay starting observation to avoid false triggers during initial render
    const startDelay = setTimeout(() => {
      observerRef.current = new IntersectionObserver(
        (entries) => {
          entries.forEach((entry) => {
            const messageId = entry.target.getAttribute('data-message-id');
            if (!messageId) return;

            if (entry.isIntersecting) {
              // Start a 500ms timer for this message
              if (!pendingVisibleRef.current.has(messageId) && !observedMessagesRef.current.has(messageId)) {
                const timer = setTimeout(() => {
                  const message = messageMap.get(messageId);
                  // Only trigger for incoming messages that haven't been marked as read
                  if (message && message.senderId !== currentUserId && message.status !== 'read') {
                    observedMessagesRef.current.add(messageId);
                    onMessageVisible(message);
                  }
                  pendingVisibleRef.current.delete(messageId);
                }, 500); // Must be visible for 500ms
                pendingVisibleRef.current.set(messageId, timer);
              }
            } else {
              // Message left viewport - cancel pending read receipt
              const timer = pendingVisibleRef.current.get(messageId);
              if (timer) {
                clearTimeout(timer);
                pendingVisibleRef.current.delete(messageId);
              }
            }
          });
        },
        { threshold: 0.8 } // Message must be 80% visible
      );
    }, 300); // Wait 300ms after render before starting to observe

    // Copy ref value for cleanup
    const pendingTimers = pendingVisibleRef.current;

    return () => {
      clearTimeout(startDelay);
      observerRef.current?.disconnect();
      // Clear all pending timers
      pendingTimers.forEach(timer => clearTimeout(timer));
      pendingTimers.clear();
    };
  }, [messages, currentUserId, onMessageVisible]);

  // Reset observed messages when conversation changes OR when read receipts are enabled
  // This allows re-evaluation of visible messages after turning read receipts back on
  useEffect(() => {
    observedMessagesRef.current.clear();
    const pendingTimers = pendingVisibleRef.current;
    pendingTimers.forEach(timer => clearTimeout(timer));
    pendingTimers.clear();
  }, [currentUserId, readReceiptsEnabled]); // Clear when user/conversation changes or read receipts toggled

  // Group messages by date
  const groupedMessages = useMemo(() => {
    const groups: Array<{ date: Date; messages: Message[] }> = [];
    let currentDate: string | null = null;
    let currentGroup: Message[] = [];

    messages.forEach((message) => {
      const messageDate = new Date(message.timestamp).toDateString();

      if (messageDate !== currentDate) {
        if (currentGroup.length > 0 && currentDate) {
          groups.push({
            date: new Date(currentDate),
            messages: currentGroup,
          });
        }
        currentDate = messageDate;
        currentGroup = [message];
      } else {
        currentGroup.push(message);
      }
    });

    // Add the last group
    if (currentGroup.length > 0 && currentDate) {
      groups.push({
        date: new Date(currentDate),
        messages: currentGroup,
      });
    }

    return groups;
  }, [messages]);

  // Scroll to bottom on new messages
  useEffect(() => {
    const newMessageCount = messages.length;
    const prevCount = prevMessageCountRef.current;

    if (newMessageCount > prevCount) {
      const scrollContainer = scrollRef.current;
      if (scrollContainer) {
        const isNearBottom =
          scrollContainer.scrollHeight - scrollContainer.scrollTop - scrollContainer.clientHeight < 100;

        if (isNearBottom || newMessageCount === prevCount + 1) {
          bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
        }
      }
    }

    prevMessageCountRef.current = newMessageCount;
  }, [messages.length]);

  // Initial scroll to bottom
  useEffect(() => {
    if (messages.length > 0 && !isLoading) {
      bottomRef.current?.scrollIntoView();
    }
  }, [isLoading, messages.length]);

  // Check if a file is downloading
  const isFileDownloading = useCallback(
    (content: string) => {
      if (!content.startsWith('[FILE:')) return false;
      try {
        const metadata = JSON.parse(content.slice(6, -1));
        return downloadingFiles.has(metadata.mediaId);
      } catch {
        return false;
      }
    },
    [downloadingFiles]
  );

  if (isLoading) {
    return (
      <ScrollArea className={cn('flex-1 p-4', className)}>
        <div className="space-y-4">
          {[...Array(8)].map((_, i) => (
            <MessageBubbleSkeleton key={i} isOwn={i % 3 === 0} />
          ))}
        </div>
      </ScrollArea>
    );
  }

  if (messages.length === 0) {
    return (
      <div className={cn('flex-1 flex items-center justify-center p-8', className)}>
        <EmptyState />
      </div>
    );
  }

  return (
    <ScrollArea
      className={cn('flex-1', className)}
      ref={scrollRef}
      role="log"
      aria-label="Message history"
      aria-live="polite"
      aria-relevant="additions"
    >
      <div className="p-4 space-y-4">
        {/* Screen reader announcement for new messages */}
        <div className="sr-only" aria-live="assertive" aria-atomic="true">
          {messages.length > 0 && `${messages.length} messages in conversation`}
        </div>

        {groupedMessages.map((group) => (
          <div key={group.date.toISOString()} role="group" aria-label={`Messages from ${group.date.toLocaleDateString()}`}>
            <DateSeparator date={group.date} />
            <div className="space-y-1 mt-2" role="list">
              {group.messages.map((message, idx) => {
                const isOwn = message.senderId === currentUserId;
                const isDownloading = isFileDownloading(message.content);
                const prevMessage = idx > 0 ? group.messages[idx - 1] : null;
                const showSpacing = !prevMessage || prevMessage.senderId !== message.senderId;

                return (
                  <div
                    key={message.id}
                    id={`message-${message.id}`}
                    data-message-id={message.id}
                    ref={(el) => {
                      // Observe incoming messages for read receipts
                      if (el && !isOwn && observerRef.current) {
                        observerRef.current.observe(el);
                      }
                    }}
                    className={cn(showSpacing && idx > 0 ? 'pt-2' : '')}
                    role="listitem"
                    aria-label={`${isOwn ? 'You' : 'Contact'} said: ${message.content.startsWith('[FILE:') ? 'File attachment' : message.content}`}
                    tabIndex={0}
                  >
                    <MessageBubble
                      message={message}
                      isOwn={isOwn}
                      onReply={onReply}
                      onDelete={isOwn ? onDelete : undefined}
                      onFileDownload={onFileDownload}
                      isDownloading={isDownloading}
                      showReadReceipts={readReceiptsEnabled}
                    />
                  </div>
                );
              })}
            </div>
          </div>
        ))}
        <div ref={bottomRef} />
      </div>
    </ScrollArea>
  );
}

/**
 * Empty state with encryption notice
 */
function EmptyState() {
  return (
    <div className="text-center max-w-xs mx-auto">
      <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center mx-auto mb-4">
        <Shield className="h-8 w-8 text-primary" />
      </div>
      <p className="text-foreground-secondary font-medium">
        Messages are end-to-end encrypted
      </p>
      <p className="text-sm text-foreground-muted mt-2">
        No one outside of this chat, not even SilentRelay, can read or listen to them.
      </p>
    </div>
  );
}

/**
 * Hook to scroll to replied message
 */
export function useScrollToMessage() {
  const containerRef = useRef<HTMLDivElement>(null);

  const scrollToMessage = useCallback((messageId: string) => {
    const element = document.getElementById(`message-${messageId}`);
    if (element && containerRef.current) {
      element.scrollIntoView({ behavior: 'smooth', block: 'center' });
      // Highlight the message briefly
      element.classList.add('bg-primary/10');
      setTimeout(() => {
        element.classList.remove('bg-primary/10');
      }, 2000);
    }
  }, []);

  return { containerRef, scrollToMessage };
}