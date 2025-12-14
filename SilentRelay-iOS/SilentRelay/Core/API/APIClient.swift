//
//  APIClient.swift
//  SilentRelay
//
//  HTTP client with certificate pinning matching web-new/src/core/api/client.ts
//

import Foundation
import CryptoKit

// MARK: - API Client
actor APIClient {
    // MARK: - Singleton
    static let shared = APIClient()

    // MARK: - Configuration
    private let baseURL: URL
    private let session: URLSession
    private var authToken: String?
    private var refreshToken: String?

    // Certificate pinning delegate
    private let sessionDelegate: CertificatePinningDelegate

    // MARK: - Init
    private init() {
        // Configure base URL from environment or default
        let urlString = ProcessInfo.processInfo.environment["API_BASE_URL"] ?? "https://api.silentrelay.com"
        self.baseURL = URL(string: urlString)!

        // Create certificate pinning delegate
        self.sessionDelegate = CertificatePinningDelegate()

        // Configure URLSession with certificate pinning
        let configuration = URLSessionConfiguration.default
        configuration.timeoutIntervalForRequest = 30
        configuration.timeoutIntervalForResource = 60
        configuration.httpAdditionalHeaders = [
            "Content-Type": "application/json",
            "Accept": "application/json"
        ]

        self.session = URLSession(
            configuration: configuration,
            delegate: sessionDelegate,
            delegateQueue: nil
        )
    }

    // MARK: - Token Management
    func setAuthToken(_ token: String?) {
        self.authToken = token
    }

    func setRefreshToken(_ token: String?) {
        self.refreshToken = token
    }

    func getAuthToken() -> String? {
        return authToken
    }

    // MARK: - Request Methods

    /// Perform a GET request
    func get<T: Decodable>(_ endpoint: APIEndpoint) async throws -> T {
        return try await request(endpoint, method: "GET")
    }

    /// Perform a POST request with body
    func post<T: Decodable, B: Encodable>(_ endpoint: APIEndpoint, body: B) async throws -> T {
        return try await request(endpoint, method: "POST", body: body)
    }

    /// Perform a POST request without expecting a response body
    func post<B: Encodable>(_ endpoint: APIEndpoint, body: B) async throws {
        let _: EmptyResponse = try await request(endpoint, method: "POST", body: body)
    }

    /// Perform a PATCH request with body
    func patch<T: Decodable, B: Encodable>(_ endpoint: APIEndpoint, body: B) async throws -> T {
        return try await request(endpoint, method: "PATCH", body: body)
    }

    /// Perform a DELETE request
    func delete(_ endpoint: APIEndpoint) async throws {
        let _: EmptyResponse = try await request(endpoint, method: "DELETE")
    }

    // MARK: - Core Request Method
    private func request<T: Decodable, B: Encodable>(
        _ endpoint: APIEndpoint,
        method: String,
        body: B? = nil as EmptyBody?
    ) async throws -> T {
        // Build URL
        let url = baseURL.appendingPathComponent(endpoint.path)

        // Create request
        var request = URLRequest(url: url)
        request.httpMethod = method

        // Add auth header if available
        if let token = authToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }

        // Add body if provided
        if let body = body, !(body is EmptyBody) {
            let encoder = JSONEncoder()
            encoder.keyEncodingStrategy = .convertToSnakeCase
            request.httpBody = try encoder.encode(body)
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        }

        // Perform request
        let (data, response) = try await session.data(for: request)

        // Validate response
        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }

        // Handle HTTP errors
        if !(200...299).contains(httpResponse.statusCode) {
            throw try parseError(data: data, statusCode: httpResponse.statusCode)
        }

        // Decode response
        if T.self == EmptyResponse.self {
            return EmptyResponse() as! T
        }

        let decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        decoder.dateDecodingStrategy = .custom { decoder in
            let container = try decoder.singleValueContainer()
            // Try ISO8601 first
            if let dateString = try? container.decode(String.self) {
                let formatter = ISO8601DateFormatter()
                formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
                if let date = formatter.date(from: dateString) {
                    return date
                }
                // Try without fractional seconds
                formatter.formatOptions = [.withInternetDateTime]
                if let date = formatter.date(from: dateString) {
                    return date
                }
            }
            // Try Unix timestamp (milliseconds)
            if let timestamp = try? container.decode(Double.self) {
                return Date(timeIntervalSince1970: timestamp / 1000)
            }
            throw DecodingError.dataCorruptedError(
                in: container,
                debugDescription: "Cannot decode date"
            )
        }

        do {
            return try decoder.decode(T.self, from: data)
        } catch {
            print("Decoding error: \(error)")
            throw APIError.decodingError(error)
        }
    }

    // MARK: - Error Parsing
    private func parseError(data: Data, statusCode: Int) throws -> APIError {
        // Try to decode error response from server
        if let errorResponse = try? JSONDecoder().decode(APIErrorResponse.self, from: data) {
            return APIError.serverError(
                code: errorResponse.code ?? statusCodeToErrorCode(statusCode),
                message: errorResponse.message ?? "Unknown error"
            )
        }

        // Map status code to error
        return APIError.httpError(statusCode: statusCode)
    }

    private func statusCodeToErrorCode(_ statusCode: Int) -> String {
        switch statusCode {
        case 400: return "BAD_REQUEST"
        case 401: return "UNAUTHORIZED"
        case 403: return "FORBIDDEN"
        case 404: return "NOT_FOUND"
        case 409: return "CONFLICT"
        case 422: return "VALIDATION_ERROR"
        case 429: return "RATE_LIMITED"
        case 500: return "INTERNAL_ERROR"
        case 502: return "BAD_GATEWAY"
        case 503: return "SERVICE_UNAVAILABLE"
        default: return "UNKNOWN"
        }
    }

    // MARK: - Token Refresh
    func refreshAuthToken() async throws {
        guard let refreshToken = refreshToken else {
            throw APIError.noRefreshToken
        }

        struct RefreshRequest: Encodable {
            let refreshToken: String
        }

        struct RefreshResponse: Decodable {
            let accessToken: String
            let refreshToken: String?
        }

        // Temporarily clear auth token to avoid using expired token
        let oldToken = authToken
        self.authToken = nil

        do {
            let response: RefreshResponse = try await post(
                .authRefresh,
                body: RefreshRequest(refreshToken: refreshToken)
            )

            self.authToken = response.accessToken
            if let newRefreshToken = response.refreshToken {
                self.refreshToken = newRefreshToken
            }
        } catch {
            // Restore old token on failure
            self.authToken = oldToken
            throw error
        }
    }
}

