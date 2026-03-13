package feishu

import (
	"encoding/json"
	"testing"
	"time"
)

// Sample im.message.receive_v1 event JSON structure based on Feishu documentation
const sampleEventJSON = `{
	"schema": "2.0",
	"header": {
		"event_id": "event_123456",
		"event_type": "im.message.receive_v1",
		"create_time": "1704067200000",
		"token": "token_abc",
		"app_id": "app_123",
		"tenant_key": "tenant_456"
	},
	"event": {
		"sender": {
			"sender_id": {
				"open_id": "ou_sender_123",
				"union_id": "on_sender_456",
				"user_id": "user_789"
			},
			"sender_type": "user",
			"tenant_key": "tenant_456"
		},
		"message": {
			"message_id": "msg_abcdefg",
			"root_id": "",
			"parent_id": "",
			"create_time": "1704067200000",
			"chat_id": "oc_chat_123",
			"chat_type": "group",
			"message_type": "text",
			"content": "{\"text\":\"Hello World\"}",
			"mentions": [
				{
					"key": "@_user_1",
					"id": {
						"open_id": "ou_mentioned_123",
						"union_id": "on_mentioned_456",
						"user_id": "user_mentioned_789"
					},
					"name": "张三",
					"tenant_key": "tenant_456"
				}
			]
		}
	}
}`

