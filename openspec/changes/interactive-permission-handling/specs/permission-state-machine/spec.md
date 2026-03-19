## ADDED Requirements

### Requirement: 权限状态结构

系统必须提供 `pendingPermission` 结构来管理权限请求的生命周期。

- 必须包含 `requestID` 字段，用于唯一标识权限请求
- 必须包含 `toolName` 字段，标识请求的工具名称
- 必须包含 `toolInput` 字段，提供工具输入预览
- 必须包含 `resolved` channel，用于阻塞/唤醒机制
- 必须包含 `result` 字段，存储用户决策结果
- 必须并发安全

#### Scenario: 创建权限请求状态

- **WHEN** 收到 `control_request` 事件
- **THEN** 系统创建 `pendingPermission` 实例
- **THEN** `resolved` channel 已初始化但未关闭
- **THEN** `result` 为 nil

#### Scenario: 存储用户决策

- **WHEN** 用户批准权限请求
- **THEN** 系统将 `result.behavior` 设为 "allow"
- **THEN** 关闭 `resolved` channel

#### Scenario: 并发访问安全

- **WHEN** 多个 goroutine 同时访问 `pendingPermission`
- **THEN** `result` 字段读写必须通过互斥锁保护
- **THEN** channel 关闭操作必须只执行一次

### Requirement: 权限请求阻塞与唤醒

Session 的消息处理必须支持在权限请求处阻塞，等待用户响应后唤醒继续。

- 当收到 `control_request` 且非 autoApprove 模式时，必须阻塞
- 用户响应后，必须通过关闭 `resolved` channel 唤醒
- 阻塞必须支持 context 取消
- 阻塞必须有超时机制（默认 5 分钟）

#### Scenario: 阻塞等待用户响应

- **GIVEN** session.autoApprove = false
- **WHEN** 收到 `control_request` 事件
- **THEN** 系统创建 `pendingPermission` 并阻塞当前处理
- **THEN** 向 events 通道发送 `EventPermissionRequest`

#### Scenario: 用户响应后唤醒

- **GIVEN** 系统正在阻塞等待权限响应
- **WHEN** 调用 `session.RespondPermission(requestID, result)`
- **THEN** 系统存储 result
- **THEN** 关闭 `resolved` channel
- **THEN** 阻塞的处理继续执行

#### Scenario: 超时自动拒绝

- **GIVEN** 系统正在阻塞等待权限响应
- **WHEN** 等待超过 5 分钟
- **THEN** 系统自动发送 deny 响应
- **THEN** 继续处理（不会永久阻塞）

#### Scenario: Context 取消

- **GIVEN** 系统正在阻塞等待权限响应
- **WHEN** context 被取消
- **THEN** 阻塞立即返回错误
- **THEN** 不会发送任何响应给 Claude

### Requirement: RespondPermission API

Session 必须提供 `RespondPermission` 方法供外部调用。

- 必须验证 requestID 匹配当前 pending 请求
- 必须支持 "allow" 和 "deny" 两种 behavior
- 对不存在的 requestID 必须返回错误
- 对已处理的 requestID 必须返回错误（幂等性）

#### Scenario: 批准权限请求

- **GIVEN** 存在 pending 权限请求 "req123"
- **WHEN** 调用 `session.RespondPermission("req123", {Behavior: "allow"})`
- **THEN** 系统发送 `control_response` 到 Claude stdin
- **THEN** 返回 nil

#### Scenario: 拒绝权限请求

- **GIVEN** 存在 pending 权限请求 "req123"
- **WHEN** 调用 `session.RespondPermission("req123", {Behavior: "deny", Message: "用户拒绝"})`
- **THEN** 系统发送 `control_response` 到 Claude stdin，包含 deny message
- **THEN** 返回 nil

#### Scenario: 无效 requestID

- **WHEN** 调用 `session.RespondPermission("invalid", ...)`
- **THEN** 返回错误 "no pending permission request with id: invalid"

#### Scenario: 重复响应

- **GIVEN** requestID "req123" 已被处理
- **WHEN** 再次调用 `session.RespondPermission("req123", ...)`
- **THEN** 返回错误 "permission request already resolved"

### Requirement: 自动批准模式

系统必须支持自动批准所有权限请求的模式。

- YOLO 模式（bypassPermissions）必须自动批准
- 自动批准时不得阻塞，直接发送 allow 响应
- 自动批准时仍需发送 `EventPermissionRequest` 事件（用于日志）

#### Scenario: YOLO 模式自动批准

- **GIVEN** session.autoApprove = true
- **WHEN** 收到 `control_request` 事件
- **THEN** 系统立即发送 allow 响应
- **THEN** 不阻塞处理流程
- **THEN** 发送 `EventPermissionRequest` 事件（标记 autoApproved）

#### Scenario: 切换到 YOLO 模式

- **GIVEN** session.autoApprove = false
- **WHEN** 调用 `session.SetAutoApprove(true)`
- **THEN** 后续权限请求自动批准
