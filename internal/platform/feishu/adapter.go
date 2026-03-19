package feishu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/smy-101/cc-connect/internal/core"
	"github.com/smy-101/cc-connect/internal/core/command"
)

// ReplyContext holds context for replying to a message.
type ReplyContext struct {
	ChatID    string
	MessageID string
}

// Adapter integrates Feishu client with the message router.
// It handles event processing, message conversion, and routing.
type Adapter struct {
	client    FeishuClient
	router    *core.Router
	converter *MessageConverter
	parser    *EventParser
	sender    *Sender

	// Event deduplication: Feishu may re-deliver events if ACK is slow
	seenEvents   map[string]time.Time
	seenEventsMu sync.Mutex
}

// NewAdapter creates a new Feishu adapter.
func NewAdapter(client FeishuClient, router *core.Router) *Adapter {
	converter := NewMessageConverter()
	return &Adapter{
		client:     client,
		router:     router,
		converter:  converter,
		parser:     NewEventParser(),
		sender:     NewSenderWithConverter(client, converter),
		seenEvents: make(map[string]time.Time),
	}
}

// Start starts the adapter by establishing the Feishu connection.
func (a *Adapter) Start(ctx context.Context) error {
	// Register event handler
	a.client.OnEvent(func(ctx context.Context, event *MessageReceiveEvent) error {
		return a.HandleEvent(ctx, event)
	})

	// Connect to Feishu
	return a.client.Connect(ctx)
}

// Stop stops the adapter by disconnecting from Feishu.
func (a *Adapter) Stop() error {
	return a.client.Disconnect()
}

// HandleEvent processes a Feishu message event and routes it.
func (a *Adapter) HandleEvent(ctx context.Context, event *MessageReceiveEvent) error {
	if event == nil {
		return errors.New("event is nil")
	}

	// Ignore messages from the bot itself (sender_type = "app") to prevent message loops
	// Feishu may deliver the bot's own messages back as events
	if event.Sender.SenderType != "" && event.Sender.SenderType != "user" {
		slog.Debug("Feishu event ignored (not from user)", "event_id", event.EventID, "sender_type", event.Sender.SenderType)
		return nil
	}

	// Deduplicate events: Feishu may re-deliver the same event if ACK was slow
	if event.EventID != "" {
		a.seenEventsMu.Lock()
		if _, seen := a.seenEvents[event.EventID]; seen {
			a.seenEventsMu.Unlock()
			slog.Debug("Feishu event deduplicated (already processed)", "event_id", event.EventID)
			return nil
		}
		a.seenEvents[event.EventID] = time.Now()
		// Evict old entries to prevent memory leak (keep last 10 minutes)
		for id, ts := range a.seenEvents {
			if time.Since(ts) > 10*time.Minute {
				delete(a.seenEvents, id)
			}
		}
		a.seenEventsMu.Unlock()
	}

	// Convert event to unified message
	msg, err := a.converter.ToUnifiedMessage(event)
	if err != nil {
		slog.Warn("Feishu event conversion failed", append(eventLogFields(event), "error", err)...)
		return fmt.Errorf("failed to convert event: %w", err)
	}
	slog.Debug("Feishu event converted", append(eventLogFields(event), unifiedMessageLogFields(msg)...)...)

	// Detect Claude Code commands (double slash): remove one slash and send to Agent
	// This must be checked before single slash detection
	if msg.Type == core.MessageTypeText && command.IsClaudeCodeCommand(msg.Content) {
		// Remove one slash: //cost → /cost
		msg.Content = strings.TrimPrefix(msg.Content, "/")
		// Keep as MessageTypeText so it flows to Agent (not converted to Command)
	} else if msg.Type == core.MessageTypeText && command.IsCommand(msg.Content) {
		// Detect cc-connect slash commands: convert text messages starting with '/' to command type
		msg.Type = core.MessageTypeCommand
	}

	// Route the message
	if err := a.router.Route(ctx, msg); err != nil {
		if errors.Is(err, core.ErrNoHandler) {
			slog.Debug("Feishu message has no registered handler", append(eventLogFields(event), unifiedMessageLogFields(msg)...)...)
			return nil
		}
		slog.Error("Feishu message routing failed", append(append(eventLogFields(event), unifiedMessageLogFields(msg)...), "error", err)...)
		return fmt.Errorf("failed to route message: %w", err)
	}
	slog.Debug("Feishu message routed", append(eventLogFields(event), unifiedMessageLogFields(msg)...)...)

	return nil
}

