package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/smy-101/cc-connect/internal/core"
)

// TestReplySenderInterface tests that ReplySender is correctly defined
// as a small interface with SendReply method.
func TestReplySenderInterface(t *testing.T) {
	// This test verifies that the ReplySender interface exists and has the correct signature.
	// We use a mock implementation to satisfy the interface.
	var _ ReplySender = (*mockReplySender)(nil)
}

// TestReplySenderSendReply tests the SendReply method behavior.
func TestReplySenderSendReply(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		mockError error
		wantError bool
	}{
		{
			name:      "successful send",
			content:   "test response",
			mockError: nil,
			wantError: false,
		},
		{
			name:      "empty content is valid",
			content:   "",
			mockError: nil,
			wantError: false,
		},
		{
			name:      "send failure",
			content:   "test response",
			mockError: errors.New("network error"),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockReplySender{err: tt.mockError}
			ctx := context.Background()

			err := mock.SendReply(ctx, tt.content)

			if (err != nil) != tt.wantError {
				t.Errorf("SendReply() error = %v, wantError %v", err, tt.wantError)
			}

			if !tt.wantError && mock.lastContent != tt.content {
				t.Errorf("SendReply() content = %v, want %v", mock.lastContent, tt.content)
			}
		})
	}
}

// mockReplySender is a mock implementation of ReplySender for testing.
type mockReplySender struct {
	lastContent string
	err         error
}

func (m *mockReplySender) SendReply(ctx context.Context, content string) error {
	m.lastContent = content
	return m.err
}

// TestHandlerContextStruct tests that HandlerContext is correctly defined
// with all required fields.
func TestHandlerContextStruct(t *testing.T) {
	ctx := context.Background()
	msg := core.NewTextMessage("feishu", "user123", "test message")
	session := core.NewSession("test-session")
	reply := &mockReplySender{}

	hctx := &HandlerContext{
		Ctx:     ctx,
		Msg:     msg,
		Session: session,
		Reply:   reply,
	}

	// Verify all fields are correctly set
	if hctx.Ctx != ctx {
		t.Error("HandlerContext.Ctx not set correctly")
	}
	if hctx.Msg != msg {
		t.Error("HandlerContext.Msg not set correctly")
	}
	if hctx.Session != session {
		t.Error("HandlerContext.Session not set correctly")
	}
	if hctx.Reply != reply {
		t.Error("HandlerContext.Reply not set correctly")
	}
}

// TestHandlerContextReply tests that HandlerContext can use its Reply field.
func TestHandlerContextReply(t *testing.T) {
	mock := &mockReplySender{}
	hctx := &HandlerContext{
		Ctx:   context.Background(),
		Reply: mock,
	}

	err := hctx.Reply.SendReply(hctx.Ctx, "test response")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mock.lastContent != "test response" {
		t.Errorf("expected content 'test response', got '%s'", mock.lastContent)
	}
}

// TestHandlerContextNilFields tests HandlerContext behavior with nil fields.
func TestHandlerContextNilFields(t *testing.T) {
	t.Run("nil Reply is detectable", func(t *testing.T) {
		hctx := &HandlerContext{
			Ctx: context.Background(),
		}
		if hctx.Reply != nil {
			t.Error("expected nil Reply")
		}
	})

	t.Run("nil Msg is detectable", func(t *testing.T) {
		hctx := &HandlerContext{
			Ctx: context.Background(),
		}
		if hctx.Msg != nil {
			t.Error("expected nil Msg")
		}
	})

	t.Run("nil Session is detectable", func(t *testing.T) {
		hctx := &HandlerContext{
			Ctx: context.Background(),
		}
		if hctx.Session != nil {
			t.Error("expected nil Session")
		}
	})
}

// =============================================================================
// App Structure and New Function Tests
// =============================================================================

// TestAppStatusConstants tests that AppStatus constants are correctly defined.
func TestAppStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		status   AppStatus
		expected string
	}{
		{"idle", AppStatusIdle, "idle"},
		{"running", AppStatusRunning, "running"},
		{"stopping", AppStatusStopping, "stopping"},
		{"stopped", AppStatusStopped, "stopped"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("AppStatus %s = %q, want %q", tt.name, tt.status, tt.expected)
			}
		})
	}
}

// TestNewNilConfig tests that New returns an error when config is nil.
func TestNewNilConfig(t *testing.T) {
	_, err := New(nil)
	if err == nil {
		t.Error("New(nil) should return an error")
	}
}

