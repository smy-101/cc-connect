# Claude Code 适配器实现任务

## 1. 基础设施准备

- [ ] 1.1 创建 `internal/agent/claudecode/` 包结构
  - **影响**: `internal/agent/claudecode/doc.go`
  - **验证**: `go build ./internal/agent/...` 成功

- [ ] 1.2 定义核心类型和错误
  - **影响**: `internal/agent/claudecode/types.go`, `internal/agent/claudecode/errors.go`
  - **TDD**: 先定义类型，确保可编译
  - **验证**: `go build ./internal/agent/...` 成功

## 2. 事件类型定义 (TDD)

- [ ] 2.1 编写事件类型测试
  - **影响**: `internal/agent/claudecode/events_test.go`
  - **测试场景**: `EventText`, `EventToolUse`, `EventResult`, `EventError` 类型
  - **验证**: `go test -run TestEventType ./internal/agent/...` 失败（红）

- [ ] 2.2 实现事件类型定义
  - **影响**: `internal/agent/claudecode/events.go`
  - **TDD**: 最小实现使测试通过
  - **验证**: `go test -run TestEventType ./internal/agent/...` 通过（绿）

## 3. 权限模式管理 (TDD)

- [ ] 3.1 编写权限模式测试：模式映射
  - **影响**: `internal/agent/claudecode/permission_test.go`
  - **测试场景**: `default`, `edit`→`acceptEdits`, `yolo`→`bypassPermissions`, `plan`
  - **验证**: `go test -run TestPermissionMode ./internal/agent/...` 失败（红）

- [ ] 3.2 实现权限模式映射
  - **影响**: `internal/agent/claudecode/permission.go`
  - **TDD**: 实现 `PermissionMode` 类型和 `parseMode()` 函数
  - **验证**: `go test -run TestPermissionMode ./internal/agent/...` 通过（绿）

- [ ] 3.3 编写权限模式测试：无效模式处理
  - **影响**: `internal/agent/claudecode/permission_test.go`
  - **测试场景**: 无效模式返回 `ErrInvalidPermissionMode`
  - **验证**: `go test -run TestInvalidPermissionMode ./internal/agent/...` 失败（红）

- [ ] 3.4 实现无效模式错误处理
  - **影响**: `internal/agent/claudecode/permission.go`
  - **TDD**: 添加验证逻辑
  - **验证**: `go test -run TestInvalidPermissionMode ./internal/agent/...` 通过（绿）

## 4. 流式输出解析 (TDD)

- [ ] 4.1 编写 JSON 事件解析测试：文本事件
  - **影响**: `internal/agent/claudecode/stream_test.go`
  - **测试场景**: `{"type":"text","text":"Hello"}` → `EventText`
  - **验证**: `go test -run TestParseTextEvent ./internal/agent/...` 失败（红）

- [ ] 4.2 实现文本事件解析
  - **影响**: `internal/agent/claudecode/stream.go`
  - **TDD**: 实现 `parseEvent()` 函数
  - **验证**: `go test -run TestParseTextEvent ./internal/agent/...` 通过（绿）

- [ ] 4.3 编写 JSON 事件解析测试：工具调用事件
  - **影响**: `internal/agent/claudecode/stream_test.go`
  - **测试场景**: `{"type":"tool_use","name":"Read",...}` → `EventToolUse`
  - **验证**: `go test -run TestParseToolUseEvent ./internal/agent/...` 失败（红）

- [ ] 4.4 实现工具调用事件解析
  - **影响**: `internal/agent/claudecode/stream.go`
  - **TDD**: 扩展 `parseEvent()` 处理 tool_use
  - **验证**: `go test -run TestParseToolUseEvent ./internal/agent/...` 通过（绿）

- [ ] 4.5 编写 JSON 事件解析测试：结果事件
  - **影响**: `internal/agent/claudecode/stream_test.go`
  - **测试场景**: `{"type":"result","subtype":"success","result":"Done"}` → `EventResult`
  - **验证**: `go test -run TestParseResultEvent ./internal/agent/...` 失败（红）

