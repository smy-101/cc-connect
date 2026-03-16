# 飞书 WebSocket 客户端规格

## ADDED Requirements

### Requirement: 建立真实飞书长连接

系统 SHALL 使用飞书官方 Go SDK 与飞书开放平台建立真实 WebSocket 长连接，而不是仅在本地内存中标记连接成功。

#### Scenario: 使用有效凭证建立连接
- **WHEN** 调用 `client.Connect(ctx)` 且 `app_id`、`app_secret` 有效
- **THEN** 系统与飞书开放平台建立真实长连接
- **THEN** `client.IsConnected()` 返回 `true`

#### Scenario: 使用无效凭证建立连接失败
- **WHEN** 调用 `client.Connect(ctx)` 且飞书平台返回认证失败
- **THEN** `Connect` 返回认证错误
- **THEN** `client.IsConnected()` 返回 `false`

#### Scenario: 连接过程中 context 被取消
- **WHEN** 调用 `client.Connect(ctx)` 且 `ctx` 在连接建立前被取消
- **THEN** 连接尝试被中断
- **THEN** `Connect` 返回 `ctx` 相关错误

### Requirement: 连接就绪反馈

系统 SHALL 在长连接真正建立后提供明确的连接就绪反馈，以便操作者继续飞书后台的事件配置流程。

#### Scenario: 成功建立长连接后输出就绪反馈
- **WHEN** 应用启动并完成飞书长连接建立
- **THEN** 用户可见日志或状态输出中包含“飞书长连接已就绪”之类的明确反馈
- **THEN** 该反馈足以指导操作者继续在飞书后台完成事件订阅配置

#### Scenario: 连接未建立时不得输出就绪反馈
- **WHEN** 应用启动失败或长连接尚未建立
- **THEN** 系统不得输出误导性的连接就绪信息

### Requirement: 接收真实事件回调

系统 SHALL 通过真实飞书长连接接收 `im.message.receive_v1` 事件，并调用已注册的事件处理器。

#### Scenario: 收到真实消息事件时触发回调
- **WHEN** 飞书开放平台通过长连接下发 `im.message.receive_v1` 事件
- **THEN** 已注册的事件处理器被调用
- **THEN** 处理器接收到解析后的事件结构

#### Scenario: 事件处理失败不应破坏连接
- **WHEN** 事件处理器返回错误
- **THEN** 错误被记录或向上返回到既有处理链路
- **THEN** 长连接保持可继续接收后续事件

### Requirement: 发送真实文本消息

系统 SHALL 通过真实飞书消息 API 向目标会话发送文本回复。

#### Scenario: 连接正常时发送文本消息
- **WHEN** 调用 `client.SendText(ctx, chatID, "Hello")` 且长连接已就绪
- **THEN** 文本消息被发送到飞书目标会话
- **THEN** 方法返回 nil

#### Scenario: 未连接时发送文本消息
- **WHEN** 调用 `client.SendText(ctx, chatID, "Hello")` 且长连接未就绪
- **THEN** 方法返回连接未就绪错误

### Requirement: 断线恢复与资源回收

系统 SHALL 在连接异常中断后执行恢复，并在主动停止时释放底层资源。

#### Scenario: 长连接异常断开后恢复
- **WHEN** 已建立的飞书长连接因网络或服务端原因意外中断
- **THEN** 系统尝试恢复连接
- **THEN** 恢复后能够继续接收后续事件

#### Scenario: 主动停止时释放资源
- **WHEN** 调用 `client.Disconnect()` 或应用进入关闭流程
- **THEN** 底层长连接被关闭
- **THEN** `client.IsConnected()` 返回 `false`
