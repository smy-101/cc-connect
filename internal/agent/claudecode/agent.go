package claudecode

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
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
	pm             *ProcessManager
	mu             sync.RWMutex
	busy           bool // true when a SendMessage is in progress
	sessionStarted bool // true if we've created at least one session
}

// NewAgent creates a new Claude Code agent
func NewAgent(config *Config) (*ClaudeCodeAgent, error) {
	if config.WorkingDir == "" {
		return nil, fmt.Errorf("working directory is required")
	}

	// Generate session ID if not provided
	sessionID := config.SessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

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

	// Don't start the main process in -p mode without a message
	// The process will be created on-demand in SendMessage
	slog.Debug("Agent started (no persistent process)", "session_id", a.sessionID)
	a.status = agent.AgentStatusRunning
	return nil
}

// Stop stops the agent
func (a *ClaudeCodeAgent) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.status != agent.AgentStatusRunning {
		return nil
	}

	if a.pm != nil {
		if err := a.pm.Stop(); err != nil {
			return err
		}
	}

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

	// Mark for resume
	a.pm.config.Resume = true
	a.mu.Unlock()

	// Stop and start
	if err := a.Stop(); err != nil {
		return fmt.Errorf("failed to stop: %w", err)
	}

	return a.Start(ctx)
}

// SetPermissionMode changes the permission mode
func (a *ClaudeCodeAgent) SetPermissionMode(mode agent.PermissionMode) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.status != agent.AgentStatusRunning {
		// Just update the mode, will be used on next start
		a.permissionMode = mode
		return nil
	}

	// Need to restart to change mode
	oldMode := a.permissionMode
	a.permissionMode = mode

	// Restart the process with new mode
	pmConfig := &ProcessConfig{
		SessionID:      a.sessionID,
		WorkingDir:     a.config.WorkingDir,
		PermissionMode: mode,
		Resume:         true, // Resume to keep context
		ClaudePath:     a.config.ClaudePath,
		AllowedTools:   a.config.AllowedTools,
		Env:            a.config.Env,
	}

	if err := a.pm.Stop(); err != nil {
		a.permissionMode = oldMode // Revert on error
		return fmt.Errorf("failed to stop process: %w", err)
	}

	a.pm = NewProcessManager(pmConfig)
	if err := a.pm.Start(context.Background()); err != nil {
		a.permissionMode = oldMode // Revert on error
		return fmt.Errorf("failed to start process: %w", err)
	}

	return nil
}

