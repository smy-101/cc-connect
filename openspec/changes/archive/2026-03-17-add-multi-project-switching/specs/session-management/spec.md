# session-management 规格 (Delta)

## Purpose

会话管理模块负责跟踪和管理用户与 AI 代理之间的对话上下文。本次变更将会话按项目隔离，每个项目有独立的 SessionManager。

## ADDED Requirements

### Requirement: 项目级会话管理器

系统 SHALL 为每个项目提供独立的 SessionManager。

#### Scenario: 项目创建时初始化 SessionManager
- **WHEN** 创建新项目实例
- **THEN** 系统 SHALL 为该项目创建独立的 SessionManager
- **AND** SessionManager SHALL 使用项目级配置

#### Scenario: 会话按项目隔离
- **WHEN** 项目 A 有会话 "feishu:channel:oc_xxx"
- **AND** 项目 B 有会话 "feishu:channel:oc_xxx"
- **THEN** 两个会话 SHALL 相互独立
- **AND** 修改项目 A 的会话 SHALL 不影响项目 B

### Requirement: 会话清除

系统 SHALL 支持清除项目的所有会话。

#### Scenario: 清除项目会话
- **WHEN** 调用 `project.ClearSessions()`
- **THEN** 系统 SHALL 销毁该项目的所有会话
- **AND** 其他项目的会话 SHALL 不受影响

#### Scenario: 切换项目时清除会话
- **WHEN** 用户切换项目且未指定 --keep
- **THEN** 系统 SHALL 清除旧项目的所有会话

## MODIFIED Requirements

### Requirement: 会话创建与获取

系统 SHALL 支持会话的自动创建和按需获取。

#### Scenario: 获取不存在的会话时自动创建
- **WHEN** 调用 `GetOrCreate(id)` 且会话不存在
- **THEN** 系统 创建新会话，状态为 `active`，返回该会话
- **AND** 会话在**当前项目**的 SessionManager 中创建

#### Scenario: 获取已存在的会话
- **WHEN** 调用 `GetOrCreate(id)` 且会话已存在
- **THEN** 系统返回现有会话，不创建新会话
- **AND** 会话从**当前项目**的 SessionManager 获取

#### Scenario: 获取会话返回副本
- **WHEN** 调用 `Get(id)` 成功获取会话
- **THEN** 系统返回会话的副本，修改副本不影响内部状态

#### Scenario: 获取不存在的会话返回 false
- **WHEN** 调用 `Get(id)` 且会话不存在
- **THEN** 系统返回 `nil, false`
