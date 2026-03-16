package command

import (
	"context"
	"strings"
	"testing"

	"github.com/smy-101/cc-connect/internal/agent"
	"github.com/smy-101/cc-connect/internal/agent/claudecode"
	"github.com/smy-101/cc-connect/internal/core"
)

// newTestExecutor creates an executor with a mock agent for testing
func newTestExecutor() (*Executor, *claudecode.MockAgent, *core.SessionManager) {
	mockAgent := claudecode.NewMockAgent(&claudecode.Config{})
	sessions := core.NewSessionManager(core.DefaultSessionConfig())
	executor := NewExecutor(mockAgent, sessions)
	return executor, mockAgent, sessions
}

func TestExecutorHelp(t *testing.T) {
	executor, _, _ := newTestExecutor()
	ctx := context.Background()
	msg := core.NewCommandMessage("feishu", "user123", "/help")

	cmd := Command{Name: "help", Args: []string{}}
	result := executor.Execute(ctx, cmd, msg)

	if result.IsError() {
		t.Errorf("help command should not return error, got: %v", result.Error)
	}

	// Check that help message contains all expected commands
	expectedCommands := []string{"/mode", "/new", "/list", "/help", "/stop"}
	for _, expectedCmd := range expectedCommands {
		if !strings.Contains(result.Message, expectedCmd) {
			t.Errorf("help message should contain %q, got: %q", expectedCmd, result.Message)
		}
	}
}

func TestExecutorHelpWithArgs(t *testing.T) {
	executor, _, _ := newTestExecutor()
	ctx := context.Background()
	msg := core.NewCommandMessage("feishu", "user123", "/help mode")

	// /help should work even with extra args (ignoring them)
	cmd := Command{Name: "help", Args: []string{"mode"}}
	result := executor.Execute(ctx, cmd, msg)

	if result.IsError() {
		t.Errorf("help command should not return error, got: %v", result.Error)
	}

	// Should still return the general help message
	if !strings.Contains(result.Message, "/mode") {
		t.Errorf("help message should contain /mode, got: %q", result.Message)
	}
}

func TestExecutorUnknownCommand(t *testing.T) {
	executor, _, _ := newTestExecutor()
	ctx := context.Background()
	msg := core.NewCommandMessage("feishu", "user123", "/unknown")

	cmd := Command{Name: "unknown", Args: []string{}}
	result := executor.Execute(ctx, cmd, msg)

	// Should return error
	if !result.IsError() {
		t.Error("unknown command should return error")
	}

	// Should contain "未知命令" message
	if !strings.Contains(result.Message, "未知命令") {
		t.Errorf("unknown command message should contain '未知命令', got: %q", result.Message)
	}

	// Should suggest /help
	if !strings.Contains(result.Message, "/help") {
		t.Errorf("unknown command message should suggest /help, got: %q", result.Message)
	}

	// Error should be ErrUnknownCommand
	if result.Error != ErrUnknownCommand {
		t.Errorf("unknown command error should be ErrUnknownCommand, got: %v", result.Error)
	}
}

func TestExecutorEmptyCommand(t *testing.T) {
	executor, _, _ := newTestExecutor()
	ctx := context.Background()
	msg := core.NewCommandMessage("feishu", "user123", "/")

	// Empty command (only slash)
	cmd := Command{Name: "", Args: []string{}}
	result := executor.Execute(ctx, cmd, msg)

	// Should return error
	if !result.IsError() {
		t.Error("empty command should return error")
	}

	// Error should be ErrEmptyCommand
	if result.Error != ErrEmptyCommand {
		t.Errorf("empty command error should be ErrEmptyCommand, got: %v", result.Error)
	}
}

// /mode command tests

func TestModeYolo(t *testing.T) {
	executor, mockAgent, _ := newTestExecutor()
	ctx := context.Background()
	msg := core.NewCommandMessage("feishu", "user123", "/mode yolo")

	cmd := Command{Name: "mode", Args: []string{"yolo"}}
	result := executor.Execute(ctx, cmd, msg)

	// Should not return error
	if result.IsError() {
		t.Errorf("/mode yolo should not return error, got: %v", result.Error)
	}

	// Should call SetPermissionMode with bypassPermissions
	if mockAgent.CurrentMode() != agent.PermissionModeBypassPermissions {
		t.Errorf("agent mode should be bypassPermissions, got: %v", mockAgent.CurrentMode())
	}

	// Success message should contain "yolo"
	if !strings.Contains(result.Message, "yolo") {
		t.Errorf("result message should contain 'yolo', got: %q", result.Message)
	}
}

