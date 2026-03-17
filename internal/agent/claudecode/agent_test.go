package claudecode

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/smy-101/cc-connect/internal/agent"
)

// TestNewAgent tests agent creation
func TestNewAgent(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "with session ID",
			config: &Config{
				SessionID:      "test-session-123",
				WorkingDir:     "/tmp/test",
				PermissionMode: agent.PermissionModeDefault,
			},
			wantErr: false,
		},
		{
			name: "without session ID (auto-generate)",
			config: &Config{
				WorkingDir:     "/tmp/test",
				PermissionMode: agent.PermissionModeDefault,
			},
			wantErr: false,
		},
		{
			name: "empty working dir",
			config: &Config{
				SessionID:      "test-session-456",
				PermissionMode: agent.PermissionModeDefault,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ag, err := NewAgent(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check session ID was set
			if ag.SessionID() == "" {
				t.Error("SessionID should not be empty")
			}

			// If session ID was provided, it should be used
			if tt.config.SessionID != "" && ag.SessionID() != tt.config.SessionID {
				t.Errorf("SessionID = %v, want %v", ag.SessionID(), tt.config.SessionID)
			}
		})
	}
}

// TestAgentStartStop tests agent start/stop
func TestAgentStartStop(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &Config{
		SessionID:      "test-session-start-stop",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
	}

	ag, err := NewAgent(config)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	ctx := context.Background()

	// Start
	if err := ag.Start(ctx); err != nil {
		t.Errorf("Start() error: %v", err)
	}

	// Status should be running
	if ag.Status() != agent.AgentStatusRunning {
		t.Errorf("Status = %v, want %v", ag.Status(), agent.AgentStatusRunning)
	}

	// Start again should fail
	if err := ag.Start(ctx); err != agent.ErrAgentAlreadyRunning {
		t.Errorf("expected ErrAgentAlreadyRunning, got %v", err)
	}

	// Stop
	if err := ag.Stop(); err != nil {
		t.Errorf("Stop() error: %v", err)
	}

	// Status should be stopped
	if ag.Status() != agent.AgentStatusStopped {
		t.Errorf("Status = %v, want %v", ag.Status(), agent.AgentStatusStopped)
	}
}

// TestAgentSetPermissionMode tests permission mode switching
func TestAgentSetPermissionMode(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &Config{
		SessionID:      "test-session-mode",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
	}

	ag, err := NewAgent(config)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	ctx := context.Background()
	_ = ag.Start(ctx)

	// Check initial mode
	if ag.CurrentMode() != agent.PermissionModeDefault {
		t.Errorf("CurrentMode = %v, want %v", ag.CurrentMode(), agent.PermissionModeDefault)
	}

	// Change mode
	if err := ag.SetPermissionMode(agent.PermissionModeBypassPermissions); err != nil {
		t.Errorf("SetPermissionMode() error: %v", err)
	}

	// Check mode was changed
	if ag.CurrentMode() != agent.PermissionModeBypassPermissions {
		t.Errorf("CurrentMode = %v, want %v", ag.CurrentMode(), agent.PermissionModeBypassPermissions)
	}

	// Session ID should remain the same (resume was used)
	originalSessionID := config.SessionID
	if ag.SessionID() != originalSessionID {
		t.Errorf("SessionID should remain %v, got %v", originalSessionID, ag.SessionID())
	}
}

// TestAgentCurrentMode tests current mode retrieval
func TestAgentCurrentMode(t *testing.T) {
	config := &Config{
		SessionID:      "test-session-current-mode",
		WorkingDir:     "/tmp/test",
		PermissionMode: agent.PermissionModeAcceptEdits,
	}

	ag, err := NewAgent(config)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	// Check initial mode
	if ag.CurrentMode() != agent.PermissionModeAcceptEdits {
		t.Errorf("CurrentMode = %v, want %v", ag.CurrentMode(), agent.PermissionModeAcceptEdits)
	}
}

// TestAgentStatus tests status transitions
func TestAgentStatus(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &Config{
		SessionID:      "test-session-status",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
	}

	ag, err := NewAgent(config)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	// Initial status should be idle
	if ag.Status() != agent.AgentStatusIdle {
		t.Errorf("Status = %v, want %v", ag.Status(), agent.AgentStatusIdle)
	}

	ctx := context.Background()

	// After start, should be running
	_ = ag.Start(ctx)
	if ag.Status() != agent.AgentStatusRunning {
		t.Errorf("Status = %v, want %v", ag.Status(), agent.AgentStatusRunning)
	}

	// After stop, should be stopped
	_ = ag.Stop()
	if ag.Status() != agent.AgentStatusStopped {
		t.Errorf("Status = %v, want %v", ag.Status(), agent.AgentStatusStopped)
	}
}

