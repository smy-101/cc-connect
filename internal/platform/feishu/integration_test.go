package feishu

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/smy-101/cc-connect/internal/core"
)

func TestFeishuToRouter(t *testing.T) {
	t.Run("event converted and routed to handler", func(t *testing.T) {
		// Create mock client
		mockClient := NewMockClient()

		// Create router
		router := core.NewRouter()

		// Track if handler was called
		var handlerCalled atomic.Bool
		var receivedMessage *core.Message

		// Register handler for text messages
		router.Register(core.MessageTypeText, func(ctx context.Context, msg *core.Message) error {
			handlerCalled.Store(true)
			receivedMessage = msg
			return nil
		})

		// Create adapter
		adapter := NewAdapter(mockClient, router)

		// Simulate receiving a message event
		event := &MessageReceiveEvent{
			EventID:     "evt_integration_001",
			MessageID:   "msg_integration_001",
			MessageType: "text",
			Content:     `{"text":"Hello from Feishu!"}`,
			ChatID:      "oc_integration_chat",
			ChatType:    "group",
			Sender: SenderInfo{
				OpenID:     "ou_integration_user",
				SenderType: "user",
			},
			CreateTime: time.Now(),
		}

		// Handle the event
		err := adapter.HandleEvent(context.Background(), event)
		if err != nil {
			t.Errorf("HandleEvent() error = %v", err)
		}

		// Verify handler was called
		if !handlerCalled.Load() {
			t.Error("handler was not called")
		}

		// Verify message was converted correctly
		if receivedMessage == nil {
			t.Fatal("receivedMessage is nil")
		}
		if receivedMessage.Content != "Hello from Feishu!" {
			t.Errorf("Content = %v, want 'Hello from Feishu!'", receivedMessage.Content)
		}
		if receivedMessage.Platform != "feishu" {
			t.Errorf("Platform = %v, want 'feishu'", receivedMessage.Platform)
		}
		if receivedMessage.UserID != "ou_integration_user" {
			t.Errorf("UserID = %v, want 'ou_integration_user'", receivedMessage.UserID)
		}
		if receivedMessage.ChannelID != "oc_integration_chat" {
			t.Errorf("ChannelID = %v, want 'oc_integration_chat'", receivedMessage.ChannelID)
		}
	})

	t.Run("unsupported message type not routed", func(t *testing.T) {
		mockClient := NewMockClient()
		router := core.NewRouter()

		// Only register text handler
		router.Register(core.MessageTypeText, func(ctx context.Context, msg *core.Message) error {
			return nil
		})

		adapter := NewAdapter(mockClient, router)

		// Send an image event (not supported)
		event := &MessageReceiveEvent{
			EventID:     "evt_image",
			MessageID:   "msg_image",
			MessageType: "image",
			Content:     `{"image_key":"img_123"}`,
			ChatID:      "oc_chat",
			ChatType:    "p2p",
			Sender:      SenderInfo{OpenID: "ou_user"},
			CreateTime:  time.Now(),
		}

		err := adapter.HandleEvent(context.Background(), event)
		if err == nil {
			t.Error("HandleEvent() should return error for unsupported message type")
		}
	})

	t.Run("send reply through adapter", func(t *testing.T) {
		mockClient := NewMockClient()
		router := core.NewRouter()

		adapter := NewAdapter(mockClient, router)

		// Send a reply
		err := adapter.SendReply(context.Background(), "oc_reply_chat", "Reply message")
		if err != nil {
			t.Errorf("SendReply() error = %v", err)
		}

		// Verify mock client received the call
		if mockClient.SendTextCalled != 1 {
			t.Errorf("SendTextCalled = %v, want 1", mockClient.SendTextCalled)
		}
		if mockClient.LastSendTextChatID != "oc_reply_chat" {
			t.Errorf("LastSendTextChatID = %v, want 'oc_reply_chat'", mockClient.LastSendTextChatID)
		}
		if mockClient.LastSendTextContent != "Reply message" {
			t.Errorf("LastSendTextContent = %v, want 'Reply message'", mockClient.LastSendTextContent)
		}
	})

	t.Run("full round trip", func(t *testing.T) {
		mockClient := NewMockClient()
		router := core.NewRouter()

		var lastReceivedMessage *core.Message

		// Register handler that echoes back
		router.Register(core.MessageTypeText, func(ctx context.Context, msg *core.Message) error {
			lastReceivedMessage = msg
			// Echo back
			return nil
		})

		adapter := NewAdapter(mockClient, router)

		// Simulate receiving a message
		event := &MessageReceiveEvent{
			EventID:     "evt_roundtrip",
			MessageID:   "msg_roundtrip",
			MessageType: "text",
			Content:     `{"text":"Ping"}`,
			ChatID:      "oc_roundtrip_chat",
			ChatType:    "p2p",
			Sender:      SenderInfo{OpenID: "ou_sender"},
			CreateTime:  time.Now(),
		}

		err := adapter.HandleEvent(context.Background(), event)
		if err != nil {
			t.Errorf("HandleEvent() error = %v", err)
		}

		// Verify message was received
		if lastReceivedMessage == nil || lastReceivedMessage.Content != "Ping" {
			t.Errorf("message not received correctly: %+v", lastReceivedMessage)
		}

		// Send reply
		err = adapter.SendReply(context.Background(), lastReceivedMessage.ChannelID, "Pong")
		if err != nil {
			t.Errorf("SendReply() error = %v", err)
		}

		// Verify reply was sent
		if mockClient.LastSendTextContent != "Pong" {
			t.Errorf("LastSendTextContent = %v, want 'Pong'", mockClient.LastSendTextContent)
		}
	})
}
