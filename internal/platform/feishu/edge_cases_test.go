package feishu

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/smy-101/cc-connect/internal/core"
)

func TestSDKClientAdditional(t *testing.T) {
	t.Run("connect with valid credentials", func(t *testing.T) {
		facade := &fakeSDKFacade{
			startFunc: func(ctx context.Context, callbacks sdkFacadeCallbacks) error {
				callbacks.OnReady()
				<-ctx.Done()
				return nil
			},
		}
		client := newSDKClientWithFacade("valid_app_id", "valid_app_secret", facade)

		ctx := context.Background()
		err := client.Connect(ctx)
		if err != nil {
			t.Errorf("Connect() error = %v", err)
		}

		if !client.IsConnected() {
			t.Error("IsConnected() should return true after successful connect")
		}

		_ = client.Disconnect()
	})

	t.Run("connect with empty credentials fails", func(t *testing.T) {
		client := newSDKClientWithFacade("", "", &fakeSDKFacade{})

		ctx := context.Background()
		err := client.Connect(ctx)
		if err == nil {
			t.Error("Connect() should fail with empty credentials")
		}
	})

	t.Run("disconnect without connect", func(t *testing.T) {
		client := newSDKClientWithFacade("app_id", "app_secret", &fakeSDKFacade{})

		err := client.Disconnect()
		if err != nil {
			t.Errorf("Disconnect() error = %v", err)
		}
	})

	t.Run("double connect", func(t *testing.T) {
		facade := &fakeSDKFacade{
			startFunc: func(ctx context.Context, callbacks sdkFacadeCallbacks) error {
				callbacks.OnReady()
				<-ctx.Done()
				return nil
			},
		}
		client := newSDKClientWithFacade("app_id", "app_secret", facade)

		ctx := context.Background()
		_ = client.Connect(ctx)
		err := client.Connect(ctx) // Second connect should be no-op
		if err != nil {
			t.Errorf("Second Connect() error = %v", err)
		}

		_ = client.Disconnect()
	})

	t.Run("send without connection fails", func(t *testing.T) {
		client := newSDKClientWithFacade("app_id", "app_secret", &fakeSDKFacade{})

		ctx := context.Background()
		err := client.SendText(ctx, "oc_chat", "Hello")
		if !errors.Is(err, ErrClientNotReady) {
			t.Errorf("SendText() error = %v, want %v", err, ErrClientNotReady)
		}
	})

	t.Run("send with connection succeeds", func(t *testing.T) {
		facade := &fakeSDKFacade{
			startFunc: func(ctx context.Context, callbacks sdkFacadeCallbacks) error {
				callbacks.OnReady()
				<-ctx.Done()
				return nil
			},
		}
		client := newSDKClientWithFacade("app_id", "app_secret", facade)

		ctx := context.Background()
		_ = client.Connect(ctx)

		err := client.SendText(ctx, "oc_chat", "Hello")
		if err != nil {
			t.Errorf("SendText() error = %v", err)
		}

		_ = client.Disconnect()
	})

	t.Run("on event registration", func(t *testing.T) {
		client := newSDKClientWithFacade("app_id", "app_secret", &fakeSDKFacade{})

		handler := func(ctx context.Context, event *MessageReceiveEvent) error {
			return nil
		}

		client.OnEvent(handler)
		// Handler is registered (no error means success)
	})
}

