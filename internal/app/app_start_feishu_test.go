package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/smy-101/cc-connect/internal/platform/feishu"
)

func TestAppStartPropagatesFeishuConnectFailure(t *testing.T) {
	application, err := createTestApp()
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	mockClient := feishu.NewMockClient()
	mockClient.ConnectError = errors.New("auth failed")
	application.feishuClientFactory = func(appID, appSecret string) feishu.FeishuClient {
		return mockClient
	}

	err = application.Start(context.Background())
	if err == nil {
		t.Fatal("Start() should return error when Feishu connect fails")
	}
	if !strings.Contains(err.Error(), "auth failed") {
		t.Fatalf("Start() error = %v, want auth failure in message", err)
	}
	if application.Status() == AppStatusRunning {
		t.Fatal("app should not report running after Feishu connect failure")
	}
}