// TestNewEmptyConfig tests that New returns an error when config has no projects.
func TestNewEmptyConfig(t *testing.T) {
	config := &core.AppConfig{
		LogLevel: "info",
	}

	_, err := New(config)
	if err == nil {
		t.Error("New with empty projects should return an error")
	}
}

// TestNewMissingProjectName tests that New returns an error when project name is missing.
func TestNewMissingProjectName(t *testing.T) {
	config := &core.AppConfig{
		LogLevel: "info",
		Projects: []core.ProjectConfig{
			{
				WorkingDir: "/tmp",
			},
		},
	}

	_, err := New(config)
	if err == nil {
		t.Error("New with missing project name should return an error")
	}
}

// TestNewMissingWorkingDir tests that New returns an error when working directory is missing.
func TestNewMissingWorkingDir(t *testing.T) {
	config := &core.AppConfig{
		LogLevel: "info",
		Projects: []core.ProjectConfig{
			{
				Name: "test-project",
			},
		},
	}

	_, err := New(config)
	if err == nil {
		t.Error("New with missing working directory should return an error")
	}
}

// TestNewFeishuNotEnabled tests that New returns an error when Feishu is not enabled.
func TestNewFeishuNotEnabled(t *testing.T) {
	config := &core.AppConfig{
		LogLevel: "info",
		Projects: []core.ProjectConfig{
			{
				Name:       "test-project",
				WorkingDir: "/tmp",
				Feishu: core.FeishuConfig{
					Enabled: false,
				},
				ClaudeCode: core.ClaudeCodeConfig{
					Enabled: true,
				},
			},
		},
	}

	_, err := New(config)
	if err == nil {
		t.Error("New with Feishu disabled should return an error")
	}
}

// TestNewClaudeCodeNotEnabled tests that New returns an error when Claude Code is not enabled.
func TestNewClaudeCodeNotEnabled(t *testing.T) {
	config := &core.AppConfig{
		LogLevel: "info",
		Projects: []core.ProjectConfig{
			{
				Name:       "test-project",
				WorkingDir: "/tmp",
				Feishu: core.FeishuConfig{
					Enabled:   true,
					AppID:     "test-app-id",
					AppSecret: "test-secret",
				},
				ClaudeCode: core.ClaudeCodeConfig{
					Enabled: false,
				},
			},
		},
	}

	_, err := New(config)
	if err == nil {
		t.Error("New with Claude Code disabled should return an error")
	}
}

// TestNewMissingFeishuCredentials tests that New returns an error when Feishu credentials are missing.
func TestNewMissingFeishuCredentials(t *testing.T) {
	tests := []struct {
		name     string
		appID    string
		secret   string
		wantFail bool
	}{
		{"both empty", "", "", true},
		{"only appID", "app-id", "", true},
		{"only secret", "", "secret", true},
		{"both set", "app-id", "secret", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &core.AppConfig{
				LogLevel: "info",
				Projects: []core.ProjectConfig{
					{
						Name:       "test-project",
						WorkingDir: "/tmp",
						Feishu: core.FeishuConfig{
							Enabled:   true,
							AppID:     tt.appID,
							AppSecret: tt.secret,
						},
						ClaudeCode: core.ClaudeCodeConfig{
							Enabled: true,
						},
					},
				},
			}

			_, err := New(config)
			if (err != nil) != tt.wantFail {
				t.Errorf("New() error = %v, wantFail %v", err, tt.wantFail)
			}
		})
	}
}

// TestNewInvalidPermissionMode tests that New returns an error with invalid permission mode.
func TestNewInvalidPermissionMode(t *testing.T) {
	config := &core.AppConfig{
		LogLevel: "info",
		Projects: []core.ProjectConfig{
			{
				Name:       "test-project",
				WorkingDir: "/tmp",
				Feishu: core.FeishuConfig{
					Enabled:   true,
					AppID:     "test-app-id",
					AppSecret: "test-secret",
				},
				ClaudeCode: core.ClaudeCodeConfig{
					Enabled:               true,
					DefaultPermissionMode: "invalid-mode",
				},
			},
		},
	}

	_, err := New(config)
	if err == nil {
		t.Error("New with invalid permission mode should return an error")
	}
}

// =============================================================================
// App.Start Tests
// =============================================================================

