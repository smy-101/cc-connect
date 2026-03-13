package core

import (
	"encoding/json"
	"regexp"
	"testing"
	"time"
)

// TestMessageTypeConstants 测试 MessageType 常量定义
func TestMessageTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		msgType  MessageType
		expected string
	}{
		{"text type", MessageTypeText, "text"},
		{"voice type", MessageTypeVoice, "voice"},
		{"image type", MessageTypeImage, "image"},
		{"command type", MessageTypeCommand, "command"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.msgType) != tt.expected {
				t.Errorf("MessageType %s = %q, want %q", tt.name, tt.msgType, tt.expected)
			}
		})
	}
}

// TestMessageStructFields 测试 Message 结构体字段存在及 JSON tag 正确
func TestMessageStructFields(t *testing.T) {
	// 创建一个测试消息
	msg := Message{
		ID:        "test-id-123",
		Platform:  "feishu",
		UserID:    "user-456",
		Content:   "Hello, World!",
		Type:      MessageTypeText,
		Timestamp: time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC),
	}

	// 序列化为 JSON 验证字段名
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	// 解析为 map 检查字段名
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// 验证 snake_case 字段名
	fieldTests := []struct {
		field    string
		expected interface{}
	}{
		{"id", "test-id-123"},
		{"platform", "feishu"},
		{"user_id", "user-456"},
		{"content", "Hello, World!"},
		{"type", "text"},
	}

	for _, ft := range fieldTests {
		t.Run("field_"+ft.field, func(t *testing.T) {
			val, exists := result[ft.field]
			if !exists {
				t.Errorf("Field %q not found in JSON", ft.field)
				return
			}
			if val != ft.expected {
				t.Errorf("Field %q = %v, want %v", ft.field, val, ft.expected)
			}
		})
	}

	// 验证 timestamp 字段存在
	if _, exists := result["timestamp"]; !exists {
		t.Error("Field 'timestamp' not found in JSON")
	}
}

// TestNewMessage 测试通用构造函数
func TestNewMessage(t *testing.T) {
	msg := NewMessage("feishu", "user-123", "Hello", MessageTypeText)

	if msg == nil {
		t.Fatal("NewMessage returned nil")
	}

	// 验证字段设置
	if msg.Platform != "feishu" {
		t.Errorf("Platform = %q, want %q", msg.Platform, "feishu")
	}
	if msg.UserID != "user-123" {
		t.Errorf("UserID = %q, want %q", msg.UserID, "user-123")
	}
	if msg.Content != "Hello" {
		t.Errorf("Content = %q, want %q", msg.Content, "Hello")
	}
	if msg.Type != MessageTypeText {
		t.Errorf("Type = %q, want %q", msg.Type, MessageTypeText)
	}

	// 验证 ID 自动生成
	if msg.ID == "" {
		t.Error("ID should be auto-generated, got empty string")
	}

	// 验证 Timestamp 自动生成
	if msg.Timestamp.IsZero() {
		t.Error("Timestamp should be auto-generated, got zero value")
	}

	// 验证 Timestamp 接近当前时间
	now := time.Now()
	diff := now.Sub(msg.Timestamp)
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Second {
		t.Errorf("Timestamp difference from now = %v, want < 1s", diff)
	}
}

// TestNewTextMessage 测试文本消息便捷构造函数
func TestNewTextMessage(t *testing.T) {
	msg := NewTextMessage("feishu", "user-123", "Hello")
	if msg.Type != MessageTypeText {
		t.Errorf("Type = %q, want %q", msg.Type, MessageTypeText)
	}
}

// TestNewVoiceMessage 测试语音消息便捷构造函数
func TestNewVoiceMessage(t *testing.T) {
	msg := NewVoiceMessage("feishu", "user-123", "voice-data")
	if msg.Type != MessageTypeVoice {
		t.Errorf("Type = %q, want %q", msg.Type, MessageTypeVoice)
	}
}

