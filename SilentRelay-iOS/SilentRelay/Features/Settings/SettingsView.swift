//
//  SettingsView.swift
//  SilentRelay
//
//  App settings view
//

import SwiftUI

struct SettingsView: View {
    @Environment(AuthManager.self) private var authManager

    @State private var showingLogoutConfirmation = false

    var body: some View {
        NavigationStack {
            List {
                // Profile section
                Section {
                    NavigationLink {
                        ProfileSettingsView()
                    } label: {
                        HStack(spacing: 16) {
                            Circle()
                                .fill(Color.blue.opacity(0.2))
                                .frame(width: 60, height: 60)
                                .overlay {
                                    Text(authManager.currentUser?.initials ?? "?")
                                        .font(.title2)
                                        .fontWeight(.medium)
                                        .foregroundStyle(.blue)
                                }

                            VStack(alignment: .leading, spacing: 4) {
                                Text(authManager.currentUser?.effectiveDisplayName ?? "User")
                                    .font(.headline)

                                if let username = authManager.currentUser?.username {
                                    Text("@\(username)")
                                        .font(.subheadline)
                                        .foregroundStyle(.secondary)
                                }
                            }
                        }
                        .padding(.vertical, 4)
                    }
                }

                // Privacy & Security
                Section("Privacy & Security") {
                    NavigationLink {
                        PrivacySettingsView()
                    } label: {
                        Label("Privacy", systemImage: "hand.raised")
                    }

                    NavigationLink {
                        SecuritySettingsView()
                    } label: {
                        Label("Security", systemImage: "lock")
                    }

                    NavigationLink {
                        LinkedDevicesView()
                    } label: {
                        Label("Linked Devices", systemImage: "iphone.and.arrow.forward")
                    }
                }

                // Notifications
                Section("Notifications") {
                    NavigationLink {
                        NotificationSettingsView()
                    } label: {
                        Label("Notifications", systemImage: "bell")
                    }
                }

                // Appearance
                Section("Appearance") {
                    NavigationLink {
                        AppearanceSettingsView()
                    } label: {
                        Label("Theme", systemImage: "paintbrush")
                    }
                }

                // About
                Section("About") {
                    NavigationLink {
                        AboutView()
                    } label: {
                        Label("About SilentRelay", systemImage: "info.circle")
                    }

                    Link(destination: URL(string: "https://silentrelay.com/help")!) {
                        Label("Help Center", systemImage: "questionmark.circle")
                    }

                    Link(destination: URL(string: "https://silentrelay.com/privacy")!) {
                        Label("Privacy Policy", systemImage: "doc.text")
                    }

                    Link(destination: URL(string: "https://silentrelay.com/terms")!) {
                        Label("Terms of Service", systemImage: "doc.text")
                    }
                }

                // Logout
                Section {
                    Button(role: .destructive) {
                        showingLogoutConfirmation = true
                    } label: {
                        Label("Log Out", systemImage: "rectangle.portrait.and.arrow.right")
                    }
                }
            }
            .navigationTitle("Settings")
            .confirmationDialog(
                "Log Out",
                isPresented: $showingLogoutConfirmation,
                titleVisibility: .visible
            ) {
                Button("Log Out", role: .destructive) {
                    Task {
                        await authManager.logout()
                    }
                }
                Button("Cancel", role: .cancel) {}
            } message: {
                Text("Are you sure you want to log out?")
            }
        }
    }
}

// MARK: - Profile Settings
struct ProfileSettingsView: View {
    @Environment(AuthManager.self) private var authManager

    @State private var displayName = ""
    @State private var username = ""
    @State private var isSaving = false

    var body: some View {
        Form {
            Section("Profile Photo") {
                HStack {
                    Spacer()
                    Circle()
                        .fill(Color.blue.opacity(0.2))
                        .frame(width: 100, height: 100)
                        .overlay {
                            Text(authManager.currentUser?.initials ?? "?")
                                .font(.title)
                                .fontWeight(.medium)
                                .foregroundStyle(.blue)
                        }
                    Spacer()
                }
                .listRowBackground(Color.clear)

                Button("Change Photo") {
                    // Photo picker
                }
            }

            Section("Profile Info") {
                TextField("Display Name", text: $displayName)
                TextField("Username", text: $username)
                    .textInputAutocapitalization(.never)
            }

            Section {
                HStack {
                    Text("Phone")
                    Spacer()
                    Text(authManager.currentUser?.phoneNumber ?? "")
                        .foregroundStyle(.secondary)
                }
            }
        }
        .navigationTitle("Edit Profile")
        .toolbar {
            ToolbarItem(placement: .confirmationAction) {
                Button("Save") {
                    Task {
                        isSaving = true
                        try? await authManager.updateProfile(
                            username: username.isEmpty ? nil : username,
                            displayName: displayName.isEmpty ? nil : displayName,
                            avatar: nil
                        )
                        isSaving = false
                    }
                }
                .disabled(isSaving)
            }
        }
        .onAppear {
            displayName = authManager.currentUser?.displayName ?? ""
            username = authManager.currentUser?.username ?? ""
        }
    }
}

