## ADDED Requirements

### Requirement: UserQuestion 结构定义

系统必须定义 `UserQuestion` 结构来表示 AskUserQuestion 工具的问题。

- 必须包含 `Question` 字段（问题文本）
- 必须包含 `Header` 字段（简短标签）
- 必须包含 `Options` 字段（选项列表）
- 必须包含 `MultiSelect` 字段（是否多选）
- 每个选项必须包含 `Label` 和 `Description`

#### Scenario: 解析 AskUserQuestion 输入

- **WHEN** Claude 调用 AskUserQuestion 工具，输入为：
  ```json
  {
    "questions": [{
      "question": "您希望使用哪种数据库？",
      "header": "数据库",
      "options": [
        {"label": "PostgreSQL", "description": "功能丰富的关系型数据库"},
        {"label": "MySQL", "description": "广泛使用的关系型数据库"},
        {"label": "SQLite", "description": "轻量级嵌入式数据库"}
      ],
      "multiSelect": false
    }]
  }
  ```
- **THEN** 系统解析为 `UserQuestion` 结构
- **THEN** `Options` 包含 3 个选项

### Requirement: AskUserQuestion 事件

Session 必须在收到 AskUserQuestion 工具调用时发送特殊事件。

- `Event.Type` 必须为 `EventPermissionRequest`
- `Event.ToolName` 必须为 "AskUserQuestion"
- `Event.Questions` 必须包含解析后的问题列表
- `Event.RequestID` 必须用于关联响应

#### Scenario: 发送 AskUserQuestion 事件

- **WHEN** 收到 `control_request`，tool_name 为 "AskUserQuestion"
- **THEN** 发送 `Event{Type: EventPermissionRequest, ToolName: "AskUserQuestion", Questions: [...], RequestID: "req123"}`

### Requirement: 问题卡片生成

Router 必须根据 `UserQuestion` 生成交互式卡片。

- 单选问题必须显示为按钮组
- 多选问题必须显示为多选按钮
- 必须显示问题描述和选项说明
- 必须支持文本回复作为答案

#### Scenario: 单选问题卡片

- **GIVEN** `UserQuestion{Question: "选择数据库", MultiSelect: false, Options: [PostgreSQL, MySQL, SQLite]}`
- **WHEN** 生成卡片
- **THEN** 卡片包含 3 个水平排列的按钮
- **THEN** 按钮值格式为 `ans:req123:PostgreSQL`

#### Scenario: 多选问题卡片

- **GIVEN** `UserQuestion{Question: "选择功能", MultiSelect: true, Options: [A, B, C]}`
- **WHEN** 生成卡片
- **THEN** 卡片提示"可选择多个"
- **THEN** 需要用户发送多条消息选择，或发送 `/answer req123 A,B` 格式

#### Scenario: 文本答案支持

- **GIVEN** AskUserQuestion 无选项（开放式问题）
- **WHEN** 生成卡片
- **THEN** 卡片提示"请直接回复您的答案"
- **THEN** 用户文本消息作为答案

### Requirement: 答案收集与响应

Router 必须收集用户答案并通过 `RespondPermission` 返回。

- 单选：用户选择的选项 label 作为答案
- 多选：合并所有选择的 label，逗号分隔
- 文本：用户输入的文本作为答案
- 答案必须通过 `updatedInput` 字段返回给 Claude

#### Scenario: 单选答案响应

- **GIVEN** pending AskUserQuestion 请求 "req123"
- **WHEN** 用户点击 "PostgreSQL" 按钮
- **THEN** 调用 `session.RespondPermission("req123", {Behavior: "allow", UpdatedInput: {"answers": ["PostgreSQL"]}})`

#### Scenario: 多选答案响应

- **GIVEN** pending 多选 AskUserQuestion 请求 "req123"
- **WHEN** 用户发送 `/answer req123 A,B`
- **THEN** 调用 `session.RespondPermission("req123", {Behavior: "allow", UpdatedInput: {"answers": ["A", "B"]}})`

#### Scenario: 文本答案响应

- **GIVEN** pending 开放式 AskUserQuestion 请求 "req123"
- **WHEN** 用户回复 "我想要 PostgreSQL 因为它功能丰富"
- **THEN** 调用 `session.RespondPermission("req123", {Behavior: "allow", UpdatedInput: {"answers": ["我想要 PostgreSQL 因为它功能丰富"]}})`

### Requirement: 命令接口

系统必须提供 `/allow`、`/deny`、`/answer` 命令供用户响应。

- `/allow [requestID]` 必须批准权限请求
- `/deny [requestID] [message]` 必须拒绝权限请求
- `/answer <requestID> <answer>` 必须回答 AskUserQuestion
- 无 requestID 时必须使用当前 pending 请求

#### Scenario: 使用 /allow 命令

- **GIVEN** 存在 pending 权限请求 "req123"
- **WHEN** 用户发送 `/allow req123`
- **THEN** 系统批准请求
- **THEN** 回复 "✅ 已批准 Bash 执行"

#### Scenario: 使用 /deny 命令

- **GIVEN** 存在 pending 权限请求 "req123"
- **WHEN** 用户发送 `/deny req123 这个命令太危险`
- **THEN** 系统拒绝请求，附带消息 "这个命令太危险"
- **THEN** 回复 "❌ 已拒绝 Bash 执行"

#### Scenario: 使用 /answer 命令

- **GIVEN** 存在 pending AskUserQuestion 请求 "req123"
- **WHEN** 用户发送 `/answer req123 PostgreSQL`
- **THEN** 系统记录答案并继续
- **THEN** 回复 "✅ 已记录您的选择：PostgreSQL"

#### Scenario: 无 requestID 时使用当前请求

- **GIVEN** 存在 pending 权限请求
- **WHEN** 用户发送 `/allow`（无 requestID）
- **THEN** 系统使用当前 pending 请求的 ID

#### Scenario: 无 pending 请求时命令失败

- **GIVEN** 无 pending 权限请求
- **WHEN** 用户发送 `/allow`
- **THEN** 回复错误 "当前没有待处理的权限请求"
