package agent

import (
	"context"
	"errors"
	"time"
)

// Common errors
var (
	// ErrAgentBusy is returned when SendMessage is called while another request is in progress
	ErrAgentBusy = errors.New("agent is busy processing another request")
	// ErrAgentNotRunning is returned when an operation requires a running agent
	ErrAgentNotRunning = errors.New("agent is not running")
	// ErrAgentAlreadyRunning is returned when Start is called on a running agent
	ErrAgentAlreadyRunning = errors.New("agent is already running")
	// ErrEmptyInput is returned when SendMessage is called with empty content
	ErrEmptyInput = errors.New("input content cannot be empty")
	// ErrInvalidPermissionMode is returned when an invalid permission mode is specified
	ErrInvalidPermissionMode = errors.New("invalid permission mode")
	// ErrClaudeNotFound is returned when Claude Code CLI is not installed or not executable
	ErrClaudeNotFound = errors.New("claude code CLI not found or not executable")
)

// PermissionMode represents Claude Code permission modes
type PermissionMode string

const (
	// PermissionModeDefault requires approval for all tools
	PermissionModeDefault PermissionMode = "default"
	// PermissionModeAcceptEdits auto-approves edit tools
	PermissionModeAcceptEdits PermissionMode = "acceptEdits"
	// PermissionModePlan auto-approves read-only tools
	PermissionModePlan PermissionMode = "plan"
	// PermissionModeBypassPermissions auto-approves all tools
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
)

// AgentStatus represents the current status of an agent
type AgentStatus string

const (
	// AgentStatusIdle indicates the agent is idle (not started)
	AgentStatusIdle AgentStatus = "idle"
	// AgentStatusStarting indicates the agent is starting
	AgentStatusStarting AgentStatus = "starting"
	// AgentStatusRunning indicates the agent is running and ready
	AgentStatusRunning AgentStatus = "running"
	// AgentStatusStopped indicates the agent has been stopped
	AgentStatusStopped AgentStatus = "stopped"
	// AgentStatusError indicates the agent encountered an error
	AgentStatusError AgentStatus = "error"
)

// StreamEventType represents the type of streaming event
type StreamEventType string

const (
	// StreamEventTypeSystem indicates a system event (init, etc.)
	StreamEventTypeSystem StreamEventType = "system"
	// StreamEventTypeText indicates a text content event
	StreamEventTypeText StreamEventType = "text"
	// StreamEventTypeToolUse indicates a tool use event
	StreamEventTypeToolUse StreamEventType = "tool_use"
	// StreamEventTypeToolResult indicates a tool result event
	StreamEventTypeToolResult StreamEventType = "tool_result"
	// StreamEventTypeResult indicates a final result event
	StreamEventTypeResult StreamEventType = "result"
	// StreamEventTypeError indicates an error event
	StreamEventTypeError StreamEventType = "error"
)

// ToolInfo contains information about a tool call
type ToolInfo struct {
	Name  string                 `json:"name"`
	ID    string                 `json:"id"`
	Input map[string]interface{} `json:"input,omitempty"`
}

// StreamEvent represents a streaming event from the agent
type StreamEvent struct {
	Type    StreamEventType `json:"type"`
	Content string          `json:"content,omitempty"`
	Tool    *ToolInfo       `json:"tool,omitempty"`
}

// DeniedTool represents a tool that was denied permission
type DeniedTool struct {
	ToolName  string                 `json:"tool_name"`
	ToolUseID string                 `json:"tool_use_id"`
	ToolInput map[string]interface{} `json:"tool_input,omitempty"`
}

// Response represents the final response from an agent
type Response struct {
	Content          string        `json:"content"`
	IsError          bool          `json:"is_error"`
	PermissionDenied bool          `json:"permission_denied"`
	DeniedTools      []DeniedTool  `json:"denied_tools,omitempty"`
	CostUSD          float64       `json:"cost_usd"`
	Duration         time.Duration `json:"duration"`
}

// EventHandler is a callback function for handling streaming events
type EventHandler func(event StreamEvent)

// Agent defines the interface for AI agent implementations
type Agent interface {
	// SendMessage sends a message to the agent and returns the response.
	// If handler is non-nil, streaming events are passed to the handler.
	// If handler is nil, the method blocks until the complete response is ready.
	SendMessage(ctx context.Context, content string, handler EventHandler) (*Response, error)

	// SetPermissionMode changes the permission mode.
	// This may require restarting the underlying process.
	SetPermissionMode(mode PermissionMode) error

	// CurrentMode returns the current permission mode.
	CurrentMode() PermissionMode

	// SessionID returns the Claude Code session ID.
	SessionID() string

	// Start starts the agent.
	Start(ctx context.Context) error

	// Stop stops the agent.
	Stop() error

	// Status returns the current agent status.
	Status() AgentStatus

	// Restart restarts the agent, preserving the session.
	Restart(ctx context.Context) error

	// RespondPermission responds to a pending permission request.
	// behavior is typically "allow", "deny", or "answer:<value>" for AskUserQuestion.
	RespondPermission(requestID, behavior string) error
}
