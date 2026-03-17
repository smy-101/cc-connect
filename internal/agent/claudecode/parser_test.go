package claudecode

import (
	"testing"

	"github.com/smy-101/cc-connect/internal/agent"
)

// TestParseEvent tests parsing of various JSONL event types
func TestParseEvent(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantType  string
		wantError bool
	}{
		{
			name:     "system/init event",
			input:    `{"type":"system","subtype":"init","session_id":"abc123","cwd":"/repo","model":"sonnet","permissionMode":"auto","tools":["Bash","Read","Write"]}`,
			wantType: "system",
		},
		{
			name:     "assistant text event",
			input:    `{"type":"assistant","session_id":"abc123","message":{"content":[{"type":"text","text":"Hello, how can I help?"}]}}`,
			wantType: "assistant",
		},
		{
			name:     "assistant tool_use event",
			input:    `{"type":"assistant","session_id":"abc123","message":{"content":[{"type":"tool_use","id":"toolu_1","name":"Bash","input":{"command":"ls -la"}}]}}`,
			wantType: "assistant",
		},
		{
			name:     "user tool_result event",
			input:    `{"type":"user","session_id":"abc123","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"file1.go\nfile2.go"}]}}`,
			wantType: "user",
		},
		{
			name:     "result/success event",
			input:    `{"type":"result","subtype":"success","session_id":"abc123","result":"Task completed.","total_cost_usd":0.0123,"duration_ms":12345}`,
			wantType: "result",
		},
		{
			name:     "result/error event with permission_denials",
			input:    `{"type":"result","subtype":"error","error":"Permission denied","permission_denials":[{"tool_name":"Bash","tool_use_id":"toolu_9","tool_input":{"command":"git fetch origin main"}}]}`,
			wantType: "result",
		},
		{
			name:      "invalid JSON",
			input:     `{not valid json`,
			wantError: true,
		},
		{
			name:      "empty input",
			input:     ``,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := ParseEvent([]byte(tt.input))
			if tt.wantError {
				if err == nil {
					t.Errorf("ParseEvent() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("ParseEvent() unexpected error: %v", err)
				return
			}
			if event.Type != tt.wantType {
				t.Errorf("ParseEvent() type = %v, want %v", event.Type, tt.wantType)
			}
		})
	}
}

// TestParseSystemInitEvent tests parsing of system/init events
func TestParseSystemInitEvent(t *testing.T) {
	input := `{"type":"system","subtype":"init","session_id":"abc123","cwd":"/repo","model":"sonnet","permissionMode":"auto","tools":["Bash","Read","Write"]}`

	event, err := ParseEvent([]byte(input))
	if err != nil {
		t.Fatalf("ParseEvent() error = %v", err)
	}

	if event.Type != "system" {
		t.Errorf("expected type system, got %s", event.Type)
	}
	if event.Subtype != "init" {
		t.Errorf("expected subtype init, got %s", event.Subtype)
	}
	if event.SessionID != "abc123" {
		t.Errorf("expected session_id abc123, got %s", event.SessionID)
	}
	if event.CWD != "/repo" {
		t.Errorf("expected cwd /repo, got %s", event.CWD)
	}
	if event.Model != "sonnet" {
		t.Errorf("expected model sonnet, got %s", event.Model)
	}
	if len(event.Tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(event.Tools))
	}
}

// TestParseAssistantTextEvent tests parsing of assistant text events
func TestParseAssistantTextEvent(t *testing.T) {
	input := `{"type":"assistant","session_id":"abc123","message":{"content":[{"type":"text","text":"Hello, how can I help?"}]}}`

	event, err := ParseEvent([]byte(input))
	if err != nil {
		t.Fatalf("ParseEvent() error = %v", err)
	}

	if event.Type != "assistant" {
		t.Errorf("expected type assistant, got %s", event.Type)
	}
	if event.SessionID != "abc123" {
		t.Errorf("expected session_id abc123, got %s", event.SessionID)
	}
	if len(event.Message.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(event.Message.Content))
	}
	if event.Message.Content[0].Type != "text" {
		t.Errorf("expected content type text, got %s", event.Message.Content[0].Type)
	}
	if event.Message.Content[0].Text != "Hello, how can I help?" {
		t.Errorf("expected text 'Hello, how can I help?', got %s", event.Message.Content[0].Text)
	}
}

