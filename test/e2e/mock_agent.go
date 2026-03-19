package e2e

import (
	"context"
	"sync"
	"time"

	"github.com/smy-101/cc-connect/internal/agent"
)

// MockAgent is a mock implementation of agent.Agent for testing.
type MockAgent struct {
	mu                   sync.RWMutex
	responses            map[string]string
	delay                time.Duration
	err                  error
	status               agent.AgentStatus
	permMode             agent.PermissionMode
	sessionID            string
	sendCount            int
	lastContent          string
	pendingRequestID     string
	pendingToolName      string
	pendingToolInput     map[string]any
	lastPermRequestID    string
	lastPermBehavior     string
	busy                 bool
}

// NewMockAgent creates a new MockAgent with default settings.
func NewMockAgent() *MockAgent {
	return &MockAgent{
		responses: make(map[string]string),
		status:    agent.AgentStatusIdle,
		permMode:  agent.PermissionModeDefault,
		sessionID: "mock-session-id",
	}
}

// SetResponse sets a predefined response for a specific input.
func (m *MockAgent) SetResponse(input, response string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[input] = response
}

// SetDefaultResponse sets a default response for any input.
func (m *MockAgent) SetDefaultResponse(response string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses["*"] = response
}

// SetDelay sets the delay before responding.
func (m *MockAgent) SetDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delay = delay
}

// SetError sets an error to be returned by SendMessage.
func (m *MockAgent) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// SendMessage implements agent.Agent.
func (m *MockAgent) SendMessage(ctx context.Context, content string, handler agent.EventHandler) (*agent.Response, error) {
	m.mu.Lock()
	m.sendCount++
	m.lastContent = content
	delay := m.delay
	err := m.err
	responses := m.responses
	m.mu.Unlock()

	// Check for error
	if err != nil {
		return nil, err
	}

	// Apply delay
	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Find response
	response := "Default response: " + content
	if resp, ok := responses[content]; ok {
		response = resp
	} else if resp, ok := responses["*"]; ok {
		response = resp
	}

	return &agent.Response{
		Content:  response,
		IsError:  false,
		Duration: delay,
	}, nil
}

// SetPermissionMode implements agent.Agent.
func (m *MockAgent) SetPermissionMode(mode agent.PermissionMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.permMode = mode
	return nil
}

// CurrentMode implements agent.Agent.
func (m *MockAgent) CurrentMode() agent.PermissionMode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.permMode
}

// SessionID implements agent.Agent.
func (m *MockAgent) SessionID() string {
	return m.sessionID
}

// Start implements agent.Agent.
func (m *MockAgent) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = agent.AgentStatusRunning
	return nil
}

// Stop implements agent.Agent.
func (m *MockAgent) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = agent.AgentStatusStopped
	return nil
}

// Status implements agent.Agent.
func (m *MockAgent) Status() agent.AgentStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

// Restart implements agent.Agent.
func (m *MockAgent) Restart(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = agent.AgentStatusRunning
	return nil
}

// SendCount returns the number of times SendMessage was called.
func (m *MockAgent) SendCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sendCount
}

// LastContent returns the last content sent to SendMessage.
func (m *MockAgent) LastContent() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastContent
}

// RespondPermission implements agent.Agent.
func (m *MockAgent) RespondPermission(requestID, behavior string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Record the call regardless
	m.lastPermRequestID = requestID
	m.lastPermBehavior = behavior

	// Check if there's a pending request with matching ID
	if m.pendingRequestID == "" || m.pendingRequestID != requestID {
		return nil // Return nil - the executor layer handles validation
	}

	// Clear pending state on success
	m.pendingRequestID = ""
	m.busy = false

	return nil
}

// HasPendingPermission returns true if there's a pending permission request.
func (m *MockAgent) HasPendingPermission() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pendingRequestID != ""
}

// SetPendingPermission sets up a pending permission request for testing.
func (m *MockAgent) SetPendingPermission(requestID, toolName string, toolInput map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pendingRequestID = requestID
	m.pendingToolName = toolName
	m.pendingToolInput = toolInput
	m.busy = true
}

// LastPermissionBehavior returns the last behavior used in RespondPermission.
func (m *MockAgent) LastPermissionBehavior() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastPermBehavior
}

// LastPermissionRequestID returns the last request ID used in RespondPermission.
func (m *MockAgent) LastPermissionRequestID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastPermRequestID
}

// IsBusy returns whether the agent is busy waiting for a permission response.
func (m *MockAgent) IsBusy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.busy
}
