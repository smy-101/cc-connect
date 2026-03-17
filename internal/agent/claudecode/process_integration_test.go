//go:build integration

package claudecode

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/smy-101/cc-connect/internal/agent"
)

// TestIntegrationProcessStartReal tests starting a real subprocess
func TestIntegrationProcessStartReal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows - signal handling differs")
	}

	tmpDir := t.TempDir()

	// Create a mock claude script that outputs JSONL
	mockClaudePath := createMockClaudeScript(t, tmpDir, `#!/bin/bash
echo '{"type":"system","subtype":"init","session_id":"test-123"}'
echo '{"type":"result","subtype":"success","result":"Done"}'
`)

	config := &ProcessConfig{
		SessionID:      "test-session-real",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
		ClaudePath:     mockClaudePath,
	}

	pm := NewProcessManager(config)
	ctx := context.Background()

	// Start the process
	err := pm.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer pm.Stop()

	// Verify process is running
	if !pm.IsRunning() {
		t.Error("Process should be running after Start()")
	}

	// Read output
	output, err := io.ReadAll(pm.Stdout())
	if err != nil {
		t.Fatalf("Failed to read stdout: %v", err)
	}

	if len(output) == 0 {
		t.Error("Expected output from process")
	}

	// Verify output contains expected JSON
	if !strings.Contains(string(output), `"type":"system"`) {
		t.Errorf("Expected system event in output, got: %s", string(output))
	}
}

// TestIntegrationProcessStopGraceful tests graceful process termination
func TestIntegrationProcessStopGraceful(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows - signal handling differs")
	}

	tmpDir := t.TempDir()

	// Create a script that handles SIGTERM gracefully
	mockClaudePath := createMockClaudeScript(t, tmpDir, `#!/bin/bash
trap 'echo "SIGTERM received"; exit 0' TERM
echo '{"type":"system","subtype":"init"}'
# Keep running until signaled
while true; do
	sleep 0.1
done
`)

	config := &ProcessConfig{
		SessionID:      "test-session-graceful",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
		ClaudePath:     mockClaudePath,
	}

	pm := NewProcessManager(config)
	ctx := context.Background()

	err := pm.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Give process time to set up signal handler
	time.Sleep(100 * time.Millisecond)

	// Stop should trigger graceful shutdown
	start := time.Now()
	err = pm.Stop()
	elapsed := time.Since(start)

	if err != nil {
		// Note: Process may report "killed" if it didn't exit within timeout
		t.Logf("Stop() returned: %v (elapsed: %v)", err, elapsed)
	}

	// Process should not be running after stop
	if pm.IsRunning() {
		t.Error("Process should not be running after Stop()")
	}
}

// TestIntegrationProcessStopForceKill tests force kill when graceful fails
func TestIntegrationProcessStopForceKill(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows - signal handling differs")
	}

	tmpDir := t.TempDir()

	// Create a script that ignores SIGTERM
	mockClaudePath := createMockClaudeScript(t, tmpDir, `#!/bin/bash
trap '' TERM
echo '{"type":"system","subtype":"init"}'
# Ignore SIGTERM and keep running
while true; do
	sleep 0.1
done
`)

	config := &ProcessConfig{
		SessionID:      "test-session-force",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
		ClaudePath:     mockClaudePath,
	}

	pm := NewProcessManager(config)
	ctx := context.Background()

	err := pm.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Give process time to set up
	time.Sleep(100 * time.Millisecond)

	// Stop should eventually force kill
	start := time.Now()
	err = pm.Stop()
	elapsed := time.Since(start)

	// Should have taken at least 2 seconds (the timeout before SIGKILL)
	if elapsed < 2*time.Second {
		t.Logf("Warning: Stop() completed quickly (%v), expected force kill delay", elapsed)
	}

	// The error should indicate it was killed
	if err == nil {
		t.Log("Stop() completed without error (process may have exited)")
	} else {
		t.Logf("Stop() returned: %v (elapsed: %v)", err, elapsed)
	}

	// Process should not be running
	if pm.IsRunning() {
		t.Error("Process should not be running after Stop()")
	}
}

