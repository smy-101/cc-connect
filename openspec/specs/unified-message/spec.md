# 统一消息模型规格

## ADDED Requirements

### Requirement: 支持四种消息类型

系统 SHALL 支持四种消息类型：`text`（文本）、`voice`（语音）、`image`（图片）、`command`（命令）。

#### Scenario: 创建文本消息
- **WHEN** 使用 `NewTextMessage(platform, userID, content)` 创建消息
- **THEN** 返回的消息类型为 `text`
- **THEN** 消息的 Platform、UserID、Content 字段被正确设置

#### Scenario: 创建语音消息
- **WHEN** 使用 `NewVoiceMessage(platform, userID, content)` 创建消息
- **THEN** 返回的消息类型为 `voice`

#### Scenario: 创建图片消息
- **WHEN** 使用 `NewImageMessage(platform, userID, content)` 创建消息
- **THEN** 返回的消息类型为 `image`

#### Scenario: 创建命令消息
- **WHEN** 使用 `NewCommandMessage(platform, userID, content)` 创建消息
- **THEN** 返回的消息类型为 `command`

#### Scenario: 使用通用构造函数创建消息
- **WHEN** 使用 `NewMessage(platform, userID, content, MessageTypeText)` 创建消息
- **THEN** 返回的消息类型与传入的类型参数一致

---

### Requirement: 消息自动生成 ID 和时间戳

系统 SHALL 为每个新创建的消息自动生成唯一 ID 和当前时间戳。

#### Scenario: ID 自动生成
- **WHEN** 创建任意类型的消息
- **THEN** 消息的 ID 字段不为空
- **THEN** ID 格式为 `<unix_nano>_<random_8chars>`

#### Scenario: 时间戳自动生成
- **WHEN** 创建任意类型的消息
- **THEN** 消息的 Timestamp 字段被设置为当前时间
- **THEN** Timestamp 与创建时间的差距在 1 秒以内

#### Scenario: ID 唯一性
- **WHEN** 连续创建 1000 个消息
- **THEN** 所有消息的 ID 互不相同

---

### Requirement: 消息字段语义

消息 SHALL 包含以下字段，字段名使用 snake_case：
- `id`：消息唯一标识（字符串）
- `platform`：来源平台标识（字符串）
- `user_id`：用户标识（字符串）
- `content`：消息内容（字符串）
- `type`：消息类型（text/voice/image/command）
- `timestamp`：消息创建时间（RFC3339 格式）

#### Scenario: 字段完整性
- **WHEN** 创建消息
- **THEN** 消息包含所有必需字段

#### Scenario: 字段命名规范
- **WHEN** 消息被序列化为 JSON
- **THEN** JSON 字段名使用 snake_case（如 `user_id` 而非 `userId`）

---

### Requirement: JSON 序列化

系统 SHALL 支持将消息序列化为 JSON 格式。

#### Scenario: 序列化文本消息
- **WHEN** 调用 `message.ToJSON()` 序列化文本消息
- **THEN** 返回有效的 JSON 字节流
- **THEN** JSON 包含所有字段

#### Scenario: 序列化包含特殊字符的消息
- **WHEN** 消息内容包含 Unicode 字符（如中文、emoji）
- **THEN** 序列化后的 JSON 正确编码这些字符

---

### Requirement: JSON 反序列化

系统 SHALL 支持从 JSON 格式反序列化消息。

#### Scenario: 反序列化有效 JSON
- **WHEN** 调用 `FromJSON(validJSONData)` 反序列化
- **THEN** 返回对应的 Message 对象
- **THEN** 所有字段被正确设置

#### Scenario: 反序列化忽略未知字段
- **WHEN** JSON 包含未知字段（如 `extra_field`）
- **THEN** 反序列化成功，未知字段被忽略
- **THEN** 已知字段被正确解析

#### Scenario: 反序列化无效 JSON
- **WHEN** JSON 格式无效（如缺少引号、括号不匹配）
- **THEN** 返回错误

#### Scenario: 反序列化缺少必需字段
- **WHEN** JSON 缺少必需字段（如缺少 `type`）
- **THEN** 返回错误

---

### Requirement: 序列化往返一致性

消息经过序列化和反序列化后 SHALL 保持数据一致。

#### Scenario: 往返一致性
- **WHEN** 创建消息并执行 ToJSON 后再 FromJSON
- **THEN** 反序列化后的消息与原消息的所有字段值相等

#### Scenario: 时间戳精度保持
- **WHEN** 消息经过序列化/反序列化往返
- **THEN** Timestamp 字段的精度保持在纳秒级别
