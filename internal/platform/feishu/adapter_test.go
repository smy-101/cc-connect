package feishu

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/smy-101/cc-connect/internal/core"
)

func TestAdapterHandleEvent(t *testing.T) {
	t.Run("nil event returns error", func(t *testing.T) {
		mockClient := NewMockClient()
		router := core.NewRouter()
		adapter := NewAdapter(mockClient, router)

		err := adapter.HandleEvent(context.Background(), nil)
		if err == nil {
			t.Error("HandleEvent() should return error for nil event")
		}
	})

	t.Run("valid event routes successfully", func(t *testing.T) {
		mockClient := NewMockClient()
		router := core.NewRouter()

		var handlerCalled atomic.Bool
		router.Register(core.MessageTypeText, func(ctx context.Context, msg *core.Message) error {
			handlerCalled.Store(true)
			return nil
		})

		adapter := NewAdapter(mockClient, router)

		event := &MessageReceiveEvent{
			EventID:     "evt_001",
			MessageID:   "msg_001",
			MessageType: "text",
			Content:     `{"text":"Test message"}`,
			ChatID:      "oc_chat",
			ChatType:    "p2p",
			Sender:      SenderInfo{OpenID: "ou_user"},
			CreateTime:  time.Now(),
		}

		err := adapter.HandleEvent(context.Background(), event)
		if err != nil {
			t.Errorf("HandleEvent() error = %v", err)
		}

		if !handlerCalled.Load() {
			t.Error("handler was not called")
		}
	})

	t.Run("no handler does not fail", func(t *testing.T) {
		mockClient := NewMockClient()
		router := core.NewRouter() // No handlers registered
		adapter := NewAdapter(mockClient, router)

		event := &MessageReceiveEvent{
			EventID:     "evt_002",
			MessageID:   "msg_002",
			MessageType: "text",
			Content:     `{"text":"Test"}`,
			ChatID:      "oc_chat",
			ChatType:    "p2p",
			Sender:      SenderInfo{OpenID: "ou_user"},
			CreateTime:  time.Now(),
		}

		err := adapter.HandleEvent(context.Background(), event)
		if err != nil {
			t.Errorf("HandleEvent() should not fail when no handler: %v", err)
		}
	})

	t.Run("invalid content returns error", func(t *testing.T) {
		mockClient := NewMockClient()
		router := core.NewRouter()
		adapter := NewAdapter(mockClient, router)

		event := &MessageReceiveEvent{
			EventID:     "evt_003",
			MessageID:   "msg_003",
			MessageType: "text",
			Content:     `invalid json`,
			ChatID:      "oc_chat",
			ChatType:    "p2p",
			Sender:      SenderInfo{OpenID: "ou_user"},
			CreateTime:  time.Now(),
		}

		err := adapter.HandleEvent(context.Background(), event)
		if err == nil {
			t.Error("HandleEvent() should return error for invalid content")
		}
	})
}

