# 多项目切换 - 实现任务

## 1. 核心类型定义 (internal/core/project.go)

**依赖**: 无
**验证**: `go test ./internal/core/... -v -run TestProject`

- [x] 1.1 编写 Project 类型测试：创建、状态管理、Agent 懒加载
- [x] 1.2 编写 ProjectRouter 类型测试：项目列表、切换逻辑、活跃项目获取
- [x] 1.3 实现 Project 结构体（Name, Config, agent, sessions, status）
- [x] 1.4 实现 ProjectRouter 结构体（projects map, active, agentFactory）
- [x] 1.5 实现 NewProjectRouter 构造函数
- [x] 1.6 实现 GetOrCreateAgent 懒加载方法
- [x] 1.7 实现 SwitchProject 切换方法（含 Stop 旧 Agent → Start 新 Agent）
- [x] 1.8 实现 ClearSessions 会话清除方法
- [x] 1.9 实现 ListProjects 项目列表方法
- [x] 1.10 运行测试确保覆盖率 > 85%

## 2. 命令解析增强 (internal/core/command/parser.go)

**依赖**: 无
**验证**: `go test ./internal/core/command/... -v -run TestParser`

- [x] 2.1 编写命令标志解析测试：`--keep`, `-k`, 无标志
- [x] 2.2 扩展 Command 结构体，添加 Flags map[string]string
- [x] 2.3 修改 Parse 函数，支持 `--flag` 和 `-f` 格式
- [x] 2.4 运行测试确保现有测试通过

## 3. /project 命令处理 (internal/core/command/)

**依赖**: 1, 2
**验证**: `go test ./internal/core/command/... -v -run TestProject`

- [x] 3.1 编写 handleProject 测试：无参数显示列表
- [x] 3.2 编写 handleProject 测试：切换到存在的项目
- [x] 3.3 编写 handleProject 测试：切换到不存在的项目
- [x] 3.4 编写 handleProject 测试：切换到当前项目
- [x] 3.5 编写 handleProject 测试：--keep 标志保留会话
- [x] 3.6 实现 handleProject 函数
- [x] 3.7 在 Executor.Execute 中添加 "project" case
- [x] 3.8 更新 handleHelp 函数，添加 /project 命令说明
- [x] 3.9 运行测试确保覆盖率 > 85%

## 4. App 重构 (internal/app/app.go)

**依赖**: 1
**验证**: `go test ./internal/app/... -v`

- [x] 4.1 编写 App 测试：使用 ProjectRouter 替代单个 Agent
- [x] 4.2 编写 App 测试：项目切换集成流程
- [x] 4.3 修改 App 结构体，使用 ProjectRouter 替代 agent.Agent
- [x] 4.4 修改 New 构造函数，初始化 ProjectRouter
- [x] 4.5 修改 Start 方法，启动 ProjectRouter 的活跃项目
- [x] 4.6 修改 Stop 方法，停止所有项目的 Agent
- [x] 4.7 添加 ProjectRouter 访问方法
- [x] 4.8 运行测试确保现有测试通过

## 5. 回复前缀支持 (internal/app/reply.go, handlers.go)

**依赖**: 4
**验证**: `go test ./internal/app/... -v -run TestReply`

- [x] 5.1 编写测试：回复消息包含项目名前缀
- [x] 5.2 修改 replySender，添加 projectName 字段
- [x] 5.3 修改 SendReply 方法，自动添加 `[项目名] ` 前缀
- [x] 5.4 修改 wrapHandler，传递项目名到 replySender
- [x] 5.5 运行测试确保前缀正确添加

## 6. 集成测试 (test/e2e/)

**依赖**: 1-5
**验证**: `go test ./test/e2e/... -v -tags=e2e`

- [x] 6.1 编写集成测试：完整切换流程（mock Agent）
- [x] 6.2 编写集成测试：切换失败回滚
- [x] 6.3 编写集成测试：会话隔离验证
- [x] 6.4 编写集成测试：项目名前缀显示
- [x] 6.5 运行所有集成测试

## 7. 更新与文档

**依赖**: 1-6
**验证**: `go test ./...`

- [x] 7.1 运行所有测试：`go test ./...`
- [x] 7.2 运行竞态检测：`go test ./... -race`
- [x] 7.3 检查测试覆盖率：`go test ./... -cover`
- [x] 7.4 更新 config.example.toml（如有需要）
- [x] 7.5 更新 CLAUDE.md（如有架构变更）

---

## 任务依赖关系

```
1. 核心类型定义 ─────┬──▶ 3. /project 命令处理 ──┐
                    │                           │
2. 命令解析增强 ────┘                           │
                                                ▼
4. App 重构 ──────────────────────────────▶ 6. 集成测试
        │                                       ▲
        ▼                                       │
5. 回复前缀支持 ────────────────────────────────┘
                                                │
                                                ▼
                                         7. 更新与文档
```

## 并行任务

以下任务可以并行进行：
- **1 和 2**：核心类型定义和命令解析增强互不依赖
- **3.1-3.5**：测试编写可以并行
- **6.1-6.4**：集成测试编写可以并行

## 预估工时

| 阶段 | 预估时间 |
|------|----------|
| 1. 核心类型定义 | 0.5 天 |
| 2. 命令解析增强 | 0.25 天 |
| 3. /project 命令处理 | 0.5 天 |
| 4. App 重构 | 0.5 天 |
| 5. 回复前缀支持 | 0.25 天 |
| 6. 集成测试 | 0.5 天 |
| 7. 更新与文档 | 0.25 天 |
| **总计** | **2.75 天** |
