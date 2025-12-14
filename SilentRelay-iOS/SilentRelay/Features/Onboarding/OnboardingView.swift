//
//  OnboardingView.swift
//  SilentRelay
//
//  New user onboarding flow
//

import SwiftUI

struct OnboardingView: View {
    @Environment(AuthManager.self) private var authManager

    @State private var currentStep = 0
    @State private var username = ""
    @State private var displayName = ""
    @State private var isLoading = false
    @State private var error: String?

    var body: some View {
        VStack(spacing: 32) {
            // Progress indicator
            HStack(spacing: 8) {
                ForEach(0..<3, id: \.self) { index in
                    Capsule()
                        .fill(index <= currentStep ? Color.blue : Color.gray.opacity(0.3))
                        .frame(height: 4)
                }
            }
            .padding(.horizontal)
            .padding(.top)

            TabView(selection: $currentStep) {
                // Step 1: Welcome
                WelcomeStepView(onContinue: { currentStep = 1 })
                    .tag(0)

                // Step 2: Profile setup
                ProfileSetupStepView(
                    username: $username,
                    displayName: $displayName,
                    onContinue: { currentStep = 2 }
                )
                .tag(1)

                // Step 3: PIN setup
                PINSetupView()
                    .tag(2)
            }
            .tabViewStyle(.page(indexDisplayMode: .never))
            .animation(.easeInOut, value: currentStep)
        }
    }
}

// MARK: - Welcome Step
struct WelcomeStepView: View {
    let onContinue: () -> Void

    var body: some View {
        VStack(spacing: 32) {
            Spacer()

            VStack(spacing: 24) {
                Image(systemName: "checkmark.shield.fill")
                    .font(.system(size: 80))
                    .foregroundStyle(.green)

                Text("Welcome to SilentRelay")
                    .font(.title)
                    .fontWeight(.bold)

                VStack(alignment: .leading, spacing: 16) {
                    FeatureRow(
                        icon: "lock.fill",
                        title: "End-to-End Encryption",
                        description: "Your messages are encrypted on your device"
                    )

                    FeatureRow(
                        icon: "person.2.fill",
                        title: "Private by Design",
                        description: "We can't read your messages"
                    )

                    FeatureRow(
                        icon: "iphone",
                        title: "Multi-Device",
                        description: "Use SilentRelay on all your devices"
                    )
                }
                .padding(.horizontal)
            }

            Spacer()

            Button(action: onContinue) {
                Text("Get Started")
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 16)
                    .background(Color.blue)
                    .foregroundStyle(.white)
                    .clipShape(RoundedRectangle(cornerRadius: 14))
            }
            .padding(.horizontal)
            .padding(.bottom, 32)
        }
    }
}

struct FeatureRow: View {
    let icon: String
    let title: String
    let description: String

    var body: some View {
        HStack(spacing: 16) {
            Image(systemName: icon)
                .font(.title2)
                .foregroundStyle(.blue)
                .frame(width: 32)

            VStack(alignment: .leading, spacing: 2) {
                Text(title)
                    .font(.headline)
                Text(description)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
    }
}

// MARK: - Profile Setup Step
struct ProfileSetupStepView: View {
    @Binding var username: String
    @Binding var displayName: String
    let onContinue: () -> Void

    @State private var isUsernameAvailable: Bool?
    @State private var isCheckingUsername = false

    var body: some View {
        VStack(spacing: 32) {
            Spacer()

            VStack(spacing: 16) {
                Image(systemName: "person.crop.circle.fill")
                    .font(.system(size: 64))
                    .foregroundStyle(.blue)

                Text("Set Up Your Profile")
                    .font(.title)
                    .fontWeight(.bold)

                Text("Choose a username so friends can find you")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }

            VStack(spacing: 20) {
                // Username field
                VStack(alignment: .leading, spacing: 8) {
                    Text("Username")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)

                    HStack {
                        Text("@")
                            .foregroundStyle(.secondary)
                        TextField("username", text: $username)
                            .textInputAutocapitalization(.never)
                            .autocorrectionDisabled()
                            .onChange(of: username) { _, _ in
                                checkUsernameAvailability()
                            }

                        if isCheckingUsername {
                            ProgressView()
                                .scaleEffect(0.8)
                        } else if let available = isUsernameAvailable {
                            Image(systemName: available ? "checkmark.circle.fill" : "xmark.circle.fill")
                                .foregroundStyle(available ? .green : .red)
                        }
                    }
                    .padding()
                    .background(.ultraThinMaterial)
                    .clipShape(RoundedRectangle(cornerRadius: 12))
                }

                // Display name field
                VStack(alignment: .leading, spacing: 8) {
                    Text("Display Name (optional)")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)

                    TextField("Your name", text: $displayName)
                        .padding()
                        .background(.ultraThinMaterial)
                        .clipShape(RoundedRectangle(cornerRadius: 12))
                }
            }
            .padding(.horizontal)

            Spacer()

            Button(action: onContinue) {
                Text("Continue")
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 16)
                    .background(canContinue ? Color.blue : Color.gray.opacity(0.3))
                    .foregroundStyle(.white)
                    .clipShape(RoundedRectangle(cornerRadius: 14))
            }
            .disabled(!canContinue)
            .padding(.horizontal)

            Button {
                onContinue()
            } label: {
                Text("Skip for now")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }
            .padding(.bottom, 32)
        }
    }

    private var canContinue: Bool {
        username.isEmpty || (username.count >= 3 && isUsernameAvailable == true)
    }

    private func checkUsernameAvailability() {
        guard username.count >= 3 else {
            isUsernameAvailable = nil
            return
        }

        isCheckingUsername = true

        // Debounce
        Task {
            try? await Task.sleep(nanoseconds: 500_000_000)

            // TODO: Check with API
            // For now, simulate
            isUsernameAvailable = true
            isCheckingUsername = false
        }
    }
}

#Preview {
    OnboardingView()
        .environment(AuthManager.shared)
}
