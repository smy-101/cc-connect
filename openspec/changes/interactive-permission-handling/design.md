## Context

### 背景

当前 cc-connect 已实现 Claude Code 的基础集成：
- `control_request` 事件识别
- `RespondPermission` API
- `autoApprove` 模式自动批准

但缺少完整的状态机机制来处理异步用户交互，导致：
1. 权限请求无法暂停执行等待用户响应
2. AskUserQuestion 工具调用无法展示选项
3. 飞书交互式卡片能力未利用

### 参考实现

chenhg5/cc-connect 采用以下架构解决此问题：
- `pendingPermission` 状态结构，包含 `resolved` channel 实现阻塞/唤醒
- `CardSender` 接口抽象平台卡片发送能力
- `handlePendingPermission` 方法拦截用户响应

### 约束

- 必须保持与现有 Session 接口的兼容性
- 飞书卡片回调需要配置回调 URL 和加密验证
- TDD 要求：状态机逻辑必须可单元测试

## Goals / Non-Goals

**Goals:**

1. 实现 `pendingPermission` 状态机，支持阻塞/唤醒机制
2. 扩展 Session 事件系统，支持 `EventPermissionRequest` 和 `EventAskUserQuestion`
3. 实现飞书交互式卡片发送和回调处理
4. 支持 AskUserQuestion 的选项展示和答案收集
5. 支持用户通过 `/mode` 命令切换权限模式

**Non-Goals:**

1. 不实现多轮对话的复杂权限策略（如"仅本次允许"）
2. 不实现权限审计日志
3. 不实现 Telegram/Slack 等其他平台的卡片适配

## Decisions

### D1: 权限状态机设计

**选择**：在 Session 层实现 `pendingPermission` 状态

```go
// internal/agent/claudecode/session.go

type pendingPermission struct {
    requestID   string
    toolName    string
    toolInput   string
    questions   []UserQuestion  // AskUserQuestion 专用
    answers     []string        // 收集的用户答案
    resolved    chan struct{}   // 阻塞/唤醒通道
    result      *PermissionResult
    resultMu    sync.Mutex
}

type claudeSession struct {
    // ... 现有字段
    pending      *pendingPermission
    pendingMu    sync.Mutex
}
```

**原因**：
- 状态与 Session 绑定，生命周期清晰
- 利用 channel 实现阻塞/唤醒，符合 Go 惯例
- 避免引入全局状态管理器

**替代方案**：
- 在 Router 层管理状态：需要跨多个组件传递状态，复杂度高
- 使用条件变量：不如 channel 直观，且需要额外同步

### D2: 事件循环改造

**选择**：SendMessage 返回特殊状态，由上层决定阻塞等待

```go
// SendMessage 返回值扩展
type SendResult struct {
    Status       SendStatus  // "completed" | "waiting_permission"
    Content      string
    Permission   *PermissionRequest  // 非 nil 时需要处理
}

type PermissionRequest struct {
    RequestID  string
    ToolName   string
    ToolInput  string
    Questions  []UserQuestion
}
```

**原因**：
- 保持 SendMessage 非阻塞，避免 goroutine 泄漏
- 上层（Router）可以决定如何通知平台
- 便于测试：可以模拟权限请求而不实际阻塞

### D3: 飞书卡片架构

**选择**：在 feishu adapter 实现 `CardSender` 接口

```go
// internal/core/interfaces.go (新增)
type CardSender interface {
    SendCard(ctx context.Context, replyCtx any, card *Card) error
    ReplyCard(ctx context.Context, replyCtx any, card *Card) error
}

// internal/core/card.go (新增)
type Card struct {
    Header   *CardHeader
    Elements []CardElement
}

type CardButton struct {
    Text  string
    Type  string  // "primary", "default", "danger"
    Value string  // callback data
}
```

**原因**：
- 接口抽象，未来可扩展其他平台
- Card 结构与平台无关，由 adapter 负责渲染
- 按钮回调统一格式，便于路由处理

### D4: 回调处理流程

**选择**：复用现有消息路由，增加 action 前缀区分

```
用户点击按钮 → 飞书回调 → Adapter 转换 → Router
    ↓
Router 识别 "perm:allow:req123" 格式
    ↓
调用 Session.RespondPermission()
    ↓
关闭 pendingPermission.resolved channel
    ↓
原 SendMessage goroutine 继续执行
```

**原因**：
- 复用现有架构，改动最小
- action 前缀便于扩展其他命令
- 路由层统一入口，便于日志和监控

## Risks / Trade-offs

### R1: 卡片回调延迟

**风险**：飞书回调可能因网络原因延迟，导致用户感知卡顿

**缓解**：
- 发送卡片时显示 "等待响应..." 提示
- 设置超时（默认 5 分钟），超时后自动拒绝
- 支持 `/allow` 命令手动批准

### R2: Session 状态不一致

**风险**：用户在等待权限时发送新消息，可能导致状态混乱

**缓解**：
- Session 增加 busy 标志，拒绝新消息
- 或将新消息排队，权限处理完成后继续

### R3: 并发安全问题

**风险**：多个 goroutine 访问 pendingPermission

**缓解**：
- 使用 sync.Mutex 保护 pending 字段
- 使用 sync.Mutex 保护 result 字段
- channel 关闭只做一次（使用 sync.Once）

## Migration Plan

### 阶段 1：核心状态机（无平台集成）

1. 添加 `pendingPermission` 结构
2. 修改 `handleControlRequest` 创建 pending 状态
3. 扩展 `SendMessage` 返回值
4. 实现 `RespondPermission` 唤醒机制
5. 单元测试覆盖状态流转

### 阶段 2：飞书卡片集成

1. 实现 `Card` 结构和 `CardBuilder`
2. 实现 feishu adapter 的 `CardSender` 接口
3. 配置飞书卡片回调 endpoint
4. 实现回调验签和解析
5. 集成测试覆盖完整流程

### 阶段 3：AskUserQuestion 支持

1. 解析 AskUserQuestion 工具输入
2. 生成选项按钮卡片
3. 处理用户选择，构造答案
4. 通过 `RespondPermission` 返回答案

### 回滚策略

- 每个阶段可独立回滚
- 状态机失败时降级为 autoApprove 模式
- 卡片发送失败时降级为文本消息

## Open Questions

1. **权限请求超时时间**：默认 5 分钟是否合适？是否需要可配置？
2. **多问题顺序**：AskUserQuestion 多个问题是一次性展示还是逐个展示？
3. **卡片消息持久化**：权限处理后是否更新卡片状态（如显示"已批准"）？
