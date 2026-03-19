## Why

当前 cc-connect 已经实现了 Claude Code 的 `control_request` 事件识别和 `RespondPermission` API，但缺少完整的状态机机制来处理异步的用户交互场景：

1. **工具权限请求**：当 Claude 需要执行 Bash、Write 等工具时，用户无法在飞书中实时批准或拒绝
2. **用户问答交互**：当 Claude 使用 AskUserQuestion 工具询问用户时，用户无法在飞书中选择选项或回复答案
3. **交互体验缺失**：缺少飞书交互式卡片支持，无法提供点击按钮式的流畅交互体验

本变更属于**阶段 3 (Claude Code 适配器)** 的增强，提前实现**阶段 6** 的部分交互能力，使 MVP 具备完整的用户交互闭环。

## What Changes

### 新增能力

- **异步权限状态机**：引入 `pendingPermission` 状态管理，支持权限请求的阻塞/唤醒机制
- **飞书交互式卡片**：实现 `CardSender` 接口，支持发送带有可点击按钮的交互式消息卡片
- **AskUserQuestion 解析与响应**：解析 Claude 的结构化问题，在飞书中展示选项按钮并收集用户答案
- **权限请求路由**：将权限请求事件路由到飞书平台，并处理用户回调响应
- **自动批准模式**：支持 `/mode yolo` 等命令切换权限模式，实现自动批准所有请求

### 修改行为

- **Session 接口扩展**：`SendMessage` 需要支持返回 "等待用户输入" 状态
- **事件流处理**：事件循环需要支持在权限请求处暂停，等待用户响应后恢复

## Capabilities

### New Capabilities

- `permission-state-machine`: 权限请求状态机，管理 pendingPermission 状态、resolved channel 阻塞/唤醒机制
- `feishu-interactive-card`: 飞书交互式卡片，实现 CardSender 接口，支持发送和接收卡片按钮回调
- `ask-user-question`: 用户问答交互，解析 AskUserQuestion 工具调用，展示选项并收集答案

### Modified Capabilities

- `claude-session`: 扩展 Session 接口，增加权限请求事件类型和 `RespondPermission` 方法的完整实现
- `message-router`: 扩展消息路由，增加权限请求事件的分发和用户响应的回传路径

## Impact

### 受影响模块

| 模块 | 影响程度 | 说明 |
|------|---------|------|
| `internal/agent/claudecode` | 高 | 需要实现 pendingPermission 状态机 |
| `internal/core` | 高 | 需要扩展消息路由支持权限事件 |
| `internal/platform/feishu` | 高 | 需要实现交互式卡片发送和回调处理 |
| `internal/agent` | 中 | 需要扩展 Agent 和 Session 接口 |
| `config` | 低 | 可能需要增加卡片回调相关配置 |

### 接口变更

```go
// AgentSession 接口扩展
type AgentSession interface {
    // 现有方法...

    // 新增：发送权限响应
    RespondPermission(requestID string, result PermissionResult) error
}

// 新增：权限请求事件
type Event struct {
    // 现有字段...
    Type         EventType
    RequestID    string         // 权限请求唯一标识
    ToolName     string         // 请求的工具名称
    ToolInput    string         // 工具输入预览
    Questions    []UserQuestion // AskUserQuestion 的结构化问题
}
```

### 外部依赖

- 飞书开放平台卡片回调 API
- 飞书消息卡片 JSON Schema
