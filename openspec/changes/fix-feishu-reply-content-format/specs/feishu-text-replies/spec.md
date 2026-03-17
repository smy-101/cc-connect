## ADDED Requirements

### Requirement: 文本回复在发送前必须转换为 Feishu text content
飞书适配器在发送文本回复或状态消息时 MUST 先将纯文本内容转换为合法的 Feishu `text` 消息 content JSON 字符串，再调用底层发送接口。

#### Scenario: 发送普通文本回复
- **WHEN** 上层通过 reply 路径发送普通文本内容
- **THEN** 飞书适配器传给底层发送接口的 `content` 是形如 `{"text":"..."}` 的 JSON 字符串，而不是裸文本

#### Scenario: 发送状态消息
- **WHEN** 应用发送“正在思考”之类的文本状态回复
- **THEN** 飞书适配器同样将该状态文本编码为合法的 Feishu `text` content 后再发送

### Requirement: 文本回复与统一消息发送保持一致的编码语义
飞书适配器在 `SendReply` 和统一消息发送路径上 SHALL 对文本内容使用一致的 Feishu content 编码语义，避免同一适配器内部出现不同的文本发送格式。

#### Scenario: reply path 与 unified message path 发送相同文本
- **WHEN** `SendReply` 与统一消息发送路径发送相同的文本内容到飞书
- **THEN** 两条路径生成的 `content` 值一致，并都满足 Feishu `text` 消息格式要求

#### Scenario: 文本包含需要转义的字符
- **WHEN** 回复文本包含引号、换行或其他需要 JSON 转义的字符
- **THEN** 飞书适配器生成合法 JSON 字符串，且文本语义在发送后保持不变

### Requirement: Reply path 的测试必须按真实飞书接口契约断言
与飞书回复发送相关的自动化测试 MUST 以真实飞书发送接口要求的 `content` 格式为断言依据，而不是仅断言调用发生。

#### Scenario: 适配层测试验证 reply content
- **WHEN** 测试通过 mock client 验证 `SendReply` 行为
- **THEN** 测试断言底层收到的是 Feishu `text` content JSON 字符串

#### Scenario: 回归测试覆盖特殊字符文本
- **WHEN** 测试发送包含特殊字符的文本回复
- **THEN** 测试验证 reply path 生成的 `content` 仍为合法 JSON 字符串，并与统一消息发送路径一致
