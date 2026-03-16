//go:build integration
// +build integration

package feishu

import (
	"context"
	"os"
	"testing"
	"time"
)

const (
	realIntegrationAppIDEnv     = "FEISHU_APP_ID"
	realIntegrationAppSecretEnv = "FEISHU_APP_SECRET"
	realIntegrationChatIDEnv    = "FEISHU_CHAT_ID"
)

func requireRealIntegrationEnv(t *testing.T) (string, string, string) {
	t.Helper()

	appID := os.Getenv(realIntegrationAppIDEnv)
	appSecret := os.Getenv(realIntegrationAppSecretEnv)
	chatID := os.Getenv(realIntegrationChatIDEnv)
	if appID == "" || appSecret == "" {
		t.Skipf("set %s and %s to run real Feishu integration tests", realIntegrationAppIDEnv, realIntegrationAppSecretEnv)
	}

	return appID, appSecret, chatID
}

func TestSDKClientIntegration(t *testing.T) {
	t.Run("initial connection state", func(t *testing.T) {
		client := NewSDKClient("test_app_id", "test_app_secret")
		if client.IsConnected() {
			t.Error("newly created client should not be connected")
		}
	})

	t.Run("disconnect without connect", func(t *testing.T) {
		client := NewSDKClient("test_app_id", "test_app_secret")
		err := client.Disconnect()
		if err != nil {
			t.Errorf("Disconnect() on unconnected client returned error: %v", err)
		}
	})

	t.Run("connect with invalid credentials", func(t *testing.T) {
		client := NewSDKClient("", "")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := client.Connect(ctx)
		if err == nil {
			t.Error("Connect() should fail with empty credentials")
		}
	})

	t.Run("connect to real feishu websocket", func(t *testing.T) {
		appID, appSecret, _ := requireRealIntegrationEnv(t)
		client := NewSDKClient(appID, appSecret)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := client.Connect(ctx); err != nil {
			t.Fatalf("Connect() failed: %v", err)
		}
		defer client.Disconnect()

		if !client.IsConnected() {
			t.Fatal("client should report connected after successful Connect()")
		}
	})

	t.Run("send text through real feishu api", func(t *testing.T) {
		appID, appSecret, chatID := requireRealIntegrationEnv(t)
		if chatID == "" {
			t.Skipf("set %s to verify real text sending", realIntegrationChatIDEnv)
		}

		client := NewSDKClient(appID, appSecret)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := client.Connect(ctx); err != nil {
			t.Fatalf("Connect() failed: %v", err)
		}
		defer client.Disconnect()

		if err := client.SendText(ctx, chatID, `{"text":"cc-connect integration ping"}`); err != nil {
			t.Fatalf("SendText() failed: %v", err)
		}
	})

	t.Run("connect and disconnect cycle", func(t *testing.T) {
		facade := &fakeSDKFacade{
			startFunc: func(ctx context.Context, callbacks sdkFacadeCallbacks) error {
				callbacks.OnReady()
				<-ctx.Done()
				return nil
			},
		}
		client := newSDKClientWithFacade("test_app_id", "test_app_secret", facade)

		ctx := context.Background()

		// Connect
		err := client.Connect(ctx)
		if err != nil {
			t.Errorf("Connect() failed: %v", err)
		}

		if !client.IsConnected() {
			t.Error("client should be connected after Connect()")
		}

		// Disconnect
		err = client.Disconnect()
		if err != nil {
			t.Errorf("Disconnect() failed: %v", err)
		}

		if client.IsConnected() {
			t.Error("client should not be connected after Disconnect()")
		}
	})
}

func TestSDKClientEventHandling(t *testing.T) {
	t.Run("register event handler", func(t *testing.T) {
		client := NewSDKClient("test_app_id", "test_app_secret")

		handlerCalled := false
		client.OnEvent(func(ctx context.Context, event *MessageReceiveEvent) error {
			handlerCalled = true
			return nil
		})

		// Handler is registered (no direct way to verify, but no error means success)
		if handlerCalled {
			t.Error("handler should not be called yet")
		}
	})
}

func TestSDKClientSendText(t *testing.T) {
	t.Run("send text without connection", func(t *testing.T) {
		client := newSDKClientWithFacade("test_app_id", "test_app_secret", &fakeSDKFacade{})

		err := client.SendText(context.Background(), "test_chat_id", "Hello")
		if err == nil {
			t.Error("SendText() without connection should return error")
		}
	})

	t.Run("send text with connection", func(t *testing.T) {
		facade := &fakeSDKFacade{
			startFunc: func(ctx context.Context, callbacks sdkFacadeCallbacks) error {
				callbacks.OnReady()
				<-ctx.Done()
				return nil
			},
		}
		client := newSDKClientWithFacade("test_app_id", "test_app_secret", facade)

		ctx := context.Background()
		_ = client.Connect(ctx)

		err := client.SendText(ctx, "test_chat_id", "Hello")
		if err != nil {
			t.Errorf("SendText() with connection returned error: %v", err)
		}
	})
}
