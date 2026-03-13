# Claude Code 适配器规格

## Purpose

Claude Code 适配器负责管理与 Claude Code CLI 的通信，提供子进程生命周期管理、流式输出解析、权限模式控制和会话集成能力。它是 cc-connect 的核心 AI 能力提供者。

## ADDED Requirements

### Requirement: 子进程生命周期管理

系统 SHALL 能够启动、停止和监控 Claude Code CLI 子进程。

#### Scenario: 启动子进程成功
- **WHEN** 调用 `agent.Start(ctx)` 且 Claude Code CLI 已安装
- **THEN** 系统启动 `claude -p --output-format stream-json` 子进程
- **AND** 进程状态变为 `Ready`

#### Scenario: 启动失败处理
- **WHEN** 调用 `agent.Start(ctx)` 且 Claude Code CLI 未安装或不可执行
- **THEN** 系统返回 `ErrClaudeNotFound` 错误
- **AND** 进程状态保持 `Idle`

#### Scenario: 停止子进程
- **WHEN** 调用 `agent.Stop()` 且进程正在运行
- **THEN** 系统发送 SIGTERM 信号
- **AND** 等待最多 2 秒让进程优雅退出
- **AND** 若进程未退出则发送 SIGKILL
- **AND** 进程状态变为 `Idle`

#### Scenario: 进程崩溃自动恢复
- **WHEN** 子进程意外退出（非正常 Stop 调用）
- **THEN** 系统记录错误日志
- **AND** 下次 `Process()` 调用时自动重启进程

### Requirement: 流式输出解析

系统 SHALL 能够解析 Claude Code CLI 的 `stream-json` 格式输出。

#### Scenario: 解析文本事件
- **WHEN** CLI 输出 `{"type":"text","text":"Hello"}`
- **THEN** 系统生成 `AgentEvent{Type: EventText, Text: "Hello"}`

#### Scenario: 解析工具调用事件
- **WHEN** CLI 输出 `{"type":"tool_use","name":"Read","input":{...}}`
- **THEN** 系统生成 `AgentEvent{Type: EventToolUse, ToolName: "Read"}`

#### Scenario: 解析成功结果事件
- **WHEN** CLI 输出 `{"type":"result","subtype":"success","result":"Done","session_id":"xxx"}`
- **THEN** 系统生成 `AgentEvent{Type: EventResult, Result: "Done"}`
- **AND** 系统记录 `session_id` 用于后续会话恢复

#### Scenario: 解析错误结果事件
- **WHEN** CLI 输出 `{"type":"result","subtype":"error","error":"..."}`
- **THEN** 系统生成 `AgentEvent{Type: EventError, Error: ...}`

#### Scenario: 处理不完整 JSON
- **WHEN** 输出缓冲区包含不完整的 JSON 行
- **THEN** 系统缓冲数据等待后续输入
- **AND** 当收到完整 JSON 行时才进行解析

#### Scenario: CLI 挂起超时处理
- **WHEN** 收到 `result` 事件后 5 秒内进程未退出
- **THEN** 系统发送 SIGTERM 强制终止进程
- **AND** 记录警告日志

### Requirement: 权限模式管理

系统 SHALL 支持四种权限模式及其别名映射。

#### Scenario: 设置权限模式
- **WHEN** 调用 `agent.SetPermissionMode("yolo")`
- **THEN** `agent.CurrentMode()` 返回 `"bypassPermissions"`
- **AND** 后续 `Process()` 调用使用 `--permission-mode bypassPermissions`

#### Scenario: 权限模式别名映射
- **WHEN** 用户使用以下任一别名设置模式
  - `"edit"` 或 `"acceptEdits"` → 内部使用 `"acceptEdits"`
  - `"yolo"` 或 `"bypassPermissions"` → 内部使用 `"bypassPermissions"`
- **THEN** `CurrentMode()` 返回规范名称
- **AND** CLI 参数使用规范名称

#### Scenario: 无效权限模式
- **WHEN** 调用 `SetPermissionMode("invalid")`
- **THEN** 系统返回 `ErrInvalidPermissionMode` 错误
- **AND** 当前模式保持不变

#### Scenario: 默认权限模式
- **WHEN** Agent 初始化后未调用 `SetPermissionMode`
- **THEN** `CurrentMode()` 返回 `"default"`

### Requirement: 会话集成

系统 SHALL 支持与 Claude Code CLI 的会话机制集成。

#### Scenario: 首次消息生成会话
- **WHEN** 调用 `Process(ctx, "hello")` 且无活跃会话
- **THEN** CLI 使用 `--session-id <uuid>` 参数启动
- **AND** 从 `result` 事件中提取 `session_id` 用于后续恢复

#### Scenario: 恢复已有会话
- **WHEN** 调用 `Process(ctx, "continue")` 且存在活跃会话
- **THEN** CLI 使用 `--resume <session-id>` 参数启动
- **AND** CLI 恢复之前的对话上下文

#### Scenario: 会话 ID 存储与获取
- **WHEN** CLI 返回 `session_id` 字段
- **THEN** 系统可通过 `agent.SessionID()` 获取该 ID
- **AND** 该 ID 可用于外部会话管理器关联

### Requirement: 消息处理接口

系统 SHALL 提供与 `core.Agent` 接口兼容的消息处理能力。

#### Scenario: 处理文本消息
- **WHEN** 调用 `Process(ctx, "Fix the bug in main.go")`
- **THEN** 系统返回事件 channel
- **AND** channel 依次发送 `EventText`、可能的 `EventToolUse`、最终 `EventResult`

#### Scenario: 处理空消息
- **WHEN** 调用 `Process(ctx, "")`
- **THEN** 系统返回 `ErrEmptyInput` 错误

#### Scenario: 上下文取消
- **WHEN** 调用 `Process(ctx, "long task")` 且 `ctx` 被取消
- **THEN** 系统终止当前进程
- **AND** channel 发送 `EventError` 并关闭

#### Scenario: 并发处理保护
- **WHEN** 同时调用两次 `Process()`
- **THEN** 第二次调用返回 `ErrAgentBusy` 错误
- **AND** 第一次调用继续正常执行

### Requirement: Mock 实现支持

系统 SHALL 提供 Mock Agent 用于测试场景。

#### Scenario: Mock 处理消息
- **WHEN** 使用 `NewMockAgent()` 创建 Agent
- **AND** 调用 `mock.Process(ctx, "test")`
- **THEN** 返回预设的响应事件
- **AND** 不启动真实子进程

#### Scenario: Mock 模式设置
- **WHEN** 调用 `mock.SetPermissionMode("yolo")`
- **THEN** `mock.CurrentMode()` 返回 `"bypassPermissions"`
- **AND** 记录模式变更用于测试断言

#### Scenario: Mock 错误模拟
- **WHEN** 配置 `mock.SetError(errors.New("simulated"))`
- **THEN** 后续 `Process()` 调用返回错误事件
