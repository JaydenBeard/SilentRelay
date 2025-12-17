/**
 * Stories Tab
 * 
 * Signal-style disappearing stories with:
 * - Your story creation
 * - Story rings with gradient indicators
 * - Story viewer with tap-to-progress
 * - 24-hour auto-expiration
 */

import { useState, useMemo, useCallback, useRef, useEffect } from 'react';
import { cn } from '@/lib/utils';
import { useAuthStore } from '@/core/store/authStore';

// UI Components
import { GlassCard } from '@/components/ui/GlassCard';
import { AvatarRing } from '@/components/ui/AvatarRing';
import { EmptyState } from '@/components/ui/EmptyState';
import { Button } from '@/components/ui/button';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import {
    CircleDot,
    Plus,
    Camera,
    Type,
    X,
} from 'lucide-react';

// Story types
interface Story {
    id: string;
    userId: string;
    type: 'image' | 'video' | 'text';
    content: string; // URL for image/video, text content for text
    backgroundColor?: string; // For text stories
    caption?: string;
    createdAt: string;
    expiresAt: string;
    viewerIds: string[];
}

interface UserStories {
    userId: string;
    displayName: string;
    username: string;
    avatarUrl?: string;
    stories: Story[];
    lastUpdated: string;
    hasNewStories: boolean;
}

// Mock data - will be replaced with real API
const mockUserStories: UserStories[] = [];

// Local storage key for persisting stories
const STORIES_STORAGE_KEY = 'silentrelay_my_stories';

// Load stories from localStorage
function loadStoriesFromStorage(): Story[] {
    try {
        const stored = localStorage.getItem(STORIES_STORAGE_KEY);
        if (!stored) return [];
        const stories: Story[] = JSON.parse(stored);
        // Filter out expired stories
        const now = new Date();
        return stories.filter(s => new Date(s.expiresAt) > now);
    } catch {
        return [];
    }
}

// Save stories to localStorage
function saveStoriesToStorage(stories: Story[]) {
    try {
        localStorage.setItem(STORIES_STORAGE_KEY, JSON.stringify(stories));
    } catch (e) {
        console.error('Failed to save stories:', e);
    }
}

