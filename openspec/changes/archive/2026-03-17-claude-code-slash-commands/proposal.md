## Why

用户在与 cc-connect 机器人聊天时，希望能够直接使用 Claude Code 的斜杠命令（如 `/cost`、`/compact`、`/review` 等），而不是只能通过 cc-connect 的内置命令。

当前问题：
- cc-connect 内置命令（`/mode`、`/project` 等）与 Claude Code 命令共用单斜杠 `/` 前缀
- 用户无法在聊天中触发 Claude Code 的原生能力（如代码审查、费用统计等）
- 两个系统的命令命名空间存在冲突（如 `/help`）

**解决方案**：使用双斜杠 `//` 作为 Claude Code 命令的前缀，单斜杠 `/` 保留给 cc-connect 内置命令。

## What Changes

- **新增双斜杠命令解析**：识别以 `//` 开头的消息，将其转换为 Claude Code 命令并发送给 Agent
- **消息分类逻辑修改**：在 Feishu adapter 中新增对 `//` 前缀的检测
- **命令帮助更新**：`/help` 输出中增加双斜杠命令的使用说明

### 命令映射规则

| 用户输入 | 系统处理 | 目标 |
|---------|---------|------|
| `/mode yolo` | cc-connect Executor | 内置命令 |
| `/project backend` | cc-connect Executor | 内置命令 |
| `//cost` | 转为 `/cost` → Agent | Claude Code 命令 |
| `//compact` | 转为 `/compact` → Agent | Claude Code 命令 |
| `//review` | 转为 `/review` → Agent | Claude Code 命令 |
| `普通文本` | 直接 → Agent | 正常对话 |

### 可用的 Claude Code 命令（经实验验证）

以下命令在 stream-json 模式下可用：
- `//cost` - 显示 token 使用统计
- `//compact` - 压缩对话历史
- `//context` - 显示上下文信息
- `//init` - 初始化项目
- `//review` - 请求代码审查
- `//pr-comments` - 查看 PR 评论
- `//release-notes` - 生成发布说明
- `//security-review` - 安全审查
- `//insights` - 项目洞察

## Capabilities

### New Capabilities

- `claude-code-commands`: 支持在聊天中通过双斜杠语法调用 Claude Code 的斜杠命令

### Modified Capabilities

- `command-system`: 扩展现有命令系统以支持双斜杠前缀的 Claude Code 命令透传

## Impact

### 受影响模块

- `internal/platform/feishu/adapter.go` - 消息分类逻辑
- `internal/core/command/parser.go` - 新增 `IsClaudeCodeCommand()` 函数
- `internal/core/command/handlers.go` - 更新 `/help` 输出
- `internal/core/message.go` - 可选：新增 `MessageTypeClaudeCommand`（或复用 `MessageTypeText`）

### API 变更

- 无公开 API 变更
- 内部消息处理流程扩展

### 依赖

- 无新增外部依赖
- 依赖 Claude Code CLI 的 stream-json 输入模式

## 验收标准

1. **功能验收**
   - 发送 `//cost` 返回费用统计信息
   - 发送 `//compact` 正确执行压缩（或返回"无消息可压缩"）
   - 发送 `/cost`（单斜杠）返回 cc-connect 的"未知命令"提示
   - 发送 `/mode` 和 `//compact` 可以正确区分路由

2. **边界验收**
   - 空命令 `//` 不应崩溃
   - 不存在的命令 `//unknown` 返回 Claude Code 的 "Unknown skill" 错误
   - 带参数的命令 `//compact focus on auth` 正确传递

3. **帮助信息**
   - `/help` 输出包含双斜杠命令的使用说明
   - 列出常用的 Claude Code 命令

## 风险与缓解

| 风险 | 影响 | 缓解措施 |
|-----|------|---------|
| Claude Code CLI 命令格式变更 | 透传失败 | 版本检测 + 错误回退提示 |
| 用户混淆单双斜杠 | 体验不佳 | 清晰的帮助文档 + 错误提示 |
| 某些命令在 stream-json 模式不可用 | 功能受限 | 维护可用命令列表，对不可用命令给出提示 |

## 实验验证

已通过 CLI 直接测试验证：
```bash
echo '{"type":"user","message":{"role":"user","content":"/cost"}}' | \
  claude -p --output-format stream-json --input-format stream-json --verbose
```

结果：`/cost` 成功返回使用统计，确认 stream-json 模式支持斜杠命令。
