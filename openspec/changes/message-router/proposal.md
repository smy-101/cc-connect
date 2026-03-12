# 消息路由

## Why

cc-connect 需要在不同来源（飞书、斜杠命令等）和处理器（AI 代理、命令处理器）之间分发消息。统一消息结构已实现，现在需要消息路由器将消息分发到正确的处理器。这是连接平台适配器和 AI 代理的核心枢纽，是阶段 1 核心消息系统的第二个能力。

## What Changes

- 新增 `internal/core/router.go`：消息路由器实现
- 新增 `internal/core/router_test.go`：TDD 测试文件
- 新增 `internal/core/errors.go`：路由相关错误定义
- 支持按消息类型（MessageType）注册处理器
- 支持消息分发和错误处理
- 并发安全，支持 context 取消

## Capabilities

### New Capabilities

- `message-router`: 消息路由能力，按消息类型分发消息到注册的处理器

### Modified Capabilities

无（这是新增能力，不修改现有规格）

## Impact

### 影响模块

- **core**: 新增路由器相关代码到 `internal/core/`

### 影响范围

- 属于 **阶段 1：核心消息系统** 的第二个切片
- 后续模块（飞书适配器、Claude Code 适配器、命令系统）将通过 Router 注册处理器

### 技术约束

- 不引入外部依赖，使用 Go 标准库
- 并发安全，使用 `sync.RWMutex`
- 支持 `context.Context` 进行取消和超时控制
- 处理器接口简单：`type Handler func(ctx context.Context, msg Message) error`

### 验收标准

- ✅ 支持按消息类型注册和注销处理器
- ✅ 路由消息到正确的处理器
- ✅ 未注册类型的消息返回明确的错误（`ErrNoHandler`）
- ✅ 处理器 panic 时返回 `ErrHandlerPanic`，不影响系统稳定性
- ✅ 并发注册和路由安全
- ✅ Context 取消时处理器可感知
- ✅ 测试覆盖率 > 85%

### 风险与缓解

| 风险 | 缓解措施 |
|------|----------|
| 处理器 panic 导致系统崩溃 | 使用 recover 捕获，返回 ErrHandlerPanic |
| 并发访问导致竞态条件 | 使用 sync.RWMutex 保护 handlers map |
| 注册 nil 处理器 | 返回错误，显式拒绝 |
