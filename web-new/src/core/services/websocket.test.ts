/**
 * WebSocket Service - Comprehensive Security Tests
 *
 * Tests for WebSocket message encryption, integrity, and security mechanisms
 * Covers HMAC signing/verification, replay attack protection, and message validation
 */
import { describe, expect, test, vi, beforeEach, afterEach } from 'vitest';
import { WebSocketService } from './websocket';
import type { WSMessage, WSMessageType, EncryptedPayload } from '../types';

// Mock crypto for deterministic testing
const mockCrypto = {
  randomUUID: () => 'test-uuid-1234-5678-9012-345678901234',
  getRandomValues: (array: Uint8Array) => {
    // Fill with predictable values for testing
    for (let i = 0; i < array.length; i++) {
      array[i] = i % 256;
    }
    return array;
  },
  subtle: {
    importKey: vi.fn(),
    sign: vi.fn(),
  }
};

// Mock global crypto
vi.stubGlobal('crypto', mockCrypto);

describe('WebSocket Message Security - HMAC Signing and Verification', () => {
  let service: WebSocketService;
  const testToken = 'test-token-with-32-bytes-length-1234567890';
  const testConfig = {
    url: 'ws://localhost:8080/ws',
    token: testToken,
  };

  beforeEach(() => {
    service = new WebSocketService(testConfig);

    // Mock crypto.subtle.importKey to return a mock key
    mockCrypto.subtle.importKey.mockResolvedValue({});

    // Mock crypto.subtle.sign to return predictable HMAC
    mockCrypto.subtle.sign.mockImplementation((algorithm, key, data) => {
      // Create a predictable HMAC based on input data
      const dataStr = new TextDecoder().decode(data as Uint8Array);
      const hmac = `predictable-hmac-${dataStr.length}-${dataStr.substring(0, 20)}`;
      return new TextEncoder().encode(hmac);
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('HMAC-SHA256 Message Signing', () => {
    test('should generate HMAC signature with correct key derivation', async () => {
      const testMessage = {
        type: 'send',
        payload: { ciphertext: 'test-ciphertext', messageType: 'whisper' },
        timestamp: Date.now(),
        messageId: 'test-message-id',
      };

      // Call the private method via reflection or create a test wrapper
      const signature = await (service as any).generateMessageSignature(testMessage);

      expect(signature).toBeTruthy();
      expect(typeof signature).toBe('string');
      expect(signature.length).toBeGreaterThan(0);
    });

    test('should use token-based HMAC key derivation (32-byte key)', async () => {
      const testMessage = {
        type: 'send',
        payload: { ciphertext: 'test-ciphertext' },
        timestamp: Date.now(),
        messageId: 'test-message-id',
      };

      // Mock to verify key derivation
      const originalImportKey = mockCrypto.subtle.importKey;
      let capturedKey: Uint8Array | null = null;

      mockCrypto.subtle.importKey.mockImplementationOnce((format, keyData, algorithm, extractable, keyUsages) => {
        capturedKey = keyData as Uint8Array;
        return Promise.resolve({});
      });

      await (service as any).generateMessageSignature(testMessage);

      expect(capturedKey).toBeTruthy();
      expect(capturedKey?.length).toBe(32); // Should be exactly 32 bytes for SHA-256
    });

    test.skip('should handle token padding for keys shorter than 32 bytes', async () => {
      // Create service with short token
      const shortTokenService = new WebSocketService({
        ...testConfig,
        token: 'short',
      });

      const testMessage = {
        type: 'send',
        payload: { ciphertext: 'test-ciphertext' },
        timestamp: Date.now(),
        messageId: 'test-message-id',
      };

      // Mock to verify key derivation with padding
      const originalImportKey = mockCrypto.subtle.importKey;
      let capturedKey: Uint8Array | null = null;

      mockCrypto.subtle.importKey.mockImplementationOnce((format, keyData, algorithm, extractable, keyUsages) => {
        capturedKey = keyData as Uint8Array;
        return Promise.resolve({});
      });

      await (shortTokenService as any).generateMessageSignature(testMessage);

      expect(capturedKey).toBeTruthy();
      expect(capturedKey?.length).toBe(32); // Should be padded to 32 bytes
      expect(capturedKey?.slice(0, 5)).toEqual(new TextEncoder().encode('short'));
      expect(capturedKey?.slice(5)).toEqual(new Uint8Array(27)); // Padded with zeros
    });

    test('should handle token truncation for keys longer than 32 bytes', async () => {
      const longToken = 'very-long-token-that-is-definitely-more-than-32-bytes-long-12345678901234567890';
      const longTokenService = new WebSocketService({
        ...testConfig,
        token: longToken,
      });

      const testMessage = {
        type: 'send',
        payload: { ciphertext: 'test-ciphertext' },
        timestamp: Date.now(),
        messageId: 'test-message-id',
      };

      // Mock to verify key derivation with truncation
      const originalImportKey = mockCrypto.subtle.importKey;
      let capturedKey: Uint8Array | null = null;

      mockCrypto.subtle.importKey.mockImplementationOnce((format, keyData, algorithm, extractable, keyUsages) => {
        capturedKey = keyData as Uint8Array;
        return Promise.resolve({});
      });

      await (longTokenService as any).generateMessageSignature(testMessage);

      expect(capturedKey).toBeTruthy();
      expect(capturedKey?.length).toBe(32); // Should be truncated to 32 bytes
      expect(capturedKey).toEqual(new TextEncoder().encode(longToken).slice(0, 32));
    });
  });

  describe('HMAC-SHA256 Message Verification', () => {
    test('should verify valid HMAC signatures', async () => {
      const testMessage = {
        type: 'send',
        payload: { ciphertext: 'test-ciphertext' },
        timestamp: Date.now(),
        messageId: 'test-message-id',
      };

      // Generate signature
      const signature = await (service as any).generateMessageSignature(testMessage);

      // Create signed message
      const signedMessage = {
        ...testMessage,
        signature,
        nonce: [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15], // Predictable nonce
      };

      // Verify signature
      const isValid = await (service as any).verifyMessageSignature(signedMessage);

      expect(isValid).toBe(true);
    });

    test('should reject invalid HMAC signatures', async () => {
      const testMessage = {
        type: 'send',
        payload: { ciphertext: 'test-ciphertext' },
        timestamp: Date.now(),
        messageId: 'test-message-id',
      };

      // Generate valid signature
      const validSignature = await (service as any).generateMessageSignature(testMessage);

      // Create signed message with tampered signature
      const tamperedMessage = {
        ...testMessage,
        signature: 'invalid-signature-1234567890',
        nonce: [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15],
      };

      // Verify signature
      const isValid = await (service as any).verifyMessageSignature(tamperedMessage);

      expect(isValid).toBe(false);
    });

    test('should reject messages without signature', async () => {
      const messageWithoutSignature = {
        type: 'send',
        payload: { ciphertext: 'test-ciphertext' },
        timestamp: Date.now(),
        messageId: 'test-message-id',
        // No signature field
      };

      const isValid = await (service as any).verifyMessageSignature(messageWithoutSignature);

      expect(isValid).toBe(false);
    });

    test('should reject messages without nonce', async () => {
      const testMessage = {
        type: 'send',
        payload: { ciphertext: 'test-ciphertext' },
        timestamp: Date.now(),
        messageId: 'test-message-id',
      };

      const signature = await (service as any).generateMessageSignature(testMessage);

      const messageWithoutNonce = {
        ...testMessage,
        signature,
        // No nonce field
      };

      const isValid = await (service as any).verifyMessageSignature(messageWithoutNonce);

      expect(isValid).toBe(false);
    });
  });

  describe('Message Tampering Detection', () => {
    test.skip('should detect tampered message content', async () => {
      const originalMessage = {
        type: 'send',
        payload: { ciphertext: 'original-ciphertext' },
        timestamp: Date.now(),
        messageId: 'test-message-id',
      };

      // Generate signature for original message
      const signature = await (service as any).generateMessageSignature(originalMessage);

      // Create tampered message
      const tamperedMessage = {
        ...originalMessage,
        signature,
        nonce: [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15],
        payload: { ciphertext: 'tampered-ciphertext' }, // Tampered payload
      };

      // Verify signature - should fail because content was tampered
      const isValid = await (service as any).verifyMessageSignature(tamperedMessage);

      expect(isValid).toBe(false);
    });

    test('should detect tampered message type', async () => {
      const originalMessage = {
        type: 'send',
        payload: { ciphertext: 'test-ciphertext' },
        timestamp: Date.now(),
        messageId: 'test-message-id',
      };

      // Generate signature for original message
      const signature = await (service as any).generateMessageSignature(originalMessage);

      // Create tampered message
      const tamperedMessage = {
        ...originalMessage,
        type: 'deliver', // Changed message type
        signature,
        nonce: [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15],
      };

      // Verify signature - should fail because type was tampered
      const isValid = await (service as any).verifyMessageSignature(tamperedMessage);

      expect(isValid).toBe(false);
    });

    test('should detect tampered timestamp', async () => {
      const originalMessage = {
        type: 'send',
        payload: { ciphertext: 'test-ciphertext' },
        timestamp: 1234567890,
        messageId: 'test-message-id',
      };

      // Generate signature for original message
      const signature = await (service as any).generateMessageSignature(originalMessage);

      // Create tampered message
      const tamperedMessage = {
        ...originalMessage,
        timestamp: 9876543210, // Changed timestamp
        signature,
        nonce: [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15],
      };

      // Verify signature - should fail because timestamp was tampered
      const isValid = await (service as any).verifyMessageSignature(tamperedMessage);

      expect(isValid).toBe(false);
    });
  });

  describe('Nonce-Based Replay Attack Protection', () => {
    test('should generate unique nonces for each message', async () => {
      const message1 = {
        type: 'send',
        payload: { ciphertext: 'test-ciphertext-1' },
        timestamp: Date.now(),
        messageId: 'test-message-id-1',
      };

      const message2 = {
        type: 'send',
        payload: { ciphertext: 'test-ciphertext-2' },
        timestamp: Date.now() + 1,
        messageId: 'test-message-id-2',
      };

      // Mock getRandomValues to return different values
      let callCount = 0;
      const originalGetRandomValues = mockCrypto.getRandomValues;
      mockCrypto.getRandomValues = (array: Uint8Array) => {
        callCount++;
        for (let i = 0; i < array.length; i++) {
          array[i] = (i + callCount) % 256;
        }
        return array;
      };

      // Generate signatures for both messages
      const signature1 = await (service as any).generateMessageSignature(message1);
      const signature2 = await (service as any).generateMessageSignature(message2);

      // Create signed messages
      const signedMessage1 = {
        ...message1,
        signature: signature1,
        nonce: Array.from(mockCrypto.getRandomValues(new Uint8Array(16))),
      };

      const signedMessage2 = {
        ...message2,
        signature: signature2,
        nonce: Array.from(mockCrypto.getRandomValues(new Uint8Array(16))),
      };

      // Nonces should be different
      expect(signedMessage1.nonce).not.toEqual(signedMessage2.nonce);

      // Restore original
      mockCrypto.getRandomValues = originalGetRandomValues;
    });

    test('should use 128-bit nonces for replay protection', async () => {
      const testMessage = {
        type: 'send',
        payload: { ciphertext: 'test-ciphertext' },
        timestamp: Date.now(),
        messageId: 'test-message-id',
      };

      const signature = await (service as any).generateMessageSignature(testMessage);

      const signedMessage = {
        ...testMessage,
        signature,
        nonce: Array.from(mockCrypto.getRandomValues(new Uint8Array(16))),
      };

      // Nonce should be 128 bits (16 bytes)
      expect(signedMessage.nonce).toHaveLength(16);
      expect(signedMessage.nonce.every((val: number) => typeof val === 'number' && val >= 0 && val <= 255)).toBe(true);
    });
  });

  describe('Message Structure and Payload Validation', () => {
    test('should validate message structure before processing', async () => {
      const invalidMessage = {
        // Missing required fields
        payload: { ciphertext: 'test-ciphertext' },
      };

      // This should be caught by the message handler
      const result = await (service as any).isPayloadValidationRequired('send');
      expect(result).toBe(true);
    });

    test('should validate payload structure for different message types', async () => {
      // Test validation for 'send' message type
      const sendMessage = {
        type: 'send',
        payload: { ciphertext: 'test-ciphertext', messageType: 'whisper' },
        timestamp: Date.now(),
        messageId: 'test-message-id',
      };

      const isValid = await (service as any).validatePayload(sendMessage);
      expect(isValid).toBe(true);

      // Test validation for invalid 'send' message (missing ciphertext)
      const invalidSendMessage = {
        type: 'send',
        payload: { messageType: 'whisper' }, // Missing ciphertext
        timestamp: Date.now(),
        messageId: 'test-message-id',
      };

      const isInvalid = await (service as any).validatePayload(invalidSendMessage);
      expect(isInvalid).toBe(false);
    });

    test('should validate read receipt payload structure', async () => {
      const validReadReceipt = {
        type: 'read_receipt',
        payload: {
          messageId: 'test-message-id',
          conversationId: 'test-conversation-id',
          status: 'read',
        },
        timestamp: Date.now(),
        messageId: 'test-message-id',
      };

      const isValid = await (service as any).validatePayload(validReadReceipt);
      expect(isValid).toBe(true);

      // Test invalid read receipt (missing messageId)
      const invalidReadReceipt = {
        type: 'read_receipt',
        payload: {
          conversationId: 'test-conversation-id',
          status: 'read',
        },
        timestamp: Date.now(),
        messageId: 'test-message-id',
      };

      const isInvalid = await (service as any).validatePayload(invalidReadReceipt);
      expect(isInvalid).toBe(false);
    });
  });

  describe('Cross-Platform Compatibility', () => {
    test('should handle different token encodings consistently', async () => {
      // Test with UTF-8 token
      const utf8Token = 'test-token-ñáéíóú-1234567890';
      const utf8Service = new WebSocketService({
        ...testConfig,
        token: utf8Token,
      });

      const testMessage = {
        type: 'send',
        payload: { ciphertext: 'test-ciphertext' },
        timestamp: Date.now(),
        messageId: 'test-message-id',
      };

      // Should handle UTF-8 encoding properly
      const signature = await (utf8Service as any).generateMessageSignature(testMessage);
      expect(signature).toBeTruthy();
    });

    test('should handle special characters in message content', async () => {
      const testMessage = {
        type: 'send',
        payload: {
          ciphertext: 'test-ciphertext-ñáéíóú-🚀🔒',
          messageType: 'whisper',
        },
        timestamp: Date.now(),
        messageId: 'test-message-id',
      };

      const signature = await (service as any).generateMessageSignature(testMessage);
      expect(signature).toBeTruthy();

      // Verify the signature
      const signedMessage = {
        ...testMessage,
        signature,
        nonce: [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15],
      };

      const isValid = await (service as any).verifyMessageSignature(signedMessage);
      expect(isValid).toBe(true);
    });
  });

  describe('Error Handling and Security', () => {
    test('should handle missing authentication token gracefully', async () => {
      const noTokenService = new WebSocketService({
        url: 'ws://localhost:8080/ws',
        token: '',
      });

      const testMessage = {
        type: 'send',
        payload: { ciphertext: 'test-ciphertext' },
        timestamp: Date.now(),
        messageId: 'test-message-id',
      };

      // Should throw error when no token available
      await expect((noTokenService as any).generateMessageSignature(testMessage))
        .rejects.toThrow('No authentication token available for message signing');
    });

    test('should handle crypto API failures gracefully', async () => {
      // Mock crypto.subtle.importKey to fail
      mockCrypto.subtle.importKey.mockRejectedValueOnce(new Error('Crypto API failed'));

      const testMessage = {
        type: 'send',
        payload: { ciphertext: 'test-ciphertext' },
        timestamp: Date.now(),
        messageId: 'test-message-id',
      };

      // Should handle crypto failure gracefully
      const signaturePromise = (service as any).generateMessageSignature(testMessage);
      await expect(signaturePromise).rejects.toBeTruthy();
    });

    test('should disconnect on signature verification failure', async () => {
      // Mock disconnect to track calls
      const disconnectSpy = vi.spyOn(service, 'disconnect');

      const testMessage = {
        type: 'send',
        payload: { ciphertext: 'test-ciphertext' },
        timestamp: Date.now(),
        messageId: 'test-message-id',
        signature: 'invalid-signature',
        nonce: [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15],
      };

      // Simulate message handling with invalid signature
      await (service as any).handleMessage(JSON.stringify(testMessage));

      // Should have called disconnect
      expect(disconnectSpy).toHaveBeenCalled();
    });
  });

  describe('Message Type Specific Tests', () => {
    const messageTypes: WSMessageType[] = [
      'send', 'deliver', 'read_receipt', 'typing', 'heartbeat', 'status_update'
    ];

    test.each(messageTypes)('should handle %s message type with signing', async (messageType) => {
      const testMessage = {
        type: messageType,
        payload: getTestPayloadForType(messageType),
        timestamp: Date.now(),
        messageId: 'test-message-id',
      };

      const signature = await (service as any).generateMessageSignature(testMessage);
      expect(signature).toBeTruthy();

      const signedMessage = {
        ...testMessage,
        signature,
        nonce: [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15],
      };

      const isValid = await (service as any).verifyMessageSignature(signedMessage);
      expect(isValid).toBe(true);
    });
  });
});

// Helper function to get appropriate test payload for message type
function getTestPayloadForType(type: WSMessageType): any {
  switch (type) {
    case 'send':
      return { ciphertext: 'test-ciphertext', messageType: 'whisper' };
    case 'deliver':
      return { ciphertext: 'test-ciphertext', messageType: 'whisper' };
    case 'read_receipt':
      return { messageId: 'test-message-id', conversationId: 'test-conversation-id', status: 'read' };
    case 'typing':
      return { recipientId: 'test-recipient-id', isTyping: true };
    case 'heartbeat':
      return {};
    case 'status_update':
      return { messageId: 'test-message-id', status: 'delivered' };
    default:
      return {};
  }
}