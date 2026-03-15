package claudecode

import (
	"context"
	"fmt"
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

	// Create process manager
	pmConfig := &ProcessConfig{
		SessionID:      a.sessionID,
		WorkingDir:     a.config.WorkingDir,
		PermissionMode: a.permissionMode,
		Resume:         false,
		ClaudePath:     a.config.ClaudePath,
		AllowedTools:   a.config.AllowedTools,
		Env:            a.config.Env,
	}

	a.pm = NewProcessManager(pmConfig)

	if err := a.pm.Start(ctx); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

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
	pmConfig := &ProcessConfig{
		SessionID:      a.sessionID,
		WorkingDir:     a.config.WorkingDir,
		PermissionMode: a.permissionMode,
		Resume:         true, // Always resume to maintain context
		ClaudePath:     a.config.ClaudePath,
		AllowedTools:   a.config.AllowedTools,
		Env:            a.config.Env,
		Message:        content,
	}

	pm := NewProcessManager(pmConfig)
	if err := pm.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}
	defer pm.Stop()

	// Collect response
	response := &agent.Response{}
	var textBuilder strings.Builder

	// Parse events from stdout
	err := ParseFromReader(pm.Stdout(), func(event *StreamEvent) error {
		// Convert to agent.StreamEvent and call handler
		agentEvent := convertToAgentEvent(event)
		if handler != nil {
			handler(agentEvent)
		}

		// Handle different event types
		switch {
		case event.IsSystemInit():
			// Extract metadata if needed
		case event.IsAssistantText():
			textBuilder.WriteString(event.GetText())
		case event.IsResultSuccess():
			response.Content = event.Result
			if textBuilder.Len() > 0 && response.Content == "" {
				response.Content = textBuilder.String()
			}
			response.CostUSD = event.TotalCostUSD
			response.Duration = time.Duration(event.DurationMs) * time.Millisecond
		case event.IsResultError():
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

	if err != nil {
		return nil, fmt.Errorf("failed to parse events: %w", err)
	}

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
