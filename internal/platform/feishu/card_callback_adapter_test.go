package feishu

import (
	"context"
	"testing"

	"github.com/smy-101/cc-connect/internal/core"
)

func TestAdapterHandleCardCallback(t *testing.T) {
	tests := []struct {
		name         string
		callbackData map[string]any
		expectError  bool
	}{
		{
			name: "handle allow callback",
			callbackData: map[string]any{
				"action": map[string]any{
					"value": map[string]any{
						"action": "perm:allow:req123",
					},
				},
				"open_id":        "ou_user123",
				"open_message_id": "om_msg123",
				"token":          "valid_token",
			},
			expectError: false,
		},
		{
			name: "handle answer callback",
			callbackData: map[string]any{
				"action": map[string]any{
					"value": map[string]any{
						"action": "ans:req789:PostgreSQL",
					},
				},
				"open_id":        "ou_user789",
				"open_message_id": "om_msg789",
				"token":          "valid_token",
			},
			expectError: false,
		},
		{
			name: "missing action returns error",
			callbackData: map[string]any{
				"open_id":        "ou_user123",
				"open_message_id": "om_msg123",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockClient()
			router := core.NewRouter()

			// Register a handler to capture the routed message
			var routedMsg *core.Message
			router.Register(core.MessageTypeCommand, func(ctx context.Context, msg *core.Message) error {
				routedMsg = msg
				return nil
			})

			adapter := NewAdapter(mockClient, router)

			err := adapter.HandleCardCallback(context.Background(), tt.callbackData)

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

			// Verify the message was routed
			if routedMsg == nil {
				t.Error("expected message to be routed")
				return
			}

			if routedMsg.Type != core.MessageTypeCommand {
				t.Errorf("expected command type, got %v", routedMsg.Type)
			}
		})
	}
}

func TestAdapterHandleCardCallbackNoHandler(t *testing.T) {
	mockClient := NewMockClient()
	router := core.NewRouter() // No handlers registered

	adapter := NewAdapter(mockClient, router)

	callbackData := map[string]any{
		"action": map[string]any{
			"value": map[string]any{
				"action": "perm:allow:req123",
			},
		},
		"open_id":        "ou_user123",
		"open_message_id": "om_msg123",
	}

	// Should not error even without handler (just logs and returns nil)
	err := adapter.HandleCardCallback(context.Background(), callbackData)
	if err != nil {
		t.Errorf("expected no error when no handler, got: %v", err)
	}
}
