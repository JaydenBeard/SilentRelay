/**
 * WebSocket Hook
 *
 * Provides WebSocket connectivity with automatic reconnection,
 * message handling, and integration with the chat store.
 */

import { useEffect, useRef, useCallback } from 'react';
import { useAuthStore } from '@/core/store/authStore';
import { useChatStore } from '@/core/store/chatStore';
import { useCallStore } from '@/core/store/callStore';
import { useSettingsStore } from '@/core/store/settingsStore';
import { WebSocketService } from '@/core/services/websocket';
import { signalProtocol } from '@/core/crypto/signal';
import { webrtcService, type CallSignal } from '@/core/services/webrtc';
import {
  exportSessionBundle,
  importSessionBundle,
  setSyncState,
  canActAsPrimary,
} from '@/core/services/deviceSync';
import type {
  EncryptedPayload,
  TypingPayload,
  ReadReceiptPayload,
  PresencePayload,
  StatusUpdatePayload,
  WSMessage,
  Message,
} from '@/core/types';

// Call signaling payload types (snake_case to match Go backend JSON)
interface CallOfferPayload {
  call_id: string;
  call_type: 'audio' | 'video';
  sender_name?: string;
  sender_avatar?: string;
  offer: RTCSessionDescriptionInit;
}

interface CallAnswerPayload {
  call_id: string;
  answer: RTCSessionDescriptionInit;
}

interface CallCandidatePayload {
  call_id: string;
  candidate: RTCIceCandidateInit;
}

interface CallHangupPayload {
  call_id: string;
  reason?: string;
}

interface MediaKeyPayload {
  media_id: string;
  recipient_id: string;
  encrypted_key: string;
  algorithm: string;
  timestamp: string;
}

// Device sync payload types
interface SyncRequestPayload {
  requesting_device_id: string;
  timestamp: number;
}

interface SyncDataPayload {
  encrypted: string;
  iv: string;
  source_device_id: string;
}

interface IdentityKeyChangedPayload {
  user_id: string;
  new_fingerprint: string;
  timestamp: number;
}

