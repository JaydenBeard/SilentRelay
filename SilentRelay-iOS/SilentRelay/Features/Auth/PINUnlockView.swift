//
//  PINUnlockView.swift
//  SilentRelay
//
//  PIN unlock for existing users with biometric option
//

import SwiftUI
import LocalAuthentication

struct PINUnlockView: View {
    @Environment(AuthManager.self) private var authManager

    @State private var pin = ""
    @State private var error: String?
    @State private var attempts = 0
    @State private var isLocked = false
    @State private var lockoutEndTime: Date?
    @State private var biometricType: BiometricType = .none

    enum BiometricType {
        case none
        case faceID
        case touchID
    }

    var body: some View {
        VStack(spacing: 32) {
            Spacer()

            // Header
            VStack(spacing: 12) {
                Image(systemName: "lock.fill")
                    .font(.system(size: 48))
                    .foregroundStyle(.blue)

                Text("Welcome Back")
                    .font(.largeTitle)
                    .fontWeight(.bold)

                Text("Enter your PIN to unlock")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }

            // PIN dots
            PINDotsView(length: 6, filled: pin.count)
                .shake(trigger: error != nil)

            // Number pad
            NumberPadView(
                onNumberTap: { number in
                    if pin.count < 6 {
                        pin += String(number)
                        error = nil

                        if pin.count == 6 {
                            verifyPIN()
                        }
                    }
                },
                onDelete: {
                    if !pin.isEmpty {
                        pin.removeLast()
                        error = nil
                    }
                }
            )
            .disabled(isLocked)

            // Error message
            if let error = error {
                Text(error)
                    .font(.caption)
                    .foregroundStyle(.red)
            }

            // Lockout message
            if isLocked, let endTime = lockoutEndTime {
                TimeRemainingView(endTime: endTime) {
                    isLocked = false
                    attempts = 0
                }
            }

            Spacer()

            // Biometric button
            if biometricType != .none && !isLocked {
                Button {
                    authenticateWithBiometrics()
                } label: {
                    HStack {
                        Image(systemName: biometricType == .faceID ? "faceid" : "touchid")
                        Text(biometricType == .faceID ? "Use Face ID" : "Use Touch ID")
                    }
                    .padding(.horizontal, 24)
                    .padding(.vertical, 12)
                    .background(.ultraThinMaterial)
                    .clipShape(Capsule())
                }
                .buttonStyle(.plain)
            }

            // Logout option
            Button {
                Task {
                    await authManager.logout()
                }
            } label: {
                Text("Use different account")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }
            .padding(.bottom, 16)
        }
        .onAppear {
            checkBiometricType()
            // Try biometric auth on appear if available
            if biometricType != .none {
                authenticateWithBiometrics()
            }
        }
    }

    private func verifyPIN() {
        Task {
            let success = await authManager.unlockWithPIN(pin)

            if success {
                // PIN correct - unlock
            } else {
                // PIN incorrect
                attempts += 1
                error = "Incorrect PIN"
                pin = ""

                // Lockout after too many attempts
                if attempts >= 5 {
                    isLocked = true
                    lockoutEndTime = Date().addingTimeInterval(30 * Double(attempts - 4))
                }
            }
        }
    }

    private func checkBiometricType() {
        let context = LAContext()
        var error: NSError?

        if context.canEvaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, error: &error) {
            switch context.biometryType {
            case .faceID:
                biometricType = .faceID
            case .touchID:
                biometricType = .touchID
            default:
                biometricType = .none
            }
        }
    }

    private func authenticateWithBiometrics() {
        let context = LAContext()
        let reason = "Unlock SilentRelay"

        context.evaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, localizedReason: reason) { success, _ in
            DispatchQueue.main.async {
                if success {
                    // Biometric auth successful - unlock
                    Task {
                        // TODO: Get stored PIN hash and unlock
                        await authManager.unlockWithPIN("") // Will be implemented properly
                    }
                }
            }
        }
    }
}

// MARK: - Time Remaining View
struct TimeRemainingView: View {
    let endTime: Date
    let onComplete: () -> Void

    @State private var timeRemaining: Int = 0

    var body: some View {
        Text("Try again in \(timeRemaining) seconds")
            .font(.caption)
            .foregroundStyle(.orange)
            .onAppear {
                startTimer()
            }
    }

    private func startTimer() {
        updateTimeRemaining()

        Timer.scheduledTimer(withTimeInterval: 1, repeats: true) { timer in
            updateTimeRemaining()

            if timeRemaining <= 0 {
                timer.invalidate()
                onComplete()
            }
        }
    }

    private func updateTimeRemaining() {
        timeRemaining = max(0, Int(endTime.timeIntervalSinceNow))
    }
}

// MARK: - Shake Modifier
extension View {
    func shake(trigger: Bool) -> some View {
        modifier(ShakeModifier(trigger: trigger))
    }
}

struct ShakeModifier: ViewModifier {
    let trigger: Bool
    @State private var shake = false

    func body(content: Content) -> some View {
        content
            .offset(x: shake ? -10 : 0)
            .animation(shake ? .default.repeatCount(3).speed(6) : .default, value: shake)
            .onChange(of: trigger) { _, newValue in
                if newValue {
                    shake = true
                    DispatchQueue.main.asyncAfter(deadline: .now() + 0.3) {
                        shake = false
                    }
                }
            }
    }
}

#Preview {
    PINUnlockView()
        .environment(AuthManager.shared)
}
