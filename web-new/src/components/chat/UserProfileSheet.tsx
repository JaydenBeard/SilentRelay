/**
 * UserProfileSheet Component
 *
 * Shows user profile information in a modal/sheet:
 * - Avatar and display name
 * - Username (copyable)
 * - Online status
 * - Friend status
 * - Actions: Add Friend, Message, Call, Block
 */

import { useState, useCallback } from 'react';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Button } from '@/components/ui/button';
import { GlassCard } from '@/components/ui/GlassCard';
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/dialog';
import {
    Copy,
    Check,
    UserPlus,
    UserCheck,
    MessageSquare,
    Phone,
    Video,
    Shield,
    Ban,
    MoreHorizontal,
} from 'lucide-react';
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';

interface UserProfileSheetProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    user: {
        id: string;
        username: string;
        displayName: string;
        avatarUrl?: string;
        isOnline?: boolean;
        lastSeen?: number;
        bio?: string;
    };
    isFriend: boolean;
    onSendFriendRequest: () => Promise<void>;
    onMessage: () => void;
    onVoiceCall: () => void;
    onVideoCall: () => void;
    onBlock?: () => void;
}

export function UserProfileSheet({
    open,
    onOpenChange,
    user,
    isFriend,
    onSendFriendRequest,
    onMessage,
    onVoiceCall,
    onVideoCall,
    onBlock,
}: UserProfileSheetProps) {
    const [copiedUsername, setCopiedUsername] = useState(false);
    const [isSendingRequest, setIsSendingRequest] = useState(false);
    const [friendRequestSent, setFriendRequestSent] = useState(false);

    // Copy username to clipboard
    const handleCopyUsername = useCallback(() => {
        navigator.clipboard.writeText(`@${user.username}`);
        setCopiedUsername(true);
        setTimeout(() => setCopiedUsername(false), 2000);
    }, [user.username]);

    // Send friend request
    const handleSendFriendRequest = useCallback(async () => {
        if (isSendingRequest || friendRequestSent) return;

        setIsSendingRequest(true);
        try {
            await onSendFriendRequest();
            setFriendRequestSent(true);
        } catch (error) {
            console.error('Failed to send friend request:', error);
        } finally {
            setIsSendingRequest(false);
        }
    }, [onSendFriendRequest, isSendingRequest, friendRequestSent]);

    // Format last seen
    const formatLastSeen = (timestamp: number): string => {
        const now = Date.now();
        const diff = now - timestamp;
        const minutes = Math.floor(diff / 60000);
        const hours = Math.floor(diff / 3600000);
        const days = Math.floor(diff / 86400000);

        if (minutes < 1) return 'just now';
        if (minutes < 60) return `${minutes}m ago`;
        if (hours < 24) return `${hours}h ago`;
        if (days < 7) return `${days}d ago`;

        return new Date(timestamp).toLocaleDateString();
    };

    // Get initials
    const initials = user.displayName?.charAt(0) || user.username?.charAt(0) || '?';

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="max-w-sm p-0 overflow-hidden">
                {/* Header with gradient background */}
                <div className="relative h-24 bg-gradient-to-br from-primary/20 via-primary/10 to-transparent">
                    <div className="absolute inset-0 bg-gradient-to-t from-background to-transparent" />
                </div>

                {/* Profile content */}
                <div className="px-6 pb-6 -mt-12 relative">
                    {/* Avatar */}
                    <div className="flex justify-center mb-4">
                        <div className="relative">
                            <Avatar className="h-24 w-24 border-4 border-background shadow-xl">
                                <AvatarImage src={user.avatarUrl} alt={user.displayName} />
                                <AvatarFallback className="text-2xl font-semibold bg-gradient-to-br from-primary/20 to-primary/10">
                                    {initials.toUpperCase()}
                                </AvatarFallback>
                            </Avatar>
                            {/* Online indicator */}
                            {user.isOnline && (
                                <span className="absolute bottom-1 right-1 w-5 h-5 bg-emerald-500 border-3 border-background rounded-full shadow-lg" />
                            )}
                        </div>
                    </div>

                    {/* Name and username */}
                    <div className="text-center mb-4">
                        <DialogHeader className="mb-1">
                            <DialogTitle className="text-xl">{user.displayName}</DialogTitle>
                        </DialogHeader>
                        <button
                            onClick={handleCopyUsername}
                            className="inline-flex items-center gap-1.5 text-foreground-muted hover:text-foreground transition-colors"
                        >
                            <span className="font-mono">@{user.username}</span>
                            {copiedUsername ? (
                                <Check className="h-3.5 w-3.5 text-success" />
                            ) : (
                                <Copy className="h-3.5 w-3.5" />
                            )}
                        </button>
                    </div>

                    {/* Status */}
                    <div className="flex justify-center mb-6">
                        {user.isOnline ? (
                            <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-emerald-500/10 text-emerald-500 text-sm font-medium">
                                <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
                                Online
                            </span>
                        ) : user.lastSeen ? (
                            <span className="text-sm text-foreground-muted">
                                Last seen {formatLastSeen(user.lastSeen)}
                            </span>
                        ) : null}
                    </div>

                    {/* Bio if exists */}
                    {user.bio && (
                        <p className="text-center text-foreground-secondary mb-6 text-sm">
                            {user.bio}
                        </p>
                    )}

                    {/* Friend status card */}
                    <GlassCard variant="subtle" className="p-3 mb-4">
                        <div className="flex items-center justify-between">
                            <div className="flex items-center gap-2">
                                {isFriend ? (
                                    <>
                                        <UserCheck className="h-4 w-4 text-success" />
                                        <span className="text-sm font-medium text-success">Friends</span>
                                    </>
                                ) : friendRequestSent ? (
                                    <>
                                        <UserPlus className="h-4 w-4 text-primary" />
                                        <span className="text-sm font-medium text-primary">Request Sent</span>
                                    </>
                                ) : (
                                    <>
                                        <UserPlus className="h-4 w-4 text-foreground-muted" />
                                        <span className="text-sm text-foreground-muted">Not friends yet</span>
                                    </>
                                )}
                            </div>
                            {!isFriend && !friendRequestSent && (
                                <Button
                                    size="sm"
                                    onClick={handleSendFriendRequest}
                                    disabled={isSendingRequest}
                                    className="rounded-full"
                                >
                                    {isSendingRequest ? 'Sending...' : 'Add Friend'}
                                </Button>
                            )}
                        </div>
                    </GlassCard>

                    {/* Encryption badge */}
                    <div className="flex justify-center mb-6">
                        <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-primary/5 border border-primary/10">
                            <Shield className="h-3.5 w-3.5 text-primary" />
                            <span className="text-xs text-primary font-medium">End-to-end encrypted</span>
                        </div>
                    </div>

                    {/* Action buttons */}
                    <div className="flex gap-2">
                        <Button
                            variant="outline"
                            className="flex-1 rounded-xl"
                            onClick={() => {
                                onMessage();
                                onOpenChange(false);
                            }}
                        >
                            <MessageSquare className="h-4 w-4 mr-2" />
                            Message
                        </Button>
                        <Button
                            variant="outline"
                            size="icon"
                            className="rounded-xl"
                            onClick={onVoiceCall}
                        >
                            <Phone className="h-4 w-4" />
                        </Button>
                        <Button
                            variant="outline"
                            size="icon"
                            className="rounded-xl"
                            onClick={onVideoCall}
                        >
                            <Video className="h-4 w-4" />
                        </Button>
                        <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                                <Button variant="outline" size="icon" className="rounded-xl">
                                    <MoreHorizontal className="h-4 w-4" />
                                </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                                {onBlock && (
                                    <DropdownMenuItem
                                        onClick={onBlock}
                                        className="text-destructive focus:text-destructive"
                                    >
                                        <Ban className="h-4 w-4 mr-2" />
                                        Block User
                                    </DropdownMenuItem>
                                )}
                            </DropdownMenuContent>
                        </DropdownMenu>
                    </div>
                </div>
            </DialogContent>
        </Dialog>
    );
}

export default UserProfileSheet;
