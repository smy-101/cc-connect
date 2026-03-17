package app

import "github.com/smy-101/cc-connect/internal/core"

func messageLogFields(msg *core.Message) []any {
	if msg == nil {
		return nil
	}

	return []any{
		"message_id", msg.ID,
		"channel_id", msg.ChannelID,
		"user_id", msg.UserID,
		"message_type", msg.Type,
		"content_length", len(msg.Content),
	}
}

func replyLogFields(channelID, content string) []any {
	return []any{
		"channel_id", channelID,
		"content_length", len(content),
	}
}
