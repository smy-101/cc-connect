## Context

### 当前状态

cc-connect 的消息处理流程：

```
Feishu 消息 → MessageConverter → 消息类型判断 → 路由到处理器
                                    │
                    ┌───────────────┼───────────────┐
                    ↓               ↓               ↓
              /xxx 开头        普通文本         其他类型
              → Command       → Text           → 按类型处理
              → Executor      → Agent
```

### 关键约束

1. **Claude Code CLI 通信**：使用 `--input-format stream-json` 模式
2. **消息格式**：`{"type":"user","message":{"role":"user","content":"..."}}`
3. **命令识别**：Claude Code 会解析 content 中的 `/xxx` 作为斜杠命令
4. **可用命令**：已验证 `/cost`、`/compact`、`/review` 等在 stream-json 模式下可用

### 相关代码位置

| 文件 | 职责 |
|-----|------|
| `internal/core/command/parser.go` | 命令解析，`IsCommand()` 检测 `/` 前缀 |
| `internal/platform/feishu/adapter.go:91` | 消息类型判断逻辑 |
| `internal/app/handlers.go` | `handleText` 和 `handleCommand` 处理器 |

## Goals / Non-Goals

**Goals:**
- 实现双斜杠 `//` 前缀的 Claude Code 命令识别
- 将 `//xxx` 转换为 `/xxx` 并透传给 Agent
- 更新帮助信息，说明双斜杠用法
- 保持与现有单斜杠命令的兼容性

**Non-Goals:**
- 不实现命令补全或命令列表获取
- 不在 cc-connect 层面验证 Claude Code 命令是否有效（透传即可）
- 不修改 Claude Code CLI 的行为

## Decisions

### Decision 1: 命令前缀设计

**选择**：双斜杠 `//` 作为 Claude Code 命令前缀

**备选方案**：

| 方案 | 示例 | 优点 | 缺点 |
|-----|------|------|------|
| A. 双斜杠 | `//cost` | 简洁，直观区分 | 无 |
| B. 专用命令转发 | `/cc cost` | 明确意图 | 输入更长 |
| C. 全部透传 | `/cost` 直接透传 | 最简单 | 与内置命令冲突 |

**理由**：双斜杠方案在简洁性和清晰度之间取得平衡，用户容易记忆。

### Decision 2: 实现位置

**选择**：在 `feishu/adapter.go` 的消息分类逻辑中新增检测

**当前代码**（adapter.go:90-93）：
```go
// Detect slash commands: convert text messages starting with '/' to command type
if msg.Type == core.MessageTypeText && command.IsCommand(msg.Content) {
    msg.Type = core.MessageTypeCommand
}
```

**新增逻辑**：
```go
// 检测 Claude Code 命令 (双斜杠)
if msg.Type == core.MessageTypeText && IsClaudeCodeCommand(msg.Content) {
    // 去掉一个斜杠，转为 /xxx 格式
    msg.Content = strings.TrimPrefix(msg.Content, "/")
    // 保持 MessageTypeText，让消息流向 Agent
}
// 检测 cc-connect 命令 (单斜杠)
if msg.Type == core.MessageTypeText && command.IsCommand(msg.Content) {
    msg.Type = core.MessageTypeCommand
}
```

**理由**：
- 最小改动原则
- 复用现有消息流
- 不需要新增 MessageType

### Decision 3: 函数放置位置

**选择**：在 `internal/core/command/parser.go` 中新增 `IsClaudeCodeCommand()`

```go
// IsClaudeCodeCommand checks if the text is a Claude Code slash command.
// A Claude Code command starts with "//" (double slash).
func IsClaudeCodeCommand(text string) bool {
    return len(text) >= 3 && text[0] == '/' && text[1] == '/'
}
```

**理由**：
- 与 `IsCommand()` 放在同一文件，便于维护
- 逻辑简单，不需要单独文件

### Decision 4: 帮助信息更新

**选择**：扩展 `/help` 命令输出

```go
helpText := `可用命令:
...existing...

Claude Code 命令 (双斜杠):
//cost        显示 token 使用统计
//compact     压缩对话历史
//review      请求代码审查
//init        初始化项目
...more...

提示: 使用 // 前缀调用 Claude Code 的原生命令
`
```

## Risks / Trade-offs

### Risk 1: Claude Code CLI 命令变更

**风险**：Claude Code CLI 可能新增、修改或删除命令

**缓解**：
- 透传模式：cc-connect 不验证命令有效性，由 Claude Code 返回错误
- 文档中注明可用命令列表可能随版本变化
- 可考虑从 system init 的 `slash_commands` 字段动态获取

### Risk 2: 用户混淆

**风险**：用户可能混淆 `/` 和 `//`

**缓解**：
- 清晰的帮助文档
- 对未知的单斜杠命令，提示"你是否想使用 Claude Code 命令？试试 //xxx"

### Risk 3: 边界情况

**场景**：用户输入 `///mode` 或 `////cost`

**处理**：
- `///xxx` → 检测为 Claude Code 命令 → 变为 `//xxx` → Agent 收到 `//xxx`
- 这会被 Claude Code 视为普通文本（不是命令），行为可接受

## Migration Plan

### 部署步骤

1. 部署新版本代码
2. 无需配置变更
3. 用户可立即使用 `//` 命令

### 回滚策略

- 纯新增功能，无破坏性变更
- 如有问题可直接回滚代码
- 无数据迁移

## Open Questions

1. **是否需要限制可用的 Claude Code 命令？**
   - 当前设计：不限制，全部透传
   - 后续可考虑：配置允许/禁止的命令列表

2. **是否需要从 Claude Code 动态获取命令列表？**
   - 当前设计：帮助信息中写死常用命令
   - 后续可考虑：从 system init 的 `slash_commands` 解析
