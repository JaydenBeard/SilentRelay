//
//  PushManager.swift
//  SilentRelay
//
//  Push notification management (APNs + PushKit)
//

import Foundation

// MARK: - Push Manager
actor PushManager {
    // MARK: - Singleton
    static let shared = PushManager()

    // MARK: - Properties
    private let apiClient = APIClient.shared
    private let keychainManager = KeychainManager.shared

    private var pushToken: String?
    private var voipToken: String?

    private init() {}

    // MARK: - Register Push Token
    func registerPushToken(_ token: String, platform: String) async {
        self.pushToken = token

        do {
            let deviceId = try await keychainManager.getOrCreateDeviceId()

            try await apiClient.post(
                .devicesPushToken,
                body: PushTokenRequest(
                    deviceId: deviceId,
                    pushToken: token,
                    voipToken: voipToken,
                    platform: platform
                )
            )
        } catch {
            print("Failed to register push token: \(error)")
        }
    }

    // MARK: - Register VoIP Token
    func registerVoIPToken(_ token: String) async {
        self.voipToken = token

        // If we already have a push token, update the registration
        if let pushToken = pushToken {
            await registerPushToken(pushToken, platform: "ios")
        }
    }

    // MARK: - Handle Push Payload
    func handlePushPayload(_ payload: [AnyHashable: Any]) async {
        // Extract notification type
        guard let notificationType = payload["type"] as? String else {
            return
        }

        switch notificationType {
        case "message":
            // Handle message notification
            if let conversationId = payload["conversation_id"] as? String {
                NotificationCenter.default.post(
                    name: .openConversation,
                    object: nil,
                    userInfo: ["conversationId": conversationId]
                )
            }

        case "call":
            // Call notifications are handled by PushKit/CallKit
            break

        default:
            break
        }
    }
}
