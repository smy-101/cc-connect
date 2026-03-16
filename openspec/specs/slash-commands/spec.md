# slash-commands Specification

## Purpose
TBD - created by archiving change add-slash-commands. Update Purpose after archive.
## Requirements
### Requirement: 命令检测

系统 SHALL 能够识别以 `/` 开头的文本消息为命令。

#### Scenario: 有效命令检测
- **WHEN** 消息内容为 `/mode`
- **THEN** 系统 SHALL 识别该消息为命令

#### Scenario: 带参数的命令检测
- **WHEN** 消息内容为 `/mode yolo`
- **THEN** 系统 SHALL 识别该消息为命令

#### Scenario: 非命令文本
- **WHEN** 消息内容为 `hello world`
- **THEN** 系统 SHALL NOT 识别该消息为命令

#### Scenario: 前导空格不识别为命令
- **WHEN** 消息内容为 ` /mode`（含前导空格）
- **THEN** 系统 SHALL NOT 识别该消息为命令

#### Scenario: 空字符串
- **WHEN** 消息内容为空字符串
- **THEN** 系统 SHALL NOT 识别该消息为命令

---

### Requirement: 命令解析

系统 SHALL 能够解析命令字符串，提取命令名和参数列表。

#### Scenario: 无参数命令解析
- **WHEN** 解析 `/help`
- **THEN** 系统 SHALL 返回 Command{Name: "help", Args: []}

#### Scenario: 单参数命令解析
- **WHEN** 解析 `/mode yolo`
- **THEN** 系统 SHALL 返回 Command{Name: "mode", Args: ["yolo"]}

#### Scenario: 多参数命令解析
- **WHEN** 解析 `/new my-session`
- **THEN** 系统 SHALL 返回 Command{Name: "new", Args: ["my-session"]}

#### Scenario: 多空格参数分割
- **WHEN** 解析 `/mode   yolo`（多个空格）
- **THEN** 系统 SHALL 返回 Command{Name: "mode", Args: ["yolo"]}

---

### Requirement: /mode 命令

系统 SHALL 支持 `/mode` 命令切换 Agent 权限模式。

#### Scenario: 切换到 yolo 模式
- **WHEN** 用户发送 `/mode yolo`
- **THEN** 系统 SHALL 调用 Agent.SetPermissionMode(bypassPermissions)
- **AND** 系统 SHALL 返回成功消息

#### Scenario: 切换到 edit 模式
- **WHEN** 用户发送 `/mode edit`
- **THEN** 系统 SHALL 调用 Agent.SetPermissionMode(acceptEdits)
- **AND** 系统 SHALL 返回成功消息

#### Scenario: 切换到 plan 模式
- **WHEN** 用户发送 `/mode plan`
- **THEN** 系统 SHALL 调用 Agent.SetPermissionMode(plan)
- **AND** 系统 SHALL 返回成功消息

#### Scenario: 切换到 default 模式
- **WHEN** 用户发送 `/mode default`
- **THEN** 系统 SHALL 调用 Agent.SetPermissionMode(default)
- **AND** 系统 SHALL 返回成功消息

#### Scenario: 无参数显示当前模式
- **WHEN** 用户发送 `/mode`（无参数）
- **THEN** 系统 SHALL 返回当前权限模式

#### Scenario: 无效模式名称
- **WHEN** 用户发送 `/mode invalid`
- **THEN** 系统 SHALL 返回错误消息，列出可用模式

---

### Requirement: /new 命令

系统 SHALL 支持 `/new` 命令创建新会话。

#### Scenario: 创建新会话（无名称）
- **WHEN** 用户发送 `/new`
- **THEN** 系统 SHALL 销毁当前会话（如存在）
- **AND** 系统 SHALL 创建新会话
- **AND** 系统 SHALL 返回成功消息

#### Scenario: 创建命名会话
- **WHEN** 用户发送 `/new my-project`
- **THEN** 系统 SHALL 创建名为 "my-project" 的新会话
- **AND** 系统 SHALL 返回包含会话名称的成功消息

---

### Requirement: /list 命令

系统 SHALL 支持 `/list` 命令列出所有活跃会话。

#### Scenario: 列出多个会话
- **WHEN** 存在多个会话
- **AND** 用户发送 `/list`
- **THEN** 系统 SHALL 返回所有会话的列表

#### Scenario: 无会话时列出
- **WHEN** 不存在任何会话
- **AND** 用户发送 `/list`
- **THEN** 系统 SHALL 返回空会话列表的消息

---

### Requirement: /help 命令

系统 SHALL 支持 `/help` 命令显示可用命令的帮助信息。

#### Scenario: 显示帮助
- **WHEN** 用户发送 `/help`
- **THEN** 系统 SHALL 返回所有可用命令的列表
- **AND** 列表 SHALL 包含 /mode、/new、/list、/help、/stop
- **AND** 每个命令 SHALL 有简要说明

---

### Requirement: /stop 命令

系统 SHALL 支持 `/stop` 命令停止 Agent。

#### Scenario: 停止运行中的 Agent
- **WHEN** Agent 状态为 running
- **AND** 用户发送 `/stop`
- **THEN** 系统 SHALL 调用 Agent.Stop()
- **AND** 系统 SHALL 返回成功消息

#### Scenario: Agent 未运行时停止
- **WHEN** Agent 状态为 idle 或 stopped
- **AND** 用户发送 `/stop`
- **THEN** 系统 SHALL 返回 "Agent 未运行" 消息

---

### Requirement: 未知命令处理

系统 SHALL 对未知命令返回友好的错误提示。

#### Scenario: 未知命令
- **WHEN** 用户发送 `/unknown`
- **THEN** 系统 SHALL 返回 "未知命令" 消息
- **AND** 消息 SHALL 建议用户使用 /help

---

### Requirement: 命令消息类型

系统 SHALL 将命令消息的类型设置为 `MessageTypeCommand`。

#### Scenario: 飞书命令消息类型
- **WHEN** 飞书收到 `/mode yolo` 文本消息
- **THEN** 系统 SHALL 将消息类型设置为 `MessageTypeCommand`
- **AND** 消息内容保持为 `/mode yolo`

---

### Requirement: 命令响应

系统 SHALL 通过原消息渠道返回命令执行结果。

#### Scenario: 命令响应返回
- **WHEN** 命令执行完成
- **THEN** 系统 SHALL 通过 Router 将响应发送回原平台
- **AND** 响应 SHALL 为文本格式