func TestModeEdit(t *testing.T) {
	executor, mockAgent, _ := newTestExecutor()
	ctx := context.Background()
	msg := core.NewCommandMessage("feishu", "user123", "/mode edit")

	cmd := Command{Name: "mode", Args: []string{"edit"}}
	result := executor.Execute(ctx, cmd, msg)

	// Should not return error
	if result.IsError() {
		t.Errorf("/mode edit should not return error, got: %v", result.Error)
	}

	// Should call SetPermissionMode with acceptEdits
	if mockAgent.CurrentMode() != agent.PermissionModeAcceptEdits {
		t.Errorf("agent mode should be acceptEdits, got: %v", mockAgent.CurrentMode())
	}
}

func TestModePlan(t *testing.T) {
	executor, mockAgent, _ := newTestExecutor()
	ctx := context.Background()
	msg := core.NewCommandMessage("feishu", "user123", "/mode plan")

	cmd := Command{Name: "mode", Args: []string{"plan"}}
	result := executor.Execute(ctx, cmd, msg)

	// Should not return error
	if result.IsError() {
		t.Errorf("/mode plan should not return error, got: %v", result.Error)
	}

	// Should call SetPermissionMode with plan
	if mockAgent.CurrentMode() != agent.PermissionModePlan {
		t.Errorf("agent mode should be plan, got: %v", mockAgent.CurrentMode())
	}
}

func TestModeDefault(t *testing.T) {
	executor, mockAgent, _ := newTestExecutor()
	ctx := context.Background()
	msg := core.NewCommandMessage("feishu", "user123", "/mode default")

	// First set to something else
	mockAgent.SetPermissionMode(agent.PermissionModeBypassPermissions)

	cmd := Command{Name: "mode", Args: []string{"default"}}
	result := executor.Execute(ctx, cmd, msg)

	// Should not return error
	if result.IsError() {
		t.Errorf("/mode default should not return error, got: %v", result.Error)
	}

	// Should call SetPermissionMode with default
	if mockAgent.CurrentMode() != agent.PermissionModeDefault {
		t.Errorf("agent mode should be default, got: %v", mockAgent.CurrentMode())
	}
}

func TestModeNoArgs(t *testing.T) {
	executor, mockAgent, _ := newTestExecutor()
	ctx := context.Background()
	msg := core.NewCommandMessage("feishu", "user123", "/mode")

	// Set current mode
	mockAgent.SetPermissionMode(agent.PermissionModePlan)

	cmd := Command{Name: "mode", Args: []string{}}
	result := executor.Execute(ctx, cmd, msg)

	// Should not return error
	if result.IsError() {
		t.Errorf("/mode with no args should not return error, got: %v", result.Error)
	}

	// Should return current mode
	if !strings.Contains(result.Message, "plan") {
		t.Errorf("result message should contain 'plan', got: %q", result.Message)
	}
}

func TestModeInvalid(t *testing.T) {
	executor, _, _ := newTestExecutor()
	ctx := context.Background()
	msg := core.NewCommandMessage("feishu", "user123", "/mode invalid")

	cmd := Command{Name: "mode", Args: []string{"invalid"}}
	result := executor.Execute(ctx, cmd, msg)

	// Should return error
	if !result.IsError() {
		t.Error("/mode invalid should return error")
	}

	// Should contain error message
	if !strings.Contains(result.Message, "无效") {
		t.Errorf("result message should contain '无效', got: %q", result.Message)
	}

	// Should list available modes
	if !strings.Contains(result.Message, "default") || !strings.Contains(result.Message, "yolo") {
		t.Errorf("result message should list available modes, got: %q", result.Message)
	}
}

// /new command tests

func TestNewNoName(t *testing.T) {
	executor, _, sessions := newTestExecutor()
	ctx := context.Background()
	msg := core.NewCommandMessage("feishu", "user123", "/new")

	// Create a session first
	sessionID := core.DeriveSessionID(msg)
	sessions.GetOrCreate(sessionID)

	cmd := Command{Name: "new", Args: []string{}}
	result := executor.Execute(ctx, cmd, msg)

	// Should not return error
	if result.IsError() {
		t.Errorf("/new should not return error, got: %v", result.Error)
	}

	// Should return success message
	if !strings.Contains(result.Message, "新会话") {
		t.Errorf("result message should contain '新会话', got: %q", result.Message)
	}

	// New session should exist
	_, exists := sessions.Get(sessionID)
	if !exists {
		t.Error("session should exist after /new command")
	}
}

