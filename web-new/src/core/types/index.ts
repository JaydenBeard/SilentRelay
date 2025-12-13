// User types
export interface User {
  id: string;
  phoneNumber: string;
  username?: string;
  displayName?: string;
  avatar?: string;
  publicKey: string;
  createdAt: number;
}

// Message types
export type MessageStatus = 'sending' | 'sent' | 'delivered' | 'read' | 'failed';
export type MessageType = 'text' | 'file' | 'voice' | 'image' | 'video';

export interface Message {
  id: string;
  conversationId: string;
  senderId: string;
  content: string;
  timestamp: number;
  status: MessageStatus;
  type: MessageType;
  replyTo?: string;
  metadata?: FileMetadata;
}

export interface FileMetadata {
  fileName: string;
  fileSize: number;
  mimeType: string;
  mediaId: string;
  thumbnail?: string;
  encryptionKey: number[];
  iv: number[];
}

// Conversation types
export type ConversationStatus = 'pending' | 'accepted' | 'blocked';

export interface Conversation {
  id: string;
  recipientId: string;
  recipientName: string;
  recipientAvatar?: string;
  lastMessage?: Message;
  unreadCount: number;
  isOnline: boolean;
  lastSeen?: number;
  isPinned: boolean;
  isMuted: boolean;
  status: ConversationStatus; // For message requests
}

// WebSocket message types
export type WSMessageType =
  | 'send'
  | 'deliver'
  | 'sent_ack'
  | 'status_update'
  | 'typing'
  | 'read_receipt'
  | 'delivery_ack'  // For marking messages as delivered (separate from read_receipt)
  | 'user_online'
  | 'user_offline'
  | 'user_blocked'
  | 'call_offer'
  | 'call_answer'
  | 'call_reject'   // Reject incoming call
  | 'call_end'      // End active call (hangup)
  | 'call_busy'     // User is busy
  | 'ice_candidate'
  | 'sync_request'
  | 'sync_data'
  | 'media_key'
  | 'heartbeat';

export interface WSMessage<T = unknown> {
  type: WSMessageType;
  payload: T;
  timestamp: string; // ISO 8601 format for Go compatibility
  messageId: string;
  sender_id?: string;  // Sender's user ID (from server on deliver messages)
  signature?: string; // HMAC signature for message authentication
  nonce?: string;   // Unique nonce for replay protection
}

// Encrypted message payloads (snake_case to match Go backend)
export interface EncryptedPayload {
  sender_id?: string;
  receiver_id?: string;
  ciphertext: string;
  message_type: 'prekey' | 'whisper';
  ephemeral_key?: string;
}

export interface TypingPayload {
  receiver_id?: string;  // Backend expects receiver_id (snake_case)
  is_typing: boolean;
}

export interface ReadReceiptPayload {
  message_ids: string[];  // Backend expects array of message IDs
  status: 'delivered' | 'read';
}

export interface PresencePayload {
  user_id: string;  // Backend sends snake_case
  isOnline?: boolean;  // Set by frontend handler
  lastSeen?: number;
  last_seen?: number;  // Backend sends snake_case
}

export interface StatusUpdatePayload {
  messageId: string;
  status: MessageStatus;
}

// Call types
export type CallType = 'audio' | 'video';
export type CallState = 'idle' | 'calling' | 'ringing' | 'connected' | 'ended';

export interface CallOffer {
  callId: string;
  callerId: string;
  callerName: string;
  callType: CallType;
  sdp: string;
}

export interface CallAnswer {
  callId: string;
  sdp: string;
}

export interface IceCandidate {
  callId: string;
  candidate: RTCIceCandidateInit;
}

// Auth types
export interface AuthState {
  token: string | null;
  refreshToken: string | null;
  user: User | null;
  deviceId: string | null;
}

// Settings types
export interface PrivacySettings {
  readReceipts: boolean;
  onlineStatus: boolean;
  lastSeen: boolean;
  typingIndicators: boolean;
}

export interface NotificationSettings {
  enabled: boolean;
  sound: boolean;
  preview: boolean;
}

export interface AppSettings {
  theme: 'dark' | 'light' | 'system';
  fontSize: 'small' | 'medium' | 'large';
  language: string;
}