// TestParseAssistantToolUseEvent tests parsing of assistant tool_use events
func TestParseAssistantToolUseEvent(t *testing.T) {
	input := `{"type":"assistant","session_id":"abc123","message":{"content":[{"type":"tool_use","id":"toolu_1","name":"Bash","input":{"command":"ls -la"}}]}}`

	event, err := ParseEvent([]byte(input))
	if err != nil {
		t.Fatalf("ParseEvent() error = %v", err)
	}

	if len(event.Message.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(event.Message.Content))
	}
	content := event.Message.Content[0]
	if content.Type != "tool_use" {
		t.Errorf("expected content type tool_use, got %s", content.Type)
	}
	if content.ID != "toolu_1" {
		t.Errorf("expected id toolu_1, got %s", content.ID)
	}
	if content.Name != "Bash" {
		t.Errorf("expected name Bash, got %s", content.Name)
	}
	if content.Input == nil {
		t.Error("expected input to be non-nil")
	}
}

// TestParseUserToolResultEvent tests parsing of user tool_result events
func TestParseUserToolResultEvent(t *testing.T) {
	input := `{"type":"user","session_id":"abc123","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"file1.go\nfile2.go"}]}}`

	event, err := ParseEvent([]byte(input))
	if err != nil {
		t.Fatalf("ParseEvent() error = %v", err)
	}

	if event.Type != "user" {
		t.Errorf("expected type user, got %s", event.Type)
	}
	if len(event.Message.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(event.Message.Content))
	}
	content := event.Message.Content[0]
	if content.Type != "tool_result" {
		t.Errorf("expected content type tool_result, got %s", content.Type)
	}
	if content.ToolUseID != "toolu_1" {
		t.Errorf("expected tool_use_id toolu_1, got %s", content.ToolUseID)
	}
	if content.Content != "file1.go\nfile2.go" {
		t.Errorf("expected content 'file1.go\\nfile2.go', got %s", content.Content)
	}
}

// TestParseResultSuccessEvent tests parsing of result/success events
func TestParseResultSuccessEvent(t *testing.T) {
	input := `{"type":"result","subtype":"success","session_id":"abc123","result":"Task completed.","total_cost_usd":0.0123,"duration_ms":12345}`

	event, err := ParseEvent([]byte(input))
	if err != nil {
		t.Fatalf("ParseEvent() error = %v", err)
	}

	if event.Type != "result" {
		t.Errorf("expected type result, got %s", event.Type)
	}
	if event.Subtype != "success" {
		t.Errorf("expected subtype success, got %s", event.Subtype)
	}
	if event.Result != "Task completed." {
		t.Errorf("expected result 'Task completed.', got %s", event.Result)
	}
	if event.TotalCostUSD != 0.0123 {
		t.Errorf("expected total_cost_usd 0.0123, got %f", event.TotalCostUSD)
	}
	if event.DurationMs != 12345 {
		t.Errorf("expected duration_ms 12345, got %d", event.DurationMs)
	}
}

// TestParseResultErrorWithPermissionDenials tests parsing of result/error with permission_denials
func TestParseResultErrorWithPermissionDenials(t *testing.T) {
	input := `{"type":"result","subtype":"error","error":"Permission denied","permission_denials":[{"tool_name":"Bash","tool_use_id":"toolu_9","tool_input":{"command":"git fetch origin main"}}]}`

	event, err := ParseEvent([]byte(input))
	if err != nil {
		t.Fatalf("ParseEvent() error = %v", err)
	}

	if event.Type != "result" {
		t.Errorf("expected type result, got %s", event.Type)
	}
	if event.Subtype != "error" {
		t.Errorf("expected subtype error, got %s", event.Subtype)
	}
	if event.Error != "Permission denied" {
		t.Errorf("expected error 'Permission denied', got %s", event.Error)
	}
	if len(event.PermissionDenials) != 1 {
		t.Fatalf("expected 1 permission_denial, got %d", len(event.PermissionDenials))
	}
	denial := event.PermissionDenials[0]
	if denial.ToolName != "Bash" {
		t.Errorf("expected tool_name Bash, got %s", denial.ToolName)
	}
	if denial.ToolUseID != "toolu_9" {
		t.Errorf("expected tool_use_id toolu_9, got %s", denial.ToolUseID)
	}
}

// TestStreamParser tests the streaming parser
func TestStreamParser(t *testing.T) {
	parser := NewStreamParser()

	// Simulate receiving data in chunks
	lines := []string{
		`{"type":"system","subtype":"init","session_id":"abc123"}`,
		`{"type":"assistant","session_id":"abc123","message":{"content":[{"type":"text","text":"Hello"}]}}`,
		`{"type":"result","subtype":"success","result":"Done"}`,
	}

	var events []*StreamEvent
	for _, line := range lines {
		parsed, err := parser.Parse([]byte(line + "\n"))
		if err != nil {
			t.Errorf("Parse() error = %v", err)
			continue
		}
		events = append(events, parsed...)
	}

	if len(events) != 3 {
		t.Errorf("expected 3 events, got %d", len(events))
	}
}

