package app

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/smy-101/cc-connect/internal/agent"
	"github.com/smy-101/cc-connect/internal/agent/claudecode"
	"github.com/smy-101/cc-connect/internal/core"
	"github.com/smy-101/cc-connect/internal/platform/feishu"
)

func TestHandleTextLogsStatusReplyFailure(t *testing.T) {
	buffer, restore := captureAppTestLogs(t, slog.LevelDebug)
	defer restore()

	application, mockClient, mockAgent := newLoggingTestApp(t)
	mockClient.SendTextError = errors.New("status send failed")
	mockAgent.SetResponse(&agent.Response{Content: "unused"})

	err := application.handleText(newLoggingHandlerContext(application, "msg-status", "secret status question"))
	if err == nil || !strings.Contains(err.Error(), "failed to send status") {
		t.Fatalf("handleText() error = %v, want status failure", err)
	}

	assertAppLogContainsAll(t, buffer.String(), []string{
		"Sending thinking reply",
		"Thinking reply failed",
		"Feishu reply send failed",
		"channel_id=oc-chat",
		"message_id=msg-status",
	})
	assertAppLogExcludesAll(t, buffer.String(), []string{"secret status question", "unused"})
}

func TestHandleTextLogsAgentInvocationFailure(t *testing.T) {
	buffer, restore := captureAppTestLogs(t, slog.LevelDebug)
	defer restore()

	application, mockClient, mockAgent := newLoggingTestApp(t)
	mockAgent.SetError(errors.New("agent failed"))

	err := application.handleText(newLoggingHandlerContext(application, "msg-agent", "super secret prompt"))
	if err != nil {
		t.Fatalf("handleText() error = %v, want nil after failure reply", err)
	}
	if mockClient.SendTextCalled != 2 {
		t.Fatalf("SendTextCalled = %d, want 2", mockClient.SendTextCalled)
	}

	assertAppLogContainsAll(t, buffer.String(), []string{
		"Sending thinking reply",
		"Claude Code request started",
		"Claude Code invocation failed",
		"channel_id=oc-chat",
		"message_id=msg-agent",
	})
	assertAppLogExcludesAll(t, buffer.String(), []string{"super secret prompt"})
}

func TestHandleTextLogsFinalReplyFailure(t *testing.T) {
	buffer, restore := captureAppTestLogs(t, slog.LevelDebug)
	defer restore()

	application, _, mockAgent := newLoggingTestApp(t)
	mockAgent.SetResponse(&agent.Response{Content: "super secret answer"})

	sendCount := 0
	application.feishu = feishu.NewAdapter(&feishu.MockClient{SendTextHandler: func(chatID, content string) error {
		sendCount++
		if sendCount == 2 {
			return errors.New("final send failed")
		}
		return nil
	}}, core.NewRouter())

	err := application.handleText(newLoggingHandlerContext(application, "msg-final", "top secret prompt"))
	if err == nil || !strings.Contains(err.Error(), "final send failed") {
		t.Fatalf("handleText() error = %v, want final reply failure", err)
	}

	assertAppLogContainsAll(t, buffer.String(), []string{
		"Sending final reply",
		"Final reply send failed",
		"Feishu reply send failed",
		"channel_id=oc-chat",
		"message_id=msg-final",
	})
	assertAppLogExcludesAll(t, buffer.String(), []string{"top secret prompt", "super secret answer"})
}

func TestHandleTextEmptyAgentResponseUsesFallbackReply(t *testing.T) {
	buffer, restore := captureAppTestLogs(t, slog.LevelDebug)
	defer restore()

	application, mockClient, mockAgent := newLoggingTestApp(t)
	mockAgent.SetResponse(&agent.Response{Content: ""})

	err := application.handleText(newLoggingHandlerContext(application, "msg-empty", "secret empty prompt"))
	if err != nil {
		t.Fatalf("handleText() error = %v", err)
	}
	if mockClient.SendTextCalled != 2 {
		t.Fatalf("SendTextCalled = %d, want 2", mockClient.SendTextCalled)
	}

	expectedContent := `{"text":"⚠️ Claude Code 未返回内容，请稍后重试。"}`
	if mockClient.LastSendTextContent != expectedContent {
		t.Fatalf("LastSendTextContent = %q, want %q", mockClient.LastSendTextContent, expectedContent)
	}

	assertAppLogContainsAll(t, buffer.String(), []string{
		"Claude Code returned empty reply",
		"Sending final reply",
		"Final reply sent",
		"channel_id=oc-chat",
		"message_id=msg-empty",
	})
	assertAppLogExcludesAll(t, buffer.String(), []string{"secret empty prompt"})
}

func newLoggingTestApp(t *testing.T) (*App, *feishu.MockClient, *claudecode.MockAgent) {
	t.Helper()

	application, err := createTestApp()
	if err != nil {
		t.Fatalf("createTestApp() error = %v", err)
	}

	mockClient := feishu.NewMockClient()
	application.feishu = feishu.NewAdapter(mockClient, core.NewRouter())

	mockAgent := claudecode.NewMockAgent(&claudecode.Config{})
	application.agent = mockAgent

	return application, mockClient, mockAgent
}

func newLoggingHandlerContext(application *App, messageID, content string) *HandlerContext {
	msg := &core.Message{
		ID:        messageID,
		Platform:  "feishu",
		UserID:    "ou-user",
		ChannelID: "oc-chat",
		Content:   content,
		Type:      core.MessageTypeText,
	}

	return &HandlerContext{
		Ctx:   context.Background(),
		Msg:   msg,
		Reply: newReplySender(application.feishu, msg.ChannelID),
	}
}

func captureAppTestLogs(t *testing.T, level slog.Level) (*bytes.Buffer, func()) {
	t.Helper()

	var buffer bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buffer, &slog.HandlerOptions{Level: level})))

	return &buffer, func() {
		slog.SetDefault(previous)
	}
}

func assertAppLogContainsAll(t *testing.T, output string, fragments []string) {
	t.Helper()

	for _, fragment := range fragments {
		if !strings.Contains(output, fragment) {
			t.Fatalf("expected %q in log output %q", fragment, output)
		}
	}
}

func assertAppLogExcludesAll(t *testing.T, output string, fragments []string) {
	t.Helper()

	for _, fragment := range fragments {
		if strings.Contains(output, fragment) {
			t.Fatalf("did not expect %q in log output %q", fragment, output)
		}
	}
}
