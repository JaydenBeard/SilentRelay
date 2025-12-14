import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { Conversation, Message, MessageStatus, ConversationStatus } from '../types';
import { encryptChatData, decryptChatData } from '../crypto/storage';
import { users } from '../api/client';

interface ChatState {
  conversations: Record<string, Conversation>;
  messages: Record<string, Message[]>;
  activeConversationId: string | null;
  presenceCache: Record<string, { isOnline: boolean; lastSeen?: number }>;
  typingUsers: Record<string, boolean>; // recipientId -> isTyping
  identityKeyChangedUsers: Record<string, boolean>; // userId -> has key changed
}

interface ChatActions {
  // Conversation actions
  setActiveConversation: (id: string | null) => void;
  addConversation: (conversation: Conversation) => void;
  updateConversation: (id: string, updates: Partial<Conversation>) => void;
  removeConversation: (id: string) => void;

  // Message actions
  addMessage: (message: Message) => void;
  addMessageOptimistically: (message: Message) => string; // Returns temp ID
  confirmMessage: (tempId: string, realId: string) => void; // Update temp ID to real ID
  failMessage: (tempId: string) => void; // Remove failed optimistic message
  updateMessage: (messageId: string, updates: Partial<Message>) => void;
  updateMessageStatus: (messageId: string, status: MessageStatus) => void;
  setMessages: (conversationId: string, messages: Message[]) => void;

  // Presence actions
  updatePresence: (userId: string, isOnline: boolean, lastSeen?: number) => void;
  syncPresenceToConversations: () => void;
  setTyping: (userId: string, isTyping: boolean) => void;

  // Utility actions
  markConversationRead: (conversationId: string) => void;
  clearAll: () => void;
  fetchUserProfile: (userId: string) => Promise<void>;
  refreshAllContactsPresence: () => Promise<void>;

  // Message request actions
  acceptConversation: (conversationId: string) => void;
  declineConversation: (conversationId: string) => void;
  blockConversation: (conversationId: string) => void;
  updateConversationStatus: (conversationId: string, status: ConversationStatus) => void;

  // Security actions
  setIdentityKeyChanged: (userId: string, changed: boolean) => void;
}

type ChatStore = ChatState & ChatActions;

// Custom storage with encryption support
const encryptedStorage = {
  getItem: async (key: string) => {
    try {
      const stored = localStorage.getItem(key);
      if (!stored) return null;

      // Check if data is encrypted (contains ':')
      if (stored.includes(':')) {
        try {
          const decrypted = await decryptChatData(stored);
          return { state: decrypted };
        } catch (error) {
          console.error('Failed to decrypt chat data:', error);
          localStorage.removeItem(key);
          return null;
        }
      } else {
        return { state: JSON.parse(stored) };
      }
    } catch (error) {
      console.error('Error reading from storage:', error);
      return null;
    }
  },
  setItem: async (key: string, value: { state: Partial<ChatStore>; version?: number }): Promise<void> => {
    try {
      const encrypted = await encryptChatData(value.state);
      localStorage.setItem(key, encrypted);
    } catch (error) {
      // Fallback to unencrypted storage when encryption isn't available
      console.warn('Encryption unavailable, storing unencrypted:', error);
      localStorage.setItem(key, JSON.stringify(value.state));
    }
  },
  removeItem: (key: string): void => {
    localStorage.removeItem(key);
  },
};

