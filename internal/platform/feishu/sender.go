package feishu

import (
	"context"
	"errors"

	"github.com/smy-101/cc-connect/internal/core"
)

// Sender handles sending messages to Feishu.
type Sender struct {
	client    FeishuClient
	converter *MessageConverter
}

// NewSender creates a new Sender with the given client.
func NewSender(client FeishuClient) *Sender {
	return &Sender{
		client:    client,
		converter: NewMessageConverter(),
	}
}

// NewSenderWithConverter creates a new Sender with a custom converter.
func NewSenderWithConverter(client FeishuClient, converter *MessageConverter) *Sender {
	return &Sender{
		client:    client,
		converter: converter,
	}
}

// SendText sends a text message to the specified chat.
func (s *Sender) SendText(ctx context.Context, chatID, content string) error {
	if chatID == "" {
		return errors.New("chatID cannot be empty")
	}
	if content == "" {
		return errors.New("content cannot be empty")
	}

	return s.client.SendText(ctx, chatID, content)
}

// SendMessage sends a unified message to Feishu.
// The message is converted to Feishu format before sending.
func (s *Sender) SendMessage(ctx context.Context, msg *core.Message) error {
	if msg == nil {
		return errors.New("message cannot be nil")
	}

	// Convert to Feishu content format
	feishuContent, err := s.converter.ToFeishuContent(msg)
	if err != nil {
		return err
	}

	// Use ChannelID as the chat ID
	chatID := msg.ChannelID
	if chatID == "" {
		return errors.New("message has no channel ID")
	}

	return s.client.SendText(ctx, chatID, feishuContent)
}

// SendUnifiedMessage is an alias for SendMessage for clarity.
func (s *Sender) SendUnifiedMessage(ctx context.Context, msg *core.Message) error {
	return s.SendMessage(ctx, msg)
}
