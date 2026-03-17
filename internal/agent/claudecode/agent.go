package claudecode

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/smy-101/cc-connect/internal/agent"
)

// Config contains configuration for ClaudeCodeAgent
type Config struct {
	SessionID      string
	WorkingDir     string
	PermissionMode agent.PermissionMode
	ClaudePath     string
	AllowedTools   []string
	Env            []string
}

// ClaudeCodeAgent implements the Agent interface for Claude Code CLI
type ClaudeCodeAgent struct {
	config         *Config
	sessionID      string
	permissionMode agent.PermissionMode
	status         agent.AgentStatus
	session        *Session // Persistent session for bidirectional communication
	mu             sync.RWMutex
	sessionMu      sync.Mutex // Protects session access
	busy           bool       // true when a SendMessage is in progress
}

// NewAgent creates a new Claude Code agent
func NewAgent(config *Config) (*ClaudeCodeAgent, error) {
	if config.WorkingDir == "" {
		return nil, fmt.Errorf("working directory is required")
	}

	// Use provided session ID for resuming, or empty for new session
	// Claude will generate a session ID when the session starts
	sessionID := config.SessionID

	// Default permission mode
	permMode := config.PermissionMode
	if permMode == "" {
		permMode = agent.PermissionModeDefault
	}

	return &ClaudeCodeAgent{
		config:         config,
		sessionID:      sessionID,
		permissionMode: permMode,
		status:         agent.AgentStatusIdle,
	}, nil
}

// SessionID returns the session ID
func (a *ClaudeCodeAgent) SessionID() string {
	return a.sessionID
}

// CurrentMode returns the current permission mode
func (a *ClaudeCodeAgent) CurrentMode() agent.PermissionMode {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.permissionMode
}

// Status returns the current agent status
func (a *ClaudeCodeAgent) Status() agent.AgentStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

// Start starts the agent
func (a *ClaudeCodeAgent) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.status == agent.AgentStatusRunning {
		return agent.ErrAgentAlreadyRunning
	}

	// Create a new session with stream mode
	// Only pass SessionID if it's a resume (has existing session to resume)
	// For new sessions, let Claude generate the session ID
	sessionConfig := &SessionConfig{
		WorkingDir:     a.config.WorkingDir,
		ClaudePath:     a.config.ClaudePath,
		SessionID:      a.sessionID, // Will be empty for new sessions
		PermissionMode: PermissionModeToCLIArg(a.permissionMode),
		AutoApprove:    a.permissionMode == agent.PermissionModeBypassPermissions,
		AllowedTools:   a.config.AllowedTools,
		Env:            a.config.Env,
	}

	session, err := newSession(ctx, sessionConfig)
	if err != nil {
		a.status = agent.AgentStatusError
		return fmt.Errorf("failed to create session: %w", err)
	}

	a.sessionMu.Lock()
	a.session = session
	// Update session ID from the created session
	a.sessionID = session.CurrentSessionID()
	a.sessionMu.Unlock()
	a.status = agent.AgentStatusRunning
	slog.Debug("Agent started with persistent session", "session_id", a.sessionID)
	return nil
}

// Stop stops the agent
func (a *ClaudeCodeAgent) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.status != agent.AgentStatusRunning {
		return nil
	}

	a.sessionMu.Lock()
	if a.session != nil {
		if err := a.session.Close(); err != nil {
			slog.Warn("Error closing session", "error", err)
		}
		a.session = nil
	}
	a.sessionMu.Unlock()

	a.status = agent.AgentStatusStopped
	return nil
}

// Restart restarts the agent, optionally changing the permission mode
func (a *ClaudeCodeAgent) Restart(ctx context.Context) error {
	a.mu.Lock()

	if a.status != agent.AgentStatusRunning {
		a.mu.Unlock()
		return agent.ErrAgentNotRunning
	}

	// Get current session ID for resume
	currentSessionID := a.sessionID
	a.mu.Unlock()

	// Stop and start
	if err := a.Stop(); err != nil {
		return fmt.Errorf("failed to stop: %w", err)
	}

	// Update session ID to resume from previous session
	a.sessionID = currentSessionID
	return a.Start(ctx)
}

// SetPermissionMode changes the permission mode
func (a *ClaudeCodeAgent) SetPermissionMode(mode agent.PermissionMode) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Update mode
	oldMode := a.permissionMode
	a.permissionMode = mode

	if a.status != agent.AgentStatusRunning {
		// Mode will be used on next start
		return nil
	}

	// Close current session
	a.sessionMu.Lock()
	if a.session != nil {
		if err := a.session.Close(); err != nil {
			slog.Warn("Error closing session during mode change", "error", err)
		}
		a.session = nil
	}
	a.sessionMu.Unlock()

	// Create new session with new mode
	sessionConfig := &SessionConfig{
		WorkingDir:     a.config.WorkingDir,
		ClaudePath:     a.config.ClaudePath,
		SessionID:      a.sessionID, // Resume existing session
		PermissionMode: PermissionModeToCLIArg(mode),
		AutoApprove:    mode == agent.PermissionModeBypassPermissions,
		AllowedTools:   a.config.AllowedTools,
		Env:            a.config.Env,
	}

	session, err := newSession(context.Background(), sessionConfig)
	if err != nil {
		a.permissionMode = oldMode // Revert on error
		a.status = agent.AgentStatusError
		return fmt.Errorf("failed to create session with new mode: %w", err)
	}

	a.sessionMu.Lock()
	a.session = session
	a.sessionMu.Unlock()

	slog.Debug("Permission mode changed", "old_mode", oldMode, "new_mode", mode)
	return nil
}

