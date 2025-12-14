//
//  SilentRelayApp.swift
//  SilentRelay
//
//  Secure messaging app with end-to-end encryption
//

import SwiftUI
import SwiftData

@main
struct SilentRelayApp: App {
    @UIApplicationDelegateAdaptor(AppDelegate.self) var appDelegate

    @State private var authManager = AuthManager.shared
    @State private var webSocketManager = WebSocketManager.shared

    var sharedModelContainer: ModelContainer = {
        let schema = Schema([
            CachedConversation.self,
            CachedMessage.self,
            CachedContact.self
        ])
        let modelConfiguration = ModelConfiguration(
            schema: schema,
            isStoredInMemoryOnly: false,
            allowsSave: true
        )

        do {
            return try ModelContainer(for: schema, configurations: [modelConfiguration])
        } catch {
            fatalError("Could not create ModelContainer: \(error)")
        }
    }()

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environment(authManager)
                .environment(webSocketManager)
        }
        .modelContainer(sharedModelContainer)
    }
}

// MARK: - Content View (Root Navigation)
struct ContentView: View {
    @Environment(AuthManager.self) private var authManager

    var body: some View {
        Group {
            switch authManager.authState {
            case .loggedOut:
                PhoneEntryView()
            case .verifying:
                VerificationView()
            case .registering:
                OnboardingView()
            case .needsPinUnlock:
                PINUnlockView()
            case .authenticated:
                MainTabView()
            }
        }
        .animation(.easeInOut, value: authManager.authState)
    }
}

// MARK: - Main Tab View
struct MainTabView: View {
    @State private var selectedTab = 0

    var body: some View {
        TabView(selection: $selectedTab) {
            ConversationListView()
                .tabItem {
                    Label("Chats", systemImage: "message.fill")
                }
                .tag(0)

            SettingsView()
                .tabItem {
                    Label("Settings", systemImage: "gear")
                }
                .tag(1)
        }
        .tint(.blue)
    }
}

// MARK: - Preview
#Preview {
    ContentView()
        .environment(AuthManager.shared)
        .environment(WebSocketManager.shared)
}
