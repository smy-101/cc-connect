package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/smy-101/cc-connect/internal/core"
	"github.com/smy-101/cc-connect/internal/core/command"
)

// wrapHandler wraps a Handler to work with core.Handler.
// It creates the HandlerContext, injects dependencies, and handles panics.
func (a *App) wrapHandler(h Handler) core.Handler {
	return func(ctx context.Context, msg *core.Message) (err error) {
		// Recover from panics
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Handler panic", "error", r, "message_id", msg.ID)
				err = fmt.Errorf("internal error: %v", r)
			}
		}()

		// Create HandlerContext
		sessionID := core.DeriveSessionID(msg)
		session := a.router.Sessions().GetOrCreate(sessionID)

		// Create ReplySender
		replySender := newReplySender(a.feishu, msg.ChannelID)

		hctx := &HandlerContext{
			Ctx:     ctx,
			Msg:     msg,
			Session: session,
			Reply:   replySender,
		}

		return h(hctx)
	}
}

// handleText handles text messages by sending them to the agent.
func (a *App) handleText(hctx *HandlerContext) error {
	// Send thinking status
	if err := hctx.Reply.SendReply(hctx.Ctx, "🤔 正在思考..."); err != nil {
		return fmt.Errorf("failed to send status: %w", err)
	}

	// Create timeout context
	ctx, cancel := context.WithTimeout(hctx.Ctx, a.agentTimeout)
	defer cancel()

	// Send to agent
	resp, err := a.agent.SendMessage(ctx, hctx.Msg.Content, nil)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return hctx.Reply.SendReply(hctx.Ctx, "⏱️ 请求超时，请简化问题或稍后重试。")
		}
		return hctx.Reply.SendReply(hctx.Ctx, fmt.Sprintf("❌ 处理失败: %v", err))
	}

	// Send response
	if resp.IsError {
		return hctx.Reply.SendReply(hctx.Ctx, fmt.Sprintf("❌ %s", resp.Content))
	}
	return hctx.Reply.SendReply(hctx.Ctx, resp.Content)
}

// handleCommand handles command messages.
func (a *App) handleCommand(hctx *HandlerContext) error {
	// Parse command
	cmd := command.Parse(hctx.Msg.Content)
	if cmd.IsEmpty() {
		return hctx.Reply.SendReply(hctx.Ctx, "❌ 无效的命令")
	}

	// Execute command
	result := a.executor.Execute(hctx.Ctx, cmd, hctx.Msg)

	// Send result
	return hctx.Reply.SendReply(hctx.Ctx, result.Message)
}