// TestAgentSendMessageEmptyInput tests SendMessage with empty input
func TestAgentSendMessageEmptyInput(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &Config{
		SessionID:      "test-session-send-empty",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
	}

	ag, err := NewAgent(config)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	ctx := context.Background()
	_, err = ag.SendMessage(ctx, "", nil)
	if err != agent.ErrEmptyInput {
		t.Errorf("expected ErrEmptyInput, got %v", err)
	}
}

// TestAgentSendMessageNotRunning tests SendMessage when agent is not running
func TestAgentSendMessageNotRunning(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &Config{
		SessionID:      "test-session-send-not-running",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
	}

	ag, err := NewAgent(config)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	ctx := context.Background()
	_, err = ag.SendMessage(ctx, "hello", nil)
	if err != agent.ErrAgentNotRunning {
		t.Errorf("expected ErrAgentNotRunning, got %v", err)
	}
}

// TestAgentSendMessageConcurrent tests that concurrent SendMessage calls return ErrAgentBusy
func TestAgentSendMessageConcurrent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &Config{
		SessionID:      "test-session-send-concurrent",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
	}

	ag, err := NewAgent(config)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	ctx := context.Background()
	if err := ag.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer ag.Stop()

	// Mark agent as busy
	ag.mu.Lock()
	ag.busy = true
	ag.mu.Unlock()

	// Try to send message while busy
	_, err = ag.SendMessage(ctx, "hello", nil)
	if err != agent.ErrAgentBusy {
		t.Errorf("expected ErrAgentBusy, got %v", err)
	}
}

func TestAgentSendMessageFallsBackToAssistantTextWhenResultEmpty(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()
	mockClaudePath := createTestClaudeScript(t, tmpDir, `#!/bin/bash
echo '{"type":"system","subtype":"init","session_id":"fallback-test"}'
echo '{"type":"assistant","session_id":"fallback-test","message":{"content":[{"type":"tool_use","id":"toolu_1","name":"Read","input":{"file":"README.md"}},{"type":"text","text":"Final answer from text block"}]}}'
echo '{"type":"result","subtype":"success","session_id":"fallback-test","result":""}'
sleep 0.1
`)

	config := &Config{
		SessionID:      "test-session-fallback-text",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
		ClaudePath:     mockClaudePath,
	}

	ag, err := NewAgent(config)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	ctx := context.Background()
	if err := ag.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer ag.Stop()

	resp, err := ag.SendMessage(ctx, "hello", nil)
	if err != nil {
		t.Fatalf("SendMessage() error: %v", err)
	}

	if resp.Content != "Final answer from text block" {
		t.Fatalf("Response.Content = %q, want %q", resp.Content, "Final answer from text block")
	}
	if resp.IsError {
		t.Fatal("Response.IsError = true, want false")
	}
}

func TestAgentSendMessageProcessFailureReturnsErrorWithoutPanic(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()
	mockClaudePath := createTestClaudeScript(t, tmpDir, `#!/bin/bash
last="${@: -1}"
if [ "$last" = "trigger failure" ]; then
	printf 'stderr failure details\n' >&2
	exit 1
fi
echo '{"type":"system","subtype":"init","session_id":"send-error-test"}'
sleep 5
`)

	config := &Config{
		SessionID:      "test-session-send-process-error",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
		ClaudePath:     mockClaudePath,
	}

	ag, err := NewAgent(config)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	ctx := context.Background()
	if err := ag.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer ag.Stop()

	_, err = ag.SendMessage(ctx, "trigger failure", nil)
	if err == nil {
		t.Fatal("SendMessage() error = nil, want process failure")
	}
	if !strings.Contains(err.Error(), "stderr failure details") {
		t.Fatalf("SendMessage() error = %v, want stderr details", err)
	}
}

func createTestClaudeScript(t *testing.T, dir, content string) string {
	t.Helper()
	scriptPath := filepath.Join(dir, "mock-claude.sh")
	if err := os.WriteFile(scriptPath, []byte(content), 0755); err != nil {
		t.Fatalf("Failed to create mock script: %v", err)
	}
	return scriptPath
}

