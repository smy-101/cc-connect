# Claude Code 适配器提案

## Why

cc-connect 需要一个 Claude Code 适配器来桥接用户消息与本地 AI 代理。当前已完成核心消息系统和飞书平台适配器，但缺少实际的 AI 能力提供者。Claude Code CLI 是最核心的 AI 代理，实现它的适配器是完成 MVP 端到端流程的关键一步。

这是 **阶段 3** 的核心实现，完成后用户可以通过飞书与 Claude Code 进行完整交互。

## What Changes

### 新增功能

- **子进程生命周期管理**: 启动、停止、重启 Claude Code CLI 子进程
- **流式输出解析**: 解析 `--output-format stream-json` 的实时输出事件
- **权限模式管理**: 支持 `default`、`edit`、`plan`、`yolo` 四种模式及别名映射
- **会话集成**: 与现有 SessionManager 集成，支持会话恢复和多轮对话
- **健康检查与恢复**: 监控子进程状态，支持自动重启

### 不变更

- 不修改现有核心消息结构
- 不修改飞书适配器接口
- 不实现斜杠命令（属于阶段 4）
- 不实现 TUI 监控（属于阶段 5）

## Capabilities

### New Capabilities

- `claude-code-adapter`: Claude Code CLI 适配器核心能力，包括子进程管理、流式输出解析、权限模式控制

### Modified Capabilities

无。本变更不修改现有能力的需求规格，仅新增适配器实现。

## Impact

### 受影响模块

```
internal/agent/claudecode/    # 新增包
├── agent.go                  # Agent 主结构
├── agent_test.go
├── process.go                # 子进程管理
├── process_test.go
├── stream.go                 # 流式输出解析
├── stream_test.go
├── permission.go             # 权限模式管理
├── permission_test.go
├── mock_agent.go             # 测试 Mock
└── doc.go
```

### 依赖关系

- 依赖 `internal/core` 的 `Message`、`Router`、`Session` 类型
- 依赖 Go 标准库 `os/exec`、`bufio`、`context`
- 外部依赖：Claude Code CLI（系统安装）

### 接口变更

新增 `core.Agent` 接口（如不存在则定义）:

```go
type Agent interface {
    Start(ctx context.Context) error
    Stop() error
    Process(ctx context.Context, input string) (<-chan Event, error)
    SetPermissionMode(mode PermissionMode) error
    CurrentMode() PermissionMode
}
```

## 验收标准

### 功能验收

1. **子进程管理**
   - [ ] 能够成功启动 Claude Code CLI 子进程
   - [ ] 能够优雅停止子进程（SIGTERM 后 SIGKILL）
   - [ ] 能够在崩溃后自动重启

2. **流式输出**
   - [ ] 能够解析 `text` 事件并输出文本内容
   - [ ] 能够解析 `tool_use` 事件并记录工具调用
   - [ ] 能够解析 `result` 事件并返回最终结果
   - [ ] 能够处理不完整的 JSON 缓冲

3. **权限模式**
   - [ ] `default` 模式所有工具需批准
   - [ ] `edit` / `acceptEdits` 模式编辑工具自动批准
   - [ ] `plan` 模式只读工具自动批准
   - [ ] `yolo` / `bypassPermissions` 模式全部自动批准

4. **会话集成**
   - [ ] 能够使用 `--session-id` 参数控制会话
   - [ ] 能够使用 `--resume` 恢复已有会话

### 质量验收

- [ ] 单元测试覆盖率 ≥ 85%
- [ ] 所有测试通过 `go test ./internal/agent/... -race`
- [ ] 能够在 Mock 模式下完成端到端测试

## 风险与缓解

| 风险 | 影响 | 缓解策略 |
|------|------|---------|
| CLI 在 result 后挂起 | 进程不退出 | 收到 result 后设置 5 秒超时，超时则强制终止 |
| 流式输出缓冲问题 | 数据丢失 | 使用 bufio.Scanner 逐行读取，处理不完整 JSON |
| 嵌套运行限制 | 无法在 Claude Code 内测试 | 使用 Mock Agent 进行单元测试，真实 CLI 仅在集成测试 |
| 权限模式映射错误 | 用户预期不符 | 完整测试所有模式别名和映射关系 |

## 阶段定位

本变更属于 **阶段 3: Claude Code 适配器**，是 MVP 范围内的核心功能。

完成后可进行端到端测试：
```
飞书消息 → Router → Claude Code Adapter → 响应 → 飞书
```
