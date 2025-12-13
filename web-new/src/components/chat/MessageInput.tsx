/**
 * MessageInput Component - Premium Design
 *
 * Intelligent message input area with:
 * - Virtual Viewport API for keyboard handling
 * - Auto-growing textarea with max height
 * - File attachment support
 * - Emoji picker
 * - Premium styling with glass effects
 */

import { useState, useRef, useCallback, useEffect, forwardRef } from 'react';
import { Button } from '@/components/ui/button';
import {
  Send,
  Smile,
  Paperclip,
  X,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { useViewport, useIsMobile } from '@/hooks/useViewport';
import { EmojiPicker } from '@/components/ui/EmojiPicker';

interface MessageInputProps {
  /** Current input value */
  value: string;
  /** Callback when value changes */
  onChange: (value: string) => void;
  /** Callback when form is submitted */
  onSubmit: () => void;
  /** Callback when file is selected */
  onFileSelect?: (file: File) => void;
  /** Whether currently sending */
  isSending?: boolean;
  /** Whether input is disabled */
  disabled?: boolean;
  /** Placeholder text */
  placeholder?: string;
  /** File upload component (optional) */
  fileUpload?: React.ReactNode;
  /** Error message to display */
  error?: string | null;
  /** Callback when typing starts/stops */
  onTypingChange?: (isTyping: boolean) => void;
  /** Additional className */
  className?: string;
}

export const MessageInput = forwardRef<HTMLTextAreaElement, MessageInputProps>(
  function MessageInput(
    {
      value,
      onChange,
      onSubmit,
      onFileSelect,
      isSending,
      disabled,
      placeholder = 'Type a message...',
      fileUpload,
      error,
      onTypingChange,
      className,
    },
    ref
  ) {
    const { isKeyboardOpen, keyboardHeight } = useViewport();
    const isMobile = useIsMobile();
    const internalRef = useRef<HTMLTextAreaElement | null>(null);
    const containerRef = useRef<HTMLDivElement>(null);
    const emojiButtonRef = useRef<HTMLButtonElement>(null);
    const [isFocused, setIsFocused] = useState(false);
    const [showEmojiPicker, setShowEmojiPicker] = useState(false);
    const typingTimeoutRef = useRef<NodeJS.Timeout | null>(null);
    const isTypingRef = useRef(false);

    // Combine refs
    const setRefs = useCallback(
      (node: HTMLTextAreaElement | null) => {
        internalRef.current = node;
        if (typeof ref === 'function') {
          ref(node);
        } else if (ref) {
          ref.current = node;
        }
      },
      [ref]
    );

    // Auto-resize textarea
    const adjustTextareaHeight = useCallback(() => {
      const textarea = internalRef.current;
      if (!textarea) return;

      textarea.style.height = 'auto';
      const maxHeight = 120;
      const newHeight = Math.min(textarea.scrollHeight, maxHeight);
      textarea.style.height = `${newHeight}px`;
      textarea.style.overflowY = textarea.scrollHeight > maxHeight ? 'auto' : 'hidden';
    }, []);

    useEffect(() => {
      adjustTextareaHeight();
    }, [value, adjustTextareaHeight]);

    // Handle input change
    const handleChange = useCallback(
      (e: React.ChangeEvent<HTMLTextAreaElement>) => {
        onChange(e.target.value);

        if (onTypingChange) {
          if (typingTimeoutRef.current) {
            clearTimeout(typingTimeoutRef.current);
          }

          if (!isTypingRef.current) {
            isTypingRef.current = true;
            onTypingChange(true);
          }

          typingTimeoutRef.current = setTimeout(() => {
            if (isTypingRef.current) {
              isTypingRef.current = false;
              onTypingChange(false);
            }
          }, 3000);
        }
      },
      [onChange, onTypingChange]
    );

    // Handle key down
    const handleKeyDown = useCallback(
      (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
        if (e.key === 'Enter' && !e.shiftKey) {
          e.preventDefault();
          if (value.trim() && !disabled && !isSending) {
            onSubmit();
            if (typingTimeoutRef.current) {
              clearTimeout(typingTimeoutRef.current);
            }
            isTypingRef.current = false;
            onTypingChange?.(false);
          }
        }
      },
      [value, disabled, isSending, onSubmit, onTypingChange]
    );

    const handleFocus = useCallback(() => setIsFocused(true), []);
    const handleBlur = useCallback(() => setIsFocused(false), []);

    // Handle file input
    const handleFileInput = useCallback(
      (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (file && onFileSelect) {
          onFileSelect(file);
        }
        e.target.value = '';
      },
      [onFileSelect]
    );

    // Cleanup
    useEffect(() => {
      return () => {
        if (typingTimeoutRef.current) {
          clearTimeout(typingTimeoutRef.current);
        }
      };
    }, []);

    // Handle emoji selection
    const handleEmojiSelect = useCallback((emoji: string) => {
      const textarea = internalRef.current;
      if (textarea) {
        // Insert emoji at cursor position
        const start = textarea.selectionStart;
        const end = textarea.selectionEnd;
        const newValue = value.substring(0, start) + emoji + value.substring(end);
        onChange(newValue);

        // Move cursor after emoji
        setTimeout(() => {
          textarea.focus();
          textarea.setSelectionRange(start + emoji.length, start + emoji.length);
        }, 0);
      } else {
        // Fallback: append to end
        onChange(value + emoji);
      }

      // Close picker after selection
      setShowEmojiPicker(false);

      // Trigger typing indicator
      if (onTypingChange && !isTypingRef.current) {
        isTypingRef.current = true;
        onTypingChange(true);
      }
    }, [value, onChange, onTypingChange]);

    // Handle submit
    const handleSubmit = useCallback(() => {
      if (value.trim() && !disabled && !isSending) {
        onSubmit();
        if (typingTimeoutRef.current) {
          clearTimeout(typingTimeoutRef.current);
        }
        isTypingRef.current = false;
        onTypingChange?.(false);
      }
    }, [value, disabled, isSending, onSubmit, onTypingChange]);

    return (
      <div
        ref={containerRef}
        className={cn('relative', className)}
        style={{
          transform: isMobile && isKeyboardOpen ? `translateY(-${keyboardHeight}px)` : undefined,
        }}
      >
        {/* Subtle top border/gradient */}
        <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-border to-transparent" />

        {/* Error message */}
        {error && (
          <div className="mx-4 mb-2 text-sm text-destructive bg-destructive/10 px-3 py-2 rounded-xl flex items-center justify-between border border-destructive/20">
            <span>{error}</span>
            <button
              onClick={() => { }}
              className="text-destructive hover:text-destructive/80 transition-colors"
              aria-label="Dismiss error"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        )}

        {/* Input area - Premium glass effect */}
        <div className="flex items-end gap-2 p-3 pt-4 bg-gradient-to-t from-background via-background to-transparent">
          {/* File attachment button */}
          {fileUpload || (
            <label className="cursor-pointer">
              <input
                type="file"
                className="sr-only"
                onChange={handleFileInput}
                accept="image/*,audio/*,video/*,application/pdf,text/plain"
                disabled={disabled}
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="h-10 w-10 rounded-xl hover:bg-foreground/5 transition-colors"
                disabled={disabled}
                asChild
              >
                <span>
                  <Paperclip className="h-5 w-5 text-foreground-muted" />
                </span>
              </Button>
            </label>
          )}

          {/* Textarea container - Premium styling */}
          <div
            className={cn(
              'flex-1 relative rounded-2xl transition-all duration-200',
              'bg-background-secondary/80 backdrop-blur-sm',
              'border',
              isFocused
                ? 'border-primary/50 shadow-sm shadow-primary/10'
                : 'border-border/50',
              disabled && 'opacity-50'
            )}
          >
            <textarea
              ref={setRefs}
              value={value}
              onChange={handleChange}
              onKeyDown={handleKeyDown}
              onFocus={handleFocus}
              onBlur={handleBlur}
              placeholder={placeholder}
              disabled={disabled || isSending}
              rows={1}
              className={cn(
                'w-full resize-none bg-transparent px-4 py-3',
                'text-[15px] text-foreground placeholder:text-foreground-muted/60',
                'focus:outline-none',
                'min-h-[48px] max-h-[120px]'
              )}
              aria-label="Message"
            />
          </div>

          {/* Emoji picker button and dropdown */}
          <div className="relative">
            <Button
              ref={emojiButtonRef}
              type="button"
              variant="ghost"
              size="icon"
              className={cn(
                "h-10 w-10 rounded-xl transition-colors",
                showEmojiPicker
                  ? "bg-primary/20 text-primary"
                  : "hover:bg-foreground/5 text-foreground-muted"
              )}
              disabled={disabled}
              onClick={() => setShowEmojiPicker(!showEmojiPicker)}
              aria-label="Add emoji"
              aria-expanded={showEmojiPicker}
            >
              <Smile className="h-5 w-5" />
            </Button>

            {/* Emoji picker popover */}
            {showEmojiPicker && (
              <div className="absolute bottom-12 right-0 z-50">
                <EmojiPicker
                  onEmojiSelect={handleEmojiSelect}
                  onClose={() => setShowEmojiPicker(false)}
                />
              </div>
            )}
          </div>

          {/* Send button - Premium styling */}
          <Button
            type="button"
            size="icon"
            className={cn(
              'h-10 w-10 rounded-xl transition-all duration-200',
              value.trim()
                ? 'bg-primary hover:bg-primary/90 text-primary-foreground shadow-md shadow-primary/25'
                : 'bg-background-secondary text-foreground-muted border border-border/50'
            )}
            disabled={!value.trim() || disabled || isSending}
            onClick={handleSubmit}
            aria-label="Send message"
          >
            {isSending ? (
              <span className="w-5 h-5 rounded-full border-2 border-current border-t-transparent animate-spin" />
            ) : (
              <Send className={cn(
                'h-5 w-5 transition-transform duration-200',
                value.trim() && 'translate-x-0.5'
              )} />
            )}
          </Button>
        </div>
      </div>
    );
  }
);

/**
 * Typing indicator component - Premium animated dots
 */
export function TypingIndicator({ name }: { name: string }) {
  return (
    <div className="flex items-center gap-2 px-4 py-2">
      <div className="flex items-center gap-1 px-3 py-2 rounded-2xl bg-background-secondary/80 border border-border/30">
        <div className="flex gap-1">
          <span
            className="w-2 h-2 rounded-full bg-foreground-muted/60 animate-bounce"
            style={{ animationDuration: '0.6s', animationDelay: '0ms' }}
          />
          <span
            className="w-2 h-2 rounded-full bg-foreground-muted/60 animate-bounce"
            style={{ animationDuration: '0.6s', animationDelay: '150ms' }}
          />
          <span
            className="w-2 h-2 rounded-full bg-foreground-muted/60 animate-bounce"
            style={{ animationDuration: '0.6s', animationDelay: '300ms' }}
          />
        </div>
        <span className="text-sm text-foreground-muted ml-2">{name}</span>
      </div>
    </div>
  );
}
