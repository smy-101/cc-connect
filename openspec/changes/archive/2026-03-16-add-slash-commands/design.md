## Context

当前系统已实现：
- `internal/core/message.go` - 统一消息模型，已定义 `MessageTypeCommand`
- `internal/core/router.go` - 消息路由器，支持按 MessageType 注册处理器
- `internal/core/session.go` - 会话管理器
- `internal/agent/agent.go` - Agent 接口，包含 `SetPermissionMode`、`Stop` 等方法
- `internal/agent/claudecode/permission.go` - 权限模式解析和别名映射
- `internal/platform/feishu/adapter.go` - 飞书适配器，处理消息收发

**缺失部分**：命令解析和执行逻辑。

## Goals / Non-Goals

**Goals:**
- 实现命令解析器，识别 `/` 开头的命令
- 实现 5 个核心命令：`/mode`、`/new`、`/list`、`/help`、`/stop`
- 在飞书适配器层检测并转换命令消息
- 通过 Router 路由命令到执行器
- 保持 TDD 开发流程，测试覆盖率 > 85%

**Non-Goals:**
- `/allow`、`/provider`、`/cron` 命令（后续阶段）
- 飞书卡片消息格式（使用纯文本响应）
- 命令历史记录
- 命令自动补全

## Decisions

### 1. 命令检测位置：平台适配器层

**决策**：在 Feishu Adapter 中检测命令，将文本消息转为 `MessageTypeCommand`。

**理由**：
- 不同平台可能有不同的命令语法（如 Slack 用 `/`，Discord 用 `!`）
- 平台适配器负责将平台特定格式转为统一消息模型
- 保持核心 `command` 包的平台无关性

**替代方案**：
- Router 中间件：集中处理但增加 Router 职责
- 独立预处理层：增加复杂度

**代码位置**：`internal/platform/feishu/adapter.go`

```
飞书事件 → MessageConverter
              │
              ├─ IsCommand(text)? ─┬─ Yes → MessageTypeCommand
              │                    │
              └────────────────────┴─ No  → MessageTypeText
```

### 2. 命令包结构

**决策**：创建 `internal/core/command/` 包，包含：

| 文件 | 职责 |
|------|------|
| `types.go` | `Command`、`CommandResult` 结构定义 |
| `parser.go` | `IsCommand()`、`Parse()` 函数 |
| `executor.go` | `Executor` 结构，持有 Agent 和 SessionManager 引用 |
| `handlers.go` | 各命令的处理器函数 |

**理由**：
- 符合 Go 项目布局规范
- 职责清晰，易于测试
- 无外部依赖

### 3. 命令结构

```go
// Command 表示解析后的命令
type Command struct {
    Name string   // 命令名，如 "mode"
    Args []string // 参数列表，如 ["yolo"]
}

// CommandResult 表示命令执行结果
type CommandResult struct {
    Message string // 返回给用户的消息
    Error   error  // 执行错误（如有）
}
```

### 4. 命令解析规则

**决策**：使用简单的字符串分割，支持空格分隔参数。

```go
// IsCommand 检查文本是否为命令（以 / 开头，不含前导空格）
func IsCommand(text string) bool {
    return len(text) > 1 && text[0] == '/'
}

// Parse 解析命令字符串
func Parse(text string) Command {
    parts := strings.Fields(text[1:]) // 去掉 / 并分割
    if len(parts) == 0 {
        return Command{}
    }
    return Command{
        Name: parts[0],
        Args: parts[1:],
    }
}
```

**理由**：
- 简单够用，无需正则表达式
- 支持未来扩展（多参数命令）

### 5. Executor 设计

