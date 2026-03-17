# 多项目切换 - 技术设计

## Context

### 当前架构

```
App
├── agent.Agent (单个)
├── core.Router
│   └── SessionManager (单个)
├── feishu.Adapter (单个)
└── command.Executor
```

当前 `App` 持有单个 `Agent` 实例，Agent 在创建时绑定 `WorkingDir`，无法动态更改。`SessionManager` 是全局共享的，未按项目隔离。

### 约束

1. **Agent 绑定 WorkingDir**：Claude Code 子进程在启动时通过 `--cwd` 参数绑定工作目录，运行时无法更改
2. **单一飞书机器人**：一个飞书应用只能创建一个机器人，所有项目共享同一个飞书连接
3. **切换延迟可接受**：2-3 秒的切换延迟在可接受范围内

## Goals / Non-Goals

**Goals:**

1. 支持运行时切换活跃项目
2. 每个项目有独立的 Agent 和 SessionManager
3. 提供 `/project` 命令管理项目切换
4. 回复消息显示项目名前缀
5. 支持切换时保留或清除会话

**Non-Goals:**

1. **不实现** Agent 预热缓存（后续优化）
2. **不实现** 并发多项目（多 Agent 同时活跃）
3. **不实现** 通道绑定（群聊自动路由到特定项目）
4. **不实现** 项目级飞书机器人（共享单一机器人）

## Decisions

### Decision 1: ProjectRouter 架构

**选择**：引入 `ProjectRouter` 管理多项目，每个项目持有独立的 Agent + SessionManager

```
App
├── ProjectRouter
│   ├── projects: map[string]*Project
│   │   ├── "frontend" → Project{Agent, SessionManager}
│   │   ├── "backend" → Project{Agent, SessionManager}
│   │   └── "devops" → Project{Agent, SessionManager}
│   └── active: *Project (当前活跃)
├── feishu.Adapter (共享)
└── command.Executor
```

**备选方案**：

| 方案 | 优点 | 缺点 |
|------|------|------|
| A. ProjectRouter | 职责清晰，易测试 | 需要新增类型 |
| B. App 直接管理 map | 代码少 | App 职责过重 |
| C. Agent 支持动态 WorkingDir | 无需架构变更 | Claude Code 不支持 |

**选择 A**，因为职责分离更清晰，便于测试和后续扩展。

### Decision 2: Agent 生命周期策略

**选择**：懒加载 + 立即切换

- Agent 在项目首次成为活跃项目时创建
- 切换项目时：Stop 旧 Agent → Start 新 Agent
- 非活跃项目的 Agent 可以被销毁（节省资源）

**备选方案**：

| 方案 | 切换延迟 | 资源占用 | 复杂度 |
|------|----------|----------|--------|
| A. 立即切换 | 2-3s | 低 | 低 |
| B. 预热缓存 | <100ms | 高 | 中 |
| C. 混合（缓存 N 个） | 2-3s / <100ms | 中 | 高 |

**选择 A**，因为：
- 实现简单
- 资源占用低
- 2-3 秒延迟用户可接受
- 后续可优化为方案 C

### Decision 3: 会话隔离策略

**选择**：每个 Project 独立 SessionManager

```go
type Project struct {
    Name       string
    Config     *ProjectConfig
    agent      agent.Agent        // 懒加载
    sessions   *SessionManager    // 独立
}
```

SessionID 格式保持不变（`platform:type:identifier`），隔离通过 Project 边界实现。

**备选方案**：

| 方案 | 隔离方式 | 优点 | 缺点 |
|------|----------|------|------|
| A. 独立 SessionManager | Project 边界 | 简单，隔离彻底 | 内存略高 |
| B. 共享 SessionManager + 前缀 | SessionID 加 project 前缀 | 内存低 | SessionID 格式变更 |

**选择 A**，因为隔离更彻底，避免 SessionID 格式变更带来的兼容性问题。

### Decision 4: 命令解析增强

**选择**：扩展 `Command` 结构体支持 Flags

```go
type Command struct {
    Name  string
    Args  []string
    Flags map[string]string  // 新增
}

// 解析 /project backend --keep
// Command{Name: "project", Args: ["backend"], Flags: {"keep": "true"}}
```

**解析规则**：
- `--flag value` → Flags["flag"] = "value"
- `--flag` → Flags["flag"] = "true"
- `-k` → Flags["k"] = "true" (短标志)

### Decision 5: 回复格式

**选择**：在 ReplySender 层添加项目名前缀

```go
func (s *replySender) SendReply(ctx context.Context, content string) error {
    prefix := fmt.Sprintf("[%s] ", s.projectName)
    return s.adapter.SendReply(ctx, s.channelID, prefix+content)
}
```

**格式示例**：
- 思考：`[frontend] 🤔 正在思考...`
- 结果：`[frontend] 已完成修改...`
- 错误：`[frontend] ❌ 处理失败...`

## Risks / Trade-offs

### Risk 1: 切换期间消息丢失

**风险**：切换过程中（2-3 秒）收到的新消息可能无法处理

**缓解**：
- 切换开始时发送"正在切换..."状态
- Feishu 消息有 ACK 机制，未处理的消息会重试
- 切换完成后正常处理积压消息

### Risk 2: Agent 进程泄漏

**风险**：切换失败或异常情况下，旧 Agent 进程可能未正确终止

