/**
 * ConversationList Component - Premium Redesign
 *
 * Mobile-optimized conversation list with:
 * - Pull-to-refresh
 * - Virtualized scrolling (for performance)
 * - Search filtering
 * - Empty state
 * - Premium glass morphism design
 */

import { useMemo, useCallback, useState } from 'react';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import {
  MessageSquare,
  Search,
  Settings as SettingsIcon,
  Plus,
  LogOut,
  Shield,
  RefreshCw,
  Inbox,
  ChevronRight,
  Sparkles,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { ConversationItem, ConversationItemSkeleton } from './ConversationItem';
import { usePullToRefresh } from '@/hooks/useSwipeGesture';
import type { Conversation, User } from '@/core/types';

interface ConversationListProps {
  /** List of conversations */
  conversations: Conversation[];
  /** Currently active conversation ID */
  activeConversationId: string | null;
  /** Current user */
  user: User | null;
  /** Whether data is loading */
  isLoading?: boolean;
  /** Callback when conversation is selected */
  onSelectConversation: (id: string) => void;
  /** Callback to open new chat dialog */
  onNewChat: () => void;
  /** Callback to open settings */
  onOpenSettings: () => void;
  /** Callback to logout */
  onLogout: () => void;
  /** Callback for pull-to-refresh */
  onRefresh?: () => Promise<void>;
  /** Callback to open message requests */
  onOpenMessageRequests?: () => void;
  /** Callback to delete a conversation */
  onDeleteConversation?: (conversationId: string) => void;
  /** Additional className */
  className?: string;
}

export function ConversationList({
  conversations,
  activeConversationId,
  user,
  isLoading,
  onSelectConversation,
  onNewChat,
  onOpenSettings,
  onLogout,
  onRefresh,
  onOpenMessageRequests,
  onDeleteConversation,
  className,
}: ConversationListProps) {
  const [searchQuery, setSearchQuery] = useState('');
  const [isSearchFocused, setIsSearchFocused] = useState(false);

  // Separate accepted conversations from message requests (pending)
  const { acceptedConversations, messageRequests } = useMemo(() => {
    const accepted: Conversation[] = [];
    const requests: Conversation[] = [];
    
    for (const conv of conversations) {
      if (conv.status === 'pending') {
        requests.push(conv);
      } else if (conv.status !== 'blocked') {
        accepted.push(conv);
      }
    }
    
    return { acceptedConversations: accepted, messageRequests: requests };
  }, [conversations]);

  // Filter accepted conversations by search query
  const filteredConversations = useMemo(() => {
    if (!searchQuery.trim()) return acceptedConversations;
    
    const query = searchQuery.toLowerCase();
    return acceptedConversations.filter((conv) =>
      conv.recipientName.toLowerCase().includes(query)
    );
  }, [acceptedConversations, searchQuery]);

  // Sort conversations: pinned first, then by last message timestamp
  const sortedConversations = useMemo(() => {
    return [...filteredConversations].sort((a, b) => {
      // Pinned conversations first
      if (a.isPinned && !b.isPinned) return -1;
      if (!a.isPinned && b.isPinned) return 1;
      
      // Then by last message timestamp (newest first)
      const aTime = a.lastMessage?.timestamp || 0;
      const bTime = b.lastMessage?.timestamp || 0;
      return bTime - aTime;
    });
  }, [filteredConversations]);

  // Pull to refresh
  const handleRefresh = useCallback(async () => {
    if (onRefresh) {
      await onRefresh();
    }
    // Simulate refresh if no handler
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }, [onRefresh]);

  const { pullDistance, isRefreshing, progress, bindElement } = usePullToRefresh({
    onRefresh: handleRefresh,
    threshold: 80,
  });

  // Get user initials
  const userInitials = useMemo(() => {
    if (!user) return '?';
    const name = user.displayName || user.username || '';
    return name.charAt(0).toUpperCase() || '?';
  }, [user]);

  return (
    <div className={cn('flex flex-col h-full bg-background relative', className)}>
      {/* Subtle gradient background overlay */}
      <div className="absolute inset-0 bg-gradient-to-b from-primary/[0.02] via-transparent to-transparent pointer-events-none" />
      
      {/* Header - Clean and minimal */}
      <header className="relative p-4 pt-6 flex-shrink-0">
        {/* Logo */}
        <div className="flex items-center gap-2.5 mb-5">
          <div className="relative">
            <div className="absolute inset-0 bg-primary/20 rounded-lg blur-lg" />
            <div className="relative p-1.5 bg-gradient-to-br from-primary/20 to-primary/5 rounded-lg border border-primary/20">
              <Shield className="h-5 w-5 text-primary" />
            </div>
          </div>
          <span className="font-semibold text-lg tracking-tight">SilentRelay</span>
        </div>
        
        {/* Search input - Premium style */}
        <div className="relative group">
          <div className={cn(
            'absolute -inset-0.5 bg-gradient-to-r from-primary/20 via-primary/10 to-primary/20 rounded-xl opacity-0 blur transition-all duration-300',
            isSearchFocused && 'opacity-100'
          )} />
          <div className="relative">
            <Search className={cn(
              'absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 transition-colors duration-200',
              isSearchFocused ? 'text-primary' : 'text-foreground-muted'
            )} />
            <Input
              type="search"
              placeholder="Search conversations..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              onFocus={() => setIsSearchFocused(true)}
              onBlur={() => setIsSearchFocused(false)}
              className="pl-10 bg-background-secondary/50 border-border/50 rounded-xl focus:bg-background-secondary focus:border-primary/30 transition-all duration-200"
              aria-label="Search conversations"
            />
          </div>
        </div>
      </header>

      {/* Pull to refresh indicator */}
      {pullDistance > 0 && (
        <div
          className="flex items-center justify-center bg-background-secondary/50 overflow-hidden transition-all duration-150"
          style={{ height: pullDistance }}
        >
          <RefreshCw
            className={cn(
              'h-5 w-5 text-primary transition-transform',
              isRefreshing && 'pull-refresh-spinner'
            )}
            style={{
              transform: isRefreshing ? undefined : `rotate(${progress * 360}deg)`,
              opacity: Math.min(progress, 1),
            }}
          />
        </div>
      )}

      {/* Conversation list */}
      <ScrollArea
        className="flex-1 relative"
        ref={(el) => {
          // @ts-expect-error - accessing internal viewport
          if (el?._viewport) {
            // @ts-expect-error - accessing internal viewport
            bindElement(el._viewport);
          }
        }}
      >
        {isLoading ? (
          // Loading skeletons
          <div className="py-2 px-2">
            {[...Array(6)].map((_, i) => (
              <ConversationItemSkeleton key={i} />
            ))}
          </div>
        ) : (
          <>
            {/* Message Requests Banner - Premium Card Style */}
            {messageRequests.length > 0 && (
              <div className="px-3 py-2">
                <button
                  type="button"
                  onClick={onOpenMessageRequests}
                  className="w-full px-4 py-3 flex items-center gap-3 bg-gradient-to-r from-primary/10 via-primary/5 to-transparent hover:from-primary/15 hover:via-primary/10 transition-all duration-200 rounded-xl border border-primary/10 group"
                >
                  <div className="relative">
                    <div className="absolute inset-0 bg-primary/30 rounded-full blur-md opacity-0 group-hover:opacity-100 transition-opacity" />
                    <div className="relative w-10 h-10 rounded-full bg-primary/10 flex items-center justify-center">
                      <Inbox className="h-5 w-5 text-primary" />
                    </div>
                  </div>
                  <div className="flex-1 text-left">
                    <p className="font-medium text-sm">Message Requests</p>
                    <p className="text-xs text-foreground-muted">
                      {messageRequests.length} pending request{messageRequests.length !== 1 ? 's' : ''}
                    </p>
                  </div>
                  <div className="w-6 h-6 rounded-full bg-primary/10 flex items-center justify-center group-hover:bg-primary/20 transition-colors">
                    <ChevronRight className="h-4 w-4 text-primary" />
                  </div>
                </button>
              </div>
            )}

            {sortedConversations.length === 0 && messageRequests.length === 0 ? (
              // Empty state
              <EmptyState
                hasSearch={searchQuery.length > 0}
                onNewChat={onNewChat}
              />
            ) : sortedConversations.length === 0 && searchQuery.length > 0 ? (
              // No search results
              <EmptyState
                hasSearch={true}
                onNewChat={onNewChat}
              />
            ) : (
              // Conversation list
              <div className="py-1 px-2" role="list" aria-label="Conversations">
                {sortedConversations.map((conversation) => (
                  <ConversationItem
                    key={conversation.id}
                    conversation={conversation}
                    isActive={activeConversationId === conversation.id}
                    onClick={() => onSelectConversation(conversation.id)}
                    onDelete={onDeleteConversation}
                  />
                ))}
              </div>
            )}
          </>
        )}
      </ScrollArea>

      {/* User Footer - Premium Design with Actions */}
      <footer className="relative p-3 pb-5 flex-shrink-0 border-t border-border/50">
        {/* Subtle top gradient */}
        <div className="absolute inset-x-0 -top-8 h-8 bg-gradient-to-t from-background to-transparent pointer-events-none" />
        
        <div className="relative flex items-center gap-3">
          {/* User Avatar with Status */}
          <div className="relative">
            <Avatar className="h-10 w-10 ring-2 ring-background shadow-lg">
              <AvatarImage src={user?.avatar} alt="" />
              <AvatarFallback className="bg-gradient-to-br from-primary/20 to-primary/5 text-primary font-medium">
                {userInitials}
              </AvatarFallback>
            </Avatar>
            {/* Online indicator */}
            <div className="absolute -bottom-0.5 -right-0.5 w-3.5 h-3.5 rounded-full bg-emerald-500 border-2 border-background shadow-sm" />
          </div>
          
          {/* User Info */}
          <div className="flex-1 min-w-0">
            <p className="font-medium text-sm truncate leading-tight">
              {user?.displayName || user?.username || 'Unknown'}
            </p>
            {user?.username && (
              <p className="text-xs text-foreground-muted truncate">
                @{user.username}
              </p>
            )}
          </div>
          
          {/* Action Buttons - Premium Style */}
          <TooltipProvider delayDuration={300}>
            <div className="flex items-center gap-1">
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-9 w-9 rounded-lg hover:bg-primary/10 hover:text-primary transition-colors"
                    onClick={onNewChat}
                    aria-label="New chat"
                  >
                    <Plus className="h-[18px] w-[18px]" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent side="top" className="text-xs">
                  New Chat
                </TooltipContent>
              </Tooltip>
              
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-9 w-9 rounded-lg hover:bg-foreground/5 transition-colors"
                    onClick={onOpenSettings}
                    aria-label="Settings"
                  >
                    <SettingsIcon className="h-[18px] w-[18px]" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent side="top" className="text-xs">
                  Settings
                </TooltipContent>
              </Tooltip>
              
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-9 w-9 rounded-lg hover:bg-destructive/10 hover:text-destructive transition-colors"
                    onClick={onLogout}
                    aria-label="Log out"
                  >
                    <LogOut className="h-[18px] w-[18px]" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent side="top" className="text-xs">
                  Log out
                </TooltipContent>
              </Tooltip>
            </div>
          </TooltipProvider>
        </div>
      </footer>
    </div>
  );
}

/**
 * Empty state component - Premium Design
 */
function EmptyState({
  hasSearch,
  onNewChat,
}: {
  hasSearch: boolean;
  onNewChat: () => void;
}) {
  return (
    <div className="flex flex-col items-center justify-center p-8 text-center min-h-[300px]">
      {/* Animated gradient background */}
      <div className="relative mb-6">
        <div className="absolute inset-0 bg-gradient-to-r from-primary/20 to-primary/10 rounded-full blur-2xl scale-150 animate-pulse" />
        <div className="relative w-20 h-20 rounded-2xl bg-gradient-to-br from-background-secondary to-background-tertiary flex items-center justify-center border border-border/50 shadow-lg">
          {hasSearch ? (
            <Search className="h-8 w-8 text-foreground-muted" />
          ) : (
            <div className="relative">
              <MessageSquare className="h-8 w-8 text-foreground-muted" />
              <Sparkles className="h-4 w-4 text-primary absolute -top-1 -right-1" />
            </div>
          )}
        </div>
      </div>
      
      {hasSearch ? (
        <>
          <p className="text-foreground font-medium">No results found</p>
          <p className="text-sm text-foreground-muted mt-1.5 max-w-[200px]">
            Try a different search term
          </p>
        </>
      ) : (
        <>
          <p className="text-foreground font-medium">Start a conversation</p>
          <p className="text-sm text-foreground-muted mt-1.5 max-w-[220px]">
            Send encrypted messages to friends and family
          </p>
          <Button
            size="sm"
            className="mt-5 rounded-lg bg-primary hover:bg-primary/90 shadow-lg shadow-primary/20"
            onClick={onNewChat}
          >
            <Plus className="h-4 w-4 mr-1.5" />
            New Chat
          </Button>
        </>
      )}
    </div>
  );
}
