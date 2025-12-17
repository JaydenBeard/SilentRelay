/**
 * Friends Tab
 * 
 * Friends management with:
 * - Friends list with glassmorphism cards
 * - Online status indicators with pulse animations
 * - Friend requests section (incoming/outgoing)
 * - Add friends functionality (sends friend request)
 */

import { useState, useMemo, useCallback, useEffect } from 'react';
import { useAuthStore } from '@/core/store/authStore';
import { useChatStore } from '@/core/store/chatStore';
import { useFriendsStore } from '@/core/store/friendsStore';

// UI Components
import { GlassCard } from '@/components/ui/GlassCard';
import { AvatarRing } from '@/components/ui/AvatarRing';
import { EmptyState } from '@/components/ui/EmptyState';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from '@/components/ui/dialog';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import {
    Users,
    UserPlus,
    Search,
    MessageSquare,
    Phone,
    Video,
    UserCheck,
    UserX,
    Clock,
    Loader2,
    X,
} from 'lucide-react';
import type { FriendRequest } from '@/core/types';

interface SearchResult {
    id: string;
    username: string;
    displayName: string;
    avatarUrl?: string;
    isOnline?: boolean;
}

export function ContactsTab() {
    const { token } = useAuthStore();
    const { setActiveConversation } = useChatStore();
    const {
        friends,
        incomingRequests,
        outgoingRequests,
        isLoading,
        refreshAll,
        sendFriendRequest,
        acceptRequest,
        declineRequest,
        cancelRequest,
    } = useFriendsStore();

    // State
    const [searchQuery, setSearchQuery] = useState('');
    const [isAddFriendOpen, setIsAddFriendOpen] = useState(false);
    const [addFriendUsername, setAddFriendUsername] = useState('');
    const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
    const [isSearching, setIsSearching] = useState(false);
    const [activeSection, setActiveSection] = useState<'all' | 'online' | 'requests'>('all');
    const [sendingRequest, setSendingRequest] = useState<string | null>(null);

    // Load friends on mount
    useEffect(() => {
        refreshAll();
    }, [refreshAll]);

    // Convert API friends to display format
    const displayFriends = useMemo(() => {
        return friends.map(f => ({
            id: f.user_id,
            username: f.username,
            displayName: f.display_name || `@${f.username}`,
            avatarUrl: f.avatar_url,
            isOnline: f.is_online,
            lastSeen: f.last_seen ? new Date(f.last_seen).getTime() : undefined,
            friendsSince: f.friends_since,
        }));
    }, [friends]);

    // Filter friends based on search and section
    const filteredFriends = useMemo(() => {
        let result = displayFriends;

        // Filter by section
        if (activeSection === 'online') {
            result = result.filter(c => c.isOnline);
        }

        // Filter by search query
        if (searchQuery.trim()) {
            const query = searchQuery.toLowerCase();
            result = result.filter(c =>
                c.displayName.toLowerCase().includes(query) ||
                c.username.toLowerCase().includes(query)
            );
        }

        // Sort: online first, then alphabetically
        return result.sort((a, b) => {
            if (a.isOnline !== b.isOnline) return a.isOnline ? -1 : 1;
            return a.displayName.localeCompare(b.displayName);
        });
    }, [displayFriends, activeSection, searchQuery]);

    // Count online friends
    const onlineCount = useMemo(() =>
        displayFriends.filter(c => c.isOnline).length,
        [displayFriends]
    );

    // Total pending requests count
    const pendingRequestsCount = incomingRequests.length + outgoingRequests.length;

    // Search for users to add as friends
    useEffect(() => {
        if (!isAddFriendOpen) return;

        const searchUsername = addFriendUsername.replace(/^@/, '').trim();
        if (searchUsername.length < 3) {
            setSearchResults([]);
            return;
        }

        setIsSearching(true);
        const timeoutId = setTimeout(async () => {
            try {
                const response = await fetch(
                    `/api/v1/users/search?q=${encodeURIComponent(searchUsername)}&limit=10`,
                    { headers: { Authorization: `Bearer ${token}` } }
                );

                if (response.ok) {
                    const results = await response.json();
                    setSearchResults(
                        (results || []).map((r: { user_id: string; username: string; display_name?: string; avatar_url?: string }) => ({
                            id: r.user_id,
                            username: r.username,
                            displayName: r.display_name || `@${r.username}`,
                            avatarUrl: r.avatar_url,
                            isOnline: false,
                        }))
                    );
                }
            } catch (error) {
                console.error('Failed to search users:', error);
            } finally {
                setIsSearching(false);
            }
        }, 300);

        return () => clearTimeout(timeoutId);
    }, [addFriendUsername, isAddFriendOpen, token]);

    // Handle starting a chat with a friend
    const handleStartChat = useCallback((friendId: string) => {
        setActiveConversation(friendId);
        // Note: Tab switching would be handled by parent
    }, [setActiveConversation]);

    // Handle sending a friend request
    const handleSendFriendRequest = useCallback(async (userId: string) => {
        setSendingRequest(userId);
        try {
            await sendFriendRequest(userId);
            setIsAddFriendOpen(false);
            setAddFriendUsername('');
            setSearchResults([]);
        } finally {
            setSendingRequest(null);
        }
    }, [sendFriendRequest]);

    // Handle accepting a friend request
    const handleAcceptRequest = useCallback(async (userId: string) => {
        await acceptRequest(userId);
    }, [acceptRequest]);

    // Handle declining a friend request
    const handleDeclineRequest = useCallback(async (userId: string) => {
        await declineRequest(userId);
    }, [declineRequest]);

    // Handle canceling an outgoing request
    const handleCancelRequest = useCallback(async (userId: string) => {
        await cancelRequest(userId);
    }, [cancelRequest]);

    return (
        <div className="h-full flex flex-col has-tab-bar">
            {/* Header */}
            <header className="flex-shrink-0 px-4 pt-4 pb-2">
                <div className="flex items-center justify-between mb-4">
                    <h1 className="text-2xl font-bold">Friends</h1>
                    <Button
                        size="icon"
                        variant="ghost"
                        onClick={() => setIsAddFriendOpen(true)}
                        className="rounded-xl"
                    >
                        <UserPlus className="h-5 w-5" />
                    </Button>
                </div>

                {/* Search */}
                <div className="relative">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-foreground-muted" />
                    <Input
                        type="text"
                        placeholder="Search friends..."
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                        className="pl-10 rounded-xl bg-background-secondary border-0"
                    />
                </div>

                {/* Section tabs */}
                <div className="flex gap-2 mt-4">
                    <Button
                        variant={activeSection === 'all' ? 'default' : 'ghost'}
                        size="sm"
                        onClick={() => setActiveSection('all')}
                        className="rounded-full"
                    >
                        All ({displayFriends.length})
                    </Button>
                    <Button
                        variant={activeSection === 'online' ? 'default' : 'ghost'}
                        size="sm"
                        onClick={() => setActiveSection('online')}
                        className="rounded-full"
                    >
                        <span className="w-2 h-2 rounded-full bg-success mr-2" />
                        Online ({onlineCount})
                    </Button>
                    {pendingRequestsCount > 0 && (
                        <Button
                            variant={activeSection === 'requests' ? 'default' : 'ghost'}
                            size="sm"
                            onClick={() => setActiveSection('requests')}
                            className="rounded-full"
                        >
                            Requests ({pendingRequestsCount})
                        </Button>
                    )}
                </div>
            </header>

            {/* Content */}
            <div className="flex-1 overflow-y-auto px-4 pb-4">
                {isLoading ? (
                    <div className="flex items-center justify-center py-12">
                        <Loader2 className="h-8 w-8 animate-spin text-primary" />
                    </div>
                ) : activeSection === 'requests' ? (
                    // Friend Requests Section
                    <div className="space-y-4 mt-4">
                        {/* Incoming Requests */}
                        {incomingRequests.length > 0 && (
                            <div>
                                <h3 className="text-sm font-medium text-foreground-muted mb-2">
                                    Incoming Requests
                                </h3>
                                <div className="space-y-3">
                                    {incomingRequests.map((request) => (
                                        <FriendRequestCard
                                            key={request.id}
                                            request={request}
                                            onAccept={() => handleAcceptRequest(request.user_id)}
                                            onDecline={() => handleDeclineRequest(request.user_id)}
                                        />
                                    ))}
                                </div>
                            </div>
                        )}

                        {/* Outgoing Requests */}
                        {outgoingRequests.length > 0 && (
                            <div>
                                <h3 className="text-sm font-medium text-foreground-muted mb-2">
                                    Sent Requests
                                </h3>
                                <div className="space-y-3">
                                    {outgoingRequests.map((request) => (
                                        <OutgoingRequestCard
                                            key={request.id}
                                            request={request}
                                            onCancel={() => handleCancelRequest(request.user_id)}
                                        />
                                    ))}
                                </div>
                            </div>
                        )}

                        {pendingRequestsCount === 0 && (
                            <EmptyState
                                icon={UserCheck}
                                title="No pending requests"
                                description="When someone sends you a friend request, it will appear here."
                            />
                        )}
                    </div>
                ) : (
                    // Friends List
                    <div className="space-y-2 mt-4">
                        {filteredFriends.length === 0 ? (
                            <EmptyState
                                icon={Users}
                                title={activeSection === 'online' ? 'No one online' : 'No friends yet'}
                                description={
                                    activeSection === 'online'
                                        ? 'None of your friends are currently online.'
                                        : 'Add friends to see them here.'
                                }
                                actionLabel={activeSection === 'all' ? 'Add Friend' : undefined}
                                onAction={activeSection === 'all' ? () => setIsAddFriendOpen(true) : undefined}
                            />
                        ) : (
                            filteredFriends.map((friend) => (
                                <FriendCard
                                    key={friend.id}
                                    friend={friend}
                                    onMessage={() => handleStartChat(friend.id)}
                                    onCall={() => {/* TODO: Initiate call */ }}
                                    onVideoCall={() => {/* TODO: Initiate video call */ }}
                                />
                            ))
                        )}
                    </div>
                )}
            </div>

            {/* Add Friend Dialog */}
            <Dialog open={isAddFriendOpen} onOpenChange={setIsAddFriendOpen}>
                <DialogContent className="max-w-md">
                    <DialogHeader>
                        <DialogTitle>Add Friend</DialogTitle>
                        <DialogDescription>
                            Search by username and send a friend request.
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
                                value={addFriendUsername.replace(/^@/, '')}
                                onChange={(e) => setAddFriendUsername(e.target.value)}
                                className="pl-8 font-mono"
                                autoFocus
                            />
                        </div>

                        {isSearching && (
                            <p className="text-sm text-foreground-muted text-center py-2">Searching...</p>
                        )}

                        {!isSearching && searchResults.length > 0 && (
                            <div className="border border-border rounded-lg divide-y divide-border max-h-60 overflow-y-auto">
                                {searchResults.map((result) => (
                                    <button
                                        key={result.id}
                                        type="button"
                                        onClick={() => handleSendFriendRequest(result.id)}
                                        disabled={sendingRequest === result.id}
                                        className="w-full px-4 py-3 flex items-center gap-3 hover:bg-background-tertiary transition-colors text-left touch-target disabled:opacity-50"
                                    >
                                        <Avatar className="h-10 w-10">
                                            <AvatarImage src={result.avatarUrl} />
                                            <AvatarFallback>
                                                {result.username.charAt(0).toUpperCase()}
                                            </AvatarFallback>
                                        </Avatar>
                                        <div className="flex-1">
                                            <p className="font-medium">@{result.username}</p>
                                            {result.displayName !== `@${result.username}` && (
                                                <p className="text-sm text-foreground-secondary">{result.displayName}</p>
                                            )}
                                        </div>
                                        {sendingRequest === result.id ? (
                                            <Loader2 className="h-5 w-5 animate-spin text-primary" />
                                        ) : (
                                            <UserPlus className="h-5 w-5 text-primary" />
                                        )}
                                    </button>
                                ))}
                            </div>
                        )}

                        {!isSearching && addFriendUsername.replace(/^@/, '').length >= 3 && searchResults.length === 0 && (
                            <p className="text-sm text-foreground-muted text-center py-4">
                                No users found with that username
                            </p>
                        )}

                        <div className="flex justify-end">
                            <Button
                                type="button"
                                variant="outline"
                                onClick={() => {
                                    setIsAddFriendOpen(false);
                                    setAddFriendUsername('');
                                }}
                            >
                                Cancel
                            </Button>
                        </div>
                    </div>
                </DialogContent>
            </Dialog>
        </div>
    );
}