func TestMessageConverterEdgeCases(t *testing.T) {
	converter := NewMessageConverter()

	t.Run("convert nil event", func(t *testing.T) {
		_, err := converter.ToUnifiedMessage(nil)
		if err == nil {
			t.Error("ToUnifiedMessage() should return error for nil event")
		}
	})

	t.Run("convert with empty sender openid", func(t *testing.T) {
		event := &MessageReceiveEvent{
			EventID:     "evt_empty_sender",
			MessageID:   "msg_empty",
			MessageType: "text",
			Content:     `{"text":"Test"}`,
			ChatID:      "oc_chat",
			ChatType:    "p2p",
			Sender:      SenderInfo{OpenID: ""}, // Empty OpenID
			CreateTime:  time.Now(),
		}

		msg, err := converter.ToUnifiedMessage(event)
		if err != nil {
			t.Errorf("ToUnifiedMessage() error = %v", err)
		}
		if msg.UserID != "" {
			t.Errorf("UserID should be empty, got %v", msg.UserID)
		}
	})

	t.Run("convert with zero create time", func(t *testing.T) {
		event := &MessageReceiveEvent{
			EventID:     "evt_zero_time",
			MessageID:   "msg_zero",
			MessageType: "text",
			Content:     `{"text":"Test"}`,
			ChatID:      "oc_chat",
			ChatType:    "p2p",
			Sender:      SenderInfo{OpenID: "ou_user"},
			CreateTime:  time.Time{}, // Zero time
		}

		msg, err := converter.ToUnifiedMessage(event)
		if err != nil {
			t.Errorf("ToUnifiedMessage() error = %v", err)
		}
		if msg.Timestamp.IsZero() {
			t.Error("Timestamp should not be zero (should use current time)")
		}
	})

	t.Run("convert with unicode content", func(t *testing.T) {
		event := &MessageReceiveEvent{
			EventID:     "evt_unicode",
			MessageID:   "msg_unicode",
			MessageType: "text",
			Content:     `{"text":"你好世界 🌍 Hello"}`,
			ChatID:      "oc_chat",
			ChatType:    "p2p",
			Sender:      SenderInfo{OpenID: "ou_user"},
			CreateTime:  time.Now(),
		}

		msg, err := converter.ToUnifiedMessage(event)
		if err != nil {
			t.Errorf("ToUnifiedMessage() error = %v", err)
		}
		expected := "你好世界 🌍 Hello"
		if msg.Content != expected {
			t.Errorf("Content = %v, want %v", msg.Content, expected)
		}
	})

	t.Run("convert with newlines and tabs", func(t *testing.T) {
		event := &MessageReceiveEvent{
			EventID:     "evt_newlines",
			MessageID:   "msg_newlines",
			MessageType: "text",
			Content:     `{"text":"Line1\nLine2\tTab"}`,
			ChatID:      "oc_chat",
			ChatType:    "p2p",
			Sender:      SenderInfo{OpenID: "ou_user"},
			CreateTime:  time.Now(),
		}

		msg, err := converter.ToUnifiedMessage(event)
		if err != nil {
			t.Errorf("ToUnifiedMessage() error = %v", err)
		}
		expected := "Line1\nLine2\tTab"
		if msg.Content != expected {
			t.Errorf("Content = %v, want %v", msg.Content, expected)
		}
	})

	t.Run("ToFeishuContent with nil message", func(t *testing.T) {
		_, err := converter.ToFeishuContent(nil)
		if err == nil {
			t.Error("ToFeishuContent() should return error for nil message")
		}
	})

	t.Run("ToFeishuContent with unsupported type", func(t *testing.T) {
		msg := &core.Message{
			ID:        "msg_unsupported",
			Platform:  "feishu",
			UserID:    "ou_user",
			ChannelID: "oc_chat",
			Content:   "test",
			Type:      core.MessageTypeImage, // Unsupported for sending
			Timestamp: time.Now(),
		}

		_, err := converter.ToFeishuContent(msg)
		if err == nil {
			t.Error("ToFeishuContent() should return error for unsupported type")
		}
	})

	t.Run("GetMentions with nil event", func(t *testing.T) {
		mentions := converter.GetMentions(nil)
		if mentions != nil {
			t.Errorf("GetMentions(nil) = %v, want nil", mentions)
		}
	})

	t.Run("GetMentions with empty mentions", func(t *testing.T) {
		event := &MessageReceiveEvent{
			EventID:     "evt_no_mentions",
			MessageID:   "msg_no_mentions",
			MessageType: "text",
			Content:     `{"text":"No mentions"}`,
			ChatID:      "oc_chat",
			ChatType:    "p2p",
			Sender:      SenderInfo{OpenID: "ou_user"},
			CreateTime:  time.Now(),
			Mentions:    []MentionInfo{}, // Empty slice
		}

		mentions := converter.GetMentions(event)
		if mentions != nil {
			t.Errorf("GetMentions() = %v, want nil", mentions)
		}
	})
}

