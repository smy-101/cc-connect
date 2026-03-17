package feishu

import (
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/smy-101/cc-connect/internal/core"
)

func sdkEventLogFields(event *larkim.P2MessageReceiveV1) []any {
	if event == nil || event.Event == nil || event.Event.Message == nil {
		return nil
	}

	message := event.Event.Message
	senderType := ""
	if event.Event.Sender != nil {
		senderType = stringValue(event.Event.Sender.SenderType)
	}

	return []any{
		"event_id", eventIDFromSDK(event),
		"message_id", stringValue(message.MessageId),
		"chat_id", stringValue(message.ChatId),
		"chat_type", stringValue(message.ChatType),
		"message_type", stringValue(message.MessageType),
		"sender_type", senderType,
		"content_length", len(stringValue(message.Content)),
	}
}

func eventLogFields(event *MessageReceiveEvent) []any {
	if event == nil {
		return nil
	}

	return []any{
		"event_id", event.EventID,
		"message_id", event.MessageID,
		"chat_id", event.ChatID,
		"chat_type", event.ChatType,
		"message_type", event.MessageType,
		"sender_type", event.Sender.SenderType,
		"content_length", len(event.Content),
	}
}

func unifiedMessageLogFields(msg *core.Message) []any {
	if msg == nil {
		return nil
	}

	return []any{
		"message_id", msg.ID,
		"channel_id", msg.ChannelID,
		"message_type", msg.Type,
		"content_length", len(msg.Content),
	}
}

func sendLogFields(chatID, content string) []any {
	return []any{
		"chat_id", chatID,
		"content_length", len(content),
	}
}
