package core

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// TestRouterErrors 测试路由器错误定义存在
func TestRouterErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{
			name: "ErrNoHandler exists",
			err:  ErrNoHandler,
			msg:  "no handler registered",
		},
		{
			name: "ErrNilHandler exists",
			err:  ErrNilHandler,
			msg:  "handler cannot be nil",
		},
		{
			name: "ErrHandlerPanic exists",
			err:  ErrHandlerPanic,
			msg:  "handler panicked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s: error should not be nil", tt.name)
			}
		})
	}

	// Verify errors can be checked with errors.Is
	t.Run("errors are comparable with errors.Is", func(t *testing.T) {
		if !errors.Is(ErrNoHandler, ErrNoHandler) {
			t.Error("ErrNoHandler should be comparable with errors.Is")
		}
		if !errors.Is(ErrNilHandler, ErrNilHandler) {
			t.Error("ErrNilHandler should be comparable with errors.Is")
		}
		if !errors.Is(ErrHandlerPanic, ErrHandlerPanic) {
			t.Error("ErrHandlerPanic should be comparable with errors.Is")
		}
	})
}

// TestNewRouter 测试创建新路由器
func TestNewRouter(t *testing.T) {
	t.Run("creates new router", func(t *testing.T) {
		r := NewRouter()
		if r == nil {
			t.Error("NewRouter should not return nil")
		}
	})

	t.Run("new router has no handlers", func(t *testing.T) {
		r := NewRouter()
		if r.HasHandler(MessageTypeText) {
			t.Error("new router should not have any handlers")
		}
		if r.HasHandler(MessageTypeVoice) {
			t.Error("new router should not have any handlers")
		}
		if r.HasHandler(MessageTypeImage) {
			t.Error("new router should not have any handlers")
		}
		if r.HasHandler(MessageTypeCommand) {
			t.Error("new router should not have any handlers")
		}
	})
}

// mockHandler 用于测试的 mock 处理器
func mockHandler(ctx context.Context, msg *Message) error {
	return nil
}

// TestRouterRegister 测试处理器注册
func TestRouterRegister(t *testing.T) {
	t.Run("register handler", func(t *testing.T) {
		r := NewRouter()
		err := r.Register(MessageTypeText, mockHandler)
		if err != nil {
			t.Errorf("Register should not return error, got: %v", err)
		}
		if !r.HasHandler(MessageTypeText) {
			t.Error("HasHandler should return true after register")
		}
	})

	t.Run("register multiple handlers", func(t *testing.T) {
		r := NewRouter()
		r.Register(MessageTypeText, mockHandler)
		r.Register(MessageTypeVoice, mockHandler)
		r.Register(MessageTypeImage, mockHandler)

		if !r.HasHandler(MessageTypeText) {
			t.Error("should have text handler")
		}
		if !r.HasHandler(MessageTypeVoice) {
			t.Error("should have voice handler")
		}
		if !r.HasHandler(MessageTypeImage) {
			t.Error("should have image handler")
		}
	})

	t.Run("override handler", func(t *testing.T) {
		r := NewRouter()
		called := false
		handler1 := func(ctx context.Context, msg *Message) error {
			called = true
			return nil
		}
		r.Register(MessageTypeText, handler1)

		called = false
		handler2 := func(ctx context.Context, msg *Message) error {
			called = true
			return nil
		}
		r.Register(MessageTypeText, handler2)

		// After override, only handler2 should be called
		msg := NewTextMessage("test", "user", "hello")
		r.Route(context.Background(), msg)

		if !called {
			t.Error("handler2 should be called after override")
		}
	})
}

// TestRouterRegisterNil 测试注册 nil 处理器
func TestRouterRegisterNil(t *testing.T) {
	t.Run("register nil handler returns error", func(t *testing.T) {
		r := NewRouter()
		// First register a valid handler
		r.Register(MessageTypeText, mockHandler)

		// Try to register nil
		err := r.Register(MessageTypeText, nil)
		if err == nil {
			t.Error("Register with nil should return error")
		}
		if !errors.Is(err, ErrNilHandler) {
			t.Errorf("error should be ErrNilHandler, got: %v", err)
		}

		// Original handler should still exist
		if !r.HasHandler(MessageTypeText) {
			t.Error("original handler should still exist")
		}
	})

	t.Run("register nil on empty type returns error", func(t *testing.T) {
		r := NewRouter()
		err := r.Register(MessageTypeVoice, nil)
		if err == nil {
			t.Error("Register with nil should return error")
		}
		if !errors.Is(err, ErrNilHandler) {
			t.Errorf("error should be ErrNilHandler, got: %v", err)
		}
	})
}

