/**
 * MainApp Container
 * 
 * Root container for authenticated app experience with 5-tab navigation:
 * - Chats: Messaging interface
 * - Contacts: Friends list and management
 * - Calls: Call history
 * - Stories: Signal-style disappearing stories
 * - Profile: User profile and settings
 */

import { useState, useMemo, useCallback } from 'react';
import { TabBar, type TabId } from '@/components/navigation';
import { useChatStore } from '@/core/store/chatStore';

// Tab components
import { ChatsTab } from '@/pages/tabs/ChatsTab';
import { ContactsTab } from '@/pages/tabs/ContactsTab';
import { CallsTab } from '@/pages/tabs/CallsTab';
import { StoriesTab } from '@/pages/tabs/StoriesTab';
import { ProfileTab } from '@/pages/tabs/ProfileTab';

export default function MainApp() {
    const [activeTab, setActiveTab] = useState<TabId>('chats');
    const [hideTabBar, setHideTabBar] = useState(false);

    // Get unread counts for badges
    const { conversations } = useChatStore();

    const badges = useMemo(() => {
        const unreadChats = Object.values(conversations).reduce(
            (sum, conv) => sum + (conv.unreadCount || 0),
            0
        );

        // Count pending message requests
        const pendingRequests = Object.values(conversations).filter(
            c => c.status === 'pending'
        ).length;

        return {
            chats: unreadChats + pendingRequests,
            // contacts, calls, stories badges can be added later
        };
    }, [conversations]);

    // Handle tab changes
    const handleTabChange = useCallback((tab: TabId) => {
        setActiveTab(tab);
        setHideTabBar(false); // Show tab bar when switching tabs
    }, []);

    // Hide tab bar when entering a chat conversation (mobile UX)
    const handleEnterChat = useCallback(() => {
        setHideTabBar(true);
    }, []);

    const handleExitChat = useCallback(() => {
        setHideTabBar(false);
    }, []);

    return (
        <div className="h-screen-dynamic flex flex-col bg-background overflow-hidden">
            {/* Tab content */}
            <main className="flex-1 overflow-hidden">
                {activeTab === 'chats' && (
                    <ChatsTab
                        onEnterChat={handleEnterChat}
                        onExitChat={handleExitChat}
                    />
                )}
                {activeTab === 'contacts' && <ContactsTab />}
                {activeTab === 'calls' && <CallsTab />}
                {activeTab === 'stories' && <StoriesTab />}
                {activeTab === 'profile' && <ProfileTab />}
            </main>

            {/* Floating Tab Bar */}
            {!hideTabBar && (
                <TabBar
                    activeTab={activeTab}
                    onTabChange={handleTabChange}
                    badges={badges}
                />
            )}
        </div>
    );
}
