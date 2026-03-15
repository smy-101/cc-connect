package claudecode

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/smy-101/cc-connect/internal/agent"
)

// TestMockAgentBasic tests basic MockAgent functionality
func TestMockAgentBasic(t *testing.T) {
	mock := NewMockAgent(&Config{
		SessionID:      "mock-session-1",
		WorkingDir:     "/tmp/test",
		PermissionMode: agent.PermissionModeDefault,
	})

	// Check session ID
	if mock.SessionID() != "mock-session-1" {
		t.Errorf("SessionID = %v, want mock-session-1", mock.SessionID())
	}

	// Check initial status
	if mock.Status() != agent.AgentStatusIdle {
		t.Errorf("Status = %v, want idle", mock.Status())
	}

	// Check initial mode
	if mock.CurrentMode() != agent.PermissionModeDefault {
		t.Errorf("CurrentMode = %v, want default", mock.CurrentMode())
	}
}

// TestMockAgentStartStop tests MockAgent start/stop
func TestMockAgentStartStop(t *testing.T) {
	mock := NewMockAgent(&Config{
		SessionID:      "mock-session-start-stop",
		WorkingDir:     "/tmp/test",
		PermissionMode: agent.PermissionModeDefault,
	})

	ctx := context.Background()

	// Start
	if err := mock.Start(ctx); err != nil {
		t.Errorf("Start() error: %v", err)
	}

	if mock.Status() != agent.AgentStatusRunning {
		t.Errorf("Status = %v, want running", mock.Status())
	}

	// Stop
	if err := mock.Stop(); err != nil {
		t.Errorf("Stop() error: %v", err)
	}

	if mock.Status() != agent.AgentStatusStopped {
		t.Errorf("Status = %v, want stopped", mock.Status())
	}
}

// TestMockAgentSendMessage tests MockAgent SendMessage
func TestMockAgentSendMessage(t *testing.T) {
	mock := NewMockAgent(&Config{
		SessionID:      "mock-session-send",
		WorkingDir:     "/tmp/test",
		PermissionMode: agent.PermissionModeDefault,
	})

	ctx := context.Background()
	_ = mock.Start(ctx)
	defer mock.Stop()

	// Set a response
	mock.SetResponse(&agent.Response{
		Content:  "Hello, world!",
		CostUSD:  0.001,
		Duration: 100 * time.Millisecond,
	})

	// Send message
	resp, err := mock.SendMessage(ctx, "test message", nil)
	if err != nil {
		t.Errorf("SendMessage() error: %v", err)
	}

	if resp.Content != "Hello, world!" {
		t.Errorf("Content = %v, want Hello, world!", resp.Content)
	}
}

// TestMockAgentSetError tests MockAgent error simulation
func TestMockAgentSetError(t *testing.T) {
	mock := NewMockAgent(&Config{
		SessionID:      "mock-session-error",
		WorkingDir:     "/tmp/test",
		PermissionMode: agent.PermissionModeDefault,
	})

	ctx := context.Background()
	_ = mock.Start(ctx)
	defer mock.Stop()

	// Set an error
	testErr := errors.New("simulated error")
	mock.SetError(testErr)

	// Send message should return error
	_, err := mock.SendMessage(ctx, "test message", nil)
	if err != testErr {
		t.Errorf("SendMessage() error = %v, want %v", err, testErr)
	}
}

// TestMockAgentSetPermissionDenied tests MockAgent permission denied simulation
func TestMockAgentSetPermissionDenied(t *testing.T) {
	mock := NewMockAgent(&Config{
		SessionID:      "mock-session-perm",
		WorkingDir:     "/tmp/test",
		PermissionMode: agent.PermissionModeDefault,
	})

	ctx := context.Background()
	_ = mock.Start(ctx)
	defer mock.Stop()

	// Set permission denied
	deniedTools := []agent.DeniedTool{
		{
			ToolName:  "Bash",
			ToolUseID: "toolu_1",
			ToolInput: map[string]interface{}{"command": "rm -rf /"},
		},
	}
	mock.SetPermissionDenied(deniedTools)

	// Send message
	resp, err := mock.SendMessage(ctx, "test message", nil)
	if err != nil {
		t.Errorf("SendMessage() error: %v", err)
	}

	if !resp.PermissionDenied {
		t.Error("PermissionDenied should be true")
	}

	if len(resp.DeniedTools) != 1 {
		t.Errorf("len(DeniedTools) = %v, want 1", len(resp.DeniedTools))
	}

	if resp.DeniedTools[0].ToolName != "Bash" {
		t.Errorf("ToolName = %v, want Bash", resp.DeniedTools[0].ToolName)
	}
}

// TestMockAgentSetPermissionMode tests MockAgent permission mode setting
func TestMockAgentSetPermissionMode(t *testing.T) {
	mock := NewMockAgent(&Config{
		SessionID:      "mock-session-mode",
		WorkingDir:     "/tmp/test",
		PermissionMode: agent.PermissionModeDefault,
	})

	ctx := context.Background()
	_ = mock.Start(ctx)
	defer mock.Stop()

	// Change mode
	if err := mock.SetPermissionMode(agent.PermissionModeBypassPermissions); err != nil {
		t.Errorf("SetPermissionMode() error: %v", err)
	}

	if mock.CurrentMode() != agent.PermissionModeBypassPermissions {
		t.Errorf("CurrentMode = %v, want bypassPermissions", mock.CurrentMode())
	}
}

