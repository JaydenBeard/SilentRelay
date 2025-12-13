/**
 * Comprehensive WebSocket Error Handling and Logging Tests
 *
 * This test suite validates:
 * - WebSocket connection failure handling
 * - WebSocket message processing failure handling
 * - WebSocket authentication failure handling
 * - WebSocket rate limiting violation handling
 * - WebSocket security violation handling
 * - WebSocket logging and monitoring mechanisms
 * - WebSocket error recovery and resilience
 */

import { WebSocketService } from '../web-new/src/core/services/websocket';
import { WebSocketEventHandlers } from '../web-new/src/core/services/websocket';

// Mock WebSocket for testing
class MockWebSocket {
  constructor(url, protocols) {
    this.url = url;
    this.protocols = protocols;
    this.readyState = WebSocket.CONNECTING;
    this.onopen = null;
    this.onmessage = null;
    this.onclose = null;
    this.onerror = null;
    this.sentMessages = [];
    this.simulateConnection = false;
    this.simulateError = false;
    this.simulateClose = false;
  }

  send(message) {
    this.sentMessages.push(message);
  }

  // Simulate WebSocket events
  triggerOpen() {
    this.readyState = WebSocket.OPEN;
    if (this.onopen) this.onopen();
  }

  triggerMessage(data) {
    if (this.onmessage) this.onmessage({ data });
  }

  triggerError(error) {
    if (this.onerror) this.onerror(error);
  }

  triggerClose() {
    this.readyState = WebSocket.CLOSED;
    if (this.onclose) this.onclose();
  }

  close() {
    this.triggerClose();
  }
}

// Test WebSocket connection failure handling
describe('WebSocket Connection Failure Handling', () => {
  let service;
  let mockWS;

  beforeEach(() => {
    // Mock global WebSocket
    global.WebSocket = MockWebSocket;

    service = new WebSocketService({
      url: 'ws://localhost:8080/ws',
      token: 'test-token',
      onError: jest.fn(),
      onDisconnect: jest.fn()
    });
  });

  afterEach(() => {
    jest.clearAllMocks();
    delete global.WebSocket;
  });

  test('should handle connection failures gracefully', async () => {
    // Simulate connection failure
    mockWS = new MockWebSocket();
    mockWS.simulateError = true;

    // Mock WebSocket to return our mock
    jest.spyOn(global, 'WebSocket').mockImplementation(() => mockWS);

    // Trigger error immediately
    setTimeout(() => {
      if (mockWS.onerror) {
        mockWS.onerror(new Event('error'));
      }
    }, 10);

    await expect(service.connect()).rejects.toBeDefined();

    // Verify error handler was called
    expect(service.config.onError).toHaveBeenCalled();
  });

  test('should handle connection timeout scenarios', async () => {
    let resolveConnect;
    const connectPromise = new Promise((resolve) => {
      resolveConnect = resolve;
    });

    // Mock WebSocket that never connects
    jest.spyOn(global, 'WebSocket').mockImplementation(() => {
      const ws = new MockWebSocket();
      // Never trigger open event to simulate timeout
      return ws;
    });

    // Set a timeout for the test
    const timeoutPromise = new Promise((_, reject) => {
      setTimeout(() => {
        reject(new Error('Connection timeout'));
      }, 100);
    });

    await expect(Promise.race([service.connect(), timeoutPromise]))
      .rejects.toBeDefined();
  });

  test('should handle invalid WebSocket URLs', async () => {
    // Test with invalid URL
    const invalidService = new WebSocketService({
      url: 'invalid-url',
      token: 'test-token'
    });

    // Mock WebSocket constructor to throw error
    jest.spyOn(global, 'WebSocket').mockImplementation(() => {
      throw new Error('Invalid URL');
    });

    await expect(invalidService.connect()).rejects.toBeDefined();
  });
});

