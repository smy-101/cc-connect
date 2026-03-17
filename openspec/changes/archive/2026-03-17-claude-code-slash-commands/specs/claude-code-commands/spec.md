# claude-code-commands Specification

## Purpose

定义 Claude Code 斜杠命令在 cc-connect 中的使用方式，允许用户通过双斜杠前缀在聊天中调用 Claude Code 的原生命令。

## ADDED Requirements

### Requirement: Claude Code 命令识别

系统 SHALL 能够识别以 `//`（双斜杠）开头的文本消息为 Claude Code 命令。

#### Scenario: 有效 Claude Code 命令检测
- **WHEN** 消息内容为 `//cost`
- **THEN** 系统 SHALL 识别该消息为 Claude Code 命令
- **AND** 系统 SHALL 将消息内容转换为 `/cost`

#### Scenario: 带参数的 Claude Code 命令检测
- **WHEN** 消息内容为 `//compact focus on auth`
- **THEN** 系统 SHALL 识别该消息为 Claude Code 命令
- **AND** 系统 SHALL 将消息内容转换为 `/compact focus on auth`

#### Scenario: 单斜杠不是 Claude Code 命令
- **WHEN** 消息内容为 `/mode`
- **THEN** 系统 SHALL NOT 识别该消息为 Claude Code 命令
- **AND** 系统 SHALL 按内置命令处理

#### Scenario: 三斜杠处理
- **WHEN** 消息内容为 `///mode`
- **THEN** 系统 SHALL NOT 识别该消息为 Claude Code 命令
- **AND** 系统 SHALL 按内置命令处理（变为 `/mode`）

#### Scenario: 只有双斜杠
- **WHEN** 消息内容为 `//`
- **THEN** 系统 SHALL NOT 识别该消息为 Claude Code 命令

#### Scenario: 空字符串
- **WHEN** 消息内容为空字符串
- **THEN** 系统 SHALL NOT 识别该消息为 Claude Code 命令

---

### Requirement: Claude Code 命令透传

系统 SHALL 将 Claude Code 命令转换为单斜杠格式后透传给 Agent。

#### Scenario: 命令透传到 Agent
- **WHEN** 用户发送 `//cost`
- **THEN** 系统 SHALL 将 `/cost` 发送给 Agent
- **AND** 消息类型 SHALL 保持为 `MessageTypeText`

#### Scenario: 命令响应返回
- **WHEN** Claude Code 执行命令完成
- **THEN** 系统 SHALL 将响应通过原消息渠道返回给用户

#### Scenario: 未知命令处理
- **WHEN** 用户发送 `//unknown-command`
- **THEN** 系统 SHALL 透传给 Agent
- **AND** Agent 返回的错误信息 SHALL 显示给用户

---

### Requirement: 帮助信息包含 Claude Code 命令

系统 SHALL 在 `/help` 输出中包含 Claude Code 命令的使用说明。

#### Scenario: 帮助信息显示 Claude Code 命令
- **WHEN** 用户发送 `/help`
- **THEN** 系统 SHALL 返回包含 "Claude Code 命令 (双斜杠 //)" 的帮助信息
- **AND** 帮助信息 SHALL 列出常用的 Claude Code 命令
- **AND** 帮助信息 SHALL 说明使用 `//` 前缀

---

### Requirement: 命令路由区分

系统 SHALL 正确区分 cc-connect 内置命令和 Claude Code 命令。

#### Scenario: 内置命令路由
- **WHEN** 用户发送 `/mode yolo`
- **THEN** 系统 SHALL 将消息路由到 cc-connect Executor
- **AND** 系统 SHALL 执行内置的 mode 命令逻辑

#### Scenario: Claude Code 命令路由
- **WHEN** 用户发送 `//cost`
- **THEN** 系统 SHALL 将消息路由到 Agent
- **AND** 系统 SHALL NOT 调用 cc-connect Executor

#### Scenario: 同名命令区分
- **WHEN** Claude Code 和 cc-connect 有同名命令（如 help）
- **THEN** `/help` SHALL 触发 cc-connect 的帮助
- **AND** `//help` SHALL 触发 Claude Code 的帮助（如果存在）
