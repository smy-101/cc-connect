package feishu

import (
	"testing"
	"time"

	"github.com/smy-101/cc-connect/internal/core"
)

func TestTextMessageToUnified(t *testing.T) {
	converter := NewMessageConverter()

	tests := []struct {
		name    string
		event   *MessageReceiveEvent
		want    *core.Message
		wantErr bool
	}{
		{
			name: "simple text message",
			event: &MessageReceiveEvent{
				EventID:     "event_123",
				MessageID:   "msg_456",
				MessageType: "text",
				Content:     `{"text":"Hello, World!"}`,
				ChatID:      "chat_789",
				ChatType:    "p2p",
				Sender: SenderInfo{
					OpenID:     "ou_abc123",
					UnionID:    "on_xyz789",
					UserID:     "user_001",
					SenderType: "user",
				},
				CreateTime: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			},
			want: &core.Message{
				Platform:  "feishu",
				UserID:    "ou_abc123",
				ChannelID: "chat_789",
				Content:   "Hello, World!",
				Type:      core.MessageTypeText,
			},
			wantErr: false,
		},
		{
			name: "text message with special characters",
			event: &MessageReceiveEvent{
				EventID:     "event_special",
				MessageID:   "msg_special",
				MessageType: "text",
				Content:     `{"text":"Hello \"World\"\nNew line\tTab"}`,
				ChatID:      "chat_special",
				ChatType:    "group",
				Sender: SenderInfo{
					OpenID:     "ou_special",
					SenderType: "user",
				},
				CreateTime: time.Now(),
			},
			want: &core.Message{
				Platform:  "feishu",
				UserID:    "ou_special",
				ChannelID: "chat_special",
				Content:   "Hello \"World\"\nNew line\tTab",
				Type:      core.MessageTypeText,
			},
			wantErr: false,
		},
		{
			name: "empty text message should error",
			event: &MessageReceiveEvent{
				EventID:     "event_empty",
				MessageID:   "msg_empty",
				MessageType: "text",
				Content:     `{"text":""}`,
				ChatID:      "chat_empty",
				ChatType:    "p2p",
				Sender: SenderInfo{
					OpenID:     "ou_empty",
					SenderType: "user",
				},
				CreateTime: time.Now(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid JSON content should error",
			event: &MessageReceiveEvent{
				EventID:     "event_invalid",
				MessageID:   "msg_invalid",
				MessageType: "text",
				Content:     `invalid json`,
				ChatID:      "chat_invalid",
				ChatType:    "p2p",
				Sender: SenderInfo{
					OpenID:     "ou_invalid",
					SenderType: "user",
				},
				CreateTime: time.Now(),
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := converter.ToUnifiedMessage(tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToUnifiedMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Platform != tt.want.Platform {
				t.Errorf("Platform = %v, want %v", got.Platform, tt.want.Platform)
			}
			if got.UserID != tt.want.UserID {
				t.Errorf("UserID = %v, want %v", got.UserID, tt.want.UserID)
			}
			if got.ChannelID != tt.want.ChannelID {
				t.Errorf("ChannelID = %v, want %v", got.ChannelID, tt.want.ChannelID)
			}
			if got.Content != tt.want.Content {
				t.Errorf("Content = %v, want %v", got.Content, tt.want.Content)
			}
			if got.Type != tt.want.Type {
				t.Errorf("Type = %v, want %v", got.Type, tt.want.Type)
			}
			if got.ID == "" {
				t.Error("ID should not be empty")
			}
			if got.Timestamp.IsZero() {
				t.Error("Timestamp should not be zero")
			}
		})
	}
}

func TestMentionExtraction(t *testing.T) {
	converter := NewMessageConverter()

	tests := []struct {
		name         string
		event        *MessageReceiveEvent
		wantMentions []MentionInfo
		wantErr      bool
	}{
		{
			name: "single mention",
			event: &MessageReceiveEvent{
				EventID:     "event_mention_1",
				MessageID:   "msg_mention_1",
				MessageType: "text",
				Content:     `{"text":"@_user_1 Hello!"}`,
				ChatID:      "chat_mention",
				ChatType:    "group",
				Sender: SenderInfo{
					OpenID:     "ou_sender",
					SenderType: "user",
				},
				Mentions: []MentionInfo{
					{
						Key:        "@_user_1",
						OpenID:     "ou_mentioned_1",
						Name:       "张三",
						TenantKey:  "tenant_123",
					},
				},
				CreateTime: time.Now(),
			},
			wantMentions: []MentionInfo{
				{
					Key:        "@_user_1",
					OpenID:     "ou_mentioned_1",
					Name:       "张三",
					TenantKey:  "tenant_123",
				},
			},
			wantErr: false,
		},
		{
			name: "multiple mentions",
			event: &MessageReceiveEvent{
				EventID:     "event_mention_multi",
				MessageID:   "msg_mention_multi",
				MessageType: "text",
				Content:     `{"text":"@_user_1 @_user_2 check this out"}`,
				ChatID:      "chat_multi",
				ChatType:    "group",
				Sender: SenderInfo{
					OpenID:     "ou_sender",
					SenderType: "user",
				},
				Mentions: []MentionInfo{
					{
						Key:        "@_user_1",
						OpenID:     "ou_mentioned_1",
						Name:       "张三",
						TenantKey:  "tenant_123",
					},
					{
						Key:        "@_user_2",
						OpenID:     "ou_mentioned_2",
						Name:       "李四",
						TenantKey:  "tenant_123",
					},
				},
				CreateTime: time.Now(),
			},
			wantMentions: []MentionInfo{
				{
					Key:        "@_user_1",
					OpenID:     "ou_mentioned_1",
					Name:       "张三",
					TenantKey:  "tenant_123",
				},
				{
					Key:        "@_user_2",
					OpenID:     "ou_mentioned_2",
					Name:       "李四",
					TenantKey:  "tenant_123",
				},
			},
			wantErr: false,
		},
		{
			name: "no mentions",
			event: &MessageReceiveEvent{
				EventID:     "event_no_mention",
				MessageID:   "msg_no_mention",
				MessageType: "text",
				Content:     `{"text":"Just a regular message"}`,
				ChatID:      "chat_no_mention",
				ChatType:    "p2p",
				Sender: SenderInfo{
					OpenID:     "ou_sender",
					SenderType: "user",
				},
				Mentions: nil,
				CreateTime: time.Now(),
			},
			wantMentions: nil,
			wantErr:      false,
		},
		{
			name: "empty mentions array",
			event: &MessageReceiveEvent{
				EventID:     "event_empty_mentions",
				MessageID:   "msg_empty_mentions",
				MessageType: "text",
				Content:     `{"text":"Another regular message"}`,
				ChatID:      "chat_empty_mentions",
				ChatType:    "p2p",
				Sender: SenderInfo{
					OpenID:     "ou_sender",
					SenderType: "user",
				},
				Mentions: []MentionInfo{},
				CreateTime: time.Now(),
			},
			wantMentions: nil,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := converter.ToUnifiedMessage(tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToUnifiedMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Test mentions extraction via GetMentions
			mentions := converter.GetMentions(tt.event)
			if len(mentions) != len(tt.wantMentions) {
				t.Errorf("GetMentions() got %d mentions, want %d", len(mentions), len(tt.wantMentions))
				return
			}

			for i, m := range mentions {
				if m.Key != tt.wantMentions[i].Key {
					t.Errorf("Mention[%d].Key = %v, want %v", i, m.Key, tt.wantMentions[i].Key)
				}
				if m.OpenID != tt.wantMentions[i].OpenID {
					t.Errorf("Mention[%d].OpenID = %v, want %v", i, m.OpenID, tt.wantMentions[i].OpenID)
				}
				if m.Name != tt.wantMentions[i].Name {
					t.Errorf("Mention[%d].Name = %v, want %v", i, m.Name, tt.wantMentions[i].Name)
				}
			}

			// Verify the message itself is valid
			if got == nil {
				t.Error("ToUnifiedMessage() returned nil message")
			}
		})
	}
}

func TestUnifiedToFeishu(t *testing.T) {
	converter := NewMessageConverter()

	tests := []struct {
		name       string
		message    *core.Message
		wantJSON   string
		wantErr    bool
	}{
		{
			name: "simple text message",
			message: &core.Message{
				ID:        "msg_001",
				Platform:  "feishu",
				UserID:    "ou_abc",
				ChannelID: "chat_123",
				Content:   "Hello, Feishu!",
				Type:      core.MessageTypeText,
				Timestamp: time.Now(),
			},
			wantJSON: `{"text":"Hello, Feishu!"}`,
			wantErr:  false,
		},
		{
			name: "message with special characters",
			message: &core.Message{
				ID:        "msg_002",
				Platform:  "feishu",
				UserID:    "ou_xyz",
				ChannelID: "chat_456",
				Content:   "Line1\nLine2\tTabbed \"quoted\"",
				Type:      core.MessageTypeText,
				Timestamp: time.Now(),
			},
			wantJSON: `{"text":"Line1\nLine2\tTabbed \"quoted\""}`,
			wantErr:  false,
		},
		{
			name: "message with unicode",
			message: &core.Message{
				ID:        "msg_003",
				Platform:  "feishu",
				UserID:    "ou_unicode",
				ChannelID: "chat_789",
				Content:   "你好，世界！🌍",
				Type:      core.MessageTypeText,
				Timestamp: time.Now(),
			},
			wantJSON: `{"text":"你好，世界！🌍"}`,
			wantErr:  false,
		},
		{
			name: "unsupported message type",
			message: &core.Message{
				ID:        "msg_004",
				Platform:  "feishu",
				UserID:    "ou_unsupported",
				ChannelID: "chat_unsupported",
				Content:   "image_key_123",
				Type:      core.MessageTypeImage,
				Timestamp: time.Now(),
			},
			wantJSON: "",
			wantErr:  true,
		},
		{
			name: "nil message should error",
			message: nil,
			wantJSON: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := converter.ToFeishuContent(tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToFeishuContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got != tt.wantJSON {
				t.Errorf("ToFeishuContent() = %v, want %v", got, tt.wantJSON)
			}
		})
	}
}
