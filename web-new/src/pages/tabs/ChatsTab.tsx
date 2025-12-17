/**
 * Chats Tab
 * 
 * Messaging interface - refactored from Chat.tsx to work within tab system
 * Features:
 * - Conversation list with unread indicators
 * - Active chat view with messages
 * - File attachments and media
 * - Voice/video call initiation
 */

import { useState, useCallback, useMemo, useEffect } from 'react';
import { useChatStore } from '@/core/store/chatStore';
import { useAuthStore } from '@/core/store/authStore';
import { useSettingsStore } from '@/core/store/settingsStore';
import { useWebSocket } from '@/hooks/useWebSocket';
import { useAuth } from '@/hooks/useAuth';
import { useMobileNavigation } from '@/hooks/useMobileNavigation';
import { useIsMobile, useViewport } from '@/hooks/useViewport';
import { cn } from '@/lib/utils';

// Chat components
import { ConversationList } from '@/components/chat/ConversationList';
import { ChatHeader } from '@/components/chat/ChatHeader';
import { MessageList } from '@/components/chat/MessageList';
import { MessageInput, TypingIndicator } from '@/components/chat/MessageInput';
import { MessageRequests } from '@/components/chat/MessageRequests';

// UI components
import { Button } from '@/components/ui/button';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Input } from '@/components/ui/input';
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from '@/components/ui/dialog';
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
    MessageSquare,
    UserPlus,
    UserCheck,
    Shield,
    Trash2,
    BellOff,
    Pin,
    MoreVertical,
    User,
} from 'lucide-react';

// Feature components
import { FileUpload } from '@/components/chat/FileUpload';
import { Settings } from '@/components/settings';
import { IncomingCall, ActiveCall, CallEnded } from '@/components/call';
import { Onboarding } from '@/components/Onboarding';
import { PinUnlock } from '@/components/auth/PinUnlock';
import { ImageLightbox } from '@/components/ui/ImageLightbox';

// Types
import type { Conversation, Message } from '@/core/types';
import { createFileMessageContent, downloadFile } from '@/core/services/fileUpload';
import type { UploadResult } from '@/core/services/fileUpload';

interface ChatsTabProps {
    onEnterChat?: () => void;
    onExitChat?: () => void;
}

