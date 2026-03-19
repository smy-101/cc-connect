package command

import (
	"context"
	"testing"

	"github.com/smy-101/cc-connect/internal/agent/claudecode"
	"github.com/smy-101/cc-connect/internal/core"
)

// mockPermissionAgent wraps MockAgent to track permission responses
type mockPermissionAgent struct {
	*claudecode.MockAgent
	respondedRequestID string
	respondedBehavior  string
	respondErr         error
}

func newMockPermissionAgent(t *testing.T) *mockPermissionAgent {
	t.Helper()
	cfg := &claudecode.Config{
		WorkingDir: t.TempDir(),
	}
	return &mockPermissionAgent{MockAgent: claudecode.NewMockAgent(cfg)}
}

func (m *mockPermissionAgent) RespondPermission(requestID, behavior string) error {
	m.respondedRequestID = requestID
	m.respondedBehavior = behavior
	return m.respondErr
}

func TestHandleAllow(t *testing.T) {
	mockAgent := newMockPermissionAgent(t)
	executor := NewExecutor(mockAgent, core.NewSessionManager(core.DefaultSessionConfig()))

	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name: "allow with request ID",
			args: []string{"req123"},
		},
		{
			name:        "allow without request ID",
			args:        []string{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := Command{Name: "allow", Args: tt.args}
			msg := &core.Message{ChannelID: "test_chat"}

			result := executor.Execute(context.Background(), cmd, msg)

			if tt.expectError {
				if result.Error == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if result.Error != nil {
				t.Errorf("unexpected error: %v", result.Error)
				return
			}

			if mockAgent.respondedRequestID != tt.args[0] {
				t.Errorf("expected requestID %s, got %s", tt.args[0], mockAgent.respondedRequestID)
			}
			if mockAgent.respondedBehavior != "allow" {
				t.Errorf("expected behavior 'allow', got %s", mockAgent.respondedBehavior)
			}
		})
	}
}

func TestHandleDeny(t *testing.T) {
	mockAgent := newMockPermissionAgent(t)
	executor := NewExecutor(mockAgent, core.NewSessionManager(core.DefaultSessionConfig()))

	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name: "deny with request ID",
			args: []string{"req456"},
		},
		{
			name:        "deny without request ID",
			args:        []string{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := Command{Name: "deny", Args: tt.args}
			msg := &core.Message{ChannelID: "test_chat"}

			result := executor.Execute(context.Background(), cmd, msg)

			if tt.expectError {
				if result.Error == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if result.Error != nil {
				t.Errorf("unexpected error: %v", result.Error)
				return
			}

			if mockAgent.respondedRequestID != tt.args[0] {
				t.Errorf("expected requestID %s, got %s", tt.args[0], mockAgent.respondedRequestID)
			}
			if mockAgent.respondedBehavior != "deny" {
				t.Errorf("expected behavior 'deny', got %s", mockAgent.respondedBehavior)
			}
		})
	}
}

func TestHandleAnswer(t *testing.T) {
	mockAgent := newMockPermissionAgent(t)
	executor := NewExecutor(mockAgent, core.NewSessionManager(core.DefaultSessionConfig()))

	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name: "answer with request ID and value",
			args: []string{"req789", "PostgreSQL"},
		},
		{
			name: "answer with multi-word value",
			args: []string{"req789", "Use", "Docker", "for", "development"},
		},
		{
			name:        "answer without request ID",
			args:        []string{},
			expectError: true,
		},
		{
			name:        "answer without value",
			args:        []string{"req789"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := Command{Name: "answer", Args: tt.args}
			msg := &core.Message{ChannelID: "test_chat"}

			result := executor.Execute(context.Background(), cmd, msg)

			if tt.expectError {
				if result.Error == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if result.Error != nil {
				t.Errorf("unexpected error: %v", result.Error)
				return
			}

			if mockAgent.respondedRequestID != tt.args[0] {
				t.Errorf("expected requestID %s, got %s", tt.args[0], mockAgent.respondedRequestID)
			}
		})
	}
}
