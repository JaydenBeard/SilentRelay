//
//  CachedContact.swift
//  SilentRelay
//
//  SwiftData model for locally cached contacts/users
//

import Foundation
import SwiftData

@Model
final class CachedContact {
    // MARK: - Properties
    @Attribute(.unique) var id: String
    var phoneNumber: String
    var username: String?
    var displayName: String?
    var avatar: String?
    var publicKey: String
    var createdAt: Date

    // Online status (cached locally)
    var isOnline: Bool
    var lastSeen: Date?

    // Identity key tracking for safety numbers
    var identityKeyFingerprint: String?
    var identityKeyChangedAt: Date?

    // Cache metadata
    var updatedAt: Date

    // MARK: - Init
    init(
        id: String,
        phoneNumber: String,
        username: String? = nil,
        displayName: String? = nil,
        avatar: String? = nil,
        publicKey: String,
        createdAt: Date = Date(),
        isOnline: Bool = false,
        lastSeen: Date? = nil,
        identityKeyFingerprint: String? = nil,
        identityKeyChangedAt: Date? = nil,
        updatedAt: Date = Date()
    ) {
        self.id = id
        self.phoneNumber = phoneNumber
        self.username = username
        self.displayName = displayName
        self.avatar = avatar
        self.publicKey = publicKey
        self.createdAt = createdAt
        self.isOnline = isOnline
        self.lastSeen = lastSeen
        self.identityKeyFingerprint = identityKeyFingerprint
        self.identityKeyChangedAt = identityKeyChangedAt
        self.updatedAt = updatedAt
    }
}

// MARK: - Display Helpers
extension CachedContact {
    /// Returns the best available display name
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

// MARK: - Conversion Extensions
extension CachedContact {
    /// Convert to API model
    func toUser() -> User {
        User(
            id: id,
            phoneNumber: phoneNumber,
            username: username,
            displayName: displayName,
            avatar: avatar,
            publicKey: publicKey,
            createdAt: createdAt
        )
    }

    /// Update from API model
    func update(from user: User) {
        phoneNumber = user.phoneNumber
        username = user.username
        displayName = user.displayName
        avatar = user.avatar
        publicKey = user.publicKey
        updatedAt = Date()
    }

    /// Create from API model
    static func from(_ user: User) -> CachedContact {
        CachedContact(
            id: user.id,
            phoneNumber: user.phoneNumber,
            username: user.username,
            displayName: user.displayName,
            avatar: user.avatar,
            publicKey: user.publicKey,
            createdAt: user.createdAt
        )
    }
}

// MARK: - Identity Key Tracking
extension CachedContact {
    /// Check if identity key has changed
    func hasIdentityKeyChanged(newFingerprint: String) -> Bool {
        guard let currentFingerprint = identityKeyFingerprint else {
            return false // First time seeing this key
        }
        return currentFingerprint != newFingerprint
    }

    /// Update identity key fingerprint
    func updateIdentityKey(fingerprint: String, hasChanged: Bool) {
        if hasChanged {
            identityKeyChangedAt = Date()
        }
        identityKeyFingerprint = fingerprint
        updatedAt = Date()
    }
}

// MARK: - Predicates
extension CachedContact {
    /// Search contacts by name or phone
    static func search(_ query: String) -> Predicate<CachedContact> {
        #Predicate<CachedContact> { contact in
            contact.displayName?.localizedStandardContains(query) == true ||
            contact.username?.localizedStandardContains(query) == true ||
            contact.phoneNumber.localizedStandardContains(query)
        }
    }
}
