//
//  WebSocketMessage.swift
//  SilentRelay
//
//  WebSocket message types matching web-new/src/core/types/index.ts
//

import Foundation

// MARK: - WebSocket Message Type
enum WSMessageType: String, Codable, Sendable {
    case send
    case deliver
    case sentAck = "sent_ack"
    case statusUpdate = "status_update"
    case typing
    case readReceipt = "read_receipt"
    case deliveryAck = "delivery_ack"
    case userOnline = "user_online"
    case userOffline = "user_offline"
    case userBlocked = "user_blocked"
    case callOffer = "call_offer"
    case callAnswer = "call_answer"
    case callReject = "call_reject"
    case callEnd = "call_end"
    case callBusy = "call_busy"
    case iceCandidate = "ice_candidate"
    case syncRequest = "sync_request"
    case syncData = "sync_data"
    case syncAck = "sync_ack"
    case identityKeyChanged = "identity_key_changed"
    case mediaKey = "media_key"
    case heartbeat
}

// MARK: - WebSocket Message
struct WSMessage<T: Codable>: Codable, Sendable where T: Sendable {
    let type: WSMessageType
    let payload: T
    let timestamp: String // ISO 8601 format
    let messageId: String
    var senderId: String?
    var signature: String?
    var nonce: String?

    enum CodingKeys: String, CodingKey {
        case type
        case payload
        case timestamp
        case messageId = "messageId"
        case senderId = "sender_id"
        case signature
        case nonce
    }

    init(
        type: WSMessageType,
        payload: T,
        timestamp: String = ISO8601DateFormatter().string(from: Date()),
        messageId: String = UUID().uuidString,
        senderId: String? = nil,
        signature: String? = nil,
        nonce: String? = nil
    ) {
        self.type = type
        self.payload = payload
        self.timestamp = timestamp
        self.messageId = messageId
        self.senderId = senderId
        self.signature = signature
        self.nonce = nonce
    }
}

// MARK: - Payload Types

/// Encrypted message payload (for send/deliver)
struct EncryptedPayload: Codable, Sendable {
    var senderId: String?
    var receiverId: String?
    let ciphertext: String
    let messageType: String // "prekey" or "whisper"
    var ephemeralKey: String?

    enum CodingKeys: String, CodingKey {
        case senderId = "sender_id"
        case receiverId = "receiver_id"
        case ciphertext
        case messageType = "message_type"
        case ephemeralKey = "ephemeral_key"
    }
}

/// Typing indicator payload
struct TypingPayload: Codable, Sendable {
    var receiverId: String?
    let isTyping: Bool

    enum CodingKeys: String, CodingKey {
        case receiverId = "receiver_id"
        case isTyping = "is_typing"
    }
}

/// Read receipt payload
struct ReadReceiptPayload: Codable, Sendable {
    let messageIds: [String]
    let status: String // "delivered" or "read"

    enum CodingKeys: String, CodingKey {
        case messageIds = "message_ids"
        case status
    }
}

/// Presence payload
struct PresencePayload: Codable, Sendable {
    let userId: String
    var isOnline: Bool?
    var lastSeen: Date?

    enum CodingKeys: String, CodingKey {
        case userId = "user_id"
        case isOnline = "is_online"
        case lastSeen = "last_seen"
    }
}

/// Status update payload
struct StatusUpdatePayload: Codable, Sendable {
    let messageId: String
    let status: MessageStatus

    enum CodingKeys: String, CodingKey {
        case messageId = "messageId"
        case status
    }
}

/// Call offer payload
struct CallOfferPayload: Codable, Sendable {
    let callId: String
    let callerId: String
    let callerName: String
    let callType: CallType
    let sdp: String

    enum CodingKeys: String, CodingKey {
        case callId = "call_id"
        case callerId = "caller_id"
        case callerName = "caller_name"
        case callType = "call_type"
        case sdp
    }
}

/// Call answer payload
struct CallAnswerPayload: Codable, Sendable {
    let callId: String
    let sdp: String

    enum CodingKeys: String, CodingKey {
        case callId = "call_id"
        case sdp
    }
}

/// ICE candidate payload
struct ICECandidatePayload: Codable, Sendable {
    let callId: String
    let candidate: String
    let sdpMid: String?
    let sdpMLineIndex: Int?

    enum CodingKeys: String, CodingKey {
        case callId = "call_id"
        case candidate
        case sdpMid = "sdp_mid"
        case sdpMLineIndex = "sdp_m_line_index"
    }
}

/// Empty payload for heartbeat, etc.
struct EmptyPayload: Codable, Sendable {}

/// Delivery acknowledgment payload
struct DeliveryAckPayload: Codable, Sendable {
    let messageId: String

    enum CodingKeys: String, CodingKey {
        case messageId = "message_id"
    }
}

/// Identity key changed payload
struct IdentityKeyChangedPayload: Codable, Sendable {
    let userId: String
    let newIdentityKey: String

    enum CodingKeys: String, CodingKey {
        case userId = "user_id"
        case newIdentityKey = "new_identity_key"
    }
}

// MARK: - Settings Types

/// Privacy settings
struct PrivacySettings: Codable, Equatable, Sendable {
    var readReceipts: Bool
    var onlineStatus: Bool
    var lastSeen: Bool
    var typingIndicators: Bool

    enum CodingKeys: String, CodingKey {
        case readReceipts = "read_receipts"
        case onlineStatus = "online_status"
        case lastSeen = "last_seen"
        case typingIndicators = "typing_indicators"
    }

    init(
        readReceipts: Bool = true,
        onlineStatus: Bool = true,
        lastSeen: Bool = true,
        typingIndicators: Bool = true
    ) {
        self.readReceipts = readReceipts
        self.onlineStatus = onlineStatus
        self.lastSeen = lastSeen
        self.typingIndicators = typingIndicators
    }
}

/// Notification settings
struct NotificationSettings: Codable, Equatable, Sendable {
    var enabled: Bool
    var sound: Bool
    var preview: Bool

    init(enabled: Bool = true, sound: Bool = true, preview: Bool = true) {
        self.enabled = enabled
        self.sound = sound
        self.preview = preview
    }
}

/// App settings
struct AppSettings: Codable, Equatable, Sendable {
    var theme: AppTheme
    var fontSize: FontSize
    var language: String

    init(theme: AppTheme = .system, fontSize: FontSize = .medium, language: String = "en") {
        self.theme = theme
        self.fontSize = fontSize
        self.language = language
    }
}

enum AppTheme: String, Codable, Sendable {
    case dark
    case light
    case system
}

enum FontSize: String, Codable, Sendable {
    case small
    case medium
    case large
}