// TestIntegrationProcessRestart tests restarting a process
func TestIntegrationProcessRestart(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows - signal handling differs")
	}

	tmpDir := t.TempDir()

	// Create a counter file to verify restart happened
	counterFile := filepath.Join(tmpDir, "counter.txt")
	_ = os.WriteFile(counterFile, []byte("0"), 0644)

	// Create a script that increments counter on each run
	mockClaudePath := createMockClaudeScript(t, tmpDir, fmt.Sprintf(`#!/bin/bash
count=$(cat %s)
count=$((count + 1))
echo $count > %s
echo '{"type":"system","subtype":"init","session_id":"test-'$count'"}'
echo '{"type":"result","subtype":"success","result":"Run '$count'"}'
`, counterFile, counterFile))

	config := &ProcessConfig{
		SessionID:      "test-session-restart",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
		ClaudePath:     mockClaudePath,
	}

	pm := NewProcessManager(config)
	ctx := context.Background()

	// First start
	err := pm.Start(ctx)
	if err != nil {
		t.Fatalf("First Start() error = %v", err)
	}

	// Wait for first run to complete
	time.Sleep(200 * time.Millisecond)
	pm.Stop()

	// Verify counter is 1
	data, _ := os.ReadFile(counterFile)
	if strings.TrimSpace(string(data)) != "1" {
		t.Errorf("Expected counter to be 1 after first run, got: %s", string(data))
	}

	// Restart with same session
	config.Resume = true
	pm = NewProcessManager(config)
	err = pm.Start(ctx)
	if err != nil {
		t.Fatalf("Second Start() error = %v", err)
	}

	// Wait for second run
	time.Sleep(200 * time.Millisecond)
	pm.Stop()

	// Verify counter is 2 (restart happened)
	data, _ = os.ReadFile(counterFile)
	if strings.TrimSpace(string(data)) != "2" {
		t.Errorf("Expected counter to be 2 after restart, got: %s", string(data))
	}
}

// TestIntegrationProcessPipes tests stdin/stdout/stderr pipe communication
func TestIntegrationProcessPipes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()

	// Create a script that echoes stdin to stdout and logs to stderr
	mockClaudePath := createMockClaudeScript(t, tmpDir, `#!/bin/bash
echo '{"type":"system","subtype":"init"}' >&2
while IFS= read -r line; do
    echo "{\"type\":\"assistant\",\"message\":{\"content\":[{\"type\":\"text\",\"text\":\"Echo: $line\"}]}}"
done
echo '{"type":"result","subtype":"success","result":"Done"}'
`)

	config := &ProcessConfig{
		SessionID:      "test-session-pipes",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
		ClaudePath:     mockClaudePath,
	}

	pm := NewProcessManager(config)
	ctx := context.Background()

	err := pm.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer pm.Stop()

	// Verify pipes are available
	if pm.Stdin() == nil {
		t.Error("Stdin() should not be nil after start")
	}
	if pm.Stdout() == nil {
		t.Error("Stdout() should not be nil after start")
	}
	if pm.Stderr() == nil {
		t.Error("Stderr() should not be nil after start")
	}

	// Write to stdin
	_, err = pm.Stdin().Write([]byte("Hello\n"))
	if err != nil {
		t.Fatalf("Failed to write to stdin: %v", err)
	}

	// Read from stdout with timeout
	stdoutCh := make(chan string)
	go func() {
		scanner := bufio.NewScanner(pm.Stdout())
		if scanner.Scan() {
			stdoutCh <- scanner.Text()
		}
	}()

	select {
	case line := <-stdoutCh:
		if !strings.Contains(line, "Echo: Hello") {
			t.Errorf("Expected 'Echo: Hello' in output, got: %s", line)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for stdout")
	}
}

// TestIntegrationProcessAlreadyRunning tests double start error
func TestIntegrationProcessAlreadyRunning(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()

	mockClaudePath := createMockClaudeScript(t, tmpDir, `#!/bin/bash
while true; do sleep 0.1; done
`)

	config := &ProcessConfig{
		SessionID:      "test-session-double",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
		ClaudePath:     mockClaudePath,
	}

	pm := NewProcessManager(config)
	ctx := context.Background()

	err := pm.Start(ctx)
	if err != nil {
		t.Fatalf("First Start() error = %v", err)
	}
	defer pm.Stop()

	// Second start should fail
	err = pm.Start(ctx)
	if err != agent.ErrAgentAlreadyRunning {
		t.Errorf("Expected ErrAgentAlreadyRunning, got: %v", err)
	}
}

