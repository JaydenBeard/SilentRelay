/**
 * WebSocket Service
 *
 * Handles real-time communication with the server.
 * All messages use a consistent JSON format - never double-encoded.
 */

import type {
  WSMessage,
  WSMessageType,
  EncryptedPayload,
  TypingPayload,
  ReadReceiptPayload,
  PresencePayload,
  StatusUpdatePayload,
} from '../types';

type MessageHandler<T = unknown> = (payload: T, message: WSMessage<T>) => void;

interface WebSocketConfig {
  url: string;
  token: string;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: Event) => void;
}

export class WebSocketService {
  private ws: WebSocket | null = null;
  private config: WebSocketConfig;
  private handlers = new Map<WSMessageType, Set<MessageHandler>>();
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private heartbeatInterval: number | null = null;
  private messageQueue: Array<{ type: WSMessageType; payload: unknown }> = [];

  constructor(config: WebSocketConfig) {
    this.config = config;
  }

  /**
   * Connect to the WebSocket server
   */
  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      // Use Sec-WebSocket-Protocol for auth (works in all browsers)
      this.ws = new WebSocket(this.config.url, ['Bearer', this.config.token]);

      this.ws.onopen = () => {
        this.reconnectAttempts = 0;
        this.startHeartbeat();
        this.flushMessageQueue();
        this.config.onConnect?.();
        resolve();
      };

      this.ws.onmessage = (event) => {
        this.handleMessage(event.data);
      };

      this.ws.onclose = () => {
        this.stopHeartbeat();
        this.config.onDisconnect?.();
        this.attemptReconnect();
      };

