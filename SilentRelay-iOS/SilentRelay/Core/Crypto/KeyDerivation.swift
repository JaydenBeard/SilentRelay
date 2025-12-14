//
//  KeyDerivation.swift
//  SilentRelay
//
//  PBKDF2 key derivation matching web-new/src/core/crypto/signal.ts
//  OWASP minimum: 600,000 iterations
//

import Foundation
import CryptoKit
import CommonCrypto

// MARK: - Key Derivation Constants
enum KeyDerivationConstants {
    /// OWASP minimum recommendation for PBKDF2-HMAC-SHA256
    /// @see https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
    static let pbkdf2Iterations: UInt32 = 600_000

    /// Minimum allowed iterations (security enforcement)
    static let minPbkdf2Iterations: UInt32 = 600_000

    /// Salt length in bytes (128 bits)
    static let saltLength = 16

    /// Derived key length in bytes (256 bits for AES-256)
    static let keyLength = 32

    /// IV length for AES-GCM (96 bits)
    static let ivLength = 12

    /// Authentication tag length for AES-GCM (128 bits)
    static let tagLength = 16
}

// MARK: - Key Derivation
enum KeyDerivation {

    // MARK: - PBKDF2 Key Derivation

    /// Derive a master encryption key from a PIN using PBKDF2-HMAC-SHA256
    ///
    /// Security Notes:
    /// - Uses 600,000 iterations (OWASP minimum recommendation)
    /// - Derives 256-bit keys for AES-256-GCM encryption
    /// - Validates iteration count meets minimum security requirements
    ///
    /// - Parameters:
    ///   - pin: User PIN/password for key derivation
    ///   - salt: Random salt (should be 16+ bytes)
    ///   - iterations: Iteration count (must be >= 600,000)
    /// - Returns: 256-bit master encryption key
    /// - Throws: KeyDerivationError if iterations are below minimum
    static func deriveMasterKey(
        from pin: String,
        salt: Data,
        iterations: UInt32 = KeyDerivationConstants.pbkdf2Iterations
    ) throws -> Data {
        // Security validation: Ensure iterations meet OWASP minimum
        guard iterations >= KeyDerivationConstants.minPbkdf2Iterations else {
            throw KeyDerivationError.iterationsTooLow(
                provided: iterations,
                minimum: KeyDerivationConstants.minPbkdf2Iterations
            )
        }

        guard let pinData = pin.data(using: .utf8) else {
            throw KeyDerivationError.invalidInput
        }

        var derivedKey = Data(count: KeyDerivationConstants.keyLength)

        let status = derivedKey.withUnsafeMutableBytes { derivedKeyBytes in
            salt.withUnsafeBytes { saltBytes in
                pinData.withUnsafeBytes { pinBytes in
                    CCKeyDerivationPBKDF(
                        CCPBKDFAlgorithm(kCCPBKDF2),
                        pinBytes.baseAddress?.assumingMemoryBound(to: Int8.self),
                        pinData.count,
                        saltBytes.baseAddress?.assumingMemoryBound(to: UInt8.self),
                        salt.count,
                        CCPseudoRandomAlgorithm(kCCPRFHmacAlgSHA256),
                        iterations,
                        derivedKeyBytes.baseAddress?.assumingMemoryBound(to: UInt8.self),
                        KeyDerivationConstants.keyLength
                    )
                }
            }
        }

        guard status == kCCSuccess else {
            throw KeyDerivationError.derivationFailed(status: status)
        }

        return derivedKey
    }

    // MARK: - Salt Generation

    /// Generate a cryptographically secure random salt
    /// - Returns: 16 bytes of random data
    static func generateSalt() -> Data {
        var salt = Data(count: KeyDerivationConstants.saltLength)
        _ = salt.withUnsafeMutableBytes { bytes in
            SecRandomCopyBytes(kSecRandomDefault, KeyDerivationConstants.saltLength, bytes.baseAddress!)
        }
        return salt
    }

    // MARK: - AES-256-GCM Encryption

    /// Encrypt data using AES-256-GCM
    /// - Parameters:
    ///   - data: Plaintext data to encrypt
    ///   - key: 256-bit encryption key
    /// - Returns: Tuple of (ciphertext, IV, tag)
    static func encrypt(_ data: Data, with key: Data) throws -> (ciphertext: Data, iv: Data, tag: Data) {
        guard key.count == KeyDerivationConstants.keyLength else {
            throw KeyDerivationError.invalidKeyLength
        }

        let symmetricKey = SymmetricKey(data: key)
        let nonce = AES.GCM.Nonce()

        let sealedBox = try AES.GCM.seal(data, using: symmetricKey, nonce: nonce)

        guard let combined = sealedBox.combined else {
            throw KeyDerivationError.encryptionFailed
        }

        // Extract components: nonce (12) + ciphertext + tag (16)
        let nonceData = Data(nonce)
        let ciphertextAndTag = combined.dropFirst(nonceData.count)
        let ciphertext = ciphertextAndTag.dropLast(KeyDerivationConstants.tagLength)
        let tag = ciphertextAndTag.suffix(KeyDerivationConstants.tagLength)

        return (Data(ciphertext), nonceData, Data(tag))
    }

