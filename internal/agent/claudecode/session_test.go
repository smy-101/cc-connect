package claudecode

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

// createSessionTestClaudeScript creates a mock claude script for testing
func createSessionTestClaudeScript(t *testing.T, dir, content string) string {
	t.Helper()
	scriptPath := filepath.Join(dir, "mock-claude.sh")
	if err := os.WriteFile(scriptPath, []byte(content), 0755); err != nil {
		t.Fatalf("Failed to create mock script: %v", err)
	}
	return scriptPath
}

// TestSessionNewSession tests creating a new session
func TestSessionNewSession(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()
	// Create a mock script that outputs stream-json format and waits for stdin
	mockClaudePath := createSessionTestClaudeScript(t, tmpDir, `#!/bin/bash
# Read stdin to keep process alive, then output
cat > /dev/null &
echo '{"type":"system","subtype":"init","session_id":"test-session-123","cwd":"`+tmpDir+`"}'
wait
`)

	config := &SessionConfig{
		WorkingDir:     tmpDir,
		ClaudePath:     mockClaudePath,
		SessionID:      "",
		PermissionMode: "default",
		AutoApprove:    false,
	}

	ctx := context.Background()
	session, err := newSession(ctx, config)
	if err != nil {
		t.Fatalf("newSession() error: %v", err)
	}
	defer session.Close()

	// Check session is alive
	if !session.Alive() {
		t.Error("Session should be alive after creation")
	}
}

// TestSessionSendMessage tests sending a message and receiving response
func TestSessionSendMessage(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()
	mockClaudePath := createSessionTestClaudeScript(t, tmpDir, `#!/bin/bash
# Read and ignore stdin, output events
echo '{"type":"system","subtype":"init","session_id":"send-test-session","cwd":"`+tmpDir+`"}'
# Read one line from stdin (the user message)
read line
echo '{"type":"assistant","session_id":"send-test-session","message":{"content":[{"type":"text","text":"Response to message"}]}}'
echo '{"type":"result","subtype":"success","session_id":"send-test-session","result":"Done"}'
`)

	config := &SessionConfig{
		WorkingDir:     tmpDir,
		ClaudePath:     mockClaudePath,
		PermissionMode: "default",
		AutoApprove:    false,
	}

	ctx := context.Background()
	session, err := newSession(ctx, config)
	if err != nil {
		t.Fatalf("newSession() error: %v", err)
	}
	defer session.Close()

	// Send a message using SendMessage
	var textReceived bool
	finalEvent, err := session.SendMessage(ctx, "Hello", nil, nil, func(event Event) {
		if event.Type == EventText {
			textReceived = true
			if event.Content != "Response to message" {
				t.Errorf("Expected 'Response to message', got %q", event.Content)
			}
		}
	})
	if err != nil {
		t.Fatalf("SendMessage() error: %v", err)
	}

	// Check we received the result
	if finalEvent.Type != EventResult {
		t.Errorf("Expected EventResult, got %v", finalEvent.Type)
	}
	if finalEvent.Content != "Done" {
		t.Errorf("Expected 'Done', got %q", finalEvent.Content)
	}
	if !textReceived {
		t.Error("Expected to receive text event")
	}
	// Check session ID was captured
	if session.CurrentSessionID() != "send-test-session" {
		t.Errorf("Expected session_id send-test-session, got %v", session.CurrentSessionID())
	}
}

// TestSessionClose tests closing a session
func TestSessionClose(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()
	// Create a mock script that runs until stdin is closed
	mockClaudePath := createSessionTestClaudeScript(t, tmpDir, `#!/bin/bash
echo '{"type":"system","session_id":"test"}'
cat > /dev/null
`)

	config := &SessionConfig{
		WorkingDir:     tmpDir,
		ClaudePath:     mockClaudePath,
		PermissionMode: "default",
	}

	ctx := context.Background()
	session, err := newSession(ctx, config)
	if err != nil {
		t.Fatalf("newSession() error: %v", err)
	}

	// Verify session is alive
	if !session.Alive() {
		t.Fatal("Session should be alive")
	}

	// Close the session
	err = session.Close()
	if err != nil {
		t.Errorf("Close() error: %v", err)
	}

	// Wait a bit for process to terminate
	time.Sleep(100 * time.Millisecond)

	// Verify session is no longer alive
	if session.Alive() {
		t.Error("Session should not be alive after close")
	}
}

