//
//  User.swift
//  SilentRelay
//
//  User model matching web-new/src/core/types/index.ts
//

import Foundation

/// Represents a user in the SilentRelay system
struct User: Codable, Identifiable, Equatable, Sendable {
    let id: String
    let phoneNumber: String
    var username: String?
    var displayName: String?
    var avatar: String?
    let publicKey: String
    let createdAt: Date

    // MARK: - Coding Keys (snake_case for backend compatibility)
    enum CodingKeys: String, CodingKey {
        case id
        case phoneNumber = "phone_number"
        case username
        case displayName = "display_name"
        case avatar
        case publicKey = "public_key"
        case createdAt = "created_at"
    }

    // MARK: - Custom Decoding for flexible date handling
    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(String.self, forKey: .id)
        phoneNumber = try container.decode(String.self, forKey: .phoneNumber)
        username = try container.decodeIfPresent(String.self, forKey: .username)
        displayName = try container.decodeIfPresent(String.self, forKey: .displayName)
        avatar = try container.decodeIfPresent(String.self, forKey: .avatar)
        publicKey = try container.decode(String.self, forKey: .publicKey)

        // Handle both Unix timestamp (number) and ISO 8601 string
        if let timestamp = try? container.decode(Double.self, forKey: .createdAt) {
            createdAt = Date(timeIntervalSince1970: timestamp / 1000) // JS uses milliseconds
        } else if let dateString = try? container.decode(String.self, forKey: .createdAt) {
            let formatter = ISO8601DateFormatter()
            formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
            createdAt = formatter.date(from: dateString) ?? Date()
        } else {
            createdAt = Date()
        }
    }

    // MARK: - Convenience Init
    init(
        id: String,
        phoneNumber: String,
        username: String? = nil,
        displayName: String? = nil,
        avatar: String? = nil,
        publicKey: String,
        createdAt: Date = Date()
    ) {
        self.id = id
        self.phoneNumber = phoneNumber
        self.username = username
        self.displayName = displayName
        self.avatar = avatar
        self.publicKey = publicKey
        self.createdAt = createdAt
    }
}

// MARK: - Display Helpers
extension User {
    /// Returns the best available display name for the user
    var effectiveDisplayName: String {
        displayName ?? username ?? phoneNumber
    }

    /// Returns initials for avatar placeholder
    var initials: String {
        let name = effectiveDisplayName
        let components = name.components(separatedBy: " ")
        if components.count >= 2 {
            let first = components[0].prefix(1)
            let last = components[1].prefix(1)
            return "\(first)\(last)".uppercased()
        }
        return String(name.prefix(2)).uppercased()
    }
}

// MARK: - User Profile Response
struct UserProfileResponse: Codable {
    let user: User
    let isOnline: Bool?
    let lastSeen: Date?

    enum CodingKeys: String, CodingKey {
        case user
        case isOnline = "is_online"
        case lastSeen = "last_seen"
    }
}

// MARK: - User Keys Response
struct UserKeysResponse: Codable {
    let userId: String
    let identityKey: String
    let signedPreKeyId: Int
    let signedPreKey: String
    let signedPreKeySignature: String
    let preKeyId: Int?
    let preKey: String?

    enum CodingKeys: String, CodingKey {
        case userId = "user_id"
        case identityKey = "identity_key"
        case signedPreKeyId = "signed_pre_key_id"
        case signedPreKey = "signed_pre_key"
        case signedPreKeySignature = "signed_pre_key_signature"
        case preKeyId = "pre_key_id"
        case preKey = "pre_key"
    }
}
