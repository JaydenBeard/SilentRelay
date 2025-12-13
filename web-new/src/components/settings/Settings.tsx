/**
 * Settings Modal Component
 */

import { useState, useEffect } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { ScrollArea } from '@/components/ui/scroll-area';
import { useSettingsStore } from '@/core/store/settingsStore';
import { useAuthStore } from '@/core/store/authStore';
import { useChatStore } from '@/core/store/chatStore';
import { cn } from '@/lib/utils';
import { signalProtocol } from '@/core/crypto/signal';
import type { User, PrivacySettings, NotificationSettings, AppSettings } from '@/core/types';
import {
  Shield,
  Bell,
  Palette,
  ChevronRight,
  Eye,
  EyeOff,
  MessageSquare,
  Volume2,
  VolumeX,
  Sun,
  Moon,
  Smartphone,
  Key,
  Trash2,
  Camera,
  AlertTriangle,
  Copy,
  Check,
  Ban,
  UserX,
  Loader2,
} from 'lucide-react';

interface SettingsProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

type SettingsSection = 'main' | 'profile' | 'privacy' | 'notifications' | 'appearance' | 'security' | 'blocked';

export function Settings({ open, onOpenChange }: SettingsProps) {
  const [section, setSection] = useState<SettingsSection>('main');
  const { user } = useAuthStore();
  const { privacy, notifications, app, updatePrivacy, updateNotifications, updateApp } = useSettingsStore();

  const handleBack = () => setSection('main');

  const renderContent = () => {
    switch (section) {
      case 'profile':
        return <ProfileSection onBack={handleBack} />;
      case 'privacy':
        return (
          <PrivacySection
            privacy={privacy}
            onUpdate={updatePrivacy}
            onBack={handleBack}
            onNavigateToBlocked={() => setSection('blocked')}
          />
        );
      case 'notifications':
        return (
          <NotificationsSection
            notifications={notifications}
            onUpdate={updateNotifications}
            onBack={handleBack}
          />
        );
      case 'appearance':
        return (
          <AppearanceSection
            app={app}
            onUpdate={updateApp}
            onBack={handleBack}
          />
        );
      case 'security':
        return <SecuritySection onBack={handleBack} />;
      case 'blocked':
        return <BlockedUsersSection onBack={handleBack} />;
      default:
        return (
          <MainSection
            user={user}
            onNavigate={setSection}
          />
        );
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md p-0 gap-0 overflow-hidden" aria-describedby="settings-dialog-description">
        <DialogDescription id="settings-dialog-description" className="sr-only">
          Manage your account settings, privacy preferences, and security options
        </DialogDescription>
        {renderContent()}
      </DialogContent>
    </Dialog>
  );
}

// Main Settings Menu
function MainSection({
  user,
  onNavigate,
}: {
  user: User | null;
  onNavigate: (section: SettingsSection) => void;
}) {
  return (
    <>
      <DialogHeader className="p-6 pb-4">
        <DialogTitle>Settings</DialogTitle>
      </DialogHeader>
      <ScrollArea className="max-h-[60vh]">
        <div className="p-4 pt-0 space-y-2">
          {/* Profile Preview */}
          <button
            onClick={() => onNavigate('profile')}
            className="w-full flex items-center gap-4 p-4 rounded-xl hover:bg-background-tertiary transition-colors"
          >
            <Avatar className="h-14 w-14">
              <AvatarImage src={user?.avatar} />
              <AvatarFallback className="text-lg">
                {user?.displayName?.charAt(0) || user?.username?.charAt(0)?.toUpperCase() || '?'}
              </AvatarFallback>
            </Avatar>
            <div className="flex-1 text-left">
              <p className="font-semibold">{user?.displayName || user?.username || 'Set your name'}</p>
              <p className="text-sm text-foreground-muted font-mono">{user?.username ? `@${user.username}` : user?.phoneNumber}</p>
            </div>
            <ChevronRight className="h-5 w-5 text-foreground-muted" />
          </button>

          <div className="border-t border-border my-4" />

          {/* Menu Items */}
          <MenuItem
            icon={<Shield className="h-5 w-5" />}
            title="Privacy"
            description="Read receipts, online status"
            onClick={() => onNavigate('privacy')}
          />
          <MenuItem
            icon={<Bell className="h-5 w-5" />}
            title="Notifications"
            description="Sound, alerts, preview"
            onClick={() => onNavigate('notifications')}
          />
          <MenuItem
            icon={<Palette className="h-5 w-5" />}
            title="Appearance"
            description="Theme, font size"
            onClick={() => onNavigate('appearance')}
          />
          <MenuItem
            icon={<Key className="h-5 w-5" />}
            title="Security"
            description="Identity key, sessions"
            onClick={() => onNavigate('security')}
          />
        </div>
      </ScrollArea>
    </>
  );
}

// Profile Section
function ProfileSection({ onBack }: { onBack: () => void }) {
  const { user, updateUser } = useAuthStore();
  const [displayName, setDisplayName] = useState(user?.displayName || '');
  const [isSaving, setIsSaving] = useState(false);

  const handleSave = async () => {
    if (!displayName.trim()) return;
    setIsSaving(true);
    try {
      const token = useAuthStore.getState().token;
      const response = await fetch('/api/v1/users/me', {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ display_name: displayName }),
      });
      if (response.ok) {
        updateUser({ displayName });
      } else {
        const error = await response.text();
        console.error('Failed to update profile:', error);
        alert('Failed to update profile. Please try again.');
      }
    } catch (error) {
      console.error('Failed to update profile:', error);
      alert('Failed to update profile. Please try again.');
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <>
      <SectionHeader title="Profile" onBack={onBack} />
      <div className="p-6 space-y-6">
        {/* Avatar */}
        <div className="flex flex-col items-center">
          <div className="relative">
            <Avatar className="h-24 w-24">
              <AvatarImage src={user?.avatar} />
              <AvatarFallback className="text-2xl">
                {user?.displayName?.charAt(0) || user?.username?.charAt(0)?.toUpperCase() || '?'}
              </AvatarFallback>
            </Avatar>
            <button className="absolute bottom-0 right-0 p-2 rounded-full bg-primary text-primary-foreground hover:bg-primary/90 transition-colors">
              <Camera className="h-4 w-4" />
            </button>
          </div>
        </div>

        {/* Display Name */}
        <div className="space-y-2">
          <label className="text-sm font-medium">Display Name</label>
          <Input
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            placeholder="Enter your name"
          />
        </div>

        {/* Username (read-only) */}
        <div className="space-y-2">
          <label className="text-sm font-medium">Username</label>
          <div className="relative">
            <span className="absolute left-3 top-1/2 -translate-y-1/2 text-foreground-muted font-mono">
              @
            </span>
            <Input
              value={user?.username || ''}
              disabled
              className="pl-8 font-mono bg-background-tertiary"
            />
          </div>
          <p className="text-xs text-foreground-muted">Your unique identifier for others to find you</p>
        </div>

        {/* Phone Number (read-only) */}
        <div className="space-y-2">
          <label className="text-sm font-medium">Phone Number</label>
          <Input
            value={user?.phoneNumber || ''}
            disabled
            className="bg-background-tertiary"
          />
          <p className="text-xs text-foreground-muted">Used only for verification, never shared</p>
        </div>

        <Button
          className="w-full"
          onClick={handleSave}
          disabled={isSaving || !displayName.trim()}
        >
          {isSaving ? 'Saving...' : 'Save Changes'}
        </Button>
      </div>
    </>
  );
}