// MARK: - Empty Types
private struct EmptyBody: Encodable {}
struct EmptyResponse: Decodable {}

// MARK: - Error Response
private struct APIErrorResponse: Decodable {
    let code: String?
    let message: String?
    let error: String?
}

// MARK: - Certificate Pinning Delegate
final class CertificatePinningDelegate: NSObject, URLSessionDelegate, @unchecked Sendable {

    // SHA-256 hashes of your server's public key certificates
    // TODO: Replace with actual certificate hashes before production
    private let pinnedCertificateHashes: Set<String> = [
        // Add your certificate SHA-256 hashes here
        // Example: "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB="
    ]

    // Whether to enforce certificate pinning (disable for development)
    private let enforcePinning: Bool

    override init() {
        #if DEBUG
        self.enforcePinning = false
        #else
        self.enforcePinning = true
        #endif
        super.init()
    }

    func urlSession(
        _ session: URLSession,
        didReceive challenge: URLAuthenticationChallenge,
        completionHandler: @escaping (URLSession.AuthChallengeDisposition, URLCredential?) -> Void
    ) {
        guard challenge.protectionSpace.authenticationMethod == NSURLAuthenticationMethodServerTrust,
              let serverTrust = challenge.protectionSpace.serverTrust else {
            completionHandler(.performDefaultHandling, nil)
            return
        }

        // Skip pinning in development
        guard enforcePinning else {
            completionHandler(.useCredential, URLCredential(trust: serverTrust))
            return
        }

        // Get the certificate chain
        let certificateCount = SecTrustGetCertificateCount(serverTrust)
        guard certificateCount > 0 else {
            completionHandler(.cancelAuthenticationChallenge, nil)
            return
        }

        // Check each certificate in the chain
        for index in 0..<certificateCount {
            guard let certificate = SecTrustCopyCertificateChain(serverTrust)?[index] as? SecCertificate else {
                continue
            }

            // Get certificate data and compute SHA-256 hash
            let certificateData = SecCertificateCopyData(certificate) as Data
            let hash = SHA256.hash(data: certificateData)
            let hashString = Data(hash).base64EncodedString()

            if pinnedCertificateHashes.contains(hashString) {
                completionHandler(.useCredential, URLCredential(trust: serverTrust))
                return
            }
        }

        // No matching certificate found
        print("Certificate pinning failed - no matching certificate")
        completionHandler(.cancelAuthenticationChallenge, nil)
    }
}
