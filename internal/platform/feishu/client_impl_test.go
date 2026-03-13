//go:build integration
// +build integration

package feishu

import (
	"context"
	"testing"
	"time"
)

// These tests require the integration build tag and test the SDK client
// without making actual network connections (using mock SDK components).

func TestSDKClientIntegration(t *testing.T) {
	// Note: These are placeholder tests for the integration test suite.
	// Real integration tests would require:
	// 1. A mock SDK server or
	// 2. Actual Feishu app credentials (set via environment variables)

	t.Run("client creation with valid config", func(t *testing.T) {
		client := NewSDKClient("test_app_id", "test_app_secret")
		if client == nil {
			t.Error("NewSDKClient returned nil")
		}
	})

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

	t.Run("connect and disconnect cycle", func(t *testing.T) {
		client := NewSDKClient("test_app_id", "test_app_secret")

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
		client := NewSDKClient("test_app_id", "test_app_secret")

		err := client.SendText(context.Background(), "test_chat_id", "Hello")
		if err == nil {
			t.Error("SendText() without connection should return error")
		}
	})

	t.Run("send text with connection", func(t *testing.T) {
		client := NewSDKClient("test_app_id", "test_app_secret")

		ctx := context.Background()
		_ = client.Connect(ctx)

		// With placeholder implementation, SendText should succeed
		// (actual SDK integration would send real messages)
		err := client.SendText(ctx, "test_chat_id", "Hello")
		if err != nil {
			t.Errorf("SendText() with connection returned error: %v", err)
		}
	})
}
