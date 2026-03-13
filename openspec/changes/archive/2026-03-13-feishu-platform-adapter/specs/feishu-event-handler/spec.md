# 飞书事件处理器规格

## Purpose

飞书事件处理器负责解析飞书开放平台推送的 `im.message.receive_v1` 事件，提取关键信息（消息内容、发送者、@提及等），并将事件转发给消息路由器。

## ADDED Requirements

### Requirement: 解析 im.message.receive_v1 事件

系统 SHALL 能够解析飞书 v2.0 版本的 `im.message.receive_v1` 事件结构。

#### Scenario: 解析完整事件结构
- **WHEN** 收到有效的 `im.message.receive_v1` 事件
- **THEN** 解析出以下字段：
  - `event_id`：事件唯一标识
  - `message_id`：消息唯一标识
  - `chat_id`：会话标识
  - `chat_type`：会话类型（p2p/group/topic_group）
  - `message_type`：消息类型
  - `content`：消息内容（JSON 字符串）
  - `create_time`：消息创建时间

#### Scenario: 解析事件头部信息
- **WHEN** 解析事件
- **THEN** 提取 `header` 中的 `app_id`、`tenant_key`、`event_type`

#### Scenario: 处理无效 JSON 事件
- **WHEN** 收到格式无效的 JSON 数据
- **THEN** 返回解析错误
- **THEN** 错误信息包含原始数据的前 100 字符

---

### Requirement: 提取发送者信息

系统 SHALL 从事件中提取完整的发送者信息。

#### Scenario: 提取发送者标识
- **WHEN** 解析消息事件
- **THEN** 提取 `sender.sender_id` 中的 `open_id`、`union_id`、`user_id`

#### Scenario: 提取发送者类型
- **WHEN** 解析消息事件
- **THEN** 提取 `sender.sender_type`（通常为 "user"）

#### Scenario: 无权限时的用户标识
- **WHEN** 应用未获取 `user_id` 权限
- **THEN** `user_id` 字段为空
- **THEN** 使用 `open_id` 作为用户标识

---

### Requirement: 解析 @提及 信息

系统 SHALL 正确解析飞书消息中的 @提及 列表。

#### Scenario: 解析 mentions 数组
- **WHEN** 事件的 `message.mentions` 数组非空
- **THEN** 为每个提及提取 `key`、`id`、`name`、`tenant_key`

#### Scenario: 处理无 mentions 事件
- **WHEN** 事件的 `message.mentions` 为空或不存在
- **THEN** 返回空的提及列表

#### Scenario: 解析 @提及 用户标识
- **WHEN** 解析单个提及
- **THEN** 提取 `id.open_id`、`id.union_id`、`id.user_id`

---

### Requirement: 消息类型判断

系统 SHALL 正确识别飞书消息类型。

#### Scenario: 识别文本消息
- **WHEN** `message_type` 为 `text`
- **THEN** 解析器标记为文本消息

#### Scenario: 识别富文本消息
- **WHEN** `message_type` 为 `post`
- **THEN** 解析器标记为富文本消息

#### Scenario: 识别图片消息
- **WHEN** `message_type` 为 `image`
- **THEN** 解析器标记为图片消息

#### Scenario: 识别语音消息
- **WHEN** `message_type` 为 `audio`
- **THEN** 解析器标记为语音消息

---

### Requirement: 解析文本消息内容

系统 SHALL 解析飞书文本消息的 JSON 内容。

#### Scenario: 解析文本内容 JSON
- **WHEN** `message_type` 为 `text` 且 `content` 为 `{"text":"Hello"}`
- **THEN** 提取文本内容 "Hello"

#### Scenario: 处理包含转义字符的内容
- **WHEN** 内容包含 JSON 转义字符
- **THEN** 正确解码转义字符

---

### Requirement: 解析富文本消息内容

系统 SHALL 解析飞书富文本消息的 JSON 结构。

#### Scenario: 解析富文本结构
- **WHEN** `message_type` 为 `post`
- **THEN** 解析 `content` 中的 `zh_cn` 或 `en_us` 内容
- **THEN** 提取标题和正文段落

#### Scenario: 提取富文本纯文本
- **WHEN** 解析富文本段落
- **THEN** 将所有文本片段拼接为纯文本

#### Scenario: 提取富文本中的 @提及
- **WHEN** 富文本包含 `{"tag":"at","user_id":"ou_xxx"}` 元素
- **THEN** 提取被 @ 用户的标识

---

### Requirement: 事件与消息路由集成

系统 SHALL 将解析后的事件转发给消息路由器。

#### Scenario: 转换并路由消息
- **WHEN** 事件解析成功
- **THEN** 将事件转换为统一消息格式
- **THEN** 调用 `router.Route(ctx, message)` 路由消息

#### Scenario: 路由失败处理
- **WHEN** 消息路由返回错误
- **THEN** 记录错误日志
- **THEN** 不影响后续事件处理

---

### Requirement: 错误日志记录

系统 SHALL 记录事件处理过程中的错误。

#### Scenario: 记录解析错误
- **WHEN** 事件解析失败
- **THEN** 记录错误级别日志
- **THEN** 日志包含事件 ID（如有）和错误原因

#### Scenario: 记录路由错误
- **WHEN** 消息路由失败
- **THEN** 记录警告级别日志
- **THEN** 日志包含消息 ID 和错误原因

---

### Requirement: 区分会话类型

系统 SHALL 正确区分单聊和群聊消息。

#### Scenario: 识别单聊消息
- **WHEN** `chat_type` 为 `p2p`
- **THEN** 标记为单聊消息
- **THEN** 会话 ID 为 `chat_id`

#### Scenario: 识别群聊消息
- **WHEN** `chat_type` 为 `group`
- **THEN** 标记为群聊消息
- **THEN** 会话 ID 为 `chat_id`

#### Scenario: 识别话题群消息
- **WHEN** `chat_type` 为 `topic_group`
- **THEN** 标记为话题群消息
- **THEN** 会话 ID 为 `chat_id`