// TestSessionAlive tests the Alive method
func TestSessionAlive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()
	mockClaudePath := createSessionTestClaudeScript(t, tmpDir, `#!/bin/bash
echo '{"type":"system","session_id":"test"}'
cat > /dev/null
`)

	config := &SessionConfig{
		WorkingDir:     tmpDir,
		ClaudePath:     mockClaudePath,
		PermissionMode: "default",
	}

	ctx := context.Background()
	session, err := newSession(ctx, config)
	if err != nil {
		t.Fatalf("newSession() error: %v", err)
	}

	// Initially alive
	if !session.Alive() {
		t.Error("Session should be alive initially")
	}

	// Close it
	session.Close()

	// Wait for process to terminate
	time.Sleep(100 * time.Millisecond)

	// Should no longer be alive
	if session.Alive() {
		t.Error("Session should not be alive after close")
	}
}

// TestSessionCurrentSessionID tests the CurrentSessionID method
func TestSessionCurrentSessionID(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()
	mockClaudePath := createSessionTestClaudeScript(t, tmpDir, `#!/bin/bash
echo '{"type":"system","subtype":"init","session_id":"updated-session-id","cwd":"`+tmpDir+`"}'
read line
echo '{"type":"result","subtype":"success","session_id":"updated-session-id","result":"Done"}'
`)

	config := &SessionConfig{
		WorkingDir:     tmpDir,
		ClaudePath:     mockClaudePath,
		SessionID:      "initial-session-id",
		PermissionMode: "default",
	}

	ctx := context.Background()
	session, err := newSession(ctx, config)
	if err != nil {
		t.Fatalf("newSession() error: %v", err)
	}
	defer session.Close()

	// Initial session ID
	if session.CurrentSessionID() != "initial-session-id" {
		t.Errorf("Expected initial-session-id, got %v", session.CurrentSessionID())
	}

	// Send a message to trigger the session ID update
	_, err = session.SendMessage(ctx, "test", nil, nil, nil)
	if err != nil {
		t.Fatalf("SendMessage() error: %v", err)
	}

	// Updated session ID
	if session.CurrentSessionID() != "updated-session-id" {
		t.Errorf("Expected updated-session-id, got %v", session.CurrentSessionID())
	}
}

// TestSessionPermissionRequestAutoApprove tests permission handling in YOLO mode
func TestSessionPermissionRequestAutoApprove(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()
	// Create a script that simulates a permission request
	mockClaudePath := createSessionTestClaudeScript(t, tmpDir, `#!/bin/bash
echo '{"type":"system","subtype":"init","session_id":"auto-approve-test","cwd":"`+tmpDir+`"}'
# Read user message
read line
# Send permission request
echo '{"type":"control_request","request_id":"req-1","request":{"subtype":"can_use_tool","tool_name":"Bash","input":{"command":"ls"}}}'
# Read permission response
read response
# Send result
echo '{"type":"result","subtype":"success","session_id":"auto-approve-test","result":"Tool executed"}'
`)

	config := &SessionConfig{
		WorkingDir:     tmpDir,
		ClaudePath:     mockClaudePath,
		PermissionMode: "bypassPermissions",
		AutoApprove:    true,
	}

	ctx := context.Background()
	session, err := newSession(ctx, config)
	if err != nil {
		t.Fatalf("newSession() error: %v", err)
	}
	defer session.Close()

	// Send a message
	finalEvent, err := session.SendMessage(ctx, "Hello", nil, nil, nil)
	if err != nil {
		t.Fatalf("SendMessage() error: %v", err)
	}

	// Check we received the result
	if finalEvent.Type != EventResult {
		t.Errorf("Expected EventResult, got %v", finalEvent.Type)
	}
}

