# 消息路由规格

## Purpose

消息路由器负责将统一消息分发到注册的处理器。它是连接平台适配器和 AI 代理的核心枢纽，支持按消息类型路由、并发安全和 Context 传递。

## Requirements

### Requirement: 处理器注册

系统 SHALL 允许按消息类型注册处理器函数。

#### Scenario: 注册文本消息处理器
- **WHEN** 调用 `router.Register(MessageTypeText, handler)` 注册处理器
- **THEN** 系统存储该处理器，后续 `HasHandler(MessageTypeText)` 返回 `true`

#### Scenario: 注册 nil 处理器
- **WHEN** 调用 `router.Register(MessageTypeText, nil)` 注册 nil 处理器
- **THEN** 系统返回 `ErrNilHandler` 错误，不修改现有处理器

#### Scenario: 覆盖已注册的处理器
- **WHEN** 对同一消息类型再次调用 `Register`
- **THEN** 新处理器覆盖旧处理器

### Requirement: 处理器注销

系统 SHALL 允许注销已注册的处理器。

#### Scenario: 注销已存在的处理器
- **WHEN** 调用 `router.Unregister(MessageTypeText)` 注销处理器
- **THEN** `HasHandler(MessageTypeText)` 返回 `false`

#### Scenario: 注销不存在的处理器
- **WHEN** 调用 `router.Unregister(MessageTypeText)` 但该类型未注册
- **THEN** 系统不报错，静默处理

### Requirement: 消息路由

系统 SHALL 将消息路由到对应类型的处理器。

#### Scenario: 路由到正确的处理器
- **WHEN** 调用 `router.Route(ctx, msg)` 且 `msg.Type` 已注册处理器
- **THEN** 系统调用该处理器并返回其结果

#### Scenario: 路由到不存在的处理器
- **WHEN** 调用 `router.Route(ctx, msg)` 且 `msg.Type` 未注册处理器
- **THEN** 系统返回 `ErrNoHandler` 错误

#### Scenario: 处理器返回错误
- **WHEN** 处理器返回非 nil 错误
- **THEN** `Route` 返回该错误

#### Scenario: 处理器 panic
- **WHEN** 处理器执行时发生 panic
- **THEN** 系统捕获 panic，返回 `ErrHandlerPanic` 包装错误，不影响后续路由

### Requirement: 并发安全

系统 SHALL 支持并发注册和路由操作。

#### Scenario: 并发路由
- **WHEN** 多个 goroutine 同时调用 `Route`
- **THEN** 所有调用正确完成，无竞态条件

#### Scenario: 并发注册和路由
- **WHEN** 一个 goroutine 调用 `Register` 同时另一个调用 `Route`
- **THEN** 操作正确完成，无竞态条件（`go test -race` 通过）

### Requirement: Context 支持

系统 SHALL 支持 context 传递给处理器。

#### Scenario: Context 取消
- **WHEN** 调用 `Route` 时传入已取消的 context
- **THEN** 处理器收到该 context，可自行检查 `ctx.Done()`

#### Scenario: Context 超时
- **WHEN** 调用 `Route` 时传入带超时的 context
- **THEN** 处理器收到该 context，可自行检查 `ctx.Err()`