// MARK: - Privacy Settings
struct PrivacySettingsView: View {
    @State private var readReceipts = true
    @State private var onlineStatus = true
    @State private var lastSeen = true
    @State private var typingIndicators = true

    var body: some View {
        Form {
            Section {
                Toggle("Read Receipts", isOn: $readReceipts)
                Toggle("Online Status", isOn: $onlineStatus)
                Toggle("Last Seen", isOn: $lastSeen)
                Toggle("Typing Indicators", isOn: $typingIndicators)
            } footer: {
                Text("These settings control what others can see about your activity.")
            }

            Section("Blocked Users") {
                NavigationLink {
                    Text("Blocked users list")
                } label: {
                    Text("Blocked Users")
                }
            }
        }
        .navigationTitle("Privacy")
    }
}

// MARK: - Security Settings
struct SecuritySettingsView: View {
    @State private var useBiometric = true
    @State private var autoLock = true

    var body: some View {
        Form {
            Section {
                Toggle("Face ID / Touch ID", isOn: $useBiometric)
                Toggle("Auto-Lock", isOn: $autoLock)
            }

            Section {
                NavigationLink {
                    Text("Change PIN")
                } label: {
                    Text("Change PIN")
                }
            }

            Section {
                NavigationLink {
                    Text("Recovery Key")
                } label: {
                    Text("View Recovery Key")
                }
            } footer: {
                Text("Your recovery key can be used to restore your account if you forget your PIN.")
            }
        }
        .navigationTitle("Security")
    }
}

// MARK: - Linked Devices
struct LinkedDevicesView: View {
    var body: some View {
        Form {
            Section {
                HStack {
                    Image(systemName: "iphone")
                        .font(.title2)
                        .foregroundStyle(.blue)
                    VStack(alignment: .leading) {
                        Text("This Device")
                            .font(.headline)
                        Text("iPhone")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
            }

            Section {
                Button {
                    // Link new device
                } label: {
                    Label("Link New Device", systemImage: "qrcode.viewfinder")
                }
            }
        }
        .navigationTitle("Linked Devices")
    }
}

// MARK: - Notification Settings
struct NotificationSettingsView: View {
    @State private var notificationsEnabled = true
    @State private var soundEnabled = true
    @State private var previewEnabled = true

    var body: some View {
        Form {
            Section {
                Toggle("Notifications", isOn: $notificationsEnabled)
                Toggle("Sound", isOn: $soundEnabled)
                Toggle("Message Preview", isOn: $previewEnabled)
            }
        }
        .navigationTitle("Notifications")
    }
}

// MARK: - Appearance Settings
struct AppearanceSettingsView: View {
    @State private var selectedTheme: AppTheme = .system

    var body: some View {
        Form {
            Section {
                Picker("Theme", selection: $selectedTheme) {
                    Text("System").tag(AppTheme.system)
                    Text("Light").tag(AppTheme.light)
                    Text("Dark").tag(AppTheme.dark)
                }
                .pickerStyle(.inline)
            }
        }
        .navigationTitle("Appearance")
    }
}

// MARK: - About View
struct AboutView: View {
    var body: some View {
        Form {
            Section {
                HStack {
                    Text("Version")
                    Spacer()
                    Text("1.0.0")
                        .foregroundStyle(.secondary)
                }

                HStack {
                    Text("Build")
                    Spacer()
                    Text("1")
                        .foregroundStyle(.secondary)
                }
            }

            Section {
                Link("Website", destination: URL(string: "https://silentrelay.com")!)
                Link("Twitter", destination: URL(string: "https://twitter.com/silentrelay")!)
            }
        }
        .navigationTitle("About")
    }
}

#Preview {
    SettingsView()
        .environment(AuthManager.shared)
}
