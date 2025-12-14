//
//  SilentRelayContainer.swift
//  SilentRelay
//
//  SwiftData container configuration
//

import SwiftUI
import SwiftData

// MARK: - Container Configuration
enum SilentRelayContainer {

    /// The shared schema for all models
    static var schema: Schema {
        Schema([
            CachedConversation.self,
            CachedMessage.self,
            CachedContact.self
        ])
    }

    /// Create the model container
    static func create() throws -> ModelContainer {
        let configuration = ModelConfiguration(
            schema: schema,
            isStoredInMemoryOnly: false,
            allowsSave: true
        )

        return try ModelContainer(for: schema, configurations: [configuration])
    }

    /// Create an in-memory container for previews
    static func createPreview() throws -> ModelContainer {
        let configuration = ModelConfiguration(
            schema: schema,
            isStoredInMemoryOnly: true
        )

        let container = try ModelContainer(for: schema, configurations: [configuration])

        // Add sample data for previews
        let context = container.mainContext

        let sampleContact = CachedContact(
            id: "user-1",
            phoneNumber: "+1234567890",
            username: "johndoe",
            displayName: "John Doe",
            avatar: nil,
            publicKey: "sample-key",
            createdAt: Date()
        )
        context.insert(sampleContact)

        let sampleConversation = CachedConversation(
            id: "conv-1",
            recipientId: "user-1",
            recipientName: "John Doe",
            recipientAvatar: nil,
            lastMessageText: "Hey, how are you?",
            lastMessageTime: Date(),
            unreadCount: 2,
            isOnline: true,
            lastSeen: Date(),
            isPinned: false,
            isMuted: false,
            status: .accepted
        )
        context.insert(sampleConversation)

        let sampleMessage = CachedMessage(
            id: "msg-1",
            conversationId: "conv-1",
            senderId: "user-1",
            content: "Hey, how are you?",
            timestamp: Date(),
            status: .delivered,
            type: .text
        )
        context.insert(sampleMessage)

        return container
    }
}
