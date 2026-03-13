# 会话管理 - 技术设计

## Context

### 背景

cc-connect 需要在消息路由过程中跟踪用户会话状态。当前 Router 仅负责消息分发，缺乏会话上下文的概念。会话管理模块将填补这一空白，为后续的代理绑定、权限模式切换、对话历史等功能提供基础。

### 约束

- **无外部依赖**：仅使用 Go 标准库
- **内存存储**：MVP 阶段不持久化，会话数据仅在内存中
- **并发安全**：必须支持多 goroutine 并发访问
- **可测试性**：时间相关逻辑必须可注入 mock

### 层次归属

```
┌─────────────────────────────────────────────────────┐
│  internal/core/          ← 会话管理属于核心域       │
│  ├── message.go          消息结构                   │
│  ├── router.go           消息路由                   │
│  └── session.go          会话管理 (新增)            │
│                                                      │
│  internal/platform/      ← 平台适配层 (未来)        │
│  internal/agent/         ← 代理适配层 (未来)        │
└─────────────────────────────────────────────────────┘
```

## Goals / Non-Goals

**Goals:**

1. 实现会话标识系统，支持群聊和私聊两种模式
2. 实现会话状态存储，包含代理绑定、权限模式、元数据
3. 实现完整的生命周期管理：创建 → 活跃 → 归档 → 销毁
4. 实现自动清理机制，防止内存泄漏
5. 提供与 Router 的集成点

**Non-Goals:**

1. ❌ 会话持久化（数据库/文件存储）→ 后续阶段
2. ❌ 对话历史完整存储 → 后续阶段，当前仅存摘要
3. ❌ 分布式会话同步 → 超出 MVP 范围
4. ❌ 会话恢复（重启后恢复）→ 后续阶段

## Decisions

### D1: 会话 ID 格式

**决定**：使用 `platform:type:identifier` 三段式格式

```
私聊: "feishu:user:ou_xxx"
群聊: "feishu:channel:oc_xxx"
```

**理由**：
- 三段式比两段式更明确，避免 userID 和 channelID 冲突
- `type` 字段明确标识是用户还是频道，便于路由逻辑判断
- 飞书的 open_id 和 chat_id 格式不同，但用 type 字段区分更通用

**替代方案**：
- 两段式 `platform:identifier` - 拒绝，无法区分用户和频道
- UUID - 拒绝，丢失语义信息，不便于调试

### D2: 会话状态模型

**决定**：使用枚举状态 + 时间戳

```go
type SessionStatus string

const (
    SessionStatusActive    SessionStatus = "active"
    SessionStatusArchived  SessionStatus = "archived"
    SessionStatusDestroyed SessionStatus = "destroyed"
)

type Session struct {
    ID              SessionID
    Status          SessionStatus
    AgentID         string            // 绑定的代理 ID
    PermissionMode  string            // 权限模式: default, edit, plan, yolo
    Metadata        map[string]string // 元数据
    CreatedAt       time.Time
    LastActiveAt    time.Time
    ArchivedAt      *time.Time
}
```

**理由**：
- 状态枚举清晰，便于状态机验证
- 时间戳支持生命周期计算和过期判断
- Metadata 使用 `map[string]string` 简单灵活，满足 MVP 需求

### D3: 生命周期与状态转换

**决定**：单向状态转换，不可逆

```
                    ┌─────────────────────────────────┐
                    │         生命周期图              │
                    └─────────────────────────────────┘

    ┌──────────┐      ┌──────────┐      ┌──────────┐
    │  Active  │─────▶│ Archived │─────▶│Destroyed │
    └──────────┘      └──────────┘      └──────────┘
         │                 │
         │ 超时未活跃       │ 超时未恢复
         │ (默认30分钟)     │ (默认24小时)
         └────────┬────────┘
                  │
                  ▼
            自动清理 goroutine
```

**转换规则**：
- `Active → Archived`：超时未活跃或手动归档
- `Archived → Destroyed`：超时未恢复或手动销毁
- 不可逆向转换（Active ← Archived 不允许）

**超时配置**：
```go
type SessionConfig struct {
    ActiveTTL      time.Duration // 活跃会话超时，默认 30 分钟
    ArchivedTTL    time.Duration // 归档会话超时，默认 24 小时
    CleanupInterval time.Duration // 清理间隔，默认 5 分钟
}
```

### D4: 并发安全策略

**决定**：使用 `sync.RWMutex` 保护 SessionManager，Session 本身不可变

```go
type SessionManager struct {
    sessions map[SessionID]*Session
    config   SessionConfig
    mu       sync.RWMutex
    now      func() time.Time  // 可注入的时间函数，便于测试
}

// Session 返回副本，保证外部修改不影响内部状态
func (m *SessionManager) Get(id SessionID) (*Session, bool) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    if s, ok := m.sessions[id]; ok {
        return s.Clone(), true
    }
    return nil, false
}
```