func TestNewWithName(t *testing.T) {
	executor, _, sessions := newTestExecutor()
	ctx := context.Background()
	msg := core.NewCommandMessage("feishu", "user123", "/new my-project")

	cmd := Command{Name: "new", Args: []string{"my-project"}}
	result := executor.Execute(ctx, cmd, msg)

	// Should not return error
	if result.IsError() {
		t.Errorf("/new my-project should not return error, got: %v", result.Error)
	}

	// Should return success message with name
	if !strings.Contains(result.Message, "my-project") {
		t.Errorf("result message should contain 'my-project', got: %q", result.Message)
	}

	// Session should exist with name in metadata
	sessionID := core.DeriveSessionID(msg)
	session, exists := sessions.Get(sessionID)
	if !exists {
		t.Error("session should exist after /new command")
	}
	if session.Metadata["name"] != "my-project" {
		t.Errorf("session name should be 'my-project', got: %q", session.Metadata["name"])
	}
}

// /list command tests

func TestListMultipleSessions(t *testing.T) {
	executor, _, sessions := newTestExecutor()
	ctx := context.Background()

	// Create multiple sessions
	session1 := sessions.GetOrCreate("feishu:user:user1")
	session1.SetMetadata("name", "session-one")
	sessions.Update("feishu:user:user1", func(s *core.Session) {
		s.SetMetadata("name", "session-one")
	})

	session2 := sessions.GetOrCreate("feishu:user:user2")
	session2.SetMetadata("name", "session-two")
	sessions.Update("feishu:user:user2", func(s *core.Session) {
		s.SetMetadata("name", "session-two")
	})

	msg := core.NewCommandMessage("feishu", "user123", "/list")
	cmd := Command{Name: "list", Args: []string{}}
	result := executor.Execute(ctx, cmd, msg)

	// Should not return error
	if result.IsError() {
		t.Errorf("/list should not return error, got: %v", result.Error)
	}

	// Should contain both session names
	if !strings.Contains(result.Message, "session-one") {
		t.Errorf("result message should contain 'session-one', got: %q", result.Message)
	}
	if !strings.Contains(result.Message, "session-two") {
		t.Errorf("result message should contain 'session-two', got: %q", result.Message)
	}
}

func TestListEmpty(t *testing.T) {
	executor, _, _ := newTestExecutor()
	ctx := context.Background()
	msg := core.NewCommandMessage("feishu", "user123", "/list")

	cmd := Command{Name: "list", Args: []string{}}
	result := executor.Execute(ctx, cmd, msg)

	// Should not return error
	if result.IsError() {
		t.Errorf("/list should not return error, got: %v", result.Error)
	}

	// Should indicate no sessions
	if !strings.Contains(result.Message, "没有") {
		t.Errorf("result message should indicate no sessions, got: %q", result.Message)
	}
}

// /stop command tests

func TestStopRunningAgent(t *testing.T) {
	executor, mockAgent, _ := newTestExecutor()
	ctx := context.Background()

	// Start the agent first
	mockAgent.Start(ctx)

	msg := core.NewCommandMessage("feishu", "user123", "/stop")
	cmd := Command{Name: "stop", Args: []string{}}
	result := executor.Execute(ctx, cmd, msg)

	// Should not return error
	if result.IsError() {
		t.Errorf("/stop should not return error, got: %v", result.Error)
	}

	// Agent should be stopped
	if mockAgent.Status() != agent.AgentStatusStopped {
		t.Errorf("agent should be stopped, got: %v", mockAgent.Status())
	}

	// Success message
	if !strings.Contains(result.Message, "停止") {
		t.Errorf("result message should contain '停止', got: %q", result.Message)
	}
}

func TestStopIdleAgent(t *testing.T) {
	executor, _, _ := newTestExecutor()
	ctx := context.Background()

	// Agent is idle (not started)
	msg := core.NewCommandMessage("feishu", "user123", "/stop")
	cmd := Command{Name: "stop", Args: []string{}}
	result := executor.Execute(ctx, cmd, msg)

	// Should not return error
	if result.IsError() {
		t.Errorf("/stop should not return error when agent is idle, got: %v", result.Error)
	}

	// Should indicate agent not running
	if !strings.Contains(result.Message, "未运行") {
		t.Errorf("result message should indicate agent not running, got: %q", result.Message)
	}
}
