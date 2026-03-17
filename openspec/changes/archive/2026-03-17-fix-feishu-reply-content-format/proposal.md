## Why

### 背景

当前真实飞书联调已经证明长连接、事件接收和消息路由主链路可用，但机器人在发送“正在思考”或最终回复时会被飞书 API 拒绝，错误为 `content is not a string in json format.`。这说明回复出站链路把纯文本直接传给了飞书发送接口，没有按飞书 `text` 消息要求包装成 JSON 字符串。

### 目标

本次变更属于阶段 2 飞书平台适配器的缺陷修复，目标是在不扩大改动面的前提下，统一飞书回复链路与现有统一消息发送链路的 content 语义，恢复真实环境下的文本回复能力，并补齐能阻止同类回归的测试约束。

## What Changes

### 范围

- 修正飞书回复发送路径，使 app 层传入的纯文本回复在进入 FeishuClient 之前被转换为合法的飞书 `text` content JSON 字符串。
- 保持 `ReplySender`、`feishu.Adapter.SendReply` 和 `FeishuClient.SendText` 的对外职责边界不变，不重定义底层 SDK facade 的发送契约。
- 统一 `SendReply` 与 `SendMessage` 的文本发送语义，避免同一适配器内出现“部分路径发送裸文本、部分路径发送飞书 JSON content”的不一致。
- 调整飞书适配层和相关集成测试断言，使测试以真实飞书 API 约束为准，而不是以当前 mock 的宽松行为为准。
- 补充至少一个针对特殊字符或换行文本的回归测试，确保 reply 路径持续复用正确的 content 转换逻辑。

### 非目标

- 不重构整个飞书发送器设计。
- 不修改 Claude Code 调用链路、消息路由或会话管理行为。
- 不引入新的消息类型、卡片消息或图片/语音发送能力。
- 不改变真实 SDK facade 的请求构造方式，只修正上游调用输入。

## Capabilities

### New Capabilities
- `feishu-text-replies`: 规范飞书适配器发送文本回复和状态消息时，必须将纯文本转换为合法的 Feishu `text` 消息 content JSON 字符串后再调用发送接口。

### Modified Capabilities
- 无

## Impact

### 影响模块

- `internal/platform/feishu`: `Adapter.SendReply`、`Sender` 的文本发送职责边界与回复链路。
- `internal/app`: 无接口变化，但其 reply 路径将恢复真实飞书可用性。
- 测试：`internal/platform/feishu` 中与 `SendReply`、reply round-trip、mock 断言相关的测试用例。

### 验收标准

- 在真实飞书环境中，机器人收到文本消息后，至少能够成功发送一条文本状态回复，不再出现 `content is not a string in json format`。
- `SendReply` 路径发送到 `FeishuClient.SendText` 的 content 必须是形如 `{"text":"..."}` 的 JSON 字符串，而不是裸文本。
- `SendMessage` 与 `SendReply` 对文本消息的发送语义保持一致，不允许一条路径发送飞书 JSON content、另一条路径发送裸文本。
- 飞书适配层测试中，`SendReply` 相关断言必须以飞书 JSON content 为准；至少包含一个带引号、换行或其他需要转义字符的回归场景。
- 受影响范围内的应用层和飞书适配层测试通过，且不需要修改 `FeishuClient.SendText` 的现有接口约定。

### 风险与缓解

- 现有 mock 测试长期基于裸文本断言，修复后会出现一批预期性失败：通过先更新适配层断言，再补回归测试收敛风险。
- 若修复位置选在 SDK facade 或 client 层，可能会模糊接口职责：通过把转换责任限定在 adapter/sender 层，保持 `FeishuClient` 契约稳定。
- 若回复路径单独实现一套 JSON 包装逻辑，未来可能再次与 `SendMessage` 漂移：通过明确要求复用既有文本 content 转换逻辑降低分叉风险。