- [ ] 4.6 实现结果事件解析
  - **影响**: `internal/agent/claudecode/stream.go`
  - **TDD**: 扩展 `parseEvent()` 处理 result
  - **验证**: `go test -run TestParseResultEvent ./internal/agent/...` 通过（绿）

- [ ] 4.7 编写缓冲处理测试：不完整 JSON
  - **影响**: `internal/agent/claudecode/stream_test.go`
  - **测试场景**: 分片 JSON `{"type":"text"` + `"text":"Hi"}` 应正确合并
  - **验证**: `go test -run TestStreamBuffer ./internal/agent/...` 失败（红）

- [ ] 4.8 实现流缓冲处理器
  - **影响**: `internal/agent/claudecode/stream.go`
  - **TDD**: 实现 `StreamParser` 类型的 `Feed()` 方法
  - **验证**: `go test -run TestStreamBuffer ./internal/agent/...` 通过（绿）

## 5. 子进程管理 (TDD)

- [ ] 5.1 编写进程启动测试
  - **影响**: `internal/agent/claudecode/process_test.go`
  - **测试场景**: 使用 Mock `exec.Cmd` 验证参数构建
  - **验证**: `go test -run TestProcessStart ./internal/agent/...` 失败（红）

- [ ] 5.2 实现进程启动逻辑
  - **影响**: `internal/agent/claudecode/process.go`
  - **TDD**: 实现 `ProcessManager.Start()` 方法
  - **验证**: `go test -run TestProcessStart ./internal/agent/...` 通过（绿）

- [ ] 5.3 编写进程停止测试
  - **影响**: `internal/agent/claudecode/process_test.go`
  - **测试场景**: SIGTERM → 等待 2s → SIGKILL 流程
  - **验证**: `go test -run TestProcessStop ./internal/agent/...` 失败（红）

- [ ] 5.4 实现进程停止逻辑
  - **影响**: `internal/agent/claudecode/process.go`
  - **TDD**: 实现 `ProcessManager.Stop()` 方法
  - **验证**: `go test -run TestProcessStop ./internal/agent/...` 通过（绿）

- [ ] 5.5 编写进程超时测试
  - **影响**: `internal/agent/claudecode/process_test.go`
  - **测试场景**: CLI 挂起时 5 秒超时终止
  - **验证**: `go test -run TestProcessTimeout ./internal/agent/...` 失败（红）

- [ ] 5.6 实现进程超时处理
  - **影响**: `internal/agent/claudecode/process.go`
  - **TDD**: 在 `Process()` 方法中添加超时处理
  - **验证**: `go test -run TestProcessTimeout ./internal/agent/...` 通过（绿）

## 6. Agent 主实现 (TDD)

- [ ] 6.1 编写 Agent 接口定义测试
  - **影响**: `internal/core/agent.go` (如不存在)
  - **TDD**: 定义 `core.Agent` 接口
  - **验证**: `go build ./internal/core/...` 成功

- [ ] 6.2 编写 Agent 初始化测试
  - **影响**: `internal/agent/claudecode/agent_test.go`
  - **测试场景**: `NewAgent()` 返回正确初始状态
  - **验证**: `go test -run TestNewAgent ./internal/agent/...` 失败（红）

- [ ] 6.3 实现 Agent 结构和构造函数
  - **影响**: `internal/agent/claudecode/agent.go`
  - **TDD**: 实现 `Agent` 结构和 `NewAgent()` 函数
  - **验证**: `go test -run TestNewAgent ./internal/agent/...` 通过（绿）

- [ ] 6.4 编写消息处理测试
  - **影响**: `internal/agent/claudecode/agent_test.go`
  - **测试场景**: `Process()` 返回事件 channel，依次发送事件
  - **验证**: `go test -run TestAgentProcess ./internal/agent/...` 失败（红）

- [ ] 6.5 实现消息处理逻辑
  - **影响**: `internal/agent/claudecode/agent.go`
  - **TDD**: 实现 `Process()` 方法，整合子进程和流解析
  - **验证**: `go test -run TestAgentProcess ./internal/agent/...` 通过（绿）