// TestIntegrationProcessSignalHandling tests that signals are properly handled
func TestIntegrationProcessSignalHandling(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows - signal handling differs")
	}

	tmpDir := t.TempDir()

	// Create a script that records signal receipt
	signalFile := filepath.Join(tmpDir, "signals.txt")
	mockClaudePath := createMockClaudeScript(t, tmpDir, fmt.Sprintf(`#!/bin/bash
trap 'echo "SIGTERM" >> %s; exit 0' TERM
trap 'echo "SIGKILL" >> %s; exit 9' KILL
echo '{"type":"system","subtype":"init"}'
while true; do
	sleep 0.1
done
`, signalFile, signalFile))

	config := &ProcessConfig{
		SessionID:      "test-session-signal",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
		ClaudePath:     mockClaudePath,
	}

	pm := NewProcessManager(config)
	ctx := context.Background()

	err := pm.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Give process time to set up handlers
	time.Sleep(100 * time.Millisecond)

	// Stop should send SIGTERM
	err = pm.Stop()
	t.Logf("Stop() returned: %v", err)

	// Check signal was received
	data, err := os.ReadFile(signalFile)
	if err != nil {
		t.Logf("Could not read signal file: %v", err)
	} else {
		signalLog := string(data)
		t.Logf("Signal log: %s", signalLog)
		if !strings.Contains(signalLog, "SIGTERM") {
			t.Error("Expected SIGTERM to be sent and logged")
		}
	}
}

// TestIntegrationProcessContext tests context cancellation
func TestIntegrationProcessContext(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()

	mockClaudePath := createMockClaudeScript(t, tmpDir, `#!/bin/bash
echo '{"type":"system","subtype":"init"}'
while true; do
	sleep 0.1
done
`)

	config := &ProcessConfig{
		SessionID:      "test-session-context",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
		ClaudePath:     mockClaudePath,
	}

	pm := NewProcessManager(config)

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := pm.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer pm.Stop()

	// Wait for context to expire and allow the process a moment to be reaped.
	<-ctx.Done()
	time.Sleep(100 * time.Millisecond)

	if pm.IsRunning() {
		t.Fatal("process should stop when context is canceled or times out")
	}
}

// TestIntegrationProcessWithMessage tests passing a message argument
func TestIntegrationProcessWithMessage(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()

	// Create a script that outputs the received message
	outputFile := filepath.Join(tmpDir, "output.txt")
	mockClaudePath := createMockClaudeScript(t, tmpDir, fmt.Sprintf(`#!/bin/bash
# Last argument is the message
message="${@: -1}"
echo "Received: $message" >> %s
echo '{"type":"system","subtype":"init"}'
echo '{"type":"result","subtype":"success","result":"Processed: '"$message"'"}'
`, outputFile))

	config := &ProcessConfig{
		SessionID:      "test-session-message",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
		ClaudePath:     mockClaudePath,
		Message:        "Hello, Claude!",
	}

	pm := NewProcessManager(config)
	ctx := context.Background()

	err := pm.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Wait for completion
	time.Sleep(200 * time.Millisecond)
	pm.Stop()

	// Verify message was passed
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Could not read output file: %v", err)
	}

	if !strings.Contains(string(data), "Hello, Claude!") {
		t.Errorf("Expected message to be passed to script, got: %s", string(data))
	}
}

// TestIntegrationProcessWithEnv tests custom environment variables
func TestIntegrationProcessWithEnv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()

	// Create a script that outputs env var
	outputFile := filepath.Join(tmpDir, "env.txt")
	mockClaudePath := createMockClaudeScript(t, tmpDir, fmt.Sprintf(`#!/bin/bash
echo "CUSTOM_VAR=$CUSTOM_VAR" >> %s
echo '{"type":"system","subtype":"init"}'
echo '{"type":"result","subtype":"success","result":"Done"}'
`, outputFile))

	config := &ProcessConfig{
		SessionID:      "test-session-env",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
		ClaudePath:     mockClaudePath,
		Env:            []string{"CUSTOM_VAR=test_value_123"},
	}

	pm := NewProcessManager(config)
	ctx := context.Background()

	err := pm.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	time.Sleep(200 * time.Millisecond)
	pm.Stop()

	// Verify env var was set
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Could not read output file: %v", err)
	}

	if !strings.Contains(string(data), "CUSTOM_VAR=test_value_123") {
		t.Errorf("Expected custom env var to be set, got: %s", string(data))
	}
}

