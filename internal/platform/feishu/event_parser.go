package feishu

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// EventParser parses Feishu im.message.receive_v1 events.
type EventParser struct{}

// NewEventParser creates a new EventParser instance.
func NewEventParser() *EventParser {
	return &EventParser{}
}

// rawEvent represents the raw JSON structure of a Feishu v2.0 event.
type rawEvent struct {
	Schema string `json:"schema"`
	Header struct {
		EventID    string `json:"event_id"`
		EventType  string `json:"event_type"`
		CreateTime string `json:"create_time"`
		Token      string `json:"token"`
		AppID      string `json:"app_id"`
		TenantKey  string `json:"tenant_key"`
	} `json:"header"`
	Event struct {
		Sender struct {
			SenderID struct {
				OpenID  string `json:"open_id"`
				UnionID string `json:"union_id"`
				UserID  string `json:"user_id"`
			} `json:"sender_id"`
			SenderType string `json:"sender_type"`
			TenantKey  string `json:"tenant_key"`
		} `json:"sender"`
		Message struct {
			MessageID   string        `json:"message_id"`
			RootID      string        `json:"root_id"`
			ParentID    string        `json:"parent_id"`
			CreateTime  string        `json:"create_time"`
			ChatID      string        `json:"chat_id"`
			ChatType    string        `json:"chat_type"`
			MessageType string        `json:"message_type"`
			Content     string        `json:"content"`
			Mentions    []rawMention  `json:"mentions"`
		} `json:"message"`
	} `json:"event"`
}

// rawMention represents a single mention in the message.
type rawMention struct {
	Key       string `json:"key"`
	ID        struct {
		OpenID  string `json:"open_id"`
		UnionID string `json:"union_id"`
		UserID  string `json:"user_id"`
	} `json:"id"`
	Name      string `json:"name"`
	TenantKey string `json:"tenant_key"`
}

// postContent represents the structure of a Feishu post message.
type postContent struct {
	ZhCN *postLocaleContent `json:"zh_cn"`
	EnUS *postLocaleContent `json:"en_us"`
}

// postLocaleContent represents the content for a specific locale.
type postLocaleContent struct {
	Title   string           `json:"title"`
	Content [][]postElement  `json:"content"`
}

// postElement represents an element in a post message.
type postElement struct {
	Tag      string `json:"tag"`
	Text     string `json:"text"`
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
}

// Parse parses a JSON byte slice into a MessageReceiveEvent.
func (p *EventParser) Parse(data []byte) (*MessageReceiveEvent, error) {
	if len(data) == 0 {
		return nil, errors.New("empty event data")
	}

	var raw rawEvent
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse event JSON: %w (data starts with: %.100s)", err, string(data))
	}

	return p.convertRawEvent(&raw)
}

// ParseFromMap parses a map (typically from SDK) into a MessageReceiveEvent.
func (p *EventParser) ParseFromMap(data map[string]any) (*MessageReceiveEvent, error) {
	if data == nil {
		return nil, errors.New("nil event data")
	}

	// Convert map to JSON then parse (ensures consistent handling)
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event map: %w", err)
	}

	return p.Parse(jsonData)
}

// convertRawEvent converts the raw event structure to MessageReceiveEvent.
func (p *EventParser) convertRawEvent(raw *rawEvent) (*MessageReceiveEvent, error) {
	if raw.Header.EventID == "" {
		return nil, errors.New("missing event_id in header")
	}
	if raw.Event.Message.MessageID == "" {
		return nil, errors.New("missing message_id in event")
	}

	event := &MessageReceiveEvent{
		EventID:     raw.Header.EventID,
		MessageID:   raw.Event.Message.MessageID,
		MessageType: raw.Event.Message.MessageType,
		Content:     raw.Event.Message.Content,
		ChatID:      raw.Event.Message.ChatID,
		ChatType:    raw.Event.Message.ChatType,
		Sender: SenderInfo{
			OpenID:     raw.Event.Sender.SenderID.OpenID,
			UnionID:    raw.Event.Sender.SenderID.UnionID,
			UserID:     raw.Event.Sender.SenderID.UserID,
			SenderType: raw.Event.Sender.SenderType,
		},
		CreateTime: p.parseTimestamp(raw.Event.Message.CreateTime),
		RawEvent:   raw,
	}

	// Convert mentions
	if len(raw.Event.Message.Mentions) > 0 {
		event.Mentions = make([]MentionInfo, len(raw.Event.Message.Mentions))
		for i, m := range raw.Event.Message.Mentions {
			event.Mentions[i] = MentionInfo{
				Key:       m.Key,
				OpenID:    m.ID.OpenID,
				UnionID:   m.ID.UnionID,
				UserID:    m.ID.UserID,
				Name:      m.Name,
				TenantKey: m.TenantKey,
			}
		}
	}

	return event, nil
}

// parseTimestamp converts a Feishu timestamp string (milliseconds) to time.Time.
func (p *EventParser) parseTimestamp(timestamp string) time.Time {
	if timestamp == "" {
		return time.Time{}
	}

	// Parse milliseconds
	ms, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return time.Time{}
	}

	return time.UnixMilli(ms).UTC()
}

// ExtractPostText extracts plain text from a post message event.
// For non-post messages, it returns an error.
func (p *EventParser) ExtractPostText(event *MessageReceiveEvent) (string, error) {
	if event == nil {
		return "", errors.New("event is nil")
	}
	if event.MessageType != "post" {
		return "", errors.New("not a post message")
	}

	var content postContent
	if err := json.Unmarshal([]byte(event.Content), &content); err != nil {
		return "", fmt.Errorf("failed to parse post content: %w", err)
	}

	// Get locale content, prefer zh_cn, fallback to en_us
	locale := content.ZhCN
	if locale == nil {
		locale = content.EnUS
	}
	if locale == nil {
		return "", errors.New("no locale content found in post message")
	}

	var builder strings.Builder

	// Add title if present
	if locale.Title != "" {
		builder.WriteString(locale.Title)
		builder.WriteString("\n")
	}

	// Process content paragraphs
	for i, paragraph := range locale.Content {
		if i > 0 {
			builder.WriteString("\n")
		}
		for _, elem := range paragraph {
			switch elem.Tag {
			case "text":
				builder.WriteString(elem.Text)
			case "at":
				builder.WriteString("@")
				builder.WriteString(elem.UserName)
			}
		}
	}

	return builder.String(), nil
}
