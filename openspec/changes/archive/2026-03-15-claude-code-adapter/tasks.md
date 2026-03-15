# Claude Code 适配器实现任务

## 1. 基础设施准备

- [x] 1.1 创建 `internal/agent/` 包结构
  - **影响**: `internal/agent/doc.go`
  - **验证**: `go build ./internal/agent/...` 成功

- [x] 1.2 定义 Agent 接口和通用类型
  - **影响**: `internal/agent/agent.go`
  - **内容**: `Agent` interface, `AgentStatus`, `PermissionMode`, `Response`, `StreamEvent`
  - **验证**: `go build ./internal/agent/...` 成功

- [x] 1.3 创建 `internal/agent/claudecode/` 子包
  - **影响**: `internal/agent/claudecode/doc.go`
  - **验证**: `go build ./internal/agent/...` 成功

## 2. 事件类型定义 (TDD)

- [x] 2.1 编写 JSONL 事件解析测试
  - **影响**: `internal/agent/claudecode/parser_test.go`
  - **测试场景**:
    - `system/init` 事件解析
    - `assistant` 文本事件解析
    - `assistant` tool_use 事件解析
    - `user` tool_result 事件解析
    - `result/success` 事件解析
    - `result/error` + permission_denials 事件解析
  - **验证**: `go test -run TestParseEvent ./internal/agent/...` 失败（红）

- [x] 2.2 实现事件类型和解析
  - **影响**: `internal/agent/claudecode/parser.go`, `internal/agent/claudecode/events.go`
  - **TDD**: 实现 `StreamParser` 类型和 `Parse()` 方法
  - **验证**: `go test -run TestParseEvent ./internal/agent/...` 通过（绿）

## 3. 权限模式管理 (TDD)

- [x] 3.1 编写权限模式测试：模式映射和别名
  - **影响**: `internal/agent/claudecode/permission_test.go`
  - **测试场景**:
    - `default` → `default`
    - `edit` → `acceptEdits`
    - `yolo` → `bypassPermissions`
    - `plan` → `plan`
    - 无效模式返回错误
  - **验证**: `go test -run TestPermissionMode ./internal/agent/...` 失败（红）

- [x] 3.2 实现权限模式映射
  - **影响**: `internal/agent/claudecode/permission.go`
  - **TDD**: 实现 `PermissionMode` 类型、`ParseMode()`、`ToCLIArg()` 方法
  - **验证**: `go test -run TestPermissionMode ./internal/agent/...` 通过（绿）

## 4. 进程管理 (TDD)

- [x] 4.1 编写进程启动测试
  - **影响**: `internal/agent/claudecode/process_test.go`
  - **测试场景**:
    - 构建正确的命令参数 (`--print`, `--output-format stream-json`, `--session-id`, `--permission-mode`)
    - 设置正确的工作目录
    - 获取 stdin/stdout/stderr 管道
  - **验证**: `go test -run TestProcessStart ./internal/agent/...` 失败（红）

- [x] 4.2 实现进程启动逻辑
  - **影响**: `internal/agent/claudecode/process.go`
  - **TDD**: 实现 `ProcessManager` 类型和 `Start()` 方法
  - **验证**: `go test -run TestProcessStart ./internal/agent/...` 通过（绿）

- [x] 4.3 编写进程停止测试
  - **影响**: `internal/agent/claudecode/process_test.go`
  - **测试场景**:
    - 正常停止：发送 SIGTERM，等待 2s
    - 强制停止：超时后发送 SIGKILL
    - 优雅关闭流程
  - **验证**: `go test -run TestProcessStop ./internal/agent/...` 失败（红）

- [x] 4.4 实现进程停止逻辑
  - **影响**: `internal/agent/claudecode/process.go`
  - **TDD**: 实现 `Stop()` 方法，包含优雅关闭和强制终止
  - **验证**: `go test -run TestProcessStop ./internal/agent/...` 通过（绿）

- [x] 4.5 编写进程重启测试
  - **影响**: `internal/agent/claudecode/process_test.go`
  - **测试场景**:
    - 重启时使用 `--resume` 恢复会话
    - 重启时可以更改权限模式
  - **验证**: `go test -run TestProcessRestart ./internal/agent/...` 失败（红）

- [x] 4.6 实现进程重启逻辑
  - **影响**: `internal/agent/claudecode/process.go`
  - **TDD**: 实现 `Restart()` 方法
  - **验证**: `go test -run TestProcessRestart ./internal/agent/...` 通过（绿）

## 5. ClaudeCodeAgent 实现 (TDD)

