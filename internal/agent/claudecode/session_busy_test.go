package claudecode

import (
	"context"
	"runtime"
	"testing"
	"time"
)

// TestSessionIsBusyWhenWaiting tests that session reports busy when waiting for permission
func TestSessionIsBusyWhenWaiting(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()
	// Create a script that will trigger permission request
	mockClaudePath := createSessionTestClaudeScript(t, tmpDir, `#!/bin/bash
echo '{"type":"system","session_id":"test-session"}'
echo '{"type":"control_request","request_id":"req-123","request":{"subtype":"can_use_tool","tool_name":"Bash","input":{"command":"ls"}}}'
# Wait for response
cat > /dev/null &
sleep 0.5
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

	// Wait for permission request to be processed
	time.Sleep(100 * time.Millisecond)

	// Session should be busy while waiting for permission
	if !session.IsBusy() {
		t.Error("Expected session to be busy while waiting for permission")
	}

	// Respond to the permission
	err = session.RespondPermission("req-123", PermissionResult{
		Behavior: "allow",
		UpdatedInput: map[string]any{"command": "ls"},
	})
	if err != nil {
		t.Fatalf("RespondPermission() error: %v", err)
	}

	// Wait for response to be processed
	time.Sleep(50 * time.Millisecond)

	// Session should no longer be busy after permission resolved
	if session.IsBusy() {
		t.Error("Expected session to not be busy after permission resolved")
	}
}

// TestSessionSendRejectedWhenBusy tests that Send returns error when session is busy
func TestSessionSendRejectedWhenBusy(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()
	mockClaudePath := createSessionTestClaudeScript(t, tmpDir, `#!/bin/bash
echo '{"type":"system","session_id":"test-session"}'
echo '{"type":"control_request","request_id":"req-123","request":{"subtype":"can_use_tool","tool_name":"Bash","input":{"command":"ls"}}}'
cat > /dev/null &
sleep 1
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

	// Wait for permission request to be processed
	time.Sleep(100 * time.Millisecond)

	// Session should be busy
	if !session.IsBusy() {
		t.Fatal("Expected session to be busy")
	}

	// Try to send a message while busy
	err = session.Send("test message", nil, nil)
	if err == nil {
		t.Error("Expected error when sending message while busy")
	}
}

// TestSessionNotBusyWhenAutoApprove tests that session is not busy in auto-approve mode
func TestSessionNotBusyWhenAutoApprove(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()
	mockClaudePath := createSessionTestClaudeScript(t, tmpDir, `#!/bin/bash
echo '{"type":"system","session_id":"test-session"}'
echo '{"type":"control_request","request_id":"req-123","request":{"subtype":"can_use_tool","tool_name":"Bash","input":{"command":"ls"}}}'
cat > /dev/null &
sleep 1
`)

	config := &SessionConfig{
		WorkingDir:     tmpDir,
		ClaudePath:     mockClaudePath,
		PermissionMode: "yolo",
		AutoApprove:    true,
	}

	ctx := context.Background()
	session, err := newSession(ctx, config)
	if err != nil {
		t.Fatalf("newSession() error: %v", err)
	}
	defer session.Close()

	// Wait for permission request to be processed
	time.Sleep(100 * time.Millisecond)

	// Session should NOT be busy in auto-approve mode
	if session.IsBusy() {
		t.Error("Expected session to not be busy in auto-approve mode")
	}
}

// TestSessionNotBusyInitially tests that new session is not busy
func TestSessionNotBusyInitially(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()
	mockClaudePath := createSessionTestClaudeScript(t, tmpDir, `#!/bin/bash
echo '{"type":"system","session_id":"test-session"}'
# Exit after a short delay
sleep 0.1
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

	// Wait for session to initialize
	time.Sleep(50 * time.Millisecond)

	// New session should not be busy
	if session.IsBusy() {
		t.Error("Expected new session to not be busy")
	}
}
