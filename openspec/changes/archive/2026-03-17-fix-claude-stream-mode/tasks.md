## 1. 事件类型与解析器更新

**目标**: 更新事件类型定义和解析器以支持新的交互式流模式

- [x] 1.1 在 `events.go` 中添加新事件类型常量
  - 添加 `EventTypeControlRequest`, `EventTypeControlCancel`
  - 添加 `EventPermissionRequest` 等事件类型
  - 验证: `go test ./internal/agent/claudecode/... -run TestEventTypes`

- [x] 1.2 更新 `StreamEvent` 结构体添加权限请求字段
  - 添加 `RequestID`, `Questions` 等字段
  - 添加 `GetRequestID()`, `HasPermissionRequest()` 等方法
  - 验证: `go test ./internal/agent/claudecode/... -run TestStreamEvent`

- [x] 1.3 更新 `parser.go` 添加 control_request 解析
  - 解析 `control_request` 事件
  - 解析 `control_cancel_request` 事件
  - 验证: `go test ./internal/agent/claudecode/... -run TestParseControlRequest`

**依赖**: 无
**影响文件**: `internal/agent/claudecode/events.go`, `internal/agent/claudecode/parser.go`

## 2. Session 会话实现

**目标**: 实现持久会话管理，支持双向通信

- [x] 2.1 创建 `session.go` 文件，定义 Session 结构体
  - 定义 `Session` 结构体（cmd, stdin, stdout, events, autoApprove 等）
  - 定义 `Event` 结构体（Type, Content, ToolName, RequestID 等）
  - 定义 `ImageAttachment`, `FileAttachment` 类型
  - 验证: 文件创建成功，编译通过

- [x] 2.2 实现 `newSession()` 函数启动持久进程
  - 构建 CLI 参数：`--input-format stream-json --permission-prompt-tool stdio`
  - 创建 stdin/stdout 管道
  - 启动进程并初始化 Session
  - 验证: `go test ./internal/agent/claudecode/... -run TestNewSession`

- [x] 2.3 实现 `readLoop()` 持续读取 stdout 事件流
  - 使用 bufio.Scanner 逐行读取
  - 根据事件类型调用对应处理函数
  - 处理进程退出和错误
  - 验证: `go test ./internal/agent/claudecode/... -run TestReadLoop`

- [x] 2.4 实现事件处理函数
  - `handleSystem()` - 处理 system 事件，提取 session_id
  - `handleAssistant()` - 处理 assistant 事件，提取 text/tool_use
  - `handleResult()` - 处理 result 事件，标记完成
  - 验证: `go test ./internal/agent/claudecode/... -run TestHandle`

- [x] 2.5 实现 `Send()` 方法发送用户消息
  - 实现纯文本消息发送
  - 实现多模态消息（图片）发送
  - 使用互斥锁保护 stdin 写入
  - 验证: `go test ./internal/agent/claudecode/... -run TestSessionSend`

- [x] 2.6 实现 `handleControlRequest()` 处理权限请求
  - 解析 control_request 事件
  - YOLO 模式自动批准
  - 非 YOLO 模式自动拒绝
  - 验证: `go test ./internal/agent/claudecode/... -run TestControlRequest`

- [x] 2.7 实现 `RespondPermission()` 发送权限响应
  - 构造 `control_response` JSON
  - 写入 stdin
  - 验证: `go test ./internal/agent/claudecode/... -run TestRespondPermission`

- [x] 2.8 实现 `Close()` 优雅关闭会话
  - 取消 context
  - 等待进程退出（最多 8 秒）
  - 超时后强制终止
  - 关闭 events 通道
  - 验证: `go test ./internal/agent/claudecode/... -run TestSessionClose`

- [x] 2.9 实现 `Alive()` 和 `CurrentSessionID()` 状态查询方法
  - 验证: `go test ./internal/agent/claudecode/... -run TestSessionState`