export function StoriesTab() {
    const { user } = useAuthStore();

    // State - load from localStorage on init
    const [myStories, setMyStories] = useState<Story[]>(() => loadStoriesFromStorage());
    const [otherStories] = useState<UserStories[]>(mockUserStories);
    const [isCreateOpen, setIsCreateOpen] = useState(false);
    const [viewingStory, setViewingStory] = useState<{
        userStories: UserStories;
        currentIndex: number;
    } | null>(null);

    // Persist stories to localStorage when they change
    useEffect(() => {
        saveStoriesToStorage(myStories);
    }, [myStories]);

    // Derive stories from contacts who have stories
    const storiesFromContacts = useMemo(() => {
        // For now, use mock data - would integrate with stories API
        return otherStories.sort((a, b) => {
            // New stories first, then by last updated
            if (a.hasNewStories !== b.hasNewStories) return a.hasNewStories ? -1 : 1;
            return new Date(b.lastUpdated).getTime() - new Date(a.lastUpdated).getTime();
        });
    }, [otherStories]);

    // Format time remaining
    const formatExpiresIn = useCallback((expiresAt: string) => {
        const expires = new Date(expiresAt);
        const now = new Date();
        const hoursLeft = Math.max(0, Math.ceil((expires.getTime() - now.getTime()) / 3600000));
        if (hoursLeft <= 1) return 'Expires soon';
        return `${hoursLeft}h left`;
    }, []);

    // Handle opening story viewer
    const handleViewStory = useCallback((userStories: UserStories) => {
        setViewingStory({ userStories, currentIndex: 0 });
    }, []);

    // Handle closing story viewer
    const handleCloseViewer = useCallback(() => {
        setViewingStory(null);
    }, []);

    // Handle story navigation
    const handleNextStory = useCallback(() => {
        if (!viewingStory) return;

        const { userStories, currentIndex } = viewingStory;
        if (currentIndex < userStories.stories.length - 1) {
            setViewingStory({ ...viewingStory, currentIndex: currentIndex + 1 });
        } else {
            // Move to next user's stories or close
            const userIndex = storiesFromContacts.findIndex(u => u.userId === userStories.userId);
            if (userIndex < storiesFromContacts.length - 1) {
                setViewingStory({
                    userStories: storiesFromContacts[userIndex + 1],
                    currentIndex: 0,
                });
            } else {
                handleCloseViewer();
            }
        }
    }, [viewingStory, storiesFromContacts, handleCloseViewer]);

    const handlePrevStory = useCallback(() => {
        if (!viewingStory) return;

        const { userStories, currentIndex } = viewingStory;
        if (currentIndex > 0) {
            setViewingStory({ ...viewingStory, currentIndex: currentIndex - 1 });
        } else {
            // Move to previous user's last story
            const userIndex = storiesFromContacts.findIndex(u => u.userId === userStories.userId);
            if (userIndex > 0) {
                const prevUser = storiesFromContacts[userIndex - 1];
                setViewingStory({
                    userStories: prevUser,
                    currentIndex: prevUser.stories.length - 1,
                });
            }
        }
    }, [viewingStory, storiesFromContacts]);

    return (
        <div className="h-full flex flex-col has-tab-bar">
            {/* Header */}
            <header className="flex-shrink-0 px-4 pt-4 pb-2">
                <h1 className="text-2xl font-bold mb-4">Stories</h1>
            </header>

            {/* Content */}
            <div className="flex-1 overflow-y-auto px-4 pb-4">
                {/* My Story Section */}
                <div className="mb-6">
                    <h2 className="text-sm font-medium text-foreground-muted mb-3">Your Story</h2>
                    <GlassCard
                        variant="subtle"
                        press
                        className="p-3 flex items-center gap-3"
                        onClick={() => myStories.length > 0 ? handleViewStory({
                            userId: user?.id || '',
                            displayName: user?.displayName || 'You',
                            username: user?.username || '',
                            avatarUrl: user?.avatarUrl,
                            stories: myStories,
                            lastUpdated: myStories[0]?.createdAt || '',
                            hasNewStories: false,
                        }) : setIsCreateOpen(true)}
                    >
                        {/* Add story button or view existing */}
                        <div className="relative">
                            {myStories.length > 0 ? (
                                <AvatarRing
                                    src={user?.avatarUrl}
                                    alt={user?.displayName || 'You'}
                                    fallback={user?.displayName?.charAt(0) || 'Y'}
                                    size="lg"
                                    hasStory
                                    storyViewed
                                />
                            ) : (
                                <div className="w-14 h-14 rounded-full bg-gradient-to-br from-primary/20 to-primary/5 flex items-center justify-center border-2 border-dashed border-primary/30">
                                    <Plus className="h-6 w-6 text-primary" />
                                </div>
                            )}
                        </div>

                        <div className="flex-1">
                            <p className="font-medium">
                                {myStories.length > 0 ? 'Your Story' : 'Add to your story'}
                            </p>
                            <p className="text-sm text-foreground-muted">
                                {myStories.length > 0
                                    ? `${myStories.length} ${myStories.length === 1 ? 'story' : 'stories'} • ${formatExpiresIn(myStories[0].expiresAt)}`
                                    : 'Share a moment with your contacts'
                                }
                            </p>
                        </div>

                        {myStories.length > 0 && (
                            <Button
                                size="icon"
                                variant="ghost"
                                onClick={(e) => {
                                    e.stopPropagation();
                                    setIsCreateOpen(true);
                                }}
                                className="rounded-full"
                            >
                                <Plus className="h-5 w-5" />
                            </Button>
                        )}
                    </GlassCard>
                </div>

                {/* Recent Stories Section */}
                <div>
                    <h2 className="text-sm font-medium text-foreground-muted mb-3">Recent Updates</h2>

                    {storiesFromContacts.length === 0 ? (
                        <EmptyState
                            icon={CircleDot}
                            title="No stories yet"
                            description="When your contacts share stories, they'll appear here for 24 hours."
                            animate={false}
                        />
                    ) : (
                        <div className="space-y-2">
                            {storiesFromContacts.map((userStory) => (
                                <GlassCard
                                    key={userStory.userId}
                                    variant="subtle"
                                    press
                                    className="p-3 flex items-center gap-3"
                                    onClick={() => handleViewStory(userStory)}
                                >
                                    <AvatarRing
                                        src={userStory.avatarUrl}
                                        alt={userStory.displayName}
                                        fallback={userStory.displayName.charAt(0)}
                                        size="lg"
                                        hasStory
                                        storyViewed={!userStory.hasNewStories}
                                    />

                                    <div className="flex-1">
                                        <p className="font-medium">{userStory.displayName}</p>
                                        <p className="text-sm text-foreground-muted">
                                            {userStory.stories.length} {userStory.stories.length === 1 ? 'story' : 'stories'} • {formatTimeAgo(userStory.lastUpdated)}
                                        </p>
                                    </div>
                                </GlassCard>
                            ))}
                        </div>
                    )}
                </div>
            </div>

            {/* Create Story Modal */}
            {isCreateOpen && (
                <CreateStoryModal
                    onClose={() => setIsCreateOpen(false)}
                    onCreateStory={(story) => {
                        setMyStories(prev => [story, ...prev]);
                        setIsCreateOpen(false);
                    }}
                />
            )}

            {/* Story Viewer */}
            {viewingStory && (
                <StoryViewer
                    userStories={viewingStory.userStories}
                    currentIndex={viewingStory.currentIndex}
                    onNext={handleNextStory}
                    onPrev={handlePrevStory}
                    onClose={handleCloseViewer}
                />
            )}
        </div>
    );
}

