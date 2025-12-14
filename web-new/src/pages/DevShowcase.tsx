/**
 * Dev Showcase Page
 *
 * Displays all UI components with mock data for quick visual reference.
 * Accessible at /dev without authentication.
 */

import { useState } from 'react';
import { Link } from 'react-router-dom';
import { Shield, ArrowLeft, Moon, Sun } from 'lucide-react';

// Chat components
import { MessageBubble, DateSeparator } from '@/components/chat/MessageBubble';

// UI components
import { Button } from '@/components/ui/button';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Input } from '@/components/ui/input';

// Types
import type { Message, Conversation } from '@/core/types';

// ============================================
// MOCK DATA
// ============================================

const mockMessages: Message[] = [
    {
        id: '1',
        conversationId: 'conv-1',
        senderId: 'other-user',
        content: 'Hey! How are you doing?',
        timestamp: Date.now() - 3600000,
        status: 'read',
        type: 'text',
    },
    {
        id: '2',
        conversationId: 'conv-1',
        senderId: 'current-user',
        content: 'I\'m doing great, thanks for asking! Just finished working on some new features.',
        timestamp: Date.now() - 3500000,
        status: 'read',
        type: 'text',
    },
    {
        id: '3',
        conversationId: 'conv-1',
        senderId: 'other-user',
        content: 'That sounds awesome! What features?',
        timestamp: Date.now() - 3400000,
        status: 'read',
        type: 'text',
    },
    {
        id: '4',
        conversationId: 'conv-1',
        senderId: 'current-user',
        content: 'End-to-end encryption improvements and a new dev showcase page 🚀',
        timestamp: Date.now() - 60000,
        status: 'delivered',
        type: 'text',
    },
    {
        id: '5',
        conversationId: 'conv-1',
        senderId: 'current-user',
        content: 'Sending now...',
        timestamp: Date.now(),
        status: 'sending',
        type: 'text',
    },
];

const mockConversations: Conversation[] = [
    {
        id: 'conv-1',
        recipientId: 'user-1',
        recipientName: 'Alice Johnson',
        recipientAvatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=Alice',
        lastMessage: mockMessages[3],
        unreadCount: 2,
        isOnline: true,
        isPinned: true,
        isMuted: false,
        status: 'accepted',
    },
    {
        id: 'conv-2',
        recipientId: 'user-2',
        recipientName: 'Bob Smith',
        recipientAvatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=Bob',
        lastMessage: {
            id: 'msg-2',
            conversationId: 'conv-2',
            senderId: 'user-2',
            content: 'Sure, let\'s meet tomorrow!',
            timestamp: Date.now() - 7200000,
            status: 'read',
            type: 'text',
        },
        unreadCount: 0,
        isOnline: false,
        lastSeen: Date.now() - 1800000,
        isPinned: false,
        isMuted: false,
        status: 'accepted',
    },
    {
        id: 'conv-3',
        recipientId: 'user-3',
        recipientName: 'Carol Davis',
        unreadCount: 0,
        isOnline: false,
        isPinned: false,
        isMuted: true,
        status: 'pending',
    },
];

// ============================================
// SECTION COMPONENTS
// ============================================

function SectionHeader({ id, title, description }: { id: string; title: string; description: string }) {
    return (
        <div id={id} className="mb-6 pt-8">
            <h2 className="text-2xl font-bold mb-2">{title}</h2>
            <p className="text-foreground-muted">{description}</p>
        </div>
    );
}

function ComponentCard({ title, children }: { title: string; children: React.ReactNode }) {
    return (
        <div className="border border-border rounded-xl p-4 mb-4">
            <h3 className="text-sm font-medium text-foreground-muted mb-4">{title}</h3>
            {children}
        </div>
    );
}

// ============================================
// CALL PREVIEW COMPONENTS (Standalone, no store dependency)
// ============================================

function IncomingCallPreview() {
    return (
        <div className="relative bg-background/95 rounded-2xl p-8 flex flex-col items-center justify-center border border-border">
            <Avatar className="h-24 w-24 mb-4 ring-4 ring-primary/20">
                <AvatarImage src="https://api.dicebear.com/7.x/avataaars/svg?seed=Alice" />
                <AvatarFallback className="text-3xl">A</AvatarFallback>
            </Avatar>
            <h2 className="text-xl font-bold">Alice Johnson</h2>
            <p className="text-foreground-muted mt-1 mb-6">Incoming video call...</p>
            <div className="flex items-center gap-6">
                <button className="p-4 rounded-full bg-destructive text-white">
                    <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M10.68 13.31a16 16 0 0 0 3.41 2.6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7 2 2 0 0 1 1.72 2v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.42 19.42 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.63A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91" /><line x1="23" x2="1" y1="1" y2="23" /></svg>
                </button>
                <button className="p-4 rounded-full bg-success text-white animate-pulse">
                    <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z" /></svg>
                </button>
            </div>
        </div>
    );
}