**依赖**: 任务 1 完成
**影响文件**: `internal/agent/claudecode/session.go`（新建）

## 3. Agent 接口适配

**目标**: 修改 ClaudeCodeAgent 使用新的 Session 实现

- [x] 3.1 更新 `ClaudeCodeAgent` 结构体
  - 将 `pm *ProcessManager` 替换为 `session *Session`
  - 添加 `sessionMu` 保护 session 访问
  - 验证: 编译通过

- [x] 3.2 更新 `Start()` 方法
  - 创建 Session 而非 ProcessManager
  - 验证: `go test ./internal/agent/claudecode/... -run TestAgentStart`

- [x] 3.3 重写 `SendMessage()` 方法
  - 调用 `session.Send()` 发送消息
  - 从 `session.Events()` 读取事件并收集响应
  - 处理 result 事件返回最终响应
  - 验证: `go test ./internal/agent/claudecode/... -run TestSendMessage`

- [x] 3.4 更新 `Stop()` 方法
  - 调用 `session.Close()`
  - 验证: `go test ./internal/agent/claudecode/... -run TestAgentStop`

- [x] 3.5 更新 `SetPermissionMode()` 方法
  - 关闭当前 session
  - 以新模式重新创建 session
  - 验证: `go test ./internal/agent/claudecode/... -run TestSetPermissionMode`

**依赖**: 任务 2 完成
**影响文件**: `internal/agent/claudecode/agent.go`

## 4. 测试与 Mock 更新

**目标**: 更新测试以适配新实现

- [x] 4.1 创建 `session_test.go` 单元测试
  - 测试 Session 启动、发送、接收、关闭
  - 使用 mock 进程脚本模拟 Claude CLI
  - 验证: `go test ./internal/agent/claudecode/... -run TestSession -v`

- [x] 4.2 更新 `mock_agent.go`
  - 更新 MockAgent 实现，模拟事件流
  - 添加 Events 通道支持
  - 验证: `go test ./internal/agent/claudecode/... -run TestMockAgent`

- [x] 4.3 更新 `agent_test.go` 测试
  - 适配新的 Agent 实现
  - 验证: `go test ./internal/agent/claudecode/... -run TestAgent -v`

- [ ] 4.4 创建集成测试（使用真实 Claude CLI）
  - 测试完整交互流程
  - 使用 build tag `integration`
  - 验证: `go test ./internal/agent/claudecode/... -tags=integration -run TestIntegration`

**依赖**: 任务 3 完成
**影响文件**: `internal/agent/claudecode/session_test.go`, `internal/agent/claudecode/mock_agent.go`, `internal/agent/claudecode/agent_test.go`

## 5. 清理与文档

**目标**: 清理旧代码，更新文档

- [x] 5.1 评估并删除或重构 `process.go`
  - 如果 ProcessManager 不再使用，删除
  - 如果有可复用代码，重构保留
  - 验证: `go build ./...`

- [x] 5.2 更新 `doc.go` 包文档
  - 说明新的交互式流模式
  - 更新使用示例
  - 验证: `go doc ./internal/agent/claudecode`

- [x] 5.3 运行完整测试套件确保覆盖率
  - 验证: `go test ./internal/agent/claudecode/... -cover`
  - 目标: 覆盖率 > 80%

**依赖**: 任务 4 完成
**影响文件**: `internal/agent/claudecode/process.go`, `internal/agent/claudecode/doc.go`

## 6. 端到端验证

**目标**: 验证完整流程可用

- [ ] 6.1 运行应用并测试 Feishu → Claude Code 通信
  - 启动应用：`go run ./cmd/cc-connect -config config.toml`
  - 从 Feishu 发送消息
  - 验证收到 Claude 响应

- [ ] 6.2 测试 YOLO 模式
  - 设置 `permission_mode = "yolo"`
  - 发送需要工具调用的消息
  - 验证工具调用被自动批准

**依赖**: 任务 5 完成
**影响**: 整体系统
