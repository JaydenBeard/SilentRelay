//
//  AppDelegate.swift
//  SilentRelay
//
//  Handles push notifications, PushKit for VoIP, and CallKit integration
//

import UIKit
import PushKit
import CallKit
import UserNotifications

class AppDelegate: NSObject, UIApplicationDelegate {

    // MARK: - Properties
    let pushManager = PushManager.shared
    let callKitManager = CallKitManager.shared

    // MARK: - Application Lifecycle
    func application(
        _ application: UIApplication,
        didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]? = nil
    ) -> Bool {
        // Register for push notifications
        registerForPushNotifications()

        // Register for VoIP push notifications
        registerForVoIPPush()

        // Configure CallKit
        configureCallKit()

        return true
    }

    // MARK: - Push Notifications
    private func registerForPushNotifications() {
        UNUserNotificationCenter.current().delegate = self

        UNUserNotificationCenter.current().requestAuthorization(
            options: [.alert, .sound, .badge]
        ) { granted, error in
            if granted {
                DispatchQueue.main.async {
                    UIApplication.shared.registerForRemoteNotifications()
                }
            }
            if let error = error {
                print("Push notification authorization error: \(error)")
            }
        }
    }

    func application(
        _ application: UIApplication,
        didRegisterForRemoteNotificationsWithDeviceToken deviceToken: Data
    ) {
        let tokenString = deviceToken.map { String(format: "%02.2hhx", $0) }.joined()
        print("APNs token: \(tokenString)")

        // Upload token to backend
        Task {
            await pushManager.registerPushToken(tokenString, platform: "ios")
        }
    }

    func application(
        _ application: UIApplication,
        didFailToRegisterForRemoteNotificationsWithError error: Error
    ) {
        print("Failed to register for push notifications: \(error)")
    }

    // MARK: - VoIP Push (PushKit)
    private func registerForVoIPPush() {
        let voipRegistry = PKPushRegistry(queue: .main)
        voipRegistry.delegate = self
        voipRegistry.desiredPushTypes = [.voIP]
    }

    // MARK: - CallKit Configuration
    private func configureCallKit() {
        callKitManager.configure()
    }
}

// MARK: - UNUserNotificationCenterDelegate
extension AppDelegate: UNUserNotificationCenterDelegate {

    // Handle notification when app is in foreground
    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        willPresent notification: UNNotification,
        withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void
    ) {
        // Show notification even when app is in foreground
        completionHandler([.banner, .sound, .badge])
    }

    // Handle notification tap
    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        didReceive response: UNNotificationResponse,
        withCompletionHandler completionHandler: @escaping () -> Void
    ) {
        let userInfo = response.notification.request.content.userInfo

        // Handle notification action
        if let conversationId = userInfo["conversation_id"] as? String {
            // Navigate to conversation
            NotificationCenter.default.post(
                name: .openConversation,
                object: nil,
                userInfo: ["conversationId": conversationId]
            )
        }

        completionHandler()
    }
}

// MARK: - PKPushRegistryDelegate (VoIP Push)
extension AppDelegate: PKPushRegistryDelegate {

    func pushRegistry(
        _ registry: PKPushRegistry,
        didUpdate pushCredentials: PKPushCredentials,
        for type: PKPushType
    ) {
        guard type == .voIP else { return }

        let tokenString = pushCredentials.token.map { String(format: "%02.2hhx", $0) }.joined()
        print("VoIP push token: \(tokenString)")

        // Upload VoIP token to backend
        Task {
            await pushManager.registerVoIPToken(tokenString)
        }
    }

    func pushRegistry(
        _ registry: PKPushRegistry,
        didReceiveIncomingPushWith payload: PKPushPayload,
        for type: PKPushType,
        completion: @escaping () -> Void
    ) {
        guard type == .voIP else {
            completion()
            return
        }

        // Extract call info from payload
        let callerId = payload.dictionaryPayload["caller_id"] as? String ?? ""
        let callerName = payload.dictionaryPayload["caller_name"] as? String ?? "Unknown"
        let callType = payload.dictionaryPayload["call_type"] as? String ?? "audio"
        let callId = payload.dictionaryPayload["call_id"] as? String ?? UUID().uuidString

        // Report incoming call to CallKit
        callKitManager.reportIncomingCall(
            callId: callId,
            callerId: callerId,
            callerName: callerName,
            hasVideo: callType == "video"
        ) { error in
            if let error = error {
                print("Failed to report incoming call: \(error)")
            }
            completion()
        }
    }

    func pushRegistry(
        _ registry: PKPushRegistry,
        didInvalidatePushTokenFor type: PKPushType
    ) {
        guard type == .voIP else { return }
        print("VoIP push token invalidated")
    }
}

// MARK: - Notification Names
extension Notification.Name {
    static let openConversation = Notification.Name("openConversation")
}
