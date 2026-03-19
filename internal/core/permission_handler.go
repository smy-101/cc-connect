package core

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// PermissionRequestHandler handles permission request events from agents.
// It sends interactive cards to the platform or falls back to text messages.
type PermissionRequestHandler struct {
	cardSender CardSender
	fallback   func(ctx context.Context, chatID, content string) error
}

// NewPermissionRequestHandler creates a new PermissionRequestHandler.
func NewPermissionRequestHandler(cardSender CardSender, fallback func(ctx context.Context, chatID, content string) error) *PermissionRequestHandler {
	return &PermissionRequestHandler{
		cardSender: cardSender,
		fallback:   fallback,
	}
}

// PermissionRequest contains the details of a permission request.
type PermissionRequest struct {
	RequestID string
	ToolName  string
	ToolInput string
	Questions []UserQuestion
}

// UserQuestion represents a question from the agent to the user.
type UserQuestion struct {
	ID          string
	Text        string
	Options     []UserQuestionOption
	MultiSelect bool
}

// UserQuestionOption represents an option for a question.
type UserQuestionOption struct {
	ID   string
	Text string
}

// HandlePermissionRequest handles a permission request event.
// It sends an interactive card if the platform supports it, otherwise falls back to text.
func (h *PermissionRequestHandler) HandlePermissionRequest(ctx context.Context, chatID string, req *PermissionRequest) error {
	if req == nil {
		return fmt.Errorf("permission request is nil")
	}
	if chatID == "" {
		return fmt.Errorf("chatID is empty")
	}

	// Build the permission request card
	card := h.buildPermissionCard(req)

	// Try to send card if platform supports it
	if h.cardSender != nil {
		if err := h.cardSender.SendCard(ctx, chatID, card); err != nil {
			slog.Warn("Failed to send permission card, falling back to text", "error", err, "chat_id", chatID)
			// Fall back to text
			return h.sendFallback(ctx, chatID, req)
		}
		return nil
	}

	// No card sender, use fallback
	return h.sendFallback(ctx, chatID, req)
}

// buildPermissionCard builds an interactive card for the permission request.
func (h *PermissionRequestHandler) buildPermissionCard(req *PermissionRequest) *Card {
	if len(req.Questions) > 0 {
		// AskUserQuestion
		return h.buildQuestionCard(req)
	}
	// Tool permission request
	return h.buildToolPermissionCard(req)
}

// buildToolPermissionCard builds a card for tool permission requests.
func (h *PermissionRequestHandler) buildToolPermissionCard(req *PermissionRequest) *Card {
	return NewCard().
		Title("🤖 Claude 需要您的确认", "blue").
		Markdown(fmt.Sprintf("**工具**: %s\n**输入**: `%s`", req.ToolName, truncate(req.ToolInput, 100))).
		ButtonsEqual(
			PrimaryBtn("✅ 允许", fmt.Sprintf("perm:allow:%s", req.RequestID)),
			DangerBtn("❌ 拒绝", fmt.Sprintf("perm:deny:%s", req.RequestID)),
		).
		Note("回复 A 允许，D 拒绝").
		Build()
}

// buildQuestionCard builds a card for AskUserQuestion requests.
func (h *PermissionRequestHandler) buildQuestionCard(req *PermissionRequest) *Card {
	builder := NewCard().Title("🤖 Claude 问您", "blue")

	// Add each question
	for _, q := range req.Questions {
		builder.Markdown(fmt.Sprintf("**%s**", q.Text))

		// Add options as buttons
		if len(q.Options) > 0 {
			var buttons []CardButton
			for _, opt := range q.Options {
				buttons = append(buttons, DefaultBtn(opt.Text, fmt.Sprintf("ans:%s:%s", req.RequestID, opt.Text)))
			}
			builder.Buttons(buttons...)
		}
	}

	return builder.Build()
}

// sendFallback sends a text message as fallback.
func (h *PermissionRequestHandler) sendFallback(ctx context.Context, chatID string, req *PermissionRequest) error {
	if h.fallback == nil {
		return fmt.Errorf("no fallback handler available")
	}

	content := buildFallbackText(req)
	return h.fallback(ctx, chatID, content)
}

// buildFallbackText builds a text representation of the permission request.
func buildFallbackText(req *PermissionRequest) string {
	var sb strings.Builder

	if len(req.Questions) > 0 {
		sb.WriteString("🤖 Claude 问您：\n\n")
		for _, q := range req.Questions {
			sb.WriteString(q.Text)
			sb.WriteString("\n")
			for _, opt := range q.Options {
				sb.WriteString("  - ")
				sb.WriteString(opt.Text)
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n回复 /answer ")
		sb.WriteString(req.RequestID)
		sb.WriteString(" <您的答案>")
		return sb.String()
	}

	sb.WriteString("🤖 Claude 需要您的确认\n\n")
	sb.WriteString("工具: ")
	sb.WriteString(req.ToolName)
	sb.WriteString("\n输入: ")
	sb.WriteString(truncate(req.ToolInput, 100))
	sb.WriteString("\n\n回复 /allow ")
	sb.WriteString(req.RequestID)
	sb.WriteString(" 允许, /deny ")
	sb.WriteString(req.RequestID)
	sb.WriteString(" 拒绝\n请求ID: ")
	sb.WriteString(req.RequestID)

	return sb.String()
}

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