**缓解**：
- 使用 `context.Context` 控制 Agent 生命周期
- Stop 时设置超时，超时后强制 kill
- 添加进程健康检查和清理机制

### Risk 3: 会话状态不一致

**风险**：`--keep` 切换时，Agent 重启后会话可能不完整

**缓解**：
- 明确告知用户 `--keep` 是"尽力保留"
- Agent 重启时使用 Claude Code 的 session resume 机制
- 会话恢复失败时提示用户

## 接口定义

### ProjectRouter

```go
// internal/core/project.go

// Project 代表一个项目实例
type Project struct {
    Name       string
    Config     *ProjectConfig
    agent      agent.Agent
    sessions   *SessionManager
    agentMu    sync.RWMutex
    status     ProjectStatus
}

type ProjectStatus string

const (
    ProjectStatusIdle     ProjectStatus = "idle"
    ProjectStatusActive   ProjectStatus = "active"
    ProjectStatusSwitching ProjectStatus = "switching"
)

// ProjectRouter 管理多项目切换
type ProjectRouter struct {
    projects   map[string]*Project
    active     *Project
    activeName string
    mu         sync.RWMutex

    // 依赖注入
    agentFactory AgentFactory
}

type AgentFactory func(config *ProjectConfig) (agent.Agent, error)

// 核心方法
func NewProjectRouter(configs []ProjectConfig, factory AgentFactory) (*ProjectRouter, error)
func (r *ProjectRouter) SwitchProject(ctx context.Context, name string, keepSession bool) error
func (r *ProjectRouter) ActiveProject() *Project
func (r *ProjectRouter) GetProject(name string) (*Project, bool)
func (r *ProjectRouter) ListProjects() []ProjectInfo
```

### Project

```go
// Project 方法
func (p *Project) GetOrCreateAgent(ctx context.Context) (agent.Agent, error)
func (p *Project) Sessions() *SessionManager
func (p *Project) ClearSessions()
func (p *Project) Status() ProjectStatus
```

### 命令处理

```go
// internal/core/command/handlers.go

func (e *Executor) handleProject(ctx context.Context, cmd Command, msg *core.Message) CommandResult {
    // /project           → 显示当前项目
    // /project <name>    → 切换项目（清除会话）
    // /project <name> -k → 切换项目（保留会话）
}
```

## 状态流转

```
┌─────────────────────────────────────────────────────────────────┐
│                     项目切换状态流转                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   用户发送 /project backend                                      │
│            │                                                    │
│            ▼                                                    │
│   ┌─────────────────┐                                          │
│   │ 验证项目存在    │─── 项目不存在 ──▶ 返回错误 + 项目列表      │
│   └────────┬────────┘                                          │
│            │ 存在                                               │
│            ▼                                                    │
│   ┌─────────────────┐                                          │
│   │ 设置状态:       │                                          │
│   │ Switching       │                                          │
│   └────────┬────────┘                                          │
│            │                                                    │
│            ▼                                                    │
│   ┌─────────────────┐                                          │
│   │ 发送"正在切换"  │                                          │
│   │ 消息            │                                          │
│   └────────┬────────┘                                          │
│            │                                                    │
│            ▼                                                    │
│   ┌─────────────────┐                                          │
│   │ Stop 旧 Agent   │                                          │
│   └────────┬────────┘                                          │
│            │                                                    │
│            ▼                                                    │
│   ┌─────────────────┐     keepSession=false                    │
│   │ 清除旧会话?     │──────────────────────▶ 清除 SessionManager │
│   └────────┬────────┘                                          │
│            │                                                    │
│            ▼                                                    │
│   ┌─────────────────┐                                          │
│   │ 创建/获取新     │                                          │
│   │ Project         │                                          │
│   └────────┬────────┘                                          │
│            │                                                    │
│            ▼                                                    │
│   ┌─────────────────┐                                          │
│   │ GetOrCreateAgent│                                          │
│   └────────┬────────┘                                          │
│            │                                                    │
│            ▼                                                    │
│   ┌─────────────────┐                                          │
│   │ Start 新 Agent  │─── 失败 ──▶ 回退到旧项目 + 错误提示       │
│   └────────┬────────┘                                          │
│            │ 成功                                               │
│            ▼                                                    │
│   ┌─────────────────┐                                          │
│   │ 更新 active     │                                          │
│   │ 设置状态: Active│                                          │
│   └────────┬────────┘                                          │
│            │                                                    │
│            ▼                                                    │
│   返回成功 + 项目信息                                            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## 错误处理

| 错误场景 | 处理方式 |
|----------|----------|
| 项目不存在 | 返回错误 + 可用项目列表 |
| Agent 启动失败 | 回退到原项目，返回错误详情 |
| Agent 停止超时 | 强制 kill，记录日志 |
| 切换过程中收到消息 | 等待切换完成后处理 |

## 测试策略

### 单元测试

- `ProjectRouter` 切换逻辑
- `Project` Agent 懒加载
- Command 解析 `--keep` 标志
- ReplySender 前缀添加

### 集成测试

- 完整切换流程（mock Agent）
- 切换失败回滚
- 会话隔离验证

### 边界测试

- 切换到当前项目（无操作）
- 切换到不存在的项目
- Agent 启动超时
- 并发切换请求
