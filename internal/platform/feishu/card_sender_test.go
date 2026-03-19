package feishu

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/smy-101/cc-connect/internal/core"
)

func TestAdapterSendCard(t *testing.T) {
	tests := []struct {
		name        string
		chatID      string
		card        *core.Card
		expectError bool
	}{
		{
			name:   "send permission request card",
			chatID: "oc_xxx",
			card: core.NewCard().
				Title("🤖 Claude 需要您的确认", "blue").
				Markdown("**工具**: Bash\n**命令**: `npm install`").
				ButtonsEqual(
					core.PrimaryBtn("✅ 允许", "perm:allow:req123"),
					core.DangerBtn("❌ 拒绝", "perm:deny:req123"),
				).
				Note("回复 A 允许，D 拒绝").
				Build(),
			expectError: false,
		},
		{
			name:   "send ask user question card",
			chatID: "oc_xxx",
			card: core.NewCard().
				Title("🤖 Claude 问您", "blue").
				Markdown("您希望使用哪种数据库？").
				ButtonsEqual(
					core.DefaultBtn("PostgreSQL", "ans:req123:PostgreSQL"),
					core.DefaultBtn("MySQL", "ans:req123:MySQL"),
				).
				Build(),
			expectError: false,
		},
		{
			name:        "send with empty chatID fails",
			chatID:      "",
			card:        core.NewCard().Title("Test", "blue").Build(),
			expectError: true,
		},
		{
			name:        "send with nil card fails",
			chatID:      "oc_xxx",
			card:        nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockClient()
			adapter := NewAdapter(mockClient, core.NewRouter())

			err := adapter.SendCard(context.Background(), tt.chatID, tt.card)

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

			// Verify SendCardCalled was incremented
			if mockClient.SendCardCalled == 0 {
				t.Error("expected SendCard to be called on client")
			}

			// Verify the card content was sent correctly
			if mockClient.LastSendCardContent != nil {
				// Should be able to unmarshal as Feishu card JSON
				var cardMap map[string]any
				if err := json.Unmarshal(mockClient.LastSendCardContent, &cardMap); err != nil {
					t.Errorf("card content should be valid JSON: %v", err)
				}

				// Verify config exists
				config, ok := cardMap["config"].(map[string]any)
				if !ok {
					t.Error("card should have config")
				}
				if config["wide_screen_mode"] != true {
					t.Error("wide_screen_mode should be true")
				}
			}
		})
	}
}

func TestAdapterReplyCard(t *testing.T) {
	tests := []struct {
		name        string
		replyCtx    *ReplyContext
		card        *core.Card
		expectError bool
	}{
		{
			name: "reply with card",
			replyCtx: &ReplyContext{
				ChatID:    "oc_xxx",
				MessageID: "om_xxx",
			},
			card: core.NewCard().
				Title("🤖 Claude 需要您的确认", "blue").
				Markdown("**工具**: Bash").
				ButtonsEqual(
					core.PrimaryBtn("✅ 允许", "perm:allow:req123"),
					core.DangerBtn("❌ 拒绝", "perm:deny:req123"),
				).
				Build(),
			expectError: false,
		},
		{
			name:        "reply with nil context fails",
			replyCtx:    nil,
			card:        core.NewCard().Title("Test", "blue").Build(),
			expectError: true,
		},
		{
			name: "reply with nil card fails",
			replyCtx: &ReplyContext{
				ChatID:    "oc_xxx",
				MessageID: "om_xxx",
			},
			card:        nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockClient()
			adapter := NewAdapter(mockClient, core.NewRouter())

			err := adapter.ReplyCard(context.Background(), tt.replyCtx, tt.card)

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

			// Verify ReplyCardCalled was incremented
			if mockClient.ReplyCardCalled == 0 {
				t.Error("expected ReplyCard to be called on client")
			}
		})
	}
}

func TestAdapterSendCardFallback(t *testing.T) {
	// Test that when card sending fails, we can fallback to text
	mockClient := NewMockClient()
	mockClient.SendCardError = ErrClientNotReady
	adapter := NewAdapter(mockClient, core.NewRouter())

	card := core.NewCard().
		Title("🤖 Claude 需要您的确认", "blue").
		Markdown("**工具**: Bash").
		Build()

	// SendCard should fail
	err := adapter.SendCard(context.Background(), "oc_xxx", card)
	if err == nil {
		t.Error("expected error when SendCard fails")
	}
}