// TestStreamParserIncompleteLine tests handling of incomplete lines
func TestStreamParserIncompleteLine(t *testing.T) {
	parser := NewStreamParser()

	// Send incomplete JSON
	parsed, err := parser.Parse([]byte(`{"type":"system"`))
	if err != nil {
		t.Errorf("Parse() unexpected error: %v", err)
	}
	if len(parsed) != 0 {
		t.Errorf("expected 0 events for incomplete line, got %d", len(parsed))
	}

	// Complete the line
	parsed, err = parser.Parse([]byte(`,"subtype":"init","session_id":"abc123"}` + "\n"))
	if err != nil {
		t.Errorf("Parse() error = %v", err)
	}
	if len(parsed) != 1 {
		t.Errorf("expected 1 event after completing line, got %d", len(parsed))
	}
}

// TestConvertToAgentEvent tests conversion from StreamEvent to agent.StreamEvent
func TestConvertToAgentEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    *StreamEvent
		wantType agent.StreamEventType
	}{
		{
			name: "system init",
			event: &StreamEvent{
				Type:    "system",
				Subtype: "init",
			},
			wantType: agent.StreamEventTypeSystem,
		},
		{
			name: "assistant text",
			event: &StreamEvent{
				Type: "assistant",
				Message: Message{
					Content: []Content{{Type: "text", Text: "Hello"}},
				},
			},
			wantType: agent.StreamEventTypeText,
		},
		{
			name: "assistant tool_use",
			event: &StreamEvent{
				Type: "assistant",
				Message: Message{
					Content: []Content{{Type: "tool_use", Name: "Bash", ID: "toolu_1"}},
				},
			},
			wantType: agent.StreamEventTypeToolUse,
		},
		{
			name: "user tool_result",
			event: &StreamEvent{
				Type: "user",
				Message: Message{
					Content: []Content{{Type: "tool_result"}},
				},
			},
			wantType: agent.StreamEventTypeToolResult,
		},
		{
			name: "result success",
			event: &StreamEvent{
				Type:    "result",
				Subtype: "success",
				Result:  "Done",
			},
			wantType: agent.StreamEventTypeResult,
		},
		{
			name: "result error",
			event: &StreamEvent{
				Type:    "result",
				Subtype: "error",
				Error:   "Something went wrong",
			},
			wantType: agent.StreamEventTypeError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ae := convertToAgentEvent(tt.event)
			if ae.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", ae.Type, tt.wantType)
			}
		})
	}
}

// TestStreamEventHasPermissionDenials tests the HasPermissionDenials method
func TestStreamEventHasPermissionDenials(t *testing.T) {
	tests := []struct {
		name       string
		event      *StreamEvent
		wantResult bool
	}{
		{
			name: "with permission denials",
			event: &StreamEvent{
				Type:    "result",
				Subtype: "error",
				PermissionDenials: []PermissionDenial{
					{ToolName: "Bash", ToolUseID: "toolu_1"},
				},
			},
			wantResult: true,
		},
		{
			name: "without permission denials",
			event: &StreamEvent{
				Type:    "result",
				Subtype: "error",
				Error:   "Some error",
			},
			wantResult: false,
		},
		{
			name: "empty permission denials slice",
			event: &StreamEvent{
				Type:              "result",
				Subtype:           "error",
				PermissionDenials: []PermissionDenial{},
			},
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.event.HasPermissionDenials()
			if result != tt.wantResult {
				t.Errorf("HasPermissionDenials() = %v, want %v", result, tt.wantResult)
			}
		})
	}
}

// TestStreamEventGetText tests the GetText method
func TestStreamEventGetText(t *testing.T) {
	tests := []struct {
		name     string
		event    *StreamEvent
		wantText string
	}{
		{
			name: "assistant text event",
			event: &StreamEvent{
				Type: "assistant",
				Message: Message{
					Content: []Content{{Type: "text", Text: "Hello, world!"}},
				},
			},
			wantText: "Hello, world!",
		},
		{
			name: "non-text event returns empty",
			event: &StreamEvent{
				Type:    "system",
				Subtype: "init",
			},
			wantText: "",
		},
		{
			name: "assistant tool_use event returns empty",
			event: &StreamEvent{
				Type: "assistant",
				Message: Message{
					Content: []Content{{Type: "tool_use", Name: "Bash"}},
				},
			},
			wantText: "",
		},
		{
			name: "assistant with empty content returns empty",
			event: &StreamEvent{
				Type:    "assistant",
				Message: Message{},
			},
			wantText: "",
		},
		{
			name: "assistant mixed content returns concatenated text blocks",
			event: &StreamEvent{
				Type: "assistant",
				Message: Message{
					Content: []Content{
						{Type: "tool_use", Name: "Read"},
						{Type: "text", Text: "Hello"},
						{Type: "text", Text: " world"},
					},
				},
			},
			wantText: "Hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := tt.event.GetText()
			if text != tt.wantText {
				t.Errorf("GetText() = %v, want %v", text, tt.wantText)
			}
		})
	}
}

