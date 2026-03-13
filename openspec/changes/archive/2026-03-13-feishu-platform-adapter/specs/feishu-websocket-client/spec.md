# 飞书 WebSocket 客户端规格

## Purpose

飞书 WebSocket 客户端负责与飞书开放平台建立长连接，接收消息事件，并支持发送回复消息。使用飞书官方 Go SDK 实现连接管理。

## ADDED Requirements

### Requirement: 建立长连接

系统 SHALL 能够使用 APP_ID 和 APP_SECRET 与飞书开放平台建立 WebSocket 长连接。

#### Scenario: 成功建立连接
- **WHEN** 调用 `client.Connect(ctx)` 且配置有效的 APP_ID 和 APP_SECRET
- **THEN** 连接成功建立
- **THEN** `client.IsConnected()` 返回 `true`

#### Scenario: 无效凭证连接失败
- **WHEN** 调用 `client.Connect(ctx)` 且 APP_ID 或 APP_SECRET 无效
- **THEN** 返回认证错误
- **THEN** `client.IsConnected()` 返回 `false`

#### Scenario: Context 取消连接
- **WHEN** 调用 `client.Connect(ctx)` 且 ctx 在连接过程中被取消
- **THEN** 连接尝试被中断
- **THEN** 返回 context 取消错误

---

### Requirement: 断开连接

系统 SHALL 支持主动断开与飞书平台的连接。

#### Scenario: 主动断开连接
- **WHEN** 调用 `client.Disconnect()`
- **THEN** WebSocket 连接被关闭
- **THEN** `client.IsConnected()` 返回 `false`

#### Scenario: 断开已断开的连接
- **WHEN** 调用 `client.Disconnect()` 且连接已断开
- **THEN** 操作静默完成，不返回错误

---

### Requirement: 连接状态检查

系统 SHALL 提供连接状态查询能力。

#### Scenario: 查询已连接状态
- **WHEN** WebSocket 连接正常建立
- **THEN** `client.IsConnected()` 返回 `true`

#### Scenario: 查询未连接状态
- **WHEN** WebSocket 连接未建立或已断开
- **THEN** `client.IsConnected()` 返回 `false`

---

### Requirement: 事件回调注册

系统 SHALL 允许注册事件处理器来接收飞书消息事件。

#### Scenario: 注册事件处理器
- **WHEN** 调用 `client.OnEvent(handler)` 注册处理器
- **THEN** 后续收到的消息事件会调用该处理器

#### Scenario: 收到消息触发回调
- **WHEN** 飞书推送 `im.message.receive_v1` 事件
- **THEN** 已注册的事件处理器被调用
- **THEN** 处理器接收到解析后的事件结构

---

### Requirement: 发送文本消息

系统 SHALL 支持通过飞书 API 发送文本消息。

#### Scenario: 发送文本消息成功
- **WHEN** 调用 `client.SendText(ctx, chatID, "Hello")` 且连接正常
- **THEN** 消息发送成功
- **THEN** 不返回错误

#### Scenario: 发送消息到无效会话
- **WHEN** 调用 `client.SendText(ctx, invalidChatID, "Hello")`
- **THEN** 返回错误，包含错误详情

#### Scenario: 发送消息时连接断开
- **WHEN** 调用 `client.SendText(ctx, chatID, "Hello")` 且连接已断开
- **THEN** 返回连接错误

---

### Requirement: 异步事件处理

系统 SHALL 异步处理消息事件，确保快速响应飞书 ACK。

#### Scenario: 事件异步处理
- **WHEN** 收到飞书消息事件
- **THEN** 事件处理器在 100ms 内返回（完成 ACK）
- **THEN** 业务逻辑在后台 goroutine 中执行

#### Scenario: 异步处理失败不影响 ACK
- **WHEN** 业务逻辑处理发生错误
- **THEN** 不影响事件处理器的正常返回
- **THEN** 错误被记录到日志

---

### Requirement: Mock 支持

系统 SHALL 提供可替换的接口设计，支持测试时使用 Mock 实现。

#### Scenario: 使用 Mock 客户端测试
- **WHEN** 在测试环境中创建 MockFeishuClient
- **THEN** 可以模拟连接、断开、发送消息等行为
- **THEN** 可以验证方法调用次数和参数

#### Scenario: Mock 事件触发
- **WHEN** 调用 Mock 客户端的 `SimulateMessageEvent(event)` 方法
- **THEN** 已注册的事件处理器被调用
- **THEN** 处理器接收到模拟的事件数据