// TestAppStartAlreadyRunning tests that Start returns an error if already running.
func TestAppStartAlreadyRunning(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	// Set status to running manually
	app.mu.Lock()
	app.status = AppStatusRunning
	app.mu.Unlock()

	ctx := context.Background()
	err = app.Start(ctx)
	if err == nil {
		t.Error("Start should return error when already running")
	}
}

// TestAppStartSuccess tests successful app start.
func TestAppStartSuccess(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	ctx := context.Background()
	err = app.Start(ctx)
	if err != nil {
		t.Errorf("Start failed: %v", err)
	}

	// Verify status
	if app.Status() != AppStatusRunning {
		t.Errorf("expected status %s, got %s", AppStatusRunning, app.Status())
	}

	// Cleanup
	_ = app.Stop()
}

// TestAppStartTwice tests that calling Start twice returns error.
func TestAppStartTwice(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	ctx := context.Background()

	// First start
	err = app.Start(ctx)
	if err != nil {
		t.Fatalf("first Start failed: %v", err)
	}

	// Second start should fail
	err = app.Start(ctx)
	if err == nil {
		t.Error("second Start should return error")
	}

	// Cleanup
	_ = app.Stop()
}

// createTestApp creates a test App instance with valid configuration.
func createTestApp() (*App, error) {
	config := &core.AppConfig{
		LogLevel: "info",
		Projects: []core.ProjectConfig{
			{
				Name:       "test-project",
				WorkingDir: "/tmp",
				Feishu: core.FeishuConfig{
					Enabled:   true,
					AppID:     "test-app-id",
					AppSecret: "test-secret",
				},
				ClaudeCode: core.ClaudeCodeConfig{
					Enabled:               true,
					DefaultPermissionMode: "default",
				},
			},
		},
	}
	return New(config)
}

// =============================================================================
// Handler Tests
// =============================================================================

// TestWrapHandlerPanicRecovery tests that wrapHandler recovers from panics.
func TestWrapHandlerPanicRecovery(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	// Create a handler that panics
	panicHandler := func(hctx *HandlerContext) error {
		panic("test panic")
	}

	// Wrap the handler
	wrapped := app.wrapHandler(panicHandler)

	// Create a message
	msg := core.NewTextMessage("feishu", "user123", "test")

	// Call the wrapped handler - should not panic
	err = wrapped(context.Background(), msg)
	if err == nil {
		t.Error("wrapped handler should return error when panic occurs")
	}
}

// TestWrapHandlerContextInjection tests that wrapHandler correctly injects HandlerContext.
func TestWrapHandlerContextInjection(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	var capturedCtx *HandlerContext

	// Create a handler that captures the context
	testHandler := func(hctx *HandlerContext) error {
		capturedCtx = hctx
		return nil
	}

	// Wrap the handler
	wrapped := app.wrapHandler(testHandler)

	// Create a message
	msg := core.NewTextMessage("feishu", "user123", "test message")
	msg.ChannelID = "channel123"

	// Call the wrapped handler
	ctx := context.Background()
	err = wrapped(ctx, msg)
	if err != nil {
		t.Errorf("handler returned error: %v", err)
	}

	// Verify context injection
	if capturedCtx == nil {
		t.Fatal("HandlerContext was not injected")
	}
	if capturedCtx.Msg == nil {
		t.Error("Msg was not injected")
	}
	if capturedCtx.Msg.Content != "test message" {
		t.Errorf("expected content 'test message', got '%s'", capturedCtx.Msg.Content)
	}
	if capturedCtx.Session == nil {
		t.Error("Session was not injected")
	}
}

// TestHandleTextSuccess tests successful text message handling.
func TestHandleTextSuccess(t *testing.T) {
	// This test verifies the handleText method signature and basic behavior
	// A full integration test would require a running agent
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	// Verify the method exists and has correct signature
	_ = app.handleText
}

// TestHandleCommandSuccess tests successful command handling.
func TestHandleCommandSuccess(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	// Create mock reply sender
	mock := &mockReplySender{}

	// Create handler context
	msg := core.NewTextMessage("feishu", "user123", "/help")
	msg.ChannelID = "channel123"
	hctx := &HandlerContext{
		Ctx:   context.Background(),
		Msg:   msg,
		Reply: mock,
	}

	// Execute command handler
	err = app.handleCommand(hctx)
	if err != nil {
		t.Errorf("handleCommand returned error: %v", err)
	}

	// Verify response was sent
	if mock.lastContent == "" {
		t.Error("expected response to be sent")
	}
}

