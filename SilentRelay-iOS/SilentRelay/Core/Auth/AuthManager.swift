//
//  AuthManager.swift
//  SilentRelay
//
//  Authentication state management
//

import Foundation
import Observation

// MARK: - Auth State
enum AuthState: Equatable {
    case loggedOut
    case verifying(phoneNumber: String)
    case registering(phoneNumber: String)
    case needsPinUnlock
    case authenticated
}

// MARK: - Auth Manager
@Observable
@MainActor
final class AuthManager {
    // MARK: - Singleton
    static let shared = AuthManager()

    // MARK: - State
    private(set) var authState: AuthState = .loggedOut
    private(set) var currentUser: User?
    private(set) var isLoading = false
    private(set) var error: APIError?

    // Verification state
    private(set) var verificationCode: String?
    private(set) var phoneNumber: String?

    // MARK: - Dependencies
    private let apiClient = APIClient.shared
    private let keychainManager = KeychainManager.shared

    // MARK: - Init
    private init() {
        Task {
            await checkAuthState()
        }
    }

    // MARK: - Check Auth State
    private func checkAuthState() async {
        // Check if we have stored tokens
        if let accessToken = await keychainManager.loadStringOptional(.accessToken) {
            await apiClient.setAuthToken(accessToken)

            // Check if encryption is set up (requires PIN unlock)
            let hasEncryptionSetup = await keychainManager.exists(.masterKeySalt)

            if hasEncryptionSetup {
                authState = .needsPinUnlock
            } else {
                // Try to fetch current user
                await fetchCurrentUser()
            }
        }
    }

    // MARK: - Request Verification Code
    func requestCode(phoneNumber: String) async {
        isLoading = true
        error = nil

        do {
            let _: EmptyResponse = try await apiClient.post(
                .authRequestCode,
                body: RequestCodeRequest(phoneNumber: phoneNumber)
            )

            self.phoneNumber = phoneNumber
            authState = .verifying(phoneNumber: phoneNumber)
        } catch let apiError as APIError {
            error = apiError
        } catch {
            self.error = .unknown(error.localizedDescription)
        }

        isLoading = false
    }

    // MARK: - Verify Code
    func verifyCode(_ code: String) async {
        guard let phoneNumber = phoneNumber else {
            error = .validationError("No phone number to verify")
            return
        }

        isLoading = true
        error = nil

        do {
            let response: VerifyCodeResponse = try await apiClient.post(
                .authVerify,
                body: VerifyCodeRequest(phoneNumber: phoneNumber, code: code)
            )

            if response.userExists {
                // Existing user - they have tokens
                if let accessToken = response.accessToken,
                   let refreshToken = response.refreshToken {
                    try await keychainManager.saveAuthTokens(
                        accessToken: accessToken,
                        refreshToken: refreshToken
                    )
                    await apiClient.setAuthToken(accessToken)
                    await apiClient.setRefreshToken(refreshToken)

                    currentUser = response.user

                    // Check if they need PIN unlock
                    let hasEncryptionSetup = await keychainManager.exists(.masterKeySalt)
                    authState = hasEncryptionSetup ? .needsPinUnlock : .authenticated
                }
            } else {
                // New user - needs to register
                verificationCode = code
                authState = .registering(phoneNumber: phoneNumber)
            }
        } catch let apiError as APIError {
            error = apiError
        } catch {
            self.error = .unknown(error.localizedDescription)
        }

        isLoading = false
    }

    // MARK: - Register New User
    func register(
        publicIdentityKey: String,
        publicSignedPrekey: String,
        signedPrekeySignature: String,
        oneTimePrekeys: [String]
    ) async {
        guard let phoneNumber = phoneNumber else {
            error = .validationError("No phone number")
            return
        }

        isLoading = true
        error = nil

        do {
            let response: RegisterResponse = try await apiClient.post(
                .authRegister,
                body: RegisterRequest(
                    phoneNumber: phoneNumber,
                    publicIdentityKey: publicIdentityKey,
                    publicSignedPrekey: publicSignedPrekey,
                    signedPrekeySignature: signedPrekeySignature,
                    oneTimePrekeys: oneTimePrekeys
                )
            )

            // Save tokens
            try await keychainManager.saveAuthTokens(
                accessToken: response.accessToken,
                refreshToken: response.refreshToken
            )
            await apiClient.setAuthToken(response.accessToken)
            await apiClient.setRefreshToken(response.refreshToken)

            currentUser = response.user
            authState = .authenticated
        } catch let apiError as APIError {
            error = apiError
        } catch {
            self.error = .unknown(error.localizedDescription)
        }

        isLoading = false
    }

    // MARK: - PIN Unlock
    func unlockWithPIN(_ pin: String) async -> Bool {
        // This will be implemented with SignalManager
        // For now, just transition to authenticated
        authState = .authenticated
        return true
    }

    // MARK: - Fetch Current User
    func fetchCurrentUser() async {
        do {
            let user: User = try await apiClient.get(.usersMe)
            currentUser = user
            authState = .authenticated
        } catch {
            // Token might be invalid
            await logout()
        }
    }

    // MARK: - Update Profile
    func updateProfile(username: String?, displayName: String?, avatar: String?) async throws {
        let updatedUser: User = try await apiClient.patch(
            .usersMe,
            body: UpdateProfileRequest(
                username: username,
                displayName: displayName,
                avatar: avatar
            )
        )
        currentUser = updatedUser
    }

    // MARK: - Logout
    func logout() async {
        do {
            try await apiClient.post(.authLogout, body: EmptyPayload())
        } catch {
            // Continue with local logout even if server call fails
        }

        // Clear local state
        try? await keychainManager.clearAuthTokens()
        await apiClient.setAuthToken(nil)
        await apiClient.setRefreshToken(nil)

        currentUser = nil
        phoneNumber = nil
        verificationCode = nil
        authState = .loggedOut
    }

    // MARK: - Clear Error
    func clearError() {
        error = nil
    }

    // MARK: - Reset to Phone Entry
    func resetToPhoneEntry() {
        phoneNumber = nil
        verificationCode = nil
        authState = .loggedOut
    }
}
