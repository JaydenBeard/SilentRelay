/**
 * Calls Tab
 * 
 * Call history with:
 * - Recent calls list with voice/video type icons
 * - Call duration and timestamp
 * - Missed calls highlighted
 * - Quick redial functionality
 */

import { useState, useMemo, useCallback } from 'react';
import { cn } from '@/lib/utils';
import { useWebSocket } from '@/hooks/useWebSocket';
import { useChatStore } from '@/core/store/chatStore';

// UI Components
import { GlassCard } from '@/components/ui/GlassCard';
import { AvatarRing } from '@/components/ui/AvatarRing';
import { EmptyState } from '@/components/ui/EmptyState';
import { Button } from '@/components/ui/button';
import {
    Phone,
    PhoneIncoming,
    PhoneOutgoing,
    PhoneMissed,
    Video,
} from 'lucide-react';

// Call types
type CallType = 'audio' | 'video';
type CallDirection = 'incoming' | 'outgoing';
type CallStatus = 'answered' | 'missed' | 'declined';

interface CallRecord {
    id: string;
    recipientId: string;
    recipientName: string;
    recipientAvatar?: string;
    type: CallType;
    direction: CallDirection;
    status: CallStatus;
    duration?: number; // in seconds
    timestamp: string;
}

// Mock call history (would be stored in a store or fetched from API)
const mockCallHistory: CallRecord[] = [
    // Empty for now - will show empty state
];

