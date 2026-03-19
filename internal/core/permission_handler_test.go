package core

import (
	"context"
	"strings"
	"testing"
)

func TestPermissionRequestHandler_HandlePermissionRequest(t *testing.T) {
	tests := []struct {
		name           string
		chatID         string
		req            *PermissionRequest
		expectError    bool
		expectCard     bool
		expectFallback bool
	}{
		{
			name:   "tool permission request",
			chatID: "oc_chat123",
			req: &PermissionRequest{
				RequestID: "req123",
				ToolName:  "Bash",
				ToolInput: `{"command": "npm install"}`,
			},
			expectCard: true,
		},
		{
			name:   "ask user question request",
			chatID: "oc_chat123",
			req: &PermissionRequest{
				RequestID: "req456",
				Questions: []UserQuestion{
					{
						Text: "Which database?",
						Options: []UserQuestionOption{
							{ID: "1", Text: "PostgreSQL"},
							{ID: "2", Text: "MySQL"},
						},
					},
				},
			},
			expectCard: true,
		},
		{
			name:        "nil request returns error",
			chatID:      "oc_chat123",
			req:         nil,
			expectError: true,
		},
		{
			name:        "empty chatID returns error",
			chatID:      "",
			req:         &PermissionRequest{RequestID: "req123"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cardSent bool
			var lastCard *Card
			var fallbackSent bool

			mockCardSender := &mockPermCardSender{
				sendCardFunc: func(ctx context.Context, chatID string, card *Card) error {
					cardSent = true
					lastCard = card
					return nil
				},
			}
			fallback := func(ctx context.Context, chatID, content string) error {
				fallbackSent = true
				return nil
			}

			handler := NewPermissionRequestHandler(mockCardSender, fallback)
			err := handler.HandlePermissionRequest(context.Background(), tt.chatID, tt.req)

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

			if tt.expectCard && !cardSent {
				t.Error("expected card to be sent")
			}

			if tt.expectFallback && !fallbackSent {
				t.Error("expected fallback to be sent")
			}

			// Verify tool permission card structure
			if tt.expectCard && lastCard != nil && len(tt.req.Questions) == 0 {
				if lastCard.Header == nil {
					t.Error("expected card header")
				}
				if lastCard.Header.Title != "🤖 Claude 需要您的确认" {
					t.Errorf("unexpected title: %s", lastCard.Header.Title)
				}
			}
		})
	}
}

func TestPermissionRequestHandler_Fallback(t *testing.T) {
	var lastFallbackContent string
	fallback := func(ctx context.Context, chatID, content string) error {
		lastFallbackContent = content
		return nil
	}

	handler := NewPermissionRequestHandler(nil, fallback) // No card sender

	req := &PermissionRequest{
		RequestID: "req123",
		ToolName:  "Bash",
		ToolInput: `{"command": "npm test"}`,
	}

	err := handler.HandlePermissionRequest(context.Background(), "oc_chat123", req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if lastFallbackContent == "" {
		t.Error("expected fallback content")
	}

	// Verify fallback contains key information
	if !strings.Contains(lastFallbackContent, "Bash") {
		t.Error("fallback should contain tool name")
	}
	if !strings.Contains(lastFallbackContent, "/allow") {
		t.Error("fallback should contain /allow command")
	}
	if !strings.Contains(lastFallbackContent, "/deny") {
		t.Error("fallback should contain /deny command")
	}
}

func TestPermissionRequestHandler_NoFallback(t *testing.T) {
	handler := NewPermissionRequestHandler(nil, nil) // No card sender, no fallback

	req := &PermissionRequest{
		RequestID: "req123",
		ToolName:  "Bash",
		ToolInput: `{"command": "npm test"}`,
	}

	err := handler.HandlePermissionRequest(context.Background(), "oc_chat123", req)
	if err == nil {
		t.Error("expected error when no card sender and no fallback")
	}
}

func TestBuildFallbackText(t *testing.T) {
	t.Run("tool permission fallback", func(t *testing.T) {
		req := &PermissionRequest{
			RequestID: "req123",
			ToolName:  "Bash",
			ToolInput: `{"command": "npm install"}`,
		}

		text := buildFallbackText(req)

		if !strings.Contains(text, "Bash") {
			t.Error("should contain tool name")
		}
		if !strings.Contains(text, "/allow req123") {
			t.Error("should contain allow command with request ID")
		}
		if !strings.Contains(text, "/deny req123") {
			t.Error("should contain deny command with request ID")
		}
	})

	t.Run("ask user question fallback", func(t *testing.T) {
		req := &PermissionRequest{
			RequestID: "req456",
			Questions: []UserQuestion{
				{
					Text: "Which database?",
					Options: []UserQuestionOption{
						{Text: "PostgreSQL"},
						{Text: "MySQL"},
					},
				},
			},
		}

		text := buildFallbackText(req)

		if !strings.Contains(text, "Claude 问您") {
			t.Error("should contain question header")
		}
		if !strings.Contains(text, "PostgreSQL") {
			t.Error("should contain option")
		}
		if !strings.Contains(text, "/answer") {
			t.Error("should contain answer command")
		}
	})
}

// mockPermCardSender implements a simple card sender for testing
type mockPermCardSender struct {
	sendCardFunc func(ctx context.Context, chatID string, card *Card) error
}

func (m *mockPermCardSender) SendCard(ctx interface{}, replyCtx interface{}, card *Card) error {
	if m.sendCardFunc != nil {
		return m.sendCardFunc(ctx.(context.Context), replyCtx.(string), card)
	}
	return nil
}

func (m *mockPermCardSender) ReplyCard(ctx interface{}, replyCtx interface{}, card *Card) error {
	return nil
}
