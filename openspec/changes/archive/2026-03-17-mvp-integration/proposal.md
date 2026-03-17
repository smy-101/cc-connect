## Why

当前项目已完成阶段 1-4 的核心模块开发（core、feishu、claudecode、command），各模块测试覆盖率均达标。但各模块独立存在，缺乏整合入口，无法作为完整系统运行。

用户无法通过飞书发送消息并获得 AI 响应的完整体验。MVP 整合将把各模块串起来，实现端到端的可用系统。

## What Changes

- **新增 `internal/app` 整合层**：负责组件初始化、生命周期管理、消息处理器注册
- **新增 `cmd/cc-connect/main.go`**：程序入口点，加载配置、启动应用、处理信号
- **新增 `test/e2e/` 端到端测试**：验证完整消息流程（飞书 → Router → Agent → 响应）
- **改进 Handler 机制**：引入 `HandlerContext` 和 `ReplySender` 接口，支持状态提示和多消息发送
- **错误处理增强**：统一错误提示、自动恢复、超时处理

## Capabilities

### New Capabilities

- `app-integration`: Application 整合层，负责组件初始化、生命周期管理、消息处理器注册和错误处理
- `e2e-flow`: 端到端消息流程，验证从飞书接收到 AI 响应的完整链路

### Modified Capabilities

无。本次变更为新增整合层，不修改现有模块的 spec 行为。

## Impact

### 新增文件
- `internal/app/app.go` - Application 整合层
- `internal/app/handlers.go` - 消息处理器注册
- `internal/app/app_test.go` - 整合层测试
- `internal/app/doc.go` - 包文档
- `cmd/cc-connect/main.go` - CLI 入口点
- `test/e2e/e2e_test.go` - 端到端测试
- `test/e2e/mock_feishu.go` - Mock 飞书服务器
- `test/e2e/mock_agent.go` - Mock Agent

### 受影响模块
- `internal/core/router.go` - 可能需要调整 Handler 签名以支持 ReplySender
- `internal/platform/feishu/adapter.go` - 可能需要暴露更多接口供 App 使用

### 依赖关系
- `internal/app` 依赖 `core`、`feishu`、`agent`、`command`
- `cmd/cc-connect` 依赖 `internal/app`、`internal/core`（配置）

### 架构变更
```
之前: 各模块独立，无整合入口
      feishu ←→ core.Router ←→ agent
                command

之后: App 整合层统一管理
      cmd/cc-connect
            │
            ▼
      internal/app (App)
            │
      ┌─────┼─────┐
      ▼     ▼     ▼
    feishu  core  agent
           command
```

## 验收标准

### 功能验收
- [ ] 启动 `cc-connect` 后能连接飞书 WebSocket
- [ ] 用户发送文本消息能收到 AI 响应
- [ ] 用户发送斜杠命令能正确执行并收到反馈
- [ ] Agent 处理时显示"正在思考..."状态提示
- [ ] Agent 处理超时时返回友好错误信息
- [ ] `/stop` 命令能中断正在进行的 Agent 处理

### 质量验收
- [ ] `internal/app` 测试覆盖率 ≥ 80%
- [ ] e2e 测试覆盖主流程（文本消息、斜杠命令、错误处理）
- [ ] 所有现有测试继续通过
- [ ] 消息处理延迟 < 500ms（不含 AI 响应时间）

### 运维验收
- [ ] 支持 SIGINT/SIGTERM 优雅关闭
- [ ] 启动失败时有明确的错误信息
- [ ] 配置文件缺失时有友好的提示

## 风险与缓解

### 风险 1: Handler 签名变更影响现有代码
- **缓解**：保持向后兼容，提供适配层或新的 Handler 类型

### 风险 2: Agent 响应时间不可控
- **缓解**：设置合理的默认超时（5分钟），提供配置项，支持用户中断

### 风险 3: 飞书连接断开时消息丢失
- **缓解**：飞书 Adapter 已实现自动重连，断线期间的消息由飞书服务端保留

### 风险 4: e2e 测试环境复杂
- **缓解**：使用 mock 实现模拟飞书和 Agent，避免依赖真实网络
