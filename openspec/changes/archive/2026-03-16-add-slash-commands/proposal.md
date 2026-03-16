## Why

用户需要通过聊天平台控制 AI 代理的行为，包括切换权限模式、管理会话、查看帮助等。当前系统只支持普通文本消息路由到 Agent，无法识别和处理以 `/` 开头的命令。

这是 Phase 4（斜杠命令系统）的核心实现，让用户无需接触终端即可控制 Agent 行为。

## What Changes

### 新增功能
- 命令解析器：识别并解析 `/` 开头的命令字符串
- 命令执行器：路由命令到对应处理器，返回执行结果
- 5 个核心命令实现：
  - `/mode [mode]` - 切换权限模式（default/edit/plan/yolo）
  - `/new [name]` - 创建新会话，清除当前上下文
  - `/list` - 列出所有活跃会话
  - `/help` - 显示可用命令帮助
  - `/stop` - 停止当前 Agent

### 修改
- Feishu Adapter：添加命令检测逻辑，将 `/` 开头的文本转为 `MessageTypeCommand`
- Router：注册 `MessageTypeCommand` 的处理器

## Capabilities

### New Capabilities
- `slash-commands`: 斜杠命令系统，包含命令解析、执行和 5 个核心命令的实现

### Modified Capabilities
- 无（这是新增能力，不改变现有 spec 要求）

## Impact

### 新增代码
- `internal/core/command/types.go` - Command、CommandResult 结构
- `internal/core/command/parser.go` - IsCommand()、Parse() 函数
- `internal/core/command/executor.go` - Executor 结构和 Execute() 方法
- `internal/core/command/handlers.go` - 各命令的处理器实现
- 对应的 `_test.go` 测试文件

### 修改代码
- `internal/platform/feishu/adapter.go` - 添加命令检测和转换

### 依赖
- 依赖现有 `internal/agent` 接口（SetPermissionMode、Stop 等）
- 依赖现有 `internal/core/session.go`（会话管理）
- 依赖现有 `internal/core/message.go`（MessageTypeCommand 已定义）
- 依赖现有 `internal/agent/claudecode/permission.go`（权限模式解析）

## 验收标准

### 功能验收
- [ ] `IsCommand("/mode")` 返回 `true`
- [ ] `IsCommand("hello")` 返回 `false`
- [ ] `Parse("/mode yolo")` 返回 `{Name: "mode", Args: ["yolo"]}`
- [ ] `/mode yolo` 成功切换 Agent 到 yolo 模式
- [ ] `/new` 创建新会话并返回确认消息
- [ ] `/list` 返回当前所有会话列表
- [ ] `/help` 返回所有可用命令的帮助文本
- [ ] `/stop` 成功停止 Agent
- [ ] 未知命令返回错误提示

### 集成验收
- [ ] 飞书收到 `/mode yolo` 文本消息后，Agent 模式已切换
- [ ] 命令执行结果通过飞书返回给用户

### 质量验收
- [ ] `internal/core/command/` 测试覆盖率 > 85%
- [ ] 所有测试通过 `go test ./...`

## 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 命令格式不统一 | 用户体验差 | 定义清晰的命令规范，/help 提供完整说明 |
| 模式切换失败 | Agent 状态不一致 | 返回明确的错误消息，不静默失败 |
| 会话创建冲突 | 数据混乱 | 使用 SessionManager 的线程安全方法 |

## 非目标

本次变更 **不** 包含：
- `/allow` 命令（临时允许工具）- 后续阶段实现
- `/provider` 命令（API 提供商管理）- 后续阶段实现
- `/cron` 命令（定时任务）- 后续阶段实现
- 飞书卡片消息格式的命令响应 - 使用纯文本即可