// Privacy Section
function PrivacySection({
  privacy,
  onUpdate,
  onBack,
  onNavigateToBlocked,
}: {
  privacy: PrivacySettings;
  onUpdate: (settings: Partial<PrivacySettings>) => void;
  onBack: () => void;
  onNavigateToBlocked?: () => void;
}) {
  return (
    <>
      <SectionHeader title="Privacy" onBack={onBack} />
      <div className="p-4 space-y-2">
        <ToggleItem
          icon={<Eye className="h-5 w-5" />}
          title="Read Receipts"
          description="Let others know when you've read their messages"
          enabled={privacy.readReceipts}
          onToggle={() => onUpdate({ readReceipts: !privacy.readReceipts })}
        />
        <ToggleItem
          icon={<MessageSquare className="h-5 w-5" />}
          title="Typing Indicators"
          description="Show when you're typing a message"
          enabled={privacy.typingIndicators}
          onToggle={() => onUpdate({ typingIndicators: !privacy.typingIndicators })}
        />
        <ToggleItem
          icon={<Smartphone className="h-5 w-5" />}
          title="Online Status"
          description="Show when you're online and last seen time"
          enabled={privacy.onlineStatus}
          onToggle={() => onUpdate({ onlineStatus: !privacy.onlineStatus })}
        />

        <div className="border-t border-border my-4" />

        {/* Blocked Users */}
        {onNavigateToBlocked && (
          <MenuItem
            icon={<Ban className="h-5 w-5" />}
            title="Blocked Users"
            description="Manage blocked contacts"
            onClick={onNavigateToBlocked}
          />
        )}
      </div>
    </>
  );
}

