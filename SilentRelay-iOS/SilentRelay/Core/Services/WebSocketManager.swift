//
//  WebSocketManager.swift
//  SilentRelay
//
//  WebSocket service with HMAC message signing matching web-new/src/core/services/websocket.ts
//

import Foundation
import Observation
import CryptoKit

// MARK: - WebSocket Manager
@Observable
@MainActor
final class WebSocketManager: NSObject {
    // MARK: - Singleton
    static let shared = WebSocketManager()

    // MARK: - State
    private(set) var isConnected = false
    private(set) var connectionError: Error?

    // MARK: - Configuration
    private let baseURL: URL
    private var webSocketTask: URLSessionWebSocketTask?
    private var session: URLSession!

    // Reconnection
    private var reconnectAttempts = 0
    private let maxReconnectAttempts = 5
    private var reconnectDelay: TimeInterval = 1.0

    // Heartbeat
    private var heartbeatTimer: Timer?
    private let heartbeatInterval: TimeInterval = 30.0

    // Message queue for offline messages
    private var messageQueue: [(type: WSMessageType, payload: Encodable)] = []

    // Token for HMAC signing
    private var authToken: String?

    // Message handlers
    private var handlers: [WSMessageType: [(Any, WSMessageType) -> Void]] = [:]

    // MARK: - Init
    private override init() {
        let urlString = ProcessInfo.processInfo.environment["WS_URL"] ?? "wss://api.silentrelay.com/ws"
        self.baseURL = URL(string: urlString)!

        super.init()

        // Create URL session
        let configuration = URLSessionConfiguration.default
        configuration.timeoutIntervalForRequest = 60
        self.session = URLSession(configuration: configuration, delegate: self, delegateQueue: nil)
    }

    // MARK: - Connect
    func connect(token: String) async {
        self.authToken = token

        // Create WebSocket request with auth protocol
        var request = URLRequest(url: baseURL)
        request.timeoutInterval = 60

        // Use Sec-WebSocket-Protocol for auth (matches web implementation)
        // Format: "Bearer, <token>"
        request.setValue("Bearer, \(token)", forHTTPHeaderField: "Sec-WebSocket-Protocol")

        // Create WebSocket task
        webSocketTask = session.webSocketTask(with: request)
        webSocketTask?.resume()

        // Start receiving messages
        receiveMessage()

        // Reset reconnection state
        reconnectAttempts = 0
        reconnectDelay = 1.0

        // Start heartbeat
        startHeartbeat()

        // Flush message queue
        await flushMessageQueue()

        await MainActor.run {
            isConnected = true
            connectionError = nil
        }
    }

    // MARK: - Disconnect
    func disconnect() {
        stopHeartbeat()
        webSocketTask?.cancel(with: .normalClosure, reason: nil)
        webSocketTask = nil
        reconnectAttempts = maxReconnectAttempts // Prevent auto-reconnect

        Task { @MainActor in
            isConnected = false
        }
    }

    // MARK: - Send Message
    func send<T: Encodable>(_ type: WSMessageType, payload: T, messageId: String? = nil) async throws {
        guard isConnected, let task = webSocketTask else {
            // Queue message for later
            messageQueue.append((type, payload))
            return
        }

        // Create message
        let timestamp = ISO8601DateFormatter().string(from: Date())
        let msgId = messageId ?? UUID().uuidString

        // Encode payload to JSON
        let encoder = JSONEncoder()
        encoder.keyEncodingStrategy = .convertToSnakeCase
        let payloadData = try encoder.encode(payload)
        let payloadJSON = String(data: payloadData, encoding: .utf8) ?? "{}"

        // Generate HMAC signature
        let signature = try generateSignature(
            type: type,
            timestamp: timestamp,
            messageId: msgId,
            payloadJSON: payloadJSON
        )

        // Generate nonce (128-bit as hex string)
        let nonce = generateNonce()

        // Build message dictionary
        var messageDict: [String: Any] = [
            "type": type.rawValue,
            "payload": try JSONSerialization.jsonObject(with: payloadData),
            "timestamp": timestamp,
            "messageId": msgId,
            "signature": signature,
            "nonce": nonce
        ]

        // Serialize to JSON
        let messageData = try JSONSerialization.data(withJSONObject: messageDict)
        let messageString = String(data: messageData, encoding: .utf8) ?? ""

        // Send
        try await task.send(.string(messageString))
    }

