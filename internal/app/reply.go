package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/smy-101/cc-connect/internal/platform/feishu"
)

// ReplySender defines the interface for sending replies to a message source.
// This is a small interface with a single method, making it easy to mock for testing.
type ReplySender interface {
	// SendReply sends a reply with the given content to the message source.
	SendReply(ctx context.Context, content string) error
}

// replySender adapts feishu.Adapter to the ReplySender interface.
// It holds a reference to the adapter, the channel ID, and optional project name for sending replies.
type replySender struct {
	adapter     *feishu.Adapter
	channelID   string
	projectName string // Optional: project name to prefix replies
}

// newReplySender creates a new ReplySender for the given channel.
func newReplySender(adapter *feishu.Adapter, channelID string) ReplySender {
	return &replySender{
		adapter:   adapter,
		channelID: channelID,
	}
}

// newReplySenderWithProject creates a new ReplySender with project name prefix.
func newReplySenderWithProject(adapter *feishu.Adapter, channelID string, projectName string) ReplySender {
	return &replySender{
		adapter:     adapter,
		channelID:   channelID,
		projectName: projectName,
	}
}

// SendReply implements ReplySender by sending the content through the Feishu adapter.
// If a project name is set, it prefixes the content with [projectName].
func (r *replySender) SendReply(ctx context.Context, content string) error {
	if r.adapter == nil {
		return fmt.Errorf("adapter is nil")
	}
	if r.channelID == "" {
		return fmt.Errorf("channelID is empty")
	}

	// Add project name prefix if set
	if r.projectName != "" {
		content = fmt.Sprintf("[%s] %s", r.projectName, content)
	}

	slog.Debug("Feishu reply send requested", replyLogFields(r.channelID, content)...)
	if err := r.adapter.SendReply(ctx, r.channelID, content); err != nil {
		slog.Error("Feishu reply send failed", append(replyLogFields(r.channelID, content), "error", err)...)
		return err
	}
	slog.Debug("Feishu reply sent", replyLogFields(r.channelID, content)...)
	return nil
}