// TestIntegrationProcessParseStream tests parsing streaming JSONL output
func TestIntegrationProcessParseStream(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()

	// Write JSONL events to a file that we can parse
	eventsFile := filepath.Join(tmpDir, "events.jsonl")
	eventsContent := `{"type":"system","subtype":"init","session_id":"stream-test","cwd":"/tmp","model":"sonnet"}
{"type":"assistant","session_id":"stream-test","message":{"content":[{"type":"text","text":"Thinking..."}]}}
{"type":"assistant","session_id":"stream-test","message":{"content":[{"type":"tool_use","id":"toolu_1","name":"Bash","input":{"command":"ls"}}]}}
{"type":"user","session_id":"stream-test","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"file1\nfile2"}]}}
{"type":"result","subtype":"success","session_id":"stream-test","result":"All done","total_cost_usd":0.0123,"duration_ms":5000}
`
	if err := os.WriteFile(eventsFile, []byte(eventsContent), 0644); err != nil {
		t.Fatalf("Failed to write events file: %v", err)
	}

	// Read and parse the file
	file, err := os.Open(eventsFile)
	if err != nil {
		t.Fatalf("Failed to open events file: %v", err)
	}
	defer file.Close()

	var events []*StreamEvent
	err = ParseFromReader(file, func(event *StreamEvent) error {
		events = append(events, event)
		return nil
	})

	if err != nil {
		t.Fatalf("ParseFromReader() error = %v", err)
	}

	// Verify we got all expected events
	expectedTypes := []string{"system", "assistant", "assistant", "user", "result"}
	if len(events) != len(expectedTypes) {
		t.Errorf("Expected %d events, got %d", len(expectedTypes), len(events))
	}

	for i, expectedType := range expectedTypes {
		if i >= len(events) {
			break
		}
		if events[i].Type != expectedType {
			t.Errorf("Event %d: expected type %s, got %s", i, expectedType, events[i].Type)
		}
	}

	// Verify result event
	if len(events) > 0 {
		lastEvent := events[len(events)-1]
		if !lastEvent.IsResultSuccess() {
			t.Error("Last event should be result/success")
		}
		if lastEvent.Result != "All done" {
			t.Errorf("Expected result 'All done', got: %s", lastEvent.Result)
		}
	}
}

// TestIntegrationProcessPermissionDenied tests handling permission denied response
func TestIntegrationProcessPermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()

	// Create a script that outputs permission denied
	mockClaudePath := createMockClaudeScript(t, tmpDir, `#!/bin/bash
echo '{"type":"system","subtype":"init","session_id":"perm-test"}'
echo '{"type":"result","subtype":"error","error":"Permission denied","permission_denials":[{"tool_name":"Bash","tool_use_id":"toolu_1","tool_input":{"command":"rm -rf /"}}]}'
`)

	config := &ProcessConfig{
		SessionID:      "test-session-perm",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
		ClaudePath:     mockClaudePath,
	}

	pm := NewProcessManager(config)
	ctx := context.Background()

	err := pm.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer pm.Stop()

	// Parse events
	var lastEvent *StreamEvent
	err = ParseFromReader(pm.Stdout(), func(event *StreamEvent) error {
		lastEvent = event
		return nil
	})

	if err != nil {
		t.Fatalf("ParseFromReader() error = %v", err)
	}

	// Verify permission denied
	if lastEvent == nil {
		t.Fatal("Expected at least one event")
	}

	if !lastEvent.IsResultError() {
		t.Error("Expected result/error event")
	}

	if !lastEvent.HasPermissionDenials() {
		t.Error("Expected permission_denials to be present")
	}

	if len(lastEvent.PermissionDenials) != 1 {
		t.Errorf("Expected 1 permission denial, got %d", len(lastEvent.PermissionDenials))
	}

	denied := lastEvent.PermissionDenials[0]
	if denied.ToolName != "Bash" {
		t.Errorf("Expected denied tool 'Bash', got: %s", denied.ToolName)
	}
}

// TestIntegrationProcessCrashRecovery tests behavior when process crashes
func TestIntegrationProcessCrashRecovery(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()

	// Create a script that exits with error after init
	mockClaudePath := createMockClaudeScript(t, tmpDir, `#!/bin/bash
echo '{"type":"system","subtype":"init","session_id":"crash-test"}'
echo '{"type":"result","subtype":"error","error":"Simulated crash"}'
exit 1
`)

	config := &ProcessConfig{
		SessionID:      "test-session-crash",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
		ClaudePath:     mockClaudePath,
	}

	pm := NewProcessManager(config)
	ctx := context.Background()

	err := pm.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Wait for process to complete
	waitErr := pm.Wait()
	t.Logf("Wait() returned: %v", waitErr)

	// Process should not be running after crash
	if pm.IsRunning() {
		t.Error("Process should not be running after crash")
	}

	// Verify we can restart
	config.Resume = true
	pm = NewProcessManager(config)
	err = pm.Start(ctx)
	if err != nil {
		t.Fatalf("Restart Start() error = %v", err)
	}
	pm.Stop()
}

