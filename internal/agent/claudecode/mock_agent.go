package claudecode

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/smy-101/cc-connect/internal/agent"
)

// MockAgent is a mock implementation of Agent for testing
type MockAgent struct {
	config         *Config
	sessionID      string
	permissionMode agent.PermissionMode
	status         agent.AgentStatus
	mu             sync.RWMutex

	// Mock responses
	response          *agent.Response
	err               error
	permissionDenied  bool
	deniedTools       []agent.DeniedTool
	streamEvents      []agent.StreamEvent

	// Call recording
	sendMessageCalls []string
}

// NewMockAgent creates a new MockAgent
func NewMockAgent(config *Config) *MockAgent {
	sessionID := config.SessionID
	if sessionID == "" {
		sessionID = "mock-session-" + time.Now().Format("20060102150405")
	}

	permMode := config.PermissionMode
	if permMode == "" {
		permMode = agent.PermissionModeDefault
	}

	return &MockAgent{
		config:         config,
		sessionID:      sessionID,
		permissionMode: permMode,
		status:         agent.AgentStatusIdle,
		sendMessageCalls: make([]string, 0),
	}
}

// SessionID returns the session ID
func (m *MockAgent) SessionID() string {
	return m.sessionID
}

// CurrentMode returns the current permission mode
func (m *MockAgent) CurrentMode() agent.PermissionMode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.permissionMode
}

// Status returns the current agent status
func (m *MockAgent) Status() agent.AgentStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

// Start starts the mock agent
func (m *MockAgent) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status == agent.AgentStatusRunning {
		return agent.ErrAgentAlreadyRunning
	}

	m.status = agent.AgentStatusRunning
	return nil
}

// Stop stops the mock agent
func (m *MockAgent) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status != agent.AgentStatusRunning {
		return nil
	}

	m.status = agent.AgentStatusStopped
	return nil
}

// Restart restarts the mock agent
func (m *MockAgent) Restart(ctx context.Context) error {
	m.mu.Lock()
	if m.status != agent.AgentStatusRunning {
		m.mu.Unlock()
		return agent.ErrAgentNotRunning
	}
	m.mu.Unlock()

	// Stop and start
	if err := m.Stop(); err != nil {
		return err
	}
	return m.Start(ctx)
}

// SetPermissionMode changes the permission mode
func (m *MockAgent) SetPermissionMode(mode agent.PermissionMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.permissionMode = mode
	return nil
}

// SendMessage sends a message and returns the mock response
func (m *MockAgent) SendMessage(ctx context.Context, content string, handler agent.EventHandler) (*agent.Response, error) {
	m.mu.Lock()
	m.sendMessageCalls = append(m.sendMessageCalls, content)
	m.mu.Unlock()

	// Check for error
	if m.err != nil {
		return nil, m.err
	}

	// Stream events if handler is provided
	if handler != nil && len(m.streamEvents) > 0 {
		for _, event := range m.streamEvents {
			handler(event)
		}
	}

	// Return permission denied response if set
	if m.permissionDenied {
		return &agent.Response{
			IsError:          true,
			PermissionDenied: true,
			DeniedTools:      m.deniedTools,
			Content:          "Permission denied",
		}, nil
	}

	// Return configured response or default
	if m.response != nil {
		return m.response, nil
	}

	return &agent.Response{
		Content: "mock response",
	}, nil
}

// SetResponse sets the response to return from SendMessage
func (m *MockAgent) SetResponse(resp *agent.Response) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.response = resp
}

// SetError sets an error to return from SendMessage
func (m *MockAgent) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// SetPermissionDenied configures the mock to return a permission denied response
func (m *MockAgent) SetPermissionDenied(tools []agent.DeniedTool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.permissionDenied = true
	m.deniedTools = tools
}

// SetStreamEvents sets the streaming events to send via handler
func (m *MockAgent) SetStreamEvents(events []agent.StreamEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.streamEvents = events
}

// GetSendMessageCalls returns the list of messages sent via SendMessage
func (m *MockAgent) GetSendMessageCalls() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]string, len(m.sendMessageCalls))
	copy(result, m.sendMessageCalls)
	return result
}

// Reset clears all mock state
func (m *MockAgent) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.response = nil
	m.err = nil
	m.permissionDenied = false
	m.deniedTools = nil
	m.streamEvents = nil
	m.sendMessageCalls = make([]string, 0)
}

// Assert that MockAgent implements Agent interface
var _ agent.Agent = (*MockAgent)(nil)

// RespondPermission implements agent.Agent.RespondPermission
func (m *MockAgent) RespondPermission(requestID, behavior string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// For testing purposes, just record the call
	return nil
}

// MockAgentError is a helper to create a mock error
var MockAgentError = errors.New("mock agent error")