**理由**：
- RWMutex 允许多读单写，性能优于互斥锁
- Session 返回副本，避免外部直接修改
- `now` 函数可注入，解决时间相关测试的困难

### D5: Router 集成方式

**决定**：扩展 Handler 签名，增加 Session 参数

```go
// 当前签名
type Handler func(ctx context.Context, msg *Message) error

// 新签名（扩展）
type SessionHandler func(ctx context.Context, msg *Message, session *Session) error

// Router 扩展
type Router struct {
    handlers map[MessageType]SessionHandler
    sessions *SessionManager
    // ...
}

// RouteWithSession 路由消息并传递会话上下文
func (r *Router) RouteWithSession(ctx context.Context, msg *Message) error {
    sessionID := DeriveSessionID(msg)
    session := r.sessions.GetOrCreate(sessionID)
    session.Touch() // 更新活跃时间
    return handler(ctx, msg, session)
}
```

**理由**：
- 保持向后兼容，原有的 `Route` 方法可保留
- Session 自动创建和管理，调用方无需关心
- `DeriveSessionID` 从 Message 提取会话 ID

### D6: 会话 ID 派生逻辑

**决定**：从 Message 结构派生 SessionID

```go
// DeriveSessionID 从消息派生会话 ID
// 规则：优先使用 channel（群聊），否则使用 userID（私聊）
func DeriveSessionID(msg *Message) SessionID {
    // Message 需要扩展 ChannelID 字段
    if msg.ChannelID != "" {
        return SessionID(fmt.Sprintf("%s:channel:%s", msg.Platform, msg.ChannelID))
    }
    return SessionID(fmt.Sprintf("%s:user:%s", msg.Platform, msg.UserID))
}
```

**对 Message 结构的影响**：
```go
type Message struct {
    ID        string      `json:"id"`
    Platform  string      `json:"platform"`
    UserID    string      `json:"user_id"`
    ChannelID string      `json:"channel_id,omitempty"` // 新增：群聊频道 ID
    Content   string      `json:"content"`
    Type      MessageType `json:"type"`
    Timestamp time.Time   `json:"timestamp"`
}
```

## Risks / Trade-offs

### R1: 内存泄漏风险

**风险**：大量用户创建会话后不活跃，导致内存无限增长

**缓解**：
- 自动清理机制定期扫描过期会话
- 可配置的 TTL 和清理间隔
- 归档状态减少活跃会话内存占用

### R2: 清理 goroutine 泄漏

**风险**：`StartCleanup` 启动的 goroutine 在程序退出时未正确停止

**缓解**：
- 使用 context 控制 goroutine 生命周期
- 提供 `StopCleanup()` 方法显式停止
- defer 在测试中确保清理

```go
func (m *SessionManager) StartCleanup(ctx context.Context) {
    go func() {
        ticker := time.NewTicker(m.config.CleanupInterval)
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                m.cleanup()
            }
        }
    }()
}
```

### R3: 时间测试困难

**风险**：超时逻辑依赖 `time.Now()`，测试不稳定

**缓解**：
- SessionManager 使用可注入的 `now` 函数
- 测试时注入 mock 时间，精确控制时间流逝

```go
// 生产使用
manager := NewSessionManager(config)

// 测试使用
mockTime := time.Now()
manager := &SessionManager{
    now: func() time.Time { return mockTime },
    // ...
}
```

### R4: Session 副本性能

**风险**：频繁 Get 操作产生大量副本，GC 压力

**缓解**：
- MVP 阶段会话数量有限，影响可接受
- 后续如需优化，可改用 sync.Map 或对象池

## Migration Plan

### 阶段 1：核心实现（本次变更）

1. 实现 Session 和 SessionManager
2. 扩展 Message 结构添加 ChannelID
3. 实现 DeriveSessionID 函数
4. 扩展 Router 集成 SessionManager
5. 完整的单元测试覆盖

### 阶段 2：集成验证（后续）

1. 平台适配器填充 ChannelID
2. 斜杠命令修改会话状态
3. 端到端测试

### 回滚策略

- SessionManager 是新增模块，不影响现有 Router 行为
- Router 的 `Route` 方法保持不变，`RouteWithSession` 是新增方法
- 可通过不调用 `RouteWithSession` 来禁用会话功能

## Open Questions

1. **会话历史摘要的存储格式**：当前 Metadata 是 `map[string]string`，是否需要更结构化的对话历史？→ 待后续阶段确定

2. **多设备同一用户**：同一用户从多个设备登录，是否共享会话？→ 当前设计按 platform:user 隔离，不同设备共享

3. **会话配额限制**：是否需要限制单个平台的最大会话数？→ MVP 不限制，后续可添加