// TestIntegrationProcessWorkingDir tests that working directory is set correctly
func TestIntegrationProcessWorkingDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	// Create a script that outputs its working directory
	outputFile := filepath.Join(tmpDir, "cwd.txt")
	mockClaudePath := createMockClaudeScript(t, tmpDir, fmt.Sprintf(`#!/bin/bash
pwd >> %s
echo '{"type":"system","subtype":"init"}'
echo '{"type":"result","subtype":"success","result":"Done"}'
`, outputFile))

	config := &ProcessConfig{
		SessionID:      "test-session-cwd",
		WorkingDir:     subDir,
		PermissionMode: agent.PermissionModeDefault,
		ClaudePath:     mockClaudePath,
	}

	pm := NewProcessManager(config)
	ctx := context.Background()

	err := pm.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	time.Sleep(200 * time.Millisecond)
	pm.Stop()

	// Verify working directory
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Could not read output file: %v", err)
	}

	actualCwd := strings.TrimSpace(string(data))
	if actualCwd != subDir {
		t.Errorf("Expected working dir %s, got %s", subDir, actualCwd)
	}
}

// TestIntegrationProcessDifferentModes tests different permission modes
func TestIntegrationProcessDifferentModes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()

	tests := []struct {
		name string
		mode agent.PermissionMode
		want string
	}{
		{"default", agent.PermissionModeDefault, "default"},
		{"acceptEdits", agent.PermissionModeAcceptEdits, "acceptEdits"},
		{"plan", agent.PermissionModePlan, "plan"},
		{"bypassPermissions", agent.PermissionModeBypassPermissions, "bypassPermissions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a script that outputs the permission-mode arg
			outputFile := filepath.Join(tmpDir, "mode_"+tt.name+".txt")
			mockClaudePath := createMockClaudeScript(t, tmpDir, fmt.Sprintf(`#!/bin/bash
# Find --permission-mode in args
for ((i=1; i<=$#; i++)); do
    if [ "${!i}" == "--permission-mode" ]; then
        j=$((i+1))
        echo "${!j}" >> %s
    fi
done
echo '{"type":"system","subtype":"init"}'
echo '{"type":"result","subtype":"success","result":"Done"}'
`, outputFile))

			config := &ProcessConfig{
				SessionID:      "test-session-mode-" + tt.name,
				WorkingDir:     tmpDir,
				PermissionMode: tt.mode,
				ClaudePath:     mockClaudePath,
			}

			pm := NewProcessManager(config)
			ctx := context.Background()

			err := pm.Start(ctx)
			if err != nil {
				t.Fatalf("Start() error = %v", err)
			}

			time.Sleep(200 * time.Millisecond)
			pm.Stop()

			// Verify mode was passed correctly
			data, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("Could not read output file: %v", err)
			}

			actualMode := strings.TrimSpace(string(data))
			if actualMode != tt.want {
				t.Errorf("Expected permission mode %s, got %s", tt.want, actualMode)
			}
		})
	}
}

// Helper function to create a mock claude script
func createMockClaudeScript(t *testing.T, dir, content string) string {
	t.Helper()
	scriptPath := filepath.Join(dir, "mock-claude.sh")
	if err := os.WriteFile(scriptPath, []byte(content), 0755); err != nil {
		t.Fatalf("Failed to create mock script: %v", err)
	}
	return scriptPath
}

// Helper to pretty print JSON
func prettyJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

// TestIntegrationProcessSignal0 tests using signal 0 to check process status
func TestIntegrationProcessSignal0(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpDir := t.TempDir()

	mockClaudePath := createMockClaudeScript(t, tmpDir, `#!/bin/bash
echo '{"type":"system","subtype":"init"}'
while true; do sleep 0.1; done
`)

	config := &ProcessConfig{
		SessionID:      "test-session-sig0",
		WorkingDir:     tmpDir,
		PermissionMode: agent.PermissionModeDefault,
		ClaudePath:     mockClaudePath,
	}

	pm := NewProcessManager(config)
	ctx := context.Background()

	// Before start, IsRunning should be false
	if pm.IsRunning() {
		t.Error("IsRunning() should be false before start")
	}

	err := pm.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// After start, IsRunning should be true
	if !pm.IsRunning() {
		t.Error("IsRunning() should be true after start")
	}

	// Verify with signal 0 directly
	pm.mu.Lock()
	process := pm.cmd.Process
	pm.mu.Unlock()

	if process != nil {
		err := process.Signal(syscall.Signal(0))
		if err != nil {
			t.Errorf("Signal(0) should succeed for running process: %v", err)
		}
	}

	pm.Stop()

	// After stop, IsRunning should be false
	if pm.IsRunning() {
		t.Error("IsRunning() should be false after stop")
	}
}