// TestRouterUnregister 测试处理器注销
func TestRouterUnregister(t *testing.T) {
	t.Run("unregister existing handler", func(t *testing.T) {
		r := NewRouter()
		r.Register(MessageTypeText, mockHandler)

		if !r.HasHandler(MessageTypeText) {
			t.Fatal("handler should be registered")
		}

		r.Unregister(MessageTypeText)

		if r.HasHandler(MessageTypeText) {
			t.Error("handler should be unregistered")
		}
	})

	t.Run("unregister non-existing handler", func(t *testing.T) {
		r := NewRouter()
		// Should not panic or error
		r.Unregister(MessageTypeText)
		r.Unregister(MessageTypeVoice)

		// Still no handlers
		if r.HasHandler(MessageTypeText) {
			t.Error("should not have handler")
		}
	})

	t.Run("unregister and re-register", func(t *testing.T) {
		r := NewRouter()
		r.Register(MessageTypeText, mockHandler)
		r.Unregister(MessageTypeText)
		r.Register(MessageTypeText, mockHandler)

		if !r.HasHandler(MessageTypeText) {
			t.Error("handler should be registered again")
		}
	})
}

// TestRouterRoute 测试消息路由基本功能
func TestRouterRoute(t *testing.T) {
	t.Run("route to correct handler", func(t *testing.T) {
		r := NewRouter()
		var receivedMsg *Message
		r.Register(MessageTypeText, func(ctx context.Context, msg *Message) error {
			receivedMsg = msg
			return nil
		})

		msg := NewTextMessage("feishu", "user123", "hello")
		err := r.Route(context.Background(), msg)

		if err != nil {
			t.Errorf("Route should not return error, got: %v", err)
		}
		if receivedMsg == nil {
			t.Fatal("handler should have received message")
		}
		if receivedMsg.Content != "hello" {
			t.Errorf("expected content 'hello', got: %q", receivedMsg.Content)
		}
		if receivedMsg.Platform != "feishu" {
			t.Errorf("expected platform 'feishu', got: %q", receivedMsg.Platform)
		}
	})

	t.Run("route different message types", func(t *testing.T) {
		r := NewRouter()

		textCalled := false
		voiceCalled := false
		imageCalled := false

		r.Register(MessageTypeText, func(ctx context.Context, msg *Message) error {
			textCalled = true
			return nil
		})
		r.Register(MessageTypeVoice, func(ctx context.Context, msg *Message) error {
			voiceCalled = true
			return nil
		})
		r.Register(MessageTypeImage, func(ctx context.Context, msg *Message) error {
			imageCalled = true
			return nil
		})

		r.Route(context.Background(), NewTextMessage("test", "user", "text"))
		if !textCalled {
			t.Error("text handler should be called")
		}

		r.Route(context.Background(), NewVoiceMessage("test", "user", "voice"))
		if !voiceCalled {
			t.Error("voice handler should be called")
		}

		r.Route(context.Background(), NewImageMessage("test", "user", "image"))
		if !imageCalled {
			t.Error("image handler should be called")
		}
	})
}