// SendMessage sends a message to the agent
func (a *ClaudeCodeAgent) SendMessage(ctx context.Context, content string, handler agent.EventHandler) (*agent.Response, error) {
	if content == "" {
		return nil, agent.ErrEmptyInput
	}

	a.mu.RLock()
	if a.status != agent.AgentStatusRunning {
		a.mu.RUnlock()
		return nil, agent.ErrAgentNotRunning
	}
	a.mu.RUnlock()

	a.sessionMu.Lock()
	if a.session == nil || !a.session.Alive() {
		a.sessionMu.Unlock()
		return nil, agent.ErrAgentNotRunning
	}
	session := a.session
	a.sessionMu.Unlock()

	a.mu.Lock()
	if a.busy {
		a.mu.Unlock()
		return nil, agent.ErrAgentBusy
	}
	a.busy = true
	a.mu.Unlock()

	// Ensure we clear busy flag when done
	defer func() {
		a.mu.Lock()
		a.busy = false
		a.mu.Unlock()
	}()

	// Collect response from events via callback
	response := &agent.Response{}
	var textBuilder strings.Builder
	startTime := time.Now()

	slog.Debug("Sending message to session", "session_id", a.sessionID, "message_length", len(content))

	// Send the message using the session's SendMessage method
	finalEvent, err := session.SendMessage(ctx, content, nil, nil, func(event Event) {
		// Convert session event to agent event and call handler
		agentEvent := convertSessionEventToAgentEvent(event)
		if handler != nil {
			handler(agentEvent)
		}

		// Accumulate text
		if event.Type == EventText && event.Content != "" {
			preview := event.Content
			if len(preview) > 50 {
				preview = preview[:50] + "..."
			}
			slog.Debug("Received text event", "text_length", len(event.Content), "text_preview", preview)
			textBuilder.WriteString(event.Content)
		}
	})

	if err != nil {
		return nil, fmt.Errorf("session send failed: %w", err)
	}

	// Process final event
	if finalEvent != nil {
		switch finalEvent.Type {
		case EventResult:
			slog.Debug("Received result event", "result_length", len(finalEvent.Content))
			response.Content = finalEvent.Content
			if textBuilder.Len() > 0 && response.Content == "" {
				response.Content = textBuilder.String()
			}
			response.Duration = time.Since(startTime)
		case EventError:
			slog.Error("Received error event", "error", finalEvent.Error)
			response.IsError = true
			if finalEvent.Content != "" {
				response.Content = finalEvent.Content
			} else if finalEvent.Error != nil {
				response.Content = finalEvent.Error.Error()
			}
		}
	}

	// Update session ID if it changed
	a.sessionMu.Lock()
	if session := a.session; session != nil {
		a.sessionID = session.CurrentSessionID()
	}
	a.sessionMu.Unlock()

	slog.Debug("SendMessage completed", "content_length", len(response.Content), "is_error", response.IsError)
	return response, nil
}

// convertSessionEventToAgentEvent converts a Session Event to agent.StreamEvent
func convertSessionEventToAgentEvent(event Event) agent.StreamEvent {
	ae := agent.StreamEvent{}

	switch event.Type {
	case EventSystem:
		ae.Type = agent.StreamEventTypeSystem
	case EventText:
		ae.Type = agent.StreamEventTypeText
		ae.Content = event.Content
	case EventToolUse:
		ae.Type = agent.StreamEventTypeToolUse
		ae.Tool = &agent.ToolInfo{
			Name:  event.ToolName,
			Input: event.ToolInput,
		}
	case EventToolResult:
		ae.Type = agent.StreamEventTypeToolResult
	case EventResult:
		ae.Type = agent.StreamEventTypeResult
		ae.Content = event.Content
	case EventError:
		ae.Type = agent.StreamEventTypeError
		ae.Content = event.Content
	case EventPermissionRequest:
		// Permission requests don't have a direct mapping in agent.StreamEvent
		// They are handled internally by the session
	}

	return ae
}

// convertToAgentEvent converts a claudecode.StreamEvent to agent.StreamEvent
// This is kept for backward compatibility with tests
func convertToAgentEvent(event *StreamEvent) agent.StreamEvent {
	ae := agent.StreamEvent{}

	switch {
	case event.IsSystemInit():
		ae.Type = agent.StreamEventTypeSystem
	case event.IsAssistantText():
		ae.Type = agent.StreamEventTypeText
		ae.Content = event.GetText()
	case event.IsAssistantToolUse():
		ae.Type = agent.StreamEventTypeToolUse
		name, id, input := event.GetToolInfo()
		ae.Tool = &agent.ToolInfo{
			Name:  name,
			ID:    id,
			Input: input,
		}
	case event.IsUserToolResult():
		ae.Type = agent.StreamEventTypeToolResult
	case event.IsResultSuccess():
		ae.Type = agent.StreamEventTypeResult
		ae.Content = event.Result
	case event.IsResultError():
		ae.Type = agent.StreamEventTypeError
		ae.Content = event.Error
	}

	return ae
}
