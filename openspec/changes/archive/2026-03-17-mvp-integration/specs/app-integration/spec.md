# App Integration 规格说明

本规格定义 Application 整合层的行为，负责组件初始化、生命周期管理、消息处理器注册和错误处理。

## ADDED Requirements

### Requirement: 应用初始化

系统 SHALL 提供 `New` 函数根据配置创建应用实例，初始化所有必要组件。

#### Scenario: 成功创建应用实例
- **WHEN** 使用有效配置调用 `New(config)`
- **THEN** 系统返回初始化完成的 `*App` 实例
- **AND** Router、Agent、Feishu Adapter、Command Executor 均已创建

#### Scenario: 配置无效时返回错误
- **WHEN** 使用无效配置调用 `New(config)`
- **THEN** 系统返回描述性错误信息
- **AND** 不创建任何资源

### Requirement: 应用生命周期管理

系统 SHALL 提供 `Start`、`Stop`、`WaitForShutdown` 方法管理应用生命周期。

#### Scenario: 成功启动应用
- **WHEN** 调用 `app.Start(ctx)`
- **THEN** Feishu Adapter 连接到飞书 WebSocket
- **AND** Agent 启动
- **AND** 消息处理器注册到 Router
- **AND** 应用状态变为 `running`

#### Scenario: 启动失败时清理资源
- **WHEN** Feishu 连接失败或 Agent 启动失败
- **THEN** 系统清理已创建的资源
- **AND** 返回描述性错误信息
- **AND** 应用状态保持 `idle`

#### Scenario: 优雅停止应用
- **WHEN** 调用 `app.Stop()`
- **THEN** 停止接收新消息
- **AND** 等待正在处理的消息完成（最多 30 秒）
- **AND** 停止 Agent
- **AND** 断开 Feishu 连接
- **AND** 应用状态变为 `stopped`

#### Scenario: 信号触发的优雅关闭
- **WHEN** 收到 SIGINT 或 SIGTERM 信号
- **THEN** 系统自动执行优雅关闭流程
- **AND** 退出码为 0

### Requirement: 消息处理器注册

系统 SHALL 自动注册文本消息和命令消息的处理器到 Router。

#### Scenario: 文本消息处理器注册
- **WHEN** 应用启动时
- **THEN** `MessageTypeText` 类型的处理器已注册
- **AND** 处理器能将消息转发给 Agent

#### Scenario: 命令消息处理器注册
- **WHEN** 应用启动时
- **THEN** `MessageTypeCommand` 类型的处理器已注册
- **AND** 处理器能将命令转发给 Executor

### Requirement: HandlerContext 注入

系统 SHALL 为每个处理器提供 `HandlerContext`，包含上下文、消息、会话和回复发送器。

#### Scenario: HandlerContext 正确注入
- **WHEN** 消息被路由到处理器
- **THEN** `HandlerContext.Ctx` 包含请求上下文
- **AND** `HandlerContext.Msg` 包含原始消息
- **AND** `HandlerContext.Session` 包含关联会话
- **AND** `HandlerContext.Reply` 可用于发送回复

### Requirement: ReplySender 接口

系统 SHALL 提供 `ReplySender` 接口，允许处理器向消息来源发送回复。

#### Scenario: 成功发送回复
- **WHEN** 调用 `reply.SendReply(ctx, "响应内容")`
- **THEN** 响应内容发送到消息来源的聊天频道
- **AND** 返回 nil

#### Scenario: 发送失败返回错误
- **WHEN** 调用 `reply.SendReply` 时网络故障
- **THEN** 返回描述性错误信息

### Requirement: 文本消息处理

系统 SHALL 正确处理文本消息，包括状态提示、Agent 调用和响应发送。

#### Scenario: 成功处理文本消息
- **WHEN** 收到文本消息 "帮我修复 bug"
- **THEN** 系统发送 "🤔 正在思考..." 状态提示
- **AND** 调用 Agent 处理消息
- **AND** 发送 Agent 响应内容

#### Scenario: Agent 处理超时
- **WHEN** Agent 处理时间超过配置的超时时间
- **THEN** 系统取消 Agent 处理
- **AND** 发送 "⏱️ 请求超时，请简化问题或稍后重试。" 错误提示

#### Scenario: Agent 处理错误
- **WHEN** Agent 返回错误
- **THEN** 系统发送 "❌ 处理失败: {错误信息}" 错误提示

### Requirement: 命令消息处理

系统 SHALL 正确处理斜杠命令，执行并返回结果。

#### Scenario: 成功执行命令
- **WHEN** 收到命令 "/mode yolo"
- **THEN** 系统调用 Executor 执行命令
- **AND** 发送命令执行结果

#### Scenario: 未知命令
- **WHEN** 收到未知命令 "/unknown"
- **THEN** 系统发送 "未知命令: /unknown\n输入 /help 查看可用命令"

### Requirement: 错误恢复

系统 SHALL 捕获处理器 panic 并发送友好错误信息。

#### Scenario: 处理器 panic 恢复
- **WHEN** 处理器执行过程中发生 panic
- **THEN** 系统捕获 panic
- **AND** 记录错误日志
- **AND** 发送 "❌ 内部错误，请稍后重试。" 给用户
- **AND** 应用继续运行

### Requirement: 配置验证

系统 SHALL 验证配置的完整性和有效性。

#### Scenario: 缺少必要配置
- **WHEN** 配置缺少 Feishu 或 Agent 必要字段
- **THEN** 系统返回描述性错误
- **AND** 列出缺失的字段

#### Scenario: 无效配置值
- **WHEN** 配置值无效（如负数超时）
- **THEN** 系统返回描述性错误
- **AND** 说明期望的值范围
