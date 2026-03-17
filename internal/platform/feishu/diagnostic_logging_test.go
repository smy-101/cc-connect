package feishu

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	"github.com/smy-101/cc-connect/internal/core"
)

func TestSDKClientHandleSDKEventLogsEventArrival(t *testing.T) {
	buffer, restore := captureTestLogs(t, slog.LevelDebug)
	defer restore()

	client := newSDKClientWithFacade("app-id", "app-secret", &fakeSDKFacade{})
	client.OnEvent(func(ctx context.Context, event *MessageReceiveEvent) error {
		return nil
	})

	err := client.handleSDKEvent(context.Background(), newSDKTextEvent("evt-log", "om-log", "oc-log", "top secret"))
	if err != nil {
		t.Fatalf("handleSDKEvent() error = %v", err)
	}

	assertLogContainsAll(t, buffer.String(), []string{
		"Feishu SDK event received",
		"event_id=evt-log",
		"message_id=om-log",
		"chat_type=p2p",
		"message_type=text",
	})
	assertLogExcludesAll(t, buffer.String(), []string{"top secret", `{"text":"top secret"}`})
}

func TestAdapterHandleEventLogsFailures(t *testing.T) {
	t.Run("conversion failure", func(t *testing.T) {
		buffer, restore := captureTestLogs(t, slog.LevelDebug)
		defer restore()

		adapter := NewAdapter(NewMockClient(), coreRouter())
		event := &MessageReceiveEvent{
			EventID:     "evt-convert",
			MessageID:   "msg-convert",
			MessageType: "text",
			Content:     "invalid json",
			ChatID:      "oc-chat",
			ChatType:    "p2p",
			Sender:      SenderInfo{SenderType: "user"},
			CreateTime:  time.Now(),
		}

		err := adapter.HandleEvent(context.Background(), event)
		if err == nil {
			t.Fatal("HandleEvent() should return conversion error")
		}

		assertLogContainsAll(t, buffer.String(), []string{
			"Feishu event conversion failed",
			"event_id=evt-convert",
			"message_id=msg-convert",
		})
		assertLogExcludesAll(t, buffer.String(), []string{"invalid json"})
	})

	t.Run("route failure", func(t *testing.T) {
		buffer, restore := captureTestLogs(t, slog.LevelDebug)
		defer restore()

		router := coreRouter()
		router.Register(core.MessageTypeText, func(ctx context.Context, msg *core.Message) error {
			return errors.New("route failed")
		})
		adapter := NewAdapter(NewMockClient(), router)
		event := &MessageReceiveEvent{
			EventID:     "evt-route",
			MessageID:   "msg-route",
			MessageType: "text",
			Content:     `{"text":"secret route body"}`,
			ChatID:      "oc-chat",
			ChatType:    "p2p",
			Sender:      SenderInfo{OpenID: "ou-user", SenderType: "user"},
			CreateTime:  time.Now(),
		}

		err := adapter.HandleEvent(context.Background(), event)
		if err == nil {
			t.Fatal("HandleEvent() should return route error")
		}

		assertLogContainsAll(t, buffer.String(), []string{
			"Feishu message routing failed",
			"event_id=evt-route",
			"message_id=msg-route",
			"channel_id=oc-chat",
		})
		assertLogExcludesAll(t, buffer.String(), []string{"secret route body"})
	})
}

func TestRealSDKFacadeSendTextLogsFailureWithoutSensitiveData(t *testing.T) {
	buffer, restore := captureTestLogs(t, slog.LevelDebug)
	defer restore()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open-apis/auth/v3/tenant_access_token/internal":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"code":0,"msg":"ok","tenant_access_token":"token","expire":7200}`)
		case "/open-apis/im/v1/messages":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"code":999,"msg":"permission denied"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	facade := &realSDKFacade{
		appID:             "app-log-id",
		appSecret:         "app-log-secret",
		baseURL:           server.URL,
		apiClient:         lark.NewClient("app-log-id", "app-log-secret", lark.WithOpenBaseUrl(server.URL)),
		reconnectNonce:    0,
		reconnectInterval: time.Second,
		pingInterval:      time.Second,
		fragments:         make(map[string][][]byte),
	}

	err := facade.SendText(context.Background(), "oc-chat", `{"text":"super secret body"}`)
	if err == nil {
		t.Fatal("SendText() should return API failure")
	}

	assertLogContainsAll(t, buffer.String(), []string{
		"Feishu API send failed",
		"chat_id=oc-chat",
	})
	assertLogExcludesAll(t, buffer.String(), []string{"app-secret", "super secret body"})
}

func captureTestLogs(t *testing.T, level slog.Level) (*bytes.Buffer, func()) {
	t.Helper()

	var buffer bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buffer, &slog.HandlerOptions{Level: level})))

	return &buffer, func() {
		slog.SetDefault(previous)
	}
}

func assertLogContainsAll(t *testing.T, output string, fragments []string) {
	t.Helper()

	for _, fragment := range fragments {
		if !strings.Contains(output, fragment) {
			t.Fatalf("expected %q in log output %q", fragment, output)
		}
	}
}

func assertLogExcludesAll(t *testing.T, output string, fragments []string) {
	t.Helper()

	for _, fragment := range fragments {
		if strings.Contains(output, fragment) {
			t.Fatalf("did not expect %q in log output %q", fragment, output)
		}
	}
}

func coreRouter() *core.Router {
	return core.NewRouter()
}
