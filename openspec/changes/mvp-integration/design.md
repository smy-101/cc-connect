## Context

当前项目已完成以下模块：
- `internal/core`: 统一消息结构、Router、SessionManager、Config（覆盖率 94.5%）
- `internal/platform/feishu`: WebSocket 连接、消息转换、事件解析（覆盖率 92.0%）
- `internal/agent/claudecode`: 子进程管理、流式输出、权限模式（覆盖率 79.3%）
- `internal/core/command`: 斜杠命令解析与执行（覆盖率 93.3%）

各模块独立存在，缺乏整合入口。本次设计将创建 Application 整合层，把各模块串起来。

### 现有接口

```go
// core.Router - 消息路由
type Router struct {
    handlers map[MessageType]Handler
}
func (r *Router) Register(mt MessageType, h Handler) error
func (r *Router) Route(ctx context.Context, msg *Message) error

// agent.Agent - AI 代理接口
type Agent interface {
    SendMessage(ctx context.Context, content string, handler EventHandler) (*Response, error)
    SetPermissionMode(mode PermissionMode) error
    Start(ctx context.Context) error
    Stop() error
}

// feishu.Adapter - 飞书适配器
type Adapter struct {
    client    FeishuClient
    router    *core.Router
}
func (a *Adapter) Start(ctx context.Context) error
func (a *Adapter) SendReply(ctx context.Context, chatID, content string) error

// command.Executor - 命令执行器
type Executor struct {
    agent    agent.Agent
    sessions *core.SessionManager
}
func (e *Executor) Execute(ctx context.Context, cmd Command, msg *core.Message) CommandResult
```

## Goals / Non-Goals

**Goals:**
- 创建 `internal/app` 整合层，统一管理组件生命周期
- 实现 `HandlerContext` + `ReplySender` 模式，支持状态提示
- 实现 `cmd/cc-connect/main.go` 入口点
- 创建 e2e 测试验证完整流程
- 统一错误处理策略

**Non-Goals:**
- 不实现流式响应（后续阶段）
- 不实现多项目支持（阶段 6）
- 不实现 TUI 界面（阶段 5）
- 不修改现有模块的核心行为

## Decisions

### 决策 1: Handler 签名设计

**选择**: 使用 `HandlerContext` 封装上下文 + `ReplySender` 接口

```go
// ReplySender 定义回复发送能力（小接口，易 mock）
type ReplySender interface {
    SendReply(ctx context.Context, content string) error
}

// HandlerContext 封装 Handler 需要的所有上下文
type HandlerContext struct {
    Ctx     context.Context
    Msg     *core.Message
    Session *core.Session
    Reply   ReplySender
}

// Handler 新签名
type Handler func(hctx *HandlerContext) error
```

**备选方案:**
1. Handler 返回响应内容，App 统一发送 - 不够灵活，无法发送状态提示
2. 响应 Channel - 复杂度高，需要 goroutine 管理

**选择理由:**
- `ReplySender` 是小接口，只有 1 个方法，易于 mock 测试
- `HandlerContext` 可扩展，未来可添加更多字段而不改变签名
- 支持发送多条消息（状态提示 + 最终响应）

### 决策 2: App 结构设计

**选择**: 单一 App struct 管理所有组件

```go
// internal/app/app.go
type App struct {
    config   *core.AppConfig
    router   *core.Router
    agent    agent.Agent
    feishu   *feishu.Adapter
    executor *command.Executor

    // 生命周期管理
    ctx      context.Context
    cancel   context.CancelFunc
    wg       sync.WaitGroup
    mu       sync.RWMutex
    status   AppStatus
}

type AppStatus string
const (
    AppStatusIdle     AppStatus = "idle"
    AppStatusRunning  AppStatus = "running"
    AppStatusStopping AppStatus = "stopping"
    AppStatusStopped  AppStatus = "stopped"
)

func New(config *core.AppConfig) (*App, error)
func (a *App) Start(ctx context.Context) error
func (a *App) Stop() error
func (a *App) WaitForShutdown()
```