/**
 * Friend Card Component
 */
function FriendCard({
    friend,
    onMessage,
    onCall,
    onVideoCall,
}: {
    friend: {
        id: string;
        username: string;
        displayName: string;
        avatarUrl?: string;
        isOnline: boolean;
        lastSeen?: number;
    };
    onMessage: () => void;
    onCall: () => void;
    onVideoCall: () => void;
}) {
    return (
        <GlassCard
            variant="subtle"
            press
            className="p-3 flex items-center gap-3"
        >
            {/* Avatar with status */}
            <AvatarRing
                src={friend.avatarUrl}
                alt={friend.displayName}
                fallback={friend.displayName.charAt(0)}
                size="lg"
                isOnline={friend.isOnline}
            />

            {/* Info */}
            <div className="flex-1 min-w-0">
                <p className="font-medium truncate">{friend.displayName}</p>
                <p className="text-sm text-foreground-muted truncate">
                    {friend.isOnline ? (
                        <span className="text-success">Online</span>
                    ) : friend.lastSeen ? (
                        `Last seen ${formatLastSeen(friend.lastSeen)}`
                    ) : (
                        '@' + friend.username
                    )}
                </p>
            </div>

            {/* Actions */}
            <div className="flex gap-1">
                <Button
                    size="icon"
                    variant="ghost"
                    onClick={onMessage}
                    className="h-9 w-9 rounded-full"
                >
                    <MessageSquare className="h-4 w-4" />
                </Button>
                <Button
                    size="icon"
                    variant="ghost"
                    onClick={onCall}
                    className="h-9 w-9 rounded-full"
                >
                    <Phone className="h-4 w-4" />
                </Button>
                <Button
                    size="icon"
                    variant="ghost"
                    onClick={onVideoCall}
                    className="h-9 w-9 rounded-full"
                >
                    <Video className="h-4 w-4" />
                </Button>
            </div>
        </GlassCard>
    );
}

