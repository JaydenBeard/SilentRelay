/**
 * Contacts Tab
 * 
 * Friends/Contacts management with:
 * - Contact list with glassmorphism cards
 * - Online status indicators with pulse animations
 * - Friend requests section (received/sent)
 * - Add friends functionality
 */

import { useState, useMemo, useCallback, useEffect } from 'react';
import { cn } from '@/lib/utils';
import { useAuthStore } from '@/core/store/authStore';
import { useChatStore } from '@/core/store/chatStore';

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
} from 'lucide-react';

// Mock data for contacts (will be replaced with real API calls)
interface Contact {
    id: string;
    username: string;
    displayName: string;
    avatarUrl?: string;
    isOnline: boolean;
    lastSeen?: string;
    hasStory?: boolean;
    storyViewed?: boolean;
}

// Mock friend requests
interface FriendRequest {
    id: string;
    username: string;
    displayName: string;
    avatarUrl?: string;
    createdAt: string;
    type: 'received' | 'sent';
}

export function ContactsTab() {
    const { user, token } = useAuthStore();
    const { conversations, setActiveConversation } = useChatStore();

    // State
    const [searchQuery, setSearchQuery] = useState('');
    const [isAddFriendOpen, setIsAddFriendOpen] = useState(false);
    const [addFriendUsername, setAddFriendUsername] = useState('');
    const [searchResults, setSearchResults] = useState<Contact[]>([]);
    const [isSearching, setIsSearching] = useState(false);
    const [activeSection, setActiveSection] = useState<'all' | 'online' | 'requests'>('all');

    // Derive contacts from conversations (users we've chatted with)
    const contacts = useMemo<Contact[]>(() => {
        return Object.values(conversations)
            .filter(c => c.status === 'accepted')
            .map(conv => ({
                id: conv.recipientId,
                username: conv.recipientName.replace(/^@/, ''),
                displayName: conv.recipientName,
                avatarUrl: conv.recipientAvatar,
                isOnline: conv.isOnline ?? false,
                lastSeen: conv.lastSeen,
                hasStory: false, // TODO: Integrate with stories
                storyViewed: false,
            }));
    }, [conversations]);

    // Filter contacts based on search and section
    const filteredContacts = useMemo(() => {
        let result = contacts;

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
    }, [contacts, activeSection, searchQuery]);

    // Mock friend requests (would come from API)
    const friendRequests = useMemo<FriendRequest[]>(() => {
        // Get pending conversations as received requests
        return Object.values(conversations)
            .filter(c => c.status === 'pending')
            .map(conv => ({
                id: conv.recipientId,
                username: conv.recipientName.replace(/^@/, ''),
                displayName: conv.recipientName,
                avatarUrl: conv.recipientAvatar,
                createdAt: conv.lastMessage?.timestamp || new Date().toISOString(),
                type: 'received' as const,
            }));
    }, [conversations]);

    // Count online contacts
    const onlineCount = useMemo(() =>
        contacts.filter(c => c.isOnline).length,
        [contacts]
    );

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

    // Handle starting a chat with a contact
    const handleStartChat = useCallback((contactId: string) => {
        setActiveConversation(contactId);
        // Note: Tab switching would be handled by parent
    }, [setActiveConversation]);

    // Handle adding a friend (start conversation)
    const handleAddFriend = useCallback((contact: Contact) => {
        // For now, this just starts a conversation
        // In future, this could send a friend request
        setIsAddFriendOpen(false);
        setAddFriendUsername('');
        handleStartChat(contact.id);
    }, [handleStartChat]);

    return (
        <div className="h-full flex flex-col has-tab-bar">
            {/* Header */}
            <header className="flex-shrink-0 px-4 pt-4 pb-2">
                <div className="flex items-center justify-between mb-4">
                    <h1 className="text-2xl font-bold">Contacts</h1>
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
                        placeholder="Search contacts..."
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
                        All ({contacts.length})
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
                    {friendRequests.length > 0 && (
                        <Button
                            variant={activeSection === 'requests' ? 'default' : 'ghost'}
                            size="sm"
                            onClick={() => setActiveSection('requests')}
                            className="rounded-full"
                        >
                            Requests ({friendRequests.length})
                        </Button>
                    )}
                </div>
            </header>

            {/* Content */}
            <div className="flex-1 overflow-y-auto px-4 pb-4">
                {activeSection === 'requests' ? (
                    // Friend Requests Section
                    <div className="space-y-3 mt-4">
                        {friendRequests.length === 0 ? (
                            <EmptyState
                                icon={UserCheck}
                                title="No pending requests"
                                description="When someone sends you a friend request, it will appear here."
                            />
                        ) : (
                            friendRequests.map((request) => (
                                <FriendRequestCard
                                    key={request.id}
                                    request={request}
                                    onAccept={() => {/* TODO */ }}
                                    onDecline={() => {/* TODO */ }}
                                />
                            ))
                        )}
                    </div>
                ) : (
                    // Contacts List
                    <div className="space-y-2 mt-4">
                        {filteredContacts.length === 0 ? (
                            <EmptyState
                                icon={Users}
                                title={activeSection === 'online' ? 'No one online' : 'No contacts yet'}
                                description={
                                    activeSection === 'online'
                                        ? 'None of your contacts are currently online.'
                                        : 'Start a conversation to add contacts.'
                                }
                                actionLabel={activeSection === 'all' ? 'Add Friend' : undefined}
                                onAction={activeSection === 'all' ? () => setIsAddFriendOpen(true) : undefined}
                            />
                        ) : (
                            filteredContacts.map((contact) => (
                                <ContactCard
                                    key={contact.id}
                                    contact={contact}
                                    onMessage={() => handleStartChat(contact.id)}
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
                            Search for someone by their username to add them as a friend.
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
                                        onClick={() => handleAddFriend(result)}
                                        className="w-full px-4 py-3 flex items-center gap-3 hover:bg-background-tertiary transition-colors text-left touch-target"
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
                                        <UserPlus className="h-5 w-5 text-primary" />
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
 * Contact Card Component
 */
function ContactCard({
    contact,
    onMessage,
    onCall,
    onVideoCall,
}: {
    contact: Contact;
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
                src={contact.avatarUrl}
                alt={contact.displayName}
                fallback={contact.displayName.charAt(0)}
                size="lg"
                isOnline={contact.isOnline}
                hasStory={contact.hasStory}
                storyViewed={contact.storyViewed}
            />

            {/* Info */}
            <div className="flex-1 min-w-0">
                <p className="font-medium truncate">{contact.displayName}</p>
                <p className="text-sm text-foreground-muted truncate">
                    {contact.isOnline ? (
                        <span className="text-success">Online</span>
                    ) : contact.lastSeen ? (
                        `Last seen ${formatLastSeen(contact.lastSeen)}`
                    ) : (
                        '@' + contact.username
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
                    src={request.avatarUrl}
                    alt={request.displayName}
                    fallback={request.displayName.charAt(0)}
                    size="lg"
                />

                <div className="flex-1 min-w-0">
                    <p className="font-medium truncate">{request.displayName}</p>
                    <p className="text-sm text-foreground-muted flex items-center gap-1">
                        <Clock className="h-3 w-3" />
                        {formatLastSeen(request.createdAt)}
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
 * Format last seen timestamp
 */
function formatLastSeen(timestamp: string): string {
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
