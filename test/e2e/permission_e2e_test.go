package e2e

import (
	"testing"

	"github.com/smy-101/cc-connect/internal/core"
	"github.com/smy-101/cc-connect/internal/core/command"
)

// TestE2EPermissionAllow tests the permission allow flow.
func TestE2EPermissionAllow(t *testing.T) {
	// Create mock agent with permission support
	mockAgent := NewMockAgent()

	// Create executor with mock agent
	executor := command.NewExecutor(mockAgent, core.NewSessionManager(core.DefaultSessionConfig()))

	// Simulate permission request
	mockAgent.SetPendingPermission("req-123", "Bash", map[string]any{"command": "ls -la"})

	// Execute /allow command
	cmd := command.Command{Name: "allow", Args: []string{"req-123"}}
	msg := &core.Message{ChannelID: "test_chat", UserID: "user123"}

	result := executor.Execute(nil, cmd, msg)
	if result.Error != nil {
		t.Fatalf("allow command failed: %v", result.Error)
	}

	// Verify permission was allowed
	if mockAgent.LastPermissionBehavior() != "allow" {
		t.Errorf("expected behavior 'allow', got %q", mockAgent.LastPermissionBehavior())
	}
}

// TestE2EPermissionDeny tests the permission deny flow.
func TestE2EPermissionDeny(t *testing.T) {
	// Create mock agent with permission support
	mockAgent := NewMockAgent()

	// Create executor with mock agent
	executor := command.NewExecutor(mockAgent, core.NewSessionManager(core.DefaultSessionConfig()))

	// Simulate permission request
	mockAgent.SetPendingPermission("req-456", "Bash", map[string]any{"command": "rm -rf"})

	// Execute /deny command
	cmd := command.Command{Name: "deny", Args: []string{"req-456"}}
	msg := &core.Message{ChannelID: "test_chat", UserID: "user123"}

	result := executor.Execute(nil, cmd, msg)
	if result.Error != nil {
		t.Fatalf("deny command failed: %v", result.Error)
	}

	// Verify permission was denied
	if mockAgent.LastPermissionBehavior() != "deny" {
		t.Errorf("expected behavior 'deny', got %q", mockAgent.LastPermissionBehavior())
	}
}

// TestE2EPermissionCardCallbackAllow tests permission allow via card callback.
func TestE2EPermissionCardCallbackAllow(t *testing.T) {
	// Create mock agent
	mockAgent := NewMockAgent()

	// Create executor
	executor := command.NewExecutor(mockAgent, core.NewSessionManager(core.DefaultSessionConfig()))

	// Simulate permission request
	mockAgent.SetPendingPermission("req-789", "Bash", map[string]any{"command": "npm test"})

	// Execute /allow command (simulating card callback that was parsed to command)
	cmd := command.Command{Name: "allow", Args: []string{"req-789"}}
	msg := &core.Message{
		Type:      core.MessageTypeCommand,
		Content:   "/allow req-789",
		ChannelID: "test_chat",
		UserID:    "user123",
	}

	result := executor.Execute(nil, cmd, msg)
	if result.Error != nil {
		t.Fatalf("command failed: %v", result.Error)
	}

	// Verify permission was allowed
	if mockAgent.LastPermissionBehavior() != "allow" {
		t.Errorf("expected behavior 'allow', got %q", mockAgent.LastPermissionBehavior())
	}
	if mockAgent.LastPermissionRequestID() != "req-789" {
		t.Errorf("expected requestID 'req-789', got %q", mockAgent.LastPermissionRequestID())
	}
}

// TestE2EPermissionTimeout tests permission timeout handling.
func TestE2EPermissionTimeout(t *testing.T) {
	// Create mock agent
	mockAgent := NewMockAgent()

	// Create executor
	executor := command.NewExecutor(mockAgent, core.NewSessionManager(core.DefaultSessionConfig()))

	// Try to respond to non-existent request (simulates timeout)
	// Note: The mock doesn't validate request IDs - this tests the executor flow
	cmd := command.Command{Name: "allow", Args: []string{"non-existent-req"}}
	msg := &core.Message{ChannelID: "test_chat", UserID: "user123"}

	result := executor.Execute(nil, cmd, msg)
	// The executor calls the agent's RespondPermission, which succeeds on mock
	// Real implementation would return error for non-existent request
	_ = result // Just verify no panic
}

// TestE2EPermissionBusyState tests that agent is busy during permission wait.
func TestE2EPermissionBusyState(t *testing.T) {
	// Create mock agent
	mockAgent := NewMockAgent()

	// Set pending permission (simulates waiting state)
	mockAgent.SetPendingPermission("req-busy", "Bash", map[string]any{"command": "ls"})

	// Check busy state
	if !mockAgent.IsBusy() {
		t.Error("expected agent to be busy during permission wait")
	}

	// Respond to permission
	executor := command.NewExecutor(mockAgent, core.NewSessionManager(core.DefaultSessionConfig()))
	cmd := command.Command{Name: "allow", Args: []string{"req-busy"}}
	msg := &core.Message{ChannelID: "test_chat"}

	result := executor.Execute(nil, cmd, msg)
	if result.Error != nil {
		t.Fatalf("allow command failed: %v", result.Error)
	}

	// Agent should no longer be busy
	if mockAgent.IsBusy() {
		t.Error("expected agent to not be busy after permission resolved")
	}
}

