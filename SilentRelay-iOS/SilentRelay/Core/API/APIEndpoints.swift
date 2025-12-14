//
//  APIEndpoints.swift
//  SilentRelay
//
//  API endpoint definitions matching web-new/src/core/api/client.ts
//

import Foundation

// MARK: - API Endpoints
enum APIEndpoint {
    // Auth
    case authRequestCode
    case authVerify
    case authRegister
    case authRefresh
    case authLogout

    // Users
    case usersMe
    case userById(id: String)
    case userKeys(id: String)
    case userPreKeys
    case userUpdateKeys
    case userProfile(id: String)
    case userSearch(query: String)
    case checkUsername(username: String)

    // Messages
    case messages(conversationId: String)

    // Media
    case mediaUploadUrl
    case mediaDownloadUrl(mediaId: String)

    // Settings
    case privacySettings

    // RTC
    case rtcTurnCredentials

    // Devices
    case devicesPushToken

    /// The path for this endpoint
    var path: String {
        switch self {
        // Auth
        case .authRequestCode:
            return "/api/v1/auth/request-code"
        case .authVerify:
            return "/api/v1/auth/verify"
        case .authRegister:
            return "/api/v1/auth/register"
        case .authRefresh:
            return "/api/v1/auth/refresh"
        case .authLogout:
            return "/api/v1/auth/logout"

        // Users
        case .usersMe:
            return "/api/v1/users/me"
        case .userById(let id):
            return "/api/v1/users/\(id)"
        case .userKeys(let id):
            return "/api/v1/users/\(id)/keys"
        case .userPreKeys:
            return "/api/v1/users/me/prekeys"
        case .userUpdateKeys:
            return "/api/v1/users/keys"
        case .userProfile(let id):
            return "/api/v1/users/\(id)/profile"
        case .userSearch(let query):
            return "/api/v1/users/search?q=\(query.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? query)"
        case .checkUsername(let username):
            return "/api/v1/users/check-username/\(username)"

        // Messages
        case .messages(let conversationId):
            return "/api/v1/messages/\(conversationId)"

        // Media
        case .mediaUploadUrl:
            return "/api/v1/media/upload-url"
        case .mediaDownloadUrl(let mediaId):
            return "/api/v1/media/\(mediaId)/download-url"

        // Settings
        case .privacySettings:
            return "/api/v1/settings/privacy"

        // RTC
        case .rtcTurnCredentials:
            return "/api/v1/rtc/turn-credentials"

        // Devices
        case .devicesPushToken:
            return "/api/v1/devices/push-token"
        }
    }
}

// MARK: - Request Bodies

/// Request verification code
struct RequestCodeRequest: Encodable {
    let phoneNumber: String

    enum CodingKeys: String, CodingKey {
        case phoneNumber = "phone_number"
    }
}

/// Verify code
struct VerifyCodeRequest: Encodable {
    let phoneNumber: String
    let code: String

    enum CodingKeys: String, CodingKey {
        case phoneNumber = "phone_number"
        case code
    }
}

struct VerifyCodeResponse: Decodable {
    let userExists: Bool
    let accessToken: String?
    let refreshToken: String?
    let user: User?

    enum CodingKeys: String, CodingKey {
        case userExists = "user_exists"
        case accessToken = "access_token"
        case refreshToken = "refresh_token"
        case user
    }
}

/// Register new user with Signal keys
struct RegisterRequest: Encodable {
    let phoneNumber: String
    let publicIdentityKey: String
    let publicSignedPrekey: String
    let signedPrekeySignature: String
    let oneTimePrekeys: [String]

    enum CodingKeys: String, CodingKey {
        case phoneNumber = "phone_number"
        case publicIdentityKey = "public_identity_key"
        case publicSignedPrekey = "public_signed_prekey"
        case signedPrekeySignature = "signed_prekey_signature"
        case oneTimePrekeys = "one_time_prekeys"
    }
}

struct RegisterResponse: Decodable {
    let accessToken: String
    let refreshToken: String
    let user: User

    enum CodingKeys: String, CodingKey {
        case accessToken = "access_token"
        case refreshToken = "refresh_token"
        case user
    }
}

/// Upload pre-keys
struct UploadPreKeysRequest: Encodable {
    let prekeys: [String]
}

/// Update profile
struct UpdateProfileRequest: Encodable {
    var username: String?
    var displayName: String?
    var avatar: String?

    enum CodingKeys: String, CodingKey {
        case username
        case displayName = "display_name"
        case avatar
    }
}

/// Upload URL request
struct UploadUrlRequest: Encodable {
    let fileName: String
    let mimeType: String
    let fileSize: Int

    enum CodingKeys: String, CodingKey {
        case fileName = "file_name"
        case mimeType = "mime_type"
        case fileSize = "file_size"
    }
}

struct UploadUrlResponse: Decodable {
    let uploadUrl: String
    let mediaId: String

    enum CodingKeys: String, CodingKey {
        case uploadUrl = "upload_url"
        case mediaId = "media_id"
    }
}

struct DownloadUrlResponse: Decodable {
    let downloadUrl: String

    enum CodingKeys: String, CodingKey {
        case downloadUrl = "download_url"
    }
}

/// Messages pagination
struct MessagesResponse: Decodable {
    let messages: [Message]
    let hasMore: Bool
    let nextCursor: String?

    enum CodingKeys: String, CodingKey {
        case messages
        case hasMore = "has_more"
        case nextCursor = "next_cursor"
    }
}

/// TURN credentials
struct TurnCredentialsResponse: Decodable {
    let urls: [String]
    let username: String
    let credential: String
    let ttl: Int
}

/// Push token registration
struct PushTokenRequest: Encodable {
    let deviceId: String
    let pushToken: String
    let voipToken: String?
    let platform: String

    enum CodingKeys: String, CodingKey {
        case deviceId = "device_id"
        case pushToken = "push_token"
        case voipToken = "voip_token"
        case platform
    }
}

/// User search response
struct UserSearchResponse: Decodable {
    let users: [User]
}

/// Check username response
struct CheckUsernameResponse: Decodable {
    let available: Bool
}