// TestNewImageMessage 测试图片消息便捷构造函数
func TestNewImageMessage(t *testing.T) {
	msg := NewImageMessage("feishu", "user-123", "image-url")
	if msg.Type != MessageTypeImage {
		t.Errorf("Type = %q, want %q", msg.Type, MessageTypeImage)
	}
}

// TestNewCommandMessage 测试命令消息便捷构造函数
func TestNewCommandMessage(t *testing.T) {
	msg := NewCommandMessage("feishu", "user-123", "/help")
	if msg.Type != MessageTypeCommand {
		t.Errorf("Type = %q, want %q", msg.Type, MessageTypeCommand)
	}
}

// TestMessageIDUnique 测试 ID 唯一性
func TestMessageIDUnique(t *testing.T) {
	ids := make(map[string]bool)
	count := 1000

	for i := 0; i < count; i++ {
		msg := NewTextMessage("feishu", "user", "test")
		if ids[msg.ID] {
			t.Errorf("Duplicate ID found: %s", msg.ID)
		}
		ids[msg.ID] = true
	}

	if len(ids) != count {
		t.Errorf("Expected %d unique IDs, got %d", count, len(ids))
	}
}

// TestMessageIDFormat 测试 ID 格式
func TestMessageIDFormat(t *testing.T) {
	// ID 格式: <unix_nano>_<random_8chars>
	// 示例: 1709234567890123456_a1b2c3d4
	idPattern := regexp.MustCompile(`^\d+_[0-9a-f]{8}$`)

	for i := 0; i < 100; i++ {
		msg := NewTextMessage("feishu", "user", "test")
		if !idPattern.MatchString(msg.ID) {
			t.Errorf("ID format invalid: %q (expected <unix_nano>_<8_hex_chars>)", msg.ID)
		}
	}
}

// TestMessageToJSON 测试 JSON 序列化
func TestMessageToJSON(t *testing.T) {
	msg := &Message{
		ID:        "test-id-123",
		Platform:  "feishu",
		UserID:    "user-456",
		Content:   "Hello, World!",
		Type:      MessageTypeText,
		Timestamp: time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC),
	}

	data, err := msg.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// 验证返回的是有效 JSON
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}

	// 验证所有字段存在
	expectedFields := []string{"id", "platform", "user_id", "content", "type", "timestamp"}
	for _, field := range expectedFields {
		if _, exists := result[field]; !exists {
			t.Errorf("Missing field %q in JSON", field)
		}
	}

	// 验证 snake_case 命名
	if _, exists := result["user_id"]; !exists {
		t.Error("Expected snake_case field 'user_id' not found")
	}
	if _, exists := result["userId"]; exists {
		t.Error("Unexpected camelCase field 'userId' found, expected snake_case")
	}
}