- [x] 5.1 编写 Agent 初始化测试
  - **影响**: `internal/agent/claudecode/agent_test.go`
  - **测试场景**:
    - `NewAgent(config)` 返回正确初始状态
    - 自动生成 SessionID（如果未提供）
    - 使用提供的 SessionID
  - **验证**: `go test -run TestNewAgent ./internal/agent/...` 通过（绿）

  - **注意**: 宯成测试需要使用临时目录或 隔离真实文件系统操作

- [x] 5.2 实现 Agent 结构和构造函数
  - **影响**: `internal/agent/claudecode/agent.go`
  - **TDD**: 实现 `ClaudeCodeAgent` 结构和 `NewAgent()` 函数
  - **验证**: `go test -run TestNewAgent ./internal/agent/...` 通过（绿）
- [x] 5.3 编写 Start/Stop 测试
  - **影响**: `internal/agent/claudecode/agent_test.go`
  - **测试场景**:
    - `Start()` 启动进程，状态变为 running
    - `Stop()` 停止进程，状态变为 stopped
    - 重复 Start 返回错误
  - **验证**: `go test -run TestAgentStartStop ./internal/agent/...` 通过（绿）
  - **注意**: 链端测试需要临时目录
- [x] 5.4 实现 Start/Stop 方法
  - **影响**: `internal/agent/claudecode/agent.go`
  - **TDD**: 实现 `Start()` 和 `Stop()` 方法，调用 ProcessManager
  - **验证**: `go test -run TestAgentStartStop ./internal/agent/...` 通过（绿）
  - **注意**: 集成测试需要临时目录

- [x] 5.5 编写 SendMessage 测试
  - **影响**: `internal/agent/claudecode/agent_test.go`
  - **测试场景**:
    - 发送消息，接收流式事件
    - 收到 `result` 事件后结束
    - 处理 permission_denials
    - 并发调用返回 ErrAgentBusy
  - **验证**: `go test -run TestAgentSendMessage ./internal/agent/...` 失败（红）

- [x] 5.6 实现 SendMessage 方法
  - **影响**: `internal/agent/claudecode/agent.go`
  - **TDD**: 实现 `SendMessage()` 方法，整合进程通信和流解析
  - **验证**: `go test -run TestAgentSendMessage ./internal/agent/...` 通过（绿）

- [x] 5.7 编写权限模式切换测试
  - **影响**: `internal/agent/claudecode/agent_test.go`
  - **测试场景**:
    - `SetPermissionMode("yolo")` 重启进程
    - 重启后模式已更改
    - 重启后会话 ID 保持不变（通过 --resume）
  - **验证**: `go test -run TestAgentSetPermissionMode ./internal/agent/...` 失败（红）

- [x] 5.8 实现权限模式切换
  - **影响**: `internal/agent/claudecode/agent.go`
  - **TDD**: 实现 `SetPermissionMode()` 方法，调用 Restart
  - **验证**: `go test -run TestAgentSetPermissionMode ./internal/agent/...` 通过（绿）

- [x] 5.9 编写健康检查测试
  - **影响**: `internal/agent/claudecode/agent_test.go`
  - **测试场景**:
    - 进程崩溃后检测
    - 自动重启恢复
  - **验证**: `go test -run TestAgentHealth ./internal/agent/...` 失败（红）

- [x] 5.10 实现健康检查
  - **影响**: `internal/agent/claudecode/agent.go`
  - **TDD**: 实现健康检查 goroutine 和自动恢复
  - **验证**: `go test -run TestAgentHealth ./internal/agent/...` 通过（绿）

## 6. AgentManager 实现 (TDD)

- [x] 6.1 编写 AgentManager 测试
  - **影响**: `internal/agent/manager_test.go`
  - **测试场景**:
    - `GetOrCreate()` 创建新 Agent
    - `GetOrCreate()` 复用已有 Agent
    - `Remove()` 停止并移除 Agent
  - **验证**: `go test -run TestAgentManager ./internal/agent/...` 失败（红）

- [x] 6.2 实现 AgentManager
  - **影响**: `internal/agent/manager.go`
  - **TDD**: 实现 `AgentManager` 类型和工厂模式
  - **验证**: `go test -run TestAgentManager ./internal/agent/...` 通过（绿）

## 7. Mock Agent 实现 (TDD)

- [x] 7.1 编写 Mock Agent 测试
  - **影响**: `internal/agent/claudecode/mock_agent_test.go`
  - **测试场景**:
    - Mock 不启动真实进程
    - 返回预设响应
    - 模拟流式事件
    - 模拟权限拒绝
  - **验证**: `go test -run TestMockAgent ./internal/agent/...` 失败（红）

- [x] 7.2 实现 Mock Agent
  - **影响**: `internal/agent/claudecode/mock_agent.go`
  - **TDD**: 实现 `MockAgent` 类型，满足 `Agent` 接口
  - **验证**: `go test -run TestMockAgent ./internal/agent/...` 通过（绿）

