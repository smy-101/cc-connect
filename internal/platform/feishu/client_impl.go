package feishu

import (
	"context"
	"errors"
	"sync"
)

// SDKClient wraps the Feishu SDK to implement FeishuClient interface.
type SDKClient struct {
	mu sync.RWMutex

	appID     string
	appSecret string

	connected    bool
	eventHandler EventHandler
}

// NewSDKClient creates a new SDK-based Feishu client.
func NewSDKClient(appID, appSecret string) *SDKClient {
	return &SDKClient{
		appID:     appID,
		appSecret: appSecret,
	}
}

// Connect establishes a WebSocket long connection with Feishu platform.
func (c *SDKClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	// TODO: Integrate with actual Feishu SDK WebSocket client
	// This is a placeholder implementation for MVP
	// The actual implementation would use:
	// - larkws.NewClient() from github.com/larksuite/oapi-sdk-go/v3/ws
	// - Register event handlers
	// - Start the connection

	// For now, simulate connection (will be replaced with actual SDK integration)
	if c.appID == "" || c.appSecret == "" {
		return errors.New("invalid app_id or app_secret")
	}

	c.connected = true
	return nil
}

// Disconnect closes the WebSocket connection.
func (c *SDKClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	// TODO: Close actual SDK connection
	c.connected = false
	return nil
}

// IsConnected returns true if the WebSocket connection is active.
func (c *SDKClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.connected
}

// SendText sends a text message to the specified chat.
func (c *SDKClient) SendText(ctx context.Context, chatID, content string) error {
	c.mu.RLock()
	connected := c.connected
	c.mu.RUnlock()

	if !connected {
		return errors.New("client is not connected")
	}

	// TODO: Implement actual message sending using Feishu SDK
	// This would use:
	// - larkim.NewSendService() to send messages
	// - larkim.NewSendTextReqBuilder() to build the request

	return nil
}

// OnEvent registers an event handler for message events.
func (c *SDKClient) OnEvent(handler EventHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.eventHandler = handler
}

// handleMessageEvent processes an incoming message event.
// This is called by the SDK when a message is received.
func (c *SDKClient) handleMessageEvent(ctx context.Context, event *MessageReceiveEvent) error {
	c.mu.RLock()
	handler := c.eventHandler
	c.mu.RUnlock()

	if handler == nil {
		return nil
	}

	return handler(ctx, event)
}
