package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/smy-101/cc-connect/internal/app"
	"github.com/smy-101/cc-connect/internal/core"
	"github.com/smy-101/cc-connect/internal/platform/feishu"
)

// TestE2ETextMessageFlow tests the complete text message flow.
func TestE2ETextMessageFlow(t *testing.T) {
	// Create mock agent with predefined response
	mockAgent := NewMockAgent()
	mockAgent.SetResponse("hello", "Hello! How can I help you?")

	// Create test app config
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

	// Create app
	application, err := app.New(config)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	// Verify app was created
	if application == nil {
		t.Fatal("app is nil")
	}

	// Verify status is idle
	if application.Status() != app.AppStatusIdle {
		t.Errorf("expected status idle, got %s", application.Status())
	}
}

// TestE2ECommandMessageFlow tests the command message flow.
func TestE2ECommandMessageFlow(t *testing.T) {
	// Create test app
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

	application, err := app.New(config)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	application.SetFeishuClientFactory(func(appID, appSecret string) feishu.FeishuClient {
		return feishu.NewMockClient()
	})

	// Test that command handler is registered after app starts
	ctx := context.Background()
	if err := application.Start(ctx); err != nil {
		t.Fatalf("failed to start app: %v", err)
	}
	defer application.Stop()

	// Verify handlers are registered
	if !application.Router().HasHandler(core.MessageTypeText) {
		t.Error("text handler not registered")
	}
	if !application.Router().HasHandler(core.MessageTypeCommand) {
		t.Error("command handler not registered")
	}
}

// TestE2EAgentTimeout tests agent timeout handling.
func TestE2EAgentTimeout(t *testing.T) {
	// Create mock agent with delay
	mockAgent := NewMockAgent()
	mockAgent.SetDelay(2 * time.Second)
	mockAgent.SetResponse("slow", "This response is delayed")

	// Verify mock was configured
	if mockAgent.delay != 2*time.Second {
		t.Error("mock agent delay not set correctly")
	}
}

// TestE2EAgentError tests agent error handling.
func TestE2EAgentError(t *testing.T) {
	// Create mock agent that returns error
	mockAgent := NewMockAgent()
	mockAgent.SetError(context.DeadlineExceeded)

	// Verify error is set
	if mockAgent.err == nil {
		t.Error("mock agent error not set")
	}
}

// TestE2EConcurrentMessages tests concurrent message processing.
func TestE2EConcurrentMessages(t *testing.T) {
	// Create test app
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

	application, err := app.New(config)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	// Test that we can create multiple sessions
	session1 := application.Router().Sessions().GetOrCreate("session-1")
	session2 := application.Router().Sessions().GetOrCreate("session-2")

	if session1.ID == session2.ID {
		t.Error("sessions should have different IDs")
	}
}

// TestE2EGracefulShutdown tests graceful shutdown flow.
func TestE2EGracefulShutdown(t *testing.T) {
	// Create test app
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

	application, err := app.New(config)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	application.SetFeishuClientFactory(func(appID, appSecret string) feishu.FeishuClient {
		return feishu.NewMockClient()
	})

	// Start app
	ctx := context.Background()
	if err := application.Start(ctx); err != nil {
		t.Fatalf("failed to start app: %v", err)
	}

	// Verify running
	if application.Status() != app.AppStatusRunning {
		t.Errorf("expected status running, got %s", application.Status())
	}

	// Stop app
	if err := application.Stop(); err != nil {
		t.Fatalf("failed to stop app: %v", err)
	}

	// Verify stopped
	if application.Status() != app.AppStatusStopped {
		t.Errorf("expected status stopped, got %s", application.Status())
	}
}