// TestMessageFromJSON 测试 JSON 反序列化
func TestMessageFromJSON(t *testing.T) {
	// 测试有效 JSON
	t.Run("valid JSON", func(t *testing.T) {
		jsonData := `{"id":"test-id","platform":"feishu","user_id":"user-123","content":"Hello","type":"text","timestamp":"2024-03-01T12:00:00Z"}`
		msg, err := FromJSON([]byte(jsonData))
		if err != nil {
			t.Fatalf("FromJSON failed: %v", err)
		}
		if msg.ID != "test-id" {
			t.Errorf("ID = %q, want %q", msg.ID, "test-id")
		}
		if msg.Platform != "feishu" {
			t.Errorf("Platform = %q, want %q", msg.Platform, "feishu")
		}
		if msg.UserID != "user-123" {
			t.Errorf("UserID = %q, want %q", msg.UserID, "user-123")
		}
		if msg.Content != "Hello" {
			t.Errorf("Content = %q, want %q", msg.Content, "Hello")
		}
		if msg.Type != MessageTypeText {
			t.Errorf("Type = %q, want %q", msg.Type, MessageTypeText)
		}
	})

	// 测试忽略未知字段
	t.Run("ignore unknown fields", func(t *testing.T) {
		jsonData := `{"id":"test-id","platform":"feishu","user_id":"user-123","content":"Hello","type":"text","timestamp":"2024-03-01T12:00:00Z","extra_field":"ignored"}`
		msg, err := FromJSON([]byte(jsonData))
		if err != nil {
			t.Fatalf("FromJSON with unknown field failed: %v", err)
		}
		if msg.ID != "test-id" {
			t.Errorf("ID = %q, want %q", msg.ID, "test-id")
		}
	})

	// 测试无效 JSON
	t.Run("invalid JSON", func(t *testing.T) {
		jsonData := `{invalid json}`
		_, err := FromJSON([]byte(jsonData))
		if err == nil {
			t.Error("Expected error for invalid JSON, got nil")
		}
	})

	// 测试缺少必需字段
	t.Run("missing required fields", func(t *testing.T) {
		tests := []struct {
			name     string
			jsonData string
		}{
			{"missing id", `{"platform":"feishu","user_id":"user-123","content":"Hello","type":"text","timestamp":"2024-03-01T12:00:00Z"}`},
			{"missing platform", `{"id":"test-id","user_id":"user-123","content":"Hello","type":"text","timestamp":"2024-03-01T12:00:00Z"}`},
			{"missing user_id", `{"id":"test-id","platform":"feishu","content":"Hello","type":"text","timestamp":"2024-03-01T12:00:00Z"}`},
			{"missing type", `{"id":"test-id","platform":"feishu","user_id":"user-123","content":"Hello","timestamp":"2024-03-01T12:00:00Z"}`},
			{"missing timestamp", `{"id":"test-id","platform":"feishu","user_id":"user-123","content":"Hello","type":"text"}`},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := FromJSON([]byte(tt.jsonData))
				if err == nil {
					t.Errorf("Expected error for %s, got nil", tt.name)
				}
			})
		}
	})
}

// TestMessageRoundTrip 测试往返一致性
func TestMessageRoundTrip(t *testing.T) {
	original := NewTextMessage("feishu", "user-123", "Hello, 世界! 🌍")

	data, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	recovered, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	// 验证所有字段一致
	if recovered.ID != original.ID {
		t.Errorf("ID mismatch: got %q, want %q", recovered.ID, original.ID)
	}
	if recovered.Platform != original.Platform {
		t.Errorf("Platform mismatch: got %q, want %q", recovered.Platform, original.Platform)
	}
	if recovered.UserID != original.UserID {
		t.Errorf("UserID mismatch: got %q, want %q", recovered.UserID, original.UserID)
	}
	if recovered.Content != original.Content {
		t.Errorf("Content mismatch: got %q, want %q", recovered.Content, original.Content)
	}
	if recovered.Type != original.Type {
		t.Errorf("Type mismatch: got %q, want %q", recovered.Type, original.Type)
	}
	if !recovered.Timestamp.Equal(original.Timestamp) {
		t.Errorf("Timestamp mismatch: got %v, want %v", recovered.Timestamp, original.Timestamp)
	}
}

// TestMessageEdge 测试边界情况
func TestMessageEdge(t *testing.T) {
	t.Run("empty content", func(t *testing.T) {
		msg := NewTextMessage("feishu", "user", "")
		data, err := msg.ToJSON()
		if err != nil {
			t.Fatalf("ToJSON with empty content failed: %v", err)
		}
		recovered, err := FromJSON(data)
		if err != nil {
			t.Fatalf("FromJSON with empty content failed: %v", err)
		}
		if recovered.Content != "" {
			t.Errorf("Content = %q, want empty", recovered.Content)
		}
	})

	t.Run("unicode characters", func(t *testing.T) {
		msg := NewTextMessage("feishu", "user", "你好世界 🎉 Hello мир")
		data, err := msg.ToJSON()
		if err != nil {
			t.Fatalf("ToJSON with unicode failed: %v", err)
		}
		recovered, err := FromJSON(data)
		if err != nil {
			t.Fatalf("FromJSON with unicode failed: %v", err)
		}
		if recovered.Content != msg.Content {
			t.Errorf("Content = %q, want %q", recovered.Content, msg.Content)
		}
	})

	t.Run("special characters", func(t *testing.T) {
		content := "line1\nline2\ttab\"quote\\backslash"
		msg := NewTextMessage("feishu", "user", content)
		data, err := msg.ToJSON()
		if err != nil {
			t.Fatalf("ToJSON with special chars failed: %v", err)
		}
		recovered, err := FromJSON(data)
		if err != nil {
			t.Fatalf("FromJSON with special chars failed: %v", err)
		}
		if recovered.Content != content {
			t.Errorf("Content = %q, want %q", recovered.Content, content)
		}
	})
}