function AudioCallPreview() {
    return (
        <div className="relative bg-gradient-to-b from-background to-background-secondary rounded-2xl p-8 flex flex-col items-center justify-center border border-border">
            <Avatar className="h-28 w-28 mb-4">
                <AvatarImage src="https://api.dicebear.com/7.x/avataaars/svg?seed=Bob" />
                <AvatarFallback className="text-4xl">B</AvatarFallback>
            </Avatar>
            <h2 className="text-2xl font-bold">Bob Smith</h2>
            <p className="text-foreground-muted mt-1 mb-6">2:34</p>
            <div className="flex items-center gap-4">
                <div className="flex flex-col items-center gap-1">
                    <button className="p-3 rounded-full bg-background-tertiary">
                        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3Z" /><path d="M19 10v2a7 7 0 0 1-14 0v-2" /><line x1="12" x2="12" y1="19" y2="22" /></svg>
                    </button>
                    <span className="text-xs text-foreground-muted">Mute</span>
                </div>
                <button className="p-3 rounded-full bg-destructive text-white">
                    <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M10.68 13.31a16 16 0 0 0 3.41 2.6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7 2 2 0 0 1 1.72 2v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.42 19.42 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.63A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91" /><line x1="23" x2="1" y1="1" y2="23" /></svg>
                </button>
                <div className="flex flex-col items-center gap-1">
                    <button className="p-3 rounded-full bg-foreground text-background">
                        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="1" x2="23" y1="1" y2="23" /><path d="M9 9v3a3 3 0 0 0 5.12 2.12M15 9.34V5a3 3 0 0 0-5.94-.6" /><path d="M17 16.95A7 7 0 0 1 5 12v-2m14 0v2a7 7 0 0 1-.11 1.23" /><line x1="12" x2="12" y1="19" y2="22" /></svg>
                    </button>
                    <span className="text-xs text-foreground-muted">Muted</span>
                </div>
            </div>
        </div>
    );
}

function VideoCallPreview() {
    return (
        <div className="relative rounded-2xl overflow-hidden border border-border" style={{ aspectRatio: '16/10' }}>
            {/* Remote video placeholder */}
            <div className="absolute inset-0 bg-gradient-to-br from-gray-800 to-gray-900 flex items-center justify-center">
                <Avatar className="h-24 w-24">
                    <AvatarImage src="https://api.dicebear.com/7.x/avataaars/svg?seed=Carol" />
                    <AvatarFallback className="text-3xl">C</AvatarFallback>
                </Avatar>
            </div>
            {/* Local video PiP */}
            <div className="absolute top-3 right-3 w-24 h-32 rounded-lg overflow-hidden bg-background-tertiary border border-border shadow-lg">
                <div className="w-full h-full flex items-center justify-center">
                    <Avatar className="h-12 w-12">
                        <AvatarFallback>You</AvatarFallback>
                    </Avatar>
                </div>
            </div>
            {/* Controls */}
            <div className="absolute bottom-0 left-0 right-0 p-4 bg-gradient-to-t from-black/80 to-transparent">
                <div className="flex items-center justify-center gap-4">
                    <button className="p-3 rounded-full bg-white/20 backdrop-blur-sm">
                        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3Z" /><path d="M19 10v2a7 7 0 0 1-14 0v-2" /><line x1="12" x2="12" y1="19" y2="22" /></svg>
                    </button>
                    <button className="p-3 rounded-full bg-white/20 backdrop-blur-sm">
                        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="m16 13 5.223 3.482a.5.5 0 0 0 .777-.416V7.87a.5.5 0 0 0-.752-.432L16 10.5" /><rect x="2" y="6" width="14" height="12" rx="2" /></svg>
                    </button>
                    <button className="p-4 rounded-full bg-destructive text-white">
                        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M10.68 13.31a16 16 0 0 0 3.41 2.6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7 2 2 0 0 1 1.72 2v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.42 19.42 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.63A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91" /><line x1="23" x2="1" y1="1" y2="23" /></svg>
                    </button>
                    <button className="p-3 rounded-full bg-white/20 backdrop-blur-sm">
                        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3m0 18h3a2 2 0 0 0 2-2v-3M3 16v3a2 2 0 0 0 2 2h3" /></svg>
                    </button>
                </div>
                <p className="text-center text-white/70 mt-2 text-sm">Carol Davis • 5:21</p>
            </div>
        </div>
    );
}

