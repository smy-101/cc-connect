# Claude Code 适配器规格

## Purpose

Claude Code 适配器负责管理与 Claude Code CLI 的通信，提供持久进程管理、流式输出解析、权限模式控制和会话集成能力。它是 cc-connect 的核心 AI 能力提供者。

## Requirements

### Requirement: 持久进程生命周期管理

系统 SHALL 为每个 Session 维护一个长期运行的 Claude Code CLI 子进程。

#### Scenario: 启动子进程成功
- **WHEN** 调用 `agent.Start(ctx)` 且 Claude Code CLI 已安装
- **THEN** 系统启动 `claude -p --output-format stream-json --session-id <uuid> --permission-mode <mode>` 子进程
- **AND** 进程状态变为 `running`

#### Scenario: 启动失败处理
- **WHEN** 调用 `agent.Start(ctx)` 且 Claude Code CLI 未安装或不可执行
- **THEN** 系统返回 `ErrClaudeNotFound` 错误
- **AND** 进程状态保持 `idle`

#### Scenario: 优雅停止子进程
- **WHEN** 调用 `agent.Stop()` 且进程正在运行
- **THEN** 系统发送 SIGTERM 信号
- **AND** 等待最多 2 秒让进程优雅退出
- **AND** 若进程未退出则发送 SIGKILL
- **AND** 进程状态变为 `stopped`

#### Scenario: 进程崩溃自动恢复
- **WHEN** 子进程意外退出（非正常 Stop 调用）
- **THEN** 系统记录错误日志
- **AND** 下次 `SendMessage()` 调用时使用 `--resume <session-id>` 自动重启进程
- **AND** Claude Code 自动恢复对话上下文

#### Scenario: 模式切换重启进程
- **WHEN** 调用 `agent.SetPermissionMode("yolo")` 且进程正在运行
- **THEN** 系统停止当前进程
- **AND** 使用 `--resume <session-id> --permission-mode bypassPermissions` 重启进程
- **AND** 返回成功，会话上下文保持

### Requirement: 流式输出解析

系统 SHALL 能够解析 Claude Code CLI 的 `stream-json` 格式输出（JSONL，每行一个 JSON 对象）。

#### Scenario: 解析 system/init 事件
- **WHEN** CLI 输出 `{"type":"system","subtype":"init","session_id":"xxx","cwd":"/repo","model":"sonnet","permissionMode":"auto","tools":["Bash","Read"]}`
- **THEN** 系统提取 `session_id`、`tools`、`permissionMode` 等元数据
- **AND** 触发 `StreamEvent{Type: "system", ...}`

#### Scenario: 解析 assistant 文本事件
- **WHEN** CLI 输出 `{"type":"assistant","session_id":"xxx","message":{"content":[{"type":"text","text":"Hello"}]}}`
- **THEN** 系统生成 `StreamEvent{Type: "text", Content: "Hello"}`

#### Scenario: 解析 assistant tool_use 事件
- **WHEN** CLI 输出 `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"toolu_1","name":"Bash","input":{"command":"ls"}}]}}`
- **THEN** 系统生成 `StreamEvent{Type: "tool_use", Tool: &ToolInfo{Name: "Bash", ID: "toolu_1", Input: {...}}}`

#### Scenario: 解析 result/success 事件
- **WHEN** CLI 输出 `{"type":"result","subtype":"success","session_id":"xxx","result":"Done.","total_cost_usd":0.0123,"duration_ms":12345}`
- **THEN** 系统生成最终响应
- **AND** `Response{Content: "Done.", CostUSD: 0.0123, Duration: 12345ms}`

#### Scenario: 解析 result/error 事件（权限拒绝）
- **WHEN** CLI 输出 `{"type":"result","subtype":"error","error":"Permission denied","permission_denials":[{"tool_name":"Bash","tool_use_id":"toolu_9","tool_input":{"command":"git fetch"}}]}`
- **THEN** 系统生成 `Response{IsError: true, PermissionDenied: true, DeniedTools: [...]}`
- **AND** 调用方可根据 `DeniedTools` 决定是否重试

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
- **THEN** 进程重启，使用 `--permission-mode bypassPermissions`
- **AND** `agent.CurrentMode()` 返回 `bypassPermissions`

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

#### Scenario: 首次启动生成会话 ID
- **WHEN** 调用 `agent.Start(ctx)` 且未提供 SessionID
- **THEN** 系统生成 UUID 作为 session-id
- **AND** CLI 使用 `--session-id <uuid>` 参数启动
- **AND** `agent.SessionID()` 返回该 UUID