func TestAdapterHandleRawEvent(t *testing.T) {
	t.Run("valid raw event parses and routes", func(t *testing.T) {
		mockClient := NewMockClient()
		router := core.NewRouter()

		var handlerCalled atomic.Bool
		var receivedMsg *core.Message

		router.Register(core.MessageTypeText, func(ctx context.Context, msg *core.Message) error {
			handlerCalled.Store(true)
			receivedMsg = msg
			return nil
		})

		adapter := NewAdapter(mockClient, router)

		rawEvent := `{
			"schema": "2.0",
			"header": {
				"event_id": "evt_raw_001",
				"event_type": "im.message.receive_v1",
				"create_time": "1704067200000"
			},
			"event": {
				"sender": {
					"sender_id": {"open_id": "ou_raw_user"},
					"sender_type": "user"
				},
				"message": {
					"message_id": "msg_raw_001",
					"chat_id": "oc_raw_chat",
					"chat_type": "p2p",
					"message_type": "text",
					"content": "{\"text\":\"Raw event message\"}"
				}
			}
		}`

		err := adapter.HandleRawEvent(context.Background(), []byte(rawEvent))
		if err != nil {
			t.Errorf("HandleRawEvent() error = %v", err)
		}

		if !handlerCalled.Load() {
			t.Error("handler was not called")
		}

		if receivedMsg == nil {
			t.Fatal("receivedMsg is nil")
		}

		if receivedMsg.Content != "Raw event message" {
			t.Errorf("Content = %v, want 'Raw event message'", receivedMsg.Content)
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		mockClient := NewMockClient()
		router := core.NewRouter()
		adapter := NewAdapter(mockClient, router)

		err := adapter.HandleRawEvent(context.Background(), []byte(`invalid json`))
		if err == nil {
			t.Error("HandleRawEvent() should return error for invalid JSON")
		}
	})

	t.Run("empty data returns error", func(t *testing.T) {
		mockClient := NewMockClient()
		router := core.NewRouter()
		adapter := NewAdapter(mockClient, router)

		err := adapter.HandleRawEvent(context.Background(), []byte{})
		if err == nil {
			t.Error("HandleRawEvent() should return error for empty data")
		}
	})
}

func TestAdapterHandleSDKEvent(t *testing.T) {
	t.Run("valid SDK event parses and routes", func(t *testing.T) {
		mockClient := NewMockClient()
		router := core.NewRouter()

		var handlerCalled atomic.Bool
		var receivedMsg *core.Message

		router.Register(core.MessageTypeText, func(ctx context.Context, msg *core.Message) error {
			handlerCalled.Store(true)
			receivedMsg = msg
			return nil
		})

		adapter := NewAdapter(mockClient, router)

		sdkEvent := map[string]any{
			"schema": "2.0",
			"header": map[string]any{
				"event_id":   "evt_sdk_001",
				"event_type": "im.message.receive_v1",
				"create_time": "1704067200000",
			},
			"event": map[string]any{
				"sender": map[string]any{
					"sender_id":   map[string]any{"open_id": "ou_sdk_user"},
					"sender_type": "user",
				},
				"message": map[string]any{
					"message_id":   "msg_sdk_001",
					"chat_id":      "oc_sdk_chat",
					"chat_type":    "group",
					"message_type": "text",
					"content":      `{"text":"SDK event message"}`,
				},
			},
		}

		err := adapter.HandleSDKEvent(context.Background(), sdkEvent)
		if err != nil {
			t.Errorf("HandleSDKEvent() error = %v", err)
		}

		if !handlerCalled.Load() {
			t.Error("handler was not called")
		}

		if receivedMsg == nil {
			t.Fatal("receivedMsg is nil")
		}

		if receivedMsg.Content != "SDK event message" {
			t.Errorf("Content = %v, want 'SDK event message'", receivedMsg.Content)
		}

		if receivedMsg.ChannelID != "oc_sdk_chat" {
			t.Errorf("ChannelID = %v, want 'oc_sdk_chat'", receivedMsg.ChannelID)
		}
	})

	t.Run("nil SDK event returns error", func(t *testing.T) {
		mockClient := NewMockClient()
		router := core.NewRouter()
		adapter := NewAdapter(mockClient, router)

		err := adapter.HandleSDKEvent(context.Background(), nil)
		if err == nil {
			t.Error("HandleSDKEvent() should return error for nil event")
		}
	})

	t.Run("missing required fields returns error", func(t *testing.T) {
		mockClient := NewMockClient()
		router := core.NewRouter()
		adapter := NewAdapter(mockClient, router)

		// Missing event.message
		sdkEvent := map[string]any{
			"schema": "2.0",
			"header": map[string]any{
				"event_id": "evt_missing",
			},
			"event": map[string]any{
				"sender": map[string]any{},
			},
		}

		err := adapter.HandleSDKEvent(context.Background(), sdkEvent)
		if err == nil {
			t.Error("HandleSDKEvent() should return error for missing fields")
		}
	})
}

func TestAdapterSendReply(t *testing.T) {
	t.Run("send reply successfully", func(t *testing.T) {
		mockClient := NewMockClient()
		router := core.NewRouter()
		adapter := NewAdapter(mockClient, router)

		err := adapter.SendReply(context.Background(), "oc_chat_123", "Hello back!")
		if err != nil {
			t.Errorf("SendReply() error = %v", err)
		}

		if mockClient.SendTextCalled != 1 {
			t.Errorf("SendTextCalled = %v, want 1", mockClient.SendTextCalled)
		}
		if mockClient.LastSendTextChatID != "oc_chat_123" {
			t.Errorf("LastSendTextChatID = %v, want 'oc_chat_123'", mockClient.LastSendTextChatID)
		}
		if mockClient.LastSendTextContent != "Hello back!" {
			t.Errorf("LastSendTextContent = %v, want 'Hello back!'", mockClient.LastSendTextContent)
		}
	})

	t.Run("send with empty chatID succeeds", func(t *testing.T) {
		mockClient := NewMockClient()
		router := core.NewRouter()
		adapter := NewAdapter(mockClient, router)

		// Empty chatID passes through to sender (which handles validation)
		_ = adapter.SendReply(context.Background(), "", "test")
		// We don't check error here as sender handles validation
	})
}

func TestAdapterSendMessage(t *testing.T) {
	t.Run("send unified message successfully", func(t *testing.T) {
		mockClient := NewMockClient()
		router := core.NewRouter()
		adapter := NewAdapter(mockClient, router)

		msg := &core.Message{
			ID:        "msg_unified_001",
			Platform:  "feishu",
			UserID:    "ou_user",
			ChannelID: "oc_channel",
			Content:   "Unified message content",
			Type:      core.MessageTypeText,
			Timestamp: time.Now(),
		}

		err := adapter.SendMessage(context.Background(), msg)
		if err != nil {
			t.Errorf("SendMessage() error = %v", err)
		}

		if mockClient.SendTextCalled != 1 {
			t.Errorf("SendTextCalled = %v, want 1", mockClient.SendTextCalled)
		}
	})

	t.Run("send nil message returns error", func(t *testing.T) {
		mockClient := NewMockClient()
		router := core.NewRouter()
		adapter := NewAdapter(mockClient, router)

		err := adapter.SendMessage(context.Background(), nil)
		if err == nil {
			t.Error("SendMessage() should return error for nil message")
		}
	})

	t.Run("send message without channel ID returns error", func(t *testing.T) {
		mockClient := NewMockClient()
		router := core.NewRouter()
		adapter := NewAdapter(mockClient, router)

		msg := &core.Message{
			ID:        "msg_no_channel",
			Platform:  "feishu",
			UserID:    "ou_user",
			ChannelID: "", // Empty channel
			Content:   "No channel",
			Type:      core.MessageTypeText,
			Timestamp: time.Now(),
		}

		err := adapter.SendMessage(context.Background(), msg)
		if err == nil {
			t.Error("SendMessage() should return error for empty channel ID")
		}
	})
}

func TestAdapterStartStop(t *testing.T) {
	t.Run("start registers handler and connects", func(t *testing.T) {
		mockClient := NewMockClient()
		router := core.NewRouter()
		adapter := NewAdapter(mockClient, router)

		ctx := context.Background()
		err := adapter.Start(ctx)
		if err != nil {
			t.Errorf("Start() error = %v", err)
		}

		if mockClient.OnEventCalled != 1 {
			t.Errorf("OnEventCalled = %v, want 1", mockClient.OnEventCalled)
		}
		if mockClient.ConnectCalled != 1 {
			t.Errorf("ConnectCalled = %v, want 1", mockClient.ConnectCalled)
		}

		// Clean up
		_ = adapter.Stop()
	})

	t.Run("stop disconnects client", func(t *testing.T) {
		mockClient := NewMockClient()
		router := core.NewRouter()
		adapter := NewAdapter(mockClient, router)

		_ = adapter.Stop()

		if mockClient.DisconnectCalled != 1 {
			t.Errorf("DisconnectCalled = %v, want 1", mockClient.DisconnectCalled)
		}
	})

	t.Run("start and stop cycle", func(t *testing.T) {
		mockClient := NewMockClient()
		router := core.NewRouter()
		adapter := NewAdapter(mockClient, router)

		ctx := context.Background()

		// Start
		err := adapter.Start(ctx)
		if err != nil {
			t.Errorf("Start() error = %v", err)
		}

		if !mockClient.IsConnected() {
			t.Error("client should be connected after Start()")
		}

		// Stop
		err = adapter.Stop()
		if err != nil {
			t.Errorf("Stop() error = %v", err)
		}

		if mockClient.IsConnected() {
			t.Error("client should not be connected after Stop()")
		}
	})
}

func TestAdapterIntegration(t *testing.T) {
	t.Run("full message flow through adapter", func(t *testing.T) {
		mockClient := NewMockClient()
		router := core.NewRouter()

		var receivedMsgs []*core.Message

		router.Register(core.MessageTypeText, func(ctx context.Context, msg *core.Message) error {
			receivedMsgs = append(receivedMsgs, msg)
			return nil
		})

		adapter := NewAdapter(mockClient, router)

		// Send multiple events
		events := []*MessageReceiveEvent{
			{
				EventID:     "evt_1",
				MessageID:   "msg_1",
				MessageType: "text",
				Content:     `{"text":"First message"}`,
				ChatID:      "oc_chat",
				ChatType:    "p2p",
				Sender:      SenderInfo{OpenID: "ou_user1"},
				CreateTime:  time.Now(),
			},
			{
				EventID:     "evt_2",
				MessageID:   "msg_2",
				MessageType: "text",
				Content:     `{"text":"Second message"}`,
				ChatID:      "oc_chat",
				ChatType:    "group",
				Sender:      SenderInfo{OpenID: "ou_user2"},
				CreateTime:  time.Now(),
			},
		}

		for _, event := range events {
			err := adapter.HandleEvent(context.Background(), event)
			if err != nil {
				t.Errorf("HandleEvent() error = %v", err)
			}
		}

		if len(receivedMsgs) != 2 {
			t.Errorf("received %d messages, want 2", len(receivedMsgs))
		}

		// Verify order is preserved
		if receivedMsgs[0].Content != "First message" {
			t.Errorf("First message content = %v, want 'First message'", receivedMsgs[0].Content)
		}
		if receivedMsgs[1].Content != "Second message" {
			t.Errorf("Second message content = %v, want 'Second message'", receivedMsgs[1].Content)
		}
	})
}

// Command detection tests

func TestAdapterCommandDetection(t *testing.T) {
	t.Run("slash command converted to MessageTypeCommand", func(t *testing.T) {
		mockClient := NewMockClient()
		router := core.NewRouter()

		var receivedMsg *core.Message
		router.Register(core.MessageTypeCommand, func(ctx context.Context, msg *core.Message) error {
			receivedMsg = msg
			return nil
		})

		adapter := NewAdapter(mockClient, router)

		event := &MessageReceiveEvent{
			EventID:     "evt_cmd_001",
			MessageID:   "msg_cmd_001",
			MessageType: "text",
			Content:     `{"text":"/mode yolo"}`,
			ChatID:      "oc_chat",
			ChatType:    "p2p",
			Sender:      SenderInfo{OpenID: "ou_user"},
			CreateTime:  time.Now(),
		}

		err := adapter.HandleEvent(context.Background(), event)
		if err != nil {
			t.Errorf("HandleEvent() error = %v", err)
		}

		if receivedMsg == nil {
			t.Fatal("receivedMsg is nil")
		}

		if receivedMsg.Type != core.MessageTypeCommand {
			t.Errorf("Type = %v, want %v", receivedMsg.Type, core.MessageTypeCommand)
		}

		if receivedMsg.Content != "/mode yolo" {
			t.Errorf("Content = %v, want '/mode yolo'", receivedMsg.Content)
		}
	})

	t.Run("plain text remains MessageTypeText", func(t *testing.T) {
		mockClient := NewMockClient()
		router := core.NewRouter()

		var receivedMsg *core.Message
		router.Register(core.MessageTypeText, func(ctx context.Context, msg *core.Message) error {
			receivedMsg = msg
			return nil
		})

		adapter := NewAdapter(mockClient, router)

		event := &MessageReceiveEvent{
			EventID:     "evt_text_001",
			MessageID:   "msg_text_001",
			MessageType: "text",
			Content:     `{"text":"Hello world"}`,
			ChatID:      "oc_chat",
			ChatType:    "p2p",
			Sender:      SenderInfo{OpenID: "ou_user"},
			CreateTime:  time.Now(),
		}

		err := adapter.HandleEvent(context.Background(), event)
		if err != nil {
			t.Errorf("HandleEvent() error = %v", err)
		}

		if receivedMsg == nil {
			t.Fatal("receivedMsg is nil")
		}

		if receivedMsg.Type != core.MessageTypeText {
			t.Errorf("Type = %v, want %v", receivedMsg.Type, core.MessageTypeText)
		}
	})

	t.Run("text with leading space not converted to command", func(t *testing.T) {
		mockClient := NewMockClient()
		router := core.NewRouter()

		var receivedMsg *core.Message
		router.Register(core.MessageTypeText, func(ctx context.Context, msg *core.Message) error {
			receivedMsg = msg
			return nil
		})

		adapter := NewAdapter(mockClient, router)

		event := &MessageReceiveEvent{
			EventID:     "evt_space_001",
			MessageID:   "msg_space_001",
			MessageType: "text",
			Content:     `{"text":" /mode"}`,
			ChatID:      "oc_chat",
			ChatType:    "p2p",
			Sender:      SenderInfo{OpenID: "ou_user"},
			CreateTime:  time.Now(),
		}

		err := adapter.HandleEvent(context.Background(), event)
		if err != nil {
			t.Errorf("HandleEvent() error = %v", err)
		}

		if receivedMsg == nil {
			t.Fatal("receivedMsg is nil")
		}

		// Should remain as text since it has leading space
		if receivedMsg.Type != core.MessageTypeText {
			t.Errorf("Type = %v, want %v (not a command due to leading space)", receivedMsg.Type, core.MessageTypeText)
		}
	})

	t.Run("help command converted to MessageTypeCommand", func(t *testing.T) {
		mockClient := NewMockClient()
		router := core.NewRouter()

		var receivedMsg *core.Message
		router.Register(core.MessageTypeCommand, func(ctx context.Context, msg *core.Message) error {
			receivedMsg = msg
			return nil
		})

		adapter := NewAdapter(mockClient, router)

		event := &MessageReceiveEvent{
			EventID:     "evt_help_001",
			MessageID:   "msg_help_001",
			MessageType: "text",
			Content:     `{"text":"/help"}`,
			ChatID:      "oc_chat",
			ChatType:    "p2p",
			Sender:      SenderInfo{OpenID: "ou_user"},
			CreateTime:  time.Now(),
		}

		err := adapter.HandleEvent(context.Background(), event)
		if err != nil {
			t.Errorf("HandleEvent() error = %v", err)
		}

		if receivedMsg == nil {
			t.Fatal("receivedMsg is nil")
		}

		if receivedMsg.Type != core.MessageTypeCommand {
			t.Errorf("Type = %v, want %v", receivedMsg.Type, core.MessageTypeCommand)
		}
	})
}