// Notifications Section
function NotificationsSection({
  notifications,
  onUpdate,
  onBack,
}: {
  notifications: NotificationSettings;
  onUpdate: (settings: Partial<NotificationSettings>) => void;
  onBack: () => void;
}) {
  return (
    <>
      <SectionHeader title="Notifications" onBack={onBack} />
      <div className="p-4 space-y-2">
        <ToggleItem
          icon={<Bell className="h-5 w-5" />}
          title="Notifications"
          description="Enable push notifications"
          enabled={notifications.enabled}
          onToggle={() => onUpdate({ enabled: !notifications.enabled })}
        />
        <ToggleItem
          icon={notifications.sound ? <Volume2 className="h-5 w-5" /> : <VolumeX className="h-5 w-5" />}
          title="Sound"
          description="Play sound for new messages"
          enabled={notifications.sound}
          onToggle={() => onUpdate({ sound: !notifications.sound })}
          disabled={!notifications.enabled}
        />
        <ToggleItem
          icon={notifications.preview ? <Eye className="h-5 w-5" /> : <EyeOff className="h-5 w-5" />}
          title="Message Preview"
          description="Show message content in notifications"
          enabled={notifications.preview}
          onToggle={() => onUpdate({ preview: !notifications.preview })}
          disabled={!notifications.enabled}
        />
      </div>
    </>
  );
}

// Appearance Section
function AppearanceSection({
  app,
  onUpdate,
  onBack,
}: {
  app: AppSettings;
  onUpdate: (settings: Partial<AppSettings>) => void;
  onBack: () => void;
}) {
  return (
    <>
      <SectionHeader title="Appearance" onBack={onBack} />
      <div className="p-4 space-y-6">
        {/* Theme */}
        <div className="space-y-3">
          <label className="text-sm font-medium">Theme</label>
          <div className="grid grid-cols-3 gap-2">
            {(['light', 'dark', 'system'] as const).map((theme) => (
              <button
                key={theme}
                onClick={() => onUpdate({ theme })}
                className={cn(
                  'flex flex-col items-center gap-2 p-4 rounded-xl border transition-colors',
                  app.theme === theme
                    ? 'border-primary bg-primary/10'
                    : 'border-border hover:bg-background-tertiary'
                )}
              >
                {theme === 'light' && <Sun className="h-6 w-6" />}
                {theme === 'dark' && <Moon className="h-6 w-6" />}
                {theme === 'system' && <Smartphone className="h-6 w-6" />}
                <span className="text-sm capitalize">{theme}</span>
              </button>
            ))}
          </div>
        </div>

        {/* Font Size */}
        <div className="space-y-3">
          <label className="text-sm font-medium">Font Size</label>
          <div className="grid grid-cols-3 gap-2">
            {(['small', 'medium', 'large'] as const).map((size) => (
              <button
                key={size}
                onClick={() => onUpdate({ fontSize: size })}
                className={cn(
                  'p-3 rounded-xl border transition-colors',
                  app.fontSize === size
                    ? 'border-primary bg-primary/10'
                    : 'border-border hover:bg-background-tertiary'
                )}
              >
                <span
                  className={cn(
                    'capitalize',
                    size === 'small' && 'text-sm',
                    size === 'medium' && 'text-base',
                    size === 'large' && 'text-lg'
                  )}
                >
                  {size}
                </span>
              </button>
            ))}
          </div>
        </div>
      </div>
    </>
  );
}

