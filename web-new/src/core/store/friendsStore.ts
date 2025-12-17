/**
 * Friends Store
 * 
 * Manages friend relationships and friend requests
 */

import { create } from 'zustand';
import type { Friend, FriendRequest, FriendshipStatus } from '../types';
import { useAuthStore } from './authStore';

interface FriendsState {
    friends: Friend[];
    incomingRequests: FriendRequest[];
    outgoingRequests: FriendRequest[];
    isLoading: boolean;
    error: string | null;
}

interface FriendsActions {
    // Fetch actions
    fetchFriends: () => Promise<void>;
    fetchRequests: () => Promise<void>;
    refreshAll: () => Promise<void>;

    // Friend request actions
    sendFriendRequest: (userId: string) => Promise<boolean>;
    acceptRequest: (userId: string) => Promise<boolean>;
    declineRequest: (userId: string) => Promise<boolean>;
    cancelRequest: (userId: string) => Promise<boolean>;
    removeFriend: (userId: string) => Promise<boolean>;

    // Status check
    getFriendshipStatus: (userId: string) => Promise<FriendshipStatus>;

    // Reset
    clearAll: () => void;
}

type FriendsStore = FriendsState & FriendsActions;

const getAuthHeaders = () => {
    const token = useAuthStore.getState().token;
    return {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
    };
};

export const useFriendsStore = create<FriendsStore>((set, get) => ({
    friends: [],
    incomingRequests: [],
    outgoingRequests: [],
    isLoading: false,
    error: null,

    fetchFriends: async () => {
        const token = useAuthStore.getState().token;
        if (!token) return; // Don't fetch if not authenticated

        try {
            const response = await fetch('/api/v1/friends', {
                headers: getAuthHeaders(),
            });
            if (!response.ok) {
                // Gracefully handle API errors (e.g., table doesn't exist yet)
                console.warn('Friends API not available');
                return;
            }
            const friends = await response.json();
            set({ friends: friends || [] });
        } catch (error) {
            console.error('Failed to fetch friends:', error);
            // Don't set error state to avoid breaking the UI
        }
    },

    fetchRequests: async () => {
        const token = useAuthStore.getState().token;
        if (!token) return; // Don't fetch if not authenticated

        try {
            const response = await fetch('/api/v1/friends/requests', {
                headers: getAuthHeaders(),
            });
            if (!response.ok) {
                // Gracefully handle API errors
                console.warn('Friend requests API not available');
                return;
            }
            const data = await response.json();
            set({
                incomingRequests: data.incoming || [],
                outgoingRequests: data.outgoing || [],
            });
        } catch (error) {
            console.error('Failed to fetch friend requests:', error);
            // Don't set error state to avoid breaking the UI
        }
    },

    refreshAll: async () => {
        set({ isLoading: true, error: null });
        try {
            await Promise.all([get().fetchFriends(), get().fetchRequests()]);
        } finally {
            set({ isLoading: false });
        }
    },

    sendFriendRequest: async (userId: string) => {
        try {
            const response = await fetch('/api/v1/friends/request', {
                method: 'POST',
                headers: getAuthHeaders(),
                body: JSON.stringify({ user_id: userId }),
            });
            if (!response.ok) {
                const error = await response.text();
                throw new Error(error);
            }
            // Refresh requests to show the new outgoing request
            await get().fetchRequests();
            return true;
        } catch (error) {
            console.error('Failed to send friend request:', error);
            return false;
        }
    },

    acceptRequest: async (userId: string) => {
        try {
            const response = await fetch('/api/v1/friends/accept', {
                method: 'POST',
                headers: getAuthHeaders(),
                body: JSON.stringify({ user_id: userId }),
            });
            if (!response.ok) throw new Error('Failed to accept request');
            // Refresh both friends and requests
            await get().refreshAll();
            return true;
        } catch (error) {
            console.error('Failed to accept friend request:', error);
            return false;
        }
    },

    declineRequest: async (userId: string) => {
        try {
            const response = await fetch('/api/v1/friends/decline', {
                method: 'POST',
                headers: getAuthHeaders(),
                body: JSON.stringify({ user_id: userId }),
            });
            if (!response.ok) throw new Error('Failed to decline request');
            // Refresh requests
            await get().fetchRequests();
            return true;
        } catch (error) {
            console.error('Failed to decline friend request:', error);
            return false;
        }
    },

    cancelRequest: async (userId: string) => {
        try {
            const response = await fetch('/api/v1/friends/cancel', {
                method: 'POST',
                headers: getAuthHeaders(),
                body: JSON.stringify({ user_id: userId }),
            });
            if (!response.ok) throw new Error('Failed to cancel request');
            // Refresh requests
            await get().fetchRequests();
            return true;
        } catch (error) {
            console.error('Failed to cancel friend request:', error);
            return false;
        }
    },

    removeFriend: async (userId: string) => {
        try {
            const response = await fetch(`/api/v1/friends/${userId}`, {
                method: 'DELETE',
                headers: getAuthHeaders(),
            });
            if (!response.ok) throw new Error('Failed to remove friend');
            // Refresh friends list
            await get().fetchFriends();
            return true;
        } catch (error) {
            console.error('Failed to remove friend:', error);
            return false;
        }
    },

    getFriendshipStatus: async (userId: string): Promise<FriendshipStatus> => {
        try {
            const response = await fetch(`/api/v1/friends/${userId}/status`, {
                headers: getAuthHeaders(),
            });
            if (!response.ok) return 'none';
            const data = await response.json();
            return data.status as FriendshipStatus;
        } catch (error) {
            console.error('Failed to get friendship status:', error);
            return 'none';
        }
    },

    clearAll: () => {
        set({
            friends: [],
            incomingRequests: [],
            outgoingRequests: [],
            isLoading: false,
            error: null,
        });
    },
}));
