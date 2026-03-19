package claudecode

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestRespondPermissionNoPending tests that RespondPermission returns error
// when there is no pending permission request
func TestRespondPermissionNoPending(t *testing.T) {
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
		AutoApprove:    false,
	}

	ctx := context.Background()
	session, err := newSession(ctx, config)
	if err != nil {
		t.Fatalf("newSession() error: %v", err)
	}
	defer session.Close()

	// Try to respond to a non-existent permission request
	err = session.RespondPermission("non-existent-req", PermissionResult{
		Behavior: "allow",
	})
	if err == nil {
		t.Error("Expected error when responding to non-existent request")
	}

	// Check the error message contains the request ID
	if !strings.Contains(err.Error(), "no pending permission request") {
		t.Errorf("Expected error to contain 'no pending permission request', got: %v", err)
	}
}

// TestRespondPermissionAlreadyResolved tests that RespondPermission returns error
// when trying to respond to an already resolved request
func TestRespondPermissionAlreadyResolved(t *testing.T) {
	pending := newPendingPermission("req-1", "Bash", map[string]any{"command": "ls"})

	// First response should succeed
	pending.setResult(&PermissionResult{Behavior: "allow"})

	// Second response should be ignored (result already set)
	pending.setResult(&PermissionResult{Behavior: "deny"})

	// Verify the first result is preserved
	if pending.getResult().Behavior != "allow" {
		t.Errorf("Expected behavior 'allow', got %q", pending.getResult().Behavior)
	}

	// Verify resolved channel is closed
	select {
	case <-pending.resolved:
		// Good, channel is closed
	default:
		t.Error("Expected resolved channel to be closed")
	}
}

// TestRespondPermissionConcurrent tests that concurrent calls to setResult
// are handled safely
func TestRespondPermissionConcurrent(t *testing.T) {
	pending := newPendingPermission("req-1", "Bash", map[string]any{"command": "ls"})

	var wg sync.WaitGroup
	results := make(chan string, 10)

	// Try to set result from multiple goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			behavior := "allow"
			if idx%2 == 0 {
				behavior = "deny"
			}
			pending.setResult(&PermissionResult{Behavior: behavior})
			results <- behavior
		}(i)
	}

	wg.Wait()
	close(results)

	// Verify only one result is stored
	storedResult := pending.getResult()
	if storedResult == nil {
		t.Fatal("Expected result to be set")
	}

	// Verify channel is closed (only once)
	select {
	case <-pending.resolved:
		// Good
	default:
		t.Error("Expected resolved channel to be closed")
	}
}

// TestPendingPermissionIsResolved tests the isResolved method
func TestPendingPermissionIsResolved(t *testing.T) {
	pending := newPendingPermission("req-1", "Bash", map[string]any{"command": "ls"})

	// Initially not resolved
	if pending.isResolved() {
		t.Error("Expected pending to not be resolved initially")
	}

	// Set result
	pending.setResult(&PermissionResult{Behavior: "allow"})

	// Now resolved
	if !pending.isResolved() {
		t.Error("Expected pending to be resolved after setResult")
	}
}

// TestRespondPermissionValidRequest tests responding to a valid pending request
func TestRespondPermissionValidRequest(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()
	// Create a script that reads permission responses
	mockClaudePath := createSessionTestClaudeScript(t, tmpDir, `#!/bin/bash
echo '{"type":"system","session_id":"test-session"}'
cat > /tmp/permission_responses.txt
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

	// Manually set up a pending permission (simulating handleControlRequest)
	session.pendingMu.Lock()
	session.pending = newPendingPermission("req-123", "Bash", map[string]any{"command": "ls -la"})
	session.pendingMu.Unlock()

	// Respond to the pending request
	err = session.RespondPermission("req-123", PermissionResult{
		Behavior:     "allow",
		UpdatedInput: map[string]any{"command": "ls -la"},
	})
	if err != nil {
		t.Errorf("RespondPermission() unexpected error: %v", err)
	}

	// Verify the pending state was cleared (marked as resolved)
	session.pendingMu.Lock()
	resolved := session.pending != nil && session.pending.isResolved()
	session.pendingMu.Unlock()

	if !resolved {
		t.Error("Expected pending permission to be resolved")
	}
}

// TestRespondPermissionWrongRequestID tests responding with wrong request ID
func TestRespondPermissionWrongRequestID(t *testing.T) {
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
		AutoApprove:    false,
	}

	ctx := context.Background()
	session, err := newSession(ctx, config)
	if err != nil {
		t.Fatalf("newSession() error: %v", err)
	}
	defer session.Close()

	// Set up a pending permission with one ID
	session.pendingMu.Lock()
	session.pending = newPendingPermission("req-correct", "Bash", map[string]any{"command": "ls"})
	session.pendingMu.Unlock()

	// Try to respond with a different ID
	err = session.RespondPermission("req-wrong", PermissionResult{
		Behavior: "allow",
	})
	if err == nil {
		t.Error("Expected error when responding with wrong request ID")
	}

	if !strings.Contains(err.Error(), "no pending permission request") {
		t.Errorf("Expected error to contain 'no pending permission request', got: %v", err)
	}
}

// TestRespondPermissionDenialWithMessage tests denying with a custom message
func TestRespondPermissionDenialWithMessage(t *testing.T) {
	pending := newPendingPermission("req-1", "Bash", map[string]any{"command": "rm -rf"})

	// Deny with message
	pending.setResult(&PermissionResult{
		Behavior: "deny",
		Message:  "This command is too dangerous",
	})

	result := pending.getResult()
	if result.Behavior != "deny" {
		t.Errorf("Expected behavior 'deny', got %q", result.Behavior)
	}
	if result.Message != "This command is too dangerous" {
		t.Errorf("Expected message 'This command is too dangerous', got %q", result.Message)
	}
}

// TestPendingPermissionChannelBlocking tests that the resolved channel blocks until result is set
func TestPendingPermissionChannelBlocking(t *testing.T) {
	pending := newPendingPermission("req-1", "Bash", map[string]any{"command": "ls"})

	// Channel should block initially
	select {
	case <-pending.resolved:
		t.Error("Expected resolved channel to block")
	case <-time.After(50 * time.Millisecond):
		// Good, channel is blocking
	}

	// Set result in a goroutine
	go func() {
		time.Sleep(100 * time.Millisecond)
		pending.setResult(&PermissionResult{Behavior: "allow"})
	}()

	// Now channel should unblock after result is set
	select {
	case <-pending.resolved:
		// Good, channel unblocked
	case <-time.After(200 * time.Millisecond):
		t.Error("Expected resolved channel to unblock after setResult")
	}
}

// Helper function for tests
func createPermissionStateTestClaudeScript(t *testing.T, dir, content string) string {
	t.Helper()
	scriptPath := filepath.Join(dir, "mock-claude.sh")
	if err := os.WriteFile(scriptPath, []byte(content), 0755); err != nil {
		t.Fatalf("Failed to create mock script: %v", err)
	}
	return scriptPath
}