func TestEventParsing(t *testing.T) {
	parser := NewEventParser()

	tests := []struct {
		name       string
		input      string
		wantErr    bool
		checkFunc  func(*testing.T, *MessageReceiveEvent)
	}{
		{
			name:    "parse complete event structure",
			input:   sampleEventJSON,
			wantErr: false,
			checkFunc: func(t *testing.T, event *MessageReceiveEvent) {
				// Check basic event info
				if event.EventID != "event_123456" {
					t.Errorf("EventID = %v, want event_123456", event.EventID)
				}
				if event.MessageID != "msg_abcdefg" {
					t.Errorf("MessageID = %v, want msg_abcdefg", event.MessageID)
				}
				if event.ChatID != "oc_chat_123" {
					t.Errorf("ChatID = %v, want oc_chat_123", event.ChatID)
				}
				if event.ChatType != "group" {
					t.Errorf("ChatType = %v, want group", event.ChatType)
				}
				if event.MessageType != "text" {
					t.Errorf("MessageType = %v, want text", event.MessageType)
				}
				if event.Content != `{"text":"Hello World"}` {
					t.Errorf("Content = %v, want {\"text\":\"Hello World\"}", event.Content)
				}

				// Check create time
				expectedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
				if !event.CreateTime.Equal(expectedTime) {
					t.Errorf("CreateTime = %v, want %v", event.CreateTime, expectedTime)
				}
			},
		},
		{
			name:    "parse p2p message event",
			input:   `{"schema":"2.0","header":{"event_id":"evt_p2p","event_type":"im.message.receive_v1","create_time":"1704067200000","app_id":"app_123","tenant_key":"tenant_456"},"event":{"sender":{"sender_id":{"open_id":"ou_p2p"},"sender_type":"user"},"message":{"message_id":"msg_p2p","chat_id":"oc_p2p_123","chat_type":"p2p","message_type":"text","content":"{\"text\":\"Private message\"}"}}}`,
			wantErr: false,
			checkFunc: func(t *testing.T, event *MessageReceiveEvent) {
				if event.ChatType != "p2p" {
					t.Errorf("ChatType = %v, want p2p", event.ChatType)
				}
				if event.Content != `{"text":"Private message"}` {
					t.Errorf("Content = %v, want {\"text\":\"Private message\"}", event.Content)
				}
			},
		},
		{
			name:    "invalid JSON should error",
			input:   `{invalid json}`,
			wantErr: true,
		},
		{
			name:    "empty JSON should error",
			input:   ``,
			wantErr: true,
		},
		{
			name:    "missing event should error",
			input:   `{"schema":"2.0","header":{"event_id":"evt_123"}}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.Parse([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, got)
			}
		})
	}
}

func TestEventParsingFromSDK(t *testing.T) {
	parser := NewEventParser()

	// Test parsing from SDK event structure (map format)
	sdkEvent := map[string]any{
		"schema": "2.0",
		"header": map[string]any{
			"event_id":   "evt_sdk",
			"event_type": "im.message.receive_v1",
			"app_id":     "app_sdk",
			"tenant_key": "tenant_sdk",
		},
		"event": map[string]any{
			"sender": map[string]any{
				"sender_id": map[string]any{
					"open_id": "ou_sdk_user",
				},
				"sender_type": "user",
			},
			"message": map[string]any{
				"message_id":   "msg_sdk",
				"chat_id":      "oc_sdk_chat",
				"chat_type":    "p2p",
				"message_type": "text",
				"content":      `{"text":"SDK message"}`,
			},
		},
	}

	got, err := parser.ParseFromMap(sdkEvent)
	if err != nil {
		t.Errorf("ParseFromMap() error = %v", err)
		return
	}

	if got.EventID != "evt_sdk" {
		t.Errorf("EventID = %v, want evt_sdk", got.EventID)
	}
	if got.MessageID != "msg_sdk" {
		t.Errorf("MessageID = %v, want msg_sdk", got.MessageID)
	}
	if got.Sender.OpenID != "ou_sdk_user" {
		t.Errorf("Sender.OpenID = %v, want ou_sdk_user", got.Sender.OpenID)
	}
}

func TestSenderExtraction(t *testing.T) {
	parser := NewEventParser()

	tests := []struct {
		name        string
		input       string
		wantSender  SenderInfo
		wantErr     bool
	}{
		{
			name: "complete sender info",
			input: `{"schema":"2.0","header":{"event_id":"evt_sender_full"},"event":{"sender":{"sender_id":{"open_id":"ou_full","union_id":"on_full","user_id":"user_full"},"sender_type":"user"},"message":{"message_id":"msg_full","chat_id":"oc_chat","chat_type":"p2p","message_type":"text","content":"{}"}}}`,
			wantSender: SenderInfo{
				OpenID:     "ou_full",
				UnionID:    "on_full",
				UserID:     "user_full",
				SenderType: "user",
			},
			wantErr: false,
		},
		{
			name: "only open_id (no user_id permission)",
			input: `{"schema":"2.0","header":{"event_id":"evt_sender_partial"},"event":{"sender":{"sender_id":{"open_id":"ou_partial"},"sender_type":"user"},"message":{"message_id":"msg_partial","chat_id":"oc_chat","chat_type":"p2p","message_type":"text","content":"{}"}}}`,
			wantSender: SenderInfo{
				OpenID:     "ou_partial",
				UnionID:    "",
				UserID:     "",
				SenderType: "user",
			},
			wantErr: false,
		},
		{
			name: "app sender type",
			input: `{"schema":"2.0","header":{"event_id":"evt_app_sender"},"event":{"sender":{"sender_id":{"open_id":"cli_app_id"},"sender_type":"app"},"message":{"message_id":"msg_app","chat_id":"oc_chat","chat_type":"p2p","message_type":"text","content":"{}"}}}`,
			wantSender: SenderInfo{
				OpenID:     "cli_app_id",
				UnionID:    "",
				UserID:     "",
				SenderType: "app",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.Parse([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if got.Sender.OpenID != tt.wantSender.OpenID {
				t.Errorf("Sender.OpenID = %v, want %v", got.Sender.OpenID, tt.wantSender.OpenID)
			}
			if got.Sender.UnionID != tt.wantSender.UnionID {
				t.Errorf("Sender.UnionID = %v, want %v", got.Sender.UnionID, tt.wantSender.UnionID)
			}
			if got.Sender.UserID != tt.wantSender.UserID {
				t.Errorf("Sender.UserID = %v, want %v", got.Sender.UserID, tt.wantSender.UserID)
			}
			if got.Sender.SenderType != tt.wantSender.SenderType {
				t.Errorf("Sender.SenderType = %v, want %v", got.Sender.SenderType, tt.wantSender.SenderType)
			}
		})
	}
}

// Helper to verify JSON structure matches expected format
func TestEventJSONStructure(t *testing.T) {
	// Verify the sample JSON is valid
	var raw map[string]any
	if err := json.Unmarshal([]byte(sampleEventJSON), &raw); err != nil {
		t.Fatalf("sampleEventJSON is not valid JSON: %v", err)
	}

	// Verify structure has expected fields
	header, ok := raw["header"].(map[string]any)
	if !ok {
		t.Fatal("header not found or wrong type")
	}
	if header["event_id"] != "event_123456" {
		t.Errorf("header.event_id = %v, want event_123456", header["event_id"])
	}

	event, ok := raw["event"].(map[string]any)
	if !ok {
		t.Fatal("event not found or wrong type")
	}

	message, ok := event["message"].(map[string]any)
	if !ok {
		t.Fatal("event.message not found or wrong type")
	}
	if message["chat_type"] != "group" {
		t.Errorf("message.chat_type = %v, want group", message["chat_type"])
	}
}

func TestPostMessage(t *testing.T) {
	parser := NewEventParser()

	// Sample post message content
	postContent := map[string]any{
		"zh_cn": map[string]any{
			"title": "会议通知",
			"content": []any{
				[]any{
					map[string]any{"tag": "text", "text": "明天下午2点开会"},
					map[string]any{"tag": "at", "user_id": "ou_attendee_1", "user_name": "张三"},
				},
				[]any{
					map[string]any{"tag": "text", "text": "请准时参加"},
				},
			},
		},
	}

	postContentBytes, _ := json.Marshal(postContent)

	postEvent := map[string]any{
		"schema": "2.0",
		"header": map[string]any{
			"event_id":   "evt_post_001",
			"event_type": "im.message.receive_v1",
			"create_time": "1704067200000",
		},
		"event": map[string]any{
			"sender": map[string]any{
				"sender_id":   map[string]any{"open_id": "ou_poster"},
				"sender_type": "user",
			},
			"message": map[string]any{
				"message_id":   "msg_post_001",
				"chat_id":      "oc_post_chat",
				"chat_type":    "group",
				"message_type": "post",
				"content":      string(postContentBytes),
			},
		},
	}

	postEventJSON, _ := json.Marshal(postEvent)

	tests := []struct {
		name           string
		input          string
		wantPlainText  string
		wantErr        bool
	}{
		{
			name:          "parse post message with title and content",
			input:         string(postEventJSON),
			wantPlainText: "会议通知\n明天下午2点开会@张三\n请准时参加",
			wantErr:       false,
		},
		{
			name: "simple post message",
			input: `{"schema":"2.0","header":{"event_id":"evt_post_simple"},"event":{"sender":{"sender_id":{"open_id":"ou_simple"},"sender_type":"user"},"message":{"message_id":"msg_simple","chat_id":"oc_simple","chat_type":"p2p","message_type":"post","content":"{\"zh_cn\":{\"content\":[[{\"tag\":\"text\",\"text\":\"Hello World\"}]]}}"}}}`,
			wantPlainText: "Hello World",
			wantErr:       false,
		},
		{
			name: "post with multiple at mentions",
			input: `{"schema":"2.0","header":{"event_id":"evt_post_multi_at"},"event":{"sender":{"sender_id":{"open_id":"ou_multi"},"sender_type":"user"},"message":{"message_id":"msg_multi_at","chat_id":"oc_multi","chat_type":"group","message_type":"post","content":"{\"zh_cn\":{\"content\":[[{\"tag\":\"at\",\"user_id\":\"ou_1\",\"user_name\":\"Alice\"},{\"tag\":\"at\",\"user_id\":\"ou_2\",\"user_name\":\"Bob\"},{\"tag\":\"text\",\"text\":\" check this\"}]]}}"}}}`,
			wantPlainText: "@Alice@Bob check this",
			wantErr:       false,
		},
		{
			name: "post with en_us locale fallback",
			input: `{"schema":"2.0","header":{"event_id":"evt_post_en"},"event":{"sender":{"sender_id":{"open_id":"ou_en"},"sender_type":"user"},"message":{"message_id":"msg_en","chat_id":"oc_en","chat_type":"p2p","message_type":"post","content":"{\"en_us\":{\"title\":\"English Title\",\"content\":[[{\"tag\":\"text\",\"text\":\"English content\"}]]}}"}}}`,
			wantPlainText: "English Title\nEnglish content",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.Parse([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if got.MessageType != "post" {
				t.Errorf("MessageType = %v, want post", got.MessageType)
			}

			plainText, err := parser.ExtractPostText(got)
			if err != nil {
				t.Errorf("ExtractPostText() error = %v", err)
				return
			}

			if plainText != tt.wantPlainText {
				t.Errorf("ExtractPostText() = %q, want %q", plainText, tt.wantPlainText)
			}
		})
	}
}