export function CallsTab() {
    const { startCall } = useWebSocket();
    const { conversations } = useChatStore();

    // State
    const [filter, setFilter] = useState<'all' | 'missed'>('all');
    const [callHistory, setCallHistory] = useState<CallRecord[]>(mockCallHistory);

    // Filter calls
    const filteredCalls = useMemo(() => {
        if (filter === 'missed') {
            return callHistory.filter(c => c.status === 'missed');
        }
        return callHistory;
    }, [callHistory, filter]);

    // Group calls by date
    const groupedCalls = useMemo(() => {
        const groups: { [key: string]: CallRecord[] } = {};

        filteredCalls.forEach(call => {
            const date = new Date(call.timestamp);
            const today = new Date();
            const yesterday = new Date(today);
            yesterday.setDate(yesterday.getDate() - 1);

            let key: string;
            if (date.toDateString() === today.toDateString()) {
                key = 'Today';
            } else if (date.toDateString() === yesterday.toDateString()) {
                key = 'Yesterday';
            } else {
                key = date.toLocaleDateString('en-US', {
                    weekday: 'long',
                    month: 'short',
                    day: 'numeric'
                });
            }

            if (!groups[key]) groups[key] = [];
            groups[key].push(call);
        });

        return groups;
    }, [filteredCalls]);

    // Handle callback
    const handleCallback = useCallback((call: CallRecord) => {
        const conversation = conversations[call.recipientId];
        startCall(
            call.recipientId,
            call.recipientName,
            call.recipientAvatar || conversation?.recipientAvatar,
            call.type
        );
    }, [startCall, conversations]);

    // Handle clear all calls
    const handleClearAll = useCallback(() => {
        setCallHistory([]);
    }, []);

    // Count missed calls
    const missedCount = useMemo(() =>
        callHistory.filter(c => c.status === 'missed').length,
        [callHistory]
    );

    return (
        <div className="h-full flex flex-col has-tab-bar">
            {/* Header */}
            <header className="flex-shrink-0 px-4 pt-4 pb-2">
                <div className="flex items-center justify-between mb-4">
                    <h1 className="text-2xl font-bold">Calls</h1>
                    {callHistory.length > 0 && (
                        <Button
                            size="sm"
                            variant="ghost"
                            onClick={handleClearAll}
                            className="text-foreground-muted"
                        >
                            Clear All
                        </Button>
                    )}
                </div>

                {/* Filter tabs */}
                <div className="flex gap-2">
                    <Button
                        variant={filter === 'all' ? 'default' : 'ghost'}
                        size="sm"
                        onClick={() => setFilter('all')}
                        className="rounded-full"
                    >
                        All
                    </Button>
                    <Button
                        variant={filter === 'missed' ? 'default' : 'ghost'}
                        size="sm"
                        onClick={() => setFilter('missed')}
                        className={cn(
                            'rounded-full',
                            missedCount > 0 && filter !== 'missed' && 'text-destructive'
                        )}
                    >
                        <PhoneMissed className="h-4 w-4 mr-1" />
                        Missed {missedCount > 0 && `(${missedCount})`}
                    </Button>
                </div>
            </header>

            {/* Content */}
            <div className="flex-1 overflow-y-auto px-4 pb-4">
                {filteredCalls.length === 0 ? (
                    <div className="h-full flex items-center justify-center">
                        <EmptyState
                            icon={Phone}
                            title={filter === 'missed' ? 'No missed calls' : 'No call history'}
                            description={
                                filter === 'missed'
                                    ? "You haven't missed any calls."
                                    : "Your call history will appear here."
                            }
                        />
                    </div>
                ) : (
                    <div className="space-y-6 mt-4">
                        {Object.entries(groupedCalls).map(([date, calls]) => (
                            <div key={date}>
                                <h3 className="text-sm font-medium text-foreground-muted mb-2 px-1">
                                    {date}
                                </h3>
                                <div className="space-y-2">
                                    {calls.map((call) => (
                                        <CallHistoryItem
                                            key={call.id}
                                            call={call}
                                            onCallback={() => handleCallback(call)}
                                        />
                                    ))}
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </div>

            {/* Start New Call FAB would go here if needed */}
        </div>
    );
}

/**
 * Call History Item Component
 */
function CallHistoryItem({
    call,
    onCallback,
}: {
    call: CallRecord;
    onCallback: () => void;
}) {
    // Get call icon based on direction and status
    const CallIcon = useMemo(() => {
        if (call.status === 'missed') return PhoneMissed;
        if (call.direction === 'incoming') return PhoneIncoming;
        return PhoneOutgoing;
    }, [call.direction, call.status]);

    // Format duration
    const formattedDuration = useMemo(() => {
        if (!call.duration) return null;
        const mins = Math.floor(call.duration / 60);
        const secs = call.duration % 60;
        if (mins === 0) return `${secs}s`;
        return `${mins}:${secs.toString().padStart(2, '0')}`;
    }, [call.duration]);

    // Format time
    const formattedTime = useMemo(() => {
        return new Date(call.timestamp).toLocaleTimeString('en-US', {
            hour: 'numeric',
            minute: '2-digit',
            hour12: true,
        });
    }, [call.timestamp]);

    return (
        <GlassCard
            variant="subtle"
            press
            className="p-3 flex items-center gap-3"
        >
            {/* Avatar */}
            <AvatarRing
                src={call.recipientAvatar}
                alt={call.recipientName}
                fallback={call.recipientName.charAt(0)}
                size="lg"
            />

            {/* Info */}
            <div className="flex-1 min-w-0">
                <p className={cn(
                    'font-medium truncate',
                    call.status === 'missed' && 'text-destructive'
                )}>
                    {call.recipientName}
                </p>
                <div className="flex items-center gap-2 text-sm text-foreground-muted">
                    <CallIcon className={cn(
                        'h-3.5 w-3.5',
                        call.status === 'missed' && 'text-destructive'
                    )} />
                    <span>{call.type === 'video' ? 'Video' : 'Voice'}</span>
                    {formattedDuration && (
                        <>
                            <span>•</span>
                            <span>{formattedDuration}</span>
                        </>
                    )}
                    <span>•</span>
                    <span>{formattedTime}</span>
                </div>
            </div>

            {/* Actions */}
            <div className="flex gap-1">
                <Button
                    size="icon"
                    variant="ghost"
                    onClick={onCallback}
                    className="h-10 w-10 rounded-full"
                >
                    {call.type === 'video' ? (
                        <Video className="h-5 w-5 text-primary" />
                    ) : (
                        <Phone className="h-5 w-5 text-primary" />
                    )}
                </Button>
            </div>
        </GlassCard>
    );
}

export default CallsTab;
