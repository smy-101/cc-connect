# project-switching Specification

## Purpose

多项目切换能力允许用户在一个 cc-connect 进程中管理多个项目，通过命令在运行时切换活跃项目。每个项目有独立的工作目录、Agent 实例和会话管理器。

## Requirements

### Requirement: 项目列表显示

系统 SHALL 支持显示当前项目和可用项目列表。

#### Scenario: 显示当前项目
- **WHEN** 用户发送 `/project`（无参数）
- **THEN** 系统 SHALL 返回当前活跃项目名称
- **AND** 系统 SHALL 返回当前项目的工作目录
- **AND** 系统 SHALL 标记当前项目为"(当前)"

#### Scenario: 显示可用项目列表
- **WHEN** 用户发送 `/project`（无参数）
- **THEN** 系统 SHALL 返回所有配置的项目列表
- **AND** 每个项目 SHALL 显示名称和工作目录

---

### Requirement: 项目切换

系统 SHALL 支持切换到指定项目。

#### Scenario: 切换到存在的项目
- **WHEN** 用户发送 `/project backend`
- **THEN** 系统 SHALL 停止当前项目的 Agent
- **AND** 系统 SHALL 创建并启动目标项目的 Agent
- **AND** 系统 SHALL 更新活跃项目为 "backend"
- **AND** 系统 SHALL 返回切换成功消息

#### Scenario: 切换到不存在的项目
- **WHEN** 用户发送 `/project unknown`
- **THEN** 系统 SHALL 返回错误消息
- **AND** 系统 SHALL 返回可用项目列表
- **AND** 活跃项目 SHALL 保持不变

#### Scenario: 切换到当前项目
- **WHEN** 用户发送 `/project frontend` 且当前已是 "frontend"
- **THEN** 系统 SHALL 返回提示消息"已是当前项目"
- **AND** 系统 SHALL NOT 重新创建 Agent

---

### Requirement: 会话控制

系统 SHALL 支持切换时控制会话行为。

#### Scenario: 默认切换清除会话
- **WHEN** 用户发送 `/project backend`（无 --keep 标志）
- **THEN** 系统 SHALL 清除旧项目的会话
- **AND** 新项目对话 SHALL 无历史上下文

#### Scenario: 保留会话切换
- **WHEN** 用户发送 `/project backend --keep` 或 `/project backend -k`
- **THEN** 系统 SHALL 保留旧项目的会话
- **AND** 切换回该项目时 SHALL 能继续之前的对话

---

### Requirement: 项目名前缀显示

系统 SHALL 在所有回复消息中显示项目名前缀。

#### Scenario: 思考状态显示项目名
- **WHEN** Agent 开始处理消息
- **THEN** 系统 SHALL 发送 `[项目名] 🤔 正在思考...`

#### Scenario: 最终回复显示项目名
- **WHEN** Agent 返回回复
- **THEN** 系统 SHALL 在回复内容前添加 `[项目名] ` 前缀

#### Scenario: 命令结果显示项目名
- **WHEN** 命令执行完成
- **THEN** 系统 SHALL 在结果消息前添加 `[项目名] ` 前缀

---

### Requirement: Agent 生命周期

系统 SHALL 按需管理 Agent 的创建和销毁。

#### Scenario: 懒加载 Agent
- **WHEN** 项目首次被切换为活跃项目
- **THEN** 系统 SHALL 创建该项目的 Agent 实例
- **AND** 系统 SHALL 启动 Agent

#### Scenario: 切换时停止旧 Agent
- **WHEN** 切换到新项目
- **THEN** 系统 SHALL 停止旧项目的 Agent
- **AND** 系统 SHALL 释放 Agent 资源

#### Scenario: Agent 启动失败回滚
- **WHEN** 新项目 Agent 启动失败
- **THEN** 系统 SHALL 回退到原活跃项目
- **AND** 系统 SHALL 返回错误详情
- **AND** 系统 SHALL 尝试恢复原 Agent（如可用）

---

### Requirement: 项目隔离

系统 SHALL 确保项目之间的隔离。

#### Scenario: Agent 工作目录隔离
- **WHEN** 项目 A 的工作目录为 `/path/to/a`
- **THEN** 项目 A 的 Agent SHALL 只能操作 `/path/to/a` 目录

#### Scenario: 会话隔离
- **WHEN** 项目 A 有活跃会话
- **AND** 切换到项目 B
- **THEN** 项目 A 的会话 SHALL 独立于项目 B 的会话
- **AND** 项目 B 的会话操作 SHALL 不影响项目 A

---

### Requirement: 切换性能

系统 SHALL 在合理时间内完成项目切换。

#### Scenario: 切换延迟
- **WHEN** 执行项目切换
- **THEN** 切换 SHALL 在 5 秒内完成（正常情况）

#### Scenario: 切换状态提示
- **WHEN** 开始切换项目
- **THEN** 系统 SHALL 立即发送"正在切换..."提示
- **AND** 提示 SHALL 包含目标项目名
