# Claude Code 适配器提案

## Why

cc-connect 需要一个 Claude Code 适配器来桥接用户消息与本地 AI 代理。当前已完成核心消息系统和飞书平台适配器，但缺少实际的 AI 能力提供者。Claude Code CLI 是最核心的 AI 代理，实现它的适配器是完成 MVP 端到端流程的关键一步。

这是 **阶段 3** 的核心实现，完成后用户可以通过飞书与 Claude Code 进行完整交互。

## What Changes

### 新增功能

- **持久进程管理**: 每个 Session 对应一个长期运行的 Claude Code 进程，保持上下文连续性
- **会话持久化与恢复**: 利用 Claude Code 内置会话存储，崩溃后可通过 `--resume` 恢复
- **流式输出解析**: 解析 `--output-format stream-json` 的 JSONL 实时输出事件
- **权限模式管理**: 支持 `default`、`edit`/`acceptEdits`、`plan`、`yolo`/`bypassPermissions` 四种模式及别名映射
- **模式切换**: 通过重启进程实现运行时权限模式切换（保留会话上下文）
- **权限请求处理**: 处理 `permission_denials`，支持用户批准后重试

### 不变更

- 不修改现有核心消息结构
- 不修改飞书适配器接口
- 不实现斜杠命令解析（属于阶段 4）
- 不实现 TUI 监控（属于阶段 5）

## Capabilities

### New Capabilities

- `claude-code-adapter`: Claude Code CLI 适配器核心能力
  - 持久进程生命周期管理
  - JSONL 流式输出解析
  - 权限模式控制与切换
  - 会话持久化与崩溃恢复
  - 与 SessionManager 集成

### Modified Capabilities

无。本变更不修改现有能力的需求规格，仅新增适配器实现。

## Impact

### 受影响模块

```
internal/agent/                   # 新增包
├── agent.go                      # Agent 接口定义
├── manager.go                    # AgentManager
└── claudecode/                   # Claude Code 实现
    ├── doc.go
    ├── agent.go                  # Agent 主结构
    ├── agent_test.go
    ├── process.go                # 子进程管理
    ├── process_test.go
    ├── parser.go                 # JSONL 解析
    ├── parser_test.go
    ├── permission.go             # 权限模式管理
    ├── permission_test.go
    ├── events.go                 # 事件类型定义
    ├── mock_agent.go             # 测试 Mock
    └── mock_agent_test.go
```

### 依赖关系

- 依赖 `internal/core` 的 `Message`、`Router`、`Session` 类型
- 依赖 Go 标准库 `os/exec`、`bufio`、`context`
- 外部依赖：Claude Code CLI（系统安装）

### 接口变更

新增 `agent.Agent` 接口：

```go
type Agent interface {
    SendMessage(ctx context.Context, content string, handler EventHandler) (*Response, error)
    SetPermissionMode(mode PermissionMode) error
    CurrentMode() PermissionMode
    SessionID() string
    Start(ctx context.Context) error
    Stop() error
    Status() AgentStatus
    Restart(ctx context.Context) error
}
```

## 验收标准

### 功能验收

1. **持久进程管理**
   - [ ] 能够成功启动 Claude Code CLI 子进程
   - [ ] 进程保持运行，支持多轮对话
   - [ ] 能够优雅停止子进程（SIGTERM 后 SIGKILL）
   - [ ] 能够在崩溃后自动重启并恢复上下文

2. **流式输出**
   - [ ] 能够解析 `system/init` 事件并提取元数据
   - [ ] 能够解析 `assistant` 事件中的文本和工具调用
   - [ ] 能够解析 `result/success` 事件并返回最终结果
   - [ ] 能够解析 `result/error` 事件中的 `permission_denials`

3. **权限模式**
   - [ ] `default` 模式所有工具需批准
   - [ ] `edit` / `acceptEdits` 模式编辑工具自动批准
   - [ ] `plan` 模式只读工具自动批准
   - [ ] `yolo` / `bypassPermissions` 模式全部自动批准
   - [ ] 支持所有别名正确映射

4. **模式切换**
   - [ ] `SetPermissionMode()` 能够切换模式
   - [ ] 切换后进程重启，会话 ID 保持（`--resume`）
   - [ ] 切换后权限行为符合新模式

5. **会话集成**
   - [ ] `Session.AgentID` 存储 Claude Code session-id
   - [ ] `Session.PermissionMode` 与 Agent 同步
   - [ ] 会话恢复后上下文不丢失

### 质量验收

- [ ] 单元测试覆盖率 ≥ 85%
- [ ] 所有测试通过 `go test ./internal/agent/... -race`
- [ ] 能够在 Mock 模式下完成端到端测试

## 风险与缓解

| 风险 | 影响 | 缓解策略 |
|------|------|---------|
| CLI 在 result 后挂起 | 进程不退出 | 收到 result 后设置 5 秒超时，超时则强制终止 |
| 嵌套运行限制 | 无法在 Claude Code 内测试 | 使用 Mock Agent 进行单元测试，真实 CLI 仅在集成测试 |
| 权限模式切换延迟 | 用户体验 | 向用户明确提示"正在切换模式..."，通常只需几秒 |
| 多进程资源占用 | 内存/CPU | 设置空闲超时，Session 归档时停止进程 |

## 阶段定位

本变更属于 **阶段 3: Claude Code 适配器**，是 MVP 范围内的核心功能。

完成后可进行端到端测试：
```
飞书消息 → Router → Claude Code Adapter → 响应 → 飞书
```

## 参考资料

- [Claude Code Headless Mode](https://code.claude.com/docs/en/headless)
- [stream-json Output Format Cheatsheet](https://takopi.dev/reference/runners/claude/stream-json-cheatsheet/)
- [Permission Mode Feature Request #11825](https://github.com/anthropics/claude-code/issues/11825)
