/**
 * TabBar Component
 * 
 * Premium floating tab bar with:
 * - 5 tabs: Chats, Contacts, Calls, Stories, Profile
 * - Glow effect that follows active tab
 * - Animated icons with scale/color transitions
 * - Unread badges
 * - Safe area padding for mobile devices
 */

import { useMemo, useRef, useEffect, useState } from 'react';
import { cn } from '@/lib/utils';
import {
    MessageSquare,
    Users,
    Phone,
    CircleDot,
    User,
} from 'lucide-react';

export type TabId = 'chats' | 'contacts' | 'calls' | 'stories' | 'profile';

interface TabConfig {
    id: TabId;
    label: string;
    icon: typeof MessageSquare;
}

const tabs: TabConfig[] = [
    { id: 'chats', label: 'Chats', icon: MessageSquare },
    { id: 'contacts', label: 'Contacts', icon: Users },
    { id: 'calls', label: 'Calls', icon: Phone },
    { id: 'stories', label: 'Stories', icon: CircleDot },
    { id: 'profile', label: 'Profile', icon: User },
];

interface TabBarProps {
    activeTab: TabId;
    onTabChange: (tab: TabId) => void;
    badges?: Partial<Record<TabId, number>>;
    className?: string;
}

export function TabBar({ activeTab, onTabChange, badges = {}, className }: TabBarProps) {
    const tabRefs = useRef<(HTMLButtonElement | null)[]>([]);
    const [glowPosition, setGlowPosition] = useState({ x: 0, y: 0 });

    // Calculate active tab index
    const activeIndex = useMemo(() =>
        tabs.findIndex(t => t.id === activeTab),
        [activeTab]
    );

    // Update glow position when active tab changes
    useEffect(() => {
        const activeTabEl = tabRefs.current[activeIndex];
        if (activeTabEl) {
            const rect = activeTabEl.getBoundingClientRect();
            const parent = activeTabEl.parentElement?.getBoundingClientRect();
            if (parent) {
                setGlowPosition({
                    x: rect.left - parent.left + rect.width / 2 - 28, // Center the 56px glow
                    y: rect.top - parent.top + rect.height / 2 - 28,
                });
            }
        }
    }, [activeIndex]);

    return (
        <nav className={cn('floating-tab-bar', className)}>
            {/* Glow indicator */}
            <div
                className="tab-glow"
                style={{
                    transform: `translate(${glowPosition.x}px, ${glowPosition.y}px)`,
                }}
            />

            {/* Tab items container */}
            <div className="flex items-center justify-around relative">
                {tabs.map((tab, index) => {
                    const Icon = tab.icon;
                    const isActive = activeTab === tab.id;
                    const badge = badges[tab.id];

                    return (
                        <button
                            key={tab.id}
                            ref={el => { tabRefs.current[index] = el; }}
                            onClick={() => onTabChange(tab.id)}
                            className={cn(
                                'tab-item',
                                isActive && 'tab-item-active'
                            )}
                            aria-label={tab.label}
                            aria-current={isActive ? 'page' : undefined}
                        >
                            {/* Badge */}
                            {badge && badge > 0 && (
                                <span className="tab-badge">
                                    {badge > 99 ? '99+' : badge}
                                </span>
                            )}

                            {/* Icon */}
                            <Icon
                                className={cn(
                                    'tab-icon',
                                    isActive && 'tab-icon-active'
                                )}
                                strokeWidth={isActive ? 2.5 : 2}
                            />

                            {/* Label */}
                            <span className="tab-label">{tab.label}</span>
                        </button>
                    );
                })}
            </div>
        </nav>
    );
}

export default TabBar;