export function useWebSocket() {
  const wsRef = useRef<WebSocketService | null>(null);
  const { token, isAuthenticated, needsOnboarding, onboardingStep } = useAuthStore();

  // Handle incoming encrypted message
  const handleMessage = useCallback(
    async (payload: EncryptedPayload, wsMessage: WSMessage<EncryptedPayload>) => {
      // Get sender_id from the outer WebSocket message (set by server)
      const senderId = wsMessage.sender_id || payload.sender_id;
      // Get current user from store (avoid stale closure)
      const currentUser = useAuthStore.getState().user;
      if (!currentUser || !senderId) {
        console.error('No sender_id in message:', { wsMessage, payload });
        return;
      }

      try {
        // Decrypt the message
        const ciphertextBytes = Uint8Array.from(atob(payload.ciphertext), (c) => c.charCodeAt(0));
        const plaintext = await signalProtocol.decryptMessage(
          senderId,
          1, // Device ID
          ciphertextBytes,
          payload.message_type
        );

        // Message decrypted successfully (content not logged for security)

        // Create message object
        const message: Message = {
          id: wsMessage.messageId || crypto.randomUUID(),
          conversationId: senderId,
          senderId: senderId,
          content: plaintext,
          timestamp: Date.now(),
          status: 'delivered',
          type: 'text',
        };

        // Use getState() to avoid stale closures
        useChatStore.getState().addMessage(message);

        // Mark sender as online (they just sent a message, so they're definitely online)
        useChatStore.getState().updatePresence(senderId, true);

        // Only send delivery receipt if conversation is ACCEPTED (privacy protection)
        // Check the conversation status from the store
        const conversation = useChatStore.getState().conversations[senderId];
        if (conversation?.status === 'accepted') {
          wsRef.current?.sendReadReceipt(message.id, message.conversationId, 'delivered');
        }
      } catch (error) {
        console.error('Failed to decrypt message:', error);

        // Check if this is a key mismatch error
        const errorMsg = String(error);
        if (errorMsg.includes('BAD_MESSAGE_KEY_ID') || errorMsg.includes('BAD_MESSAGE_FORMAT')) {
          console.error('Key mismatch detected. The sender may have used outdated keys.');
          console.error('To fix: Clear IndexedDB (Application > IndexedDB > keyval-store) and re-login');

          // Add a placeholder message showing decryption failed
          const failedMessage: Message = {
            id: wsMessage.messageId || crypto.randomUUID(),
            conversationId: senderId,
            senderId: senderId,
            content: '🔐 Unable to decrypt message (key mismatch). Please ask sender to resend.',
            timestamp: Date.now(),
            status: 'delivered',
            type: 'text',
          };
          useChatStore.getState().addMessage(failedMessage);
        }
      }
    },
    [] // No dependencies - uses getState() for stability
  );

  // Handle typing indicator with timeout management
  const typingTimeoutsRef = useRef<Record<string, NodeJS.Timeout>>({});

  const handleTyping = useCallback((payload: TypingPayload, message: WSMessage<TypingPayload>) => {
    const senderId = message.sender_id;
    if (!senderId) return;

    // Clear existing timeout for this sender
    if (typingTimeoutsRef.current[senderId]) {
      clearTimeout(typingTimeoutsRef.current[senderId]);
      delete typingTimeoutsRef.current[senderId];
    }

    // Use getState() for stability
    useChatStore.getState().setTyping(senderId, payload.is_typing);

    // If they're typing, they're online - update presence
    if (payload.is_typing) {
      useChatStore.getState().updatePresence(senderId, true);
    }

    // Auto-clear typing after 5 seconds (in case we miss the is_typing=false)
    if (payload.is_typing) {
      typingTimeoutsRef.current[senderId] = setTimeout(() => {
        useChatStore.getState().setTyping(senderId, false);
        delete typingTimeoutsRef.current[senderId];
      }, 5000);
    }
  }, []); // No dependencies - uses getState() for stability

  // Handle read receipt
  const handleReadReceipt = useCallback(
    (payload: ReadReceiptPayload) => {
      // Backend sends message_ids array
      payload.message_ids?.forEach((msgId) => {
        useChatStore.getState().updateMessageStatus(msgId, payload.status === 'read' ? 'read' : 'delivered');
      });
    },
    [] // No dependencies - uses getState() for stability
  );

  // Handle presence update
  const handlePresence = useCallback(
    (payload: PresencePayload & { isOnline: boolean }) => {
      // Backend sends user_id and last_seen in snake_case
      // Convert last_seen from Unix seconds to milliseconds for JavaScript Date
      const lastSeenMs = payload.last_seen
        ? payload.last_seen * 1000  // Convert Unix seconds to milliseconds
        : payload.lastSeen;
      useChatStore.getState().updatePresence(payload.user_id, payload.isOnline, lastSeenMs);
    },
    [] // No dependencies - uses getState() for stability
  );

  // Handle being blocked by another user
  const handleUserBlocked = useCallback(
    (payload: { blocker_id: string }) => {
      // When someone blocks us, remove the conversation from our view
      if (payload.blocker_id) {
        useChatStore.getState().removeConversation(payload.blocker_id);
      }
    },
    [] // No dependencies - uses getState() for stability
  );

  // Handle status update (sent_ack)
  const handleStatusUpdate = useCallback(
    (payload: StatusUpdatePayload, message: WSMessage<StatusUpdatePayload>) => {
      // The message ID is on the outer WSMessage, not in the payload
      const messageId = message.messageId || payload.messageId;
      const status = payload.status;

      if (messageId && status) {
        useChatStore.getState().updateMessageStatus(messageId, status as 'delivered' | 'read');
      }
    },
    [] // No dependencies - uses getState() for stability
  );

  // Handle call offer - extract sender_id from message envelope
  const handleCallOffer = useCallback((payload: CallOfferPayload, message: WSMessage<CallOfferPayload>) => {
    const senderId = message.sender_id || '';
    // Try to get sender name from payload, or fallback to fetching profile
    const senderName = payload.sender_name || `User ${senderId.slice(0, 8)}...`;
    const senderAvatar = payload.sender_avatar;

    webrtcService.handleIncomingCall(
      payload.call_id,
      senderId,
      senderName,
      senderAvatar,
      payload.call_type,
      payload.offer
    );
  }, []);

  // Handle call answer
  const handleCallAnswer = useCallback((payload: CallAnswerPayload, message: WSMessage<CallAnswerPayload>) => {
    console.log('[WS] Received call_answer:', {
      answer: payload.answer ? 'present' : 'missing',
      sender_id: message.sender_id,
      type: message.type
    });
    webrtcService.handleAnswer(payload.answer);
  }, []);

  // Handle ICE candidate
  const handleCallCandidate = useCallback((payload: CallCandidatePayload, message: WSMessage<CallCandidatePayload>) => {
    console.log('[WS] Received ice_candidate:', {
      candidate: payload.candidate ? 'present' : 'missing',
      sender_id: message.sender_id,
      type: message.type
    });
    webrtcService.handleIceCandidate(payload.candidate);
  }, []);

  // Handle call hangup
  const handleCallHangup = useCallback((payload: CallHangupPayload, message: WSMessage<CallHangupPayload>) => {
    console.log('[WS] Received call_end:', {
      reason: payload.reason,
      sender_id: message.sender_id,
      type: message.type
    });
    webrtcService.handleRemoteHangup(payload.reason);
  }, []);

  // Handle media key exchange
  const handleMediaKey = useCallback((_payload: MediaKeyPayload) => {
    // TODO: Store the media key for decryption when downloading media
    // This would involve updating a media key store and triggering decryption
  }, []);

  // Handle sync request from another device (this device is primary)
  const handleSyncRequest = useCallback(async (payload: SyncRequestPayload, message: WSMessage<SyncRequestPayload>) => {
    console.log('[Sync] Received sync request from device:', payload.requesting_device_id);

    // Check if we can act as primary
    const isPrimary = await canActAsPrimary();
    if (!isPrimary) {
      console.log('[Sync] Cannot act as primary - no account');
      return;
    }

    try {
      // Export our session bundle
      const bundle = await exportSessionBundle();

      // Get the master key from signalProtocol to encrypt the bundle
      // Note: The recipient will need to use the same PIN to decrypt
      // For now, we send the bundle as-is (already has encrypted account pickle)

      // Send sync data back to the requesting device
      wsRef.current?.send('sync_data', {
        target_device_id: message.sender_id,
        bundle: bundle,
        source_device_id: useAuthStore.getState().user?.id,
      });

      console.log('[Sync] Sent session bundle to requesting device');
    } catch (error) {
      console.error('[Sync] Failed to export sessions:', error);
    }
  }, []);

  // Handle sync data received from primary device (this device is new)
  const handleSyncData = useCallback(async (payload: SyncDataPayload) => {
    console.log('[Sync] Received sync data from device:', payload.source_device_id);

    try {
      // If encrypted, decrypt it first
      if (payload.encrypted && payload.iv) {
        // We need the master key - this should be available if user already entered PIN
        // For now, handle the unencrypted bundle case
        console.log('[Sync] Encrypted bundle received - decryption requires master key');
        setSyncState({ status: 'failed', message: 'Encrypted sync not yet implemented' });
        return;
      }

      // Import the session bundle
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const bundle = (payload as any).bundle;
      if (bundle) {
        await importSessionBundle(bundle);
        setSyncState({ status: 'success', message: 'Sessions synced successfully!' });
      }
    } catch (error) {
      console.error('[Sync] Failed to import sessions:', error);
      setSyncState({ status: 'failed', message: 'Failed to import sessions' });
    }
  }, []);

  // Handle identity key change notification
  const handleIdentityKeyChanged = useCallback((payload: IdentityKeyChangedPayload) => {
    console.log('[Security] Identity key changed for user:', payload.user_id);
    // Store this in the chat store to show safety number banner
    useChatStore.getState().setIdentityKeyChanged(payload.user_id, true);
  }, []);

  // Set up WebRTC service handlers
  useEffect(() => {
    const { setCurrentCall, setLocalStream, setRemoteStream } = useCallStore.getState();

    webrtcService.setHandlers({
      onCallStateChange: (state) => {
        setCurrentCall(state);
        if (state.status === 'ended') {
          setLocalStream(null);
          setRemoteStream(null);
        } else {
          setLocalStream(webrtcService.getLocalStream());
        }
      },
      onRemoteStream: (stream) => {
        setRemoteStream(stream);
      },
      onSignal: (signal: CallSignal) => {
        // Send signal through WebSocket
        if (!wsRef.current) {
          console.warn('[WS] Cannot send signal - WebSocket not connected');
          return;
        }

        // Get current user info for call offers
        const currentUser = useAuthStore.getState().user;

        console.log('[WS] Sending signal:', signal.type, 'to peer:', signal.peerId);

        switch (signal.type) {
          case 'offer':
            console.log('[WS] Sending call_offer with SDP');
            wsRef.current.send('call_offer', {
              call_id: signal.callId,
              recipient_id: signal.peerId,
              call_type: webrtcService.getCurrentCall()?.type || 'audio',
              sender_name: currentUser?.displayName || currentUser?.username,
              sender_avatar: currentUser?.avatar,
              offer: signal.data,
            });
            break;
          case 'answer':
            console.log('[WS] Sending call_answer with SDP to:', signal.peerId);
            wsRef.current.send('call_answer', {
              call_id: signal.callId,
              recipient_id: signal.peerId,
              answer: signal.data,
            });
            break;
          case 'candidate':
            console.log('[WS] Sending ice_candidate to:', signal.peerId);
            wsRef.current.send('ice_candidate', {
              call_id: signal.callId,
              recipient_id: signal.peerId,
              candidate: signal.data,
            });
            break;
          case 'hangup':
            console.log('[WS] Sending call_end to:', signal.peerId);
            wsRef.current.send('call_end', {
              call_id: signal.callId,
              recipient_id: signal.peerId,
            });
            break;
        }
      },
    });
  }, []);

  // Initialize Signal Protocol on mount
  // NOTE: Keys are generated during REGISTRATION only (in useAuth.ts)
  // This effect just initializes the protocol - it should NOT regenerate keys
  useEffect(() => {
    const initSignal = async () => {
      try {
        await signalProtocol.initialize();
      } catch {
        // Signal protocol initialization failed - will retry on next message
      }
    };

    if (isAuthenticated) {
      initSignal();
    }
  }, [isAuthenticated]);

  // Connect to WebSocket
  // Only connect WebSocket after onboarding is complete (PIN encryption set up)
  const isOnboardingComplete = !needsOnboarding || onboardingStep === 'complete';

  useEffect(() => {
    if (!isAuthenticated || !token || !isOnboardingComplete) {
      return;
    }

    const wsUrl = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws`;

    const ws = new WebSocketService({
      url: wsUrl,
      token,
      onConnect: () => {
        // Use getState() for stability
        useChatStore.getState().syncPresenceToConversations();
        // Fetch fresh presence data for all contacts
        useChatStore.getState().refreshAllContactsPresence();
      },
      onDisconnect: () => {
        // WebSocket disconnected - will auto-reconnect
      },
      onError: () => {
        // WebSocket error - will auto-reconnect
      },
    });

    // Set up handlers
    ws.on('deliver', handleMessage);
    ws.on('typing', handleTyping);
    ws.on('read_receipt', handleReadReceipt);
    ws.on('user_online', (payload: PresencePayload) => handlePresence({ ...payload, isOnline: true }));
    ws.on('user_offline', (payload: PresencePayload) => handlePresence({ ...payload, isOnline: false }));
    ws.on('user_blocked', handleUserBlocked);
    ws.on('status_update', handleStatusUpdate);
    ws.on('sent_ack', handleStatusUpdate);

    // Call signaling handlers
    ws.on('call_offer', handleCallOffer);
    ws.on('call_answer', handleCallAnswer);
    ws.on('ice_candidate', handleCallCandidate);
    ws.on('call_end', handleCallHangup);

    // Media key exchange handler
    ws.on('media_key', handleMediaKey);

    // Device sync handlers
    ws.on('sync_request', handleSyncRequest);
    ws.on('sync_data', handleSyncData);
    ws.on('identity_key_changed', handleIdentityKeyChanged);

    // Connect
    ws.connect().catch(console.error);
    wsRef.current = ws;

    return () => {
      ws.disconnect();
      wsRef.current = null;
    };
  }, [
    isAuthenticated,
    token,
    isOnboardingComplete,
    handleMessage,
    handleTyping,
    handleReadReceipt,
    handlePresence,
    handleUserBlocked,
    handleStatusUpdate,
    handleCallOffer,
    handleCallAnswer,
    handleCallCandidate,
    handleCallHangup,
    handleMediaKey,
    handleSyncRequest,
    handleSyncData,
    handleIdentityKeyChanged,
  ]);

  // Helper function to decode base64 to Uint8Array
  const base64ToUint8Array = useCallback((base64: string): Uint8Array => {
    const binaryString = atob(base64);
    const bytes = new Uint8Array(binaryString.length);
    for (let i = 0; i < binaryString.length; i++) {
      bytes[i] = binaryString.charCodeAt(i);
    }
    return bytes;
  }, []);

  // Send message function
  const sendMessage = useCallback(
    async (recipientId: string, content: string): Promise<Message | null> => {
      // Get current state to avoid stale closures
      const currentUser = useAuthStore.getState().user;
      const currentToken = useAuthStore.getState().token;

      if (!wsRef.current || !currentUser) return null;

      try {
        // Check if we have a session, if not create one
        const hasSession = await signalProtocol.hasSession(recipientId, 1);
        if (!hasSession) {
          // Fetch recipient's keys and create session
          const response = await fetch(`/api/v1/users/${recipientId}/keys`, {
            headers: { Authorization: `Bearer ${currentToken}` },
          });
          if (!response.ok) throw new Error('Failed to fetch recipient keys');

          const keys = await response.json();

          // Parse backend format to Signal Protocol format
          // Backend returns: identity_key, signed_prekey, signed_prekey_signature (all base64 strings)
          // Backend returns: onetime_prekey_id, onetime_prekey (optional, for one-time prekey)
          await signalProtocol.createSession(recipientId, 1, {
            registrationId: 1, // Default since server doesn't store this
            identityKey: base64ToUint8Array(keys.identity_key),
            signedPreKeyId: 1, // Default since server doesn't store signed prekey ID
            signedPreKey: base64ToUint8Array(keys.signed_prekey),
            signedPreKeySignature: base64ToUint8Array(keys.signed_prekey_signature),
            preKeyId: keys.onetime_prekey_id,
            preKey: keys.onetime_prekey ? base64ToUint8Array(keys.onetime_prekey) : undefined,
          });
        }

        // Encrypt the message
        const encrypted = await signalProtocol.encryptMessage(recipientId, 1, content);

        // Create local message
        const message: Message = {
          id: crypto.randomUUID(),
          conversationId: recipientId,
          senderId: currentUser.id,
          content,
          timestamp: Date.now(),
          status: 'sending',
          type: 'text',
        };

        // Add to store immediately for optimistic UI
        useChatStore.getState().addMessage(message);

        // Send via WebSocket (snake_case to match Go backend)
        const payload: EncryptedPayload = {
          receiver_id: recipientId,
          ciphertext: btoa(String.fromCharCode(...encrypted.ciphertext)),
          message_type: encrypted.messageType,
        };

        // Pass the local message ID so server uses it for status updates
        await wsRef.current.sendEncryptedMessage(payload, message.id);

        // Update status to sent (will be updated to delivered/read via receipts)
        useChatStore.getState().updateMessageStatus(message.id, 'sent');

        return message;
      } catch (error) {
        console.error('Failed to send message:', error);
        return null;
      }
    },
    [base64ToUint8Array] // Only depends on stable utility function
  );

  // Send typing indicator
  // Send typing indicator (respects privacy settings)
  const sendTyping = useCallback(
    (recipientId: string, isTyping: boolean) => {
      // Check if user has typing indicators enabled
      const { privacy } = useSettingsStore.getState();
      if (!privacy.typingIndicators) return;

      wsRef.current?.sendTypingIndicator(recipientId, isTyping);
    },
    []
  );

  // Send read receipt (respects privacy settings)
  const sendReadReceipt = useCallback(
    (messageId: string, conversationId: string) => {
      // Check if user has read receipts enabled
      const { privacy } = useSettingsStore.getState();
      if (!privacy.readReceipts) return;

      wsRef.current?.sendReadReceipt(messageId, conversationId, 'read');
    },
    []
  );

  // Start a call
  const startCall = useCallback(
    async (
      peerId: string,
      peerName: string,
      peerAvatar: string | undefined,
      callType: 'audio' | 'video'
    ) => {
      try {
        await webrtcService.startCall(peerId, peerName, peerAvatar, callType);
      } catch (error) {
        console.error('Failed to start call:', error);
        throw error;
      }
    },
    []
  );

  // Send media key
  const sendMediaKey = useCallback(
    (mediaId: string, recipientId: string, encryptedKey: string, algorithm: string) => {
      wsRef.current?.send('media_key', {
        media_id: mediaId,
        recipient_id: recipientId,
        encrypted_key: encryptedKey,
        algorithm,
        timestamp: new Date().toISOString(),
      });
    },
    []
  );

  return {
    isConnected: wsRef.current?.isConnected() ?? false,
    sendMessage,
    sendTyping,
    sendReadReceipt,
    sendMediaKey,
    startCall,
  };
}
