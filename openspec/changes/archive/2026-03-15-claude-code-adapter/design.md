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
3. **输出约束**: 流式输出使用 `--output-format stream-json`，JSONL 格式（每行一个 JSON）
4. **权限约束**: 权限模式必须在启动时设置，无法运行时切换（[GitHub Issue #11825](https://github.com/anthropics/claude-code/issues/11825)）
5. **会话持久化**: Claude Code 内置会话持久化，存储在 `~/.claude/projects/<project-hash>/<session-id>.jsonl`

### 相关方

- **用户**: 通过飞书与 Claude Code 交互
- **核心路由**: `core.Router` 负责消息分发
- **会话管理**: `core.SessionManager` 负责会话状态

## Goals / Non-Goals

**Goals:**

1. 实现稳定的 Claude Code CLI 子进程管理（持久进程）
2. 正确解析 stream-json 输出事件（JSONL 格式）
3. 支持四种权限模式及其别名，支持模式切换（通过重启进程）
4. 利用 Claude Code 内置会话持久化实现崩溃恢复
5. 与现有 Session 系统集成
6. 提供可测试的 Mock 实现

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
- JSONL 格式便于逐行解析

**命令模板**:
```bash
claude -p \
  --output-format stream-json \
  --permission-mode <mode> \
  --session-id <uuid> \
  [--resume] \
  "user message"
```

### D2: 进程生命周期管理（关键变更）

**选择**: **持久进程** - 每个 Session 对应一个长期运行的进程

**原因**:
1. **上下文保持**: 多轮对话需要连续的上下文
2. **响应速度**: 避免每次请求都启动新进程的开销
3. **会话持久化**: Claude Code 内置会话持久化，崩溃后可恢复

**进程与 Session 映射**:
```
┌─────────────────┐     ┌─────────────────────────┐
│ core.Session    │     │ ClaudeCodeAgent         │
│ ID: feishu:     │────▶│ SessionID: uuid-aaa     │
│   channel:oc_xxx│     │ Process: *exec.Cmd      │
│ AgentID: uuid   │     │ Status: running         │
│ PermissionMode: │     │ PermissionMode: edit    │
│   edit          │     └─────────────────────────┘
└─────────────────┘                 │
                                    ▼
                          ┌─────────────────┐
                          │ Claude CLI      │
                          │ --session-id    │
                          │ --permission-mode│
                          └─────────────────┘
```

**状态流转**:
```
┌───────────┐     Start()      ┌──────────┐
│  Created  │ ───────────────▶ │ Starting │
└───────────┘                  └──────────┘
                                    │
                                    ▼
┌───────────┐               ┌──────────┐
│  Stopped  │◀──────────────│ Running  │◀─────┐
└───────────┘  Stop()/Error └──────────┘      │
      │                      │    ▲           │
      │                      │    │ Process() │
      │                      ▼    │           │
      │                 ┌──────────┐          │
      └─────────────────│  Idle    │──────────┘
         Restart()      └──────────┘
```

### D3: 崩溃恢复策略

**选择**: 利用 Claude Code 内置会话持久化

**Claude Code 会话存储结构**:
```
~/.claude/
├── projects/
│   └── <project-path-hash>/
│       └── <session-id>.jsonl     ← 会话完整历史
│           └── tool-results/      ← 工具调用结果
├── session-env/<session-id>/      ← 会话环境变量
└── history.jsonl                  ← 用户命令历史
```

**恢复流程**:
```
进程崩溃检测
      │
      ▼
┌─────────────────┐
│ 保存 SessionID  │
└─────────────────┘
      │
      ▼
┌─────────────────┐
│ 重新启动进程    │  claude --resume <session-id> --permission-mode <mode>
└─────────────────┘
      │
      ▼
┌─────────────────┐
│ 上下文自动恢复  │  ✅ 对话历史不丢失
└─────────────────┘
```

### D4: 流式输出解析

**选择**: 使用 `bufio.Scanner` 逐行读取，JSONL 格式解析

**事件格式**（基于实际调研 [stream-json cheatsheet](https://takopi.dev/reference/runners/claude/stream-json-cheatsheet/)）:

```jsonl
{"type":"system","subtype":"init","session_id":"xxx","cwd":"/repo","model":"sonnet","permissionMode":"auto","tools":["Bash","Read","Write"]}
{"type":"assistant","session_id":"xxx","message":{"content":[{"type":"text","text":"Planning..."}]}}
{"type":"assistant","session_id":"xxx","message":{"content":[{"type":"tool_use","id":"toolu_1","name":"Bash","input":{"command":"ls"}}]}}
{"type":"user","session_id":"xxx","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"file1.go\nfile2.go"}]}}
{"type":"result","subtype":"success","session_id":"xxx","result":"Done.","total_cost_usd":0.0123,"duration_ms":12345}
```

**事件类型处理**:
| 事件类型 | subtype | 处理方式 |
|---------|---------|---------|
| `system` | `init` | 提取 session_id, tools, permissionMode |
| `assistant` | - | 提取文本或工具调用 |
| `user` | - | 记录工具结果 |
| `result` | `success` | 发送最终结果，准备结束 |
| `result` | `error` | 检查 permission_denials，发送错误 |

**处理流程**:
```
stdout (JSONL) ──► Scanner ──► Line ──► json.Unmarshal ──► Event
                                                    │
                    ┌───────────────────────────────┼───────────────────────────────┐
                    ▼                               ▼                               ▼
              type: system                   type: assistant                 type: result
              提取元数据                      提取内容/工具                    结束处理
```

### D5: 权限模式映射

**选择**: 内部使用规范名称，支持用户友好别名

**映射表**:
```go
var modeAliases = map[string]string{
    "default":           "default",
    "edit":              "acceptEdits",
    "acceptEdits":       "acceptEdits",
    "plan":              "plan",
    "yolo":              "bypassPermissions",
    "bypassPermissions": "bypassPermissions",
}
```

**权限模式切换**:
由于权限模式无法运行时切换，切换模式需要重启进程：

```
用户: /mode yolo
      │
      ▼
┌─────────────────┐
│ 1. 验证新模式   │
└─────────────────┘
      │
      ▼
┌─────────────────┐
│ 2. 停止当前进程 │  等待当前请求完成或超时(30s)
└─────────────────┘
      │
      ▼
┌─────────────────┐
│ 3. 启动新进程   │  claude --resume <id> --permission-mode yolo
└─────────────────┘
      │
      ▼
┌─────────────────┐
│ 4. 更新 Session │  session.PermissionMode = "yolo"
└─────────────────┘
```

### D6: 权限请求处理（混合策略）

**选择**: 分层权限处理

```
┌───────────────────────────────────────────────────────────────────────┐
│                    权限处理混合策略                                    │
├───────────────────────────────────────────────────────────────────────┤
│                                                                       │
│   层级 1: 全局权限模式                                                │
│   ─────────────────────                                              │
│   /mode yolo    → --permission-mode bypassPermissions                │
│   /mode edit    → --permission-mode acceptEdits                      │
│   /mode plan    → --permission-mode plan                             │
│   /mode default → --permission-mode default                          │
│                                                                       │
│   层级 2: 单次请求批准                                                │
│   ─────────────────────                                              │
│   收到 permission_denials:                                           │
│   1. 解析被拒绝的工具和参数                                           │
│   2. 向飞书用户展示清晰的批准请求                                     │
│   3. 用户批准 → 重试并添加 --allowedTools                            │
│   4. 用户拒绝 → 告知用户并结束                                       │
│                                                                       │
│   层级 3: 会话级批准累积                                              │
│   ─────────────────────                                              │
│   用户批准过的工具缓存到 Session.Metadata:                           │
│   "approved_tools": "Bash(go test),Read"                             │
│                                                                       │
└───────────────────────────────────────────────────────────────────────┘
```

**permission_denials 格式**:
```json
{
  "type": "result",
  "subtype": "error",
  "error": "Permission denied",
  "permission_denials": [{
    "tool_name": "Bash",
    "tool_use_id": "toolu_9",
    "tool_input": {"command": "git fetch origin main"}
  }]
}
```

### D7: 测试策略

**选择**: 接口抽象 + Mock 实现

**接口设计**:
```go
// internal/agent/agent.go
type Agent interface {
    // SendMessage 发送消息并获取响应
    // 支持 streaming 回调，如果 handler 为 nil 则等待完整响应
    SendMessage(ctx context.Context, content string, handler EventHandler) (*Response, error)

    // SetPermissionMode 设置权限模式（可能重启进程）
    SetPermissionMode(mode PermissionMode) error

    // CurrentMode 返回当前权限模式
    CurrentMode() PermissionMode

    // SessionID 返回 Claude Code 会话 ID
    SessionID() string

    // Start 启动 Agent
    Start(ctx context.Context) error

    // Stop 停止 Agent
    Stop() error

    // Status 返回当前状态
    Status() AgentStatus

    // Restart 重启 Agent
    Restart(ctx context.Context) error
}

type EventHandler func(event StreamEvent)

type StreamEvent struct {
    Type    string      // "text", "tool_use", "tool_result", "result", "error"
    Content string
    Tool    *ToolInfo
}

type Response struct {
    Content          string
    IsError          bool
    PermissionDenied bool
    DeniedTools      []DeniedTool
    CostUSD          float64
    Duration         time.Duration
}
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
- 记录警告日志

### R2: 嵌套运行限制
**风险**: 无法在 Claude Code 内测试真实 CLI
**影响**: 集成测试受限
**缓解**:
- 使用 Mock Agent 覆盖 90%+ 测试场景
- CI 环境中取消 `CLAUDECODE` 环境变量运行集成测试

### R3: 权限模式切换延迟
**风险**: 模式切换需要重启进程，有延迟
**影响**: 用户体验
**缓解**:
- 向用户明确提示"正在切换模式..."
- 重启通常只需几秒

### R4: 多进程资源占用
**风险**: 每个 Session 一个进程，资源占用随 Session 增长
**影响**: 内存/CPU 占用
**缓解**:
- 设置空闲超时（如 1 小时无活动自动停止）
- Session 归档时停止对应进程

## 包结构

```
internal/agent/
├── agent.go              # Agent 接口、通用类型
├── manager.go            # AgentManager
└── claudecode/
    ├── doc.go            # 包文档
    ├── agent.go          # ClaudeCodeAgent 实现
    ├── agent_test.go
    ├── process.go        # 子进程管理
    ├── process_test.go
    ├── parser.go         # stream-json 解析
    ├── parser_test.go
    ├── permission.go     # 权限模式管理
    ├── permission_test.go
    ├── events.go         # 事件类型定义
    ├── mock_agent.go     # Mock 实现
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
│       │ RegisterSessionHandler(MessageTypeText, handler)                   │
│       │ handler 调用 Agent.SendMessage()                                   │
│       ▼                                                                     │
│   agent.Agent (interface) ◄──────── claudecode.ClaudeCodeAgent             │
│       │                                    │                                │
│       │                                    │ 子进程通信                      │
│       ▼                                    ▼                                │
│   core.SessionManager               Claude Code CLI                         │
│       │                                    │                                │
│       │ AgentID = session-id              │ --session-id                   │
│       │ PermissionMode                    │ --permission-mode              │
│       │ Metadata["approved_tools"]        │ --allowedTools                 │
│       ▼                                    ▼                                │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 参考资料

- [Claude Code Headless Mode](https://code.claude.com/docs/en/headless)
- [stream-json Output Format Cheatsheet](https://takopi.dev/reference/runners/claude/stream-json-cheatsheet/)
- [Permission Mode Feature Request #11825](https://github.com/anthropics/claude-code/issues/11825)