- [x] 7.3 实现 Mock 响应配置
  - **影响**: `internal/agent/claudecode/mock_agent.go`
  - **功能**: `SetResponse()`, `SetError()`, `SetPermissionDenied()` 方法
  - **验证**: `go test -run TestMockAgentResponse ./internal/agent/...` 通过

## 8. 集成与验收

- [x] 8.1 编写与 core.Session 集成测试
  - **影响**: `internal/agent/claudecode/integration_test.go`
  - **测试场景**:
    - Session.AgentID 存储 Claude Code session-id
    - Session.PermissionMode 同步
    - Session.Metadata["approved_tools"] 累积
  - **验证**: `go test -run TestSessionIntegration ./internal/agent/...` 通过

- [x] 8.2 编写与 core.Router 集成测试
  - **影响**: `internal/agent/claudecode/integration_test.go`
  - **测试场景**:
    - Router 注册文本消息处理器
    - 消息流转：Router → Agent → 响应
  - **验证**: `go test -run TestRouterIntegration ./internal/agent/...` 通过

- [x] 8.3 运行完整测试套件
  - **验证**: `go test ./internal/agent/... -v -cover` 覆盖率 ≥ 85%
  - **状态**:
    - `internal/agent`: 90.9% ✓
    - `internal/agent/claudecode`: 79.3% (未达 85% 目标)
  - **覆盖率限制原因**:
    - `SendMessage` (20.5%): 内部创建 ProcessManager 并启动真实子进程，需重构支持依赖注入
    - `Stop` (51.9%): SIGKILL 超时路径难以触发
    - `Restart` (36.4%): 依赖 Stop 的完整路径
  - **改进方案**:
    - 短期: 标注为已知限制
    - 中期: 重构 ClaudeCodeAgent 使用依赖注入
    - 长期: 添加 CI 集成测试 (`-tags=integration`)

- [x] 8.4 运行竞态检测
  - **验证**: `go test ./internal/agent/... -race` 通过

- [x] 8.5 运行全部测试
  - **验证**: `go test ./...` 通过

- [x] 8.6 添加进程集成测试 (需要 -tags=integration)
  - **影响**: `internal/agent/claudecode/process_integration_test.go`
  - **测试场景**:
    - 真实子进程启动/停止
    - 优雅关闭 vs 强制终止
    - 进程重启恢复
    - 管道通信 (stdin/stdout/stderr)
    - 信号处理 (SIGTERM/SIGKILL)
    - 工作目录设置
    - 环境变量传递
    - 不同权限模式参数
    - 流式 JSONL 解析
    - 权限拒绝处理
    - 进程崩溃恢复
  - **验证**: `go test -tags=integration ./internal/agent/... -v` 通过

---

## 任务依赖关系

```
1.x (基础)
    │
    ├──▶ 2.x (事件解析) ──┐
    │                    │
    └──▶ 3.x (权限模式) ──┼──▶ 4.x (进程管理) ──▶ 5.x (Agent) ──▶ 6.x (Manager)
                         │                                              │
                         └──────────────────────────────────────────────┘
                                        │
                                        ▼
                                    7.x (Mock) ──▶ 8.x (集成)
```

**可并行任务**:
- 2.x (事件解析) 和 3.x (权限模式) 可并行
- 7.x (Mock) 可在 5.x 完成后与 6.x 并行

**关键路径**: 1.x → 2.x/3.x → 4.x → 5.x → 6.x → 8.x

## 预计工时

| 阶段 | 任务数 | 预计时间 |
|------|--------|---------|
| 1. 基础设施 | 3 | 0.5 天 |
| 2. 事件解析 | 2 | 1 天 |
| 3. 权限模式 | 2 | 0.5 天 |
| 4. 进程管理 | 6 | 1.5 天 |
| 5. Agent 实现 | 10 | 2.5 天 |
| 6. AgentManager | 2 | 0.5 天 |
| 7. Mock 实现 | 3 | 0.5 天 |
| 8. 集成验收 | 5 | 1 天 |
| **总计** | **33** | **8 天** |

## 实现顺序建议

```
Week 1:
  Day 1-2: 任务 1.x + 2.x + 3.x (基础设施 + 事件 + 权限)
  Day 3-4: 任务 4.x (进程管理)
  Day 5: 任务 5.1-5.4 (Agent 基础)

Week 2:
  Day 1-2: 任务 5.5-5.10 (Agent 完整实现)
  Day 3: 任务 6.x + 7.x (Manager + Mock)
  Day 4-5: 任务 8.x (集成测试 + 验收)
```
