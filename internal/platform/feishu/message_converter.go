package feishu

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/smy-101/cc-connect/internal/core"
)

// MessageConverter handles conversion between Feishu message format
// and the unified message model.
type MessageConverter struct{}

// NewMessageConverter creates a new MessageConverter instance.
func NewMessageConverter() *MessageConverter {
	return &MessageConverter{}
}

// textContent represents the JSON structure of a Feishu text message content.
type textContent struct {
	Text string `json:"text"`
}

// ToUnifiedMessage converts a Feishu MessageReceiveEvent to a unified Message.
func (c *MessageConverter) ToUnifiedMessage(event *MessageReceiveEvent) (*core.Message, error) {
	if event == nil {
		return nil, errors.New("event is nil")
	}

	switch event.MessageType {
	case "text":
		return c.textToUnifiedMessage(event)
	default:
		return nil, errors.New("unsupported message type: " + event.MessageType)
	}
}

// textToUnifiedMessage converts a Feishu text message to unified message.
func (c *MessageConverter) textToUnifiedMessage(event *MessageReceiveEvent) (*core.Message, error) {
	var content textContent
	if err := json.Unmarshal([]byte(event.Content), &content); err != nil {
		return nil, errors.New("failed to parse text content: " + err.Error())
	}

	if content.Text == "" {
		return nil, errors.New("message content cannot be empty")
	}

	msg := &core.Message{
		ID:        generateFeishuMessageID(event.MessageID),
		Platform:  "feishu",
		UserID:    event.Sender.OpenID,
		ChannelID: event.ChatID,
		Content:   content.Text,
		Type:      core.MessageTypeText,
		Timestamp: event.CreateTime,
	}

	// If CreateTime is zero, use current time
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	return msg, nil
}

// GetMentions extracts the mention information from a Feishu message event.
// Returns the list of mentions or nil if there are no mentions.
func (c *MessageConverter) GetMentions(event *MessageReceiveEvent) []MentionInfo {
	if event == nil || len(event.Mentions) == 0 {
		return nil
	}

	// Return a copy to avoid modification of the original
	mentions := make([]MentionInfo, len(event.Mentions))
	copy(mentions, event.Mentions)
	return mentions
}

// ToFeishuContent converts a unified Message to Feishu API content format.
// Returns the JSON string that can be used as the content parameter in Feishu API calls.
func (c *MessageConverter) ToFeishuContent(msg *core.Message) (string, error) {
	if msg == nil {
		return "", errors.New("message is nil")
	}

	switch msg.Type {
	case core.MessageTypeText:
		content := textContent{Text: msg.Content}
		data, err := json.Marshal(content)
		if err != nil {
			return "", errors.New("failed to marshal text content: " + err.Error())
		}
		return string(data), nil
	default:
		return "", errors.New("unsupported message type for Feishu: " + string(msg.Type))
	}
}

// generateFeishuMessageID creates a unique message ID based on the Feishu message ID.
func generateFeishuMessageID(msgID string) string {
	if msgID == "" {
		return core.GenerateMessageID()
	}
	return "feishu_" + msgID
}
