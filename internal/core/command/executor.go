package command

import (
	"context"

	"github.com/smy-101/cc-connect/internal/agent"
	"github.com/smy-101/cc-connect/internal/core"
)

// Executor executes slash commands and returns results.
// It holds references to the Agent and SessionManager for command operations.
type Executor struct {
	agent    agent.Agent
	sessions *core.SessionManager
}

// NewExecutor creates a new command executor.
func NewExecutor(ag agent.Agent, sessions *core.SessionManager) *Executor {
	return &Executor{
		agent:    ag,
		sessions: sessions,
	}
}

// Execute executes a command and returns the result.
// It routes commands to the appropriate handler based on the command name.
func (e *Executor) Execute(ctx context.Context, cmd Command, msg *core.Message) CommandResult {
	// Handle empty command
	if cmd.IsEmpty() {
		return CommandResult{
			Message: "无效的命令",
			Error:   ErrEmptyCommand,
		}
	}

	// Route to handlers
	switch cmd.Name {
	case "help":
		return e.handleHelp(ctx, cmd, msg)
	case "mode":
		return e.handleMode(ctx, cmd, msg)
	case "new":
		return e.handleNew(ctx, cmd, msg)
	case "list":
		return e.handleList(ctx, cmd, msg)
	case "stop":
		return e.handleStop(ctx, cmd, msg)
	default:
		return CommandResult{
			Message: "未知命令: /" + cmd.Name + "\n输入 /help 查看可用命令",
			Error:   ErrUnknownCommand,
		}
	}
}