/**
 * Friend Request Card Component
 */
function FriendRequestCard({
    request,
    onAccept,
    onDecline,
}: {
    request: FriendRequest;
    onAccept: () => void;
    onDecline: () => void;
}) {
    return (
        <GlassCard variant="default" className="p-4">
            <div className="flex items-center gap-3">
                <AvatarRing
                    src={request.avatar_url}
                    alt={request.display_name || request.username}
                    fallback={(request.display_name || request.username).charAt(0)}
                    size="lg"
                />

                <div className="flex-1 min-w-0">
                    <p className="font-medium truncate">{request.display_name || `@${request.username}`}</p>
                    <p className="text-sm text-foreground-muted flex items-center gap-1">
                        <Clock className="h-3 w-3" />
                        {formatLastSeen(new Date(request.created_at).getTime())}
                    </p>
                </div>
            </div>

            <div className="flex gap-2 mt-3">
                <Button
                    variant="default"
                    size="sm"
                    onClick={onAccept}
                    className="flex-1 rounded-xl"
                >
                    <UserCheck className="h-4 w-4 mr-2" />
                    Accept
                </Button>
                <Button
                    variant="outline"
                    size="sm"
                    onClick={onDecline}
                    className="flex-1 rounded-xl"
                >
                    <UserX className="h-4 w-4 mr-2" />
                    Decline
                </Button>
            </div>
        </GlassCard>
    );
}