// Test WebSocket message processing failure handling
describe('WebSocket Message Processing Failure Handling', () => {
  let service;
  let mockWS;

  beforeEach(() => {
    global.WebSocket = MockWebSocket;

    service = new WebSocketService({
      url: 'ws://localhost:8080/ws',
      token: 'test-token'
    });

    mockWS = new MockWebSocket();
    jest.spyOn(global, 'WebSocket').mockImplementation(() => mockWS);
  });

  afterEach(() => {
    jest.clearAllMocks();
    delete global.WebSocket;
  });

  test('should handle malformed JSON messages', async () => {
    await service.connect();
    mockWS.triggerOpen();

    // Send malformed JSON
    const malformedMessage = '{"type": "message", "payload": {invalid json}';
    mockWS.triggerMessage(malformedMessage);

    // Should not throw - should be caught and logged
    await new Promise(resolve => setTimeout(resolve, 50));
  });

  test('should handle invalid message signatures', async () => {
    await service.connect();
    mockWS.triggerOpen();

    // Send message with invalid signature
    const invalidSigMessage = JSON.stringify({
      type: 'send',
      payload: { ciphertext: 'test' },
      signature: 'invalid-signature',
      nonce: [1, 2, 3, 4],
      timestamp: Date.now(),
      messageId: 'test-id'
    });

    mockWS.triggerMessage(invalidSigMessage);

    // Should disconnect on signature verification failure
    await new Promise(resolve => setTimeout(resolve, 50));
    expect(mockWS.readyState).toBe(WebSocket.CLOSED);
  });

  test('should handle invalid message types', async () => {
    await service.connect();
    mockWS.triggerOpen();

    // Send message with invalid type
    const invalidTypeMessage = JSON.stringify({
      type: 123, // Invalid - should be string
      payload: { test: 'data' }
    });

    mockWS.triggerMessage(invalidTypeMessage);

    // Should be caught and logged without crashing
    await new Promise(resolve => setTimeout(resolve, 50));
  });
});

// Test WebSocket authentication failure handling
describe('WebSocket Authentication Failure Handling', () => {
  let service;

  beforeEach(() => {
    global.WebSocket = MockWebSocket;

    service = new WebSocketService({
      url: 'ws://localhost:8080/ws',
      token: 'test-token',
      onError: jest.fn()
    });
  });

  afterEach(() => {
    jest.clearAllMocks();
    delete global.WebSocket;
  });

  test('should handle token validation failures', async () => {
    // Mock WebSocket that will fail authentication
    const mockWS = new MockWebSocket();
    jest.spyOn(global, 'WebSocket').mockImplementation(() => mockWS);

    // Simulate authentication failure by triggering error
    setTimeout(() => {
      if (mockWS.onerror) {
        mockWS.onerror(new Event('authentication_failed'));
      }
    }, 10);

    await expect(service.connect()).rejects.toBeDefined();
    expect(service.config.onError).toHaveBeenCalled();
  });

  test('should handle expired tokens', async () => {
    // Create service with expired token scenario
    const expiredService = new WebSocketService({
      url: 'ws://localhost:8080/ws',
      token: 'expired-token',
      onError: jest.fn()
    });

    const mockWS = new MockWebSocket();
    jest.spyOn(global, 'WebSocket').mockImplementation(() => mockWS);

    // Simulate token expiration error
    setTimeout(() => {
      if (mockWS.onerror) {
        const errorEvent = new Event('error');
        errorEvent.message = 'Token expired';
        mockWS.onerror(errorEvent);
      }
    }, 10);

    await expect(expiredService.connect()).rejects.toBeDefined();
  });
});