// SendMessage sends a message to the agent
func (a *ClaudeCodeAgent) SendMessage(ctx context.Context, content string, handler agent.EventHandler) (*agent.Response, error) {
	if content == "" {
		return nil, agent.ErrEmptyInput
	}

	a.mu.Lock()
	if a.status != agent.AgentStatusRunning {
		a.mu.Unlock()
		return nil, agent.ErrAgentNotRunning
	}
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

	// Create a new process for this message
	// Use --session-id for first message, --resume for subsequent messages
	pmConfig := &ProcessConfig{
		SessionID:      a.sessionID,
		WorkingDir:     a.config.WorkingDir,
		PermissionMode: a.permissionMode,
		Resume:         a.sessionStarted, // Resume only if we've started a session before
		ClaudePath:     a.config.ClaudePath,
		AllowedTools:   a.config.AllowedTools,
		Env:            a.config.Env,
		Message:        content,
	}

	slog.Debug("Starting Claude Code subprocess", "session_id", a.sessionID, "message", content, "resume", a.sessionStarted)
	pm := NewProcessManager(pmConfig)
	if err := pm.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	// Close stdin so Claude CLI sees EOF and starts processing the message.
	// Without this, Claude waits indefinitely for stdin input in pipe mode.
	if err := pm.CloseStdin(); err != nil {
		slog.Warn("Failed to close stdin", "error", err)
	}
	stderrReader := pm.Stderr()
	var stderrOutput []byte
	var stderrErr error
	stderrDone := make(chan struct{})
	if stderrReader != nil {
		go func() {
			stderrOutput, stderrErr = io.ReadAll(stderrReader)
			if len(stderrOutput) > 0 {
				slog.Debug("Claude stderr output", "output", string(stderrOutput))
			}
			close(stderrDone)
		}()
	} else {
		close(stderrDone)
	}

	// Check if process is still running after a brief moment
	time.Sleep(100 * time.Millisecond)
	if !pm.IsRunning() {
		slog.Error("Claude process exited immediately after starting")
		// Try to get the exit error
		waitErr := pm.Wait()
		<-stderrDone
		if waitErr != nil {
			slog.Error("Process exit error", "error", waitErr, "stderr", string(stderrOutput))
			return nil, fmt.Errorf("claude process exited immediately: %w", waitErr)
		}
		return nil, fmt.Errorf("claude process exited immediately with no error")
	}

	defer pm.Stop()

	// Collect response
	response := &agent.Response{}
	var textBuilder strings.Builder

	// Parse events from stdout with timeout detection
	slog.Debug("Starting to parse events from stdout")
	parseDone := make(chan error, 1)
	go func() {
		err := ParseFromReader(pm.Stdout(), func(event *StreamEvent) error {
			// Convert to agent.StreamEvent and call handler
			agentEvent := convertToAgentEvent(event)
			if handler != nil {
				handler(agentEvent)
			}

			if text := event.GetText(); text != "" {
				preview := text
				if len(preview) > 50 {
					preview = preview[:50] + "..."
				}
				slog.Debug("Received text from Claude", "text_length", len(text), "text_preview", preview)
				textBuilder.WriteString(text)
			}

			// Handle different event types
			switch {
			case event.IsSystemInit():
				slog.Debug("Received system init event", "session_id", event.SessionID)
				// Extract metadata if needed
			case event.IsResultSuccess():
				slog.Debug("Received result success event", "result_length", len(event.Result))
				response.Content = event.Result
				if textBuilder.Len() > 0 && response.Content == "" {
					response.Content = textBuilder.String()
				}
				response.CostUSD = event.TotalCostUSD
				response.Duration = time.Duration(event.DurationMs) * time.Millisecond
			case event.IsResultError():
				slog.Error("Received result error event", "error", event.Error)
				response.IsError = true
				response.Content = event.Error
				if event.HasPermissionDenials() {
					response.PermissionDenied = true
					response.DeniedTools = make([]agent.DeniedTool, len(event.PermissionDenials))
					for i, d := range event.PermissionDenials {
						response.DeniedTools[i] = agent.DeniedTool{
							ToolName:  d.ToolName,
							ToolUseID: d.ToolUseID,
							ToolInput: d.ToolInput,
						}
					}
				}
			}

			return nil
		})
		parseDone <- err
	}()

	// Wait for parsing to complete or context to cancel
	select {
	case err := <-parseDone:
		if err != nil {
			slog.Error("Failed to parse events", "error", err)
			return nil, fmt.Errorf("failed to parse events: %w", err)
		}
		slog.Debug("Finished parsing events", "text_length", textBuilder.Len(), "has_content", response.Content != "")
	case <-ctx.Done():
		slog.Warn("Context cancelled while parsing events", "error", ctx.Err())
		return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
	}
	slog.Debug("Waiting for process to exit")
	waitErr := pm.Wait()
	<-stderrDone

	// If we already got a parsed response (success or error), return it even if
	// the process exited with a non-zero status. Claude CLI exits with status 1
	// for API errors (e.g., 429 rate limit) but still produces valid stream-json output.
	if response.Content != "" {
		if waitErr != nil {
			slog.Warn("Claude process exited with error but response was parsed",
				"error", waitErr, "content_length", len(response.Content), "is_error", response.IsError)
		}
		// Mark session as started for future calls
		a.mu.Lock()
		a.sessionStarted = true
		a.mu.Unlock()
		slog.Debug("Returning parsed response", "content_length", len(response.Content), "is_error", response.IsError)
		return response, nil
	}

	if waitErr != nil {
		slog.Error("Claude process failed", "error", waitErr, "stderr_error", stderrErr, "stderr_output", string(stderrOutput))
		if stderrErr != nil {
			return nil, fmt.Errorf("claude process failed: %w (failed to read stderr: %v)", waitErr, stderrErr)
		}
		stderrText := strings.TrimSpace(string(stderrOutput))
		if stderrText != "" {
			return nil, fmt.Errorf("claude process failed: %w: %s", waitErr, stderrText)
		}
		return nil, fmt.Errorf("claude process failed: %w", waitErr)
	}

	// Mark session as started for future calls
	a.mu.Lock()
	a.sessionStarted = true
	a.mu.Unlock()

	slog.Debug("Process exited successfully", "content_length", len(response.Content))
	return response, nil
}

// convertToAgentEvent converts a claudecode.StreamEvent to agent.StreamEvent
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
