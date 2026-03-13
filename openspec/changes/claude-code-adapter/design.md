# Claude Code 适配器设计

## Context

### 背景与现状

cc-connect 项目已完成：
- 核心消息系统 (`internal/core/`)
- 飞书平台适配器 (`internal/platform/feishu/`)

现在需要实现 Claude Code 适配器，将用户消息转发给 Claude Code CLI，并将响应返回给消息路由。

### 约束条件

1. **环境约束**: Claude Code CLI 无法在嵌套会话中运行（`CLAUDECODE=1` 时拒绝启动）
2. **进程约束**: CLI 使用 `--print` 非交互模式，通过 stdin/stdout 通信
3. **输出约束**: 流式输出使用 `--output-format stream-json`，需要逐行解析 JSON
4. **已知问题**: CLI 在发送 `result` 事件后可能挂起，需要超时处理

### 相关方

- **用户**: 通过飞书与 Claude Code 交互
- **核心路由**: `core.Router` 负责消息分发
- **会话管理**: `core.SessionManager` 负责会话状态

## Goals / Non-Goals

**Goals:**

1. 实现稳定的 Claude Code CLI 子进程管理
2. 正确解析流式 JSON 输出事件
3. 支持四种权限模式及其别名
4. 与现有会话系统集成
5. 提供可测试的 Mock 实现

**Non-Goals:**

1. 不实现斜杠命令解析（阶段 4）
2. 不实现 TUI 监控界面（阶段 5）
3. 不支持多代理并行（阶段 6）
4. 不处理语音、图像等多模态输入（阶段 6）

## Decisions

### D1: 子进程通信模式

**选择**: 使用 `--print --output-format stream-json` 模式

**原因**:
- 非交互模式，适合程序集成
- 流式输出支持实时响应
- JSON 格式便于解析

**替代方案**:
- ❌ 交互模式: 需要 PTY，复杂且不稳定
- ❌ `--output-format json`: 单次输出，无流式体验
- ❌ Anthropic API: 需要 API Key，不支持 Claude Code 的工具调用

**命令模板**:
```bash
claude -p \
  --output-format stream-json \
  --permission-mode <mode> \
  --session-id <uuid> \
  [--resume] \
  "user message"
```

### D2: 进程生命周期管理

**选择**: 每次消息创建新进程，不复用长期进程

**原因**:
- CLI 设计为单次请求-响应模式
- 避免长期进程的状态管理复杂性
- 与 `--resume` 配合实现会话连续性

**状态流转**:
```
┌─────────┐    Start()    ┌──────────┐    Process()    ┌───────────┐
│  Idle   │ ────────────► │  Ready   │ ──────────────► │  Running  │
└─────────┘               └──────────┘                 └───────────┘
     ▲                                                       │
     │                                                       │
     └──────────────────── Stop() / Timeout ◄────────────────┘
```

### D3: 流式输出解析策略

**选择**: 使用 `bufio.Scanner` 逐行读取，JSON 解析

**处理流程**:
```
stdout ──► Scanner ──► Line ──► json.Unmarshal ──► Event
                                              │
                                              ├── 成功 ──► 发送到 channel
                                              │
                                              └── 失败 ──► 缓冲等待更多数据
```

**事件类型处理**:
| 事件类型 | 处理方式 |
|---------|---------|
| `text` | 直接转发文本内容 |
| `tool_use` | 记录工具调用，继续等待 |
| `tool_result` | 记录工具结果，继续等待 |
| `result` | 发送最终结果，准备结束 |
| `error` | 发送错误，准备结束 |

### D4: 权限模式映射

**选择**: 内部使用规范名称，支持用户友好别名

**映射表**:
```go
var modeAliases = map[string]string{
    "default":             "default",
    "edit":                "acceptEdits",
    "acceptEdits":         "acceptEdits",
    "plan":                "plan",
    "yolo":                "bypassPermissions",
    "bypassPermissions":   "bypassPermissions",
}
```

**CLI 参数映射**:
- 内部 `acceptEdits` → CLI `--permission-mode acceptEdits`
- 内部 `bypassPermissions` → CLI `--permission-mode bypassPermissions`

### D5: 会话集成策略

