//
//  Message.swift
//  SilentRelay
//
//  Message model matching web-new/src/core/types/index.ts
//

import Foundation

// MARK: - Message Status
enum MessageStatus: String, Codable, Sendable {
    case sending
    case sent
    case delivered
    case read
    case failed
}

// MARK: - Message Type
enum MessageType: String, Codable, Sendable {
    case text
    case file
    case voice
    case image
    case video
    case call
}

// MARK: - Message
struct Message: Codable, Identifiable, Equatable, Sendable {
    let id: String
    let conversationId: String
    let senderId: String
    var content: String
    let timestamp: Date
    var status: MessageStatus
    let type: MessageType
    var replyTo: String?
    var metadata: FileMetadata?
    var callMetadata: CallMetadata?

    // MARK: - Coding Keys
    enum CodingKeys: String, CodingKey {
        case id
        case conversationId = "conversation_id"
        case senderId = "sender_id"
        case content
        case timestamp
        case status
        case type
        case replyTo = "reply_to"
        case metadata
        case callMetadata = "call_metadata"
    }

    // MARK: - Custom Decoding
    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(String.self, forKey: .id)
        conversationId = try container.decode(String.self, forKey: .conversationId)
        senderId = try container.decode(String.self, forKey: .senderId)
        content = try container.decode(String.self, forKey: .content)
        status = try container.decode(MessageStatus.self, forKey: .status)
        type = try container.decode(MessageType.self, forKey: .type)
        replyTo = try container.decodeIfPresent(String.self, forKey: .replyTo)
        metadata = try container.decodeIfPresent(FileMetadata.self, forKey: .metadata)
        callMetadata = try container.decodeIfPresent(CallMetadata.self, forKey: .callMetadata)

        // Handle timestamp as number (Unix ms) or ISO string
        if let timestampMs = try? container.decode(Double.self, forKey: .timestamp) {
            timestamp = Date(timeIntervalSince1970: timestampMs / 1000)
        } else if let dateString = try? container.decode(String.self, forKey: .timestamp) {
            let formatter = ISO8601DateFormatter()
            formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
            timestamp = formatter.date(from: dateString) ?? Date()
        } else {
            timestamp = Date()
        }
    }

    // MARK: - Convenience Init
    init(
        id: String,
        conversationId: String,
        senderId: String,
        content: String,
        timestamp: Date = Date(),
        status: MessageStatus = .sending,
        type: MessageType = .text,
        replyTo: String? = nil,
        metadata: FileMetadata? = nil,
        callMetadata: CallMetadata? = nil
    ) {
        self.id = id
        self.conversationId = conversationId
        self.senderId = senderId
        self.content = content
        self.timestamp = timestamp
        self.status = status
        self.type = type
        self.replyTo = replyTo
        self.metadata = metadata
        self.callMetadata = callMetadata
    }
}

// MARK: - File Metadata
struct FileMetadata: Codable, Equatable, Sendable {
    let fileName: String
    let fileSize: Int
    let mimeType: String
    let mediaId: String
    var thumbnail: String?
    let encryptionKey: [UInt8]
    let iv: [UInt8]

    enum CodingKeys: String, CodingKey {
        case fileName = "file_name"
        case fileSize = "file_size"
        case mimeType = "mime_type"
        case mediaId = "media_id"
        case thumbnail
        case encryptionKey = "encryption_key"
        case iv
    }
}

// MARK: - Call Metadata
struct CallMetadata: Codable, Equatable, Sendable {
    let callType: CallType
    var duration: Int? // in seconds
    let endReason: CallEndReason
    let direction: CallDirection

    enum CodingKeys: String, CodingKey {
        case callType = "call_type"
        case duration
        case endReason = "end_reason"
        case direction
    }
}

enum CallType: String, Codable, Sendable {
    case audio
    case video
}

enum CallEndReason: String, Codable, Sendable {
    case completed
    case declined
    case missed
    case failed
    case busy
    case cancelled
}

enum CallDirection: String, Codable, Sendable {
    case incoming
    case outgoing
}

// MARK: - Message Helpers
extension Message {
    /// Check if this message is from the current user
    func isFromCurrentUser(_ currentUserId: String) -> Bool {
        senderId == currentUserId
    }

    /// Format timestamp for display
    var formattedTime: String {
        let formatter = DateFormatter()
        let calendar = Calendar.current

        if calendar.isDateInToday(timestamp) {
            formatter.dateFormat = "h:mm a"
        } else if calendar.isDateInYesterday(timestamp) {
            return "Yesterday"
        } else if calendar.isDate(timestamp, equalTo: Date(), toGranularity: .weekOfYear) {
            formatter.dateFormat = "EEEE"
        } else {
            formatter.dateFormat = "MMM d"
        }

        return formatter.string(from: timestamp)
    }
}

// MARK: - Encrypted Message (WebSocket payload)
struct EncryptedMessage: Codable, Sendable {
    let senderId: String?
    let receiverId: String?
    let ciphertext: String
    let messageType: String // "prekey" or "whisper"
    let ephemeralKey: String?

    enum CodingKeys: String, CodingKey {
        case senderId = "sender_id"
        case receiverId = "receiver_id"
        case ciphertext
        case messageType = "message_type"
        case ephemeralKey = "ephemeral_key"
    }
}
