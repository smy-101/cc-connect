# E2E Flow 规格说明

本规格定义端到端消息流程的行为，验证从飞书接收到 AI 响应的完整链路。

## ADDED Requirements

### Requirement: 端到端文本消息流程

系统 SHALL 正确处理从飞书接收到 AI 响应的完整文本消息流程。

#### Scenario: 完整文本消息流程
- **WHEN** 用户通过飞书发送文本消息 "Hello"
- **THEN** 系统收到消息并转换为统一消息格式
- **AND** 消息被路由到文本处理器
- **AND** Agent 收到处理请求
- **AND** 响应发送回飞书
- **AND** 用户收到响应

#### Scenario: 消息处理延迟要求
- **WHEN** 消息从飞书到达系统
- **THEN** 状态提示（"正在思考..."）在 500ms 内发送
- **AND** 状态提示发送不依赖 Agent 响应时间

### Requirement: 端到端命令消息流程

系统 SHALL 正确处理从飞书接收到命令执行的完整命令流程。

#### Scenario: /mode 命令流程
- **WHEN** 用户通过飞书发送 "/mode yolo"
- **THEN** 消息被识别为命令类型
- **AND** 命令被解析为 `{Name: "mode", Args: ["yolo"]}`
- **AND** Agent 权限模式切换到 `bypassPermissions`
- **AND** 响应包含 "已切换到 yolo 模式"

#### Scenario: /help 命令流程
- **WHEN** 用户通过飞书发送 "/help"
- **THEN** 响应包含可用命令列表
- **AND** 响应包含每个命令的简要说明

#### Scenario: /new 命令流程
- **WHEN** 用户通过飞书发送 "/new"
- **THEN** 当前会话被清除
- **AND** 创建新的会话
- **AND** 响应包含 "已创建新会话"

### Requirement: 错误处理端到端流程

系统 SHALL 在各环节错误时提供友好的用户反馈。

#### Scenario: Agent 不可用
- **WHEN** Agent 未启动或已崩溃
- **AND** 用户发送文本消息
- **THEN** 用户收到 "❌ Agent 不可用，请稍后重试。"

#### Scenario: Agent 响应超时
- **WHEN** Agent 处理时间超过超时限制
- **THEN** 用户收到 "⏱️ 请求超时，请简化问题或稍后重试。"
- **AND** Agent 处理被取消

#### Scenario: 飞书发送失败
- **WHEN** 响应无法发送到飞书
- **THEN** 系统记录错误日志
- **AND** 应用继续运行

### Requirement: 会话连续性

系统 SHALL 保持同一会话的消息上下文连续性。

#### Scenario: 同一会话的消息连续性
- **WHEN** 用户在同一聊天频道连续发送消息
- **THEN** 消息使用相同的 Session ID
- **AND** Agent 能访问之前的对话上下文

#### Scenario: 不同会话的隔离
- **WHEN** 用户在不同聊天频道发送消息
- **THEN** 消息使用不同的 Session ID
- **AND** Agent 的对话上下文相互隔离

### Requirement: 并发消息处理

系统 SHALL 能处理来自不同会话的并发消息。

#### Scenario: 并发消息处理
- **WHEN** 两个用户同时发送消息
- **THEN** 两个消息并行处理
- **AND** 响应发送给正确的用户

#### Scenario: 同一会话串行处理
- **WHEN** 同一用户连续发送多条消息
- **THEN** 消息按顺序处理
- **AND** 后续消息等待前一条完成

### Requirement: 优雅关闭流程

系统 SHALL 在关闭时正确处理进行中的请求。

#### Scenario: 优雅关闭等待处理完成
- **WHEN** 收到关闭信号
- **AND** 有消息正在处理中
- **THEN** 系统等待处理完成（最多 30 秒）
- **AND** 发送响应后关闭

#### Scenario: 优雅关闭超时强制退出
- **WHEN** 收到关闭信号
- **AND** 消息处理超过 30 秒未完成
- **THEN** 系统强制关闭
- **AND** 用户收到处理中断通知（如可能）

### Requirement: E2E 测试可验证性

系统 SHALL 提供 mock 组件支持端到端测试。

#### Scenario: Mock Agent 预设响应
- **WHEN** 使用 Mock Agent 配置响应 "测试响应"
- **AND** 发送消息 "测试"
- **THEN** 用户收到 "测试响应"

#### Scenario: Mock Agent 模拟延迟
- **WHEN** 使用 Mock Agent 配置 2 秒延迟
- **AND** 发送消息
- **THEN** 状态提示在 500ms 内发送
- **AND** 最终响应在 2 秒后发送

#### Scenario: Mock Agent 模拟错误
- **WHEN** 使用 Mock Agent 配置返回错误
- **AND** 发送消息
- **THEN** 用户收到错误提示信息

#### Scenario: Mock Feishu 事件注入
- **WHEN** 向 Mock Feishu 注入消息事件
- **THEN** 系统处理该事件
- **AND** 响应可通过 Mock Feishu 获取
