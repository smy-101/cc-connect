package agent

import (
	"context"
	"testing"
)

// MockAgent implements Agent interface for testing
type MockAgent struct {
	sessionID      string
	permissionMode PermissionMode
	status         AgentStatus
	busy           bool
}

func NewMockAgent(sessionID string) *MockAgent {
	return &MockAgent{
		sessionID:      sessionID,
		permissionMode: PermissionModeDefault,
		status:         AgentStatusIdle,
	}
}

func (m *MockAgent) SendMessage(ctx context.Context, content string, handler EventHandler) (*Response, error) {
	return &Response{Content: "mock response"}, nil
}

func (m *MockAgent) SetPermissionMode(mode PermissionMode) error {
	m.permissionMode = mode
	return nil
}

func (m *MockAgent) CurrentMode() PermissionMode {
	return m.permissionMode
}

func (m *MockAgent) SessionID() string {
	return m.sessionID
}

func (m *MockAgent) Start(ctx context.Context) error {
	m.status = AgentStatusRunning
	return nil
}

func (m *MockAgent) Stop() error {
	m.status = AgentStatusStopped
	return nil
}

func (m *MockAgent) Status() AgentStatus {
	return m.status
}

func (m *MockAgent) Restart(ctx context.Context) error {
	return nil
}

func (m *MockAgent) RespondPermission(requestID, behavior string) error {
	return nil
}

// TestAgentManagerGetOrCreate tests GetOrCreate functionality
func TestAgentManagerGetOrCreate(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()

	// Create a new agent
	agent1 := NewMockAgent("session-1")
	ag1, err := manager.GetOrCreate("session-1", func() (Agent, error) {
		return agent1, nil
	})
	if err != nil {
		t.Fatalf("GetOrCreate() error: %v", err)
	}

	// Start the agent
	if err := ag1.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	// Get the same agent again (should return the same instance)
	ag2, err := manager.GetOrCreate("session-1", func() (Agent, error) {
		return NewMockAgent("session-1-new"), nil
	})
	if err != nil {
		t.Fatalf("GetOrCreate() second call error: %v", err)
	}

	if ag1 != ag2 {
		t.Error("GetOrCreate should return the same agent for the same session")
	}

	// Create another agent for a different session
	ag3, err := manager.GetOrCreate("session-2", func() (Agent, error) {
		return NewMockAgent("session-2"), nil
	})
	if err != nil {
		t.Fatalf("GetOrCreate() for session-2 error: %v", err)
	}

	if ag1 == ag3 {
		t.Error("GetOrCreate should return different agents for different sessions")
	}
}

// TestAgentManagerRemove tests Remove functionality
func TestAgentManagerRemove(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()

	// Create and start an agent
	agent1 := NewMockAgent("session-remove")
	ag, err := manager.GetOrCreate("session-remove", func() (Agent, error) {
		return agent1, nil
	})
	if err != nil {
		t.Fatalf("GetOrCreate() error: %v", err)
	}

	if err := ag.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	// Remove the agent
	if err := manager.Remove("session-remove"); err != nil {
		t.Fatalf("Remove() error: %v", err)
	}

	// GetOrCreate should create a new agent now
	agent2 := NewMockAgent("session-remove-new")
	ag2, err := manager.GetOrCreate("session-remove", func() (Agent, error) {
		return agent2, nil
	})
	if err != nil {
		t.Fatalf("GetOrCreate() after remove error: %v", err)
	}

	if ag == ag2 {
		t.Error("GetOrCreate after Remove should return a new agent")
	}
}

// TestAgentManagerGet tests Get functionality
func TestAgentManagerGet(t *testing.T) {
	manager := NewManager()

	// Get non-existent agent should return nil
	ag := manager.Get("non-existent")
	if ag != nil {
		t.Error("Get() for non-existent session should return nil")
	}

	// Create an agent
	agent1 := NewMockAgent("session-get")
	_, err := manager.GetOrCreate("session-get", func() (Agent, error) {
		return agent1, nil
	})
	if err != nil {
		t.Fatalf("GetOrCreate() error: %v", err)
	}

	// Get should return the agent
	ag = manager.Get("session-get")
	if ag == nil {
		t.Error("Get() should return the agent")
	}
}

// TestAgentManagerList tests List functionality
func TestAgentManagerList(t *testing.T) {
	manager := NewManager()

	// Create multiple agents
	_, _ = manager.GetOrCreate("session-list-1", func() (Agent, error) {
		return NewMockAgent("session-list-1"), nil
	})
	_, _ = manager.GetOrCreate("session-list-2", func() (Agent, error) {
		return NewMockAgent("session-list-2"), nil
	})

	// List should return all session IDs
	sessions := manager.List()
	if len(sessions) != 2 {
		t.Errorf("List() returned %d sessions, want 2", len(sessions))
	}

	// Check that both sessions are in the list
	found1, found2 := false, false
	for _, s := range sessions {
		if s == "session-list-1" {
			found1 = true
		}
		if s == "session-list-2" {
			found2 = true
		}
	}
	if !found1 || !found2 {
		t.Error("List() should contain both sessions")
	}
}

// TestAgentManagerCount tests Count functionality
func TestAgentManagerCount(t *testing.T) {
	manager := NewManager()

	// Initially should be 0
	if manager.Count() != 0 {
		t.Errorf("Count() = %v, want 0", manager.Count())
	}

	// Add agents
	_, _ = manager.GetOrCreate("session-count-1", func() (Agent, error) {
		return NewMockAgent("session-count-1"), nil
	})
	if manager.Count() != 1 {
		t.Errorf("Count() = %v, want 1", manager.Count())
	}

	_, _ = manager.GetOrCreate("session-count-2", func() (Agent, error) {
		return NewMockAgent("session-count-2"), nil
	})
	if manager.Count() != 2 {
		t.Errorf("Count() = %v, want 2", manager.Count())
	}
}

// TestAgentManagerStopAll tests StopAll functionality
func TestAgentManagerStopAll(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()

	// Create and start multiple agents
	ag1, _ := manager.GetOrCreate("session-stopall-1", func() (Agent, error) {
		return NewMockAgent("session-stopall-1"), nil
	})
	ag1.Start(ctx)

	ag2, _ := manager.GetOrCreate("session-stopall-2", func() (Agent, error) {
		return NewMockAgent("session-stopall-2"), nil
	})
	ag2.Start(ctx)

	// Stop all
	if err := manager.StopAll(); err != nil {
		t.Errorf("StopAll() error: %v", err)
	}

	// Count should be 0
	if manager.Count() != 0 {
		t.Errorf("Count() after StopAll = %v, want 0", manager.Count())
	}
}

// TestAgentManagerRemoveNonExistent tests Remove on non-existent session
func TestAgentManagerRemoveNonExistent(t *testing.T) {
	manager := NewManager()

	// Remove non-existent should return nil
	if err := manager.Remove("non-existent"); err != nil {
		t.Errorf("Remove() for non-existent session should return nil, got %v", err)
	}
}
