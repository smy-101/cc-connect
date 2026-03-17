# Tasks

## Overview

实现双斜杠命令功能，允许用户在聊天中使用 Claude Code 的斜杠命令。

## Implementation Order

按 TDD 原则，先写测试再实现。

---

## Task 1: 添加 IsClaudeCodeCommand 函数 ✅

**文件**: `internal/core/command/parser.go`

**描述**: 新增函数检测双斜杠前缀

**实现**:
```go
// IsClaudeCodeCommand checks if the given text is a Claude Code slash command.
// A Claude Code command must start with "//" (double slash) followed by at least one character.
func IsClaudeCodeCommand(text string) bool {
    return len(text) >= 3 && text[0] == '/' && text[1] == '/'
}
```

**验收标准**:
- `//cost` → true
- `//compact focus` → true
- `/mode` → false
- `///mode` → false
- `//` → false
- 空字符串 → false

---

## Task 2: 更新 feishu adapter 消息分类逻辑 ✅

**文件**: `internal/platform/feishu/adapter.go`

**描述**: 在 HandleEvent 中添加双斜杠命令检测，在单斜杠检测之前

**实现位置**: `HandleEvent` 函数，约第 90-93 行

**当前代码**:
```go
// Detect slash commands: convert text messages starting with '/' to command type
if msg.Type == core.MessageTypeText && command.IsCommand(msg.Content) {
    msg.Type = core.MessageTypeCommand
}
```

**修改为**:
```go
// Detect Claude Code commands (double slash): remove one slash and send to Agent
if msg.Type == core.MessageTypeText && command.IsClaudeCodeCommand(msg.Content) {
    // Remove one slash: //cost → /cost
    msg.Content = strings.TrimPrefix(msg.Content, "/")
    // Keep as MessageTypeText so it flows to Agent
}

// Detect cc-connect slash commands: convert text messages starting with '/' to command type
if msg.Type == core.MessageTypeText && command.IsCommand(msg.Content) {
    msg.Type = core.MessageTypeCommand
}
```

**验收标准**:
- `//cost` → Content 变为 `/cost`，Type 保持 `MessageTypeText`
- `//compact focus on auth` → Content 变为 `/compact focus on auth`
- `/mode yolo` → Type 变为 `MessageTypeCommand`，Content 不变
- 普通文本 → 无变化

---

## Task 3: 更新 /help 命令输出 ✅

**文件**: `internal/core/command/handlers.go`

**描述**: 在帮助信息中添加 Claude Code 命令说明

**实现**: 更新 `handleHelp` 函数的帮助文本

```go
helpText := `可用命令:

/mode [mode]  - 切换权限模式
  可用模式: default, edit, plan, yolo
  无参数时显示当前模式

/new [name]   - 创建新会话
  清除当前上下文，开始新对话
  可选参数: 会话名称

/list         - 列出所有活跃会话

/help         - 显示此帮助信息

/stop         - 停止当前 Agent

/project [name] [--keep|-k] - 项目管理
  无参数时显示项目列表
  /project <name> - 切换到指定项目
  --keep / -k - 切换时保留会话

Claude Code 命令 (双斜杠 //):
//cost        显示 token 使用统计
//compact     压缩对话历史
//review      请求代码审查
//init        初始化项目 CLAUDE.md
//pr-comments 查看 PR 评论
//security-review 安全审查

提示: 使用 // 前缀调用 Claude Code 的原生命令
`
```

**验收标准**:
- `/help` 输出包含 "Claude Code 命令 (双斜杠 //)" 部分
- 列出至少 5 个常用 Claude Code 命令

---

## Task 4: 编写单元测试 ✅

**文件**: `internal/core/command/parser_test.go`

**描述**: 为 `IsClaudeCodeCommand` 编写测试

```go
func TestIsClaudeCodeCommand(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected bool
    }{
        {"double slash command", "//cost", true},
        {"double slash with args", "//compact focus on auth", true},
        {"single slash", "/mode", false},
        {"triple slash", "///mode", false},
        {"only double slash", "//", false},
        {"no slash", "hello", false},
        {"empty string", "", false},
        {"text starting with double slash", "// test", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := command.IsClaudeCodeCommand(tt.input); got != tt.expected {
                t.Errorf("IsClaudeCodeCommand(%q) = %v, want %v", tt.input, got, tt.expected)
            }
        })
    }
}
```

---

## Task 5: 更新 adapter 测试 ✅

**文件**: `internal/platform/feishu/adapter_test.go`

**描述**: 添加双斜杠命令处理的测试用例

**测试场景**:
1. 双斜杠命令被正确转换
2. 单斜杠命令仍被识别为 command 类型
3. 普通文本不受影响

---

## Task 6: 集成测试验证 ✅

**描述**: 手动或自动测试端到端流程

**测试步骤**:
1. 启动 cc-connect
2. 通过 Feishu 发送 `//cost`
3. 验证返回费用统计信息
4. 发送 `/cost`（单斜杠）
5. 验证返回 "未知命令" 提示
6. 发送 `/help`
7. 验证帮助信息包含双斜杠命令说明

---

## Dependencies

- 无外部依赖
- 依赖 Claude Code CLI 的 stream-json 模式支持斜杠命令（已通过实验验证）

## Estimated Effort

小型变更，约 2-3 小时

## Risks

| 风险 | 缓解措施 |
|-----|---------|
| Claude Code CLI 版本差异 | 透传模式，由 CLI 处理错误 |
| 用户混淆 / 和 // | 清晰的帮助信息 |
