//
//  SignalManager.swift
//  SilentRelay
//
//  Signal Protocol wrapper using libsignal-client
//  Matches web-new/src/core/crypto/signal.ts functionality
//

import Foundation
import CryptoKit

// MARK: - Signal Manager
/// Manages Signal Protocol encryption using libsignal-client
///
/// This class provides:
/// - Identity key generation and management
/// - Pre-key bundle generation for X3DH key exchange
/// - Session creation and message encryption/decryption
/// - PIN-based key protection with PBKDF2
///
/// Note: Requires libsignal-client SPM package to be added to the project
actor SignalManager {
    // MARK: - Singleton
    static let shared = SignalManager()

    // MARK: - State
    private var isInitialized = false
    private var masterKey: Data?
    private var encryptionEnabled = false

    // Registration info
    private var registrationId: UInt32 = 0
    private var deviceId: UInt32 = 1

    // Key storage
    private let keychainManager = KeychainManager.shared

    // Session cache (LRU with max 100 sessions)
    private var sessionCache: [String: SessionData] = [:]
    private var sessionAccessOrder: [String] = []
    private let maxSessionCacheSize = 100

    // MARK: - Init
    private init() {}

    // MARK: - Initialization

    /// Initialize the Signal Protocol manager
    func initialize() async throws {
        guard !isInitialized else { return }

        // Check if encryption is enabled
        encryptionEnabled = await keychainManager.exists(.masterKeySalt)

        // Load registration and device IDs if they exist
        if let regId = await keychainManager.loadIntOptional(.registrationId) {
            registrationId = UInt32(regId)
        }

        if let devId = await keychainManager.loadIntOptional(.signalDeviceId) {
            deviceId = UInt32(devId)
        }

        isInitialized = true
    }

    // MARK: - Encryption Setup

    /// Set up encryption with a user PIN
    /// - Parameter pin: User's PIN (minimum 6 characters)
    func setupEncryption(pin: String) async throws {
        guard pin.count >= 6 else {
            throw SignalError.pinTooShort
        }

        // Generate random salt
        let salt = KeyDerivation.generateSalt()

        // Derive master key from PIN
        masterKey = try KeyDerivation.deriveMasterKey(from: pin, salt: salt)

        // Store salt in keychain
        try await keychainManager.save(salt, for: .masterKeySalt)

        // Generate new identity if needed
        if registrationId == 0 {
            try await generateIdentity()
        }

        encryptionEnabled = true
    }

    /// Unlock encryption with PIN
    /// - Parameter pin: User's PIN
    /// - Returns: True if PIN is correct
    func unlockWithPIN(_ pin: String) async throws -> Bool {
        guard encryptionEnabled else {
            throw SignalError.encryptionNotEnabled
        }

        guard pin.count >= 6 else {
            throw SignalError.pinTooShort
        }

        // Load salt
        let salt = try await keychainManager.load(.masterKeySalt)

        // Derive master key
        let derivedKey = try KeyDerivation.deriveMasterKey(from: pin, salt: salt)

        // Verify by trying to decrypt identity key
        let encryptedIdentity = try await keychainManager.load(.encryptedIdentityKey)
        let iv = try await keychainManager.load(.identityKeyIV)

        do {
            // Try to decrypt - if it works, PIN is correct
            _ = try KeyDerivation.decrypt(
                ciphertext: encryptedIdentity,
                with: derivedKey,
                iv: iv,
                tag: Data() // Tag is appended to ciphertext
            )

            masterKey = derivedKey
            return true
        } catch {
            return false
        }
    }

    /// Check if encryption is enabled
    func isEncryptionEnabled() -> Bool {
        encryptionEnabled
    }

    /// Check if keys are unlocked
    func isUnlocked() -> Bool {
        masterKey != nil
    }

    // MARK: - Identity Generation

    /// Generate a new Signal Protocol identity
    private func generateIdentity() async throws {
        // Generate registration ID (1-16380)
        registrationId = UInt32.random(in: 1...16380)

        // Device ID starts at 1
        deviceId = 1

        // Store IDs
        try await keychainManager.save(Int(registrationId), for: .registrationId)
        try await keychainManager.save(Int(deviceId), for: .signalDeviceId)

        // Note: Actual key generation requires libsignal-client
        // This is a placeholder for the structure
    }

    /// Get registration ID
    func getRegistrationId() async throws -> UInt32 {
        try await ensureInitialized()
        return registrationId
    }

    /// Get device ID
    func getDeviceId() async throws -> UInt32 {
        try await ensureInitialized()
        return deviceId
    }

    // MARK: - Pre-Key Bundle Generation

    /// Generate keys for registration
    /// Returns public keys to send to server
    func generateRegistrationKeys() async throws -> RegistrationKeys {
        try await ensureInitialized()
        guard isUnlocked() else {
            throw SignalError.keysLocked
        }

        // Note: This requires libsignal-client for actual implementation
        // Placeholder structure showing what would be generated

        return RegistrationKeys(
            publicIdentityKey: "", // Base64 encoded Curve25519 public key
            publicSignedPrekey: "", // Base64 encoded signed pre-key
            signedPrekeySignature: "", // Ed25519 signature
            oneTimePrekeys: [] // Array of one-time pre-keys
        )
    }

    /// Generate additional one-time pre-keys
    func generatePreKeys(count: Int) async throws -> [String] {
        try await ensureInitialized()
        guard isUnlocked() else {
            throw SignalError.keysLocked
        }

        // Note: Requires libsignal-client
        return []
    }

    // MARK: - Session Management

    /// Create a session with a recipient using their pre-key bundle
    func createSession(
        recipientId: String,
        deviceId: UInt32,
        bundle: PreKeyBundle
    ) async throws {
        try await ensureInitialized()
        guard isUnlocked() else {
            throw SignalError.keysLocked
        }

        let sessionKey = "\(recipientId):\(deviceId)"

        // Note: Actual session creation requires libsignal-client
        // This stores session data structure

        let sessionData = SessionData(
            recipientId: recipientId,
            deviceId: deviceId,
            createdAt: Date(),
            lastUsed: Date()
        )

        // LRU cache management
        if sessionCache.count >= maxSessionCacheSize && sessionCache[sessionKey] == nil {
            if let oldest = sessionAccessOrder.first {
                sessionCache.removeValue(forKey: oldest)
                sessionAccessOrder.removeFirst()
            }
        }

        sessionCache[sessionKey] = sessionData
        updateSessionAccessOrder(sessionKey)
    }

    /// Check if session exists with recipient
    func hasSession(recipientId: String, deviceId: UInt32) async -> Bool {
        let sessionKey = "\(recipientId):\(deviceId)"
        return sessionCache[sessionKey] != nil
    }

    /// Delete session with recipient
    func deleteSession(recipientId: String, deviceId: UInt32) async {
        let sessionKey = "\(recipientId):\(deviceId)"
        sessionCache.removeValue(forKey: sessionKey)
        sessionAccessOrder.removeAll { $0 == sessionKey }
    }

    // MARK: - Message Encryption

    /// Encrypt a message for a recipient
    func encryptMessage(
        recipientId: String,
        deviceId: UInt32,
        plaintext: String
    ) async throws -> EncryptedMessageData {
        try await ensureInitialized()
        guard isUnlocked() else {
            throw SignalError.keysLocked
        }

        let sessionKey = "\(recipientId):\(deviceId)"

        guard sessionCache[sessionKey] != nil else {
            throw SignalError.noSession
        }

        // Update session access order
        updateSessionAccessOrder(sessionKey)

        // Note: Actual encryption requires libsignal-client
        // Placeholder showing structure

        return EncryptedMessageData(
            ciphertext: Data(), // Encrypted message bytes
            messageType: .whisper // or .prekey for first message
        )
    }

    /// Decrypt a received message
    func decryptMessage(
        senderId: String,
        deviceId: UInt32,
        ciphertext: Data,
        messageType: SignalMessageType
    ) async throws -> String {
        try await ensureInitialized()
        guard isUnlocked() else {
            throw SignalError.keysLocked
        }

        let sessionKey = "\(senderId):\(deviceId)"

        // For prekey messages, session will be created automatically
        if messageType == .prekey && sessionCache[sessionKey] == nil {
            // Create inbound session from prekey message
            let sessionData = SessionData(
                recipientId: senderId,
                deviceId: deviceId,
                createdAt: Date(),
                lastUsed: Date()
            )
            sessionCache[sessionKey] = sessionData
        }

        guard sessionCache[sessionKey] != nil else {
            throw SignalError.noSession
        }

        updateSessionAccessOrder(sessionKey)

        // Note: Actual decryption requires libsignal-client
        return ""
    }

    // MARK: - Key Access

    /// Get identity public key for sharing
    func getIdentityPublicKey() async throws -> Data? {
        try await ensureInitialized()
        guard isUnlocked() else { return nil }

        // Note: Requires libsignal-client
        return nil
    }

    /// Get public keys formatted for server upload
    func getPublicKeysForServer() async throws -> ServerPublicKeys? {
        try await ensureInitialized()
        guard isUnlocked() else { return nil }

        // Note: Requires libsignal-client
        return nil
    }

    // MARK: - Clear Data

    /// Clear all stored data (logout/reset)
    func clear() async throws {
        // Clear session cache
        sessionCache.removeAll()
        sessionAccessOrder.removeAll()

        // Clear keychain data
        try await keychainManager.delete(.masterKeySalt)
        try await keychainManager.delete(.encryptedIdentityKey)
        try await keychainManager.delete(.identityKeyIV)
        try await keychainManager.delete(.registrationId)
        try await keychainManager.delete(.signalDeviceId)

        // Reset state
        masterKey = nil
        encryptionEnabled = false
        registrationId = 0
        deviceId = 1
        isInitialized = false
    }

    // MARK: - Private Helpers

    private func ensureInitialized() async throws {
        if !isInitialized {
            try await initialize()
        }
    }

    private func updateSessionAccessOrder(_ key: String) {
        sessionAccessOrder.removeAll { $0 == key }
        sessionAccessOrder.append(key)
    }
}