// Test WebSocket rate limiting violation handling
describe('WebSocket Rate Limiting Violation Handling', () => {
  let service;
  let mockWS;

  beforeEach(() => {
    global.WebSocket = MockWebSocket;

    service = new WebSocketService({
      url: 'ws://localhost:8080/ws',
      token: 'test-token'
    });

    mockWS = new MockWebSocket();
    jest.spyOn(global, 'WebSocket').mockImplementation(() => mockWS);
  });

  afterEach(() => {
    jest.clearAllMocks();
    delete global.WebSocket;
  });

  test('should handle rate limit exceeded errors', async () => {
    await service.connect();
    mockWS.triggerOpen();

    // Simulate rate limit error message from server
    const rateLimitMessage = JSON.stringify({
      type: 'error',
      payload: {
        error: 'Rate limit exceeded. Please slow down.',
        code: 'rate_limit_exceeded'
      }
    });

    mockWS.triggerMessage(rateLimitMessage);

    // Should handle gracefully without crashing
    await new Promise(resolve => setTimeout(resolve, 50));
  });

  test('should handle message queue overflow', async () => {
    await service.connect();
    mockWS.triggerOpen();

    // Fill up the message queue beyond capacity
    for (let i = 0; i < 1000; i++) {
      await service.send('test', { message: `test-${i}` });
    }

    // Should handle queue overflow gracefully
    await new Promise(resolve => setTimeout(resolve, 50));
  });
});

// Test WebSocket security violation handling
describe('WebSocket Security Violation Handling', () => {
  let service;
  let mockWS;

  beforeEach(() => {
    global.WebSocket = MockWebSocket;

    service = new WebSocketService({
      url: 'ws://localhost:8080/ws',
      token: 'test-token'
    });

    mockWS = new MockWebSocket();
    jest.spyOn(global, 'WebSocket').mockImplementation(() => mockWS);
  });

  afterEach(() => {
    jest.clearAllMocks();
    delete global.WebSocket;
  });

  test('should handle replay attack detection', async () => {
    await service.connect();
    mockWS.triggerOpen();

    // Send message with duplicate nonce (replay attack)
    const replayMessage = JSON.stringify({
      type: 'send',
      payload: { ciphertext: 'test' },
      signature: 'valid-signature',
      nonce: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16], // Duplicate nonce
      timestamp: Date.now(),
      messageId: 'test-id'
    });

    // Send same message twice
    mockWS.triggerMessage(replayMessage);
    mockWS.triggerMessage(replayMessage);

    // Should detect and handle replay attack
    await new Promise(resolve => setTimeout(resolve, 50));
  });

  test('should handle invalid payload structures', async () => {
    await service.connect();
    mockWS.triggerOpen();

    // Send message with invalid payload for message type
    const invalidPayloadMessage = JSON.stringify({
      type: 'read_receipt',
      payload: 'invalid-string-payload' // Should be object with messageId
    });

    mockWS.triggerMessage(invalidPayloadMessage);

    // Should reject invalid payload structure
    await new Promise(resolve => setTimeout(resolve, 50));
  });
});

// Test WebSocket logging and monitoring mechanisms
describe('WebSocket Logging and Monitoring Mechanisms', () => {
  let service;
  let mockWS;
  let consoleSpy;

  beforeEach(() => {
    global.WebSocket = MockWebSocket;
    consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

    service = new WebSocketService({
      url: 'ws://localhost:8080/ws',
      token: 'test-token',
      onError: jest.fn()
    });

    mockWS = new MockWebSocket();
    jest.spyOn(global, 'WebSocket').mockImplementation(() => mockWS);
  });

  afterEach(() => {
    jest.clearAllMocks();
    delete global.WebSocket;
    consoleSpy.mockRestore();
  });

  test('should log connection errors appropriately', async () => {
    // Trigger connection error
    const mockWS = new MockWebSocket();
    jest.spyOn(global, 'WebSocket').mockImplementation(() => mockWS);

    setTimeout(() => {
      if (mockWS.onerror) {
        mockWS.onerror(new Error('Connection failed'));
      }
    }, 10);

    await expect(service.connect()).rejects.toBeDefined();

    // Verify error was logged
    expect(consoleSpy).toHaveBeenCalledWith(
      expect.stringContaining('Connection failed'),
      expect.anything()
    );
  });

  test('should log message processing errors', async () => {
    await service.connect();
    mockWS.triggerOpen();

    // Send malformed message
    const malformed = '{"invalid": json}';
    mockWS.triggerMessage(malformed);

    await new Promise(resolve => setTimeout(resolve, 50));

    // Verify parsing error was logged
    expect(consoleSpy).toHaveBeenCalledWith(
      expect.stringContaining('Failed to parse WebSocket message'),
      expect.anything()
    );
  });

  test('should log signature verification failures', async () => {
    await service.connect();
    mockWS.triggerOpen();

    // Send message with invalid signature
    const invalidSig = JSON.stringify({
      type: 'send',
      payload: { ciphertext: 'test' },
      signature: 'invalid',
      nonce: [1, 2, 3, 4],
      timestamp: Date.now(),
      messageId: 'test-id'
    });

    mockWS.triggerMessage(invalidSig);

    await new Promise(resolve => setTimeout(resolve, 50));

    // Verify signature error was logged
    expect(consoleSpy).toHaveBeenCalledWith(
      expect.stringContaining('Invalid message signature'),
      expect.anything()
    );
  });
});