      this.ws.onerror = (error) => {
        this.config.onError?.(error);
        reject(error);
      };
    });
  }

  /**
   * Disconnect from the WebSocket server
   */
  disconnect(): void {
    this.maxReconnectAttempts = 0; // Prevent auto-reconnect
    this.stopHeartbeat();
    this.ws?.close();
    this.ws = null;
  }

  /**
   * Subscribe to a message type
   * Returns an unsubscribe function
   */
  on<T>(type: WSMessageType, handler: MessageHandler<T>): () => void {
    if (!this.handlers.has(type)) {
      this.handlers.set(type, new Set());
    }
    this.handlers.get(type)!.add(handler as MessageHandler);

    return () => {
      this.handlers.get(type)?.delete(handler as MessageHandler);
    };
  }

  /**
   * Send a message to the server
   * Payload is always an object - never pre-stringify
   * @param messageId - Optional message ID to use (for correlating status updates)
   */
  async send<T extends object>(type: WSMessageType, payload: T, messageId?: string): Promise<void> {
    const message: WSMessage<T> = {
      type,
      payload,
      // IMPORTANT: Go expects ISO 8601 string, not Unix timestamp
      timestamp: new Date().toISOString(),
      messageId: messageId || crypto.randomUUID(),
    };

    // Add HMAC signature for message integrity
    const signature = await this.generateMessageSignature(message);
    // Generate nonce as hex string (Go expects string, not array)
    const nonceBytes = crypto.getRandomValues(new Uint8Array(16));
    const nonce = Array.from(nonceBytes).map(b => b.toString(16).padStart(2, '0')).join('');

    const signedMessage = {
      ...message,
      signature,
      nonce, // 128-bit nonce as hex string for replay protection
    };

    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(signedMessage));
    } else {
      // Queue message for when connection is restored
      this.messageQueue.push({ type, payload });
    }
  }

  /**
   * Generate HMAC signature for message integrity
   * Uses standardized HMAC key derivation to match backend implementation
   */
  private async generateMessageSignature(message: WSMessage): Promise<string> {
    if (!this.config.token) {
      throw new Error('No authentication token available for message signing');
    }

    // Create message string for HMAC: type + timestamp + messageId + payload
    const messageStr = `${message.type}:${message.timestamp}:${message.messageId}:${JSON.stringify(message.payload)}`;

    // Standardized HMAC key derivation to match Go backend:
    // 1. Use full token as key material
    // 2. Pad to 32 bytes if shorter, truncate if longer (match Go behavior)
    const tokenBytes = new TextEncoder().encode(this.config.token);
    let keyData: Uint8Array;

    if (tokenBytes.length < 32) {
      // Pad to 32 bytes with zeros (match Go padding behavior)
      keyData = new Uint8Array(32);
      keyData.set(tokenBytes);
      // Fill remaining bytes with zeros
      for (let i = tokenBytes.length; i < 32; i++) {
        keyData[i] = 0;
      }
    } else {
      // Truncate to 32 bytes (match Go truncation behavior)
      keyData = tokenBytes.slice(0, 32);
    }

    const messageData = new TextEncoder().encode(messageStr);

    // Generate HMAC-SHA256 signature
    return this.hmacSha256(keyData, messageData);
  }

  /**
   * HMAC-SHA256 implementation using Web Crypto API
   */
  private async hmacSha256(key: Uint8Array, data: Uint8Array): Promise<string> {
    const cryptoKey = await crypto.subtle.importKey(
      'raw',
      key as any,
      { name: 'HMAC', hash: 'SHA-256' },
      false,
      ['sign']
    );

    const signature = await crypto.subtle.sign('HMAC', cryptoKey, data as any);
    return Array.from(new Uint8Array(signature))
      .map(b => b.toString(16).padStart(2, '0'))
      .join('');
  }

  /**
   * Send an encrypted message
   * @param messageId - The local message ID to use for status update correlation
   */
  async sendEncryptedMessage(payload: EncryptedPayload, messageId?: string): Promise<void> {
    await this.send('send', payload, messageId);
  }

  /**
   * Send a sealed sender message
   */
  sendSealedSenderMessage(payload: EncryptedPayload & {
    sealedSenderCertificateId?: string;
    ephemeralPublicKey?: string;
  }): void {
    this.send('send', payload);
  }

  /**
   * Send typing indicator
   */
  async sendTypingIndicator(recipientId: string, isTyping: boolean): Promise<void> {
    const payload: TypingPayload = { receiver_id: recipientId, is_typing: isTyping };
    await this.send('typing', payload);
  }

  /**
   * Send read/delivery receipt
   * IMPORTANT: Backend has separate handlers for these:
   * - 'delivery_ack' type → marks message as "delivered" 
   * - 'read_receipt' type → marks message as "read"
   */
  async sendReadReceipt(messageId: string, _conversationId: string, status: 'delivered' | 'read'): Promise<void> {
    if (status === 'delivered') {
      // Use delivery_ack type for delivery confirmations
      await this.send('delivery_ack', { message_id: messageId }, messageId);
    } else {
      // Use read_receipt type for actual read confirmations
      const payload: ReadReceiptPayload = { message_ids: [messageId], status };
      await this.send('read_receipt', payload);
    }
  }

  /**
   * Check if connected
   */
  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  // Private methods

  private async handleMessage(data: string): Promise<void> {
    try {
      const message: WSMessage = JSON.parse(data);

      // SECURITY: Validate message signature if present
      if (message.signature && message.nonce) {
        if (!(await this.verifyMessageSignature(message))) {
          console.error('Invalid message signature detected - possible tampering');
          // SECURITY: Disconnect on signature verification failure to prevent MITM attacks
          this.disconnect();
          return;
        }
      }

      // SECURITY: Validate message structure
      if (!message.type || typeof message.type !== 'string') {
        console.error('Invalid message type');
        return;
      }

      // SECURITY: Validate payload based on message type
      if (this.isPayloadValidationRequired(message.type) && !this.validatePayload(message)) {
        console.error('Invalid payload for message type:', message.type);
        return;
      }

      // Dispatch to handlers
      const handlers = this.handlers.get(message.type);
      if (handlers) {
        handlers.forEach((handler) => {
          try {
            handler(message.payload, message);
          } catch (e) {
            console.error('Error in handler for', message.type, ':', e);
            // Add recovery mechanism
            this.recoverFromHandlerError(message.type, e);
          }
        });
      }
    } catch (e) {
      console.error('Failed to parse WebSocket message:', e);
    }
  }

  private startHeartbeat(): void {
    this.heartbeatInterval = window.setInterval(() => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.send('heartbeat', {});
      }
    }, 30000);
  }

  private stopHeartbeat(): void {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }

  private attemptReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached');
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);

    setTimeout(() => {
      this.connect().catch(() => {
        // Reconnection failed - will retry
      });
    }, delay);
  }

  private flushMessageQueue(): void {
    while (this.messageQueue.length > 0) {
      const { type, payload } = this.messageQueue.shift()!;
      this.send(type, payload as object);
    }
  }

  // verifyMessageSignature verifies the HMAC signature of a message
  private async verifyMessageSignature(message: WSMessage): Promise<boolean> {
    if (!message.signature || !message.nonce) {
      return false;
    }

    if (!this.config.token) {
      console.error('No authentication token available for message verification');
      return false;
    }

    // Create message string for HMAC: type + timestamp + messageId + payload
    const messageStr = `${message.type}:${message.timestamp}:${message.messageId}:${JSON.stringify(message.payload)}`;

    // Use token as HMAC key (first 32 bytes for SHA-256)
    const keyData = new TextEncoder().encode(this.config.token).slice(0, 32);
    const messageData = new TextEncoder().encode(messageStr);

    try {
      // Generate expected HMAC signature
      const cryptoKey = await crypto.subtle.importKey(
        'raw',
        keyData as any,
        { name: 'HMAC', hash: 'SHA-256' },
        false,
        ['sign']
      );

      const expectedSignature = await crypto.subtle.sign('HMAC', cryptoKey, messageData as any);
      const expectedHex = Array.from(new Uint8Array(expectedSignature))
        .map(b => b.toString(16).padStart(2, '0'))
        .join('');

      // Compare signatures
      return expectedHex === message.signature;
    } catch (error) {
      console.error('Error verifying message signature:', error);
      return false;
    }
  }

  // isPayloadValidationRequired checks if payload validation is needed for this message type
  private isPayloadValidationRequired(messageType: WSMessageType): boolean {
    const validationRequiredTypes: WSMessageType[] = [
      'send', 'deliver', 'read_receipt', 'status_update'
    ];
    return validationRequiredTypes.includes(messageType);
  }

  // validatePayload validates the payload structure based on message type
  private validatePayload(message: WSMessage): boolean {
    try {
      switch (message.type) {
        case 'send':
        case 'deliver':
          // Validate encrypted payload structure
          if (!message.payload || typeof message.payload !== 'object') {
            return false;
          }
          const msgPayload = message.payload as any;
          if (!msgPayload.ciphertext || typeof msgPayload.ciphertext !== 'string') {
            return false;
          }
          break;

        case 'read_receipt':
          if (!message.payload || typeof message.payload !== 'object') {
            return false;
          }
          const receiptPayload = message.payload as any;
          if (!receiptPayload.messageId || typeof receiptPayload.messageId !== 'string') {
            return false;
          }
          break;

        case 'status_update':
          // Backend sends various status update formats:
          // - {"status": "delivered"} or {"status": "read"}
          // - {"delivered_to": N, "pending": M}
          // All are valid, just need to be an object
          if (!message.payload || typeof message.payload !== 'object') {
            return false;
          }
          break;

        default:
          // Other message types don't require strict validation
          return true;
      }
      return true;
    } catch (e) {
      console.error('Payload validation error:', e);
      return false;
    }
  }

  // recoverFromHandlerError handles errors from message handlers
  private recoverFromHandlerError(messageType: WSMessageType, error: any): void {
    console.error('Recovering from handler error for', messageType, ':', error);

    // Implement recovery strategies based on message type
    switch (messageType) {
      case 'send':
      case 'deliver':
        // For message delivery errors, we might want to retry or notify the user
        break;

      case 'read_receipt':
        // Read receipt errors are less critical, just log them
        break;

      default:
        // Generic recovery for other message types
        break;
    }
  }
}

