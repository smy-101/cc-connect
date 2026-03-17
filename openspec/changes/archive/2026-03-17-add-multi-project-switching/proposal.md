# 多项目切换支持

## Why

当前 cc-connect 只能同时操作一个项目的工作目录。用户（个人使用场景）需要同时管理多个项目，且需要频繁切换。由于飞书一个应用只能创建一个机器人，必须通过统一入口管理多个项目。

**解决问题**：让用户可以在一个 cc-connect 进程中管理多个项目，通过命令快速切换。

## What Changes

### 新增功能

- **项目切换命令** `/project`
  - `/project` - 显示当前项目和可用项目列表
  - `/project <name>` - 切换到指定项目（默认清除会话）
  - `/project <name> --keep` / `-k` - 切换项目但保留会话

- **项目名前缀显示**
  - 所有回复消息带 `[项目名]` 前缀，让用户清楚知道当前在哪个项目

- **项目隔离**
  - 每个项目有独立的 Agent 实例（绑定不同 WorkingDir）
  - 每个项目有独立的会话管理器
  - 切换项目时按需创建/销毁 Agent

### 修改行为

- **App 启动**：使用 `ProjectRouter` 管理多项目，而非单个 Agent
- **消息处理**：从 `ProjectRouter` 获取当前活跃项目的 Agent
- **回复格式**：增加项目名前缀

## Capabilities

### New Capabilities

- `project-switching`: 多项目切换能力，支持在运行时切换活跃项目，管理项目级 Agent 和会话

### Modified Capabilities

- `command-system`: 新增 `/project` 命令
- `session-management`: 会话按项目隔离，SessionID 格式变更

## Impact

### 代码变更

| 文件/目录 | 变更类型 | 说明 |
|----------|---------|------|
| `internal/core/project.go` | 新增 | Project, ProjectRouter 类型定义 |
| `internal/core/project_test.go` | 新增 | 项目管理单元测试 |
| `internal/app/app.go` | 修改 | 使用 ProjectRouter 替代单个 Agent |
| `internal/app/handlers.go` | 修改 | 回复消息增加项目名前缀 |
| `internal/core/command/executor.go` | 修改 | 添加 project 命令路由 |
| `internal/core/command/handlers.go` | 修改 | 添加 handleProject 函数 |
| `internal/core/command/parser.go` | 修改 | 支持 --keep/-k 标志解析 |

### API 变更

无外部 API 变更，仅内部架构调整。

### 依赖变更

无新增外部依赖。

## 验收标准

### 功能验收

1. **项目列表**
   - 输入 `/project` 显示当前项目名、工作目录、可用项目列表
   - 当前项目有明确标识

2. **项目切换**
   - `/project <name>` 成功切换到目标项目
   - 切换后 Agent 工作目录正确
   - 切换耗时 < 5 秒（可接受范围）

3. **会话控制**
   - 默认切换清除会话，新对话无历史上下文
   - `--keep` 切换保留会话，可继续之前的对话

4. **项目名前缀**
   - 思考状态消息：`[project] 🤔 正在思考...`
   - 最终回复消息：`[project] 回复内容...`
   - 命令执行结果：`[project] ✅ 命令结果`

5. **错误处理**
   - 切换到不存在的项目：提示可用项目列表
   - Agent 启动失败：回退到原项目，提示错误

### 测试覆盖

- `project.go` 测试覆盖率 > 85%
- 集成测试：完整切换流程验证
- 边界测试：切换过程中消息处理

## 风险与缓解

### 风险 1：切换延迟影响用户体验

- **风险**：切换需要启动新 Agent（2-3 秒），期间无法处理消息
- **缓解**：切换时发送"正在切换..."提示；后续可考虑预热缓存

### 风险 2：会话数据隔离不彻底

- **风险**：切换项目时会话可能混淆
- **缓解**：每个项目独立 SessionManager，严格按 project name 隔离

### 风险 3：Agent 资源泄漏

- **风险**：频繁切换可能导致 Agent 进程未正确清理
- **缓解**：ProjectRouter 管理 Agent 生命周期，Stop 时确保进程终止

## 阶段归属

本变更属于 **阶段 6：高级功能** 的子集（多项目管理），提前实现以满足当前用户需求。
