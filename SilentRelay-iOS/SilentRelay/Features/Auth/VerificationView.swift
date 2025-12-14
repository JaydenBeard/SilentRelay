//
//  VerificationView.swift
//  SilentRelay
//
//  SMS verification code entry
//

import SwiftUI

struct VerificationView: View {
    @Environment(AuthManager.self) private var authManager

    @State private var code = ""
    @State private var codeDigits: [String] = Array(repeating: "", count: 6)
    @FocusState private var focusedField: Int?

    var body: some View {
        VStack(spacing: 32) {
            Spacer()

            // Header
            VStack(spacing: 12) {
                Image(systemName: "message.badge.filled.fill")
                    .font(.system(size: 48))
                    .foregroundStyle(.blue)

                Text("Verification")
                    .font(.largeTitle)
                    .fontWeight(.bold)

                if case .verifying(let phone) = authManager.authState {
                    Text("Enter the code sent to \(phone)")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                        .multilineTextAlignment(.center)
                }
            }

            // Code input fields
            HStack(spacing: 8) {
                ForEach(0..<6, id: \.self) { index in
                    CodeDigitField(
                        digit: $codeDigits[index],
                        isFocused: focusedField == index
                    )
                    .focused($focusedField, equals: index)
                    .onChange(of: codeDigits[index]) { oldValue, newValue in
                        handleDigitChange(at: index, oldValue: oldValue, newValue: newValue)
                    }
                }
            }
            .padding(.horizontal)

            // Error message
            if let error = authManager.error {
                Text(error.localizedDescription)
                    .font(.caption)
                    .foregroundStyle(.red)
                    .padding(.horizontal)
            }

            // Resend code
            Button {
                Task {
                    if case .verifying(let phone) = authManager.authState {
                        await authManager.requestCode(phoneNumber: phone)
                    }
                }
            } label: {
                Text("Resend code")
                    .font(.subheadline)
                    .foregroundStyle(.blue)
            }
            .disabled(authManager.isLoading)

            Spacer()

            // Verify button
            Button {
                Task {
                    let fullCode = codeDigits.joined()
                    await authManager.verifyCode(fullCode)
                }
            } label: {
                HStack {
                    if authManager.isLoading {
                        ProgressView()
                            .tint(.white)
                    } else {
                        Text("Verify")
                    }
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 16)
                .background(isValidCode ? Color.blue : Color.gray.opacity(0.3))
                .foregroundStyle(.white)
                .clipShape(RoundedRectangle(cornerRadius: 14))
            }
            .disabled(!isValidCode || authManager.isLoading)
            .padding(.horizontal)

            // Back button
            Button {
                authManager.resetToPhoneEntry()
            } label: {
                Text("Change phone number")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }
            .padding(.bottom, 16)
        }
        .onAppear {
            focusedField = 0
        }
    }

    private var isValidCode: Bool {
        codeDigits.allSatisfy { !$0.isEmpty }
    }

    private func handleDigitChange(at index: Int, oldValue: String, newValue: String) {
        // Handle paste of full code
        if newValue.count > 1 {
            let digits = Array(newValue.prefix(6))
            for (i, digit) in digits.enumerated() {
                if i < 6 {
                    codeDigits[i] = String(digit)
                }
            }
            focusedField = min(digits.count, 5)
            return
        }

        // Move to next field on input
        if !newValue.isEmpty && index < 5 {
            focusedField = index + 1
        }

        // Move to previous field on delete
        if newValue.isEmpty && oldValue.isEmpty && index > 0 {
            focusedField = index - 1
        }
    }
}

// MARK: - Code Digit Field
struct CodeDigitField: View {
    @Binding var digit: String
    let isFocused: Bool

    var body: some View {
        TextField("", text: $digit)
            .keyboardType(.numberPad)
            .textContentType(.oneTimeCode)
            .multilineTextAlignment(.center)
            .font(.title)
            .fontWeight(.semibold)
            .frame(width: 48, height: 56)
            .background(.ultraThinMaterial)
            .clipShape(RoundedRectangle(cornerRadius: 12))
            .overlay(
                RoundedRectangle(cornerRadius: 12)
                    .stroke(isFocused ? Color.blue : Color.clear, lineWidth: 2)
            )
            .onChange(of: digit) { _, newValue in
                // Only allow single digit
                if newValue.count > 1 {
                    digit = String(newValue.suffix(1))
                }
                // Only allow numbers
                digit = digit.filter { $0.isNumber }
            }
    }
}

#Preview {
    VerificationView()
        .environment(AuthManager.shared)
}
