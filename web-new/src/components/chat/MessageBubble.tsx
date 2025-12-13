/**
 * MessageBubble Component - Premium Design
 *
 * Touch-optimized message bubble with:
 * - Swipe-to-reply gesture
 * - Swipe-to-delete/archive gesture
 * - Read receipts indicators
 * - Timestamp display
 * - File message handling
 * - Premium styling with shadows and gradients
 */

import { memo, useMemo, useState, useRef, useCallback } from 'react';
import { Check, Reply, Trash2, Download, Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';
import { useSwipeGesture } from '@/hooks/useSwipeGesture';
import type { Message } from '@/core/types';
import { sanitizeMessageContent, isSafeThumbnailUrl } from '@/core/utils/sanitization';

interface MessageBubbleProps {
  /** Message data */
  message: Message;
  /** Whether this is the current user's message */
  isOwn: boolean;
  /** Callback when reply is triggered */
  onReply?: (message: Message) => void;
  /** Callback when delete is triggered */
  onDelete?: (message: Message) => void;
  /** Callback for file download */
  onFileDownload?: (metadata: FileMetadata) => void;
  /** Whether file is currently downloading */
  isDownloading?: boolean;
  /** Whether to show read receipts (reciprocal privacy: if you hide yours, you can't see others') */
  showReadReceipts?: boolean;
  /** Additional className */
  className?: string;
}

interface FileMetadata {
  fileName: string;
  fileSize: number;
  mimeType: string;
  mediaId: string;
  thumbnail?: string;
  encryptionKey: number[];
  iv: number[];
}

// Check if content is a file message
function isFileMessage(content: string): boolean {
  return content.startsWith('[FILE:') && content.endsWith(']');
}

// Parse file message content
function parseFileMessage(content: string): FileMetadata | null {
  try {
    const jsonStr = content.slice(6, -1);
    return JSON.parse(jsonStr);
  } catch {
    return null;
  }
}

// Format timestamp
function formatMessageTime(timestamp: number): string {
  return new Date(timestamp).toLocaleTimeString([], {
    hour: '2-digit',
    minute: '2-digit',
  });
}

// Format file size
function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

// Get file icon based on mime type
function getFileIcon(mimeType: string): string {
  if (mimeType.startsWith('image/')) return '🖼️';
  if (mimeType.startsWith('audio/')) return '🎵';
  if (mimeType.startsWith('video/')) return '🎬';
  if (mimeType === 'application/pdf') return '📄';
  return '📎';
}

export const MessageBubble = memo(function MessageBubble({
  message,
  isOwn,
  onReply,
  onDelete,
  onFileDownload,
  isDownloading,
  showReadReceipts = true,
  className,
}: MessageBubbleProps) {
  const [_showActions, setShowActions] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  // Parse file metadata if this is a file message
  const fileMetadata = useMemo(() => {
    if (isFileMessage(message.content)) {
      return parseFileMessage(message.content);
    }
    return null;
  }, [message.content]);

  // Swipe gesture for reply (swipe right) and delete (swipe left)
  const { swipeState, handlers } = useSwipeGesture({
    horizontal: true,
    vertical: false,
    threshold: 60,
    maxDistance: 80,
    onSwipeRight: () => {
      if (onReply) {
        onReply(message);
      }
    },
    onSwipeLeft: () => {
      if (isOwn && onDelete) {
        setShowActions(true);
      }
    },
    onSwipeEnd: () => {
      // Reset after a delay
      setTimeout(() => setShowActions(false), 2000);
    },
  });

  // Handle file download
  const handleDownload = useCallback(() => {
    if (fileMetadata && onFileDownload) {
      onFileDownload(fileMetadata);
    }
  }, [fileMetadata, onFileDownload]);

  // Get status indicator matching landing page design
  // Shows "Delivered ✓" or "Read ✓" (with teal check for read)
  // Reciprocal privacy: if showReadReceipts is false, downgrade 'read' to 'delivered'
  const StatusIndicator = useMemo(() => {
    // Apply reciprocal privacy: don't show 'read' if user has read receipts disabled
    const displayStatus = (message.status === 'read' && !showReadReceipts)
      ? 'delivered'
      : message.status;

    switch (displayStatus) {
      case 'sending':
        return (
          <span className="w-3 h-3 rounded-full border border-current border-t-transparent animate-spin inline-block" />
        );
      case 'sent':
        return (
          <span className="flex items-center gap-1 opacity-60">
            <Check className="h-3 w-3" />
          </span>
        );
      case 'delivered':
        return (
          <span className="flex items-center gap-1">
            <span className="text-[10px]">Delivered</span>
            <Check className="h-3 w-3" />
          </span>
        );
      case 'read':
        return (
          <span className="flex items-center gap-1 text-primary">
            <span className="text-[10px]">Read</span>
            <Check className="h-3 w-3" />
          </span>
        );
      case 'failed':
        return <span className="text-destructive text-xs">Failed</span>;
      default:
        return null;
    }
  }, [message.status, showReadReceipts]);

  // Calculate swipe transform
  const swipeTransform = useMemo(() => {
    if (swipeState.isSwiping) {
      // Limit the visual offset
      const offset = Math.min(Math.abs(swipeState.offsetX), 60) * Math.sign(swipeState.offsetX);
      return `translateX(${offset}px)`;
    }
    return 'translateX(0)';
  }, [swipeState]);

  return (
    <div
      ref={containerRef}
      className={cn(
        'swipeable-message relative',
        isOwn ? 'flex justify-end' : 'flex justify-start',
        className
      )}
      {...handlers}
    >
      {/* Reply indicator (shown on swipe right) */}
      {swipeState.direction === 'right' && (
        <div
          className={cn(
            'absolute left-0 top-1/2 -translate-y-1/2 flex items-center justify-center',
            'w-10 h-10 rounded-full bg-background-tertiary',
            'transition-opacity duration-150',
            swipeState.offsetX > 30 ? 'opacity-100' : 'opacity-50'
          )}
        >
          <Reply className="h-5 w-5 text-foreground-muted" />
        </div>
      )}

      {/* Delete indicator (shown on swipe left for own messages) */}
      {swipeState.direction === 'left' && isOwn && (
        <div
          className={cn(
            'absolute right-0 top-1/2 -translate-y-1/2 flex items-center justify-center',
            'w-10 h-10 rounded-full bg-destructive/20',
            'transition-opacity duration-150',
            Math.abs(swipeState.offsetX) > 30 ? 'opacity-100' : 'opacity-50'
          )}
        >
          <Trash2 className="h-5 w-5 text-destructive" />
        </div>
      )}

      {/* Message bubble */}
      <div
        className="transition-transform duration-150 ease-out"
        style={{ transform: swipeTransform }}
      >
        {fileMetadata ? (
          // File message
          <FileMessageContent
            metadata={fileMetadata}
            isOwn={isOwn}
            onDownload={handleDownload}
            isDownloading={isDownloading}
            timestamp={message.timestamp}
            statusIndicator={StatusIndicator}
          />
        ) : (
          // Text message - Premium styling with darker green
          <div className="space-y-1">
            <div
              className={cn(
                'max-w-[280px] sm:max-w-[320px] md:max-w-[400px]',
                'rounded-2xl px-4 py-2.5',
                'shadow-sm',
                isOwn
                  ? 'bg-gradient-to-br from-emerald-600 to-emerald-700 text-white rounded-br-sm shadow-emerald-900/30'
                  : 'bg-background-secondary text-foreground rounded-bl-sm border border-border/50'
              )}
            >
              <p className="text-[15px] leading-relaxed whitespace-pre-wrap break-words">
                {sanitizeMessageContent(message.content)}
              </p>
            </div>
            {/* Timestamp and status below bubble */}
            <div
              className={cn(
                'flex items-center gap-1.5 px-1',
                isOwn ? 'justify-end' : 'justify-start'
              )}
            >
              <span className="text-[10px] tabular-nums text-foreground-muted">
                {formatMessageTime(message.timestamp)}
              </span>
              {isOwn && (
                <span className="text-foreground-muted">{StatusIndicator}</span>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
});

/**
 * File message content component - Premium styling
 */
function FileMessageContent({
  metadata,
  isOwn,
  onDownload,
  isDownloading,
  timestamp,
  statusIndicator,
}: {
  metadata: FileMetadata;
  isOwn: boolean;
  onDownload: () => void;
  isDownloading?: boolean;
  timestamp: number;
  statusIndicator: React.ReactNode;
}) {
  const isImage = metadata.mimeType.startsWith('image/');

  return (
    <div className="space-y-1">
      <button
        onClick={onDownload}
        disabled={isDownloading}
        className={cn(
          'block rounded-2xl overflow-hidden',
          'min-w-[200px] max-w-[280px]',
          'transition-all duration-200 active:scale-[0.98]',
          'shadow-sm',
          isOwn
            ? 'bg-gradient-to-br from-emerald-600 to-emerald-700 shadow-emerald-900/30'
            : 'bg-background-secondary border border-border/50'
        )}
      >
        {/* Thumbnail preview for images */}
        {isImage && metadata.thumbnail && isSafeThumbnailUrl(metadata.thumbnail) && (
          <div className="aspect-video bg-background-tertiary relative">
            <img
              src={metadata.thumbnail}
              alt=""
              className="w-full h-full object-cover"
            />
            <div className="absolute inset-0 flex items-center justify-center bg-black/40 backdrop-blur-[1px]">
              {isDownloading ? (
                <Loader2 className="h-8 w-8 text-white animate-spin" />
              ) : (
                <div className="w-12 h-12 rounded-full bg-white/20 backdrop-blur-sm flex items-center justify-center">
                  <Download className="h-6 w-6 text-white" />
                </div>
              )}
            </div>
          </div>
        )}

        {/* File info */}
        <div className="flex items-center gap-3 p-3">
          <span className="text-2xl">{getFileIcon(metadata.mimeType)}</span>
          <div className="flex-1 min-w-0 text-left">
            <p
              className={cn(
                'text-sm font-medium truncate',
                isOwn ? 'text-white' : 'text-foreground'
              )}
            >
              {sanitizeMessageContent(metadata.fileName)}
            </p>
            <p
              className={cn(
                'text-xs',
                isOwn ? 'text-white/70' : 'text-foreground-muted'
              )}
            >
              {formatFileSize(metadata.fileSize)}
              {isDownloading && ' • Downloading...'}
            </p>
          </div>
          {!isImage && (
            <div className="flex-shrink-0">
              {isDownloading ? (
                <Loader2 className={cn('h-5 w-5 animate-spin', isOwn ? 'text-white/70' : 'text-foreground-muted')} />
              ) : (
                <Download className={cn('h-5 w-5', isOwn ? 'text-white/70' : 'text-foreground-muted')} />
              )}
            </div>
          )}
        </div>
      </button>

      {/* Timestamp and status */}
      <div
        className={cn(
          'flex items-center gap-1 px-1',
          isOwn ? 'justify-end' : 'justify-start'
        )}
      >
        <span className="text-[10px] text-foreground-muted tabular-nums">
          {formatMessageTime(timestamp)}
        </span>
        {isOwn && (
          <span className="text-foreground-muted">{statusIndicator}</span>
        )}
      </div>
    </div>
  );
}

/**
 * Skeleton loading state for message bubble
 */
export function MessageBubbleSkeleton({ isOwn }: { isOwn: boolean }) {
  return (
    <div className={cn('flex', isOwn ? 'justify-end' : 'justify-start')}>
      <div
        className={cn(
          'skeleton rounded-2xl',
          isOwn
            ? 'bg-primary/20 rounded-br-sm'
            : 'bg-background-secondary rounded-bl-sm'
        )}
        style={{
          width: `${Math.random() * 100 + 120}px`,
          height: '44px',
        }}
      />
    </div>
  );
}

/**
 * Date separator component - Premium styling
 */
export function DateSeparator({ date }: { date: Date }) {
  const text = useMemo(() => {
    const today = new Date();
    const yesterday = new Date(today);
    yesterday.setDate(yesterday.getDate() - 1);

    if (date.toDateString() === today.toDateString()) {
      return 'Today';
    }
    if (date.toDateString() === yesterday.toDateString()) {
      return 'Yesterday';
    }
    return date.toLocaleDateString(undefined, {
      weekday: 'long',
      month: 'long',
      day: 'numeric',
    });
  }, [date]);

  return (
    <div className="flex items-center justify-center py-4">
      <div className="flex items-center gap-3">
        <div className="h-px w-8 bg-gradient-to-r from-transparent to-border" />
        <span className="text-[11px] font-medium text-foreground-muted/80 uppercase tracking-wider">
          {text}
        </span>
        <div className="h-px w-8 bg-gradient-to-l from-transparent to-border" />
      </div>
    </div>
  );
}