/**
 * Create Story Modal
 */
function CreateStoryModal({
    onClose,
    onCreateStory,
}: {
    onClose: () => void;
    onCreateStory: (story: Story) => void;
}) {
    const { user } = useAuthStore();
    const [storyType, setStoryType] = useState<'photo' | 'text' | null>(null);
    const [textContent, setTextContent] = useState('');
    const [bgColor, setBgColor] = useState('#8B5CF6');
    const fileInputRef = useRef<HTMLInputElement>(null);

    const backgroundColors = [
        '#8B5CF6', // Purple
        '#EC4899', // Pink
        '#EF4444', // Red
        '#F97316', // Orange
        '#22C55E', // Green
        '#06B6D4', // Cyan
        '#3B82F6', // Blue
        '#1F2937', // Dark
    ];

    const handleImageSelect = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (!file) return;

        // For demo, create a local URL - would upload to server
        const url = URL.createObjectURL(file);

        const story: Story = {
            id: `story-${Date.now()}`,
            userId: user?.id || '',
            type: 'image',
            content: url,
            createdAt: new Date().toISOString(),
            expiresAt: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
            viewerIds: [],
        };

        onCreateStory(story);
    }, [user, onCreateStory]);

    const handleCreateTextStory = useCallback(() => {
        if (!textContent.trim()) return;

        const story: Story = {
            id: `story-${Date.now()}`,
            userId: user?.id || '',
            type: 'text',
            content: textContent.trim(),
            backgroundColor: bgColor,
            createdAt: new Date().toISOString(),
            expiresAt: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
            viewerIds: [],
        };

        onCreateStory(story);
    }, [textContent, bgColor, user, onCreateStory]);

    return (
        <div className="fixed inset-0 z-50 bg-background flex flex-col">
            {/* Header */}
            <header className="flex items-center justify-between p-4 border-b border-border">
                <Button size="icon" variant="ghost" onClick={onClose}>
                    <X className="h-5 w-5" />
                </Button>
                <h2 className="text-lg font-semibold">Create Story</h2>
                <div className="w-10" /> {/* Spacer */}
            </header>

            {/* Content */}
            {storyType === null ? (
                <div className="flex-1 flex items-center justify-center p-8">
                    <div className="grid grid-cols-2 gap-4 max-w-sm w-full">
                        <GlassCard
                            variant="default"
                            press
                            className="p-8 flex flex-col items-center gap-4 cursor-pointer"
                            onClick={() => fileInputRef.current?.click()}
                        >
                            <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center">
                                <Camera className="h-8 w-8 text-primary" />
                            </div>
                            <span className="font-medium">Photo</span>
                            <input
                                ref={fileInputRef}
                                type="file"
                                accept="image/*"
                                className="hidden"
                                onChange={handleImageSelect}
                            />
                        </GlassCard>

                        <GlassCard
                            variant="default"
                            press
                            className="p-8 flex flex-col items-center gap-4 cursor-pointer"
                            onClick={() => setStoryType('text')}
                        >
                            <div className="w-16 h-16 rounded-full bg-accent/10 flex items-center justify-center">
                                <Type className="h-8 w-8 text-accent" />
                            </div>
                            <span className="font-medium">Text</span>
                        </GlassCard>
                    </div>
                </div>
            ) : (
                // Text story creation
                <div className="flex-1 flex flex-col">
                    {/* Preview */}
                    <div
                        className="flex-1 flex items-center justify-center p-8"
                        style={{ backgroundColor: bgColor }}
                    >
                        <textarea
                            value={textContent}
                            onChange={(e) => setTextContent(e.target.value)}
                            placeholder="Type your story..."
                            className="w-full max-w-md text-2xl font-semibold text-white text-center bg-transparent border-none outline-none resize-none placeholder:text-white/50"
                            rows={4}
                            autoFocus
                        />
                    </div>

                    {/* Color picker */}
                    <div className="p-4 pb-safe border-t border-border bg-background">
                        <div className="flex justify-center gap-3 mb-4">
                            {backgroundColors.map((color) => (
                                <button
                                    key={color}
                                    onClick={() => setBgColor(color)}
                                    className={cn(
                                        'w-8 h-8 rounded-full transition-transform',
                                        bgColor === color && 'scale-125 ring-2 ring-white ring-offset-2 ring-offset-background'
                                    )}
                                    style={{ backgroundColor: color }}
                                />
                            ))}
                        </div>

                        <Button
                            onClick={handleCreateTextStory}
                            disabled={!textContent.trim()}
                            className="w-full rounded-xl mb-4"
                            size="lg"
                        >
                            Share Story
                        </Button>
                    </div>
                </div>
            )}
        </div>
    );
}

