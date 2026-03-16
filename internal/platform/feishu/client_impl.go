package feishu

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

var (
	// ErrClientNotReady is returned when an operation requires an active Feishu connection.
	ErrClientNotReady            = errors.New("client is not ready")
	errInvalidCredentials        = errors.New("invalid app_id or app_secret")
	errConnectStoppedBeforeReady = errors.New("feishu websocket stopped before ready")
)

type clientState string

const (
	stateDisconnected clientState = "disconnected"
	stateConnecting   clientState = "connecting"
	stateReady        clientState = "ready"
	stateStopping     clientState = "stopping"
)

type sdkFacadeCallbacks struct {
	OnReady        func()
	OnDisconnected func(error)
	OnMessage      func(context.Context, *larkim.P2MessageReceiveV1) error
}

type sdkFacade interface {
	Start(ctx context.Context, callbacks sdkFacadeCallbacks) error
	Stop(ctx context.Context) error
	SendText(ctx context.Context, chatID, content string) error
}

// SDKClient wraps the Feishu SDK facade to implement FeishuClient interface.
type SDKClient struct {
	mu sync.RWMutex

	appID     string
	appSecret string

	facade        sdkFacade
	state         clientState
	connectCancel context.CancelFunc
	eventSeq      atomic.Int64
	eventHandler  EventHandler
}

// NewSDKClient creates a new SDK-based Feishu client.
func NewSDKClient(appID, appSecret string) *SDKClient {
	return newSDKClientWithFacade(appID, appSecret, newRealSDKFacade(appID, appSecret))
}

func newSDKClientWithFacade(appID, appSecret string, facade sdkFacade) *SDKClient {
	return &SDKClient{
		appID:     appID,
		appSecret: appSecret,
		facade:    facade,
		state:     stateDisconnected,
	}
}

// Connect establishes a WebSocket long connection with Feishu platform.
func (c *SDKClient) Connect(ctx context.Context) error {
	c.mu.Lock()

	if c.state == stateReady {
		c.mu.Unlock()
		return nil
	}
	if c.state == stateConnecting {
		c.mu.Unlock()
		return errors.New("client is already connecting")
	}
	if c.appID == "" || c.appSecret == "" {
		c.mu.Unlock()
		return errInvalidCredentials
	}

	connectCtx, cancel := context.WithCancel(ctx)
	c.connectCancel = cancel
	c.state = stateConnecting
	facade := c.facade
	c.mu.Unlock()

	readyCh := make(chan struct{})
	readyOnce := sync.Once{}
	errCh := make(chan error, 1)

	go func() {
		err := facade.Start(connectCtx, sdkFacadeCallbacks{
			OnReady: func() {
				c.mu.Lock()
				if c.state != stateStopping {
					c.state = stateReady
				}
				c.mu.Unlock()
				readyOnce.Do(func() { close(readyCh) })
			},
			OnDisconnected: func(err error) {
				c.mu.Lock()
				if c.state != stateStopping {
					c.state = stateDisconnected
				}
				c.mu.Unlock()
			},
			OnMessage: c.handleSDKEvent,
		})

		c.mu.Lock()
		if c.state != stateStopping {
			c.state = stateDisconnected
		}
		c.connectCancel = nil
		c.mu.Unlock()

		errCh <- err
	}()

	select {
	case <-readyCh:
		return nil
	case err := <-errCh:
		if err == nil {
			return errConnectStoppedBeforeReady
		}
		return err
	case <-ctx.Done():
		cancel()
		return ctx.Err()
	}
}

// Disconnect closes the WebSocket connection.
func (c *SDKClient) Disconnect() error {
	c.mu.Lock()
	if c.state == stateDisconnected {
		c.mu.Unlock()
		return nil
	}
	c.state = stateStopping
	cancel := c.connectCancel
	facade := c.facade
	c.connectCancel = nil
	c.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if facade != nil {
		if err := facade.Stop(context.Background()); err != nil {
			return err
		}
	}

	c.mu.Lock()
	c.state = stateDisconnected
	c.mu.Unlock()
	return nil
}

// IsConnected returns true if the WebSocket connection is active.
func (c *SDKClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.state == stateReady
}

// SendText sends a text message to the specified chat.
func (c *SDKClient) SendText(ctx context.Context, chatID, content string) error {
	c.mu.RLock()
	ready := c.state == stateReady
	facade := c.facade
	c.mu.RUnlock()

	if !ready {
		return ErrClientNotReady
	}
	return facade.SendText(ctx, chatID, content)
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

func (c *SDKClient) handleSDKEvent(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	converted, err := convertSDKMessageEvent(event, c.eventSeq.Add(1))
	if err != nil {
		return err
	}

	if err := c.handleMessageEvent(ctx, converted); err != nil {
		return nil
	}

	return nil
}