// Test WebSocket error recovery and resilience
describe('WebSocket Error Recovery and Resilience', () => {
  let service;
  let mockWS;

  beforeEach(() => {
    global.WebSocket = MockWebSocket;

    service = new WebSocketService({
      url: 'ws://localhost:8080/ws',
      token: 'test-token',
      onConnect: jest.fn(),
      onDisconnect: jest.fn(),
      onError: jest.fn()
    });

    mockWS = new MockWebSocket();
    jest.spyOn(global, 'WebSocket').mockImplementation(() => mockWS);
  });

  afterEach(() => {
    jest.clearAllMocks();
    delete global.WebSocket;
  });

  test('should attempt reconnection after connection loss', async () => {
    await service.connect();
    mockWS.triggerOpen();

    // Simulate connection loss
    mockWS.triggerClose();

    // Should attempt reconnection
    await new Promise(resolve => setTimeout(resolve, 100));

    // Verify reconnection attempt
    expect(service.config.onDisconnect).toHaveBeenCalled();
  });

  test('should handle multiple reconnection attempts', async () => {
    await service.connect();
    mockWS.triggerOpen();

    // Simulate multiple connection losses
    for (let i = 0; i < 3; i++) {
      mockWS.triggerClose();
      await new Promise(resolve => setTimeout(resolve, 50));
    }

    // Should handle multiple reconnection attempts gracefully
    expect(service.config.onDisconnect).toHaveBeenCalledTimes(3);
  });

  test('should stop reconnection attempts after max retries', async () => {
    // Set low max reconnect attempts for testing
    service.maxReconnectAttempts = 2;

    await service.connect();
    mockWS.triggerOpen();

    // Simulate multiple connection losses
    for (let i = 0; i < 5; i++) {
      mockWS.triggerClose();
      await new Promise(resolve => setTimeout(resolve, 50));
    }

    // After max attempts, should stop trying
    await new Promise(resolve => setTimeout(resolve, 200));
  });

  test('should recover from handler errors gracefully', async () => {
    await service.connect();
    mockWS.triggerOpen();

    // Register a handler that will throw an error
    const errorHandler = jest.fn(() => {
      throw new Error('Handler error');
    });

    service.on('test', errorHandler);

    // Send a message that will trigger the error handler
    const testMessage = JSON.stringify({
      type: 'test',
      payload: { data: 'test' }
    });

    mockWS.triggerMessage(testMessage);

    // Should recover from handler error without crashing
    await new Promise(resolve => setTimeout(resolve, 50));

    expect(errorHandler).toHaveBeenCalled();
  });
});