/**
 * Story Viewer Component
 */
function StoryViewer({
    userStories,
    currentIndex,
    onNext,
    onPrev,
    onClose,
}: {
    userStories: UserStories;
    currentIndex: number;
    onNext: () => void;
    onPrev: () => void;
    onClose: () => void;
}) {
    const story = userStories.stories[currentIndex];
    const [progress, setProgress] = useState(0);

    // Auto-advance timer
    useEffect(() => {
        setProgress(0);
        const duration = story.type === 'video' ? 15000 : 5000;
        const interval = 50;
        let elapsed = 0;

        const timer = setInterval(() => {
            elapsed += interval;
            setProgress((elapsed / duration) * 100);

            if (elapsed >= duration) {
                onNext();
            }
        }, interval);

        return () => clearInterval(timer);
    }, [story, onNext]);

    // Touch handling
    const handleTouchStart = useCallback((e: React.TouchEvent | React.MouseEvent) => {
        const x = 'touches' in e ? e.touches[0].clientX : e.clientX;
        const width = window.innerWidth;

        if (x < width / 3) {
            onPrev();
        } else if (x > (width * 2) / 3) {
            onNext();
        }
    }, [onNext, onPrev]);

    return (
        <div className="fixed inset-0 z-[100] bg-black flex flex-col">
            {/* Progress bars */}
            <div className="absolute top-0 left-0 right-0 flex gap-1 p-2 z-10">
                {userStories.stories.map((_, idx) => (
                    <div
                        key={idx}
                        className="flex-1 h-0.5 bg-white/30 rounded-full overflow-hidden"
                    >
                        <div
                            className="h-full bg-white transition-all duration-100"
                            style={{
                                width: idx < currentIndex ? '100%' : idx === currentIndex ? `${progress}%` : '0%',
                            }}
                        />
                    </div>
                ))}
            </div>

            {/* Header */}
            <div className="absolute top-0 left-0 right-0 p-4 pt-8 flex items-center gap-3 z-10 bg-gradient-to-b from-black/50 to-transparent">
                <Avatar className="h-10 w-10 border-2 border-white">
                    <AvatarImage src={userStories.avatarUrl} />
                    <AvatarFallback>{userStories.displayName.charAt(0)}</AvatarFallback>
                </Avatar>
                <div className="flex-1">
                    <p className="font-medium text-white">{userStories.displayName}</p>
                    <p className="text-sm text-white/70">{formatTimeAgo(story.createdAt)}</p>
                </div>
                <Button
                    size="icon"
                    variant="ghost"
                    onClick={onClose}
                    className="text-white hover:bg-white/20"
                >
                    <X className="h-5 w-5" />
                </Button>
            </div>

            {/* Story content */}
            <div
                className="flex-1 flex items-center justify-center"
                onClick={handleTouchStart}
                style={story.type === 'text' ? { backgroundColor: story.backgroundColor } : undefined}
            >
                {story.type === 'image' && (
                    <img
                        src={story.content}
                        alt="Story"
                        className="max-w-full max-h-full object-contain"
                    />
                )}
                {story.type === 'text' && (
                    <p className="text-2xl font-semibold text-white text-center px-8 max-w-lg">
                        {story.content}
                    </p>
                )}
            </div>

            {/* Caption if present */}
            {story.caption && (
                <div className="absolute bottom-0 left-0 right-0 p-4 bg-gradient-to-t from-black/50 to-transparent">
                    <p className="text-white text-center">{story.caption}</p>
                </div>
            )}
        </div>
    );
}

/**
 * Format time ago
 */
function formatTimeAgo(timestamp: string): string {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMins / 60);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;

    return date.toLocaleDateString();
}

export default StoriesTab;
