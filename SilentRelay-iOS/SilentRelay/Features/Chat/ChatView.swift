//
//  ChatView.swift
//  SilentRelay
//
//  Individual chat conversation view
//

import SwiftUI
import SwiftData

struct ChatView: View {
    let conversation: CachedConversation

    @Environment(\.modelContext) private var modelContext
    @Environment(AuthManager.self) private var authManager

    @Query private var messages: [CachedMessage]
    @State private var messageText = ""
    @State private var isShowingAttachments = false
    @State private var isShowingCallOptions = false

    init(conversation: CachedConversation) {
        self.conversation = conversation
        let conversationId = conversation.id
        _messages = Query(
            filter: #Predicate<CachedMessage> { $0.conversationId == conversationId },
            sort: \.timestamp
        )
    }

    var body: some View {
        VStack(spacing: 0) {
            // Messages list
            ScrollViewReader { proxy in
                ScrollView {
                    LazyVStack(spacing: 8) {
                        ForEach(messages) { message in
                            MessageBubbleView(
                                message: message,
                                isFromCurrentUser: message.senderId == authManager.currentUser?.id
                            )
                            .id(message.id)
                        }
                    }
                    .padding(.horizontal)
                    .padding(.vertical, 8)
                }
                .onChange(of: messages.count) { _, _ in
                    if let lastMessage = messages.last {
                        withAnimation {
                            proxy.scrollTo(lastMessage.id, anchor: .bottom)
                        }
                    }
                }
            }

            Divider()

            // Message input
            MessageInputView(
                text: $messageText,
                onSend: sendMessage,
                onAttachment: { isShowingAttachments = true }
            )
        }
        .navigationTitle(conversation.recipientName)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .principal) {
                VStack(spacing: 2) {
                    Text(conversation.recipientName)
                        .font(.headline)
                    if conversation.isOnline {
                        Text("Online")
                            .font(.caption)
                            .foregroundStyle(.green)
                    } else if let lastSeen = conversation.lastSeen {
                        Text("Last seen \(lastSeen, style: .relative)")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
            }

            ToolbarItemGroup(placement: .topBarTrailing) {
                Button {
                    isShowingCallOptions = true
                } label: {
                    Image(systemName: "phone.fill")
                }

                Menu {
                    Button {
                        // View contact
                    } label: {
                        Label("View Contact", systemImage: "person.circle")
                    }

                    Button {
                        // Mute
                    } label: {
                        Label(
                            conversation.isMuted ? "Unmute" : "Mute",
                            systemImage: conversation.isMuted ? "bell" : "bell.slash"
                        )
                    }

                    Divider()

                    Button(role: .destructive) {
                        // Block
                    } label: {
                        Label("Block", systemImage: "hand.raised")
                    }
                } label: {
                    Image(systemName: "ellipsis.circle")
                }
            }
        }
        .confirmationDialog("Call", isPresented: $isShowingCallOptions) {
            Button {
                startCall(video: false)
            } label: {
                Label("Voice Call", systemImage: "phone.fill")
            }

            Button {
                startCall(video: true)
            } label: {
                Label("Video Call", systemImage: "video.fill")
            }

            Button("Cancel", role: .cancel) {}
        }
        .sheet(isPresented: $isShowingAttachments) {
            AttachmentPickerView()
        }
    }

    private func sendMessage() {
        guard !messageText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            return
        }

        let text = messageText
        messageText = ""

        // Create local message
        let message = CachedMessage(
            id: UUID().uuidString,
            conversationId: conversation.id,
            senderId: authManager.currentUser?.id ?? "",
            content: text,
            timestamp: Date(),
            status: .sending,
            type: .text
        )

        modelContext.insert(message)

        // TODO: Send via WebSocket and Signal encryption
    }

    private func startCall(video: Bool) {
        // TODO: Implement call start
    }
}

// MARK: - Message Bubble View
struct MessageBubbleView: View {
    let message: CachedMessage
    let isFromCurrentUser: Bool

    var body: some View {
        HStack {
            if isFromCurrentUser { Spacer(minLength: 60) }

            VStack(alignment: isFromCurrentUser ? .trailing : .leading, spacing: 4) {
                // Message content
                switch message.type {
                case .text:
                    Text(message.content)
                        .padding(.horizontal, 12)
                        .padding(.vertical, 8)
                        .background(isFromCurrentUser ? Color.blue : Color(.systemGray5))
                        .foregroundStyle(isFromCurrentUser ? .white : .primary)
                        .clipShape(RoundedRectangle(cornerRadius: 18))

                case .image:
                    ImageMessageView(message: message)

                case .file:
                    FileMessageView(message: message)

                case .voice:
                    VoiceMessageView(message: message)

                case .video:
                    VideoMessageView(message: message)

                case .call:
                    CallMessageView(message: message)
                }

                // Time and status
                HStack(spacing: 4) {
                    Text(message.timestamp, style: .time)
                        .font(.caption2)
                        .foregroundStyle(.secondary)

                    if isFromCurrentUser {
                        MessageStatusIcon(status: message.status)
                    }
                }
            }

            if !isFromCurrentUser { Spacer(minLength: 60) }
        }
    }
}