- [ ] 6.6 编写会话集成测试
  - **影响**: `internal/agent/claudecode/agent_test.go`
  - **测试场景**: 首次调用生成 session_id，后续调用使用 `--resume`
  - **验证**: `go test -run TestAgentSession ./internal/agent/...` 失败（红）

- [ ] 6.7 实现会话管理
  - **影响**: `internal/agent/claudecode/agent.go`
  - **TDD**: 添加 `sessionID` 字段和 `--resume` 参数逻辑
  - **验证**: `go test -run TestAgentSession ./internal/agent/...` 通过（绿）

- [ ] 6.8 编写并发保护测试
  - **影响**: `internal/agent/claudecode/agent_test.go`
  - **测试场景**: 同时调用 `Process()` 第二次返回 `ErrAgentBusy`
  - **验证**: `go test -run TestAgentConcurrency ./internal/agent/...` 失败（红）

- [ ] 6.9 实现并发保护
  - **影响**: `internal/agent/claudecode/agent.go`
  - **TDD**: 添加 `sync.Mutex` 保护 `running` 状态
  - **验证**: `go test -run TestAgentConcurrency ./internal/agent/...` 通过（绿）

## 7. Mock Agent 实现 (TDD)

- [ ] 7.1 编写 Mock Agent 测试
  - **影响**: `internal/agent/claudecode/mock_agent_test.go`
  - **测试场景**: Mock 不启动真实进程，返回预设响应
  - **验证**: `go test -run TestMockAgent ./internal/agent/...` 失败（红）

- [ ] 7.2 实现 Mock Agent
  - **影响**: `internal/agent/claudecode/mock_agent.go`
  - **TDD**: 实现 `MockAgent` 类型，满足 `core.Agent` 接口
  - **验证**: `go test -run TestMockAgent ./internal/agent/...` 通过（绿）

- [ ] 7.3 实现 Mock 响应配置
  - **影响**: `internal/agent/claudecode/mock_agent.go`
  - **功能**: `SetResponse()`, `SetError()` 方法配置 Mock 行为
  - **验证**: `go test -run TestMockAgentResponse ./internal/agent/...` 通过

## 8. 集成与验收

- [ ] 8.1 编写与 Router 集成测试
  - **影响**: `internal/agent/claudecode/integration_test.go`
  - **测试场景**: Router 注册 Agent，消息流转完整
  - **验证**: `go test -run TestAgentRouterIntegration ./internal/agent/...` 通过

- [ ] 8.2 运行完整测试套件
  - **验证**: `go test ./internal/agent/... -v -cover` 覆盖率 ≥ 85%

- [ ] 8.3 运行竞态检测
  - **验证**: `go test ./internal/agent/... -race` 通过

- [ ] 8.4 运行全部测试
  - **验证**: `go test ./...` 通过

---

## 任务依赖关系

```
1.x (基础) ──► 2.x (事件) ──► 3.x (权限) ──► 4.x (流解析)
                                              │
                                              ▼
                                    5.x (子进程) ──► 6.x (Agent) ──► 7.x (Mock) ──► 8.x (集成)
```

**可并行任务**:
- 2.x (事件) 和 3.x (权限) 可并行
- 4.x 子任务之间有依赖，需顺序执行

**关键路径**: 1.x → 2.x/3.x → 4.x → 5.x → 6.x → 7.x → 8.x

## 预计工时

| 阶段 | 任务数 | 预计时间 |
|------|--------|---------|
| 1. 基础设施 | 2 | 0.5 天 |
| 2. 事件类型 | 2 | 0.5 天 |
| 3. 权限模式 | 4 | 1 天 |
| 4. 流式解析 | 8 | 2 天 |
| 5. 子进程管理 | 6 | 1.5 天 |
| 6. Agent 主实现 | 9 | 2 天 |
| 7. Mock 实现 | 3 | 0.5 天 |
| 8. 集成验收 | 4 | 0.5 天 |
| **总计** | **38** | **8.5 天** |
