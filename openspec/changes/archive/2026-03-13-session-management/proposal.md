# 会话管理 - 提案

## Why

cc-connect 作为聊天平台与 AI 代理的桥梁，需要跟踪每个对话的上下文状态。没有会话管理，无法实现：
- 同一用户在不同会话中使用不同代理
- 跨消息保持权限模式设置
- 对话历史追踪和恢复
- 群聊与私聊的隔离处理

会话管理是连接消息路由、代理适配器和斜杠命令的核心纽带。

## What Changes

### 新增能力

- **会话标识系统**：支持 `platform:channel`（群聊）和 `platform:userID`（私聊）两种复合键
- **会话状态管理**：存储绑定代理、权限模式、配置、元数据、对话历史摘要
- **生命周期管理**：创建 → 激活 → 过期 → 归档 → 销毁的完整流程
- **自动清理机制**：过期会话自动归档，长时间未活跃会话自动销毁
- **路由器集成**：消息路由时自动获取或创建会话，传递会话上下文给处理器

### 不变更的内容

- 消息结构（已由 unified-message 定义）
- 路由器核心逻辑（仅扩展集成点）
- 代理适配器接口（仅接收会话上下文参数）

## Capabilities

### New Capabilities

- `session-management`: 会话标识、状态存储、生命周期管理、自动清理、路由器集成

### Modified Capabilities

无。会话管理是独立的新能力域，不修改现有规格。

## Impact

### 代码影响

```
internal/core/
├── session.go          # 新增：Session 结构和 Manager
├── session_test.go     # 新增：TDD 测试
└── router.go           # 修改：集成会话上下文传递
```

### API 影响

```go
// 新增类型
type SessionID string           // 复合键 "platform:channel" 或 "platform:userID"
type Session struct { ... }     // 会话状态
type SessionManager struct { ... } // 会话管理器

// 新增接口
func (m *SessionManager) GetOrCreate(id SessionID) *Session
func (m *SessionManager) Get(id SessionID) (*Session, bool)
func (m *SessionManager) Archive(id SessionID) error
func (m *SessionManager) Destroy(id SessionID) error
func (m *SessionManager) StartCleanup(ctx context.Context) // 后台清理

// Session 方法
func (s *Session) BindAgent(agentID string)
func (s *Session) SetPermissionMode(mode string)
func (s *Session) Touch() // 更新最后活跃时间

// Router 扩展
func (r *Router) RouteWithContext(ctx context.Context, msg *Message, session *Session) error
```

### 依赖影响

- 无外部依赖新增
- 内部依赖：`internal/core/message.go`、`internal/core/router.go`

## 验收标准

| 场景 | 预期结果 |
|------|----------|
| 新用户发消息 | 自动创建会话，状态为 active |
| 群聊消息路由 | 使用 `platform:channel` 作为会话 ID |
| 私聊消息路由 | 使用 `platform:userID` 作为会话 ID |
| 会话过期（超时未活跃） | 自动归档，状态变为 archived |
| 归档会话超时 | 自动销毁，释放资源 |
| 路由消息 | 处理器收到完整的会话上下文 |
| 并发访问同一会话 | 无竞态条件，状态一致性保证 |

## 风险与缓解

| 风险 | 缓解措施 |
|------|----------|
| 会话 ID 冲突 | 使用明确的复合键格式，群聊用 channel，私聊用 userID |
| 内存泄漏（会话无限增长） | 自动清理机制，可配置过期时间 |
| 并发访问竞态 | sync.RWMutex 保护，参考 Router 实现 |
| 清理 goroutine 泄漏 | 使用 context 控制，提供 Stop() 方法 |

## 所属阶段

**阶段 1: 核心消息系统** - 会话管理是核心消息系统的关键组件。
