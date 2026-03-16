package core

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// 路由器错误定义
var (
	// ErrNoHandler 没有为消息类型注册处理器
	ErrNoHandler = errors.New("no handler registered for message type")
	// ErrNilHandler 处理器不能为 nil
	ErrNilHandler = errors.New("handler cannot be nil")
	// ErrHandlerPanic 处理器发生 panic
	ErrHandlerPanic = errors.New("handler panicked")
)

// Handler 处理消息的函数签名
type Handler func(ctx context.Context, msg *Message) error

// SessionHandler 带会话上下文的处理器函数签名
type SessionHandler func(ctx context.Context, msg *Message, session *Session) error

// Router 消息路由器
type Router struct {
	handlers        map[MessageType]Handler
	sessionHandlers map[MessageType]SessionHandler
	sessions        *SessionManager
	mu              sync.RWMutex
}

// NewRouter 创建新路由器
func NewRouter() *Router {
	return &Router{
		handlers:        make(map[MessageType]Handler),
		sessionHandlers: make(map[MessageType]SessionHandler),
		sessions:        NewSessionManager(DefaultSessionConfig()),
	}
}

// Sessions 返回 SessionManager
func (r *Router) Sessions() *SessionManager {
	return r.sessions
}

// HasHandler 检查是否已注册处理器
func (r *Router) HasHandler(mt MessageType) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.handlers[mt]
	return ok
}

// HasSessionHandler 检查是否已注册带会话的处理器
func (r *Router) HasSessionHandler(mt MessageType) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.sessionHandlers[mt]
	return ok
}

// Register 注册处理器
func (r *Router) Register(mt MessageType, h Handler) error {
	if h == nil {
		return ErrNilHandler
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[mt] = h
	return nil
}

// RegisterSessionHandler 注册带会话的处理器
func (r *Router) RegisterSessionHandler(mt MessageType, h SessionHandler) error {
	if h == nil {
		return ErrNilHandler
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessionHandlers[mt] = h
	return nil
}

// Unregister 注销处理器
func (r *Router) Unregister(mt MessageType) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.handlers, mt)
	delete(r.sessionHandlers, mt)
}

// Route 路由消息到对应处理器
func (r *Router) Route(ctx context.Context, msg *Message) (err error) {
	r.mu.RLock()
	handler := r.handlers[msg.Type]
	r.mu.RUnlock()

	if handler == nil {
		return ErrNoHandler
	}

	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("%w: %v", ErrHandlerPanic, rec)
		}
	}()

	return handler(ctx, msg)
}

// RouteWithSession 路由消息并传递会话上下文
func (r *Router) RouteWithSession(ctx context.Context, msg *Message) (err error) {
	r.mu.RLock()
	handler := r.sessionHandlers[msg.Type]
	r.mu.RUnlock()

	if handler == nil {
		return ErrNoHandler
	}

	// 派生会话 ID 并获取/创建会话
	sessionID := DeriveSessionID(msg)
	session := r.sessions.GetOrCreate(sessionID)

	// 更新活跃时间
	r.sessions.Update(sessionID, func(s *Session) {
		s.Touch()
	})

	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("%w: %v", ErrHandlerPanic, rec)
		}
	}()

	return handler(ctx, msg, session)
}
