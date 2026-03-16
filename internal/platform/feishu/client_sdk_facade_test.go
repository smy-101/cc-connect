package feishu

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	larkevent "github.com/larksuite/oapi-sdk-go/v3/event"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

func TestSDKClientFacadeConnect(t *testing.T) {
	t.Run("connect waits for ready signal", func(t *testing.T) {
		facade := &fakeSDKFacade{}
		ready := make(chan struct{})
		allowReady := make(chan struct{})

		facade.startFunc = func(ctx context.Context, callbacks sdkFacadeCallbacks) error {
			close(ready)
			<-allowReady
			callbacks.OnReady()
			<-ctx.Done()
			return nil
		}

		client := newSDKClientWithFacade("app-id", "app-secret", facade)

		errCh := make(chan error, 1)
		go func() {
			errCh <- client.Connect(context.Background())
		}()

		select {
		case <-ready:
		case <-time.After(time.Second):
			t.Fatal("facade was not started")
		}

		if client.IsConnected() {
			t.Fatal("client should not report connected before ready")
		}

		close(allowReady)

		select {
		case err := <-errCh:
			if err != nil {
				t.Fatalf("Connect() error = %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("Connect() did not return after ready")
		}

		if !client.IsConnected() {
			t.Fatal("client should report connected after ready")
		}

		if err := client.Disconnect(); err != nil {
			t.Fatalf("Disconnect() error = %v", err)
		}
		if client.IsConnected() {
			t.Fatal("client should be disconnected after Disconnect()")
		}
	})

	t.Run("connect propagates authentication failure", func(t *testing.T) {
		wantErr := errors.New("auth failed")
		facade := &fakeSDKFacade{
			startFunc: func(ctx context.Context, callbacks sdkFacadeCallbacks) error {
				return wantErr
			},
		}

		client := newSDKClientWithFacade("app-id", "bad-secret", facade)
		err := client.Connect(context.Background())
		if !errors.Is(err, wantErr) {
			t.Fatalf("Connect() error = %v, want %v", err, wantErr)
		}
		if client.IsConnected() {
			t.Fatal("client should stay disconnected after auth failure")
		}
	})

	t.Run("connect returns context cancellation before ready", func(t *testing.T) {
		facade := &fakeSDKFacade{
			startFunc: func(ctx context.Context, callbacks sdkFacadeCallbacks) error {
				<-ctx.Done()
				return ctx.Err()
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		client := newSDKClientWithFacade("app-id", "app-secret", facade)
		err := client.Connect(ctx)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Connect() error = %v, want context canceled", err)
		}
		if client.IsConnected() {
			t.Fatal("client should stay disconnected after context cancellation")
		}
	})

	t.Run("connect is a no-op when already connected", func(t *testing.T) {
		facade := &fakeSDKFacade{}
		facade.startFunc = func(ctx context.Context, callbacks sdkFacadeCallbacks) error {
			callbacks.OnReady()
			<-ctx.Done()
			return nil
		}

		client := newSDKClientWithFacade("app-id", "app-secret", facade)
		if err := client.Connect(context.Background()); err != nil {
			t.Fatalf("first Connect() error = %v", err)
		}
		if err := client.Connect(context.Background()); err != nil {
			t.Fatalf("second Connect() error = %v", err)
		}
		if facade.startCalls != 1 {
			t.Fatalf("startCalls = %d, want 1", facade.startCalls)
		}
	})
}

func TestSDKClientFacadeStateTransitions(t *testing.T) {
	facade := &fakeSDKFacade{}
	reconnected := make(chan struct{}, 1)
	facade.startFunc = func(ctx context.Context, callbacks sdkFacadeCallbacks) error {
		callbacks.OnReady()
		callbacks.OnDisconnected(errors.New("socket dropped"))
		if clientReady(callbacks) {
			reconnected <- struct{}{}
		}
		callbacks.OnReady()
		<-ctx.Done()
		return nil
	}

	client := newSDKClientWithFacade("app-id", "app-secret", facade)
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	select {
	case <-reconnected:
	case <-time.After(time.Second):
		t.Fatal("expected reconnect transition")
	}

	if !client.IsConnected() {
		t.Fatal("client should report connected after reconnect")
	}
}

func TestSDKClientFacadeEventBridge(t *testing.T) {
	facade := &fakeSDKFacade{}
	connected := make(chan struct{})
	facade.startFunc = func(ctx context.Context, callbacks sdkFacadeCallbacks) error {
		callbacks.OnReady()
		close(connected)
		if err := callbacks.OnMessage(ctx, newSDKTextEvent("evt-1", "om-1", "oc-chat", "hello")); err != nil {
			return err
		}
		<-ctx.Done()
		return nil
	}

	client := newSDKClientWithFacade("app-id", "app-secret", facade)

	eventCh := make(chan *MessageReceiveEvent, 1)
	client.OnEvent(func(ctx context.Context, event *MessageReceiveEvent) error {
		eventCh <- event
		return errors.New("route failed")
	})

	err := client.Connect(context.Background())
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	select {
	case <-connected:
	case <-time.After(time.Second):
		t.Fatal("client did not become ready")
	}

	select {
	case event := <-eventCh:
		if event.EventID != "evt-1" {
			t.Fatalf("event.EventID = %q, want evt-1", event.EventID)
		}
		if event.ChatID != "oc-chat" {
			t.Fatalf("event.ChatID = %q, want oc-chat", event.ChatID)
		}
	case <-time.After(time.Second):
		t.Fatal("event handler was not called")
	}

	if !client.IsConnected() {
		t.Fatal("event handler failure should not break connection state")
	}
	if facade.sendCalls != 0 {
		t.Fatalf("unexpected sendCalls = %d", facade.sendCalls)
	}
}

func TestSDKClientFacadeSendText(t *testing.T) {
	t.Run("send text requires ready connection", func(t *testing.T) {
		client := newSDKClientWithFacade("app-id", "app-secret", &fakeSDKFacade{})
		err := client.SendText(context.Background(), "oc-chat", `{"text":"hello"}`)
		if !errors.Is(err, ErrClientNotReady) {
			t.Fatalf("SendText() error = %v, want %v", err, ErrClientNotReady)
		}
	})

	t.Run("send text delegates to facade", func(t *testing.T) {
		facade := &fakeSDKFacade{}
		facade.startFunc = func(ctx context.Context, callbacks sdkFacadeCallbacks) error {
			callbacks.OnReady()
			<-ctx.Done()
			return nil
		}

		client := newSDKClientWithFacade("app-id", "app-secret", facade)
		if err := client.Connect(context.Background()); err != nil {
			t.Fatalf("Connect() error = %v", err)
		}

		if err := client.SendText(context.Background(), "oc-chat", `{"text":"hello"}`); err != nil {
			t.Fatalf("SendText() error = %v", err)
		}
		if facade.lastSendChatID != "oc-chat" {
			t.Fatalf("lastSendChatID = %q, want oc-chat", facade.lastSendChatID)
		}
		if facade.lastSendContent != `{"text":"hello"}` {
			t.Fatalf("lastSendContent = %q", facade.lastSendContent)
		}
	})
}

type fakeSDKFacade struct {
	mu sync.Mutex

	startCalls int
	stopCalls  int
	sendCalls  int

	lastSendChatID  string
	lastSendContent string

	startFunc func(ctx context.Context, callbacks sdkFacadeCallbacks) error
	stopFunc  func(ctx context.Context) error
	sendFunc  func(ctx context.Context, chatID, content string) error
}

func (f *fakeSDKFacade) Start(ctx context.Context, callbacks sdkFacadeCallbacks) error {
	f.mu.Lock()
	f.startCalls++
	startFunc := f.startFunc
	f.mu.Unlock()
	if startFunc == nil {
		return nil
	}
	return startFunc(ctx, callbacks)
}

func (f *fakeSDKFacade) Stop(ctx context.Context) error {
	f.mu.Lock()
	f.stopCalls++
	stopFunc := f.stopFunc
	f.mu.Unlock()
	if stopFunc == nil {
		return nil
	}
	return stopFunc(ctx)
}

func (f *fakeSDKFacade) SendText(ctx context.Context, chatID, content string) error {
	f.mu.Lock()
	f.sendCalls++
	f.lastSendChatID = chatID
	f.lastSendContent = content
	sendFunc := f.sendFunc
	f.mu.Unlock()
	if sendFunc == nil {
		return nil
	}
	return sendFunc(ctx, chatID, content)
}

func newSDKTextEvent(eventID, messageID, chatID, text string) *larkim.P2MessageReceiveV1 {
	messageType := "text"
	chatType := "p2p"
	senderType := "user"
	openID := "ou-user"
	content := `{"text":"` + text + `"}`
	createTime := "1710000000000"

	return &larkim.P2MessageReceiveV1{
		EventV2Base: &larkevent.EventV2Base{
			Header: &larkevent.EventHeader{
				EventID:   eventID,
				EventType: "im.message.receive_v1",
			},
		},
		Event: &larkim.P2MessageReceiveV1Data{
			Sender: &larkim.EventSender{
				SenderId: &larkim.UserId{
					OpenId: &openID,
				},
				SenderType: &senderType,
			},
			Message: &larkim.EventMessage{
				MessageId:   &messageID,
				ChatId:      &chatID,
				ChatType:    &chatType,
				MessageType: &messageType,
				Content:     &content,
				CreateTime:  &createTime,
			},
		},
	}
}

func clientReady(callbacks sdkFacadeCallbacks) bool {
	return callbacks.OnReady != nil && callbacks.OnDisconnected != nil
}
