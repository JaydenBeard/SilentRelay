//
//  iCloudKeychainManager.swift
//  SilentRelay
//
//  iCloud Keychain sync for master key backup across devices
//

import Foundation
import Security

// MARK: - iCloud Keychain Manager
actor iCloudKeychainManager {
    // MARK: - Singleton
    static let shared = iCloudKeychainManager()

    // MARK: - Configuration
    private let service = "com.silentrelay.ios.sync"

    // Use kSecAttrSynchronizable for iCloud sync
    private let accessibility = kSecAttrAccessibleAfterFirstUnlock

    private init() {}

    // MARK: - Sync Keys
    enum SyncKey: String {
        case masterKeyBackup = "master_key_backup"
        case masterKeySaltBackup = "master_key_salt_backup"
        case identityKeyBackup = "identity_key_backup"
        case recoveryKey = "recovery_key"
    }

    // MARK: - Save to iCloud Keychain
    func save(_ data: Data, for key: SyncKey) throws {
        // Delete existing item first
        try? delete(key)

        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key.rawValue,
            kSecValueData as String: data,
            kSecAttrAccessible as String: accessibility,
            kSecAttrSynchronizable as String: true  // Enable iCloud sync
        ]

        let status = SecItemAdd(query as CFDictionary, nil)

        guard status == errSecSuccess else {
            throw iCloudKeychainError.saveFailed(status)
        }
    }

    /// Save a string value
    func save(_ string: String, for key: SyncKey) throws {
        guard let data = string.data(using: .utf8) else {
            throw iCloudKeychainError.encodingFailed
        }
        try save(data, for: key)
    }

    // MARK: - Load from iCloud Keychain
    func load(_ key: SyncKey) throws -> Data {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key.rawValue,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne,
            kSecAttrSynchronizable as String: true
        ]

        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)

        guard status == errSecSuccess else {
            if status == errSecItemNotFound {
                throw iCloudKeychainError.itemNotFound
            }
            throw iCloudKeychainError.loadFailed(status)
        }

        guard let data = result as? Data else {
            throw iCloudKeychainError.invalidData
        }

        return data
    }

    /// Load a string value
    func loadString(_ key: SyncKey) throws -> String {
        let data = try load(key)
        guard let string = String(data: data, encoding: .utf8) else {
            throw iCloudKeychainError.decodingFailed
        }
        return string
    }

    /// Load with optional result
    func loadOptional(_ key: SyncKey) -> Data? {
        try? load(key)
    }

    func loadStringOptional(_ key: SyncKey) -> String? {
        try? loadString(key)
    }

    // MARK: - Delete
    func delete(_ key: SyncKey) throws {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key.rawValue,
            kSecAttrSynchronizable as String: true
        ]

        let status = SecItemDelete(query as CFDictionary)

        guard status == errSecSuccess || status == errSecItemNotFound else {
            throw iCloudKeychainError.deleteFailed(status)
        }
    }

    // MARK: - Check if backup exists
    func hasBackup() -> Bool {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: SyncKey.masterKeyBackup.rawValue,
            kSecReturnData as String: false,
            kSecAttrSynchronizable as String: true
        ]

        let status = SecItemCopyMatching(query as CFDictionary, nil)
        return status == errSecSuccess
    }

    // MARK: - Clear all synced data
    func clearAll() throws {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrSynchronizable as String: true
        ]

        let status = SecItemDelete(query as CFDictionary)

        guard status == errSecSuccess || status == errSecItemNotFound else {
            throw iCloudKeychainError.deleteFailed(status)
        }
    }
}

// MARK: - iCloud Keychain Error
enum iCloudKeychainError: LocalizedError {
    case saveFailed(OSStatus)
    case loadFailed(OSStatus)
    case deleteFailed(OSStatus)
    case itemNotFound
    case invalidData
    case encodingFailed
    case decodingFailed
    case iCloudNotAvailable

    var errorDescription: String? {
        switch self {
        case .saveFailed(let status):
            return "Failed to save to iCloud Keychain: \(status)"
        case .loadFailed(let status):
            return "Failed to load from iCloud Keychain: \(status)"
        case .deleteFailed(let status):
            return "Failed to delete from iCloud Keychain: \(status)"
        case .itemNotFound:
            return "Item not found in iCloud Keychain"
        case .invalidData:
            return "Invalid data in iCloud Keychain"
        case .encodingFailed:
            return "Failed to encode data"
        case .decodingFailed:
            return "Failed to decode data"
        case .iCloudNotAvailable:
            return "iCloud is not available"
        }
    }
}

// MARK: - Backup Operations
extension iCloudKeychainManager {
    /// Backup master key to iCloud (encrypted with recovery key)
    func backupMasterKey(encryptedKey: Data, salt: Data) async throws {
        try save(encryptedKey, for: .masterKeyBackup)
        try save(salt, for: .masterKeySaltBackup)
    }

    /// Restore master key from iCloud
    func restoreMasterKey() async throws -> (encryptedKey: Data, salt: Data) {
        let encryptedKey = try load(.masterKeyBackup)
        let salt = try load(.masterKeySaltBackup)
        return (encryptedKey, salt)
    }

    /// Generate and store recovery key
    func generateRecoveryKey() async throws -> String {
        // Generate a random 32-character recovery key
        var bytes = [UInt8](repeating: 0, count: 24)
        _ = SecRandomCopyBytes(kSecRandomDefault, bytes.count, &bytes)

        // Format as groups of 5 characters for readability
        let base32 = Data(bytes).base64EncodedString()
            .replacingOccurrences(of: "+", with: "")
            .replacingOccurrences(of: "/", with: "")
            .replacingOccurrences(of: "=", with: "")
            .prefix(25)

        let recoveryKey = String(base32)
            .enumerated()
            .map { $0.offset > 0 && $0.offset % 5 == 0 ? "-\($0.element)" : String($0.element) }
            .joined()

        try save(recoveryKey, for: .recoveryKey)
        return recoveryKey
    }

    /// Verify recovery key
    func verifyRecoveryKey(_ key: String) async -> Bool {
        guard let storedKey = loadStringOptional(.recoveryKey) else {
            return false
        }
        return key.replacingOccurrences(of: "-", with: "") ==
               storedKey.replacingOccurrences(of: "-", with: "")
    }
}