// MARK: - Message Status Icon
struct MessageStatusIcon: View {
    let status: CachedMessageStatus

    var body: some View {
        switch status {
        case .sending:
            Image(systemName: "clock")
                .font(.caption2)
                .foregroundStyle(.secondary)
        case .sent:
            Image(systemName: "checkmark")
                .font(.caption2)
                .foregroundStyle(.secondary)
        case .delivered:
            Image(systemName: "checkmark")
                .font(.caption2)
                .foregroundStyle(.blue)
        case .read:
            Image(systemName: "checkmark")
                .font(.caption2)
                .foregroundStyle(.blue)
                .overlay {
                    Image(systemName: "checkmark")
                        .font(.caption2)
                        .foregroundStyle(.blue)
                        .offset(x: 4)
                }
        case .failed:
            Image(systemName: "exclamationmark.circle")
                .font(.caption2)
                .foregroundStyle(.red)
        }
    }
}

// MARK: - Message Input View
struct MessageInputView: View {
    @Binding var text: String
    let onSend: () -> Void
    let onAttachment: () -> Void

    var body: some View {
        HStack(spacing: 12) {
            Button(action: onAttachment) {
                Image(systemName: "plus.circle.fill")
                    .font(.title2)
                    .foregroundStyle(.blue)
            }

            TextField("Message", text: $text, axis: .vertical)
                .textFieldStyle(.plain)
                .padding(.horizontal, 12)
                .padding(.vertical, 8)
                .background(.ultraThinMaterial)
                .clipShape(RoundedRectangle(cornerRadius: 20))
                .lineLimit(1...5)

            Button(action: onSend) {
                Image(systemName: "arrow.up.circle.fill")
                    .font(.title2)
                    .foregroundStyle(text.isEmpty ? .gray : .blue)
            }
            .disabled(text.isEmpty)
        }
        .padding(.horizontal)
        .padding(.vertical, 8)
        .background(.bar)
    }
}

// MARK: - Placeholder Views for Media Messages
struct ImageMessageView: View {
    let message: CachedMessage
    var body: some View {
        RoundedRectangle(cornerRadius: 12)
            .fill(Color.gray.opacity(0.3))
            .frame(width: 200, height: 150)
            .overlay {
                Image(systemName: "photo")
                    .foregroundStyle(.secondary)
            }
    }
}

struct FileMessageView: View {
    let message: CachedMessage
    var body: some View {
        HStack {
            Image(systemName: "doc.fill")
            Text(message.fileMetadata?.fileName ?? "File")
        }
        .padding()
        .background(Color(.systemGray5))
        .clipShape(RoundedRectangle(cornerRadius: 12))
    }
}

struct VoiceMessageView: View {
    let message: CachedMessage
    var body: some View {
        HStack {
            Image(systemName: "play.circle.fill")
            Text("Voice message")
        }
        .padding()
        .background(Color(.systemGray5))
        .clipShape(RoundedRectangle(cornerRadius: 12))
    }
}

struct VideoMessageView: View {
    let message: CachedMessage
    var body: some View {
        RoundedRectangle(cornerRadius: 12)
            .fill(Color.gray.opacity(0.3))
            .frame(width: 200, height: 150)
            .overlay {
                Image(systemName: "play.circle.fill")
                    .font(.largeTitle)
                    .foregroundStyle(.white)
            }
    }
}

struct CallMessageView: View {
    let message: CachedMessage
    var body: some View {
        HStack {
            Image(systemName: message.callMetadata?.callType == .video ? "video.fill" : "phone.fill")
            Text(message.content)
        }
        .padding()
        .background(Color(.systemGray6))
        .clipShape(RoundedRectangle(cornerRadius: 12))
    }
}

struct AttachmentPickerView: View {
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            List {
                Button {
                    // Photo
                } label: {
                    Label("Photo Library", systemImage: "photo")
                }

                Button {
                    // Camera
                } label: {
                    Label("Camera", systemImage: "camera")
                }

                Button {
                    // File
                } label: {
                    Label("Document", systemImage: "doc")
                }
            }
            .navigationTitle("Attach")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
            }
        }
        .presentationDetents([.medium])
    }
}
