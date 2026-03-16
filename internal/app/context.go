package app

import (
	"context"

	"github.com/smy-101/cc-connect/internal/core"
)

// HandlerContext encapsulates all the context needed by message handlers.
// It provides access to the request context, message, session, and reply sender.
type HandlerContext struct {
	// Ctx is the request context for cancellation and deadlines.
	Ctx context.Context
	// Msg is the unified message being processed.
	Msg *core.Message
	// Session is the associated session for this message.
	Session *core.Session
	// Reply provides the ability to send replies back to the message source.
	Reply ReplySender
}

// Handler is the function signature for message handlers using HandlerContext.
type Handler func(hctx *HandlerContext) error