**备选方案:**
1. 分层结构（Service 层 + App 层）- 过度设计，当前规模不需要
2. 依赖注入容器 - 增加复杂度，当前依赖关系简单

**选择理由:**
- 简单直接，符合当前项目规模
- 所有组件集中管理，便于生命周期控制
- 易于测试，可以逐个 mock 组件

### 决策 3: 消息处理器注册方式

**选择**: 在 App 内部注册处理器，封装 HandlerContext 构建

```go
// internal/app/handlers.go
func (a *App) registerHandlers() error {
    // 文本消息处理器
    a.router.Register(core.MessageTypeText, a.wrapHandler(a.handleText))

    // 命令消息处理器
    a.router.Register(core.MessageTypeCommand, a.wrapHandler(a.handleCommand))

    return nil
}

// wrapHandler 将新的 Handler 签名适配到 core.Handler
func (a *App) wrapHandler(h Handler) core.Handler {
    return func(ctx context.Context, msg *core.Message) error {
        hctx := &HandlerContext{
            Ctx:     ctx,
            Msg:     msg,
            Session: a.router.Sessions().GetOrCreate(core.DeriveSessionID(msg)),
            Reply:   &replySender{adapter: a.feishu, channelID: msg.ChannelID},
        }
        return h(hctx)
    }
}
```

**选择理由:**
- 不修改 `core.Router` 签名，保持向后兼容
- 处理器逻辑集中管理，便于维护
- 自动注入 ReplySender，处理器无需了解飞书细节

### 决策 4: Agent 异步响应策略

**选择**: MVP 阶段使用"思考提示 + 最终响应"模式

```go
func (a *App) handleText(hctx *HandlerContext) error {
    // 1. 发送思考提示
    if err := hctx.Reply.SendReply(hctx.Ctx, "🤔 正在思考..."); err != nil {
        return fmt.Errorf("发送状态提示失败: %w", err)
    }

    // 2. 调用 Agent（带超时）
    ctx, cancel := context.WithTimeout(hctx.Ctx, a.config.AgentTimeout)
    defer cancel()

    resp, err := a.agent.SendMessage(ctx, hctx.Msg.Content, nil)
    if err != nil {
        if errors.Is(err, context.DeadlineExceeded) {
            return hctx.Reply.SendReply(hctx.Ctx, "⏱️ 请求超时，请简化问题或稍后重试。")
        }
        return hctx.Reply.SendReply(hctx.Ctx, fmt.Sprintf("❌ 处理失败: %v", err))
    }

    // 3. 发送最终响应
    return hctx.Reply.SendReply(hctx.Ctx, resp.Content)
}
```

**备选方案:**
1. 流式响应 - 飞书支持消息更新，但实现复杂度高，延后到后续阶段
2. 不发送状态提示 - 用户体验差，长时间无反馈

**选择理由:**
- 实现简单，MVP 阶段足够
- 用户有明确的反馈
- 支持超时和取消

### 决策 5: 错误处理策略

**选择**: 分层错误处理 + 统一用户提示

```
┌─────────────────────────────────────────────────────────────────┐
│ 错误层级         │ 处理策略                                      │
├─────────────────────────────────────────────────────────────────┤
│ 平台层 (飞书)     │ 自动重连（已在 feishu.Adapter 实现）           │
│ 代理层 (Agent)    │ 自动重启 + 通知用户                           │
│ 处理层 (Handler)  │ 捕获 panic + 友好错误提示                     │
│ 应用层 (App)      │ 优雅关闭 + 资源清理                           │
└─────────────────────────────────────────────────────────────────┘
```

```go
// wrapHandler 中捕获 panic
func (a *App) wrapHandler(h Handler) core.Handler {
    return func(ctx context.Context, msg *core.Message) (err error) {
        defer func() {
            if r := recover(); r != nil {
                log.Printf("Handler panic: %v", r)
                err = fmt.Errorf("内部错误，请稍后重试")
            }
        }()
        // ...
    }
}
```