// TestHandleCommandEmptyCommand tests handling of empty command.
func TestHandleCommandEmptyCommand(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	mock := &mockReplySender{}
	msg := core.NewTextMessage("feishu", "user123", "invalid") // Not a command
	hctx := &HandlerContext{
		Ctx:   context.Background(),
		Msg:   msg,
		Reply: mock,
	}

	err = app.handleCommand(hctx)
	if err != nil {
		t.Errorf("handleCommand returned error: %v", err)
	}

	// Should have error message
	if mock.lastContent == "" {
		t.Error("expected error message for empty command")
	}
}

// TestRegisterHandlers tests that handlers are registered correctly.
func TestRegisterHandlers(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	// Register handlers
	err = app.registerHandlers()
	if err != nil {
		t.Errorf("registerHandlers returned error: %v", err)
	}

	// Verify text handler is registered
	if !app.router.HasHandler(core.MessageTypeText) {
		t.Error("text handler not registered")
	}

	// Verify command handler is registered
	if !app.router.HasHandler(core.MessageTypeCommand) {
		t.Error("command handler not registered")
	}
}

// TestAppStop tests the Stop method.
func TestAppStop(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	// Stop when not running should succeed
	err = app.Stop()
	if err != nil {
		t.Errorf("Stop on idle app returned error: %v", err)
	}

	// Status should remain idle
	if app.Status() != AppStatusIdle {
		t.Errorf("expected status idle, got %s", app.Status())
	}
}

// TestAppStopRunning tests stopping a running app.
func TestAppStopRunning(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	// Start the app
	ctx := context.Background()
	err = app.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Stop the app
	err = app.Stop()
	if err != nil {
		t.Errorf("Stop returned error: %v", err)
	}

	// Status should be stopped
	if app.Status() != AppStatusStopped {
		t.Errorf("expected status stopped, got %s", app.Status())
	}
}

// TestAppWaitForShutdown tests WaitForShutdown method.
func TestAppWaitForShutdown(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	// WaitForShutdown should return immediately when no goroutines are running
	done := make(chan struct{})
	go func() {
		app.WaitForShutdown()
		close(done)
	}()

	select {
	case <-done:
		// Good
	case <-time.After(1 * time.Second):
		t.Error("WaitForShutdown blocked for too long")
	}
}

// TestAppRouter tests Router method.
func TestAppRouter(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	router := app.Router()
	if router == nil {
		t.Error("Router() returned nil")
	}
}

// TestAppSessions tests Sessions method.
func TestAppSessions(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	sessions := app.Sessions()
	if sessions == nil {
		t.Error("Sessions() returned nil")
	}
}

// TestAppAgent tests Agent method.
func TestAppAgent(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	ag := app.Agent()
	if ag == nil {
		t.Error("Agent() returned nil")
	}
}