/**
 * Outgoing Request Card Component
 */
function OutgoingRequestCard({
    request,
    onCancel,
}: {
    request: FriendRequest;
    onCancel: () => void;
}) {
    return (
        <GlassCard variant="subtle" className="p-4">
            <div className="flex items-center gap-3">
                <AvatarRing
                    src={request.avatar_url}
                    alt={request.display_name || request.username}
                    fallback={(request.display_name || request.username).charAt(0)}
                    size="lg"
                />

                <div className="flex-1 min-w-0">
                    <p className="font-medium truncate">{request.display_name || `@${request.username}`}</p>
                    <p className="text-sm text-foreground-muted">
                        Request sent {formatLastSeen(new Date(request.created_at).getTime())}
                    </p>
                </div>

                <Button
                    size="icon"
                    variant="ghost"
                    onClick={onCancel}
                    className="h-9 w-9 rounded-full text-foreground-muted hover:text-destructive"
                >
                    <X className="h-4 w-4" />
                </Button>
            </div>
        </GlassCard>
    );
}

/**
 * Format last seen timestamp
 */
function formatLastSeen(timestamp: number): string {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMins / 60);
    const diffDays = Math.floor(diffHours / 24);

    if (diffMins < 1) return 'just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;

    return date.toLocaleDateString();
}

export default ContactsTab;
