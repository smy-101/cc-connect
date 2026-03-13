// Package feishu provides a platform adapter for Feishu (飞书) messaging platform.
// It implements the WebSocket long connection client for receiving messages
// and sending replies without requiring a public IP address.
package feishu

import (
	"context"
	"time"
)

// FeishuClient defines the interface for Feishu platform client.
// This interface abstracts the SDK implementation to enable mocking in tests.
type FeishuClient interface {
	// Connect establishes a WebSocket long connection with Feishu platform.
	// This is a blocking call that maintains the connection.
	Connect(ctx context.Context) error

	// Disconnect closes the WebSocket connection.
	Disconnect() error

	// IsConnected returns true if the WebSocket connection is active.
	IsConnected() bool

	// SendText sends a text message to the specified chat.
	// The content should be in Feishu format: `{"text":"message content"}`
	SendText(ctx context.Context, chatID, content string) error

	// OnEvent registers an event handler for message events.
	OnEvent(handler EventHandler)
}

// EventHandler is the callback function type for handling message events.
type EventHandler func(ctx context.Context, event *MessageReceiveEvent) error

// MessageReceiveEvent represents a parsed im.message.receive_v1 event.
type MessageReceiveEvent struct {
	// EventID is the unique identifier for this event.
	EventID string

	// MessageID is the unique identifier for the message.
	MessageID string

	// MessageType is the type of message: "text", "post", "image", "audio", etc.
	MessageType string

	// Content is the raw JSON content string of the message.
	Content string

	// ChatID is the identifier of the chat (session).
	ChatID string

	// ChatType is the type of chat: "p2p", "group", "topic_group".
	ChatType string

	// Sender contains information about the message sender.
	Sender SenderInfo

	// Mentions contains the list of @mentions in the message.
	Mentions []MentionInfo

	// CreateTime is when the message was created.
	CreateTime time.Time

	// RawEvent contains the original SDK event for debugging purposes.
	RawEvent any
}

// SenderInfo contains information about the message sender.
type SenderInfo struct {
	// OpenID is the sender's open_id.
	OpenID string

	// UnionID is the sender's union_id (may be empty if no permission).
	UnionID string

	// UserID is the sender's user_id (may be empty if no permission).
	UserID string

	// SenderType is the type of sender, typically "user".
	SenderType string
}

// MentionInfo contains information about an @mentioned user.
type MentionInfo struct {
	// Key is the mention key used in the message content (e.g., "@_user_1").
	Key string

	// OpenID is the mentioned user's open_id.
	OpenID string

	// UnionID is the mentioned user's union_id.
	UnionID string

	// UserID is the mentioned user's user_id.
	UserID string

	// Name is the display name of the mentioned user.
	Name string

	// TenantKey is the tenant key of the mentioned user.
	TenantKey string
}