// TestRouterRouteNoHandler 测试路由到未注册的处理器
func TestRouterRouteNoHandler(t *testing.T) {
	t.Run("no handler returns ErrNoHandler", func(t *testing.T) {
		r := NewRouter()
		msg := NewTextMessage("test", "user", "hello")
		err := r.Route(context.Background(), msg)

		if err == nil {
			t.Error("Route should return error when no handler")
		}
		if !errors.Is(err, ErrNoHandler) {
			t.Errorf("error should be ErrNoHandler, got: %v", err)
		}
	})

	t.Run("no handler for specific type", func(t *testing.T) {
		r := NewRouter()
		r.Register(MessageTypeText, mockHandler)

		// Route voice message, which has no handler
		msg := NewVoiceMessage("test", "user", "hello")
		err := r.Route(context.Background(), msg)

		if !errors.Is(err, ErrNoHandler) {
			t.Errorf("error should be ErrNoHandler, got: %v", err)
		}
	})

	t.Run("no handler after unregister", func(t *testing.T) {
		r := NewRouter()
		r.Register(MessageTypeText, mockHandler)
		r.Unregister(MessageTypeText)

		msg := NewTextMessage("test", "user", "hello")
		err := r.Route(context.Background(), msg)

		if !errors.Is(err, ErrNoHandler) {
			t.Errorf("error should be ErrNoHandler, got: %v", err)
		}
	})
}

// TestRouterRouteError 测试处理器返回错误
func TestRouterRouteError(t *testing.T) {
	t.Run("handler returns error", func(t *testing.T) {
		r := NewRouter()
		expectedErr := errors.New("handler error")
		r.Register(MessageTypeText, func(ctx context.Context, msg *Message) error {
			return expectedErr
		})

		msg := NewTextMessage("test", "user", "hello")
		err := r.Route(context.Background(), msg)

		if err == nil {
			t.Error("Route should return handler error")
		}
		if !errors.Is(err, expectedErr) {
			t.Errorf("error should be handler error, got: %v", err)
		}
	})

	t.Run("handler returns different errors", func(t *testing.T) {
		r := NewRouter()

		customErr := errors.New("custom error")
		r.Register(MessageTypeText, func(ctx context.Context, msg *Message) error {
			return customErr
		})

		msg := NewTextMessage("test", "user", "hello")
		err := r.Route(context.Background(), msg)

		if !errors.Is(err, customErr) {
			t.Errorf("error should be custom error, got: %v", err)
		}
	})
}

// TestRouterRoutePanic 测试处理器 panic 恢复
func TestRouterRoutePanic(t *testing.T) {
	t.Run("handler panic returns ErrHandlerPanic", func(t *testing.T) {
		r := NewRouter()
		r.Register(MessageTypeText, func(ctx context.Context, msg *Message) error {
			panic("handler panic!")
		})

		msg := NewTextMessage("test", "user", "hello")
		err := r.Route(context.Background(), msg)

		if err == nil {
			t.Error("Route should return error when handler panics")
		}
		if !errors.Is(err, ErrHandlerPanic) {
			t.Errorf("error should wrap ErrHandlerPanic, got: %v", err)
		}
	})

	t.Run("router still works after panic", func(t *testing.T) {
		r := NewRouter()
		panicCalled := false
		r.Register(MessageTypeText, func(ctx context.Context, msg *Message) error {
			if !panicCalled {
				panicCalled = true
				panic("first call panic")
			}
			return nil
		})

		msg := NewTextMessage("test", "user", "hello")

		// First call panics
		err1 := r.Route(context.Background(), msg)
		if !errors.Is(err1, ErrHandlerPanic) {
			t.Errorf("first call should return ErrHandlerPanic, got: %v", err1)
		}

		// Router should still work
		err2 := r.Route(context.Background(), msg)
		if err2 != nil {
			t.Errorf("second call should succeed, got: %v", err2)
		}
	})

	t.Run("panic with different types", func(t *testing.T) {
		r := NewRouter()
		r.Register(MessageTypeText, func(ctx context.Context, msg *Message) error {
			panic(42) // panic with int
		})

		msg := NewTextMessage("test", "user", "hello")
		err := r.Route(context.Background(), msg)

		if !errors.Is(err, ErrHandlerPanic) {
			t.Errorf("error should wrap ErrHandlerPanic, got: %v", err)
		}
	})
}

