# 飞书平台适配器

## Why

当前 cc-connect 已完成核心消息系统（阶段 1），包括统一消息结构、消息路由、会话管理和配置管理。为实现用户通过飞书聊天控制本地 AI 代理的核心价值，需要实现飞书平台适配器（阶段 2），建立飞书与 cc-connect 之间的消息桥梁。

飞书长连接模式具有独特优势：无需公网 IP、无需内网穿透、内置加密传输，非常适合本地开发和部署场景。

## What Changes

### 新增能力

1. **飞书 WebSocket 客户端**：基于官方 Go SDK 建立与飞书开放平台的长连接
2. **事件接收与解析**：订阅 `im.message.receive_v1` 事件，解析消息和 @提及
3. **消息格式转换**：飞书消息格式 ↔ 统一消息模型双向转换
4. **消息发送**：通过飞书 API 发送文本消息回复用户
5. **连接管理**：心跳检测、自动重连、连接状态管理

### 修改范围

- 新增 `internal/platform/feishu/` 目录及相关实现
- 与现有 `internal/core/router.go` 集成，将飞书事件注入消息路由
- 扩展 `internal/core/config.go` 已支持的 FeishuConfig 配置项使用

## Capabilities

### New Capabilities

- `feishu-websocket-client`: 飞书 WebSocket 长连接客户端，负责连接建立、心跳、重连、事件接收
- `feishu-message-converte`: 飞书消息格式与统一消息模型的双向转换器
- `feishu-event-handler`: 飞书事件处理器，解析 im.message.receive_v1 事件并提取 @提及

### Modified Capabilities

无。本次变更为新增平台适配器，不修改现有核心能力的规格要求。

## Impact

### 代码影响

| 模块 | 影响类型 | 说明 |
|------|----------|------|
| `internal/platform/feishu/` | 新增 | 飞书适配器核心实现 |
| `internal/core/router.go` | 集成 | 接收飞书适配器注入的消息 |
| `cmd/cc-connect/` | 新增 | 主程序启动飞书客户端 |

### 外部依赖

| 依赖 | 版本 | 用途 |
|------|------|------|
| `github.com/larksuite/oapi-sdk-go/v3` | latest | 飞书官方 Go SDK |

### 配置变更

使用已有的 `FeishuConfig` 配置结构：
- `app_id`: 飞书应用 ID
- `app_secret`: 飞书应用密钥
- `enabled`: 是否启用飞书平台

## 验收标准

### 功能验收

1. **连接建立**：使用有效 APP_ID/APP_SECRET 成功建立 WebSocket 长连接
2. **消息接收**：能接收单聊消息和群聊 @机器人消息
3. **@提及解析**：正确解析消息中的 @提及 用户信息
4. **消息转换**：飞书文本消息正确转换为统一消息模型的 text 类型
5. **消息发送**：能通过 API 回复文本消息到飞书
6. **断线重连**：网络中断后自动重连，不丢失消息处理能力

### 质量验收

1. **测试覆盖率**：飞书适配器核心代码覆盖率 ≥ 85%
2. **Mock 测试**：WebSocket 连接使用 mock，不依赖真实网络
3. **并发安全**：`go test -race` 通过
4. **错误处理**：连接失败、消息解析失败有明确的错误日志

## 风险与缓解

| 风险 | 影响 | 缓解策略 |
|------|------|----------|
| SDK 版本兼容性 | SDK API 变化导致编译失败 | 定义接口抽象层隔离 SDK 实现 |
| 3秒超时限制 | 消息处理超时触发重推 | 快速响应 ACK，业务逻辑异步处理 |
| 敏感权限审核 | im:message.group_msg 需飞书审核 | MVP 阶段使用 group_at_msg 权限 |
| 集群模式分发 | 多实例时消息随机分发 | MVP 单实例部署，后续可加消息队列 |