#### Scenario: 使用已有会话 ID
- **WHEN** 创建 Agent 时提供 Config.SessionID
- **THEN** 系统使用该 SessionID
- **AND** 如果是恢复场景，使用 `--resume` 参数

#### Scenario: 恢复已有会话
- **WHEN** 进程崩溃后重启
- **THEN** CLI 使用 `--resume <session-id>` 参数启动
- **AND** Claude Code 自动恢复对话历史

#### Scenario: 会话 ID 存储与获取
- **WHEN** 需要将 Agent 与 core.Session 关联
- **THEN** 可通过 `agent.SessionID()` 获取会话 ID
- **AND** 该 ID 存储在 `core.Session.AgentID` 中

### Requirement: 消息发送接口

系统 SHALL 提供消息发送能力，支持流式和阻塞两种模式。

#### Scenario: 发送消息并接收流式事件
- **WHEN** 调用 `agent.SendMessage(ctx, "Fix the bug", handler)` 且 handler 非空
- **THEN** 系统写入消息到进程 stdin
- **AND** 对每个解析的事件调用 `handler(event)`
- **AND** 收到 `result` 事件后返回最终响应

#### Scenario: 发送消息并等待完整响应
- **WHEN** 调用 `agent.SendMessage(ctx, "Hello", nil)` 且 handler 为 nil
- **THEN** 系统收集所有事件
- **AND** 返回聚合后的 `Response`

#### Scenario: 处理空消息
- **WHEN** 调用 `SendMessage(ctx, "", nil)`
- **THEN** 系统返回 `ErrEmptyInput` 错误

#### Scenario: 上下文取消
- **WHEN** 调用 `SendMessage(ctx, "long task", handler)` 且 `ctx` 被取消
- **THEN** 系统终止当前请求（但不停止进程）
- **AND** 返回 `ctx.Err()`

#### Scenario: 并发处理保护
- **WHEN** 同时调用两次 `SendMessage()`
- **THEN** 第二次调用返回 `ErrAgentBusy` 错误
- **AND** 第一次调用继续正常执行

### Requirement: 权限请求处理

系统 SHALL 支持处理权限拒绝后的用户批准流程。

#### Scenario: 检测权限拒绝
- **WHEN** `SendMessage()` 返回的 `Response.PermissionDenied` 为 true
- **THEN** 调用方可从 `Response.DeniedTools` 获取被拒绝的工具列表
- **AND** 向用户展示批准请求

#### Scenario: 批准后重试
- **WHEN** 用户批准工具调用
- **AND** 调用 `agent.SendMessage(ctx, content, handler)` 并在配置中添加 `--allowedTools`
- **THEN** 请求重新执行
- **AND** 被批准的工具自动通过

#### Scenario: 会话级工具批准累积
- **WHEN** 用户批准某个工具调用
- **THEN** 可将工具添加到 Session.Metadata["approved_tools"]
- **AND** 后续请求自动包含该工具在 `--allowedTools` 中

### Requirement: Mock 实现支持

系统 SHALL 提供 Mock Agent 用于测试场景。

#### Scenario: Mock 发送消息
- **WHEN** 使用 `NewMockAgent()` 创建 Agent
- **AND** 调用 `mock.SendMessage(ctx, "test", nil)`
- **THEN** 返回预设的响应
- **AND** 不启动真实子进程

#### Scenario: Mock 模式设置
- **WHEN** 调用 `mock.SetPermissionMode("yolo")`
- **THEN** `mock.CurrentMode()` 返回 `"bypassPermissions"`
- **AND** 记录模式变更用于测试断言

#### Scenario: Mock 流式事件
- **WHEN** 配置 `mock.SetStreamEvents([]StreamEvent{...})`
- **AND** 调用 `mock.SendMessage(ctx, "test", handler)`
- **THEN** 对每个预设事件调用 handler

#### Scenario: Mock 权限拒绝
- **WHEN** 配置 `mock.SetPermissionDenied([]DeniedTool{...})`
- **THEN** `SendMessage()` 返回 `Response{PermissionDenied: true, DeniedTools: [...]}`

#### Scenario: Mock 错误模拟
- **WHEN** 配置 `mock.SetError(errors.New("simulated"))`
- **THEN** 后续 `SendMessage()` 调用返回错误