// MARK: - Supporting Types

struct RegistrationKeys {
    let publicIdentityKey: String
    let publicSignedPrekey: String
    let signedPrekeySignature: String
    let oneTimePrekeys: [String]
}

struct PreKeyBundle {
    let registrationId: UInt32
    let deviceId: UInt32
    let identityKey: Data
    let signedPreKeyId: UInt32
    let signedPreKey: Data
    let signedPreKeySignature: Data
    let preKeyId: UInt32?
    let preKey: Data?
}

struct SessionData {
    let recipientId: String
    let deviceId: UInt32
    let createdAt: Date
    var lastUsed: Date
}

struct EncryptedMessageData {
    let ciphertext: Data
    let messageType: SignalMessageType
}

enum SignalMessageType: String {
    case prekey
    case whisper
}

struct ServerPublicKeys {
    let publicIdentityKey: String
    let publicSignedPrekey: String
    let signedPrekeySignature: String
}

// MARK: - Signal Error
enum SignalError: LocalizedError {
    case notInitialized
    case encryptionNotEnabled
    case keysLocked
    case pinTooShort
    case noSession
    case encryptionFailed
    case decryptionFailed
    case invalidKey
    case sessionCreationFailed

    var errorDescription: String? {
        switch self {
        case .notInitialized:
            return "Signal Protocol not initialized"
        case .encryptionNotEnabled:
            return "Encryption is not enabled"
        case .keysLocked:
            return "Keys are locked - unlock with PIN first"
        case .pinTooShort:
            return "PIN must be at least 6 characters"
        case .noSession:
            return "No session exists for this recipient"
        case .encryptionFailed:
            return "Failed to encrypt message"
        case .decryptionFailed:
            return "Failed to decrypt message"
        case .invalidKey:
            return "Invalid encryption key"
        case .sessionCreationFailed:
            return "Failed to create session"
        }
    }
}
