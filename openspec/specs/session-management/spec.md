# 会话管理规格

## Purpose

会话管理模块负责跟踪和管理用户与 AI 代理之间的对话上下文。它提供会话标识、状态存储、生命周期管理和自动清理功能，是连接消息路由、代理适配器和斜杠命令的核心纽带。

## ADDED Requirements

### Requirement: 会话标识派生

系统 SHALL 从消息结构中自动派生会话标识。

#### Scenario: 私聊消息派生会话 ID
- **WHEN** 消息的 `ChannelID` 为空且 `UserID` 为 "ou_xxx"
- **THEN** 系统派生会话 ID 为 `"feishu:user:ou_xxx"` 格式

#### Scenario: 群聊消息派生会话 ID
- **WHEN** 消息的 `ChannelID` 为 "oc_xxx"
- **THEN** 系统派生会话 ID 为 `"feishu:channel:oc_xxx"` 格式

#### Scenario: 优先使用群聊标识
- **WHEN** 消息同时包含 `UserID` 和 `ChannelID`
- **THEN** 系统优先使用 `ChannelID` 派生会话 ID（群聊优先）

### Requirement: 会话创建与获取

系统 SHALL 支持会话的自动创建和按需获取。

#### Scenario: 获取不存在的会话时自动创建
- **WHEN** 调用 `GetOrCreate(id)` 且会话不存在
- **THEN** 系统创建新会话，状态为 `active`，返回该会话

#### Scenario: 获取已存在的会话
- **WHEN** 调用 `GetOrCreate(id)` 且会话已存在
- **THEN** 系统返回现有会话，不创建新会话

#### Scenario: 获取会话返回副本
- **WHEN** 调用 `Get(id)` 成功获取会话
- **THEN** 系统返回会话的副本，修改副本不影响内部状态

#### Scenario: 获取不存在的会话返回 false
- **WHEN** 调用 `Get(id)` 且会话不存在
- **THEN** 系统返回 `nil, false`

### Requirement: 会话状态存储

系统 SHALL 存储会话的状态信息。

#### Scenario: 存储代理绑定
- **WHEN** 调用 `session.BindAgent("claudecode")`
- **THEN** 会话的 `AgentID` 字段被设置为 "claudecode"

#### Scenario: 存储权限模式
- **WHEN** 调用 `session.SetPermissionMode("yolo")`
- **THEN** 会话的 `PermissionMode` 字段被设置为 "yolo"

#### Scenario: 存储元数据
- **WHEN** 调用 `session.SetMetadata("key", "value")`
- **THEN** 会话的 `Metadata` map 中存储 "key" → "value" 映射

#### Scenario: 更新活跃时间
- **WHEN** 调用 `session.Touch()`
- **THEN** 会话的 `LastActiveAt` 字段更新为当前时间

### Requirement: 会话生命周期管理

系统 SHALL 管理会话的完整生命周期。

#### Scenario: 新会话状态为 active
- **WHEN** 创建新会话
- **THEN** 会话 `Status` 为 `active`，`CreatedAt` 和 `LastActiveAt` 设置为当前时间

#### Scenario: 手动归档会话
- **WHEN** 调用 `manager.Archive(id)` 且会话状态为 `active`
- **THEN** 会话 `Status` 变为 `archived`，`ArchivedAt` 设置为当前时间

#### Scenario: 归档不存在的会话
- **WHEN** 调用 `manager.Archive(id)` 且会话不存在
- **THEN** 系统返回错误 `ErrSessionNotFound`

#### Scenario: 手动销毁会话
- **WHEN** 调用 `manager.Destroy(id)` 且会话存在
- **THEN** 会话从管理器中移除，资源释放

#### Scenario: 状态转换不可逆
- **WHEN** 会话状态为 `archived`
- **THEN** 无法将状态转换回 `active`（调用 `Archive` 无效或返回错误）

### Requirement: 自动清理机制

系统 SHALL 自动清理过期会话以防止内存泄漏。

