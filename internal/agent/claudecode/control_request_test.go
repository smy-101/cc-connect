package claudecode

import (
	"context"
	"runtime"
	"testing"
	"time"
)

// TestHandleControlRequestCreatesPending tests that handleControlRequest
// creates a pending state when autoApprove is false
func TestHandleControlRequestCreatesPending(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()
	// Script that sends system message then exits after timeout
	// The test needs the session to stay alive during the permission check
	mockClaudePath := createSessionTestClaudeScript(t, tmpDir, `#!/bin/bash
echo '{"type":"system","session_id":"test-session"}'
# Wait a bit then exit (test will close us)
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

	// Set a short permission timeout so the test doesn't hang
	session.SetPermissionTimeout(100 * time.Millisecond)

	// Drain events in background
	go func() {
		for range session.Events() {
		}
	}()

	// Simulate handleControlRequest being called with a control request
	raw := map[string]interface{}{
		"type":       "control_request",
		"request_id": "req-test-123",
		"request": map[string]interface{}{
			"subtype":   "can_use_tool",
			"tool_name": "Bash",
			"input": map[string]interface{}{
				"command": "ls -la",
			},
		},
	}

	// Run handleControlRequest in goroutine since it now blocks
	done := make(chan struct{})
	go func() {
		defer close(done)
		session.handleControlRequest(raw)
	}()

	// Wait a bit for the pending state to be created
	time.Sleep(20 * time.Millisecond)

	// Check that pending state was created
	session.pendingMu.Lock()
	pending := session.pending
	session.pendingMu.Unlock()

	if pending == nil {
		t.Fatal("Expected pending state to be created")
	}
	if pending.requestID != "req-test-123" {
		t.Errorf("Expected requestID 'req-test-123', got %q", pending.requestID)
	}
	if pending.toolName != "Bash" {
		t.Errorf("Expected toolName 'Bash', got %q", pending.toolName)
	}
	if pending.toolInput != "ls -la" {
		t.Errorf("Expected toolInput 'ls -la', got %q", pending.toolInput)
	}
	if pending.isResolved() {
		t.Error("Expected pending to not be resolved initially")
	}

	// Wait for the timeout to complete (with timeout to prevent test hang)
	select {
	case <-done:
		// Good, completed
	case <-time.After(500 * time.Millisecond):
		t.Error("handleControlRequest did not complete in time")
	}
}

// TestHandleControlRequestAutoApproveNoPending tests that auto-approve mode
// does not create a pending state
func TestHandleControlRequestAutoApproveNoPending(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()
	mockClaudePath := createSessionTestClaudeScript(t, tmpDir, `#!/bin/bash
echo '{"type":"system","session_id":"test-session"}'
cat > /dev/null
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

	// Simulate handleControlRequest
	raw := map[string]interface{}{
		"type":       "control_request",
		"request_id": "req-auto-123",
		"request": map[string]interface{}{
			"subtype":   "can_use_tool",
			"tool_name": "Bash",
			"input": map[string]interface{}{
				"command": "ls",
			},
		},
	}

	session.handleControlRequest(raw)

	// Check that NO pending state was created (auto-approve)
	session.pendingMu.Lock()
	pending := session.pending
	session.pendingMu.Unlock()

	if pending != nil {
		t.Error("Expected NO pending state in auto-approve mode")
	}
}

// TestHandleControlRequestEventFields tests that the event sent by handleControlRequest
// contains the correct fields
func TestHandleControlRequestEventFields(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()
	mockClaudePath := createSessionTestClaudeScript(t, tmpDir, `#!/bin/bash
echo '{"type":"system","session_id":"test-session"}'
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

	// Simulate handleControlRequest in a goroutine
	go func() {
		raw := map[string]interface{}{
			"type":       "control_request",
			"request_id": "req-event-123",
			"request": map[string]interface{}{
				"subtype":   "can_use_tool",
				"tool_name": "Read",
				"input": map[string]interface{}{
					"file_path": "/tmp/test.go",
				},
			},
		}
		session.handleControlRequest(raw)
	}()

	// Read the event from the events channel
	select {
	case evt := <-session.Events():
		if evt.Type != EventPermissionRequest {
			t.Errorf("Expected EventPermissionRequest, got %v", evt.Type)
		}
		if evt.RequestID != "req-event-123" {
			t.Errorf("Expected RequestID 'req-event-123', got %q", evt.RequestID)
		}
		if evt.ToolName != "Read" {
			t.Errorf("Expected ToolName 'Read', got %q", evt.ToolName)
		}
		if evt.ToolInput == nil {
			t.Error("Expected ToolInput to be set")
		}
		if evt.AutoApproved {
			t.Error("Expected AutoApproved to be false in non-auto-approve mode")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for event")
	}
}

// TestHandleControlRequestAutoApproveEvent tests that auto-approve mode
// sends an event marked as auto-approved
func TestHandleControlRequestAutoApproveEvent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()
	mockClaudePath := createSessionTestClaudeScript(t, tmpDir, `#!/bin/bash
echo '{"type":"system","session_id":"test-session"}'
cat > /dev/null
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

	// Simulate handleControlRequest in a goroutine
	go func() {
		raw := map[string]interface{}{
			"type":       "control_request",
			"request_id": "req-auto-event-123",
			"request": map[string]interface{}{
				"subtype":   "can_use_tool",
				"tool_name": "Bash",
				"input": map[string]interface{}{
					"command": "echo hello",
				},
			},
		}
		session.handleControlRequest(raw)
	}()

	// Read the event from the events channel
	select {
	case evt := <-session.Events():
		if evt.Type != EventPermissionRequest {
			t.Errorf("Expected EventPermissionRequest, got %v", evt.Type)
		}
		if !evt.AutoApproved {
			t.Error("Expected AutoApproved to be true in auto-approve mode")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for event")
	}
}

// TestHandleControlRequestAskUserQuestion tests parsing AskUserQuestion tool input
func TestHandleControlRequestAskUserQuestion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()
	mockClaudePath := createSessionTestClaudeScript(t, tmpDir, `#!/bin/bash
echo '{"type":"system","session_id":"test-session"}'
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

	// Simulate handleControlRequest with AskUserQuestion
	go func() {
		raw := map[string]interface{}{
			"type":       "control_request",
			"request_id": "req-ask-123",
			"request": map[string]interface{}{
				"subtype":   "can_use_tool",
				"tool_name": "AskUserQuestion",
				"input": map[string]interface{}{
					"questions": []interface{}{
						map[string]interface{}{
							"question": "Which database?",
							"header":   "Database",
							"options": []interface{}{
								map[string]interface{}{"label": "PostgreSQL", "description": "Advanced RDBMS"},
								map[string]interface{}{"label": "MySQL", "description": "Popular RDBMS"},
							},
							"multiSelect": false,
						},
					},
				},
			},
		}
		session.handleControlRequest(raw)
	}()

	// Read the event
	select {
	case evt := <-session.Events():
		if evt.ToolName != "AskUserQuestion" {
			t.Errorf("Expected ToolName 'AskUserQuestion', got %q", evt.ToolName)
		}
		if len(evt.Questions) == 0 {
			t.Fatal("Expected Questions to be parsed")
		}
		if evt.Questions[0].Question != "Which database?" {
			t.Errorf("Expected question 'Which database?', got %q", evt.Questions[0].Question)
		}
		if len(evt.Questions[0].Options) != 2 {
			t.Errorf("Expected 2 options, got %d", len(evt.Questions[0].Options))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for event")
	}

	// Check pending state has questions
	session.pendingMu.Lock()
	pending := session.pending
	session.pendingMu.Unlock()

	if pending == nil {
		t.Fatal("Expected pending state to be created")
	}
	if len(pending.questions) == 0 {
		t.Error("Expected pending.questions to be set")
	}
}