// TestAgentHealthCheck tests health check functionality
func TestAgentHealthCheck(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &Config{
		SessionID:      "test-session-health",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
	}

	ag, err := NewAgent(config)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	ctx := context.Background()
	if err := ag.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	// Status should be running
	if ag.Status() != agent.AgentStatusRunning {
		t.Errorf("Status = %v, want %v", ag.Status(), agent.AgentStatusRunning)
	}

	// Simulate process crash by stopping it internally
	ag.mu.Lock()
	if ag.pm != nil && ag.pm.cmd != nil && ag.pm.cmd.Process != nil {
		ag.pm.cmd.Process.Kill()
	}
	ag.mu.Unlock()

	// Wait a bit for the process to exit
	time.Sleep(100 * time.Millisecond)

	// Check if crash was detected
	ag.mu.RLock()
	processRunning := ag.pm != nil && ag.pm.IsRunning()
	ag.mu.RUnlock()

	// Process should no longer be running
	if processRunning {
		t.Error("process should have been killed")
	}

	// Clean up
	ag.Stop()
}

// TestAgentRestart tests agent restart functionality
func TestAgentRestart(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &Config{
		SessionID:      "test-session-restart",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
	}

	ag, err := NewAgent(config)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	ctx := context.Background()
	if err := ag.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	// Status should be running
	if ag.Status() != agent.AgentStatusRunning {
		t.Errorf("Status = %v, want %v", ag.Status(), agent.AgentStatusRunning)
	}

	// Restart
	if err := ag.Restart(ctx); err != nil {
		t.Errorf("Restart() error: %v", err)
	}

	// Status should still be running
	if ag.Status() != agent.AgentStatusRunning {
		t.Errorf("Status after restart = %v, want %v", ag.Status(), agent.AgentStatusRunning)
	}

	// Session ID should remain the same
	if ag.SessionID() != config.SessionID {
		t.Errorf("SessionID = %v, want %v", ag.SessionID(), config.SessionID)
	}

	// Clean up
	ag.Stop()
}

// TestAgentRestartNotRunning tests restart when not running
func TestAgentRestartNotRunning(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &Config{
		SessionID:      "test-session-restart-not-running",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
	}

	ag, err := NewAgent(config)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	ctx := context.Background()

	// Restart without starting should fail
	err = ag.Restart(ctx)
	if err != agent.ErrAgentNotRunning {
		t.Errorf("expected ErrAgentNotRunning, got %v", err)
	}
}

// TestAgentSetPermissionModeWhenNotRunning tests SetPermissionMode when agent is not running
func TestAgentSetPermissionModeWhenNotRunning(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &Config{
		SessionID:      "test-session-mode-not-running",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
	}

	ag, err := NewAgent(config)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	// Change mode without starting should just update the mode
	if err := ag.SetPermissionMode(agent.PermissionModeBypassPermissions); err != nil {
		t.Errorf("SetPermissionMode() error: %v", err)
	}

	// Mode should be changed
	if ag.CurrentMode() != agent.PermissionModeBypassPermissions {
		t.Errorf("CurrentMode = %v, want bypassPermissions", ag.CurrentMode())
	}
}

// TestAgentStopWhenNotRunning tests Stop when agent is not running
func TestAgentStopWhenNotRunning(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &Config{
		SessionID:      "test-session-stop-not-running",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
	}

	ag, err := NewAgent(config)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	// Stop without starting should succeed (no-op)
	if err := ag.Stop(); err != nil {
		t.Errorf("Stop() error: %v", err)
	}
}

// TestAgentStopWhenAlreadyStopped tests Stop when agent is already stopped
func TestAgentStopWhenAlreadyStopped(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &Config{
		SessionID:      "test-session-stop-stopped",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
	}

	ag, err := NewAgent(config)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	ctx := context.Background()
	if err := ag.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	// First stop
	if err := ag.Stop(); err != nil {
		t.Errorf("Stop() error: %v", err)
	}

	// Second stop should be no-op
	if err := ag.Stop(); err != nil {
		t.Errorf("Second Stop() error: %v", err)
	}
}

// TestAgentNewAgentWithDefaults tests NewAgent with default values
func TestAgentNewAgentWithDefaults(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Config with minimal fields
	config := &Config{
		WorkingDir: tmpDir,
		// SessionID empty - should auto-generate
		// PermissionMode empty - should default to default
	}

	ag, err := NewAgent(config)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	// SessionID should be auto-generated
	if ag.SessionID() == "" {
		t.Error("SessionID should be auto-generated")
	}

	// PermissionMode should default to default
	if ag.CurrentMode() != agent.PermissionModeDefault {
		t.Errorf("CurrentMode = %v, want default", ag.CurrentMode())
	}
}

// TestAgentNewAgentWithEmptyPermissionMode tests NewAgent with empty permission mode
func TestAgentNewAgentWithEmptyPermissionMode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &Config{
		SessionID:  "test-session-empty-mode",
		WorkingDir: tmpDir,
		// PermissionMode is empty
	}

	ag, err := NewAgent(config)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	if ag.CurrentMode() != agent.PermissionModeDefault {
		t.Errorf("CurrentMode = %v, want default", ag.CurrentMode())
	}
}