func TestEventParserEdgeCases(t *testing.T) {
	parser := NewEventParser()

	t.Run("parse with minimal fields", func(t *testing.T) {
		data := `{
			"schema": "2.0",
			"header": {"event_id": "evt_minimal"},
			"event": {
				"sender": {"sender_id": {"open_id": "ou_user"}},
				"message": {
					"message_id": "msg_minimal",
					"chat_id": "oc_chat",
					"chat_type": "p2p",
					"message_type": "text",
					"content": "{\"text\":\"Minimal\"}"
				}
			}
		}`

		event, err := parser.Parse([]byte(data))
		if err != nil {
			t.Errorf("Parse() error = %v", err)
		}
		if event.EventID != "evt_minimal" {
			t.Errorf("EventID = %v, want evt_minimal", event.EventID)
		}
	})

	t.Run("parse with topic_group chat type", func(t *testing.T) {
		data := `{
			"schema": "2.0",
			"header": {"event_id": "evt_topic"},
			"event": {
				"sender": {"sender_id": {"open_id": "ou_user"}},
				"message": {
					"message_id": "msg_topic",
					"chat_id": "oc_topic",
					"chat_type": "topic_group",
					"message_type": "text",
					"content": "{\"text\":\"Topic\"}"
				}
			}
		}`

		event, err := parser.Parse([]byte(data))
		if err != nil {
			t.Errorf("Parse() error = %v", err)
		}
		if event.ChatType != "topic_group" {
			t.Errorf("ChatType = %v, want topic_group", event.ChatType)
		}
	})

	t.Run("parse with missing header event_id", func(t *testing.T) {
		data := `{
			"schema": "2.0",
			"header": {},
			"event": {
				"sender": {"sender_id": {}},
				"message": {
					"message_id": "msg_no_evt",
					"chat_id": "oc_chat",
					"chat_type": "p2p",
					"message_type": "text",
					"content": "{\"text\":\"Test\"}"
				}
			}
		}`

		_, err := parser.Parse([]byte(data))
		if err == nil {
			t.Error("Parse() should return error for missing event_id")
		}
	})

	t.Run("parse with missing message_id", func(t *testing.T) {
		data := `{
			"schema": "2.0",
			"header": {"event_id": "evt_no_msg"},
			"event": {
				"sender": {"sender_id": {"open_id": "ou_user"}},
				"message": {
					"chat_id": "oc_chat",
					"chat_type": "p2p",
					"message_type": "text",
					"content": "{\"text\":\"Test\"}"
				}
			}
		}`

		_, err := parser.Parse([]byte(data))
		if err == nil {
			t.Error("Parse() should return error for missing message_id")
		}
	})

	t.Run("ParseFromMap with nil", func(t *testing.T) {
		_, err := parser.ParseFromMap(nil)
		if err == nil {
			t.Error("ParseFromMap(nil) should return error")
		}
	})

	t.Run("ExtractPostText with non-post message", func(t *testing.T) {
		event := &MessageReceiveEvent{
			MessageType: "text",
			Content:     `{"text":"Not a post"}`,
		}

		_, err := parser.ExtractPostText(event)
		if err == nil {
			t.Error("ExtractPostText() should return error for non-post message")
		}
	})

	t.Run("ExtractPostText with nil event", func(t *testing.T) {
		_, err := parser.ExtractPostText(nil)
		if err == nil {
			t.Error("ExtractPostText(nil) should return error")
		}
	})
}

