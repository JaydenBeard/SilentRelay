//
//  PINSetupView.swift
//  SilentRelay
//
//  PIN setup for encryption key protection
//

import SwiftUI

struct PINSetupView: View {
    @Environment(AuthManager.self) private var authManager

    @State private var pin = ""
    @State private var confirmPin = ""
    @State private var step: PINSetupStep = .create
    @State private var error: String?
    @State private var isLoading = false

    @FocusState private var isPinFieldFocused: Bool

    enum PINSetupStep {
        case create
        case confirm
    }

    var body: some View {
        VStack(spacing: 32) {
            Spacer()

            // Header
            VStack(spacing: 12) {
                Image(systemName: step == .create ? "lock.fill" : "lock.badge.clock.fill")
                    .font(.system(size: 48))
                    .foregroundStyle(.blue)
                    .animation(.easeInOut, value: step)

                Text(step == .create ? "Create PIN" : "Confirm PIN")
                    .font(.largeTitle)
                    .fontWeight(.bold)

                Text(step == .create
                     ? "This PIN will protect your encryption keys"
                     : "Enter your PIN again to confirm")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                    .multilineTextAlignment(.center)
            }

            // PIN dots
            PINDotsView(
                length: 6,
                filled: step == .create ? pin.count : confirmPin.count
            )

            // Hidden text field for keyboard
            SecureField("", text: step == .create ? $pin : $confirmPin)
                .keyboardType(.numberPad)
                .textContentType(.oneTimeCode)
                .focused($isPinFieldFocused)
                .opacity(0)
                .frame(height: 1)
                .onChange(of: pin) { _, newValue in
                    validatePinInput(&pin, newValue)
                    if pin.count == 6 {
                        handlePinComplete()
                    }
                }
                .onChange(of: confirmPin) { _, newValue in
                    validatePinInput(&confirmPin, newValue)
                    if confirmPin.count == 6 {
                        handleConfirmPinComplete()
                    }
                }

            // Number pad
            NumberPadView(
                onNumberTap: { number in
                    if step == .create {
                        if pin.count < 6 {
                            pin += String(number)
                        }
                    } else {
                        if confirmPin.count < 6 {
                            confirmPin += String(number)
                        }
                    }
                },
                onDelete: {
                    if step == .create {
                        if !pin.isEmpty {
                            pin.removeLast()
                        }
                    } else {
                        if !confirmPin.isEmpty {
                            confirmPin.removeLast()
                        }
                    }
                }
            )

            // Error message
            if let error = error {
                Text(error)
                    .font(.caption)
                    .foregroundStyle(.red)
            }

            Spacer()

            // Security note
            VStack(spacing: 8) {
                HStack(spacing: 4) {
                    Image(systemName: "info.circle")
                    Text("Your PIN is never sent to our servers")
                }
                .font(.caption)
                .foregroundStyle(.secondary)

                Text("If you forget your PIN, you will need to reset your account")
                    .font(.caption2)
                    .foregroundStyle(.tertiary)
                    .multilineTextAlignment(.center)
            }
            .padding(.horizontal)
            .padding(.bottom, 16)
        }
        .onAppear {
            isPinFieldFocused = true
        }
    }

    private func validatePinInput(_ pin: inout String, _ newValue: String) {
        // Only allow numbers
        pin = newValue.filter { $0.isNumber }
        // Limit to 6 digits
        if pin.count > 6 {
            pin = String(pin.prefix(6))
        }
        error = nil
    }

    private func handlePinComplete() {
        guard pin.count == 6 else { return }

        // Validate PIN strength
        if isWeakPIN(pin) {
            error = "PIN is too simple. Please choose a stronger PIN."
            pin = ""
            return
        }

        // Move to confirmation step
        withAnimation {
            step = .confirm
        }
    }

    private func handleConfirmPinComplete() {
        guard confirmPin.count == 6 else { return }

        if pin == confirmPin {
            // PINs match - proceed with encryption setup
            Task {
                isLoading = true
                // TODO: Set up encryption with SignalManager
                // For now, just complete the flow
                await authManager.unlockWithPIN(pin)
                isLoading = false
            }
        } else {
            error = "PINs don't match. Please try again."
            confirmPin = ""
        }
    }

    private func isWeakPIN(_ pin: String) -> Bool {
        // Check for simple patterns
        let weakPINs = ["000000", "111111", "123456", "654321", "123123"]
        if weakPINs.contains(pin) {
            return true
        }

        // Check for all same digits
        if Set(pin).count == 1 {
            return true
        }

        // Check for sequential digits
        let digits = pin.compactMap { Int(String($0)) }
        var isAscending = true
        var isDescending = true
        for i in 1..<digits.count {
            if digits[i] != digits[i-1] + 1 { isAscending = false }
            if digits[i] != digits[i-1] - 1 { isDescending = false }
        }
        if isAscending || isDescending { return true }

        return false
    }
}

// MARK: - PIN Dots View
struct PINDotsView: View {
    let length: Int
    let filled: Int

    var body: some View {
        HStack(spacing: 16) {
            ForEach(0..<length, id: \.self) { index in
                Circle()
                    .fill(index < filled ? Color.blue : Color.gray.opacity(0.3))
                    .frame(width: 16, height: 16)
                    .animation(.easeInOut(duration: 0.1), value: filled)
            }
        }
    }
}

// MARK: - Number Pad View
struct NumberPadView: View {
    let onNumberTap: (Int) -> Void
    let onDelete: () -> Void

    private let buttons: [[NumberPadButton]] = [
        [.number(1), .number(2), .number(3)],
        [.number(4), .number(5), .number(6)],
        [.number(7), .number(8), .number(9)],
        [.empty, .number(0), .delete]
    ]

    var body: some View {
        VStack(spacing: 12) {
            ForEach(buttons.indices, id: \.self) { row in
                HStack(spacing: 24) {
                    ForEach(buttons[row].indices, id: \.self) { col in
                        NumberPadButtonView(
                            button: buttons[row][col],
                            onTap: { handleTap(buttons[row][col]) }
                        )
                    }
                }
            }
        }
    }

    private func handleTap(_ button: NumberPadButton) {
        switch button {
        case .number(let num):
            onNumberTap(num)
        case .delete:
            onDelete()
        case .empty:
            break
        }
    }
}

enum NumberPadButton {
    case number(Int)
    case delete
    case empty
}

struct NumberPadButtonView: View {
    let button: NumberPadButton
    let onTap: () -> Void

    var body: some View {
        Button(action: onTap) {
            Group {
                switch button {
                case .number(let num):
                    Text("\(num)")
                        .font(.title)
                        .fontWeight(.medium)
                case .delete:
                    Image(systemName: "delete.left")
                        .font(.title2)
                case .empty:
                    Color.clear
                }
            }
            .frame(width: 72, height: 72)
            .background(button.isInteractive ? Color(.systemGray6) : Color.clear)
            .clipShape(Circle())
        }
        .buttonStyle(.plain)
        .disabled(!button.isInteractive)
    }
}

extension NumberPadButton {
    var isInteractive: Bool {
        switch self {
        case .number, .delete: return true
        case .empty: return false
        }
    }
}

#Preview {
    PINSetupView()
        .environment(AuthManager.shared)
}
