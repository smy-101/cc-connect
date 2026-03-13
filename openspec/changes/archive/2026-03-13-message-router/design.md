# 消息路由 - 技术设计

## Context

统一消息结构（`Message`）已实现，支持 `text`、`voice`、`image`、`command` 四种类型。现在需要实现消息路由器，将消息分发到正确的处理器。

当前状态：
- `internal/core/message.go` - 消息结构体和构造函数
- `internal/core/message_id.go` - ID 生成器
- 消息类型定义：`MessageTypeText`, `MessageTypeVoice`, `MessageTypeImage`, `MessageTypeCommand`

路由器是核心域组件，位于 `internal/core/`，不依赖平台适配层或代理适配层。

## Goals / Non-Goals

**Goals:**

- 实现按消息类型路由的简单路由器
- 支持处理器注册和注销
- 并发安全
- 支持 context 取消
- 处理器 panic 恢复，保证系统稳定性
- 可测试性高，覆盖率 > 85%

**Non-Goals:**

- 会话级路由（属于会话管理变更）
- 多代理中继（属于阶段 6 高级功能）
- 流式响应处理（属于 Claude Code 适配器变更）
- 消息优先级或队列（当前不需要）

## Decisions

### 1. 处理器接口设计

**决定：使用函数类型**

```go
type Handler func(ctx context.Context, msg Message) error
```

**理由：**
- 简单直接，符合 Go 习惯
- 函数可以是有闭包，也可以是方法
- 不需要额外的接口类型

**替代方案：**
- 接口类型 `type Handler interface { Handle(...) error }` - 过度抽象，当前不需要多方法

### 2. 路由策略

**决定：按消息类型路由**

```go
func (r *Router) Route(ctx context.Context, msg Message) error {
    handler := r.handlers[msg.Type]
    if handler == nil {
        return ErrNoHandler
    }
    return handler(ctx, msg)
}
```

**理由：**
- MVP 阶段最简单的策略
- 消息类型是已知的固定字段
- 后续可扩展为按类型 + 会话的组合策略

**替代方案：**
- 按平台路由 - 不够灵活，同一平台可能有不同消息类型
- 按会话路由 - 需要先实现会话管理

### 3. 并发安全

**决定：使用 `sync.RWMutex`**

```go
type Router struct {
    handlers map[MessageType]Handler
    mu       sync.RWMutex
}
```

**理由：**
- 读多写少场景，RWMutex 性能更好
- 标准库实现，无外部依赖

### 4. Panic 恢复

**决定：捕获处理器 panic，返回 ErrHandlerPanic**

```go
func (r *Router) Route(ctx context.Context, msg Message) (err error) {
    defer func() {
        if rec := recover(); rec != nil {
            err = fmt.Errorf("%w: %v", ErrHandlerPanic, rec)
        }
    }()
    // ...
}
```

**理由：**
- 单个处理器崩溃不应影响整个系统
- 错误信息保留 panic 详情，便于调试

### 5. 错误类型

**决定：定义哨兵错误**

```go
var (
    ErrNoHandler   = errors.New("no handler registered for message type")
    ErrNilHandler  = errors.New("handler cannot be nil")
    ErrHandlerPanic = errors.New("handler panicked")
)
```

**理由：**
- 支持 `errors.Is()` 检查
- 明确的错误类型，便于调用方处理

## Risks / Trade-offs

| 风险 | 缓解措施 |
|------|----------|
| 处理器执行时间过长 | 通过 context 传递超时，处理器自行检查 `ctx.Done()` |
| 处理器内存泄漏 | 无法在路由器层面解决，依赖处理器正确实现 |
| 大量并发消息 | 路由器本身无状态，瓶颈在处理器；后续可引入队列 |
| 处理器返回的错误信息丢失 | 直接返回原始 error，不包装 |

## 接口定义

```go
// internal/core/router.go

package core

import (
    "context"
    "errors"
    "fmt"
    "sync"
)

// 错误定义
var (
    ErrNoHandler    = errors.New("no handler registered for message type")
    ErrNilHandler   = errors.New("handler cannot be nil")
    ErrHandlerPanic = errors.New("handler panicked")
)

// Handler 处理消息的函数签名
type Handler func(ctx context.Context, msg Message) error

// Router 消息路由器
type Router struct {
    handlers map[MessageType]Handler
    mu       sync.RWMutex
}

// NewRouter 创建新路由器
func NewRouter() *Router

// Register 注册处理器
func (r *Router) Register(mt MessageType, h Handler) error

// Unregister 注销处理器
func (r *Router) Unregister(mt MessageType)

// Route 路由消息到对应处理器
func (r *Router) Route(ctx context.Context, msg Message) error

// HasHandler 检查是否已注册处理器
func (r *Router) HasHandler(mt MessageType) bool
```

## 状态流转

```
┌─────────────┐     ┌─────────────────┐     ┌─────────────────┐
│ 消息到达     │────▶│ 查找处理器       │────▶│ 执行处理器       │
│ Route(msg)  │     │ handlers[type]  │     │ handler(ctx,msg)│
└─────────────┘     └─────────────────┘     └─────────────────┘
                           │                       │
                           ▼                       ▼
                    ┌─────────────┐          ┌─────────────┐
                    │ 无处理器     │          │ 处理成功     │
                    │ ErrNoHandler│          │ nil         │
                    └─────────────┘          └─────────────┘
                                                    │
                           ┌────────────────────────┴────────┐
                           ▼                                 ▼
                    ┌─────────────┐                   ┌─────────────┐
                    │ 处理器错误   │                   │ 处理器 panic │
                    │ 返回 error  │                   │ ErrHandlerPanic│
                    └─────────────┘                   └─────────────┘
```

## 可测试性

路由器设计为纯内存操作，无 IO 依赖，测试策略：

1. **单元测试**：使用 mock 处理器验证路由逻辑
2. **并发测试**：使用 `go test -race` 验证竞态条件
3. **覆盖率**：`go test -cover` 验证 > 85%

```go
// 测试示例
func TestRouterRoute(t *testing.T) {
    r := NewRouter()

    // 注册 mock 处理器
    called := false
    r.Register(MessageTypeText, func(ctx context.Context, msg Message) error {
        called = true
        return nil
    })

    // 路由消息
    msg := NewTextMessage("test", "user", "hello")
    err := r.Route(context.Background(), msg)

    assert.NoError(t, err)
    assert.True(t, called)
}
```
