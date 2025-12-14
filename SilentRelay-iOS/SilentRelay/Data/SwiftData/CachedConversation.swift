//
//  CachedConversation.swift
//  SilentRelay
//
//  SwiftData model for locally cached conversations
//

import Foundation
import SwiftData

@Model
final class CachedConversation {
    // MARK: - Properties
    @Attribute(.unique) var id: String
    var recipientId: String
    var recipientName: String
    var recipientAvatar: String?

    // Last message preview
    var lastMessageText: String?
    var lastMessageTime: Date?

    // State
    var unreadCount: Int
    var isOnline: Bool
    var lastSeen: Date?
    var isPinned: Bool
    var isMuted: Bool
    var status: CachedConversationStatus

    // Relationships
    @Relationship(deleteRule: .cascade, inverse: \CachedMessage.conversation)
    var messages: [CachedMessage]?

    // Timestamps
    var createdAt: Date
    var updatedAt: Date

    // MARK: - Init
    init(
        id: String,
        recipientId: String,
        recipientName: String,
        recipientAvatar: String? = nil,
        lastMessageText: String? = nil,
        lastMessageTime: Date? = nil,
        unreadCount: Int = 0,
        isOnline: Bool = false,
        lastSeen: Date? = nil,
        isPinned: Bool = false,
        isMuted: Bool = false,
        status: CachedConversationStatus = .accepted,
        messages: [CachedMessage]? = nil,
        createdAt: Date = Date(),
        updatedAt: Date = Date()
    ) {
        self.id = id
        self.recipientId = recipientId
        self.recipientName = recipientName
        self.recipientAvatar = recipientAvatar
        self.lastMessageText = lastMessageText
        self.lastMessageTime = lastMessageTime
        self.unreadCount = unreadCount
        self.isOnline = isOnline
        self.lastSeen = lastSeen
        self.isPinned = isPinned
        self.isMuted = isMuted
        self.status = status
        self.messages = messages
        self.createdAt = createdAt
        self.updatedAt = updatedAt
    }
}

// MARK: - Status Enum
enum CachedConversationStatus: String, Codable {
    case pending
    case accepted
    case blocked
}

// MARK: - Conversion Extensions
extension CachedConversation {
    /// Convert to API model
    func toConversation() -> Conversation {
        Conversation(
            id: id,
            recipientId: recipientId,
            recipientName: recipientName,
            recipientAvatar: recipientAvatar,
            lastMessage: nil, // Messages loaded separately
            unreadCount: unreadCount,
            isOnline: isOnline,
            lastSeen: lastSeen,
            isPinned: isPinned,
            isMuted: isMuted,
            status: ConversationStatus(rawValue: status.rawValue) ?? .accepted
        )
    }

    /// Update from API model
    func update(from conversation: Conversation) {
        recipientName = conversation.recipientName
        recipientAvatar = conversation.recipientAvatar
        unreadCount = conversation.unreadCount
        isOnline = conversation.isOnline
        lastSeen = conversation.lastSeen
        isPinned = conversation.isPinned
        isMuted = conversation.isMuted
        status = CachedConversationStatus(rawValue: conversation.status.rawValue) ?? .accepted

        if let lastMsg = conversation.lastMessage {
            lastMessageText = lastMsg.content
            lastMessageTime = lastMsg.timestamp
        }

        updatedAt = Date()
    }

    /// Create from API model
    static func from(_ conversation: Conversation) -> CachedConversation {
        CachedConversation(
            id: conversation.id,
            recipientId: conversation.recipientId,
            recipientName: conversation.recipientName,
            recipientAvatar: conversation.recipientAvatar,
            lastMessageText: conversation.lastMessage?.content,
            lastMessageTime: conversation.lastMessage?.timestamp,
            unreadCount: conversation.unreadCount,
            isOnline: conversation.isOnline,
            lastSeen: conversation.lastSeen,
            isPinned: conversation.isPinned,
            isMuted: conversation.isMuted,
            status: CachedConversationStatus(rawValue: conversation.status.rawValue) ?? .accepted
        )
    }
}

// MARK: - Sorting
extension CachedConversation {
    /// Sort descriptor for conversation list
    static var defaultSort: [SortDescriptor<CachedConversation>] {
        [
            SortDescriptor(\.isPinned, order: .reverse),
            SortDescriptor(\.lastMessageTime, order: .reverse)
        ]
    }
}
