//
//  CachedMessage.swift
//  SilentRelay
//
//  SwiftData model for locally cached messages
//

import Foundation
import SwiftData

@Model
final class CachedMessage {
    // MARK: - Properties
    @Attribute(.unique) var id: String
    var conversationId: String
    var senderId: String
    var content: String
    var timestamp: Date
    var status: CachedMessageStatus
    var type: CachedMessageType
    var replyTo: String?

    // File metadata (stored as JSON)
    var fileMetadataJSON: Data?

    // Call metadata (stored as JSON)
    var callMetadataJSON: Data?

    // Relationship
    var conversation: CachedConversation?

    // MARK: - Init
    init(
        id: String,
        conversationId: String,
        senderId: String,
        content: String,
        timestamp: Date = Date(),
        status: CachedMessageStatus = .sending,
        type: CachedMessageType = .text,
        replyTo: String? = nil,
        fileMetadataJSON: Data? = nil,
        callMetadataJSON: Data? = nil,
        conversation: CachedConversation? = nil
    ) {
        self.id = id
        self.conversationId = conversationId
        self.senderId = senderId
        self.content = content
        self.timestamp = timestamp
        self.status = status
        self.type = type
        self.replyTo = replyTo
        self.fileMetadataJSON = fileMetadataJSON
        self.callMetadataJSON = callMetadataJSON
        self.conversation = conversation
    }
}

// MARK: - Status Enum
enum CachedMessageStatus: String, Codable {
    case sending
    case sent
    case delivered
    case read
    case failed
}

// MARK: - Type Enum
enum CachedMessageType: String, Codable {
    case text
    case file
    case voice
    case image
    case video
    case call
}

// MARK: - Metadata Accessors
extension CachedMessage {
    /// Get file metadata
    var fileMetadata: FileMetadata? {
        guard let data = fileMetadataJSON else { return nil }
        return try? JSONDecoder().decode(FileMetadata.self, from: data)
    }

    /// Set file metadata
    func setFileMetadata(_ metadata: FileMetadata?) {
        guard let metadata = metadata else {
            fileMetadataJSON = nil
            return
        }
        fileMetadataJSON = try? JSONEncoder().encode(metadata)
    }

    /// Get call metadata
    var callMetadata: CallMetadata? {
        guard let data = callMetadataJSON else { return nil }
        return try? JSONDecoder().decode(CallMetadata.self, from: data)
    }

    /// Set call metadata
    func setCallMetadata(_ metadata: CallMetadata?) {
        guard let metadata = metadata else {
            callMetadataJSON = nil
            return
        }
        callMetadataJSON = try? JSONEncoder().encode(metadata)
    }
}

// MARK: - Conversion Extensions
extension CachedMessage {
    /// Convert to API model
    func toMessage() -> Message {
        Message(
            id: id,
            conversationId: conversationId,
            senderId: senderId,
            content: content,
            timestamp: timestamp,
            status: MessageStatus(rawValue: status.rawValue) ?? .sending,
            type: MessageType(rawValue: type.rawValue) ?? .text,
            replyTo: replyTo,
            metadata: fileMetadata,
            callMetadata: callMetadata
        )
    }

    /// Update from API model
    func update(from message: Message) {
        content = message.content
        status = CachedMessageStatus(rawValue: message.status.rawValue) ?? .sending
        setFileMetadata(message.metadata)
        setCallMetadata(message.callMetadata)
    }

    /// Create from API model
    static func from(_ message: Message) -> CachedMessage {
        let cached = CachedMessage(
            id: message.id,
            conversationId: message.conversationId,
            senderId: message.senderId,
            content: message.content,
            timestamp: message.timestamp,
            status: CachedMessageStatus(rawValue: message.status.rawValue) ?? .sending,
            type: CachedMessageType(rawValue: message.type.rawValue) ?? .text,
            replyTo: message.replyTo
        )
        cached.setFileMetadata(message.metadata)
        cached.setCallMetadata(message.callMetadata)
        return cached
    }
}

// MARK: - Sorting
extension CachedMessage {
    /// Sort descriptor for messages (oldest first)
    static var chronologicalSort: [SortDescriptor<CachedMessage>] {
        [SortDescriptor(\.timestamp, order: .forward)]
    }

    /// Sort descriptor for messages (newest first)
    static var reverseChronologicalSort: [SortDescriptor<CachedMessage>] {
        [SortDescriptor(\.timestamp, order: .reverse)]
    }
}

// MARK: - Predicates
extension CachedMessage {
    /// Predicate for messages in a conversation
    static func forConversation(_ conversationId: String) -> Predicate<CachedMessage> {
        #Predicate<CachedMessage> { message in
            message.conversationId == conversationId
        }
    }

    /// Predicate for unread messages
    static var unread: Predicate<CachedMessage> {
        #Predicate<CachedMessage> { message in
            message.status != .read
        }
    }

    /// Predicate for failed messages
    static var failed: Predicate<CachedMessage> {
        #Predicate<CachedMessage> { message in
            message.status == .failed
        }
    }
}
