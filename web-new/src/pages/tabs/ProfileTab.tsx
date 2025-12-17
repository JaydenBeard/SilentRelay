/**
 * Profile Tab
 * 
 * Full profile page with:
 * - Large avatar with gradient accent ring
 * - Display name and username
 * - Bio/status message
 * - QR code for easy contact sharing
 * - Settings access
 * - Account actions
 */

import { useState, useCallback, useRef } from 'react';
import { cn } from '@/lib/utils';
import { useAuthStore } from '@/core/store/authStore';
import { useAuth } from '@/hooks/useAuth';

// UI Components
import { GlassCard } from '@/components/ui/GlassCard';
import { AvatarRing } from '@/components/ui/AvatarRing';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Settings } from '@/components/settings';
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from '@/components/ui/dialog';
import {
    Camera,
    Edit2,
    QrCode,
    Shield,
    Bell,
    Palette,
    HelpCircle,
    LogOut,
    ChevronRight,
    Copy,
    Check,
    Link,
} from 'lucide-react';

export function ProfileTab() {
    const { user, updateUser } = useAuthStore();
    const { logout } = useAuth();

    // State
    const [isEditingName, setIsEditingName] = useState(false);
    const [newDisplayName, setNewDisplayName] = useState(user?.displayName || '');
    const [isEditingBio, setIsEditingBio] = useState(false);
    const [newBio, setNewBio] = useState(user?.bio || '');
    const [isSettingsOpen, setIsSettingsOpen] = useState(false);
    const [isQrCodeOpen, setIsQrCodeOpen] = useState(false);
    const [copiedUsername, setCopiedUsername] = useState(false);
    const [isAvatarUploading, setIsAvatarUploading] = useState(false);
    const fileInputRef = useRef<HTMLInputElement>(null);

    // Handle display name update
    const handleUpdateDisplayName = useCallback(async () => {
        if (newDisplayName.trim() && newDisplayName !== user?.displayName) {
            // TODO: API call to update display name
            updateUser({ displayName: newDisplayName.trim() });
        }
        setIsEditingName(false);
    }, [newDisplayName, user, updateUser]);

    // Handle bio update
    const handleUpdateBio = useCallback(async () => {
        if (newBio !== user?.bio) {
            // TODO: API call to update bio
            updateUser({ bio: newBio });
        }
        setIsEditingBio(false);
    }, [newBio, user, updateUser]);

    // Handle avatar upload
    const handleAvatarUpload = useCallback(async (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (!file) return;

        setIsAvatarUploading(true);
        try {
            // For demo, create a local URL - would upload to server
            const url = URL.createObjectURL(file);
            updateUser({ avatarUrl: url });

            // TODO: Actual upload to server
            // const formData = new FormData();
            // formData.append('avatar', file);
            // const response = await fetch('/api/v1/users/me/avatar', {
            //   method: 'POST',
            //   body: formData,
            //   headers: { Authorization: `Bearer ${token}` }
            // });
        } catch (error) {
            console.error('Failed to upload avatar:', error);
        } finally {
            setIsAvatarUploading(false);
        }
    }, [updateUser]);

    // Handle copy username
    const handleCopyUsername = useCallback(() => {
        if (user?.username) {
            navigator.clipboard.writeText(`@${user.username}`);
            setCopiedUsername(true);
            setTimeout(() => setCopiedUsername(false), 2000);
        }
    }, [user]);

    // Menu items
    const menuItems = [
        {
            icon: Shield,
            label: 'Privacy & Security',
            onClick: () => setIsSettingsOpen(true),
            color: 'text-primary',
        },
        {
            icon: Bell,
            label: 'Notifications',
            onClick: () => setIsSettingsOpen(true),
            color: 'text-warning',
        },
        {
            icon: Palette,
            label: 'Appearance',
            onClick: () => setIsSettingsOpen(true),
            color: 'text-accent',
        },
        {
            icon: Link,
            label: 'Linked Devices',
            onClick: () => setIsSettingsOpen(true),
            color: 'text-cyan-500',
        },
        {
            icon: HelpCircle,
            label: 'Help & Support',
            onClick: () => window.open('mailto:support@silentrelay.com.au'),
            color: 'text-foreground-muted',
        },
    ];

    return (
        <div className="h-full flex flex-col has-tab-bar overflow-y-auto">
            {/* Profile Header */}
            <div className="flex flex-col items-center px-6 pt-8 pb-6">
                {/* Avatar with edit button */}
                <div className="relative mb-4">
                    <AvatarRing
                        src={user?.avatarUrl}
                        alt={user?.displayName || 'Profile'}
                        fallback={user?.displayName?.charAt(0) || user?.username?.charAt(0) || '?'}
                        size="xl"
                        className="w-24 h-24"
                    />
                    <button
                        onClick={() => fileInputRef.current?.click()}
                        disabled={isAvatarUploading}
                        className={cn(
                            'absolute -bottom-1 -right-1 w-8 h-8 rounded-full',
                            'bg-primary text-primary-foreground',
                            'flex items-center justify-center',
                            'border-2 border-background shadow-lg',
                            'transition-transform hover:scale-110',
                            isAvatarUploading && 'opacity-50 cursor-not-allowed'
                        )}
                    >
                        <Camera className="h-4 w-4" />
                    </button>
                    <input
                        ref={fileInputRef}
                        type="file"
                        accept="image/*"
                        className="hidden"
                        onChange={handleAvatarUpload}
                    />
                </div>

                {/* Display name */}
                {isEditingName ? (
                    <div className="flex items-center gap-2 mb-1">
                        <Input
                            value={newDisplayName}
                            onChange={(e) => setNewDisplayName(e.target.value)}
                            className="text-center text-xl font-semibold w-48"
                            autoFocus
                            onBlur={handleUpdateDisplayName}
                            onKeyDown={(e) => e.key === 'Enter' && handleUpdateDisplayName()}
                        />
                    </div>
                ) : (
                    <button
                        onClick={() => {
                            setNewDisplayName(user?.displayName || '');
                            setIsEditingName(true);
                        }}
                        className="flex items-center gap-2 mb-1 group"
                    >
                        <h1 className="text-xl font-semibold">{user?.displayName || 'Set display name'}</h1>
                        <Edit2 className="h-4 w-4 text-foreground-muted opacity-0 group-hover:opacity-100 transition-opacity" />
                    </button>
                )}

                {/* Username */}
                <button
                    onClick={handleCopyUsername}
                    className="flex items-center gap-2 text-foreground-muted hover:text-foreground transition-colors"
                >
                    <span className="font-mono">@{user?.username || 'username'}</span>
                    {copiedUsername ? (
                        <Check className="h-4 w-4 text-success" />
                    ) : (
                        <Copy className="h-4 w-4" />
                    )}
                </button>

                {/* Bio */}
                <div className="mt-4 max-w-xs text-center">
                    {isEditingBio ? (
                        <div className="space-y-2">
                            <Input
                                value={newBio}
                                onChange={(e) => setNewBio(e.target.value)}
                                placeholder="Write a bio..."
                                className="text-center"
                                autoFocus
                                maxLength={150}
                            />
                            <div className="flex justify-center gap-2">
                                <Button size="sm" variant="ghost" onClick={() => setIsEditingBio(false)}>
                                    Cancel
                                </Button>
                                <Button size="sm" onClick={handleUpdateBio}>
                                    Save
                                </Button>
                            </div>
                        </div>
                    ) : (
                        <button
                            onClick={() => {
                                setNewBio(user?.bio || '');
                                setIsEditingBio(true);
                            }}
                            className="text-foreground-secondary hover:text-foreground transition-colors"
                        >
                            {user?.bio || 'Tap to add a bio...'}
                        </button>
                    )}
                </div>

                {/* QR Code Button */}
                <Button
                    variant="outline"
                    onClick={() => setIsQrCodeOpen(true)}
                    className="mt-4 rounded-xl gap-2"
                >
                    <QrCode className="h-4 w-4" />
                    Share QR Code
                </Button>
            </div>

            {/* Menu Section */}
            <div className="px-4 pb-4 space-y-2">
                <GlassCard variant="subtle" className="divide-y divide-border overflow-hidden">
                    {menuItems.map((item) => (
                        <button
                            key={item.label}
                            onClick={item.onClick}
                            className="w-full px-4 py-3 flex items-center gap-3 hover:bg-background-tertiary/50 transition-colors"
                        >
                            <item.icon className={cn('h-5 w-5', item.color)} />
                            <span className="flex-1 text-left">{item.label}</span>
                            <ChevronRight className="h-4 w-4 text-foreground-muted" />
                        </button>
                    ))}
                </GlassCard>

                {/* Logout Button */}
                <GlassCard variant="subtle" className="overflow-hidden">
                    <button
                        onClick={logout}
                        className="w-full px-4 py-3 flex items-center gap-3 text-destructive hover:bg-destructive/10 transition-colors"
                    >
                        <LogOut className="h-5 w-5" />
                        <span className="flex-1 text-left">Log Out</span>
                    </button>
                </GlassCard>

                {/* Version info */}
                <p className="text-center text-xs text-foreground-muted pt-4">
                    SilentRelay v1.0.0 • End-to-end encrypted
                </p>
            </div>

            {/* Settings Modal */}
            <Settings open={isSettingsOpen} onOpenChange={setIsSettingsOpen} />

            {/* QR Code Modal */}
            <Dialog open={isQrCodeOpen} onOpenChange={setIsQrCodeOpen}>
                <DialogContent className="max-w-sm">
                    <DialogHeader>
                        <DialogTitle>Your QR Code</DialogTitle>
                        <DialogDescription>
                            Others can scan this code to add you as a contact
                        </DialogDescription>
                    </DialogHeader>

                    <div className="flex flex-col items-center py-4">
                        {/* QR Code placeholder - would use a QR library */}
                        <div className="w-48 h-48 bg-white rounded-xl flex items-center justify-center mb-4">
                            <div className="w-40 h-40 border-2 border-black rounded grid grid-cols-5 grid-rows-5 gap-0.5 p-2">
                                {/* Simple QR pattern placeholder */}
                                {Array.from({ length: 25 }).map((_, i) => (
                                    <div
                                        key={i}
                                        className={cn(
                                            'bg-black rounded-sm',
                                            Math.random() > 0.5 && 'bg-transparent'
                                        )}
                                    />
                                ))}
                            </div>
                        </div>

                        <p className="font-mono text-lg mb-2">@{user?.username}</p>

                        <Button
                            variant="outline"
                            onClick={handleCopyUsername}
                            className="rounded-xl gap-2"
                        >
                            {copiedUsername ? (
                                <>
                                    <Check className="h-4 w-4" />
                                    Copied!
                                </>
                            ) : (
                                <>
                                    <Copy className="h-4 w-4" />
                                    Copy Username
                                </>
                            )}
                        </Button>
                    </div>
                </DialogContent>
            </Dialog>
        </div>
    );
}

export default ProfileTab;