### 决策 6: e2e 测试策略

**选择**: 使用 mock 实现，不依赖真实网络

```go
// test/e2e/mock_agent.go
type MockAgent struct {
    agent.Agent
    responses map[string]string
    delay     time.Duration
}

func (m *MockAgent) SendMessage(ctx context.Context, content string, handler agent.EventHandler) (*agent.Response, error) {
    if m.delay > 0 {
        select {
        case <-time.After(m.delay):
        case <-ctx.Done():
            return nil, ctx.Err()
        }
    }
    if resp, ok := m.responses[content]; ok {
        return &agent.Response{Content: resp}, nil
    }
    return &agent.Response{Content: "默认响应: " + content}, nil
}

// test/e2e/mock_feishu.go
type MockFeishuServer struct {
    events    chan *feishu.MessageReceiveEvent
    responses []string
    mu        sync.Mutex
}

func (m *MockFeishuServer) SendEvent(event *feishu.MessageReceiveEvent)
func (m *MockFeishuServer) GetResponses() []string
```

**选择理由:**
- 测试稳定，不依赖外部服务
- 测试速度快
- 可模拟各种场景（超时、错误、慢响应）

## Risks / Trade-offs

### 风险 1: Handler 签名变更导致现有代码需要适配
- **缓解**: 使用 `wrapHandler` 适配层，不修改 `core.Router` 接口
- **影响**: 仅新增代码，现有模块无需修改

### 风险 2: Agent 响应时间不可控
- **缓解**: 设置 5 分钟默认超时，可配置，支持用户 `/stop` 中断
- **Trade-off**: 超时时间过短会导致正常请求失败，过长影响用户体验

### 风险 3: 飞书断线期间消息丢失
- **缓解**: 飞书服务端保留未送达消息，重连后自动接收
- **Trade-off**: 无法处理断线期间用户期望的即时响应

### 风险 4: 并发消息处理导致状态不一致
- **缓解**: 使用 `sync.RWMutex` 保护共享状态，Session 级别隔离
- **Trade-off**: 增加锁竞争，但当前并发量低，影响可忽略

### 风险 5: e2e 测试覆盖不足
- **缓解**: 优先覆盖主流程（文本消息、命令），逐步增加边界场景
- **Trade-off**: 初期可能遗漏一些边界情况

## Migration Plan

### 阶段 1: 创建整合层骨架（第 1 天）
1. 创建 `internal/app/app.go` 基础结构
2. 创建 `internal/app/handlers.go` 处理器注册
3. 编写单元测试验证初始化逻辑

### 阶段 2: 实现消息处理（第 2 天）
1. 实现文本消息处理器
2. 实现命令消息处理器
3. 编写处理器单元测试

### 阶段 3: 实现入口点（第 3 天）
1. 创建 `cmd/cc-connect/main.go`
2. 实现配置加载
3. 实现信号处理和优雅关闭

### 阶段 4: e2e 测试（第 4 天）
1. 创建 mock 组件
2. 编写 e2e 测试用例
3. 验证完整流程

### 回滚策略
- 所有新增代码在独立目录，不影响现有模块
- 如需回滚，删除 `internal/app` 和 `cmd/cc-connect` 目录即可

## Open Questions

1. **配置文件路径**: 默认从哪里读取配置？`./config.toml` 还是 `~/.cc-connect/config.toml`？
   - **建议**: 优先 `./config.toml`（当前目录），支持 `-config` 参数指定路径

2. **日志格式**: 使用什么日志库？结构化日志还是简单日志？
   - **建议**: 使用 Go 标准库 `log/slog`（Go 1.21+），支持结构化日志，无额外依赖

3. **版本信息**: 如何管理和显示版本？
   - **建议**: 使用 `-ldflags` 在编译时注入版本信息，`cc-connect --version` 显示
