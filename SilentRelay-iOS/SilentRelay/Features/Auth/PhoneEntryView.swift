//
//  PhoneEntryView.swift
//  SilentRelay
//
//  Phone number entry for authentication
//

import SwiftUI

struct PhoneEntryView: View {
    @Environment(AuthManager.self) private var authManager

    @State private var phoneNumber = ""
    @State private var countryCode = "+1"
    @FocusState private var isPhoneFieldFocused: Bool

    var body: some View {
        NavigationStack {
            VStack(spacing: 32) {
                Spacer()

                // Logo and title
                VStack(spacing: 16) {
                    Image(systemName: "lock.shield.fill")
                        .font(.system(size: 64))
                        .foregroundStyle(.blue)

                    Text("SilentRelay")
                        .font(.largeTitle)
                        .fontWeight(.bold)

                    Text("Secure, private messaging")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }

                // Phone input
                VStack(alignment: .leading, spacing: 8) {
                    Text("Enter your phone number")
                        .font(.headline)

                    HStack(spacing: 12) {
                        // Country code picker
                        Menu {
                            Button("+1 (US)") { countryCode = "+1" }
                            Button("+44 (UK)") { countryCode = "+44" }
                            Button("+61 (AU)") { countryCode = "+61" }
                            Button("+49 (DE)") { countryCode = "+49" }
                            Button("+33 (FR)") { countryCode = "+33" }
                        } label: {
                            HStack {
                                Text(countryCode)
                                    .fontWeight(.medium)
                                Image(systemName: "chevron.down")
                                    .font(.caption)
                            }
                            .padding(.horizontal, 12)
                            .padding(.vertical, 14)
                            .background(.ultraThinMaterial)
                            .clipShape(RoundedRectangle(cornerRadius: 12))
                        }
                        .buttonStyle(.plain)

                        // Phone number field
                        TextField("Phone number", text: $phoneNumber)
                            .keyboardType(.phonePad)
                            .textContentType(.telephoneNumber)
                            .padding(.horizontal, 16)
                            .padding(.vertical, 14)
                            .background(.ultraThinMaterial)
                            .clipShape(RoundedRectangle(cornerRadius: 12))
                            .focused($isPhoneFieldFocused)
                    }

                    Text("We'll send you a verification code")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                .padding(.horizontal)

                // Error message
                if let error = authManager.error {
                    Text(error.localizedDescription)
                        .font(.caption)
                        .foregroundStyle(.red)
                        .padding(.horizontal)
                }

                Spacer()

                // Continue button
                Button {
                    Task {
                        let fullNumber = countryCode + phoneNumber.replacingOccurrences(of: " ", with: "")
                        await authManager.requestCode(phoneNumber: fullNumber)
                    }
                } label: {
                    HStack {
                        if authManager.isLoading {
                            ProgressView()
                                .tint(.white)
                        } else {
                            Text("Continue")
                        }
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 16)
                    .background(isValidPhoneNumber ? Color.blue : Color.gray.opacity(0.3))
                    .foregroundStyle(.white)
                    .clipShape(RoundedRectangle(cornerRadius: 14))
                }
                .disabled(!isValidPhoneNumber || authManager.isLoading)
                .padding(.horizontal)

                // Terms
                Text("By continuing, you agree to our [Terms of Service](https://silentrelay.com/terms) and [Privacy Policy](https://silentrelay.com/privacy)")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .multilineTextAlignment(.center)
                    .padding(.horizontal)
                    .padding(.bottom, 16)
            }
            .onAppear {
                isPhoneFieldFocused = true
            }
        }
    }

    private var isValidPhoneNumber: Bool {
        let digits = phoneNumber.filter { $0.isNumber }
        return digits.count >= 10
    }
}

#Preview {
    PhoneEntryView()
        .environment(AuthManager.shared)
}