export const useChatStore = create<ChatStore>()(
  persist(
    (set, get) => ({
      conversations: {},
      messages: {},
      activeConversationId: null,
      presenceCache: {},
      typingUsers: {},
      identityKeyChangedUsers: {},

      // Conversation actions
      setActiveConversation: (id) => set({ activeConversationId: id }),

      addConversation: (conversation) =>
        set((state) => ({
          conversations: {
            ...state.conversations,
            [conversation.id]: conversation,
          },
        })),

      updateConversation: (id, updates) =>
        set((state) => {
          const existing = state.conversations[id];
          if (!existing) return state;
          return {
            conversations: {
              ...state.conversations,
              [id]: { ...existing, ...updates },
            },
          };
        }),

      removeConversation: (id) =>
        set((state) => {
          const { [id]: _, ...rest } = state.conversations;
          const { [id]: __, ...restMessages } = state.messages;
          return {
            conversations: rest,
            messages: restMessages,
            activeConversationId:
              state.activeConversationId === id ? null : state.activeConversationId,
          };
        }),

      // Message actions
      addMessage: (message) => {
        const state = get();
        const conversationMessages = state.messages[message.conversationId] || [];

        // Check for duplicate
        if (conversationMessages.some((m) => m.id === message.id)) {
          return;
        }

        // Check if this is a new conversation
        const isNewConversation = !state.conversations[message.conversationId];

        // Update or create conversation
        const existingConversation = state.conversations[message.conversationId];

        // Determine conversation status for new conversations:
        // - If there are existing messages, this is a returning contact (accepted)
        // - If the message is from self, we initiated it (accepted)
        // - Otherwise, it's a new message request (pending)
        const hasExistingMessages = conversationMessages.length > 0;
        const isSelfMessage = message.senderId === 'self' || message.senderId === state.conversations[message.conversationId]?.recipientId;
        const newConversationStatus: ConversationStatus =
          hasExistingMessages || isSelfMessage ? 'accepted' : 'pending';

        const updatedConversation = existingConversation
          ? {
            ...existingConversation,
            lastMessage: message,
            unreadCount:
              state.activeConversationId !== message.conversationId &&
                message.senderId !== 'self'
                ? existingConversation.unreadCount + 1
                : existingConversation.unreadCount,
          }
          : {
            // Create new conversation for incoming message from new sender
            id: message.conversationId,
            recipientId: message.senderId,
            recipientName: `User ${message.senderId.slice(0, 8)}...`, // Placeholder name - will be updated
            lastMessage: message,
            unreadCount: state.activeConversationId !== message.conversationId ? 1 : 0,
            isOnline: false,
            isPinned: false,
            isMuted: false,
            status: newConversationStatus,
          };

        set({
          messages: {
            ...state.messages,
            [message.conversationId]: [...conversationMessages, message],
          },
          conversations: {
            ...state.conversations,
            [message.conversationId]: updatedConversation,
          },
        });

        // Fetch user profile for new conversations (async, don't block)
        if (isNewConversation && message.senderId !== 'self') {
          get().fetchUserProfile(message.senderId);
        }
      },

      addMessageOptimistically: (message) => {
        const tempId = `temp-${Date.now()}-${Math.random()}`;
        const optimisticMessage = { ...message, id: tempId, status: 'sending' as MessageStatus };

        get().addMessage(optimisticMessage);
        return tempId;
      },

      confirmMessage: (tempId, realId) =>
        set((state) => {
          const newMessages = { ...state.messages };
          for (const convId of Object.keys(newMessages)) {
            const idx = newMessages[convId].findIndex((m) => m.id === tempId);
            if (idx !== -1) {
              newMessages[convId] = [...newMessages[convId]];
              newMessages[convId][idx] = {
                ...newMessages[convId][idx],
                id: realId,
                status: 'sent' as MessageStatus
              };
              break;
            }
          }
          return { messages: newMessages };
        }),

      failMessage: (tempId) =>
        set((state) => {
          const newMessages = { ...state.messages };
          for (const convId of Object.keys(newMessages)) {
            newMessages[convId] = newMessages[convId].filter((m) => m.id !== tempId);
          }
          return { messages: newMessages };
        }),

      updateMessage: (messageId, updates) =>
        set((state) => {
          const newMessages = { ...state.messages };
          for (const convId of Object.keys(newMessages)) {
            const idx = newMessages[convId].findIndex((m) => m.id === messageId);
            if (idx !== -1) {
              newMessages[convId] = [...newMessages[convId]];
              newMessages[convId][idx] = { ...newMessages[convId][idx], ...updates };
              break;
            }
          }
          return { messages: newMessages };
        }),

      updateMessageStatus: (messageId, status) =>
        get().updateMessage(messageId, { status }),

      setMessages: (conversationId, messages) =>
        set((state) => ({
          messages: {
            ...state.messages,
            [conversationId]: messages,
          },
        })),

      // Presence actions
      // Note: When lastSeen is undefined, we CLEAR it (privacy mode)
      // This is intentional - if no lastSeen is sent, user has privacy enabled
      updatePresence: (userId, isOnline, lastSeen) =>
        set((state) => {
          // Update presence cache - clear lastSeen if not provided (privacy mode)
          const newPresenceCache = {
            ...state.presenceCache,
            [userId]: { isOnline, lastSeen }, // Don't preserve old lastSeen
          };

          // Update matching conversations
          const newConversations = { ...state.conversations };
          for (const convId of Object.keys(newConversations)) {
            if (newConversations[convId].recipientId === userId) {
              newConversations[convId] = {
                ...newConversations[convId],
                isOnline,
                lastSeen, // Don't preserve old lastSeen - clear it if not provided
              };
            }
          }

          return {
            presenceCache: newPresenceCache,
            conversations: newConversations,
          };
        }),

      syncPresenceToConversations: () =>
        set((state) => {
          const newConversations = { ...state.conversations };
          for (const convId of Object.keys(newConversations)) {
            const recipientId = newConversations[convId].recipientId;
            const presence = state.presenceCache[recipientId];
            if (presence) {
              newConversations[convId] = {
                ...newConversations[convId],
                isOnline: presence.isOnline,
                lastSeen: presence.lastSeen,
              };
            }
          }
          return { conversations: newConversations };
        }),

      setTyping: (userId, isTyping) =>
        set((state) => ({
          typingUsers: {
            ...state.typingUsers,
            [userId]: isTyping,
          },
        })),

      // Utility actions
      markConversationRead: (conversationId) =>
        set((state) => {
          const conversation = state.conversations[conversationId];
          if (!conversation) return state;
          return {
            conversations: {
              ...state.conversations,
              [conversationId]: { ...conversation, unreadCount: 0 },
            },
          };
        }),

      clearAll: () =>
        set({
          conversations: {},
          messages: {},
          activeConversationId: null,
          presenceCache: {},
          typingUsers: {},
          identityKeyChangedUsers: {},
        }),

      fetchUserProfile: async (userId: string) => {
        try {
          const profile = await users.getUserProfile(userId);
          const displayName = profile.display_name || profile.username || `User ${userId.slice(0, 8)}...`;

          set((state) => {
            // Update all conversations with this user
            const newConversations = { ...state.conversations };
            for (const convId of Object.keys(newConversations)) {
              if (newConversations[convId].recipientId === userId) {
                newConversations[convId] = {
                  ...newConversations[convId],
                  recipientName: displayName,
                  recipientAvatar: profile.avatar_url,
                  isOnline: profile.is_online || false,
                  lastSeen: profile.last_seen ? new Date(profile.last_seen).getTime() : undefined,
                };
              }
            }
            return { conversations: newConversations };
          });
        } catch (error) {
          console.error('Failed to fetch user profile:', error);
        }
      },

      refreshAllContactsPresence: async () => {
        const state = get();
        const conversationIds = Object.keys(state.conversations);

        // Get unique recipient IDs
        const recipientIds = new Set<string>();
        for (const convId of conversationIds) {
          const conv = state.conversations[convId];
          if (conv?.recipientId) {
            recipientIds.add(conv.recipientId);
          }
        }

        // Fetch presence for each contact (in parallel, limited batches)
        const fetchPromises = Array.from(recipientIds).map(userId =>
          get().fetchUserProfile(userId)
        );

        await Promise.allSettled(fetchPromises);
      },

      // Message request actions
      acceptConversation: (conversationId) => {
        const state = get();
        const conversation = state.conversations[conversationId];
        if (!conversation) return;

        // Update status to accepted
        set({
          conversations: {
            ...state.conversations,
            [conversationId]: { ...conversation, status: 'accepted' as ConversationStatus },
          },
        });

        // Fetch user's profile to get current online status
        get().fetchUserProfile(conversation.recipientId);
      },

      declineConversation: (conversationId) =>
        set((state) => {
          // Remove the conversation and its messages
          const { [conversationId]: _, ...remainingConversations } = state.conversations;
          const { [conversationId]: __, ...remainingMessages } = state.messages;
          return {
            conversations: remainingConversations,
            messages: remainingMessages,
            activeConversationId:
              state.activeConversationId === conversationId ? null : state.activeConversationId,
          };
        }),

      blockConversation: async (conversationId) => {
        const state = get();
        const conversation = state.conversations[conversationId];
        if (!conversation) return;

        // Remove the conversation locally (so if they're unblocked later, new messages appear as requests)
        const { [conversationId]: _, ...remainingConversations } = state.conversations;
        const { [conversationId]: __, ...remainingMessages } = state.messages;
        set({
          conversations: remainingConversations,
          messages: remainingMessages,
          activeConversationId:
            state.activeConversationId === conversationId ? null : state.activeConversationId,
        });

        // Call backend API to persist block
        try {
          const { useAuthStore } = await import('./authStore');
          const token = useAuthStore.getState().token;
          await fetch('/api/v1/users/block', {
            method: 'POST',
            headers: {
              Authorization: `Bearer ${token}`,
              'Content-Type': 'application/json',
            },
            body: JSON.stringify({ user_id: conversation.recipientId }),
          });
        } catch (error) {
          console.error('Failed to block user on server:', error);
        }
      },

      updateConversationStatus: (conversationId, status) =>
        set((state) => {
          const conversation = state.conversations[conversationId];
          if (!conversation) return state;
          return {
            conversations: {
              ...state.conversations,
              [conversationId]: { ...conversation, status },
            },
          };
        }),

      // Security actions
      setIdentityKeyChanged: (userId, changed) =>
        set((state) => ({
          identityKeyChangedUsers: {
            ...state.identityKeyChangedUsers,
            [userId]: changed,
          },
        })),
    }),
    {
      name: 'chat-storage',
      storage: encryptedStorage,
      partialize: (state) => ({
        conversations: state.conversations,
        messages: state.messages,
        presenceCache: state.presenceCache,
      }),
    }
  )
);