// Security Section
function SecuritySection({ onBack }: { onBack: () => void }) {
  const { user, logout } = useAuthStore();
  const { clearAll } = useChatStore();
  const [showKey, setShowKey] = useState(false);
  const [identityKey, setIdentityKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [deleteConfirmation, setDeleteConfirmation] = useState('');

  // Load identity key from Signal protocol
  useEffect(() => {
    const loadIdentityKey = async () => {
      try {
        await signalProtocol.initialize();
        const publicKey = await signalProtocol.getIdentityPublicKey();
        if (publicKey) {
          // Convert to base64 for display
          const base64Key = btoa(String.fromCharCode(...publicKey));
          setIdentityKey(base64Key);
        } else {
          setIdentityKey(user?.publicKey || null);
        }
      } catch (error) {
        console.error('Failed to load identity key:', error);
        setIdentityKey(user?.publicKey || null);
      }
    };
    loadIdentityKey();
  }, [user?.publicKey]);

  const handleCopyKey = async () => {
    if (identityKey) {
      await navigator.clipboard.writeText(identityKey);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const handleClearSessions = async () => {
    if (!confirm('This will clear all encryption sessions. You may need to re-establish connections with your contacts.')) {
      return;
    }

    // Clear sessions from IndexedDB
    const { del, keys } = await import('idb-keyval');
    const allKeys = await keys();
    const sessionKeys = allKeys.filter((k) => String(k).startsWith('signal:session:'));
    await Promise.all(sessionKeys.map((k) => del(k)));

    alert('Sessions cleared successfully');
  };

  const handleDeleteAccount = async () => {
    if (deleteConfirmation !== 'DELETE') {
      alert('Please type DELETE to confirm account deletion');
      return;
    }

    setIsDeleting(true);
    try {
      const token = useAuthStore.getState().token;
      const response = await fetch('/api/v1/users/me', {
        method: 'DELETE',
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });

      if (response.ok) {
        // Clear all local data
        const { clear } = await import('idb-keyval');
        await clear();
        clearAll();
        logout();
        alert('Your account has been permanently deleted.');
      } else {
        const error = await response.text();
        alert(`Failed to delete account: ${error}`);
      }
    } catch (error) {
      console.error('Failed to delete account:', error);
      alert('Failed to delete account. Please try again.');
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <>
      <SectionHeader title="Security" onBack={onBack} />
      <ScrollArea className="max-h-[60vh]">
        <div className="p-4 space-y-4">
          {/* Identity Key */}
          <div className="p-4 rounded-xl bg-background-tertiary space-y-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <Key className="h-5 w-5 text-primary" />
                <div>
                  <p className="font-medium">Identity Key</p>
                  <p className="text-xs text-foreground-muted">Your unique cryptographic identity</p>
                </div>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowKey(!showKey)}
              >
                {showKey ? 'Hide' : 'Show'}
              </Button>
            </div>
            {showKey && identityKey && (
              <div className="space-y-2">
                <div className="p-3 rounded-lg bg-background font-mono text-xs break-all">
                  {identityKey}
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  className="w-full"
                  onClick={handleCopyKey}
                >
                  {copied ? (
                    <>
                      <Check className="h-4 w-4 mr-2" />
                      Copied!
                    </>
                  ) : (
                    <>
                      <Copy className="h-4 w-4 mr-2" />
                      Copy Key
                    </>
                  )}
                </Button>
              </div>
            )}
            {showKey && !identityKey && (
              <div className="p-3 rounded-lg bg-background text-sm text-foreground-muted">
                No identity key available. Keys are generated during first message exchange.
              </div>
            )}
          </div>

          {/* Clear Sessions */}
          <button
            onClick={handleClearSessions}
            className="w-full flex items-center gap-3 p-4 rounded-xl hover:bg-background-tertiary transition-colors text-amber-500"
          >
            <Trash2 className="h-5 w-5" />
            <div className="text-left">
              <p className="font-medium">Clear All Sessions</p>
              <p className="text-xs text-foreground-muted">Reset encryption sessions with all contacts</p>
            </div>
          </button>

          <div className="border-t border-border my-4" />

          {/* Danger Zone - Delete Account */}
          <div className="p-4 rounded-xl border border-destructive/50 bg-destructive/5 space-y-4">
            <div className="flex items-center gap-3 text-destructive">
              <AlertTriangle className="h-5 w-5" />
              <div>
                <p className="font-medium">Danger Zone</p>
                <p className="text-xs text-destructive/80">Irreversible actions</p>
              </div>
            </div>

            <div className="space-y-3">
              <p className="text-sm text-foreground-secondary">
                Permanently delete your account and all associated data. This action cannot be undone.
              </p>
              <Input
                type="text"
                placeholder='Type "DELETE" to confirm'
                value={deleteConfirmation}
                onChange={(e) => setDeleteConfirmation(e.target.value)}
                className="border-destructive/50"
              />
              <Button
                variant="destructive"
                className="w-full"
                onClick={handleDeleteAccount}
                disabled={isDeleting || deleteConfirmation !== 'DELETE'}
              >
                {isDeleting ? 'Deleting...' : 'Delete My Account'}
              </Button>
            </div>
          </div>
        </div>
      </ScrollArea>
    </>
  );
}

// Blocked Users Section
interface BlockedUser {
  user_id: string;
  username?: string;
  display_name?: string;
  avatar_url?: string;
  blocked_at: string;
}

function BlockedUsersSection({ onBack }: { onBack: () => void }) {
  const [blockedUsers, setBlockedUsers] = useState<BlockedUser[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [unblocking, setUnblocking] = useState<string | null>(null);

  // Fetch blocked users on mount
  useEffect(() => {
    const fetchBlockedUsers = async () => {
      try {
        const token = useAuthStore.getState().token;
        const response = await fetch('/api/v1/users/blocked', {
          headers: { Authorization: `Bearer ${token}` },
        });
        if (response.ok) {
          const users = await response.json();
          setBlockedUsers(users || []);
        }
      } catch (error) {
        console.error('Failed to fetch blocked users:', error);
      } finally {
        setIsLoading(false);
      }
    };
    fetchBlockedUsers();
  }, []);

  const handleUnblock = async (userId: string) => {
    setUnblocking(userId);
    try {
      const token = useAuthStore.getState().token;
      const response = await fetch('/api/v1/users/unblock', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ user_id: userId }),
      });
      if (response.ok) {
        setBlockedUsers((prev) => prev.filter((u) => u.user_id !== userId));
      }
    } catch (error) {
      console.error('Failed to unblock user:', error);
    } finally {
      setUnblocking(null);
    }
  };

  return (
    <>
      <SectionHeader title="Blocked Users" onBack={onBack} />
      <ScrollArea className="max-h-[60vh]">
        <div className="p-4">
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-6 w-6 animate-spin text-foreground-muted" />
            </div>
          ) : blockedUsers.length === 0 ? (
            <div className="text-center py-8">
              <UserX className="h-12 w-12 mx-auto text-foreground-muted mb-3" />
              <p className="text-foreground-muted">No blocked users</p>
              <p className="text-sm text-foreground-muted/70 mt-1">
                Users you block won't be able to contact you
              </p>
            </div>
          ) : (
            <div className="space-y-2">
              {blockedUsers.map((user) => (
                <div
                  key={user.user_id}
                  className="flex items-center gap-3 p-3 rounded-xl bg-background-tertiary"
                >
                  <Avatar className="h-10 w-10">
                    <AvatarImage src={user.avatar_url} />
                    <AvatarFallback>
                      {(user.display_name || user.username || '?').charAt(0).toUpperCase()}
                    </AvatarFallback>
                  </Avatar>
                  <div className="flex-1 min-w-0">
                    <p className="font-medium truncate">
                      {user.display_name || user.username || 'Unknown'}
                    </p>
                    {user.username && (
                      <p className="text-sm text-foreground-muted truncate">@{user.username}</p>
                    )}
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleUnblock(user.user_id)}
                    disabled={unblocking === user.user_id}
                  >
                    {unblocking === user.user_id ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      'Unblock'
                    )}
                  </Button>
                </div>
              ))}
            </div>
          )}
        </div>
      </ScrollArea>
    </>
  );
}