    /// Encrypt data and return as combined format (iv:ciphertext:tag in base64)
    static func encryptToString(_ string: String, with key: Data) throws -> String {
        guard let data = string.data(using: .utf8) else {
            throw KeyDerivationError.invalidInput
        }

        let (ciphertext, iv, tag) = try encrypt(data, with: key)

        // Combine: iv + ciphertext + tag
        var combined = Data()
        combined.append(iv)
        combined.append(ciphertext)
        combined.append(tag)

        return combined.base64EncodedString()
    }

    // MARK: - AES-256-GCM Decryption

    /// Decrypt data using AES-256-GCM
    /// - Parameters:
    ///   - ciphertext: Encrypted data
    ///   - key: 256-bit encryption key
    ///   - iv: Initialization vector (12 bytes)
    ///   - tag: Authentication tag (16 bytes)
    /// - Returns: Decrypted plaintext
    static func decrypt(ciphertext: Data, with key: Data, iv: Data, tag: Data) throws -> Data {
        guard key.count == KeyDerivationConstants.keyLength else {
            throw KeyDerivationError.invalidKeyLength
        }

        guard iv.count == KeyDerivationConstants.ivLength else {
            throw KeyDerivationError.invalidIVLength
        }

        let symmetricKey = SymmetricKey(data: key)

        // Combine ciphertext and tag for CryptoKit
        var combined = Data()
        combined.append(iv)
        combined.append(ciphertext)
        combined.append(tag)

        let sealedBox = try AES.GCM.SealedBox(combined: combined)
        let plaintext = try AES.GCM.open(sealedBox, using: symmetricKey)

        return plaintext
    }

    /// Decrypt from combined base64 format
    static func decryptFromString(_ base64String: String, with key: Data) throws -> String {
        guard let combined = Data(base64Encoded: base64String) else {
            throw KeyDerivationError.invalidInput
        }

        // Parse: iv (12) + ciphertext + tag (16)
        guard combined.count > KeyDerivationConstants.ivLength + KeyDerivationConstants.tagLength else {
            throw KeyDerivationError.invalidInput
        }

        let iv = combined.prefix(KeyDerivationConstants.ivLength)
        let tag = combined.suffix(KeyDerivationConstants.tagLength)
        let ciphertext = combined.dropFirst(KeyDerivationConstants.ivLength).dropLast(KeyDerivationConstants.tagLength)

        let plaintext = try decrypt(ciphertext: Data(ciphertext), with: key, iv: Data(iv), tag: Data(tag))

        guard let string = String(data: plaintext, encoding: .utf8) else {
            throw KeyDerivationError.invalidInput
        }

        return string
    }

    // MARK: - Key Validation

    /// Validate that a derived key can decrypt test data
    /// Used to verify PIN correctness without storing the PIN
    static func validateKey(_ key: Data, against encryptedTestData: String) -> Bool {
        do {
            _ = try decryptFromString(encryptedTestData, with: key)
            return true
        } catch {
            return false
        }
    }
}

// MARK: - Key Derivation Error
enum KeyDerivationError: LocalizedError {
    case iterationsTooLow(provided: UInt32, minimum: UInt32)
    case invalidInput
    case invalidKeyLength
    case invalidIVLength
    case derivationFailed(status: Int32)
    case encryptionFailed
    case decryptionFailed

    var errorDescription: String? {
        switch self {
        case .iterationsTooLow(let provided, let minimum):
            return "PBKDF2 iterations too low: \(provided). Minimum required: \(minimum)"
        case .invalidInput:
            return "Invalid input data"
        case .invalidKeyLength:
            return "Key must be 32 bytes (256 bits)"
        case .invalidIVLength:
            return "IV must be 12 bytes (96 bits)"
        case .derivationFailed(let status):
            return "Key derivation failed with status: \(status)"
        case .encryptionFailed:
            return "Encryption failed"
        case .decryptionFailed:
            return "Decryption failed"
        }
    }
}

// MARK: - Secure Memory
extension Data {
    /// Zero out sensitive data in memory
    mutating func secureZero() {
        guard count > 0 else { return }
        withUnsafeMutableBytes { bytes in
            memset(bytes.baseAddress, 0, count)
        }
    }
}
