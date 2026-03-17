## 1. 锁定失败场景

- [x] 1.1 在 `internal/platform/feishu/adapter_test.go` 中先把 `SendReply` 相关断言改为飞书 `text` content JSON 字符串，制造当前实现下的失败测试。
- [x] 1.2 在 `internal/platform/feishu/integration_test.go` 中更新 reply round-trip 断言，使其验证 reply path 发送的是合法飞书 JSON content，而不是裸文本。
- [x] 1.3 为 reply path 增加一个包含引号或换行的特殊字符回归测试，验证发送内容与统一消息发送路径保持一致。

## 2. 实现最小修复

- [x] 2.1 在 `internal/platform/feishu` 中将 `Adapter.SendReply` 接到现有文本 content 转换逻辑上，保持 `ReplySender` 和 `FeishuClient.SendText` 的接口语义不变。
- [x] 2.2 如实现需要，在 `Sender` 内补一个最小辅助入口或复用 `SendMessage` 路径，但不得在 SDK facade 或 app 层新增飞书 JSON 拼装逻辑。
- [x] 2.3 复查 reply path 与 unified message path 的文本发送语义，确保两条路径对普通文本和特殊字符文本生成一致的 `content`。

## 3. 验证与收口

- [x] 3.1 运行 `go test ./internal/platform/feishu/... -v`，确认飞书适配层测试通过且 reply path 回归场景覆盖到位。
- [x] 3.2 运行 `go test ./internal/app/... -v`，确认 app 层 reply 抽象和现有消息处理链路无回归。
- [ ] 3.3 使用真实飞书配置复测一次文本消息收发，确认不再出现 `content is not a string in json format`，并将联调结果回填到相关 OpenSpec change 或验收记录。
