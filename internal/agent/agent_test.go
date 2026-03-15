package agent

import (
	"context"
	"testing"
	"time"
)

// TestAgentInterface ensures the Agent interface is satisfied by implementations
func TestAgentInterface(t *testing.T) {
	// This is a compile-time check to ensure the interface is properly defined
	// Real implementations will be tested in their respective packages
	var _ Agent = (*mockAgentForInterfaceTest)(nil)
}

// TestPermissionModeTypes tests PermissionMode type values
func TestPermissionModeTypes(t *testing.T) {
	modes := []PermissionMode{
		PermissionModeDefault,
		PermissionModeAcceptEdits,
		PermissionModePlan,
		PermissionModeBypassPermissions,
	}

	for _, mode := range modes {
		if string(mode) == "" {
			t.Errorf("PermissionMode should not be empty string")
		}
	}
}

// TestAgentStatusTypes tests AgentStatus type values
func TestAgentStatusTypes(t *testing.T) {
	statuses := []AgentStatus{
		AgentStatusIdle,
		AgentStatusStarting,
		AgentStatusRunning,
		AgentStatusStopped,
		AgentStatusError,
	}

	for _, status := range statuses {
		if string(status) == "" {
			t.Errorf("AgentStatus should not be empty string")
		}
	}
}

// TestStreamEventType tests StreamEvent type structure
func TestStreamEventType(t *testing.T) {
	event := StreamEvent{
		Type:    StreamEventTypeText,
		Content: "Hello",
	}

	if event.Type != StreamEventTypeText {
		t.Errorf("expected StreamEventTypeText, got %s", event.Type)
	}
	if event.Content != "Hello" {
		t.Errorf("expected Hello, got %s", event.Content)
	}
}

// TestStreamEventWithTool tests StreamEvent with ToolInfo
func TestStreamEventWithTool(t *testing.T) {
	event := StreamEvent{
		Type: StreamEventTypeToolUse,
		Tool: &ToolInfo{
			Name:  "Bash",
			ID:    "toolu_1",
			Input: map[string]interface{}{"command": "ls"},
		},
	}

	if event.Tool == nil {
		t.Fatal("Tool should not be nil")
	}
	if event.Tool.Name != "Bash" {
		t.Errorf("expected Bash, got %s", event.Tool.Name)
	}
}

// TestResponseType tests Response type structure
func TestResponseType(t *testing.T) {
	resp := Response{
		Content:          "Done",
		IsError:          false,
		PermissionDenied: false,
		CostUSD:          0.0123,
		Duration:         5 * time.Second,
	}

	if resp.Content != "Done" {
		t.Errorf("expected Done, got %s", resp.Content)
	}
	if resp.CostUSD != 0.0123 {
		t.Errorf("expected 0.0123, got %f", resp.CostUSD)
	}
}

// TestResponseWithDeniedTools tests Response with DeniedTools
func TestResponseWithDeniedTools(t *testing.T) {
	resp := Response{
		Content:          "",
		IsError:          true,
		PermissionDenied: true,
		DeniedTools: []DeniedTool{
			{
				ToolName:   "Bash",
				ToolUseID:  "toolu_9",
				ToolInput:  map[string]interface{}{"command": "git fetch"},
			},
		},
	}

	if len(resp.DeniedTools) != 1 {
		t.Fatalf("expected 1 denied tool, got %d", len(resp.DeniedTools))
	}
	if resp.DeniedTools[0].ToolName != "Bash" {
		t.Errorf("expected Bash, got %s", resp.DeniedTools[0].ToolName)
	}
}

// mockAgentForInterfaceTest is a minimal mock for compile-time interface check
type mockAgentForInterfaceTest struct{}

func (m *mockAgentForInterfaceTest) SendMessage(ctx context.Context, content string, handler EventHandler) (*Response, error) {
	return &Response{}, nil
}

func (m *mockAgentForInterfaceTest) SetPermissionMode(mode PermissionMode) error {
	return nil
}

func (m *mockAgentForInterfaceTest) CurrentMode() PermissionMode {
	return PermissionModeDefault
}

func (m *mockAgentForInterfaceTest) SessionID() string {
	return ""
}

func (m *mockAgentForInterfaceTest) Start(ctx context.Context) error {
	return nil
}

func (m *mockAgentForInterfaceTest) Stop() error {
	return nil
}

func (m *mockAgentForInterfaceTest) Status() AgentStatus {
	return AgentStatusIdle
}

func (m *mockAgentForInterfaceTest) Restart(ctx context.Context) error {
	return nil
}