function CallEndedPreview() {
    return (
        <div className="bg-background-secondary border border-border rounded-xl shadow-lg p-4 flex items-center gap-4 max-w-sm">
            <Avatar className="h-12 w-12">
                <AvatarImage src="https://api.dicebear.com/7.x/avataaars/svg?seed=Alice" />
                <AvatarFallback>A</AvatarFallback>
            </Avatar>
            <div className="flex-1">
                <p className="font-medium">Alice Johnson</p>
                <p className="text-sm text-foreground-muted">Call ended · 2:34</p>
            </div>
            <button className="p-2 hover:bg-background-tertiary rounded-lg transition-colors">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M18 6 6 18" /><path d="m6 6 12 12" /></svg>
            </button>
        </div>
    );
}

// ============================================
// CONVERSATION ITEM PREVIEW
// ============================================

function ConversationItemPreview({ conversation }: { conversation: Conversation }) {
    const formatTime = (timestamp?: number) => {
        if (!timestamp) return '';
        const date = new Date(timestamp);
        return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    };

    return (
        <div className="flex items-center gap-3 p-3 hover:bg-background-tertiary rounded-xl transition-colors cursor-pointer">
            <div className="relative">
                <Avatar className="h-12 w-12">
                    <AvatarImage src={conversation.recipientAvatar} />
                    <AvatarFallback>{conversation.recipientName.charAt(0)}</AvatarFallback>
                </Avatar>
                {conversation.isOnline && (
                    <div className="absolute bottom-0 right-0 w-3 h-3 rounded-full bg-success border-2 border-background-secondary" />
                )}
            </div>
            <div className="flex-1 min-w-0">
                <div className="flex items-center justify-between">
                    <h3 className="font-medium truncate">{conversation.recipientName}</h3>
                    <span className="text-xs text-foreground-muted">
                        {formatTime(conversation.lastMessage?.timestamp)}
                    </span>
                </div>
                <div className="flex items-center justify-between mt-0.5">
                    <p className="text-sm text-foreground-muted truncate">
                        {conversation.lastMessage?.content || 'No messages yet'}
                    </p>
                    {conversation.unreadCount > 0 && (
                        <span className="min-w-[20px] h-5 px-1.5 rounded-full bg-primary text-primary-foreground text-xs font-medium flex items-center justify-center">
                            {conversation.unreadCount}
                        </span>
                    )}
                </div>
            </div>
        </div>
    );
}

// ============================================
// MAIN DEV SHOWCASE PAGE
// ============================================