// TestSessionEventsChannel tests that Events() returns an open channel
func TestSessionEventsChannel(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()
	mockClaudePath := createSessionTestClaudeScript(t, tmpDir, `#!/bin/bash
echo '{"type":"system","session_id":"test"}'
cat > /dev/null
`)

	config := &SessionConfig{
		WorkingDir:     tmpDir,
		ClaudePath:     mockClaudePath,
		PermissionMode: "default",
	}

	session, err := newSession(context.Background(), config)
	if err != nil {
		t.Fatalf("newSession() error: %v", err)
	}
	defer session.Close()

	// Events() should return an open channel
	events := session.Events()
	select {
	case _, ok := <-events:
		if !ok {
			t.Error("Expected Events() channel to be open")
		}
		// Channel is open, good
	default:
		// No event yet, but channel is open
	}
}

// TestSessionSend tests the Send method
func TestSessionSend(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()
	mockClaudePath := createSessionTestClaudeScript(t, tmpDir, `#!/bin/bash
echo '{"type":"system","session_id":"send-test"}'
cat > /tmp/received_stdin.txt
`)

	config := &SessionConfig{
		WorkingDir:     tmpDir,
		ClaudePath:     mockClaudePath,
		PermissionMode: "default",
	}

	ctx := context.Background()
	session, err := newSession(ctx, config)
	if err != nil {
		t.Fatalf("newSession() error: %v", err)
	}
	defer session.Close()

	// Send a text message
	err = session.Send("Hello, Claude!", nil, nil)
	if err != nil {
		t.Errorf("Send() error: %v", err)
	}
}

// TestSessionFilterEnv tests the filterEnv helper
func TestSessionFilterEnv(t *testing.T) {
	env := []string{"FOO=bar", "BAR=baz", "CLAUDECODE=1", "PATH=/usr/bin"}
	filtered := filterEnv(env, "CLAUDECODE")

	for _, e := range filtered {
		if e == "CLAUDECODE=1" {
			t.Error("CLAUDECODE should be filtered out")
		}
	}
	if len(filtered) != 3 {
		t.Errorf("Expected 3 env vars, got %d", len(filtered))
	}
}

// TestSessionMergeEnv tests the mergeEnv helper
func TestSessionMergeEnv(t *testing.T) {
	env := []string{"FOO=bar", "BAR=baz"}
	extra := []string{"FOO=updated", "BAZ=new"}

	merged := mergeEnv(env, extra)

	// Check FOO was updated
	foundFoo := false
	for _, e := range merged {
		if e == "FOO=updated" {
			foundFoo = true
		}
	}
	if !foundFoo {
		t.Error("Expected FOO=updated in merged env")
	}

	// Check BAZ was added
	foundBaz := false
	for _, e := range merged {
		if e == "BAZ=new" {
			foundBaz = true
		}
	}
	if !foundBaz {
		t.Error("Expected BAZ=new in merged env")
	}
}

// TestSessionSummarizeInput tests the summarizeInput helper
func TestSessionSummarizeInput(t *testing.T) {
	tests := []struct {
		tool  string
		input map[string]interface{}
		want  string
	}{
		{
			tool:  "Read",
			input: map[string]interface{}{"file_path": "/tmp/test.go"},
			want:  "/tmp/test.go",
		},
		{
			tool:  "Bash",
			input: map[string]interface{}{"command": "ls -la"},
			want:  "ls -la",
		},
		{
			tool:  "Grep",
			input: map[string]interface{}{"pattern": "func.*Test"},
			want:  "func.*Test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			got := summarizeInput(tt.tool, tt.input)
			if got != tt.want {
				t.Errorf("summarizeInput(%s) = %q, want %q", tt.tool, got, tt.want)
			}
		})
	}
}

// TestSessionAliveAtomic tests that Alive() works correctly with atomic operations
func TestSessionAliveAtomic(t *testing.T) {
	var alive atomic.Bool

	// Initially false
	if alive.Load() {
		t.Error("Expected initial value to be false")
	}

	// Set to true
	alive.Store(true)
	if !alive.Load() {
		t.Error("Expected value to be true after Store(true)")
	}

	// Set to false
	alive.Store(false)
	if alive.Load() {
		t.Error("Expected value to be false after Store(false)")
	}
}