**选择**: 使用 CLI 的 `--session-id` 和 `--resume` 参数

**实现**:
1. 首次对话: 生成 UUID，使用 `--session-id`
2. 后续对话: 使用 `--resume <session-id>`
3. 会话 ID 存储在 `core.Session` 的 `Metadata` 中

### D6: 错误处理与恢复

**选择**: 多层错误处理，自动恢复

**错误类型与处理**:
| 错误类型 | 处理策略 |
|---------|---------|
| 进程启动失败 | 返回错误，记录日志 |
| 进程崩溃 | 自动重启（最多 3 次） |
| 解析错误 | 缓冲数据，等待更多输入 |
| 超时 | SIGTERM → 等待 2s → SIGKILL |
| CLI 挂起 | 收到 result 后 5s 超时终止 |

### D7: 测试策略

**选择**: 接口抽象 + Mock 实现

**接口设计**:
```go
// internal/core/agent.go
type Agent interface {
    Start(ctx context.Context) error
    Stop() error
    Process(ctx context.Context, input string) (<-chan AgentEvent, error)
    SetPermissionMode(mode PermissionMode) error
    CurrentMode() PermissionMode
}

type AgentEvent struct {
    Type     AgentEventType
    Text     string
    ToolName string
    Result   string
    Error    error
}

type AgentEventType int

const (
    EventText AgentEventType = iota
    EventToolUse
    EventResult
    EventError
)
```

**测试层次**:
1. **单元测试**: Mock Agent，测试路由和会话集成
2. **集成测试**: 真实 CLI，测试完整流程（需要 `CLAUDECODE` 未设置）
3. **E2E 测试**: 飞书 → 路由 → Claude Code → 飞书

## Risks / Trade-offs

### R1: CLI 挂起风险
**风险**: CLI 在 `result` 事件后不退出
**影响**: 进程泄漏，资源占用
**缓解**:
- 收到 `result` 后启动 5 秒超时计时器
- 超时后发送 SIGTERM，2 秒后 SIGKILL
- 记录警告日志，标记会话异常

### R2: 嵌套运行限制
**风险**: 无法在 Claude Code 内测试真实 CLI
**影响**: 集成测试受限
**缓解**:
- 使用 Mock Agent 覆盖 90%+ 测试场景
- CI 环境中取消 `CLAUDECODE` 环境变量运行集成测试
- 文档明确说明测试限制

### R3: 流式输出缓冲
**风险**: 非 TTY 环境下输出可能缓冲
**影响**: 响应延迟
**缓解**:
- 使用 `bufio.Scanner` 而非 `ReadString`
- 测试验证实时性
- 必要时考虑 `script` 命令模拟 PTY

### R4: 会话状态一致性
**风险**: CLI 和 SessionManager 状态不同步
**影响**: 会话恢复失败
**缓解**:
- 会话 ID 由 CLI 生成（首次）或复用
- SessionManager 只存储元数据
- 不尝试同步完整会话状态

## 包结构

```
internal/agent/claudecode/
├── doc.go                 # 包文档
├── agent.go               # Agent 实现，实现 core.Agent 接口
├── agent_test.go          # TDD 测试
├── process.go             # 子进程管理（启动、停止、重启）
├── process_test.go
├── stream.go              # 流式输出解析
├── stream_test.go
├── permission.go          # 权限模式管理
├── permission_test.go
├── events.go              # 事件类型定义
├── mock_agent.go          # Mock 实现（用于测试）
└── mock_agent_test.go
```

## 接口边界

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              调用关系                                        │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   core.Router                                                               │
│       │                                                                     │
│       │ Register(MessageTypeText, handler)                                 │
│       │ handler 调用 Agent.Process()                                       │
│       ▼                                                                     │
│   core.Agent (interface) ◄──────── claudecode.Agent (implementation)       │
│       │                                    │                                │
│       │                                    │ 子进程通信                      │
│       ▼                                    ▼                                │
│   core.SessionManager               Claude Code CLI                         │
│       │                                    │                                │
│       │ 存储 session_id                    │ --session-id                   │
│       │ 存储 permission_mode               │ --permission-mode              │
│       ▼                                    ▼                                │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```
