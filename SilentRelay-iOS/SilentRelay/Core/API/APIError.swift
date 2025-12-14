//
//  APIError.swift
//  SilentRelay
//
//  API error types matching web-new/src/lib/errors.ts
//

import Foundation

// MARK: - API Error
enum APIError: LocalizedError, Equatable {
    // Network errors
    case noConnection
    case timeout
    case invalidResponse

    // Auth errors
    case unauthorized
    case noRefreshToken
    case sessionExpired
    case invalidCredentials

    // HTTP errors
    case httpError(statusCode: Int)
    case serverError(code: String, message: String)

    // Client errors
    case decodingError(Error)
    case encodingError(Error)
    case validationError(String)

    // Crypto errors
    case encryptionFailed
    case decryptionFailed
    case keyGenerationFailed

    // Generic
    case unknown(String)

    // MARK: - Error Description
    var errorDescription: String? {
        switch self {
        case .noConnection:
            return "No internet connection"
        case .timeout:
            return "Request timed out"
        case .invalidResponse:
            return "Invalid response from server"
        case .unauthorized:
            return "Authentication required"
        case .noRefreshToken:
            return "No refresh token available"
        case .sessionExpired:
            return "Your session has expired. Please sign in again."
        case .invalidCredentials:
            return "Invalid credentials"
        case .httpError(let statusCode):
            return "HTTP error: \(statusCode)"
        case .serverError(_, let message):
            return message
        case .decodingError(let error):
            return "Failed to parse response: \(error.localizedDescription)"
        case .encodingError(let error):
            return "Failed to encode request: \(error.localizedDescription)"
        case .validationError(let message):
            return message
        case .encryptionFailed:
            return "Failed to encrypt data"
        case .decryptionFailed:
            return "Failed to decrypt data"
        case .keyGenerationFailed:
            return "Failed to generate encryption keys"
        case .unknown(let message):
            return message
        }
    }

    // MARK: - Error Code
    var errorCode: String {
        switch self {
        case .noConnection: return "NET-001"
        case .timeout: return "NET-002"
        case .invalidResponse: return "NET-003"
        case .unauthorized: return "AUTH-001"
        case .noRefreshToken: return "AUTH-002"
        case .sessionExpired: return "AUTH-005"
        case .invalidCredentials: return "AUTH-003"
        case .httpError(let statusCode): return "HTTP-\(statusCode)"
        case .serverError(let code, _): return code
        case .decodingError: return "CLI-001"
        case .encodingError: return "CLI-002"
        case .validationError: return "CLI-003"
        case .encryptionFailed: return "CRYPTO-001"
        case .decryptionFailed: return "CRYPTO-003"
        case .keyGenerationFailed: return "CRYPTO-002"
        case .unknown: return "SYS-001"
        }
    }

    // MARK: - Equatable
    static func == (lhs: APIError, rhs: APIError) -> Bool {
        switch (lhs, rhs) {
        case (.noConnection, .noConnection),
             (.timeout, .timeout),
             (.invalidResponse, .invalidResponse),
             (.unauthorized, .unauthorized),
             (.noRefreshToken, .noRefreshToken),
             (.sessionExpired, .sessionExpired),
             (.invalidCredentials, .invalidCredentials),
             (.encryptionFailed, .encryptionFailed),
             (.decryptionFailed, .decryptionFailed),
             (.keyGenerationFailed, .keyGenerationFailed):
            return true
        case (.httpError(let a), .httpError(let b)):
            return a == b
        case (.serverError(let c1, let m1), .serverError(let c2, let m2)):
            return c1 == c2 && m1 == m2
        case (.validationError(let a), .validationError(let b)):
            return a == b
        case (.unknown(let a), .unknown(let b)):
            return a == b
        default:
            return false
        }
    }

    // MARK: - Retry Logic
    var isRetryable: Bool {
        switch self {
        case .noConnection, .timeout:
            return true
        case .httpError(let statusCode):
            return statusCode >= 500 || statusCode == 429
        default:
            return false
        }
    }

    // MARK: - User-Facing
    var isUserError: Bool {
        switch self {
        case .invalidCredentials, .validationError:
            return true
        default:
            return false
        }
    }
}

// MARK: - Error Categories (matching web)
enum ErrorCategory: String {
    case auth = "AUTH"
    case network = "NET"
    case crypto = "CRYPTO"
    case message = "MSG"
    case media = "MEDIA"
    case call = "CALL"
    case system = "SYS"
}

// MARK: - Result Extension
extension Result where Failure == APIError {
    var errorMessage: String? {
        switch self {
        case .success:
            return nil
        case .failure(let error):
            return error.errorDescription
        }
    }
}