func TestSenderEdgeCases(t *testing.T) {
	t.Run("send with empty chatID", func(t *testing.T) {
		mockClient := NewMockClient()
		sender := NewSender(mockClient)

		err := sender.SendText(context.Background(), "", "Hello")
		if err == nil {
			t.Error("SendText() should return error for empty chatID")
		}
	})

	t.Run("send with empty content", func(t *testing.T) {
		mockClient := NewMockClient()
		sender := NewSender(mockClient)

		err := sender.SendText(context.Background(), "oc_chat", "")
		if err == nil {
			t.Error("SendText() should return error for empty content")
		}
	})

	t.Run("send message with nil", func(t *testing.T) {
		mockClient := NewMockClient()
		sender := NewSender(mockClient)

		err := sender.SendMessage(context.Background(), nil)
		if err == nil {
			t.Error("SendMessage() should return error for nil message")
		}
	})

	t.Run("send message with empty channel", func(t *testing.T) {
		mockClient := NewMockClient()
		sender := NewSender(mockClient)

		msg := &core.Message{
			ID:        "msg_no_channel",
			Platform:  "feishu",
			UserID:    "ou_user",
			ChannelID: "",
			Content:   "Test",
			Type:      core.MessageTypeText,
			Timestamp: time.Now(),
		}

		err := sender.SendMessage(context.Background(), msg)
		if err == nil {
			t.Error("SendMessage() should return error for empty channel")
		}
	})

	t.Run("send propagates client error", func(t *testing.T) {
		mockClient := NewMockClient()
		mockClient.SendTextError = context.Canceled
		sender := NewSender(mockClient)

		err := sender.SendText(context.Background(), "oc_chat", "Hello")
		if err == nil {
			t.Error("SendText() should return error when client fails")
		}
	})
}

func TestMockClientMethods(t *testing.T) {
	t.Run("simulate message event", func(t *testing.T) {
		mockClient := NewMockClient()

		var receivedEvent *MessageReceiveEvent
		mockClient.OnEvent(func(ctx context.Context, event *MessageReceiveEvent) error {
			receivedEvent = event
			return nil
		})

		testEvent := &MessageReceiveEvent{
			EventID:     "evt_sim",
			MessageID:   "msg_sim",
			MessageType: "text",
			Content:     `{"text":"Simulated"}`,
			ChatID:      "oc_chat",
		}

		err := mockClient.SimulateMessageEvent(context.Background(), testEvent)
		if err != nil {
			t.Errorf("SimulateMessageEvent() error = %v", err)
		}

		if receivedEvent == nil {
			t.Error("Event was not received by handler")
		}
		if receivedEvent.EventID != "evt_sim" {
			t.Errorf("EventID = %v, want evt_sim", receivedEvent.EventID)
		}
	})

	t.Run("simulate without handler", func(t *testing.T) {
		mockClient := NewMockClient()

		// No handler registered
		testEvent := &MessageReceiveEvent{
			EventID: "evt_no_handler",
		}

		err := mockClient.SimulateMessageEvent(context.Background(), testEvent)
		if err != nil {
			t.Errorf("SimulateMessageEvent() without handler error = %v", err)
		}
	})

	t.Run("reset clears all state", func(t *testing.T) {
		mockClient := NewMockClient()

		// Set some state
		_ = mockClient.Connect(context.Background())
		_ = mockClient.SendText(context.Background(), "oc_chat", "Hello")
		mockClient.OnEvent(func(ctx context.Context, event *MessageReceiveEvent) error {
			return nil
		})

		// Reset
		mockClient.Reset()

		if mockClient.connected {
			t.Error("Reset() should set connected to false")
		}
		if mockClient.ConnectCalled != 0 {
			t.Errorf("ConnectCalled = %v, want 0", mockClient.ConnectCalled)
		}
		if mockClient.SendTextCalled != 0 {
			t.Errorf("SendTextCalled = %v, want 0", mockClient.SendTextCalled)
		}
		if mockClient.OnEventCalled != 0 {
			t.Errorf("OnEventCalled = %v, want 0", mockClient.OnEventCalled)
		}
	})
}
