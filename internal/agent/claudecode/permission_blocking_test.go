package claudecode

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"
)

// TestPermissionBlockingWaitsForResponse tests that the session blocks
// until a response is received
func TestPermissionBlockingWaitsForResponse(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()
	// Script exits after a short time to allow session to close
	mockClaudePath := createSessionTestClaudeScript(t, tmpDir, `#!/bin/bash
echo '{"type":"system","session_id":"test-session"}'
sleep 2
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

	// Start reading events in background to prevent blocking
	go func() {
		for range session.Events() {
			// Drain events
		}
	}()

	// Simulate handleControlRequest
	raw := map[string]interface{}{
		"type":       "control_request",
		"request_id": "req-block-123",
		"request": map[string]interface{}{
			"subtype":   "can_use_tool",
			"tool_name": "Bash",
			"input": map[string]interface{}{
				"command": "ls",
			},
		},
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		// This should block until we respond
		session.handleControlRequest(raw)
		// After respond, this should continue
	}()

	// Wait a bit for the goroutine to start and block
	time.Sleep(50 * time.Millisecond)

	// Verify pending state exists
	session.pendingMu.Lock()
	pending := session.pending
	session.pendingMu.Unlock()

	if pending == nil {
		t.Fatal("Expected pending state to be created")
	}

	// Now respond to the permission
	err = session.RespondPermission("req-block-123", PermissionResult{
		Behavior: "allow",
	})
	if err != nil {
		t.Fatalf("RespondPermission() error: %v", err)
	}

	// Wait for handleControlRequest to complete with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Good, completed
	case <-time.After(500 * time.Millisecond):
		t.Error("handleControlRequest did not complete in time")
	}
}

// TestPermissionBlockingTimeout tests that blocking times out
// and auto-denies after the timeout period
func TestPermissionBlockingTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()
	// Script exits after a short time
	mockClaudePath := createSessionTestClaudeScript(t, tmpDir, `#!/bin/bash
echo '{"type":"system","session_id":"test-session"}'
sleep 2
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

	// Set a short permission timeout for testing
	session.SetPermissionTimeout(100 * time.Millisecond)

	// Simulate handleControlRequest
	raw := map[string]interface{}{
		"type":       "control_request",
		"request_id": "req-timeout-123",
		"request": map[string]interface{}{
			"subtype":   "can_use_tool",
			"tool_name": "Bash",
			"input": map[string]interface{}{
				"command": "ls",
			},
		},
	}

	start := time.Now()
	session.handleControlRequest(raw)
	elapsed := time.Since(start)

	// Should have timed out after ~100ms
	if elapsed < 80*time.Millisecond {
		t.Errorf("Expected timeout after ~100ms, but only waited %v", elapsed)
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("Timeout took too long: %v", elapsed)
	}

	// Verify session is no longer busy after timeout
	if session.IsBusy() {
		t.Error("Expected session to not be busy after timeout")
	}
}

// TestPermissionBlockingContextCancel tests that blocking is cancelled
// when the context is cancelled
func TestPermissionBlockingContextCancel(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()
	// Script exits after a short time
	mockClaudePath := createSessionTestClaudeScript(t, tmpDir, `#!/bin/bash
echo '{"type":"system","session_id":"test-session"}'
sleep 2
`)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := &SessionConfig{
		WorkingDir:     tmpDir,
		ClaudePath:     mockClaudePath,
		PermissionMode: "default",
		AutoApprove:    false,
	}

	session, err := newSession(ctx, config)
	if err != nil {
		t.Fatalf("newSession() error: %v", err)
	}
	defer session.Close()

	// Simulate handleControlRequest
	raw := map[string]interface{}{
		"type":       "control_request",
		"request_id": "req-cancel-123",
		"request": map[string]interface{}{
			"subtype":   "can_use_tool",
			"tool_name": "Bash",
			"input": map[string]interface{}{
				"command": "ls",
			},
		},
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Cancel after a short delay
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	session.handleControlRequest(raw)
	elapsed := time.Since(start)

	// Should have been cancelled quickly
	if elapsed > 200*time.Millisecond {
		t.Errorf("Context cancel took too long: %v", elapsed)
	}

	wg.Wait()
}

// TestPermissionBlockingUserResponse tests that user response unblocks
// the waiting permission request
func TestPermissionBlockingUserResponse(t *testing.T) {
	pending := newPendingPermission("req-user-123", "Bash", map[string]any{"command": "ls"})

	var wg sync.WaitGroup
	var unblocked bool

	wg.Add(1)
	go func() {
		defer wg.Done()
		// Wait for resolution
		<-pending.resolved
		unblocked = true
	}()

	// Verify it's blocking initially
	time.Sleep(50 * time.Millisecond)
	if unblocked {
		t.Error("Should still be blocked")
	}

	// Set result (simulating user response)
	pending.setResult(&PermissionResult{Behavior: "allow"})

	// Wait for unblock
	wg.Wait()

	if !unblocked {
		t.Error("Should have been unblocked after setResult")
	}
}