// TestStreamEventGetToolInfo tests the GetToolInfo method
func TestStreamEventGetToolInfo(t *testing.T) {
	tests := []struct {
		name      string
		event     *StreamEvent
		wantName  string
		wantID    string
		wantInput map[string]interface{}
	}{
		{
			name: "assistant tool_use event",
			event: &StreamEvent{
				Type: "assistant",
				Message: Message{
					Content: []Content{{
						Type:  "tool_use",
						Name:  "Bash",
						ID:    "toolu_1",
						Input: map[string]interface{}{"command": "ls"},
					}},
				},
			},
			wantName:  "Bash",
			wantID:    "toolu_1",
			wantInput: map[string]interface{}{"command": "ls"},
		},
		{
			name: "non-tool_use event returns empty",
			event: &StreamEvent{
				Type:    "system",
				Subtype: "init",
			},
			wantName:  "",
			wantID:    "",
			wantInput: nil,
		},
		{
			name: "assistant text event returns empty",
			event: &StreamEvent{
				Type: "assistant",
				Message: Message{
					Content: []Content{{Type: "text", Text: "Hello"}},
				},
			},
			wantName:  "",
			wantID:    "",
			wantInput: nil,
		},
		{
			name: "assistant with empty content returns empty",
			event: &StreamEvent{
				Type:    "assistant",
				Message: Message{},
			},
			wantName:  "",
			wantID:    "",
			wantInput: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, id, input := tt.event.GetToolInfo()
			if name != tt.wantName {
				t.Errorf("GetToolInfo() name = %v, want %v", name, tt.wantName)
			}
			if id != tt.wantID {
				t.Errorf("GetToolInfo() id = %v, want %v", id, tt.wantID)
			}
			if input != nil && tt.wantInput != nil {
				if input["command"] != tt.wantInput["command"] {
					t.Errorf("GetToolInfo() input = %v, want %v", input, tt.wantInput)
				}
			} else if (input == nil) != (tt.wantInput == nil) {
				t.Errorf("GetToolInfo() input = %v, want %v", input, tt.wantInput)
			}
		})
	}
}

// TestStreamEventTypeChecks tests all the Is* methods
func TestStreamEventTypeChecks(t *testing.T) {
	// Test IsSystemInit
	t.Run("IsSystemInit", func(t *testing.T) {
		tests := []struct {
			event *StreamEvent
			want  bool
		}{
			{&StreamEvent{Type: "system", Subtype: "init"}, true},
			{&StreamEvent{Type: "system", Subtype: "other"}, false},
			{&StreamEvent{Type: "assistant", Subtype: "init"}, false},
		}
		for i, tt := range tests {
			if got := tt.event.IsSystemInit(); got != tt.want {
				t.Errorf("test %d: IsSystemInit() = %v, want %v", i, got, tt.want)
			}
		}
	})

	// Test IsResultSuccess
	t.Run("IsResultSuccess", func(t *testing.T) {
		tests := []struct {
			event *StreamEvent
			want  bool
		}{
			{&StreamEvent{Type: "result", Subtype: "success"}, true},
			{&StreamEvent{Type: "result", Subtype: "error"}, false},
			{&StreamEvent{Type: "assistant", Subtype: "success"}, false},
		}
		for i, tt := range tests {
			if got := tt.event.IsResultSuccess(); got != tt.want {
				t.Errorf("test %d: IsResultSuccess() = %v, want %v", i, got, tt.want)
			}
		}
	})

	// Test IsResultError
	t.Run("IsResultError", func(t *testing.T) {
		tests := []struct {
			event *StreamEvent
			want  bool
		}{
			{&StreamEvent{Type: "result", Subtype: "error"}, true},
			{&StreamEvent{Type: "result", Subtype: "success"}, false},
			{&StreamEvent{Type: "assistant", Subtype: "error"}, false},
		}
		for i, tt := range tests {
			if got := tt.event.IsResultError(); got != tt.want {
				t.Errorf("test %d: IsResultError() = %v, want %v", i, got, tt.want)
			}
		}
	})
}