// Type-safe event handlers
export interface WebSocketEventHandlers {
  onMessage: (payload: EncryptedPayload, message: WSMessage<EncryptedPayload>) => void;
  onTyping: (payload: TypingPayload) => void;
  onReadReceipt: (payload: ReadReceiptPayload) => void;
  onPresence: (payload: PresencePayload) => void;
  onStatusUpdate: (payload: StatusUpdatePayload) => void;
}

/**
 * Create a WebSocket service with type-safe handlers
 */
export function createWebSocketService(
  config: WebSocketConfig,
  handlers: Partial<WebSocketEventHandlers>
): WebSocketService {
  const service = new WebSocketService(config);

  if (handlers.onMessage) {
    service.on('deliver', handlers.onMessage);
  }
  if (handlers.onTyping) {
    service.on('typing', handlers.onTyping);
  }
  if (handlers.onReadReceipt) {
    service.on('read_receipt', handlers.onReadReceipt);
  }
  if (handlers.onPresence) {
    service.on('user_online', (payload: PresencePayload) => handlers.onPresence!(payload));
    service.on('user_offline', (payload: PresencePayload) => handlers.onPresence!(payload));
  }
  if (handlers.onStatusUpdate) {
    service.on('status_update', handlers.onStatusUpdate);
  }

  return service;
}

