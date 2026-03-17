package claudecode

import "strings"

// Event types for streaming events from Claude Code CLI
const (
	EventTypeSystem            = "system"
	EventTypeAssistant         = "assistant"
	EventTypeUser              = "user"
	EventTypeResult            = "result"
	EventTypeControlRequest    = "control_request"
	EventTypeControlCancel     = "control_cancel_request"
)

// StreamEvent represents a parsed streaming event from Claude Code CLI
type StreamEvent struct {
	// Type is the event type: system, assistant, user, result, control_request, control_cancel_request
	Type string `json:"type"`
	// Subtype is the event subtype (e.g., init, success, error)
	Subtype string `json:"subtype,omitempty"`
	// SessionID is the Claude Code session identifier
	SessionID string `json:"session_id,omitempty"`
	// CWD is the current working directory
	CWD string `json:"cwd,omitempty"`
	// Model is the model being used (e.g., sonnet, opus)
	Model string `json:"model,omitempty"`
	// PermissionMode is the current permission mode
	PermissionMode string `json:"permissionMode,omitempty"`
	// Tools is the list of available tools
	Tools []string `json:"tools,omitempty"`
	// Message contains the message content for assistant/user events
	Message Message `json:"message,omitempty"`
	// Result is the final result text
	Result string `json:"result,omitempty"`
	// Error is the error message for error events
	Error string `json:"error,omitempty"`
	// TotalCostUSD is the total cost in USD
	TotalCostUSD float64 `json:"total_cost_usd,omitempty"`
	// DurationMs is the duration in milliseconds
	DurationMs int64 `json:"duration_ms,omitempty"`
	// PermissionDenials contains denied tool requests
	PermissionDenials []PermissionDenial `json:"permission_denials,omitempty"`

	// Control request fields (for permission prompts)
	// RequestID is the unique identifier for control_request events
	RequestID string `json:"request_id,omitempty"`
	// Request contains the control request details
	Request *ControlRequest `json:"request,omitempty"`
}

// ControlRequest represents a control request from Claude Code (e.g., permission prompts)
type ControlRequest struct {
	// Subtype is the type of control request (e.g., "can_use_tool")
	Subtype string `json:"subtype,omitempty"`
	// ToolName is the name of the tool being requested (for can_use_tool)
	ToolName string `json:"tool_name,omitempty"`
	// ToolInput is the input for the tool (for can_use_tool)
	ToolInput map[string]interface{} `json:"input,omitempty"`
}

// Message represents a message in assistant/user events
type Message struct {
	Content []Content `json:"content,omitempty"`
}

// Content represents a content block in a message
type Content struct {
	// Type is the content type: text, tool_use, tool_result
	Type string `json:"type"`
	// Text is the text content for text type
	Text string `json:"text,omitempty"`
	// ID is the tool use ID for tool_use type
	ID string `json:"id,omitempty"`
	// Name is the tool name for tool_use type
	Name string `json:"name,omitempty"`
	// Input is the tool input for tool_use type
	Input map[string]interface{} `json:"input,omitempty"`
	// ToolUseID is the referenced tool use ID for tool_result type
	ToolUseID string `json:"tool_use_id,omitempty"`
	// Content is the tool result content for tool_result type
	Content string `json:"content,omitempty"`
}

// PermissionDenial represents a denied tool request
type PermissionDenial struct {
	ToolName  string                 `json:"tool_name"`
	ToolUseID string                 `json:"tool_use_id"`
	ToolInput map[string]interface{} `json:"tool_input,omitempty"`
}

// IsSystemInit returns true if this is a system/init event
func (e *StreamEvent) IsSystemInit() bool {
	return e.Type == "system" && e.Subtype == "init"
}

// IsAssistantText returns true if this is an assistant text event
func (e *StreamEvent) IsAssistantText() bool {
	if e.Type != "assistant" || len(e.Message.Content) == 0 {
		return false
	}
	return e.Message.Content[0].Type == "text"
}

// IsAssistantToolUse returns true if this is an assistant tool_use event
func (e *StreamEvent) IsAssistantToolUse() bool {
	if e.Type != "assistant" || len(e.Message.Content) == 0 {
		return false
	}
	return e.Message.Content[0].Type == "tool_use"
}

// IsUserToolResult returns true if this is a user tool_result event
func (e *StreamEvent) IsUserToolResult() bool {
	if e.Type != "user" || len(e.Message.Content) == 0 {
		return false
	}
	return e.Message.Content[0].Type == "tool_result"
}

// IsResultSuccess returns true if this is a result/success event
func (e *StreamEvent) IsResultSuccess() bool {
	return e.Type == "result" && e.Subtype == "success"
}

// IsResultError returns true if this is a result/error event
func (e *StreamEvent) IsResultError() bool {
	return e.Type == "result" && e.Subtype == "error"
}

// HasPermissionDenials returns true if this event has permission denials
func (e *StreamEvent) HasPermissionDenials() bool {
	return len(e.PermissionDenials) > 0
}

// IsControlRequest returns true if this is a control_request event
func (e *StreamEvent) IsControlRequest() bool {
	return e.Type == EventTypeControlRequest
}

// IsControlCancel returns true if this is a control_cancel_request event
func (e *StreamEvent) IsControlCancel() bool {
	return e.Type == EventTypeControlCancel
}

// GetRequestID returns the request ID for control_request events
func (e *StreamEvent) GetRequestID() string {
	return e.RequestID
}

// HasPermissionRequest returns true if this event is a permission request
func (e *StreamEvent) HasPermissionRequest() bool {
	return e.IsControlRequest() && e.Request != nil && e.Request.Subtype == "can_use_tool"
}

// GetToolName returns the tool name for permission request events
func (e *StreamEvent) GetToolName() string {
	if e.Request != nil {
		return e.Request.ToolName
	}
	return ""
}

// GetText returns the text content for assistant text events
func (e *StreamEvent) GetText() string {
	if e.Type != "assistant" || len(e.Message.Content) == 0 {
		return ""
	}

	parts := make([]string, 0, len(e.Message.Content))
	for _, content := range e.Message.Content {
		if content.Type == "text" && content.Text != "" {
			parts = append(parts, content.Text)
		}
	}

	return strings.Join(parts, "")
}

// GetToolInfo returns tool info for assistant tool_use events
func (e *StreamEvent) GetToolInfo() (name, id string, input map[string]interface{}) {
	if !e.IsAssistantToolUse() {
		return "", "", nil
	}
	c := e.Message.Content[0]
	return c.Name, c.ID, c.Input
}
