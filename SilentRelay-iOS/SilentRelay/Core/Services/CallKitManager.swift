//
//  CallKitManager.swift
//  SilentRelay
//
//  CallKit integration for native iOS call UI
//

import Foundation
import CallKit
import AVFoundation

// MARK: - CallKit Manager
@MainActor
final class CallKitManager: NSObject {
    // MARK: - Singleton
    static let shared = CallKitManager()

    // MARK: - Properties
    private var provider: CXProvider?
    private let callController = CXCallController()

    // Active calls
    private var activeCalls: [UUID: CallInfo] = [:]

    // Callbacks
    var onCallStarted: ((UUID, String, Bool) -> Void)?
    var onCallEnded: ((UUID) -> Void)?
    var onCallAnswered: ((UUID) -> Void)?
    var onCallMuted: ((UUID, Bool) -> Void)?

    private override init() {
        super.init()
    }

    // MARK: - Configure
    func configure() {
        let configuration = CXProviderConfiguration()
        configuration.supportsVideo = true
        configuration.maximumCallsPerCallGroup = 1
        configuration.maximumCallGroups = 1
        configuration.supportedHandleTypes = [.generic]
        configuration.iconTemplateImageData = nil // Add app icon data here

        provider = CXProvider(configuration: configuration)
        provider?.setDelegate(self, queue: .main)
    }

    // MARK: - Report Incoming Call
    func reportIncomingCall(
        callId: String,
        callerId: String,
        callerName: String,
        hasVideo: Bool,
        completion: @escaping (Error?) -> Void
    ) {
        let uuid = UUID()
        let update = CXCallUpdate()
        update.remoteHandle = CXHandle(type: .generic, value: callerId)
        update.localizedCallerName = callerName
        update.hasVideo = hasVideo
        update.supportsHolding = false
        update.supportsGrouping = false
        update.supportsUngrouping = false
        update.supportsDTMF = false

        // Store call info
        activeCalls[uuid] = CallInfo(
            callId: callId,
            callerId: callerId,
            callerName: callerName,
            hasVideo: hasVideo,
            isOutgoing: false
        )

        provider?.reportNewIncomingCall(with: uuid, update: update) { error in
            if let error = error {
                self.activeCalls.removeValue(forKey: uuid)
            }
            completion(error)
        }
    }

    // MARK: - Start Outgoing Call
    func startOutgoingCall(
        to userId: String,
        displayName: String,
        hasVideo: Bool
    ) async throws -> UUID {
        let uuid = UUID()
        let handle = CXHandle(type: .generic, value: userId)

        let startCallAction = CXStartCallAction(call: uuid, handle: handle)
        startCallAction.isVideo = hasVideo
        startCallAction.contactIdentifier = displayName

        let transaction = CXTransaction(action: startCallAction)

        try await callController.request(transaction)

        // Store call info
        activeCalls[uuid] = CallInfo(
            callId: uuid.uuidString,
            callerId: userId,
            callerName: displayName,
            hasVideo: hasVideo,
            isOutgoing: true
        )

        return uuid
    }

    // MARK: - End Call
    func endCall(uuid: UUID) async throws {
        let endCallAction = CXEndCallAction(call: uuid)
        let transaction = CXTransaction(action: endCallAction)
        try await callController.request(transaction)
    }

    // MARK: - Report Call Connected
    func reportCallConnected(uuid: UUID) {
        provider?.reportOutgoingCall(with: uuid, connectedAt: Date())
    }

    // MARK: - Report Call Ended
    func reportCallEnded(uuid: UUID, reason: CXCallEndedReason) {
        provider?.reportCall(with: uuid, endedAt: Date(), reason: reason)
        activeCalls.removeValue(forKey: uuid)
    }

    // MARK: - Mute Call
    func setMuted(uuid: UUID, muted: Bool) async throws {
        let muteAction = CXSetMutedCallAction(call: uuid, muted: muted)
        let transaction = CXTransaction(action: muteAction)
        try await callController.request(transaction)
    }

    // MARK: - Get Call Info
    func getCallInfo(uuid: UUID) -> CallInfo? {
        activeCalls[uuid]
    }
}

// MARK: - CXProviderDelegate
extension CallKitManager: CXProviderDelegate {
    nonisolated func providerDidReset(_ provider: CXProvider) {
        Task { @MainActor in
            // End all active calls
            for uuid in self.activeCalls.keys {
                self.onCallEnded?(uuid)
            }
            self.activeCalls.removeAll()
        }
    }

    nonisolated func provider(_ provider: CXProvider, perform action: CXStartCallAction) {
        // Configure audio session
        configureAudioSession()

        Task { @MainActor in
            self.onCallStarted?(action.callUUID, action.handle.value, action.isVideo)
        }

        action.fulfill()
    }

    nonisolated func provider(_ provider: CXProvider, perform action: CXAnswerCallAction) {
        // Configure audio session
        configureAudioSession()

        Task { @MainActor in
            self.onCallAnswered?(action.callUUID)
        }

        action.fulfill()
    }

    nonisolated func provider(_ provider: CXProvider, perform action: CXEndCallAction) {
        Task { @MainActor in
            self.onCallEnded?(action.callUUID)
            self.activeCalls.removeValue(forKey: action.callUUID)
        }

        action.fulfill()
    }

    nonisolated func provider(_ provider: CXProvider, perform action: CXSetMutedCallAction) {
        Task { @MainActor in
            self.onCallMuted?(action.callUUID, action.isMuted)
        }

        action.fulfill()
    }

    nonisolated func provider(_ provider: CXProvider, didActivate audioSession: AVAudioSession) {
        // Audio session activated - WebRTC can start using audio
    }

    nonisolated func provider(_ provider: CXProvider, didDeactivate audioSession: AVAudioSession) {
        // Audio session deactivated
    }

    // MARK: - Audio Session Configuration
    private nonisolated func configureAudioSession() {
        do {
            let audioSession = AVAudioSession.sharedInstance()
            try audioSession.setCategory(.playAndRecord, mode: .voiceChat, options: [.allowBluetooth, .defaultToSpeaker])
            try audioSession.setActive(true)
        } catch {
            print("Failed to configure audio session: \(error)")
        }
    }
}

// MARK: - Call Info
struct CallInfo {
    let callId: String
    let callerId: String
    let callerName: String
    let hasVideo: Bool
    let isOutgoing: Bool
    var startTime: Date?
    var connectedTime: Date?
}
