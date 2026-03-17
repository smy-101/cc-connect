## 1. 日志级别接线

- [x] 1.1 为 `cmd/cc-connect` 增加围绕 `config.LogLevel` 的失败测试，覆盖 `debug` 与 `info` 级别下的 logger 初始化行为。
- [x] 1.2 在 `cmd/cc-connect` 中实现 `log_level` 到 `slog` 级别的解析与接线，确保加载配置后使用配置级别重建默认 logger。

## 2. 飞书链路诊断日志

- [x] 2.1 为 `internal/platform/feishu` 增加失败测试，覆盖 SDK 事件到达、事件转换失败、消息发送失败等关键路径的日志触发点。
- [x] 2.2 在 `internal/platform/feishu/client_impl.go` 中补充连接生命周期与 SDK 事件入口日志，记录 `event_id`、`message_id`、`chat_type`、`message_type` 等关键字段。
- [x] 2.3 在 `internal/platform/feishu/adapter.go` 与 `sdk_facade.go` 中补充事件转换、路由处理与发送结果日志，确保失败路径不再静默。

## 3. 应用处理诊断日志

- [x] 3.1 为 `internal/app` 增加失败测试，覆盖状态回复发送、Claude Code 调用失败、最终回复发送失败等阶段的日志触发点。
- [x] 3.2 在 `internal/app/handlers.go` 与 `reply.go` 中补充文本处理链路日志，记录“正在思考”回复、agent 调用和最终回复发送阶段。
- [x] 3.3 约束日志字段，避免输出 `app_secret`、完整用户消息正文和其它敏感内容，只保留联调所需元数据或长度信息。

## 4. 验证与联调

- [x] 4.1 运行受影响范围内的测试，例如 `go test ./internal/platform/feishu/... -v` 与 `go test ./internal/app/... -v`，确认新增日志未改变既有行为。
- [ ] 4.2 使用真实飞书配置在 `debug` 级别下完成一次单聊文本联调，确认日志足以判断消息停留在事件接收、转换、路由、回复或 agent 调用的哪个阶段。
- [x] 4.3 根据最终实现结果更新相关联调文档或 README 中的排障说明，补充“如何用 debug 日志定位飞书单聊无响应”的最小指导。