```go
// Executor 命令执行器
type Executor struct {
    agent    agent.Agent          // AI 代理
    sessions *core.SessionManager // 会话管理器
}

// Execute 执行命令并返回结果
func (e *Executor) Execute(ctx context.Context, cmd Command, msg *core.Message) CommandResult {
    switch cmd.Name {
    case "mode":
        return e.handleMode(ctx, cmd, msg)
    case "new":
        return e.handleNew(ctx, cmd, msg)
    case "list":
        return e.handleList(ctx, cmd, msg)
    case "help":
        return e.handleHelp(ctx, cmd, msg)
    case "stop":
        return e.handleStop(ctx, cmd, msg)
    default:
        return CommandResult{
            Message: fmt.Sprintf("未知命令: /%s\n输入 /help 查看可用命令", cmd.Name),
            Error:   fmt.Errorf("unknown command: %s", cmd.Name),
        }
    }
}
```

### 6. 命令处理器设计

**`/mode` 处理器**：
```go
func (e *Executor) handleMode(ctx context.Context, cmd Command, msg *core.Message) CommandResult {
    if len(cmd.Args) == 0 {
        // 无参数，显示当前模式
        currentMode := e.agent.CurrentMode()
        return CommandResult{
            Message: fmt.Sprintf("当前权限模式: %s", currentMode),
        }
    }

    modeArg := cmd.Args[0]
    mode, err := claudecode.ParsePermissionMode(modeArg)
    if err != nil {
        return CommandResult{
            Message: fmt.Sprintf("无效的权限模式: %s\n可用模式: default, edit, plan, yolo", modeArg),
            Error:   err,
        }
    }

    if err := e.agent.SetPermissionMode(mode); err != nil {
        return CommandResult{
            Message: fmt.Sprintf("切换模式失败: %v", err),
            Error:   err,
        }
    }

    desc := claudecode.PermissionModeDescription(mode)
    return CommandResult{
        Message: fmt.Sprintf("✅ 已切换到 %s 模式\n%s", modeArg, desc),
    }
}
```

**`/new` 处理器**：
```go
func (e *Executor) handleNew(ctx context.Context, cmd Command, msg *core.Message) CommandResult {
    sessionID := core.DeriveSessionID(msg)

    // 销毁旧会话（如有）
    e.sessions.Destroy(sessionID)

    // 创建新会话
    newSession := e.sessions.GetOrCreate(sessionID)

    var sessionName string
    if len(cmd.Args) > 0 {
        sessionName = cmd.Args[0]
        newSession.SetMetadata("name", sessionName)
    }

    if sessionName != "" {
        return CommandResult{Message: fmt.Sprintf("✅ 已创建新会话: %s", sessionName)}
    }
    return CommandResult{Message: "✅ 已创建新会话"}
}
```

### 7. Feishu 集成

修改 `adapter.go`，在消息转换时检测命令：

```go
func (a *Adapter) handleEvent(ctx context.Context, event *FeishuEvent) error {
    msg, err := a.converter.ConvertEvent(event)
    if err != nil {
        return err
    }

    // 检测命令
    if msg.Type == core.MessageTypeText && command.IsCommand(msg.Content) {
        msg.Type = core.MessageTypeCommand
    }

    return a.router.Route(ctx, msg)
}
```

## Risks / Trade-offs

| 风险 | 缓解措施 |
|------|----------|
| 命令参数解析不够灵活 | 当前使用简单分割，后续可升级为更复杂的解析器 |
| 模式切换时 Agent 需重启 | 已有 `SetPermissionMode` 实现，会自动处理重启 |
| 会话销毁可能丢失状态 | `/new` 是显式操作，用户已知会清除上下文 |
| 命令响应延迟 | 命令处理在内存中完成，延迟可忽略 |

## 错误处理策略

1. **未知命令**：返回友好提示，建议使用 `/help`
2. **无效参数**：返回参数格式说明
3. **Agent 操作失败**：返回具体错误信息，不静默失败
4. **会话不存在**：`/list` 返回空列表，不报错

## 测试策略

1. **单元测试**：
   - `parser_test.go`：测试 `IsCommand`、`Parse` 的各种输入
   - `executor_test.go`：使用 `MockAgent` 测试各命令处理器

2. **集成测试**：
   - 测试 Feishu Adapter 正确转换命令消息
   - 测试 Router 正确路由到 CommandExecutor

3. **覆盖率目标**：> 85%