// TestRouterConcurrentRoute 测试并发路由
func TestRouterConcurrentRoute(t *testing.T) {
	t.Run("concurrent route calls", func(t *testing.T) {
		r := NewRouter()
		var callCount atomic.Int64
		r.Register(MessageTypeText, func(ctx context.Context, msg *Message) error {
			callCount.Add(1)
			return nil
		})

		const goroutines = 100
		done := make(chan bool, goroutines)

		for i := 0; i < goroutines; i++ {
			go func() {
				msg := NewTextMessage("test", "user", "hello")
				r.Route(context.Background(), msg)
				done <- true
			}()
		}

		for i := 0; i < goroutines; i++ {
			<-done
		}

		if callCount.Load() != goroutines {
			t.Errorf("expected %d calls, got %d", goroutines, callCount.Load())
		}
	})
}

// TestRouterConcurrentRegister 测试并发注册和路由
func TestRouterConcurrentRegister(t *testing.T) {
	t.Run("concurrent register and route", func(t *testing.T) {
		r := NewRouter()

		const goroutines = 50
		done := make(chan bool, goroutines*2)

		// Start register goroutines
		for i := 0; i < goroutines; i++ {
			go func(idx int) {
				msgType := MessageType(fmt.Sprintf("type%d", idx%4))
				r.Register(msgType, func(ctx context.Context, msg *Message) error {
					return nil
				})
				done <- true
			}(i)
		}

		// Start route goroutines
		for i := 0; i < goroutines; i++ {
			go func(idx int) {
				msgType := MessageType(fmt.Sprintf("type%d", idx%4))
				msg := &Message{Type: msgType}
				r.Route(context.Background(), msg) // May return ErrNoHandler, that's ok
				done <- true
			}(i)
		}

		for i := 0; i < goroutines*2; i++ {
			<-done
		}
	})

	t.Run("concurrent register unregister and route", func(t *testing.T) {
		r := NewRouter()
		r.Register(MessageTypeText, mockHandler)

		const goroutines = 50
		done := make(chan bool, goroutines*3)

		// Register goroutines
		for i := 0; i < goroutines; i++ {
			go func() {
				r.Register(MessageTypeText, mockHandler)
				done <- true
			}()
		}

		// Unregister goroutines
		for i := 0; i < goroutines; i++ {
			go func() {
				r.Unregister(MessageTypeText)
				done <- true
			}()
		}

		// Route goroutines
		for i := 0; i < goroutines; i++ {
			go func() {
				msg := NewTextMessage("test", "user", "hello")
				r.Route(context.Background(), msg) // May return ErrNoHandler, that's ok
				done <- true
			}()
		}

		for i := 0; i < goroutines*3; i++ {
			<-done
		}
	})
}

// TestRouterContextCancel 测试 Context 取消
func TestRouterContextCancel(t *testing.T) {
	t.Run("context is passed to handler", func(t *testing.T) {
		r := NewRouter()
		var receivedCtx context.Context
		r.Register(MessageTypeText, func(ctx context.Context, msg *Message) error {
			receivedCtx = ctx
			return nil
		})

		ctx := context.WithValue(context.Background(), "key", "value")
		msg := NewTextMessage("test", "user", "hello")
		r.Route(ctx, msg)

		if receivedCtx == nil {
			t.Fatal("handler should receive context")
		}
		if receivedCtx.Value("key") != "value" {
			t.Error("context value should be passed through")
		}
	})

	t.Run("handler can check cancelled context", func(t *testing.T) {
		r := NewRouter()
		var ctxErr error
		r.Register(MessageTypeText, func(ctx context.Context, msg *Message) error {
			ctxErr = ctx.Err()
			return nil
		})

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		msg := NewTextMessage("test", "user", "hello")
		r.Route(ctx, msg)

		if ctxErr != context.Canceled {
			t.Errorf("context should be cancelled, got: %v", ctxErr)
		}
	})

	t.Run("handler respects context timeout", func(t *testing.T) {
		r := NewRouter()
		var ctxErr error
		r.Register(MessageTypeText, func(ctx context.Context, msg *Message) error {
			time.Sleep(10 * time.Millisecond)
			ctxErr = ctx.Err()
			return nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		msg := NewTextMessage("test", "user", "hello")
		r.Route(ctx, msg)

		// After sleeping, context should be deadline exceeded
		if ctxErr != context.DeadlineExceeded {
			t.Errorf("context should be deadline exceeded, got: %v", ctxErr)
		}
	})
}
