# 飞书消息转换器规格

## Purpose

飞书消息转换器负责在飞书消息格式和 cc-connect 统一消息模型之间进行双向转换，确保消息在不同系统间正确传递。

## ADDED Requirements

### Requirement: 飞书文本消息转统一消息

系统 SHALL 能够将飞书文本消息转换为 cc-connect 统一消息模型的 text 类型。

#### Scenario: 转换简单文本消息
- **WHEN** 调用 `converter.ToUnifiedMessage(feishuTextEvent)`
- **THEN** 返回统一消息，类型为 `text`
- **THEN** 消息的 `content` 字段包含飞书消息的文本内容
- **THEN** 消息的 `platform` 字段为 `feishu`
- **THEN** 消息的 `user_id` 字段为飞书用户的 `open_id`

#### Scenario: 转换包含 @提及 的文本消息
- **WHEN** 飞书消息内容为 `@_user_1 你好`
- **THEN** 转换后的消息内容保持 `@_user_1 你好` 格式
- **THEN** @提及 信息被提取到消息的元数据中

#### Scenario: 转换空文本消息
- **WHEN** 飞书消息内容为空字符串
- **THEN** 返回错误，提示消息内容不能为空

---

### Requirement: 飞书富文本消息转统一消息

系统 SHALL 能够将飞书富文本（post）消息转换为统一消息模型。

#### Scenario: 转换富文本为纯文本
- **WHEN** 调用 `converter.ToUnifiedMessage(feishuPostEvent)`
- **THEN** 富文本内容被提取为纯文本格式
- **THEN** 消息类型为 `text`

#### Scenario: 提取富文本中的 @提及
- **WHEN** 富文本包含 `{"tag": "at", "user_id": "ou_xxx"}` 元素
- **THEN** @提及 信息被正确提取
- **THEN** 文本中的 @ 部分被替换为标准格式

---

### Requirement: 统一消息转飞书发送格式

系统 SHALL 能够将统一消息转换为飞书 API 所需的发送格式。

#### Scenario: 转换文本消息为飞书格式
- **WHEN** 调用 `converter.ToFeishuContent(unifiedTextMessage)`
- **THEN** 返回飞书文本消息的 JSON 格式：`{"text": "内容"}`

#### Scenario: 转换包含特殊字符的消息
- **WHEN** 统一消息内容包含双引号、换行符等特殊字符
- **THEN** JSON 序列化正确转义这些字符

---

### Requirement: 用户标识转换

系统 SHALL 正确处理飞书用户标识与统一消息用户标识的映射。

#### Scenario: 使用 open_id 作为用户标识
- **WHEN** 转换飞书消息为统一消息
- **THEN** 使用飞书的 `open_id` 作为统一消息的 `user_id`

#### Scenario: 保留原始用户信息
- **WHEN** 转换消息时
- **THEN** 在消息元数据中保留飞书的 `union_id` 和 `user_id`（如有权限）

---

### Requirement: 会话标识转换

系统 SHALL 正确处理飞书会话标识与统一消息会话的映射。

#### Scenario: 单聊会话标识
- **WHEN** 飞书消息的 `chat_type` 为 `p2p`
- **THEN** 使用飞书的 `chat_id` 作为会话标识

#### Scenario: 群聊会话标识
- **WHEN** 飞书消息的 `chat_type` 为 `group`
- **THEN** 使用飞书的 `chat_id` 作为会话标识

---

### Requirement: @提及信息提取

系统 SHALL 从飞书消息中提取完整的 @提及 信息。

#### Scenario: 提取单个 @提及
- **WHEN** 飞书消息包含一个 @提及
- **THEN** 提取被提及用户的 `key`、`open_id`、`name`

#### Scenario: 提取多个 @提及
- **WHEN** 飞书消息包含多个 @提及
- **THEN** 所有被提及用户的信息都被提取
- **THEN** 提取顺序与消息中出现顺序一致

#### Scenario: 无 @提及消息
- **WHEN** 飞书消息不包含任何 @提及
- **THEN** 提取结果为空列表，不返回错误

---

### Requirement: 非支持消息类型处理

系统 SHALL 优雅处理暂不支持的消息类型。

#### Scenario: 收到图片消息
- **WHEN** 飞书消息类型为 `image`
- **THEN** 返回统一消息，类型为 `image`
- **THEN** content 为图片 key（后续阶段实现图片下载）

#### Scenario: 收到语音消息
- **WHEN** 飞书消息类型为 `audio`
- **THEN** 返回统一消息，类型为 `voice`
- **THEN** content 为语音 key（后续阶段实现语音处理）

#### Scenario: 收到未知消息类型
- **WHEN** 飞书消息类型为系统未知的类型
- **THEN** 返回错误，记录原始消息类型