export function ChatsTab({ onEnterChat, onExitChat }: ChatsTabProps) {
    // Store hooks
    const {
        conversations,
        messages,
        activeConversationId,
        setActiveConversation,
        addConversation,
        markConversationRead,
        removeConversation,
        updateConversation,
        typingUsers,
        acceptConversation,
        declineConversation,
        blockConversation,
    } = useChatStore();
    const { user, needsOnboarding, needsPinUnlock, onboardingStep } = useAuthStore();
    const { privacy } = useSettingsStore();
    const { logout } = useAuth();
    const { sendMessage, sendTyping, sendReadReceipt, startCall } = useWebSocket();

    // Viewport and navigation hooks
    const isMobile = useIsMobile();
    useViewport(); // Initialize viewport tracking

    // Memoize navigation callbacks to prevent infinite render loops
    const handleNavigateToChat = useCallback((id: string) => {
        setActiveConversation(id);
        onEnterChat?.();
    }, [setActiveConversation, onEnterChat]);

    const handleNavigateToList = useCallback(() => {
        onExitChat?.();
    }, [onExitChat]);

    const {
        navigateToChat,
        navigateToList,
        getListPanelClasses,
        getChatPanelClasses,
    } = useMobileNavigation({
        onNavigateToChat: handleNavigateToChat,
        onNavigateToList: handleNavigateToList,
    });

    // Local state
    const [messageInput, setMessageInput] = useState('');
    const [isNewChatOpen, setIsNewChatOpen] = useState(false);
    const [newChatUsername, setNewChatUsername] = useState('');
    const [searchResults, setSearchResults] = useState<Array<{
        user_id: string;
        username: string;
        display_name?: string;
        avatar_url?: string;
    }>>([]);
    const [isSearching, setIsSearching] = useState(false);
    const [isSettingsOpen, setIsSettingsOpen] = useState(false);
    const [fileError, setFileError] = useState<string | null>(null);
    const [downloadingFiles, setDownloadingFiles] = useState<Set<string>>(new Set());
    const [isSending, setIsSending] = useState(false);
    const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false);
    const [conversationToDelete, setConversationToDelete] = useState<string | null>(null);
    const [isMessageRequestsOpen, setIsMessageRequestsOpen] = useState(false);
    const [lightboxImage, setLightboxImage] = useState<{ src: string; fileName: string } | null>(null);

    // Computed values
    const conversationList = useMemo(() => Object.values(conversations), [conversations]);
    const activeConversation = activeConversationId ? conversations[activeConversationId] : null;
    const activeMessages = activeConversationId ? messages[activeConversationId] || [] : [];
    const messageRequests = useMemo(
        () => conversationList.filter((c) => c.status === 'pending'),
        [conversationList]
    );

    // Computed online status
    const recipientId = activeConversation?.recipientId;
    const hasReceivedReply = recipientId ? activeMessages.some(m => m.senderId === recipientId) : false;
    const canSeePresence = privacy.onlineStatus;
    const recipientOnline = hasReceivedReply && canSeePresence && (activeConversation?.isOnline ?? false);
    const recipientLastSeen = hasReceivedReply && canSeePresence ? activeConversation?.lastSeen : undefined;

    // Mark conversation as read locally when selected
    useEffect(() => {
        if (activeConversationId && (activeConversation?.unreadCount ?? 0) > 0) {
            markConversationRead(activeConversationId);
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [activeConversationId, activeConversation?.unreadCount]);

    // Handle when a message becomes visible in the viewport
    const handleMessageVisible = useCallback((message: Message) => {
        if (activeConversation?.status === 'accepted' && activeConversationId) {
            sendReadReceipt(message.id, activeConversationId);
        }
    }, [activeConversation?.status, activeConversationId, sendReadReceipt]);

    // Search for users as user types
    useEffect(() => {
        if (!isNewChatOpen) return;

        const searchUsername = newChatUsername.replace(/^@/, '').trim();
        if (searchUsername.length < 3) {
            setSearchResults([]);
            return;
        }

        setIsSearching(true);
        const timeoutId = setTimeout(async () => {
            try {
                const response = await fetch(`/api/v1/users/search?q=${encodeURIComponent(searchUsername)}&limit=10`, {
                    headers: {
                        Authorization: `Bearer ${useAuthStore.getState().token}`,
                    },
                });

                if (response.ok) {
                    const results = await response.json();
                    setSearchResults(results || []);
                }
            } catch (error) {
                console.error('Failed to search users:', error);
            } finally {
                setIsSearching(false);
            }
        }, 300);

        return () => clearTimeout(timeoutId);
    }, [newChatUsername, isNewChatOpen]);

    // Handle conversation selection
    const handleSelectConversation = useCallback((id: string) => {
        if (isMobile) {
            navigateToChat(id);
        } else {
            if (activeConversationId === id) {
                setActiveConversation(null);
            } else {
                setActiveConversation(id);
            }
        }
    }, [isMobile, navigateToChat, setActiveConversation, activeConversationId]);

    // Handle back navigation
    const handleBack = useCallback(() => {
        navigateToList();
        setActiveConversation(null);
    }, [navigateToList, setActiveConversation]);

    // Handle typing indicator
    const handleTypingChange = useCallback((isTyping: boolean) => {
        if (activeConversation) {
            sendTyping(activeConversation.recipientId, isTyping);
        }
    }, [activeConversation, sendTyping]);

    // Handle send message
    const handleSendMessage = useCallback(async () => {
        if (!messageInput.trim() || !activeConversation || isSending) return;

        const content = messageInput.trim();
        setMessageInput('');
        setIsSending(true);

        try {
            await sendMessage(activeConversation.recipientId, content);
        } catch (error) {
            console.error('Failed to send message:', error);
            setMessageInput(content);
        } finally {
            setIsSending(false);
        }
    }, [messageInput, activeConversation, isSending, sendMessage]);

    // Handle user selection from search
    const handleSelectUser = useCallback((foundUser: {
        user_id: string;
        username: string;
        display_name?: string;
        avatar_url?: string;
    }) => {
        setIsNewChatOpen(false);

        const conversation: Conversation = {
            id: foundUser.user_id,
            recipientId: foundUser.user_id,
            recipientName: foundUser.display_name || `@${foundUser.username}`,
            recipientAvatar: foundUser.avatar_url,
            unreadCount: 0,
            isOnline: false,
            isPinned: false,
            isMuted: false,
            status: 'accepted',
        };

        addConversation(conversation);
        handleSelectConversation(conversation.id);
    }, [addConversation, handleSelectConversation]);

    // Handle file upload
    const handleFileUploadComplete = useCallback(async (result: UploadResult) => {
        if (!activeConversation) return;

        const content = createFileMessageContent(result.metadata);
        setIsSending(true);
        try {
            await sendMessage(activeConversation.recipientId, content);
        } finally {
            setIsSending(false);
            setFileError(null);
        }
    }, [activeConversation, sendMessage]);

    // Handle file download
    const handleFileDownload = useCallback(async (metadata: import('@/core/types').FileMetadata) => {
        const mediaId = metadata.mediaId;
        if (downloadingFiles.has(mediaId)) return;

        setDownloadingFiles(prev => new Set(prev).add(mediaId));

        try {
            const blob = await downloadFile(metadata);
            const url = URL.createObjectURL(blob);

            if (metadata.mimeType.startsWith('image/')) {
                setLightboxImage({ src: url, fileName: metadata.fileName });
            } else {
                const a = document.createElement('a');
                a.href = url;
                a.download = metadata.fileName;
                document.body.appendChild(a);
                a.click();
                document.body.removeChild(a);
                URL.revokeObjectURL(url);
            }
        } catch (error) {
            console.error('Failed to download file:', error);
            setFileError('Failed to download file');
        } finally {
            setDownloadingFiles(prev => {
                const next = new Set(prev);
                next.delete(mediaId);
                return next;
            });
        }
    }, [downloadingFiles]);

    // Handle lightbox close
    const handleCloseLightbox = useCallback(() => {
        if (lightboxImage) {
            URL.revokeObjectURL(lightboxImage.src);
        }
        setLightboxImage(null);
    }, [lightboxImage]);

    // Handle download from lightbox
    const handleDownloadFromLightbox = useCallback(() => {
        if (!lightboxImage) return;
        const a = document.createElement('a');
        a.href = lightboxImage.src;
        a.download = lightboxImage.fileName;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
    }, [lightboxImage]);

    // Handle call
    const handleCall = useCallback((type: 'audio' | 'video') => {
        if (!activeConversation) return;
        startCall(
            activeConversation.recipientId,
            activeConversation.recipientName,
            activeConversation.recipientAvatar,
            type
        );
    }, [activeConversation, startCall]);

    // Handle delete chat
    const handleDeleteChat = useCallback(() => {
        if (activeConversationId) {
            setConversationToDelete(activeConversationId);
            setIsDeleteConfirmOpen(true);
        }
    }, [activeConversationId]);

    // Confirm delete chat
    const confirmDeleteChat = useCallback(() => {
        if (conversationToDelete) {
            removeConversation(conversationToDelete);
            setIsDeleteConfirmOpen(false);
            setConversationToDelete(null);
            if (isMobile) {
                navigateToList();
            }
        }
    }, [conversationToDelete, removeConversation, isMobile, navigateToList]);

    // Handle toggle mute
    const handleToggleMute = useCallback(() => {
        if (activeConversation) {
            updateConversation(activeConversation.id, { isMuted: !activeConversation.isMuted });
        }
    }, [activeConversation, updateConversation]);

    // Handle toggle pin
    const handleTogglePin = useCallback(() => {
        if (activeConversation) {
            updateConversation(activeConversation.id, { isPinned: !activeConversation.isPinned });
        }
    }, [activeConversation, updateConversation]);

    // Check if this user is already a friend (has accepted conversation with messages from them)
    const isFriend = useMemo(() => {
        if (!activeConversation) return false;
        // Consider them a friend if they've replied (conversation is active both ways)
        return activeMessages.some(m => m.senderId === activeConversation.recipientId);
    }, [activeConversation, activeMessages]);

    // Handle send friend request (for now this is a placeholder - would be an API call)
    const handleSendFriendRequest = useCallback(async () => {
        if (!activeConversation) return;
        // TODO: Implement actual friend request API
        // For now, we'll just show a confirmation that they'll see friend request once they reply
        console.log('Friend request would be sent to:', activeConversation.recipientId);
    }, [activeConversation]);

    // Handle view user profile
    const handleViewProfile = useCallback(() => {
        if (!activeConversation) return;
        // TODO: Navigate to user profile view
        console.log('View profile:', activeConversation.recipientId);
    }, [activeConversation]);

    // Show PIN unlock for existing users who need to unlock their encrypted data
    if (needsPinUnlock) {
        return <PinUnlock />;
    }

    // Show onboarding wizard for new users
    if (needsOnboarding && onboardingStep !== 'complete') {
        return <Onboarding />;
    }

    return (
        <div className="h-full flex overflow-hidden">
            {/* CONVERSATION LIST PANEL */}
            <aside
                className={cn(
                    'flex flex-col bg-background',
                    isMobile && 'mobile-panel',
                    isMobile && getListPanelClasses(),
                    !isMobile && 'w-80 lg:w-[360px] xl:w-[400px] border-r border-border flex-shrink-0'
                )}
            >
                <ConversationList
                    conversations={conversationList}
                    activeConversationId={activeConversationId}
                    user={user}
                    onSelectConversation={handleSelectConversation}
                    onNewChat={() => setIsNewChatOpen(true)}
                    onOpenSettings={() => setIsSettingsOpen(true)}
                    onLogout={logout}
                    onOpenMessageRequests={() => setIsMessageRequestsOpen(true)}
                    onDeleteConversation={removeConversation}
                />
            </aside>

            {/* CHAT VIEW PANEL */}
            <main
                className={cn(
                    'flex-1 flex flex-col bg-background',
                    isMobile && 'mobile-panel',
                    isMobile && getChatPanelClasses(),
                    !isMobile && 'min-w-0'
                )}
            >
                {activeConversation ? (
                    <>
                        {/* Chat Header with dropdown menu */}
                        <div className="relative flex items-center">
                            <div className="flex-1">
                                <ChatHeader
                                    recipientName={activeConversation.recipientName}
                                    recipientAvatar={activeConversation.recipientAvatar}
                                    isOnline={recipientOnline}
                                    lastSeen={recipientLastSeen}
                                    onBack={isMobile ? handleBack : undefined}
                                    onVoiceCall={() => handleCall('audio')}
                                    onVideoCall={() => handleCall('video')}
                                    onMoreOptions={undefined}
                                />
                            </div>
                            {/* More Options Dropdown */}
                            <div className="absolute right-2 top-1/2 -translate-y-1/2 z-10">
                                <DropdownMenu>
                                    <DropdownMenuTrigger asChild>
                                        <Button variant="ghost" size="icon" className="touch-target">
                                            <MoreVertical className="h-5 w-5" />
                                        </Button>
                                    </DropdownMenuTrigger>
                                    <DropdownMenuContent align="end" className="w-52">
                                        <DropdownMenuItem onClick={handleViewProfile}>
                                            <User className="h-4 w-4 mr-2" />
                                            View Profile
                                        </DropdownMenuItem>
                                        {!isFriend && (
                                            <DropdownMenuItem onClick={handleSendFriendRequest}>
                                                <UserPlus className="h-4 w-4 mr-2" />
                                                Add Friend
                                            </DropdownMenuItem>
                                        )}
                                        {isFriend && (
                                            <DropdownMenuItem disabled className="text-success">
                                                <UserCheck className="h-4 w-4 mr-2" />
                                                Friends
                                            </DropdownMenuItem>
                                        )}
                                        <DropdownMenuSeparator />
                                        <DropdownMenuItem onClick={handleTogglePin}>
                                            <Pin className="h-4 w-4 mr-2" />
                                            {activeConversation.isPinned ? 'Unpin Chat' : 'Pin Chat'}
                                        </DropdownMenuItem>
                                        <DropdownMenuItem onClick={handleToggleMute}>
                                            <BellOff className="h-4 w-4 mr-2" />
                                            {activeConversation.isMuted ? 'Unmute' : 'Mute Notifications'}
                                        </DropdownMenuItem>
                                        <DropdownMenuSeparator />
                                        <DropdownMenuItem
                                            onClick={handleDeleteChat}
                                            className="text-destructive focus:text-destructive"
                                        >
                                            <Trash2 className="h-4 w-4 mr-2" />
                                            Delete Chat
                                        </DropdownMenuItem>
                                    </DropdownMenuContent>
                                </DropdownMenu>
                            </div>
                        </div>

                        {/* Message List */}
                        <MessageList
                            messages={activeMessages}
                            currentUserId={user?.id || ''}
                            onFileDownload={handleFileDownload}
                            downloadingFiles={downloadingFiles}
                            onMessageVisible={handleMessageVisible}
                            readReceiptsEnabled={privacy.readReceipts}
                        />

                        {/* Typing Indicator */}
                        {activeConversation &&
                            activeConversation.status === 'accepted' &&
                            typingUsers[activeConversation.recipientId] && (
                                <TypingIndicator name={activeConversation.recipientName} />
                            )}

                        {/* Message Input */}
                        <MessageInput
                            value={messageInput}
                            onChange={setMessageInput}
                            onSubmit={handleSendMessage}
                            onTypingChange={handleTypingChange}
                            isSending={isSending}
                            error={fileError}
                            fileUpload={
                                <FileUpload
                                    onUploadComplete={handleFileUploadComplete}
                                    onError={(error) => {
                                        setFileError(error);
                                        setTimeout(() => setFileError(null), 5000);
                                    }}
                                />
                            }
                        />
                    </>
                ) : (
                    /* Empty State */
                    <ChatsEmptyState
                        onNewChat={() => setIsNewChatOpen(true)}
                        isMobile={isMobile}
                    />
                )}
            </main>

            {/* NEW CHAT DIALOG */}
            <Dialog
                open={isNewChatOpen}
                onOpenChange={(open) => {
                    setIsNewChatOpen(open);
                    if (!open) {
                        setNewChatUsername('');
                        setSearchResults([]);
                    }
                }}
            >
                <DialogContent className="max-w-md" aria-describedby="new-chat-description">
                    <DialogHeader>
                        <DialogTitle>New Chat</DialogTitle>
                        <DialogDescription id="new-chat-description">
                            Search by username to start a new conversation
                        </DialogDescription>
                    </DialogHeader>
                    <div className="space-y-4">
                        <div className="relative">
                            <span className="absolute left-3 top-1/2 -translate-y-1/2 text-foreground-muted font-mono">
                                @
                            </span>
                            <Input
                                type="text"
                                placeholder="username"
                                value={newChatUsername.replace(/^@/, '')}
                                onChange={(e) => setNewChatUsername(e.target.value)}
                                className="pl-8 font-mono"
                                autoFocus
                            />
                        </div>

                        {/* Search Results */}
                        {isSearching && (
                            <p className="text-sm text-foreground-muted text-center py-2">Searching...</p>
                        )}

                        {!isSearching && searchResults.length > 0 && (
                            <div className="border border-border rounded-lg divide-y divide-border max-h-60 overflow-y-auto">
                                {searchResults.map((result) => (
                                    <button
                                        key={result.user_id}
                                        type="button"
                                        onClick={() => handleSelectUser(result)}
                                        className="w-full px-4 py-3 flex items-center gap-3 hover:bg-background-tertiary transition-colors text-left touch-target"
                                    >
                                        <Avatar className="h-10 w-10">
                                            <AvatarImage src={result.avatar_url} />
                                            <AvatarFallback>
                                                {result.username?.charAt(0).toUpperCase() || '?'}
                                            </AvatarFallback>
                                        </Avatar>
                                        <div>
                                            <p className="font-medium">@{result.username}</p>
                                            {result.display_name && (
                                                <p className="text-sm text-foreground-secondary">{result.display_name}</p>
                                            )}
                                        </div>
                                    </button>
                                ))}
                            </div>
                        )}

                        {!isSearching && newChatUsername.replace(/^@/, '').length >= 3 && searchResults.length === 0 && (
                            <p className="text-sm text-foreground-muted text-center py-4">
                                No users found with that username
                            </p>
                        )}

                        {newChatUsername.replace(/^@/, '').length > 0 && newChatUsername.replace(/^@/, '').length < 3 && (
                            <p className="text-sm text-foreground-muted text-center py-2">
                                Type at least 3 characters to search
                            </p>
                        )}

                        <div className="flex justify-end">
                            <Button type="button" variant="outline" onClick={() => setIsNewChatOpen(false)}>
                                Cancel
                            </Button>
                        </div>
                    </div>
                </DialogContent>
            </Dialog>

            {/* SETTINGS MODAL */}
            <Settings open={isSettingsOpen} onOpenChange={setIsSettingsOpen} />

            {/* MESSAGE REQUESTS */}
            <MessageRequests
                open={isMessageRequestsOpen}
                onOpenChange={setIsMessageRequestsOpen}
                requests={messageRequests}
                onAccept={acceptConversation}
                onDecline={declineConversation}
                onBlock={blockConversation}
                onSelectConversation={handleSelectConversation}
            />

            {/* DELETE CONFIRMATION */}
            <Dialog open={isDeleteConfirmOpen} onOpenChange={setIsDeleteConfirmOpen}>
                <DialogContent className="max-w-sm" aria-describedby="delete-chat-description">
                    <DialogHeader>
                        <DialogTitle>Delete Chat</DialogTitle>
                        <DialogDescription id="delete-chat-description">
                            Are you sure you want to delete this conversation? This action cannot be undone.
                        </DialogDescription>
                    </DialogHeader>
                    <div className="flex gap-3 justify-end mt-4">
                        <Button variant="outline" onClick={() => setIsDeleteConfirmOpen(false)}>
                            Cancel
                        </Button>
                        <Button variant="destructive" onClick={confirmDeleteChat}>
                            Delete
                        </Button>
                    </div>
                </DialogContent>
            </Dialog>

            {/* CALL UI */}
            <IncomingCall />
            <ActiveCall />
            <CallEnded />

            {/* IMAGE LIGHTBOX */}
            {lightboxImage && (
                <ImageLightbox
                    src={lightboxImage.src}
                    fileName={lightboxImage.fileName}
                    onClose={handleCloseLightbox}
                    onDownload={handleDownloadFromLightbox}
                />
            )}
        </div>
    );
}

/**
 * Empty state when no conversation is selected
 */
function ChatsEmptyState({
    onNewChat,
    isMobile,
}: {
    onNewChat: () => void;
    isMobile: boolean;
}) {
    if (isMobile) {
        return null;
    }

    return (
        <div className="flex-1 flex items-center justify-center p-8 relative">
            {/* Background gradient */}
            <div className="absolute inset-0 bg-gradient-to-br from-primary/[0.02] via-transparent to-primary/[0.03] pointer-events-none" />

            <div className="text-center max-w-sm relative">
                {/* Icon with glow effect */}
                <div className="relative mb-6 inline-block empty-state-float">
                    <div className="absolute inset-0 bg-primary/20 rounded-2xl blur-2xl scale-150 empty-state-pulse" />
                    <div className="relative w-20 h-20 rounded-2xl bg-gradient-to-br from-background-secondary to-background-tertiary flex items-center justify-center border border-border/50 shadow-lg">
                        <MessageSquare className="h-9 w-9 text-foreground-muted" />
                    </div>
                </div>

                <h2 className="text-xl font-semibold mb-2">Welcome to SilentRelay</h2>
                <p className="text-foreground-muted">
                    Select a conversation to start messaging, or start a new chat.
                </p>

                <Button
                    className="mt-6 rounded-xl shadow-lg shadow-primary/20 px-6"
                    onClick={onNewChat}
                >
                    <UserPlus className="h-4 w-4 mr-2" />
                    Start New Chat
                </Button>

                {/* Encryption badge */}
                <div className="flex justify-center mt-4">
                    <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-primary/5 border border-primary/10">
                        <Shield className="h-3.5 w-3.5 text-primary" />
                        <span className="text-xs text-primary font-medium">End-to-end encrypted</span>
                    </div>
                </div>
            </div>
        </div>
    );
}

export default ChatsTab;