    // MARK: - Generate HMAC Signature
    /// Matches web implementation exactly:
    /// - Format: "type:timestamp:messageId:payloadJSON"
    /// - Key: token padded/truncated to 32 bytes
    private func generateSignature(
        type: WSMessageType,
        timestamp: String,
        messageId: String,
        payloadJSON: String
    ) throws -> String {
        guard let token = authToken else {
            throw WebSocketError.notAuthenticated
        }

        // Create message string
        let messageStr = "\(type.rawValue):\(timestamp):\(messageId):\(payloadJSON)"

        // Pad/truncate token to 32 bytes (match Go backend)
        var keyData = Data(token.utf8)
        if keyData.count < 32 {
            // Pad with zeros
            keyData.append(Data(count: 32 - keyData.count))
        } else if keyData.count > 32 {
            // Truncate
            keyData = keyData.prefix(32)
        }

        // Generate HMAC-SHA256
        let key = SymmetricKey(data: keyData)
        let signature = HMAC<SHA256>.authenticationCode(
            for: Data(messageStr.utf8),
            using: key
        )

        // Convert to hex string
        return signature.map { String(format: "%02x", $0) }.joined()
    }

    // MARK: - Generate Nonce
    private func generateNonce() -> String {
        var bytes = [UInt8](repeating: 0, count: 16) // 128 bits
        _ = SecRandomCopyBytes(kSecRandomDefault, bytes.count, &bytes)
        return bytes.map { String(format: "%02x", $0) }.joined()
    }

    // MARK: - Verify Signature
    private func verifySignature(_ message: [String: Any]) -> Bool {
        guard let signature = message["signature"] as? String,
              let _ = message["nonce"] as? String,
              let type = message["type"] as? String,
              let timestamp = message["timestamp"] as? String,
              let messageId = message["messageId"] as? String,
              let payload = message["payload"] else {
            return false
        }

        do {
            let payloadData = try JSONSerialization.data(withJSONObject: payload)
            let payloadJSON = String(data: payloadData, encoding: .utf8) ?? "{}"

            let expectedSignature = try generateSignature(
                type: WSMessageType(rawValue: type) ?? .heartbeat,
                timestamp: timestamp,
                messageId: messageId,
                payloadJSON: payloadJSON
            )

            return signature == expectedSignature
        } catch {
            return false
        }
    }

    // MARK: - Receive Messages
    private func receiveMessage() {
        webSocketTask?.receive { [weak self] result in
            guard let self = self else { return }

            switch result {
            case .success(let message):
                Task { @MainActor in
                    await self.handleMessage(message)
                }
                // Continue receiving
                self.receiveMessage()

            case .failure(let error):
                Task { @MainActor in
                    await self.handleDisconnection(error: error)
                }
            }
        }
    }

