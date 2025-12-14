//
//  Conversation.swift
//  SilentRelay
//
//  Conversation model matching web-new/src/core/types/index.ts
//

import Foundation

// MARK: - Conversation Status
enum ConversationStatus: String, Codable, Sendable {
    case pending   // Message request not yet accepted
    case accepted  // Normal conversation
    case blocked   // User has been blocked
}

// MARK: - Conversation
struct Conversation: Codable, Identifiable, Equatable, Sendable {
    let id: String
    let recipientId: String
    var recipientName: String
    var recipientAvatar: String?
    var lastMessage: Message?
    var unreadCount: Int
    var isOnline: Bool
    var lastSeen: Date?
    var isPinned: Bool
    var isMuted: Bool
    var status: ConversationStatus

    // MARK: - Coding Keys
    enum CodingKeys: String, CodingKey {
        case id
        case recipientId = "recipient_id"
        case recipientName = "recipient_name"
        case recipientAvatar = "recipient_avatar"
        case lastMessage = "last_message"
        case unreadCount = "unread_count"
        case isOnline = "is_online"
        case lastSeen = "last_seen"
        case isPinned = "is_pinned"
        case isMuted = "is_muted"
        case status
    }

    // MARK: - Custom Decoding
    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(String.self, forKey: .id)
        recipientId = try container.decode(String.self, forKey: .recipientId)
        recipientName = try container.decode(String.self, forKey: .recipientName)
        recipientAvatar = try container.decodeIfPresent(String.self, forKey: .recipientAvatar)
        lastMessage = try container.decodeIfPresent(Message.self, forKey: .lastMessage)
        unreadCount = try container.decodeIfPresent(Int.self, forKey: .unreadCount) ?? 0
        isOnline = try container.decodeIfPresent(Bool.self, forKey: .isOnline) ?? false
        isPinned = try container.decodeIfPresent(Bool.self, forKey: .isPinned) ?? false
        isMuted = try container.decodeIfPresent(Bool.self, forKey: .isMuted) ?? false
        status = try container.decodeIfPresent(ConversationStatus.self, forKey: .status) ?? .accepted

        // Handle lastSeen as number or ISO string
        if let lastSeenMs = try? container.decode(Double.self, forKey: .lastSeen) {
            lastSeen = Date(timeIntervalSince1970: lastSeenMs / 1000)
        } else if let dateString = try? container.decode(String.self, forKey: .lastSeen) {
            let formatter = ISO8601DateFormatter()
            formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
            lastSeen = formatter.date(from: dateString)
        } else {
            lastSeen = nil
        }
    }

    // MARK: - Convenience Init
    init(
        id: String,
        recipientId: String,
        recipientName: String,
        recipientAvatar: String? = nil,
        lastMessage: Message? = nil,
        unreadCount: Int = 0,
        isOnline: Bool = false,
        lastSeen: Date? = nil,
        isPinned: Bool = false,
        isMuted: Bool = false,
        status: ConversationStatus = .accepted
    ) {
        self.id = id
        self.recipientId = recipientId
        self.recipientName = recipientName
        self.recipientAvatar = recipientAvatar
        self.lastMessage = lastMessage
        self.unreadCount = unreadCount
        self.isOnline = isOnline
        self.lastSeen = lastSeen
        self.isPinned = isPinned
        self.isMuted = isMuted
        self.status = status
    }
}

// MARK: - Conversation Helpers
extension Conversation {
    /// Returns initials for avatar placeholder
    var initials: String {
        let components = recipientName.components(separatedBy: " ")
        if components.count >= 2 {
            let first = components[0].prefix(1)
            let last = components[1].prefix(1)
            return "\(first)\(last)".uppercased()
        }
        return String(recipientName.prefix(2)).uppercased()
    }

    /// Format last seen time
    var formattedLastSeen: String? {
        guard let lastSeen = lastSeen else { return nil }

        let formatter = RelativeDateTimeFormatter()
        formatter.unitsStyle = .short
        return formatter.localizedString(for: lastSeen, relativeTo: Date())
    }

    /// Preview text for conversation list
    var previewText: String {
        guard let message = lastMessage else {
            return "No messages yet"
        }

        switch message.type {
        case .text:
            return message.content
        case .image:
            return "Photo"
        case .video:
            return "Video"
        case .file:
            return message.metadata?.fileName ?? "File"
        case .voice:
            return "Voice message"
        case .call:
            guard let callMeta = message.callMetadata else {
                return "Call"
            }
            let icon = callMeta.callType == .video ? "Video" : "Voice"
            switch callMeta.endReason {
            case .completed:
                let duration = callMeta.duration ?? 0
                let minutes = duration / 60
                let seconds = duration % 60
                return "\(icon) call (\(minutes):\(String(format: "%02d", seconds)))"
            case .missed:
                return "Missed \(icon.lowercased()) call"
            case .declined:
                return "Declined \(icon.lowercased()) call"
            default:
                return "\(icon) call"
            }
        }
    }
}

// MARK: - Sorting
extension Conversation {
    /// Sort key for conversation list (pinned first, then by last message time)
    var sortOrder: (Bool, Date) {
        (isPinned, lastMessage?.timestamp ?? Date.distantPast)
    }
}
