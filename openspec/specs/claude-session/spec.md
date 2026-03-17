## ADDED Requirements

### Requirement: Session 进程启动

Session 必须使用交互式流模式启动 Claude Code 进程，支持持久双向通信。

- 进程必须使用 `--input-format stream-json` 参数
- 进程必须使用 `--permission-prompt-tool stdio` 参数
- 进程必须使用 `--output-format stream-json` 参数
- 进程启动后必须保持运行，直到显式关闭

#### Scenario: 启动新会话

- **WHEN** 调用 `newSession(ctx, workDir, model, sessionID, mode, tools, env)`
- **THEN** 系统启动 Claude 进程，参数包含 `--input-format stream-json --permission-prompt-tool stdio`
- **THEN** 返回 Session 实例，events 通道可用

#### Scenario: 恢复已有会话

- **WHEN** 调用 `newSession` 时 sessionID 非空
- **THEN** 系统使用 `--resume <sessionID>` 参数启动进程
- **THEN** 新会话继承之前会话的上下文

### Requirement: 发送用户消息

Session 必须支持通过 stdin 发送 JSON 格式的用户消息。

- 消息必须使用 `{"type":"user","message":{"role":"user","content":"..."}}` 格式
- 必须支持纯文本消息
- 必须支持包含图片的多模态消息（base64 编码）
- 必须并发安全

#### Scenario: 发送纯文本消息

- **WHEN** 调用 `session.Send("你好", nil, nil)`
- **THEN** 系统向 stdin 写入 JSON：`{"type":"user","message":{"role":"user","content":"你好"}}`
- **THEN** 返回 nil 表示成功

#### Scenario: 发送带图片的消息

- **WHEN** 调用 `session.Send("分析这张图", images, nil)`，images 包含一张 PNG 图片
- **THEN** 系统向 stdin 写入 JSON，content 为数组，包含 image 和 text 两个 part
- **THEN** 图片使用 base64 编码

#### Scenario: 并发发送消息

- **WHEN** 两个 goroutine 同时调用 `session.Send()`
- **THEN** 系统使用互斥锁保护 stdin 写入
- **THEN** 两条消息按顺序写入，不会交错

### Requirement: 事件流解析

Session 必须从 stdout 持续读取并解析事件流。

- 必须支持 `system` 事件类型，提取 session_id
- 必须支持 `assistant` 事件类型，提取文本内容和工具调用
- 必须支持 `result` 事件类型，提取最终结果
- 必须支持 `control_request` 事件类型，处理权限请求
- 必须支持 `control_cancel_request` 事件类型

#### Scenario: 解析 system 事件

- **WHEN** stdout 输出 `{"type":"system","session_id":"abc123"}`
- **THEN** 系统更新内部 sessionID 为 "abc123"
- **THEN** 向 events 通道发送 `Event{Type: EventSystem, SessionID: "abc123"}`

#### Scenario: 解析 assistant 文本事件

- **WHEN** stdout 输出 `{"type":"assistant","message":{"content":[{"type":"text","text":"你好"}]}}`
- **THEN** 系统向 events 通道发送 `Event{Type: EventText, Content: "你好"}`

#### Scenario: 解析 assistant 工具调用事件

- **WHEN** stdout 输出 `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"file_path":"test.go"}}]}}`
- **THEN** 系统向 events 通道发送 `Event{Type: EventToolUse, ToolName: "Read", ToolInput: "test.go"}`

#### Scenario: 解析 result 事件

- **WHEN** stdout 输出 `{"type":"result","result":"完成","session_id":"abc123"}`
- **THEN** 系统向 events 通道发送 `Event{Type: EventResult, Content: "完成", SessionID: "abc123", Done: true}`

#### Scenario: 解析 control_request 事件

- **WHEN** stdout 输出 `{"type":"control_request","request_id":"req1","request":{"subtype":"can_use_tool","tool_name":"Bash","input":{"command":"ls"}}}`
- **THEN** 系统根据 autoApprove 设置处理权限请求
- **THEN** 向 events 通道发送 `Event{Type: EventPermissionRequest, RequestID: "req1", ToolName: "Bash"}`

### Requirement: 权限请求处理

Session 必须支持响应 Claude Code 的权限请求。

- YOLO 模式（bypassPermissions）必须自动批准所有权限请求
- 非 YOLO 模式必须自动拒绝权限请求（暂不实现用户交互）
- 响应必须通过 stdin 发送 `control_response` 格式

#### Scenario: YOLO 模式自动批准

- **GIVEN** session.autoApprove = true
- **WHEN** 收到 `control_request` 事件
- **THEN** 系统立即发送 `control_response`，behavior 为 "allow"

#### Scenario: 非 YOLO 模式自动拒绝

- **GIVEN** session.autoApprove = false
- **WHEN** 收到 `control_request` 事件
- **THEN** 系统立即发送 `control_response`，behavior 为 "deny"
- **THEN** 响应包含友好的拒绝消息

### Requirement: 会话生命周期管理

Session 必须支持优雅启动和关闭。

- 启动时必须验证 Claude CLI 可用
- 关闭时必须先取消 context，等待进程退出
- 关闭超时后必须强制终止进程
- 关闭时必须关闭 stdin 和 events 通道

#### Scenario: 正常关闭会话

- **WHEN** 调用 `session.Close()`
- **THEN** 系统取消 context
- **THEN** 等待进程退出（最多 8 秒）
- **THEN** 关闭 events 通道

#### Scenario: 强制终止超时进程

- **WHEN** 调用 `session.Close()` 且进程 8 秒内未退出
- **THEN** 系统发送 SIGKILL 强制终止进程
- **THEN** 关闭 events 通道

#### Scenario: 检查会话存活状态

- **WHEN** 调用 `session.Alive()`
- **THEN** 返回 true 如果进程正在运行
- **THEN** 返回 false 如果进程已退出

### Requirement: 错误处理

Session 必须正确处理各种错误情况。

- 进程启动失败必须返回错误
- stdin 写入失败必须返回错误
- stdout 读取错误必须发送到 events 通道
- 进程异常退出必须发送错误事件

#### Scenario: 进程启动失败

- **WHEN** Claude CLI 不在 PATH 中
- **THEN** `newSession()` 返回错误，包含 "claude CLI not found"

#### Scenario: 进程异常退出

- **WHEN** Claude 进程非正常退出（如 OOM）
- **THEN** 系统向 events 通道发送 `Event{Type: EventError, Error: ...}`
- **THEN** 关闭 events 通道

#### Scenario: 向已关闭会话发送消息

- **WHEN** 调用 `session.Send()` 时进程已退出
- **THEN** 返回错误 "session process is not running"
