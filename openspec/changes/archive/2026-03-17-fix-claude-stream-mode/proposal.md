## Why

当前 Claude Code 适配器使用 `-p` 单次执行模式，每条消息启动新进程后立即退出。这导致无法处理 Claude Code 的权限请求（permission prompts），也无法实现真正的双向通信。参考项目 chenhg5/cc-connect 使用 `--input-format stream-json --permission-prompt-tool stdio` 的交互式流模式，保持进程持久运行并通过 stdin/stdout 进行双向通信。

本变更属于阶段 3（Claude Code 适配器）的核心修复，是 MVP 可用的前置条件。

## What Changes

- **将 Claude Code 进程从单次执行模式切换到交互式流模式**
  - 移除 `-p` 标志，改用 `--input-format stream-json`
  - 添加 `--permission-prompt-tool stdio` 以支持权限交互
  - 进程持久运行，不再每条消息后退出

- **实现持久的会话进程管理**
  - 进程启动后保持运行，支持多条消息
  - 通过 stdin 发送 JSON 格式的用户消息
  - 从 stdout 持续读取事件流

- **实现权限请求处理**
  - 解析 `control_request` 事件
  - 通过 stdin 发送 `control_response` 响应
  - 支持自动批准（YOLO 模式）和用户交互确认

- **实现消息发送接口**
  - 支持 `Send(prompt, images, files)` 方法
  - JSON 格式: `{"type":"user","message":{"role":"user","content":"..."}}`

## Capabilities

### New Capabilities

- `claude-session`: Claude Code 会话管理能力，包括持久进程、双向通信、权限处理

### Modified Capabilities

无（这是对现有 agent/claudecode 实现的重构，不改变外部接口语义）

## Impact

### 受影响模块

| 模块 | 影响程度 | 说明 |
|------|----------|------|
| `internal/agent/claudecode/process.go` | 高 | 重写进程管理逻辑 |
| `internal/agent/claudecode/agent.go` | 高 | 改为使用持久会话 |
| `internal/agent/claudecode/parser.go` | 中 | 添加 control_request 解析 |
| `internal/agent/agent.go` | 低 | 可能需要调整接口 |
| `internal/app/handlers.go` | 低 | 调用方式不变 |

### 依赖变更

- 无新增外部依赖
- 依赖 Claude Code CLI 支持 `--input-format stream-json` 和 `--permission-prompt-tool stdio` 参数

### 兼容性

- Agent 接口保持不变，外部调用方无需修改
- 现有测试需要适配新的进程管理模式
