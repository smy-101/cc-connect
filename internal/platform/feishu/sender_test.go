package feishu

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/smy-101/cc-connect/internal/core"
)

func TestSendTextMessage(t *testing.T) {
	// Create a mock client for testing
	mockClient := NewMockClient()

	// Create sender with mock client
	sender := NewSender(mockClient)

	tests := []struct {
		name      string
		chatID    string
		content   string
		setupMock func()
		wantErr   bool
	}{
		{
			name:    "send simple text message",
			chatID:  "oc_chat_123",
			content: "Hello, World!",
			setupMock: func() {
				mockClient.Reset()
			},
			wantErr: false,
		},
		{
			name:    "send message with unicode",
			chatID:  "oc_chat_unicode",
			content: "你好，世界！🌍",
			setupMock: func() {
				mockClient.Reset()
			},
			wantErr: false,
		},
		{
			name:    "send message with special characters",
			chatID:  "oc_chat_special",
			content: "Line1\nLine2\tTab \"quoted\"",
			setupMock: func() {
				mockClient.Reset()
			},
			wantErr: false,
		},
		{
			name:    "send empty message",
			chatID:  "oc_chat_empty",
			content: "",
			setupMock: func() {
				mockClient.Reset()
			},
			wantErr: true, // Empty message should error
		},
		{
			name:    "send with network error",
			chatID:  "oc_chat_error",
			content: "This should fail",
			setupMock: func() {
				mockClient.Reset()
				mockClient.SendTextError = errors.New("network error")
			},
			wantErr: true,
		},
		{
			name:    "send to invalid chat",
			chatID:  "",
			content: "Hello",
			setupMock: func() {
				mockClient.Reset()
			},
			wantErr: true, // Empty chat ID should error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := sender.SendText(context.Background(), tt.chatID, tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("SendText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if mockClient.LastSendTextChatID != tt.chatID {
					t.Errorf("chatID = %v, want %v", mockClient.LastSendTextChatID, tt.chatID)
				}
				if mockClient.LastSendTextContent != tt.content {
					t.Errorf("content = %v, want %v", mockClient.LastSendTextContent, tt.content)
				}
				if mockClient.SendTextCalled != 1 {
					t.Errorf("SendTextCalled = %v, want 1", mockClient.SendTextCalled)
				}
			}
		})
	}
}

func TestSendTextWithConverter(t *testing.T) {
	mockClient := NewMockClient()
	converter := NewMessageConverter()
	sender := NewSenderWithConverter(mockClient, converter)

	t.Run("send using unified message", func(t *testing.T) {
		mockClient.Reset()

		// Create a unified message
		msg := &core.Message{
			ID:        "msg_001",
			Platform:  "feishu",
			UserID:    "ou_test",
			ChannelID: "oc_test_chat",
			Content:   "Hello from unified message",
			Type:      core.MessageTypeText,
			Timestamp: time.Now(),
		}

		err := sender.SendMessage(context.Background(), msg)
		if err != nil {
			t.Errorf("SendMessage() error = %v", err)
		}

		// Verify the content was converted correctly
		expectedContent := `{"text":"Hello from unified message"}`
		if mockClient.LastSendTextContent != expectedContent {
			t.Errorf("content = %v, want %v", mockClient.LastSendTextContent, expectedContent)
		}
	})

	t.Run("send message without channel ID", func(t *testing.T) {
		mockClient.Reset()

		msg := &core.Message{
			ID:        "msg_002",
			Platform:  "feishu",
			UserID:    "ou_test",
			ChannelID: "", // Empty channel ID
			Content:   "No channel",
			Type:      core.MessageTypeText,
			Timestamp: time.Now(),
		}

		err := sender.SendMessage(context.Background(), msg)
		if err == nil {
			t.Error("SendMessage() should fail with empty channel ID")
		}
	})

	t.Run("send nil message", func(t *testing.T) {
		mockClient.Reset()

		err := sender.SendMessage(context.Background(), nil)
		if err == nil {
			t.Error("SendMessage() should fail with nil message")
		}
	})
}

func TestSendUnifiedMessage(t *testing.T) {
	mockClient := NewMockClient()
	sender := NewSender(mockClient)

	msg := &core.Message{
		ID:        "msg_unified_001",
		Platform:  "feishu",
		UserID:    "ou_test",
		ChannelID: "oc_unified_chat",
		Content:   "Hello from SendUnifiedMessage",
		Type:      core.MessageTypeText,
		Timestamp: time.Now(),
	}

	if err := sender.SendUnifiedMessage(context.Background(), msg); err != nil {
		t.Fatalf("SendUnifiedMessage() error = %v", err)
	}
	if mockClient.LastSendTextChatID != "oc_unified_chat" {
		t.Fatalf("LastSendTextChatID = %q", mockClient.LastSendTextChatID)
	}
}

func TestSendErrors(t *testing.T) {
	mockClient := NewMockClient()
	sender := NewSender(mockClient)

	tests := []struct {
		name      string
		chatID    string
		content   string
		setupMock func()
		wantErr   bool
		errMsg    string
	}{
		{
			name:    "invalid chat ID error",
			chatID:  "",
			content: "test",
			setupMock: func() {
				mockClient.Reset()
			},
			wantErr: true,
			errMsg:  "chatID cannot be empty",
		},
		{
			name:    "empty content error",
			chatID:  "oc_chat",
			content: "",
			setupMock: func() {
				mockClient.Reset()
			},
			wantErr: true,
			errMsg:  "content cannot be empty",
		},
		{
			name:    "client error propagation",
			chatID:  "oc_chat",
			content: "test",
			setupMock: func() {
				mockClient.Reset()
				mockClient.SendTextError = fmt.Errorf("permission denied")
			},
			wantErr: true,
			errMsg:  "permission denied",
		},
		{
			name:    "unsupported message type",
			chatID:  "oc_chat",
			content: "test",
			setupMock: func() {
				mockClient.Reset()
			},
			wantErr: false,
			errMsg:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := sender.SendText(context.Background(), tt.chatID, tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("SendText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if err.Error() != tt.errMsg {
					t.Errorf("error message = %q, want %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}