// TestConvertToAgentEventAllTypes tests all event type conversions
func TestConvertToAgentEventAllTypes(t *testing.T) {
	tests := []struct {
		name        string
		event       *StreamEvent
		wantType    agent.StreamEventType
		wantContent string
		wantTool    bool
	}{
		{
			name:     "unknown type",
			event:    &StreamEvent{Type: "unknown"},
			wantType: "",
		},
		{
			name: "assistant text with content",
			event: &StreamEvent{
				Type: "assistant",
				Message: Message{
					Content: []Content{{Type: "text", Text: "Hello"}},
				},
			},
			wantType:    agent.StreamEventTypeText,
			wantContent: "Hello",
		},
		{
			name: "assistant tool_use with info",
			event: &StreamEvent{
				Type: "assistant",
				Message: Message{
					Content: []Content{{
						Type:  "tool_use",
						Name:  "Bash",
						ID:    "toolu_1",
						Input: map[string]interface{}{"command": "ls"},
					}},
				},
			},
			wantType: agent.StreamEventTypeToolUse,
			wantTool: true,
		},
		{
			name: "result success with content",
			event: &StreamEvent{
				Type:    "result",
				Subtype: "success",
				Result:  "Done!",
			},
			wantType:    agent.StreamEventTypeResult,
			wantContent: "Done!",
		},
		{
			name: "result error with message",
			event: &StreamEvent{
				Type:    "result",
				Subtype: "error",
				Error:   "Something failed",
			},
			wantType:    agent.StreamEventTypeError,
			wantContent: "Something failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ae := convertToAgentEvent(tt.event)
			if ae.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", ae.Type, tt.wantType)
			}
			if tt.wantContent != "" && ae.Content != tt.wantContent {
				t.Errorf("Content = %v, want %v", ae.Content, tt.wantContent)
			}
			if tt.wantTool && ae.Tool == nil {
				t.Error("expected Tool to be set")
			}
		})
	}
}

// TestStreamEventHelperMethods tests various StreamEvent helper methods
func TestStreamEventHelperMethods(t *testing.T) {
	t.Run("IsAssistantText with empty content", func(t *testing.T) {
		event := &StreamEvent{Type: "assistant"}
		if event.IsAssistantText() {
			t.Error("IsAssistantText should be false with empty content")
		}
	})

	t.Run("IsAssistantToolUse with empty content", func(t *testing.T) {
		event := &StreamEvent{Type: "assistant"}
		if event.IsAssistantToolUse() {
			t.Error("IsAssistantToolUse should be false with empty content")
		}
	})

	t.Run("IsUserToolResult with empty content", func(t *testing.T) {
		event := &StreamEvent{Type: "user"}
		if event.IsUserToolResult() {
			t.Error("IsUserToolResult should be false with empty content")
		}
	})

	t.Run("GetText with empty content", func(t *testing.T) {
		event := &StreamEvent{Type: "assistant"}
		if text := event.GetText(); text != "" {
			t.Errorf("GetText should return empty string, got %v", text)
		}
	})

	t.Run("GetToolInfo with empty content", func(t *testing.T) {
		event := &StreamEvent{Type: "assistant"}
		name, id, input := event.GetToolInfo()
		if name != "" || id != "" || input != nil {
			t.Errorf("GetToolInfo should return empty values, got name=%v, id=%v, input=%v", name, id, input)
		}
	})
}

// TestAgentConfigFields tests that all Config fields are properly used
func TestAgentConfigFields(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &Config{
		SessionID:      "test-config-session",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModePlan,
		ClaudePath:     "/custom/claude",
		AllowedTools:   []string{"Bash", "Read"},
		Env:            []string{"CUSTOM_VAR=value"},
	}

	ag, err := NewAgent(config)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	// Verify session ID is used
	if ag.SessionID() != "test-config-session" {
		t.Errorf("SessionID = %v, want test-config-session", ag.SessionID())
	}

	// Verify permission mode is used
	if ag.CurrentMode() != agent.PermissionModePlan {
		t.Errorf("CurrentMode = %v, want plan", ag.CurrentMode())
	}

	// Verify config is stored
	ag.mu.RLock()
	if ag.config.ClaudePath != "/custom/claude" {
		t.Errorf("ClaudePath = %v, want /custom/claude", ag.config.ClaudePath)
	}
	if len(ag.config.AllowedTools) != 2 {
		t.Errorf("len(AllowedTools) = %v, want 2", len(ag.config.AllowedTools))
	}
	if len(ag.config.Env) != 1 {
		t.Errorf("len(Env) = %v, want 1", len(ag.config.Env))
	}
	ag.mu.RUnlock()
}