// TestE2EAskUserQuestionAnswer tests AskUserQuestion answer flow.
func TestE2EAskUserQuestionAnswer(t *testing.T) {
	// Create mock agent
	mockAgent := NewMockAgent()

	// Create executor
	executor := command.NewExecutor(mockAgent, core.NewSessionManager(core.DefaultSessionConfig()))

	// Simulate AskUserQuestion request
	mockAgent.SetPendingPermission("req-question", "AskUserQuestion", map[string]any{
		"questions": []map[string]any{
			{
				"question": "Which database?",
				"header":   "Database",
				"options": []map[string]any{
					{"label": "PostgreSQL"},
					{"label": "MySQL"},
					{"label": "SQLite"},
				},
			},
		},
	})

	// Execute /answer command (simulates card button click)
	cmd := command.Command{Name: "answer", Args: []string{"req-question", "PostgreSQL"}}
	msg := &core.Message{ChannelID: "test_chat", UserID: "user123"}

	result := executor.Execute(nil, cmd, msg)
	if result.Error != nil {
		t.Fatalf("answer command failed: %v", result.Error)
	}

	// Verify the answer was recorded
	if mockAgent.LastPermissionRequestID() != "req-question" {
		t.Errorf("expected requestID 'req-question', got %q", mockAgent.LastPermissionRequestID())
	}

	behavior := mockAgent.LastPermissionBehavior()
	if behavior != "answer:PostgreSQL" {
		t.Errorf("expected behavior 'answer:PostgreSQL', got %q", behavior)
	}
}

// TestE2EAskUserQuestionMultiWordAnswer tests answer with multiple words.
func TestE2EAskUserQuestionMultiWordAnswer(t *testing.T) {
	// Create mock agent
	mockAgent := NewMockAgent()

	// Create executor
	executor := command.NewExecutor(mockAgent, core.NewSessionManager(core.DefaultSessionConfig()))

	// Simulate AskUserQuestion request
	mockAgent.SetPendingPermission("req-multi", "AskUserQuestion", map[string]any{
		"questions": []map[string]any{
			{
				"question": "Describe your project:",
			},
		},
	})

	// Execute /answer command with multi-word answer
	cmd := command.Command{Name: "answer", Args: []string{"req-multi", "A", "Go", "web", "application"}}
	msg := &core.Message{ChannelID: "test_chat", UserID: "user123"}

	result := executor.Execute(nil, cmd, msg)
	if result.Error != nil {
		t.Fatalf("answer command failed: %v", result.Error)
	}

	// Verify the full answer was recorded
	behavior := mockAgent.LastPermissionBehavior()
	expected := "answer:A Go web application"
	if behavior != expected {
		t.Errorf("expected behavior %q, got %q", expected, behavior)
	}
}

// TestE2EAskUserQuestionCardCallback tests AskUserQuestion via card callback parsing.
func TestE2EAskUserQuestionCardCallback(t *testing.T) {
	// This tests the flow: Card button -> ParseCardCallback -> /answer command -> RespondPermission

	// Create mock agent
	mockAgent := NewMockAgent()

	// Create executor
	executor := command.NewExecutor(mockAgent, core.NewSessionManager(core.DefaultSessionConfig()))

	// Simulate AskUserQuestion request
	mockAgent.SetPendingPermission("req-cb", "AskUserQuestion", map[string]any{
		"questions": []map[string]any{
			{
				"question": "Choose a framework:",
				"options": []map[string]any{
					{"label": "React"},
					{"label": "Vue"},
					{"label": "Svelte"},
				},
			},
		},
	})

	// Simulate card callback parsing result: "ans:req-cb:React" -> "/answer req-cb React"
	cmd := command.Command{Name: "answer", Args: []string{"req-cb", "React"}}
	msg := &core.Message{
		Type:      core.MessageTypeCommand,
		Content:   "/answer req-cb React",
		ChannelID: "test_chat",
		UserID:    "user123",
	}

	result := executor.Execute(nil, cmd, msg)
	if result.Error != nil {
		t.Fatalf("answer command failed: %v", result.Error)
	}

	// Verify
	if mockAgent.LastPermissionRequestID() != "req-cb" {
		t.Errorf("expected requestID 'req-cb', got %q", mockAgent.LastPermissionRequestID())
	}
	if mockAgent.LastPermissionBehavior() != "answer:React" {
		t.Errorf("expected behavior 'answer:React', got %q", mockAgent.LastPermissionBehavior())
	}
}