// TestMessageChannelID 测试 Message ChannelID 字段
func TestMessageChannelID(t *testing.T) {
	t.Run("ChannelID 字段存在", func(t *testing.T) {
		msg := &Message{
			ID:        "test-id",
			Platform:  "feishu",
			UserID:    "ou_xxx",
			ChannelID: "oc_yyy",
			Content:   "test",
			Type:      MessageTypeText,
			Timestamp: time.Now(),
		}
		if msg.ChannelID != "oc_yyy" {
			t.Errorf("ChannelID = %q, want %q", msg.ChannelID, "oc_yyy")
		}
	})

	t.Run("ChannelID JSON 序列化", func(t *testing.T) {
		msg := &Message{
			ID:        "test-id",
			Platform:  "feishu",
			UserID:    "ou_xxx",
			ChannelID: "oc_yyy",
			Content:   "test",
			Type:      MessageTypeText,
			Timestamp: time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC),
		}

		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}

		if result["channel_id"] != "oc_yyy" {
			t.Errorf("channel_id = %v, want %q", result["channel_id"], "oc_yyy")
		}
	})

	t.Run("ChannelID JSON 反序列化", func(t *testing.T) {
		jsonData := `{"id":"test-id","platform":"feishu","user_id":"ou_xxx","channel_id":"oc_yyy","content":"test","type":"text","timestamp":"2024-03-01T12:00:00Z"}`
		msg, err := FromJSON([]byte(jsonData))
		if err != nil {
			t.Fatalf("FromJSON failed: %v", err)
		}
		if msg.ChannelID != "oc_yyy" {
			t.Errorf("ChannelID = %q, want %q", msg.ChannelID, "oc_yyy")
		}
	})

	t.Run("ChannelID 可选（私聊消息）", func(t *testing.T) {
		// 私聊消息没有 ChannelID
		msg := &Message{
			ID:        "test-id",
			Platform:  "feishu",
			UserID:    "ou_xxx",
			Content:   "test",
			Type:      MessageTypeText,
			Timestamp: time.Now(),
		}

		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}

		// ChannelID 为空时不应出现在 JSON 中 (omitempty)
		if _, exists := result["channel_id"]; exists {
			t.Error("channel_id should not appear in JSON when empty (omitempty)")
		}
	})

	t.Run("ChannelID 不影响反序列化", func(t *testing.T) {
		// 不包含 channel_id 的 JSON 应该正常反序列化
		jsonData := `{"id":"test-id","platform":"feishu","user_id":"ou_xxx","content":"test","type":"text","timestamp":"2024-03-01T12:00:00Z"}`
		msg, err := FromJSON([]byte(jsonData))
		if err != nil {
			t.Fatalf("FromJSON failed: %v", err)
		}
		if msg.ChannelID != "" {
			t.Errorf("ChannelID = %q, want empty", msg.ChannelID)
		}
	})

	t.Run("往返一致性", func(t *testing.T) {
		original := &Message{
			ID:        "test-id",
			Platform:  "feishu",
			UserID:    "ou_xxx",
			ChannelID: "oc_yyy",
			Content:   "test",
			Type:      MessageTypeText,
			Timestamp: time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC),
		}

		data, err := original.ToJSON()
		if err != nil {
			t.Fatalf("ToJSON failed: %v", err)
		}

		recovered, err := FromJSON(data)
		if err != nil {
			t.Fatalf("FromJSON failed: %v", err)
		}

		if recovered.ChannelID != original.ChannelID {
			t.Errorf("ChannelID = %q, want %q", recovered.ChannelID, original.ChannelID)
		}
	})
}