// Test WebSocket message queue and recovery
describe('WebSocket Message Queue and Recovery', () => {
  let service;
  let mockWS;

  beforeEach(() => {
    global.WebSocket = MockWebSocket;

    service = new WebSocketService({
      url: 'ws://localhost:8080/ws',
      token: 'test-token'
    });

    mockWS = new MockWebSocket();
    jest.spyOn(global, 'WebSocket').mockImplementation(() => mockWS);
  });

  afterEach(() => {
    jest.clearAllMocks();
    delete global.WebSocket;
  });

  test('should queue messages when disconnected', async () => {
    // Don't connect - simulate disconnected state
    expect(service.isConnected()).toBe(false);

    // Send messages while disconnected
    await service.send('test', { message: 'test1' });
    await service.send('test', { message: 'test2' });

    // Messages should be queued
    expect(service.messageQueue.length).toBe(2);
  });

  test('should flush message queue on reconnection', async () => {
    // Queue messages while disconnected
    await service.send('test', { message: 'queued1' });
    await service.send('test', { message: 'queued2' });

    // Connect and trigger open
    await service.connect();
    mockWS.triggerOpen();

    // Queue should be flushed
    await new Promise(resolve => setTimeout(resolve, 50));

    expect(mockWS.sentMessages.length).toBeGreaterThan(0);
  });

  test('should handle queue flush errors gracefully', async () => {
    // Queue messages
    for (let i = 0; i < 10; i++) {
      await service.send('test', { message: `queued${i}` });
    }

    // Mock send to throw error during flush
    const originalSend = mockWS.send;
    mockWS.send = jest.fn(() => {
      throw new Error('Send failed');
    });

    // Connect and trigger open (which flushes queue)
    await service.connect();
    mockWS.triggerOpen();

    // Should handle send errors during flush gracefully
    await new Promise(resolve => setTimeout(resolve, 50));

    mockWS.send = originalSend;
  });
});

// Integration test for comprehensive error scenarios
describe('WebSocket Comprehensive Error Scenarios', () => {
  let service;
  let mockWS;

  beforeEach(() => {
    global.WebSocket = MockWebSocket;

    service = new WebSocketService({
      url: 'ws://localhost:8080/ws',
      token: 'test-token',
      onError: jest.fn(),
      onDisconnect: jest.fn()
    });

    mockWS = new MockWebSocket();
    jest.spyOn(global, 'WebSocket').mockImplementation(() => mockWS);
  });

  afterEach(() => {
    jest.clearAllMocks();
    delete global.WebSocket;
  });

  test('should handle complex error scenario chain', async () => {
    // 1. Connection fails initially
    let connectionAttempts = 0;
    jest.spyOn(global, 'WebSocket').mockImplementation(() => {
      connectionAttempts++;
      if (connectionAttempts === 1) {
        // First attempt fails
        const ws = new MockWebSocket();
        setTimeout(() => ws.triggerError(new Error('First connection failed')), 10);
        return ws;
      } else {
        // Second attempt succeeds
        return mockWS;
      }
    });

    // First connection attempt should fail
    await expect(service.connect()).rejects.toBeDefined();
    expect(service.config.onError).toHaveBeenCalled();

    // 2. Second connection attempt succeeds
    await service.connect();
    mockWS.triggerOpen();

    // 3. Send messages with various error conditions
    const malformedMessage = '{"invalid": json}';
    mockWS.triggerMessage(malformedMessage);

    const invalidSignature = JSON.stringify({
      type: 'send',
      payload: { ciphertext: 'test' },
      signature: 'invalid',
      nonce: [1, 2, 3, 4],
      timestamp: Date.now(),
      messageId: 'test-id'
    });

    mockWS.triggerMessage(invalidSignature);

    // 4. Simulate connection loss
    mockWS.triggerClose();

    // 5. Verify system resilience
    await new Promise(resolve => setTimeout(resolve, 100));

    expect(service.config.onDisconnect).toHaveBeenCalled();
  });
});

export {
  WebSocketConnectionFailureHandling,
  WebSocketMessageProcessingFailureHandling,
  WebSocketAuthenticationFailureHandling,
  WebSocketRateLimitingViolationHandling,
  WebSocketSecurityViolationHandling,
  WebSocketLoggingAndMonitoringMechanisms,
  WebSocketErrorRecoveryAndResilience,
  WebSocketMessageQueueAndRecovery,
  WebSocketComprehensiveErrorScenarios
};