//
//  KeychainManager.swift
//  SilentRelay
//
//  Secure storage for sensitive data using iOS Keychain
//

import Foundation
import Security

// MARK: - Keychain Manager
actor KeychainManager {
    // MARK: - Singleton
    static let shared = KeychainManager()

    // MARK: - Configuration
    private let service = "com.silentrelay.ios"

    // Accessibility level for different data types
    // Using afterFirstUnlockThisDeviceOnly for security-sensitive data
    private let defaultAccessibility = kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly

    private init() {}

    // MARK: - Keychain Keys
    enum Key: String {
        // Auth tokens
        case accessToken = "access_token"
        case refreshToken = "refresh_token"
        case deviceId = "device_id"

        // Encryption keys
        case masterKeySalt = "master_key_salt"
        case encryptedIdentityKey = "encrypted_identity_key"
        case identityKeyIV = "identity_key_iv"

        // Signal Protocol
        case registrationId = "registration_id"
        case signalDeviceId = "signal_device_id"

        // PIN
        case pinHash = "pin_hash"
        case biometricEnabled = "biometric_enabled"
    }

    // MARK: - Save Data
    func save(_ data: Data, for key: Key) throws {
        // Delete existing item first
        try? delete(key)

        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key.rawValue,
            kSecValueData as String: data,
            kSecAttrAccessible as String: defaultAccessibility
        ]

        let status = SecItemAdd(query as CFDictionary, nil)

        guard status == errSecSuccess else {
            throw KeychainError.saveFailed(status)
        }
    }

    /// Save a string value
    func save(_ string: String, for key: Key) throws {
        guard let data = string.data(using: .utf8) else {
            throw KeychainError.encodingFailed
        }
        try save(data, for: key)
    }

    /// Save an integer value
    func save(_ value: Int, for key: Key) throws {
        let data = withUnsafeBytes(of: value) { Data($0) }
        try save(data, for: key)
    }

    /// Save a boolean value
    func save(_ value: Bool, for key: Key) throws {
        try save(value ? 1 : 0, for: key)
    }

    // MARK: - Load Data
    func load(_ key: Key) throws -> Data {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key.rawValue,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne
        ]

        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)

        guard status == errSecSuccess else {
            if status == errSecItemNotFound {
                throw KeychainError.itemNotFound
            }
            throw KeychainError.loadFailed(status)
        }

        guard let data = result as? Data else {
            throw KeychainError.invalidData
        }

        return data
    }

    /// Load a string value
    func loadString(_ key: Key) throws -> String {
        let data = try load(key)
        guard let string = String(data: data, encoding: .utf8) else {
            throw KeychainError.decodingFailed
        }
        return string
    }

    /// Load an integer value
    func loadInt(_ key: Key) throws -> Int {
        let data = try load(key)
        return data.withUnsafeBytes { $0.load(as: Int.self) }
    }

    /// Load a boolean value
    func loadBool(_ key: Key) throws -> Bool {
        let value = try loadInt(key)
        return value != 0
    }

    /// Load with optional result (doesn't throw for not found)
    func loadOptional(_ key: Key) -> Data? {
        try? load(key)
    }

    func loadStringOptional(_ key: Key) -> String? {
        try? loadString(key)
    }

    func loadIntOptional(_ key: Key) -> Int? {
        try? loadInt(key)
    }

    func loadBoolOptional(_ key: Key) -> Bool? {
        try? loadBool(key)
    }

    // MARK: - Delete
    func delete(_ key: Key) throws {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key.rawValue
        ]

        let status = SecItemDelete(query as CFDictionary)

        guard status == errSecSuccess || status == errSecItemNotFound else {
            throw KeychainError.deleteFailed(status)
        }
    }

    // MARK: - Clear All
    func clearAll() throws {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service
        ]

        let status = SecItemDelete(query as CFDictionary)

        guard status == errSecSuccess || status == errSecItemNotFound else {
            throw KeychainError.deleteFailed(status)
        }
    }

    // MARK: - Check Exists
    func exists(_ key: Key) -> Bool {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key.rawValue,
            kSecReturnData as String: false
        ]

        let status = SecItemCopyMatching(query as CFDictionary, nil)
        return status == errSecSuccess
    }
}

// MARK: - Keychain Error
enum KeychainError: LocalizedError {
    case saveFailed(OSStatus)
    case loadFailed(OSStatus)
    case deleteFailed(OSStatus)
    case itemNotFound
    case invalidData
    case encodingFailed
    case decodingFailed

    var errorDescription: String? {
        switch self {
        case .saveFailed(let status):
            return "Failed to save to keychain: \(status)"
        case .loadFailed(let status):
            return "Failed to load from keychain: \(status)"
        case .deleteFailed(let status):
            return "Failed to delete from keychain: \(status)"
        case .itemNotFound:
            return "Item not found in keychain"
        case .invalidData:
            return "Invalid data in keychain"
        case .encodingFailed:
            return "Failed to encode data"
        case .decodingFailed:
            return "Failed to decode data"
        }
    }
}

// MARK: - Convenience Extensions
extension KeychainManager {
    /// Save auth tokens after login
    func saveAuthTokens(accessToken: String, refreshToken: String) async throws {
        try save(accessToken, for: .accessToken)
        try save(refreshToken, for: .refreshToken)
    }

    /// Load auth tokens
    func loadAuthTokens() async throws -> (accessToken: String, refreshToken: String) {
        let accessToken = try loadString(.accessToken)
        let refreshToken = try loadString(.refreshToken)
        return (accessToken, refreshToken)
    }

    /// Clear auth tokens on logout
    func clearAuthTokens() async throws {
        try delete(.accessToken)
        try delete(.refreshToken)
    }

    /// Get or create device ID
    func getOrCreateDeviceId() async throws -> String {
        if let existingId = loadStringOptional(.deviceId) {
            return existingId
        }

        let newId = UUID().uuidString
        try save(newId, for: .deviceId)
        return newId
    }
}