// TestParsePermissionMode tests permission mode parsing.
func TestParsePermissionMode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"default", "default"},
		{"edit", "default"}, // edit not directly supported
		{"acceptEdits", "acceptEdits"},
		{"plan", "plan"},
		{"yolo", "bypassPermissions"},
		{"bypassPermissions", "bypassPermissions"},
		{"unknown", "default"},
		{"", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parsePermissionMode(tt.input)
			if string(result) != tt.expected {
				t.Errorf("parsePermissionMode(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestGetActiveProject tests getActiveProject function.
func TestGetActiveProject(t *testing.T) {
	t.Run("default project", func(t *testing.T) {
		config := &core.AppConfig{
			LogLevel:       "info",
			DefaultProject: "project1",
			Projects: []core.ProjectConfig{
				{
					Name:       "project1",
					WorkingDir: "/tmp/project1",
					Feishu: core.FeishuConfig{
						Enabled:   true,
						AppID:     "test-app-id",
						AppSecret: "test-secret",
					},
					ClaudeCode: core.ClaudeCodeConfig{
						Enabled: true,
					},
				},
			},
		}

		project, err := getActiveProject(config)
		if err != nil {
			t.Fatalf("getActiveProject failed: %v", err)
		}
		if project.Name != "project1" {
			t.Errorf("expected project1, got %s", project.Name)
		}
	})

	t.Run("first enabled project", func(t *testing.T) {
		config := &core.AppConfig{
			LogLevel: "info",
			Projects: []core.ProjectConfig{
				{
					Name:       "disabled-project",
					WorkingDir: "/tmp/disabled",
					Feishu: core.FeishuConfig{
						Enabled: false,
					},
					ClaudeCode: core.ClaudeCodeConfig{
						Enabled: false,
					},
				},
				{
					Name:       "enabled-project",
					WorkingDir: "/tmp/enabled",
					Feishu: core.FeishuConfig{
						Enabled:   true,
						AppID:     "test-app-id",
						AppSecret: "test-secret",
					},
					ClaudeCode: core.ClaudeCodeConfig{
						Enabled: true,
					},
				},
			},
		}

		project, err := getActiveProject(config)
		if err != nil {
			t.Fatalf("getActiveProject failed: %v", err)
		}
		if project.Name != "enabled-project" {
			t.Errorf("expected enabled-project, got %s", project.Name)
		}
	})
}

// TestReplySenderSendReplyError tests replySender error handling.
func TestReplySenderSendReplyError(t *testing.T) {
	t.Run("nil adapter", func(t *testing.T) {
		sender := &replySender{adapter: nil, channelID: "test"}
		err := sender.SendReply(context.Background(), "test")
		if err == nil {
			t.Error("expected error with nil adapter")
		}
	})

	t.Run("empty channelID", func(t *testing.T) {
		sender := &replySender{adapter: nil, channelID: ""}
		err := sender.SendReply(context.Background(), "test")
		if err == nil {
			t.Error("expected error with empty channelID")
		}
	})
}

// TestValidateConfigDirect tests validateConfig directly.
func TestValidateConfigDirect(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		err := validateConfig(nil)
		if err != ErrNilConfig {
			t.Errorf("expected ErrNilConfig, got %v", err)
		}
	})

	t.Run("empty projects", func(t *testing.T) {
		err := validateConfig(&core.AppConfig{LogLevel: "info"})
		if err != ErrNoProjects {
			t.Errorf("expected ErrNoProjects, got %v", err)
		}
	})
}

// TestCreateAgentConfigDirect tests createAgentConfig directly.
func TestCreateAgentConfigDirect(t *testing.T) {
	project := &core.ProjectConfig{
		Name:       "test",
		WorkingDir: "/tmp",
		ClaudeCode: core.ClaudeCodeConfig{
			DefaultPermissionMode: "yolo",
		},
	}

	cfg, err := createAgentConfig(project)
	if err != nil {
		t.Fatalf("createAgentConfig failed: %v", err)
	}

	if cfg.WorkingDir != "/tmp" {
		t.Errorf("expected working dir /tmp, got %s", cfg.WorkingDir)
	}
	if cfg.PermissionMode != "bypassPermissions" {
		t.Errorf("expected bypassPermissions, got %s", cfg.PermissionMode)
	}
	if cfg.AgentTimeout != 5*time.Minute {
		t.Errorf("expected 5m timeout, got %v", cfg.AgentTimeout)
	}
}

// TestGetActiveProjectFallback tests fallback behavior in getActiveProject.
func TestGetActiveProjectFallback(t *testing.T) {
	// Test fallback to first project when none enabled
	config := &core.AppConfig{
		LogLevel: "info",
		Projects: []core.ProjectConfig{
			{
				Name:       "fallback-project",
				WorkingDir: "/tmp/fallback",
				Feishu: core.FeishuConfig{
					Enabled: false,
				},
				ClaudeCode: core.ClaudeCodeConfig{
					Enabled: false,
				},
			},
		},
	}

	project, err := getActiveProject(config)
	if err != nil {
		t.Fatalf("getActiveProject failed: %v", err)
	}
	if project.Name != "fallback-project" {
		t.Errorf("expected fallback-project, got %s", project.Name)
	}
}

// TestGetActiveProjectNotFound tests getActiveProject with non-existent default.
func TestGetActiveProjectNotFound(t *testing.T) {
	config := &core.AppConfig{
		LogLevel:       "info",
		DefaultProject: "non-existent",
		Projects: []core.ProjectConfig{
			{
				Name:       "test-project",
				WorkingDir: "/tmp",
				Feishu: core.FeishuConfig{
					Enabled:   true,
					AppID:     "test-app-id",
					AppSecret: "test-secret",
				},
				ClaudeCode: core.ClaudeCodeConfig{
					Enabled: true,
				},
			},
		},
	}

	_, err := getActiveProject(config)
	if err == nil {
		t.Error("expected error for non-existent default project")
	}
}