// HandleRawEvent processes a raw Feishu event (JSON bytes) and routes it.
func (a *Adapter) HandleRawEvent(ctx context.Context, data []byte) error {
	event, err := a.parser.Parse(data)
	if err != nil {
		return fmt.Errorf("failed to parse event: %w", err)
	}

	return a.HandleEvent(ctx, event)
}

// HandleSDKEvent processes an SDK event (map format) and routes it.
func (a *Adapter) HandleSDKEvent(ctx context.Context, data map[string]any) error {
	event, err := a.parser.ParseFromMap(data)
	if err != nil {
		return fmt.Errorf("failed to parse SDK event: %w", err)
	}

	return a.HandleEvent(ctx, event)
}

// SendReply sends a text reply to a chat.
func (a *Adapter) SendReply(ctx context.Context, chatID, content string) error {
	return a.sender.SendMessage(ctx, &core.Message{
		ChannelID: chatID,
		Content:   content,
		Type:      core.MessageTypeText,
	})
}

// SendMessage sends a unified message through Feishu.
func (a *Adapter) SendMessage(ctx context.Context, msg *core.Message) error {
	return a.sender.SendMessage(ctx, msg)
}

// SendCard sends an interactive card message to the specified chat.
func (a *Adapter) SendCard(ctx context.Context, chatID string, card *core.Card) error {
	if card == nil {
		return errors.New("card cannot be nil")
	}
	if chatID == "" {
		return errors.New("chatID cannot be empty")
	}

	cardMap := renderCardMap(card)
	cardJSON, err := json.Marshal(cardMap)
	if err != nil {
		return fmt.Errorf("failed to marshal card: %w", err)
	}

	return a.client.SendCard(ctx, chatID, cardJSON)
}

// ReplyCard sends an interactive card as a reply to a message.
func (a *Adapter) ReplyCard(ctx context.Context, replyCtx *ReplyContext, card *core.Card) error {
	if card == nil {
		return errors.New("card cannot be nil")
	}
	if replyCtx == nil {
		return errors.New("replyCtx cannot be nil")
	}
	if replyCtx.ChatID == "" {
		return errors.New("chatID cannot be empty")
	}
	if replyCtx.MessageID == "" {
		return errors.New("messageID cannot be empty")
	}

	cardMap := renderCardMap(card)
	cardJSON, err := json.Marshal(cardMap)
	if err != nil {
		return fmt.Errorf("failed to marshal card: %w", err)
	}

	return a.client.ReplyCard(ctx, replyCtx.ChatID, replyCtx.MessageID, cardJSON)
}

// HandleCardCallback processes a card interaction callback from Feishu.
// The callback data is parsed and converted to a command message, then routed.
func (a *Adapter) HandleCardCallback(ctx context.Context, data map[string]any) error {
	msg, err := ParseCardCallback(data)
	if err != nil {
		slog.Warn("Feishu card callback parsing failed", "error", err)
		return fmt.Errorf("failed to parse card callback: %w", err)
	}

	slog.Debug("Feishu card callback parsed",
		"user_id", msg.UserID,
		"channel_id", msg.ChannelID,
		"content", msg.Content,
	)

	// Route the command message
	if err := a.router.Route(ctx, msg); err != nil {
		if errors.Is(err, core.ErrNoHandler) {
			slog.Debug("Feishu card callback has no registered handler", "content", msg.Content)
			return nil
		}
		slog.Error("Feishu card callback routing failed", "error", err, "content", msg.Content)
		return fmt.Errorf("failed to route card callback: %w", err)
	}

	slog.Debug("Feishu card callback routed", "content", msg.Content)
	return nil
}
