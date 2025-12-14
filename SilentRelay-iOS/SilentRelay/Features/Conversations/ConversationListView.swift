//
//  ConversationListView.swift
//  SilentRelay
//
//  Main conversation list view
//

import SwiftUI
import SwiftData

struct ConversationListView: View {
    @Environment(\.modelContext) private var modelContext
    @Query(sort: CachedConversation.defaultSort) private var conversations: [CachedConversation]

    @State private var searchText = ""
    @State private var showingNewChat = false
    @State private var selectedConversation: CachedConversation?

    var body: some View {
        NavigationSplitView {
            List(selection: $selectedConversation) {
                // Message requests section
                if hasMessageRequests {
                    Section {
                        NavigationLink {
                            MessageRequestsView()
                        } label: {
                            Label("Message Requests", systemImage: "person.badge.plus")
                                .badge(messageRequestCount)
                        }
                    }
                }

                // Conversations
                Section {
                    ForEach(filteredConversations) { conversation in
                        NavigationLink(value: conversation) {
                            ConversationRowView(conversation: conversation)
                        }
                        .swipeActions(edge: .trailing) {
                            Button(role: .destructive) {
                                deleteConversation(conversation)
                            } label: {
                                Label("Delete", systemImage: "trash")
                            }

                            Button {
                                toggleMute(conversation)
                            } label: {
                                Label(
                                    conversation.isMuted ? "Unmute" : "Mute",
                                    systemImage: conversation.isMuted ? "bell" : "bell.slash"
                                )
                            }
                            .tint(.orange)
                        }
                        .swipeActions(edge: .leading) {
                            Button {
                                togglePin(conversation)
                            } label: {
                                Label(
                                    conversation.isPinned ? "Unpin" : "Pin",
                                    systemImage: conversation.isPinned ? "pin.slash" : "pin"
                                )
                            }
                            .tint(.blue)
                        }
                    }
                }
            }
            .navigationTitle("Chats")
            .searchable(text: $searchText, prompt: "Search conversations")
            .toolbar {
                ToolbarItem(placement: .primaryAction) {
                    Button {
                        showingNewChat = true
                    } label: {
                        Image(systemName: "square.and.pencil")
                    }
                }
            }
            .sheet(isPresented: $showingNewChat) {
                ContactSearchView()
            }
            .overlay {
                if conversations.isEmpty {
                    ContentUnavailableView(
                        "No Conversations",
                        systemImage: "message",
                        description: Text("Start a new chat to begin messaging")
                    )
                }
            }
        } detail: {
            if let conversation = selectedConversation {
                ChatView(conversation: conversation)
            } else {
                ContentUnavailableView(
                    "Select a Chat",
                    systemImage: "message",
                    description: Text("Choose a conversation from the list")
                )
            }
        }
    }

    private var filteredConversations: [CachedConversation] {
        guard !searchText.isEmpty else {
            return conversations.filter { $0.status == .accepted }
        }

        return conversations.filter { conversation in
            conversation.status == .accepted &&
            conversation.recipientName.localizedCaseInsensitiveContains(searchText)
        }
    }

    private var hasMessageRequests: Bool {
        conversations.contains { $0.status == .pending }
    }

    private var messageRequestCount: Int {
        conversations.filter { $0.status == .pending }.count
    }

    private func deleteConversation(_ conversation: CachedConversation) {
        modelContext.delete(conversation)
    }

    private func toggleMute(_ conversation: CachedConversation) {
        conversation.isMuted.toggle()
    }

    private func togglePin(_ conversation: CachedConversation) {
        conversation.isPinned.toggle()
    }
}

// MARK: - Conversation Row View
struct ConversationRowView: View {
    let conversation: CachedConversation

    var body: some View {
        HStack(spacing: 12) {
            // Avatar
            ZStack(alignment: .bottomTrailing) {
                Circle()
                    .fill(Color.blue.opacity(0.2))
                    .frame(width: 50, height: 50)
                    .overlay {
                        Text(conversation.initials)
                            .font(.headline)
                            .foregroundStyle(.blue)
                    }

                // Online indicator
                if conversation.isOnline {
                    Circle()
                        .fill(.green)
                        .frame(width: 12, height: 12)
                        .overlay {
                            Circle()
                                .stroke(.background, lineWidth: 2)
                        }
                }
            }

            // Content
            VStack(alignment: .leading, spacing: 4) {
                HStack {
                    Text(conversation.recipientName)
                        .font(.headline)

                    if conversation.isMuted {
                        Image(systemName: "bell.slash.fill")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }

                    Spacer()

                    if let time = conversation.lastMessageTime {
                        Text(formatTime(time))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }

                HStack {
                    Text(conversation.lastMessageText ?? "No messages")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                        .lineLimit(1)

                    Spacer()

                    if conversation.unreadCount > 0 {
                        Text("\(conversation.unreadCount)")
                            .font(.caption2)
                            .fontWeight(.semibold)
                            .foregroundStyle(.white)
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(Color.blue)
                            .clipShape(Capsule())
                    }
                }
            }
        }
        .padding(.vertical, 4)
    }

    private func formatTime(_ date: Date) -> String {
        let calendar = Calendar.current
        let formatter = DateFormatter()

        if calendar.isDateInToday(date) {
            formatter.dateFormat = "h:mm a"
        } else if calendar.isDateInYesterday(date) {
            return "Yesterday"
        } else if calendar.isDate(date, equalTo: Date(), toGranularity: .weekOfYear) {
            formatter.dateFormat = "EEE"
        } else {
            formatter.dateFormat = "M/d/yy"
        }

        return formatter.string(from: date)
    }
}

// MARK: - Placeholder Views
struct MessageRequestsView: View {
    var body: some View {
        Text("Message Requests")
            .navigationTitle("Message Requests")
    }
}

struct ContactSearchView: View {
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            Text("Search for contacts")
                .navigationTitle("New Chat")
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("Cancel") {
                            dismiss()
                        }
                    }
                }
        }
    }
}

#Preview {
    ConversationListView()
        .modelContainer(try! SilentRelayContainer.createPreview())
}