    // MARK: - Handle Message
    private func handleMessage(_ message: URLSessionWebSocketTask.Message) async {
        switch message {
        case .string(let text):
            guard let data = text.data(using: .utf8),
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
                return
            }

            // Verify signature if present
            if json["signature"] != nil && json["nonce"] != nil {
                guard verifySignature(json) else {
                    print("Invalid message signature - disconnecting")
                    disconnect()
                    return
                }
            }

            // Get message type
            guard let typeStr = json["type"] as? String,
                  let type = WSMessageType(rawValue: typeStr) else {
                return
            }

            // Dispatch to handlers
            if let typeHandlers = handlers[type] {
                for handler in typeHandlers {
                    handler(json["payload"] ?? [:], type)
                }
            }

        case .data:
            // Binary messages not expected
            break

        @unknown default:
            break
        }
    }

    // MARK: - Message Handlers
    func on<T>(_ type: WSMessageType, handler: @escaping (T, WSMessageType) -> Void) -> UUID {
        let id = UUID()
        let wrappedHandler: (Any, WSMessageType) -> Void = { payload, msgType in
            if let typedPayload = payload as? T {
                handler(typedPayload, msgType)
            }
        }

        if handlers[type] == nil {
            handlers[type] = []
        }
        handlers[type]?.append(wrappedHandler)

        return id
    }

    // MARK: - Heartbeat
    private func startHeartbeat() {
        stopHeartbeat()

        heartbeatTimer = Timer.scheduledTimer(withTimeInterval: heartbeatInterval, repeats: true) { [weak self] _ in
            Task {
                try? await self?.send(.heartbeat, payload: EmptyPayload())
            }
        }
    }

    private func stopHeartbeat() {
        heartbeatTimer?.invalidate()
        heartbeatTimer = nil
    }

    // MARK: - Reconnection
    private func handleDisconnection(error: Error?) async {
        await MainActor.run {
            isConnected = false
            connectionError = error
        }

        stopHeartbeat()

        // Attempt reconnection
        guard reconnectAttempts < maxReconnectAttempts else {
            print("Max reconnection attempts reached")
            return
        }

        reconnectAttempts += 1
        let delay = reconnectDelay * pow(2.0, Double(reconnectAttempts - 1))

        try? await Task.sleep(nanoseconds: UInt64(delay * 1_000_000_000))

        if let token = authToken {
            await connect(token: token)
        }
    }

    // MARK: - Message Queue
    private func flushMessageQueue() async {
        while !messageQueue.isEmpty {
            let (type, payload) = messageQueue.removeFirst()
            if let encodablePayload = payload as? any Encodable {
                try? await send(type, payload: encodablePayload)
            }
        }
    }
}

// MARK: - URLSessionWebSocketDelegate
extension WebSocketManager: URLSessionWebSocketDelegate {
    nonisolated func urlSession(
        _ session: URLSession,
        webSocketTask: URLSessionWebSocketTask,
        didOpenWithProtocol protocol: String?
    ) {
        Task { @MainActor in
            self.isConnected = true
        }
    }

    nonisolated func urlSession(
        _ session: URLSession,
        webSocketTask: URLSessionWebSocketTask,
        didCloseWith closeCode: URLSessionWebSocketTask.CloseCode,
        reason: Data?
    ) {
        Task { @MainActor in
            await self.handleDisconnection(error: nil)
        }
    }
}

// MARK: - Convenience Methods
extension WebSocketManager {
    /// Send encrypted message
    func sendEncryptedMessage(_ payload: EncryptedPayload, messageId: String? = nil) async throws {
        try await send(.send, payload: payload, messageId: messageId)
    }

    /// Send typing indicator
    func sendTypingIndicator(to receiverId: String, isTyping: Bool) async throws {
        try await send(.typing, payload: TypingPayload(receiverId: receiverId, isTyping: isTyping))
    }

    /// Send read receipt
    func sendReadReceipt(messageIds: [String]) async throws {
        try await send(.readReceipt, payload: ReadReceiptPayload(messageIds: messageIds, status: "read"))
    }

    /// Send delivery acknowledgment
    func sendDeliveryAck(messageId: String) async throws {
        try await send(.deliveryAck, payload: DeliveryAckPayload(messageId: messageId))
    }
}

// MARK: - WebSocket Error
enum WebSocketError: LocalizedError {
    case notAuthenticated
    case connectionFailed
    case signatureVerificationFailed

    var errorDescription: String? {
        switch self {
        case .notAuthenticated:
            return "Not authenticated"
        case .connectionFailed:
            return "WebSocket connection failed"
        case .signatureVerificationFailed:
            return "Message signature verification failed"
        }
    }
}
