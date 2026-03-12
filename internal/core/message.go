package core

import (
	"encoding/json"
	"errors"
	"time"
)

// MessageType 消息类型枚举
type MessageType string

const (
	// MessageTypeText 文本消息类型
	MessageTypeText MessageType = "text"
	// MessageTypeVoice 语音消息类型
	MessageTypeVoice MessageType = "voice"
	// MessageTypeImage 图片消息类型
	MessageTypeImage MessageType = "image"
	// MessageTypeCommand 命令消息类型
	MessageTypeCommand MessageType = "command"
)

// Message 统一消息结构
type Message struct {
	ID        string      `json:"id"`
	Platform  string      `json:"platform"`
	UserID    string      `json:"user_id"`
	Content   string      `json:"content"`
	Type      MessageType `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
}

// NewMessage 创建新消息（通用构造函数）
func NewMessage(platform, userID, content string, msgType MessageType) *Message {
	return &Message{
		ID:        GenerateMessageID(),
		Platform:  platform,
		UserID:    userID,
		Content:   content,
		Type:      msgType,
		Timestamp: time.Now(),
	}
}

// NewTextMessage 创建文本消息（便捷方法）
func NewTextMessage(platform, userID, content string) *Message {
	return NewMessage(platform, userID, content, MessageTypeText)
}

// NewVoiceMessage 创建语音消息
func NewVoiceMessage(platform, userID, content string) *Message {
	return NewMessage(platform, userID, content, MessageTypeVoice)
}

// NewImageMessage 创建图片消息
func NewImageMessage(platform, userID, content string) *Message {
	return NewMessage(platform, userID, content, MessageTypeImage)
}

// NewCommandMessage 创建命令消息
func NewCommandMessage(platform, userID, content string) *Message {
	return NewMessage(platform, userID, content, MessageTypeCommand)
}

// ToJSON 序列化为 JSON 字节流
func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// FromJSON 从 JSON 字节流反序列化
func FromJSON(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	// 验证必需字段
	if msg.ID == "" {
		return nil, errors.New("missing required field: id")
	}
	if msg.Platform == "" {
		return nil, errors.New("missing required field: platform")
	}
	if msg.UserID == "" {
		return nil, errors.New("missing required field: user_id")
	}
	if msg.Type == "" {
		return nil, errors.New("missing required field: type")
	}
	if msg.Timestamp.IsZero() {
		return nil, errors.New("missing required field: timestamp")
	}

	return &msg, nil
}