// Reusable Components
function SectionHeader({ title, onBack }: { title: string; onBack: () => void }) {
  return (
    <DialogHeader className="p-4 border-b border-border">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" onClick={onBack}>
          <ChevronRight className="h-5 w-5 rotate-180" />
        </Button>
        <DialogTitle>{title}</DialogTitle>
      </div>
    </DialogHeader>
  );
}

function MenuItem({
  icon,
  title,
  description,
  onClick,
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className="w-full flex items-center gap-4 p-4 rounded-xl hover:bg-background-tertiary transition-colors"
    >
      <div className="p-2 rounded-lg bg-background-tertiary">{icon}</div>
      <div className="flex-1 text-left">
        <p className="font-medium">{title}</p>
        <p className="text-sm text-foreground-muted">{description}</p>
      </div>
      <ChevronRight className="h-5 w-5 text-foreground-muted" />
    </button>
  );
}

function ToggleItem({
  icon,
  title,
  description,
  enabled,
  onToggle,
  disabled,
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
  enabled: boolean;
  onToggle: () => void;
  disabled?: boolean;
}) {
  return (
    <button
      onClick={onToggle}
      disabled={disabled}
      className={cn(
        'w-full flex items-center gap-4 p-4 rounded-xl transition-colors',
        disabled ? 'opacity-50 cursor-not-allowed' : 'hover:bg-background-tertiary'
      )}
    >
      <div className="p-2 rounded-lg bg-background-tertiary">{icon}</div>
      <div className="flex-1 text-left">
        <p className="font-medium">{title}</p>
        <p className="text-sm text-foreground-muted">{description}</p>
      </div>
      <div
        className={cn(
          'w-12 h-7 rounded-full p-1 transition-colors',
          enabled ? 'bg-primary' : 'bg-background-tertiary'
        )}
      >
        <div
          className={cn(
            'w-5 h-5 rounded-full bg-white transition-transform',
            enabled ? 'translate-x-5' : 'translate-x-0'
          )}
        />
      </div>
    </button>
  );
}
