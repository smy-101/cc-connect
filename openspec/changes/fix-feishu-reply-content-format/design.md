## Context

当前真实飞书联调暴露的问题很集中：入站事件能够到达 `Adapter.HandleEvent`，消息也能进入现有 handler 链路，但回复发送在真实 Feishu API 处失败，错误为 `content is not a string in json format`。这说明问题不在长连接、事件解析或路由，而在回复出站链路的 content 语义。

现状中存在两条文本出站路径：

- `SendMessage` 会先通过 `MessageConverter.ToFeishuContent` 把统一消息转换为飞书 `text` 消息的 JSON 字符串。
- `SendReply` 会把 app 层传入的纯文本直接透传给 `FeishuClient.SendText`。

而 `FeishuClient.SendText` 的职责已经约定为“发送符合飞书格式的 content 字符串”。因此这不是 SDK facade 的缺陷，而是上游 reply path 没有履行平台适配责任。约束是本次修复必须保持接口边界稳定，不把飞书 JSON 格式泄漏到 app 层，也不重新定义底层 client 契约。

## Goals / Non-Goals

**Goals:**

- 让 `SendReply` 和 `SendMessage` 对文本回复遵循同一套飞书 content 语义。
- 把“纯文本 -> 飞书 JSON content”的转换责任明确固定在飞书适配层，而不是 app 层或 SDK 层。
- 以最小改动恢复真实飞书环境下的文本回复能力。
- 调整测试，使 mock 断言和真实飞书 API 约束一致，并增加防回归场景。

**Non-Goals:**

- 不重构 `FeishuClient`、`SDKClient` 或 `realSDKFacade` 的公开职责。
- 不在本次变更中引入新的消息格式抽象。
- 不修改 Claude Code、路由、会话或命令系统的业务行为。
- 不扩展卡片消息、图片、语音或富文本发送能力。

## Decisions

### D1: 文本回复转换责任保留在飞书适配层

**选择**：继续让 app 层只传纯文本回复，由飞书适配层在调用 `FeishuClient.SendText` 之前完成飞书 content 转换。

**理由**：

- app 层不应知道飞书 `text` 消息必须是 JSON 字符串这一平台细节。
- `FeishuClient.SendText` 现有注释和真实 SDK 调用已经把它定义为“发送飞书格式 content”，改它会扩大影响面。
- 问题根因就在适配层职责缺失，把转换逻辑补回这一层即可闭环。

**替代方案**：

- 在 app 层直接构造飞书 JSON：会让平台细节泄漏到上层，破坏抽象边界。
- 在 SDK facade 内自动包装裸文本：会让 `SendText` 同时接受两种语义，接口含义变模糊，测试也更难约束。

### D2: 回复路径复用既有文本 content 转换逻辑

**选择**：`SendReply` 不单独实现一套 JSON 包装，而是复用当前已经服务于 `SendMessage` 的文本转换逻辑。

**理由**：

- 既有 `MessageConverter.ToFeishuContent` 已经覆盖 JSON 编码和转义语义，是仓库内唯一正确的飞书文本 content 构造入口。
- 复用现有转换逻辑可以保证 `SendReply` 与 `SendMessage` 对换行、引号等特殊字符行为一致。
- 这样改动最小，风险最低。

**替代方案**：

- 在 `SendReply` 里手写 `{"text":"..."}` 拼接：容易漏掉转义规则，且未来再次漂移。
- 让 `Sender.SendText` 从“发送飞书格式 content”改成“接收纯文本并自动转换”：可行，但会牵动更多现有测试和命名语义，不是最小修复路径。

### D3: 测试以真实飞书契约为准，而不是以宽松 mock 为准

**选择**：更新 reply 相关测试断言，使 mock client 记录到的 `content` 必须是飞书 JSON 字符串；补一个特殊字符回归测试验证 reply path 确实复用统一转换逻辑。

**理由**：

- 当前问题之所以漏过，正是因为测试默认把裸文本当成正确输出。
- reply path 的核心风险不是“有没有调用 SendText”，而是“调用时 content 的格式是否满足真实 API 要求”。
- 对特殊字符场景做回归测试，可以覆盖最容易被手写 JSON 拼接破坏的部分。

**替代方案**：

- 保持现有测试不变，只依赖真实联调发现问题：反馈太晚，且不利于 CI 防回归。
- 只改一个 happy path 断言，不覆盖转义字符：仍可能让新的格式错误漏网。

## Risks / Trade-offs

- [reply path 复用 `SendMessage` 语义时需要构造最小统一消息对象] → 通过仅填充 `ChannelID`、`Content`、`Type` 等必要字段控制额外耦合。
- [更新测试后，现有大量基于裸文本的断言会同时失败] → 优先收敛到 `adapter` 和 `integration` 这两个直接验证 reply path 的测试层，避免无关测试扩散。
- [若未来有人再次直接调用底层 `SendText` 发送裸文本] → 通过新增 spec 和回归测试，把 reply path 的正确行为固化为仓库约束。

## Migration Plan

1. 先补或更新 reply path 失败测试，明确 `SendReply` 发送的必须是飞书 JSON content。
2. 在飞书适配层把 `SendReply` 接到现有的文本 content 转换逻辑上，保持 SDK facade 不变。
3. 调整适配层和 reply round-trip 测试断言，新增特殊字符回归测试。
4. 运行受影响测试，确认 `SendReply` 与 `SendMessage` 文本语义一致。

**回滚策略：**

- 若修复引入额外副作用，可回滚本次适配层改动，恢复到当前行为；底层 client 契约和 app 层接口不会受影响，因此回滚范围清晰。

## Open Questions

- 最小实现是让 `Adapter.SendReply` 直接走 `SendMessage`，还是在 `Sender` 内新增一个仅供 reply path 使用的辅助入口；以改动最少、测试最稳的方式为准。
- 是否需要在 `MockClient` 增加可选的飞书 content 校验模式，帮助未来更早发现类似问题；若超出本次最小修复，可暂留后续改进。
