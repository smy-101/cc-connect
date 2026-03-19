## MODIFIED Requirements

### Requirement: 事件流解析

Session 必须从 stdout 持续读取并解析事件流。

- 必须支持 `system` 事件类型，提取 session_id
- 必须支持 `assistant` 事件类型，提取文本内容和工具调用
- 必须支持 `result` 事件类型，提取最终结果
- 必须支持 `control_request` 事件类型，处理权限请求
- 必须支持 `control_cancel_request` 事件类型
- **MODIFIED**: `control_request` 必须创建 pendingPermission 状态并阻塞处理
- **ADDED**: 必须支持 `control_request` 中的 `AskUserQuestion` 工具，解析 questions 字段

#### Scenario: 解析 control_request 事件

- **WHEN** stdout 输出 `{"type":"control_request","request_id":"req1","request":{"subtype":"can_use_tool","tool_name":"Bash","input":{"command":"ls"}}}`
- **THEN** 系统创建 `pendingPermission{requestID: "req1", toolName: "Bash"}`
- **THEN** 向 events 通道发送 `Event{Type: EventPermissionRequest, RequestID: "req1", ToolName: "Bash"}`
- **THEN** 阻塞等待用户响应（除非 autoApprove = true）

#### Scenario: 解析 AskUserQuestion control_request

- **WHEN** stdout 输出 `{"type":"control_request","request_id":"req2","request":{"subtype":"can_use_tool","tool_name":"AskUserQuestion","input":{"questions":[{"question":"选择数据库","options":[{"label":"PostgreSQL"}]}]}}}`
- **THEN** 系统创建 `pendingPermission{requestID: "req2", toolName: "AskUserQuestion", Questions: [...]}`
- **THEN** 向 events 通道发送 `Event{Type: EventPermissionRequest, RequestID: "req2", ToolName: "AskUserQuestion", Questions: [...]}`

### Requirement: 权限请求处理

Session 必须支持响应 Claude Code 的权限请求。

- YOLO 模式（bypassPermissions）必须自动批准所有权限请求
- **MODIFIED**: 非 YOLO 模式必须阻塞等待 `RespondPermission` 调用
- 响应必须通过 stdin 发送 `control_response` 格式
- **ADDED**: 必须支持通过 `RespondPermission` 方法响应

#### Scenario: YOLO 模式自动批准

- **GIVEN** session.autoApprove = true
- **WHEN** 收到 `control_request` 事件
- **THEN** 系统立即发送 `control_response`，behavior 为 "allow"
- **THEN** 不阻塞处理流程

#### Scenario: 非 YOLO 模式阻塞等待

- **GIVEN** session.autoApprove = false
- **WHEN** 收到 `control_request` 事件
- **THEN** 系统创建 `pendingPermission` 并阻塞
- **THEN** 等待 `RespondPermission` 调用
- **THEN** 收到响应后发送 `control_response`

#### Scenario: 超时自动拒绝

- **GIVEN** session.autoApprove = false 且正在等待权限响应
- **WHEN** 等待超过 5 分钟
- **THEN** 系统自动发送 `control_response`，behavior 为 "deny"
- **THEN** message 为 "等待超时，自动拒绝"

### Requirement: RespondPermission 方法

**ADDED**: Session 必须提供 `RespondPermission` 方法。

- 必须验证 requestID 匹配当前 pending 请求
- 必须支持 "allow" 和 "deny" 两种 behavior
- allow 时必须包含 `updatedInput` 字段
- deny 时可选包含 `message` 字段
- 必须关闭 pending.resolved channel 唤醒阻塞

#### Scenario: 批准权限请求

- **GIVEN** 存在 pending 权限请求 "req1"
- **WHEN** 调用 `session.RespondPermission("req1", {Behavior: "allow", UpdatedInput: {...}})`
- **THEN** 发送 `{"type":"control_response","response":{"subtype":"success","request_id":"req1","response":{"behavior":"allow","updatedInput":{...}}}}`
- **THEN** 关闭 pending.resolved channel

#### Scenario: 拒绝权限请求

- **GIVEN** 存在 pending 权限请求 "req1"
- **WHEN** 调用 `session.RespondPermission("req1", {Behavior: "deny", Message: "用户拒绝"})`
- **THEN** 发送 `{"type":"control_response","response":{"subtype":"success","request_id":"req1","response":{"behavior":"deny","message":"用户拒绝"}}}`
- **THEN** 关闭 pending.resolved channel

#### Scenario: AskUserQuestion 答案响应

- **GIVEN** 存在 pending AskUserQuestion 请求 "req2"
- **WHEN** 调用 `session.RespondPermission("req2", {Behavior: "allow", UpdatedInput: {"answers": ["PostgreSQL"]}})`
- **THEN** 发送包含 answers 的 control_response
- **THEN** Claude 收到用户选择的答案

## ADDED Requirements

### Requirement: Event 结构扩展

Event 结构必须支持权限请求相关字段。

- 必须包含 `RequestID` 字段（权限请求唯一标识）
- 必须包含 `ToolName` 字段（工具名称）
- 必须包含 `ToolInput` 字段（工具输入预览）
- 必须包含 `ToolInputRaw` 字段（原始工具输入，用于 allow 响应）
- 必须包含 `Questions` 字段（AskUserQuestion 的问题列表）

#### Scenario: 权限请求事件字段

- **WHEN** 发送权限请求事件
- **THEN** `Event{Type: EventPermissionRequest, RequestID: "req1", ToolName: "Bash", ToolInput: "ls -la", ToolInputRaw: {"command": "ls -la"}}`

#### Scenario: AskUserQuestion 事件字段

- **WHEN** 发送 AskUserQuestion 事件
- **THEN** `Event{Type: EventPermissionRequest, ToolName: "AskUserQuestion", Questions: [{Question: "选择数据库", Options: [...]}]}`

### Requirement: pendingPermission 结构

**ADDED**: Session 必须内部维护 `pendingPermission` 状态。

```go
type pendingPermission struct {
    requestID   string
    toolName    string
    toolInput   string
    toolInputRaw map[string]any
    questions   []UserQuestion
    resolved    chan struct{}
    result      *PermissionResult
    resultOnce  sync.Once
}
```

- 必须使用 mutex 保护 result 字段
- 必须使用 sync.Once 确保 channel 只关闭一次
- 必须支持超时检测

#### Scenario: pending 状态生命周期

- **WHEN** 收到 control_request
- **THEN** 创建 pendingPermission，resolved channel 打开
- **WHEN** 用户响应
- **THEN** 存储 result，关闭 resolved channel
- **THEN** 阻塞的处理继续执行

### Requirement: PermissionResult 结构

**ADDED**: 定义权限响应结果结构。

```go
type PermissionResult struct {
    Behavior     string         `json:"behavior"`     // "allow" or "deny"
    UpdatedInput map[string]any `json:"updatedInput"` // for allow
    Message      string         `json:"message"`      // for deny
}
```

#### Scenario: allow 结果

- **WHEN** 用户批准请求
- **THEN** `PermissionResult{Behavior: "allow", UpdatedInput: {"command": "ls"}}`

#### Scenario: deny 结果

- **WHEN** 用户拒绝请求
- **THEN** `PermissionResult{Behavior: "deny", Message: "用户拒绝此操作"}`
