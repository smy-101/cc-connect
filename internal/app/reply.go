package app

import (
	"context"
	"fmt"

	"github.com/smy-101/cc-connect/internal/platform/feishu"
)

// ReplySender defines the interface for sending replies to a message source.
// This is a small interface with a single method, making it easy to mock for testing.
type ReplySender interface {
	// SendReply sends a reply with the given content to the message source.
	SendReply(ctx context.Context, content string) error
}

// replySender adapts feishu.Adapter to the ReplySender interface.
// It holds a reference to the adapter and the channel ID for sending replies.
type replySender struct {
	adapter   *feishu.Adapter
	channelID string
}

// newReplySender creates a new ReplySender for the given channel.
func newReplySender(adapter *feishu.Adapter, channelID string) ReplySender {
	return &replySender{
		adapter:   adapter,
		channelID: channelID,
	}
}

// SendReply implements ReplySender by sending the content through the Feishu adapter.
func (r *replySender) SendReply(ctx context.Context, content string) error {
	if r.adapter == nil {
		return fmt.Errorf("adapter is nil")
	}
	if r.channelID == "" {
		return fmt.Errorf("channelID is empty")
	}
	return r.adapter.SendReply(ctx, r.channelID, content)
}