#### Scenario: 活跃会话超时自动归档
- **WHEN** 会话状态为 `active` 且 `LastActiveAt` 超过配置的 `ActiveTTL`（默认 30 分钟）
- **THEN** 清理器将会话状态变为 `archived`

#### Scenario: 归档会话超时自动销毁
- **WHEN** 会话状态为 `archived` 且 `ArchivedAt` 超过配置的 `ArchivedTTL`（默认 24 小时）
- **THEN** 清理器将会话从管理器中移除

#### Scenario: 清理器定时执行
- **WHEN** 调用 `manager.StartCleanup(ctx)` 且 context 未取消
- **THEN** 清理器每隔 `CleanupInterval`（默认 5 分钟）执行一次清理

#### Scenario: 清理器响应 context 取消
- **WHEN** 传入 `StartCleanup` 的 context 被取消
- **THEN** 清理 goroutine 正确退出，不泄漏

### Requirement: 并发安全

系统 SHALL 支持并发访问会话管理器。

#### Scenario: 并发获取会话
- **WHEN** 多个 goroutine 同时调用 `GetOrCreate` 获取同一会话
- **THEN** 所有调用正确完成，只创建一个会话，无竞态条件

#### Scenario: 并发读写会话
- **WHEN** 一个 goroutine 修改会话状态同时另一个 goroutine 读取会话
- **THEN** 读取操作获得一致的状态，无竞态条件（`go test -race` 通过）

#### Scenario: 并发清理和访问
- **WHEN** 清理 goroutine 销毁会话同时另一 goroutine 访问该会话
- **THEN** 操作正确完成，无竞态条件，无 panic

### Requirement: Message 结构扩展

系统 SHALL 扩展 Message 结构以支持群聊标识。

#### Scenario: Message 包含 ChannelID 字段
- **WHEN** 创建群聊消息
- **THEN** `Message` 结构包含可选的 `ChannelID` 字段

#### Scenario: ChannelID 可选
- **WHEN** 创建私聊消息且不设置 `ChannelID`
- **THEN** 消息仍然有效，序列化时不包含 `channel_id` 字段

#### Scenario: JSON 序列化兼容性
- **WHEN** 序列化包含 `ChannelID` 的消息为 JSON
- **THEN** 输出包含 `"channel_id": "oc_xxx"` 字段

#### Scenario: JSON 反序列化兼容性
- **WHEN** 反序列化包含 `channel_id` 字段的 JSON
- **THEN** `Message.ChannelID` 正确设置为对应值

### Requirement: Router 集成

系统 SHALL 在消息路由时自动管理会话上下文。

#### Scenario: 路由时自动获取会话
- **WHEN** 调用 `router.RouteWithSession(ctx, msg)` 且使用已注册的处理器
- **THEN** 系统自动派生 SessionID，获取或创建会话，传递给处理器

#### Scenario: 路由时更新会话活跃时间
- **WHEN** 调用 `router.RouteWithSession(ctx, msg)`
- **THEN** 会话的 `LastActiveAt` 被更新

#### Scenario: 处理器接收会话上下文
- **WHEN** 处理器被调用
- **THEN** 处理器函数签名包含 `session *Session` 参数

### Requirement: 可配置的时间函数

系统 SHALL 支持注入时间函数以便于测试。

#### Scenario: 默认使用系统时间
- **WHEN** 创建 SessionManager 且未指定 `now` 函数
- **THEN** 系统使用 `time.Now()` 获取当前时间

#### Scenario: 注入 mock 时间
- **WHEN** 创建 SessionManager 时指定自定义 `now` 函数
- **THEN** 所有时间相关操作使用该函数返回的时间

---

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

---

### Requirement: 会话清除

系统 SHALL 支持清除项目的所有会话。

#### Scenario: 清除项目会话
- **WHEN** 调用 `project.ClearSessions()`
- **THEN** 系统 SHALL 销毁该项目的所有会话
- **AND** 其他项目的会话 SHALL 不受影响

#### Scenario: 切换项目时清除会话
- **WHEN** 用户切换项目且未指定 --keep
- **THEN** 系统 SHALL 清除旧项目的所有会话