export default function DevShowcase() {
    const [isDark, setIsDark] = useState(true);

    const toggleTheme = () => {
        setIsDark(!isDark);
        document.documentElement.classList.toggle('dark', !isDark);
        document.documentElement.classList.toggle('light', isDark);
    };

    return (
        <div className="min-h-screen bg-background">
            {/* Header */}
            <header className="sticky top-0 z-50 bg-background/80 backdrop-blur-lg border-b border-border">
                <div className="max-w-5xl mx-auto px-6 py-4 flex items-center justify-between">
                    <div className="flex items-center gap-4">
                        <Link to="/" className="p-2 hover:bg-background-tertiary rounded-lg transition-colors">
                            <ArrowLeft className="h-5 w-5" />
                        </Link>
                        <div className="flex items-center gap-2">
                            <Shield className="h-6 w-6 text-primary" />
                            <span className="font-semibold">Dev Showcase</span>
                        </div>
                    </div>
                    <button
                        onClick={toggleTheme}
                        className="p-2 hover:bg-background-tertiary rounded-lg transition-colors"
                    >
                        {isDark ? <Sun className="h-5 w-5" /> : <Moon className="h-5 w-5" />}
                    </button>
                </div>
            </header>

            {/* Navigation */}
            <nav className="sticky top-[65px] z-40 bg-background/80 backdrop-blur-lg border-b border-border">
                <div className="max-w-5xl mx-auto px-6 py-3 flex gap-4 overflow-x-auto">
                    <a href="#calls" className="text-sm text-foreground-muted hover:text-foreground whitespace-nowrap">Calls</a>
                    <a href="#messages" className="text-sm text-foreground-muted hover:text-foreground whitespace-nowrap">Messages</a>
                    <a href="#conversations" className="text-sm text-foreground-muted hover:text-foreground whitespace-nowrap">Conversations</a>
                    <a href="#buttons" className="text-sm text-foreground-muted hover:text-foreground whitespace-nowrap">Buttons</a>
                    <a href="#inputs" className="text-sm text-foreground-muted hover:text-foreground whitespace-nowrap">Inputs</a>
                    <a href="#avatars" className="text-sm text-foreground-muted hover:text-foreground whitespace-nowrap">Avatars</a>
                </div>
            </nav>

            {/* Content */}
            <main className="max-w-5xl mx-auto px-6 pb-20">
                {/* Call Components */}
                <SectionHeader
                    id="calls"
                    title="Call Components"
                    description="Incoming, active, and ended call UI states"
                />

                <div className="grid md:grid-cols-2 gap-4">
                    <ComponentCard title="Incoming Call">
                        <IncomingCallPreview />
                    </ComponentCard>

                    <ComponentCard title="Audio Call (Connected)">
                        <AudioCallPreview />
                    </ComponentCard>

                    <ComponentCard title="Video Call (Connected)">
                        <VideoCallPreview />
                    </ComponentCard>

                    <ComponentCard title="Call Ended Notification">
                        <CallEndedPreview />
                    </ComponentCard>
                </div>

                {/* Message Components */}
                <SectionHeader
                    id="messages"
                    title="Message Bubbles"
                    description="Different message states and types"
                />

                <ComponentCard title="Message Thread">
                    <div className="space-y-3 max-w-md mx-auto">
                        <DateSeparator date={new Date(Date.now() - 3600000)} />
                        {mockMessages.map((msg) => (
                            <MessageBubble
                                key={msg.id}
                                message={msg}
                                isOwn={msg.senderId === 'current-user'}
                            />
                        ))}
                    </div>
                </ComponentCard>

                {/* Conversation List */}
                <SectionHeader
                    id="conversations"
                    title="Conversation List"
                    description="Conversation items with various states"
                />

                <ComponentCard title="Conversations">
                    <div className="space-y-1 max-w-md">
                        {mockConversations.map((conv) => (
                            <ConversationItemPreview key={conv.id} conversation={conv} />
                        ))}
                    </div>
                </ComponentCard>

                {/* Buttons */}
                <SectionHeader
                    id="buttons"
                    title="Buttons"
                    description="Button variants and states"
                />

                <ComponentCard title="Button Variants">
                    <div className="flex flex-wrap gap-3">
                        <Button>Primary</Button>
                        <Button variant="secondary">Secondary</Button>
                        <Button variant="outline">Outline</Button>
                        <Button variant="ghost">Ghost</Button>
                        <Button variant="destructive">Destructive</Button>
                        <Button disabled>Disabled</Button>
                    </div>
                </ComponentCard>

                <ComponentCard title="Button Sizes">
                    <div className="flex flex-wrap items-center gap-3">
                        <Button size="sm">Small</Button>
                        <Button size="default">Default</Button>
                        <Button size="lg">Large</Button>
                    </div>
                </ComponentCard>

                {/* Inputs */}
                <SectionHeader
                    id="inputs"
                    title="Inputs"
                    description="Form input components"
                />

                <ComponentCard title="Input Variants">
                    <div className="space-y-4 max-w-sm">
                        <Input placeholder="Default input" />
                        <Input placeholder="Disabled input" disabled />
                        <Input placeholder="With value" defaultValue="Hello world" />
                        <Input type="password" placeholder="Password" defaultValue="secret123" />
                    </div>
                </ComponentCard>

                {/* Avatars */}
                <SectionHeader
                    id="avatars"
                    title="Avatars"
                    description="User avatar components"
                />

                <ComponentCard title="Avatar Sizes">
                    <div className="flex items-end gap-4">
                        <Avatar className="h-8 w-8">
                            <AvatarImage src="https://api.dicebear.com/7.x/avataaars/svg?seed=Small" />
                            <AvatarFallback>S</AvatarFallback>
                        </Avatar>
                        <Avatar className="h-10 w-10">
                            <AvatarImage src="https://api.dicebear.com/7.x/avataaars/svg?seed=Medium" />
                            <AvatarFallback>M</AvatarFallback>
                        </Avatar>
                        <Avatar className="h-12 w-12">
                            <AvatarImage src="https://api.dicebear.com/7.x/avataaars/svg?seed=Large" />
                            <AvatarFallback>L</AvatarFallback>
                        </Avatar>
                        <Avatar className="h-16 w-16">
                            <AvatarImage src="https://api.dicebear.com/7.x/avataaars/svg?seed=XLarge" />
                            <AvatarFallback>XL</AvatarFallback>
                        </Avatar>
                        <Avatar className="h-24 w-24">
                            <AvatarImage src="https://api.dicebear.com/7.x/avataaars/svg?seed=XXLarge" />
                            <AvatarFallback>XXL</AvatarFallback>
                        </Avatar>
                    </div>
                </ComponentCard>

                <ComponentCard title="Avatar Fallbacks">
                    <div className="flex items-center gap-4">
                        <Avatar className="h-12 w-12">
                            <AvatarImage src="https://broken-image.png" />
                            <AvatarFallback>JD</AvatarFallback>
                        </Avatar>
                        <Avatar className="h-12 w-12">
                            <AvatarFallback className="bg-primary text-primary-foreground">SR</AvatarFallback>
                        </Avatar>
                        <Avatar className="h-12 w-12">
                            <AvatarFallback className="bg-destructive text-white">!</AvatarFallback>
                        </Avatar>
                    </div>
                </ComponentCard>
            </main>
        </div>
    );
}