// TestMockAgentStreaming tests MockAgent streaming events
func TestMockAgentStreaming(t *testing.T) {
	mock := NewMockAgent(&Config{
		SessionID:      "mock-session-stream",
		WorkingDir:     "/tmp/test",
		PermissionMode: agent.PermissionModeDefault,
	})

	ctx := context.Background()
	_ = mock.Start(ctx)
	defer mock.Stop()

	// Set streaming events
	events := []agent.StreamEvent{
		{Type: agent.StreamEventTypeText, Content: "Hello"},
		{Type: agent.StreamEventTypeText, Content: " world"},
		{Type: agent.StreamEventTypeResult, Content: "Done!"},
	}
	mock.SetStreamEvents(events)

	// Collect events
	var receivedEvents []agent.StreamEvent
	handler := func(event agent.StreamEvent) {
		receivedEvents = append(receivedEvents, event)
	}

	// Send message
	_, err := mock.SendMessage(ctx, "test message", handler)
	if err != nil {
		t.Errorf("SendMessage() error: %v", err)
	}

	// Check events
	if len(receivedEvents) != 3 {
		t.Errorf("len(receivedEvents) = %v, want 3", len(receivedEvents))
	}
}

// TestMockAgentRecordedCalls tests that calls are recorded
func TestMockAgentRecordedCalls(t *testing.T) {
	mock := NewMockAgent(&Config{
		SessionID:      "mock-session-record",
		WorkingDir:     "/tmp/test",
		PermissionMode: agent.PermissionModeDefault,
	})

	ctx := context.Background()
	_ = mock.Start(ctx)
	defer mock.Stop()

	// Send multiple messages
	_, _ = mock.SendMessage(ctx, "message 1", nil)
	_, _ = mock.SendMessage(ctx, "message 2", nil)
	_, _ = mock.SendMessage(ctx, "message 3", nil)

	// Check recorded calls
	calls := mock.GetSendMessageCalls()
	if len(calls) != 3 {
		t.Errorf("len(calls) = %v, want 3", len(calls))
	}

	if calls[0] != "message 1" {
		t.Errorf("calls[0] = %v, want message 1", calls[0])
	}
}

// TestMockAgentRestart tests MockAgent restart
func TestMockAgentRestart(t *testing.T) {
	mock := NewMockAgent(&Config{
		SessionID:      "mock-session-restart",
		WorkingDir:     "/tmp/test",
		PermissionMode: agent.PermissionModeDefault,
	})

	ctx := context.Background()

	// Restart without starting should fail
	err := mock.Restart(ctx)
	if err != agent.ErrAgentNotRunning {
		t.Errorf("expected ErrAgentNotRunning, got %v", err)
	}

	// Start
	if err := mock.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	if mock.Status() != agent.AgentStatusRunning {
		t.Errorf("Status = %v, want running", mock.Status())
	}

	// Restart
	if err := mock.Restart(ctx); err != nil {
		t.Errorf("Restart() error: %v", err)
	}

	// Should still be running
	if mock.Status() != agent.AgentStatusRunning {
		t.Errorf("Status after restart = %v, want running", mock.Status())
	}
}

// TestMockAgentReset tests MockAgent reset
func TestMockAgentReset(t *testing.T) {
	mock := NewMockAgent(&Config{
		SessionID:      "mock-session-reset",
		WorkingDir:     "/tmp/test",
		PermissionMode: agent.PermissionModeDefault,
	})

	ctx := context.Background()
	_ = mock.Start(ctx)

	// Set some state
	mock.SetError(errors.New("test error"))
	mock.SetPermissionDenied([]agent.DeniedTool{{ToolName: "Bash"}})
	mock.SetResponse(&agent.Response{Content: "test"})
	_, _ = mock.SendMessage(ctx, "test", nil)

	// Reset
	mock.Reset()

	// Verify state is cleared
	calls := mock.GetSendMessageCalls()
	if len(calls) != 0 {
		t.Errorf("len(calls) after reset = %v, want 0", len(calls))
	}

	// Should return default response after reset
	resp, err := mock.SendMessage(ctx, "new message", nil)
	if err != nil {
		t.Errorf("SendMessage() error after reset: %v", err)
	}
	if resp.Content != "mock response" {
		t.Errorf("Content after reset = %v, want 'mock response'", resp.Content)
	}
}

// TestMockAgentAutoGenerateSessionID tests auto-generated session ID
func TestMockAgentAutoGenerateSessionID(t *testing.T) {
	mock := NewMockAgent(&Config{
		WorkingDir:     "/tmp/test",
		PermissionMode: agent.PermissionModeDefault,
	})

	if mock.SessionID() == "" {
		t.Error("SessionID should be auto-generated")
	}

	if mock.CurrentMode() != agent.PermissionModeDefault {
		t.Errorf("CurrentMode = %v, want default", mock.CurrentMode())
	}
}

// TestMockAgentStartAlreadyRunning tests starting an already running agent
func TestMockAgentStartAlreadyRunning(t *testing.T) {
	mock := NewMockAgent(&Config{
		SessionID:      "mock-session-already-running",
		WorkingDir:     "/tmp/test",
		PermissionMode: agent.PermissionModeDefault,
	})

	ctx := context.Background()
	if err := mock.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	// Start again should fail
	err := mock.Start(ctx)
	if err != agent.ErrAgentAlreadyRunning {
		t.Errorf("expected ErrAgentAlreadyRunning, got %v", err)
	}
}

// TestMockAgentStopNotRunning tests stopping a non-running agent
func TestMockAgentStopNotRunning(t *testing.T) {
	mock := NewMockAgent(&Config{
		SessionID:      "mock-session-stop-not-running",
		WorkingDir:     "/tmp/test",
		PermissionMode: agent.PermissionModeDefault,
	})

	// Stop without starting should succeed
	if err := mock.Stop(); err != nil {
		t.Errorf("Stop() error: %v", err)
	}
}
