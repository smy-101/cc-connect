package feishu

import (
	"testing"

	"github.com/smy-101/cc-connect/internal/core"
)

func TestParseCardCallback(t *testing.T) {
	tests := []struct {
		name           string
		callbackData   map[string]any
		expectedMsg    *core.Message
		expectError    bool
	}{
		{
			name: "parse allow button callback",
			callbackData: map[string]any{
				"action": map[string]any{
					"value": map[string]any{
						"action": "perm:allow:req123",
					},
				},
				"open_id":      "ou_user123",
				"open_message_id": "om_msg123",
				"token":        "valid_token",
			},
			expectedMsg: &core.Message{
				Type:      core.MessageTypeCommand,
				Content:   "/allow req123",
				UserID:    "ou_user123",
				ChannelID: "om_msg123",
			},
			expectError: false,
		},
		{
			name: "parse deny button callback",
			callbackData: map[string]any{
				"action": map[string]any{
					"value": map[string]any{
						"action": "perm:deny:req456",
					},
				},
				"open_id":      "ou_user456",
				"open_message_id": "om_msg456",
				"token":        "valid_token",
			},
			expectedMsg: &core.Message{
				Type:      core.MessageTypeCommand,
				Content:   "/deny req456",
				UserID:    "ou_user456",
				ChannelID: "om_msg456",
			},
			expectError: false,
		},
		{
			name: "parse answer button callback",
			callbackData: map[string]any{
				"action": map[string]any{
					"value": map[string]any{
						"action": "ans:req789:PostgreSQL",
					},
				},
				"open_id":      "ou_user789",
				"open_message_id": "om_msg789",
				"token":        "valid_token",
			},
			expectedMsg: &core.Message{
				Type:      core.MessageTypeCommand,
				Content:   "/answer req789 PostgreSQL",
				UserID:    "ou_user789",
				ChannelID: "om_msg789",
			},
			expectError: false,
		},
		{
			name: "parse answer with colon in value",
			callbackData: map[string]any{
				"action": map[string]any{
					"value": map[string]any{
						"action": "ans:req123:Use Docker for development",
					},
				},
				"open_id":      "ou_user123",
				"open_message_id": "om_msg123",
				"token":        "valid_token",
			},
			expectedMsg: &core.Message{
				Type:      core.MessageTypeCommand,
				Content:   "/answer req123 Use Docker for development",
				UserID:    "ou_user123",
				ChannelID: "om_msg123",
			},
			expectError: false,
		},
		{
			name: "missing action value returns error",
			callbackData: map[string]any{
				"action": map[string]any{
					"value": map[string]any{},
				},
				"open_id":      "ou_user123",
				"open_message_id": "om_msg123",
				"token":        "valid_token",
			},
			expectError: true,
		},
		{
			name: "missing action returns error",
			callbackData: map[string]any{
				"open_id":      "ou_user123",
				"open_message_id": "om_msg123",
				"token":        "valid_token",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := ParseCardCallback(tt.callbackData)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if msg.Type != tt.expectedMsg.Type {
				t.Errorf("expected type %v, got %v", tt.expectedMsg.Type, msg.Type)
			}
			if msg.Content != tt.expectedMsg.Content {
				t.Errorf("expected content %q, got %q", tt.expectedMsg.Content, msg.Content)
			}
			if msg.UserID != tt.expectedMsg.UserID {
				t.Errorf("expected UserID %q, got %q", tt.expectedMsg.UserID, msg.UserID)
			}
			if msg.ChannelID != tt.expectedMsg.ChannelID {
				t.Errorf("expected ChannelID %q, got %q", tt.expectedMsg.ChannelID, msg.ChannelID)
			}
		})
	}
}

func TestParseActionToCommand(t *testing.T) {
	tests := []struct {
		name          string
		action        string
		expectedCmd   string
		expectedError bool
	}{
		{
			name:        "perm:allow action",
			action:      "perm:allow:req123",
			expectedCmd: "/allow req123",
		},
		{
			name:        "perm:deny action",
			action:      "perm:deny:req456",
			expectedCmd: "/deny req456",
		},
		{
			name:        "ans action with simple value",
			action:      "ans:req789:PostgreSQL",
			expectedCmd: "/answer req789 PostgreSQL",
		},
		{
			name:        "ans action with complex value",
			action:      "ans:req123:Use Docker:latest",
			expectedCmd: "/answer req123 Use Docker:latest",
		},
		{
			name:        "unknown prefix passes through",
			action:      "custom:action:value",
			expectedCmd: "/custom action value",
		},
		{
			name:          "empty action returns error",
			action:        "",
			expectedError: true,
		},
		{
			name:          "malformed action returns error",
			action:        "invalid",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := parseActionToCommand(tt.action)

			if tt.expectedError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if cmd != tt.expectedCmd {
				t.Errorf("expected command %q, got %q", tt.expectedCmd, cmd)
			}
		})
	}
}
