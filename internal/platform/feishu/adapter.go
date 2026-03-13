package feishu

import (
	"context"
	"errors"
	"fmt"

	"github.com/smy-101/cc-connect/internal/core"
)

// Adapter integrates Feishu client with the message router.
// It handles event processing, message conversion, and routing.
type Adapter struct {
	client    FeishuClient
	router    *core.Router
	converter *MessageConverter
	parser    *EventParser
	sender    *Sender
}

// NewAdapter creates a new Feishu adapter.
func NewAdapter(client FeishuClient, router *core.Router) *Adapter {
	converter := NewMessageConverter()
	return &Adapter{
		client:    client,
		router:    router,
		converter: converter,
		parser:    NewEventParser(),
		sender:    NewSenderWithConverter(client, converter),
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

	// Convert event to unified message
	msg, err := a.converter.ToUnifiedMessage(event)
	if err != nil {
		return fmt.Errorf("failed to convert event: %w", err)
	}

	// Route the message
	if err := a.router.Route(ctx, msg); err != nil {
		if errors.Is(err, core.ErrNoHandler) {
			// No handler registered, log but don't fail
			return nil
		}
		return fmt.Errorf("failed to route message: %w", err)
	}

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
	return a.sender.SendText(ctx, chatID, content)
}

// SendMessage sends a unified message through Feishu.
func (a *Adapter) SendMessage(ctx context.Context, msg *core.Message) error {
	return a.sender.SendMessage(ctx, msg)
}
