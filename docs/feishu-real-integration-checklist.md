# 飞书真实联调清单

本文档用于验收 cc-connect 的真实飞书 WebSocket 长连接能力。默认单元测试和常规 `go test ./...` 不会触发真实外网依赖；只有在显式提供环境变量并运行带 `integration` build tag 的测试时，才会访问飞书开放平台。

## 前置条件

1. 已在飞书开放平台创建自建应用，并启用机器人能力与 WebSocket 长连接。
2. 本地已安装 Claude Code CLI，并能在目标项目目录正常启动。
3. 已准备以下环境变量：

```bash
export FEISHU_APP_ID="your-app-id"
export FEISHU_APP_SECRET="your-app-secret"
# 可选：用于验证真实发消息
export FEISHU_CHAT_ID="oc_xxx"
```

## 推荐启动顺序

1. 编译或直接运行 cc-connect。
2. 观察日志，直到出现 `Feishu long connection ready`。
3. 只有在看到该日志之后，再进入飞书开放平台继续完成事件订阅配置。
4. 在飞书会话中向机器人发送文本消息，验证接收与回复。

## 启动成功标志

- CLI 日志先输出 `Process initialized, waiting for Feishu long connection readiness`
- 随后输出 `Feishu long connection ready`
- 若凭证错误或长连接失败，进程应直接退出，不应打印就绪日志

## 真实集成验证入口

```bash
# 仅验证真实连接
go test ./internal/platform/feishu/... -tags=integration -run TestSDKClientIntegration/connect_to_real_feishu_websocket -v

# 连接 + 真实文本发送（需要 FEISHU_CHAT_ID）
go test ./internal/platform/feishu/... -tags=integration -run TestSDKClientIntegration/send_text_through_real_feishu_api -v
```

## 验收清单

- [ ] 使用有效 `FEISHU_APP_ID` / `FEISHU_APP_SECRET` 成功建立长连接
- [ ] 启动日志明确显示长连接 ready，而不是仅显示进程启动
- [ ] 在 ready 之后完成飞书后台事件订阅配置
- [ ] 接收到真实 `im.message.receive_v1` 文本事件
- [ ] 机器人成功发送文本回复
- [ ] 人为断开网络或关闭连接后，连接能够恢复并继续收消息

## 当前环境记录

- 日期：2026-03-16
- 状态：未完成真实联调
- 原因：当前工作区未检测到 `FEISHU_APP_ID` / `FEISHU_APP_SECRET` / `FEISHU_CHAT_ID` 环境变量，无法在本地执行最小真实验收

## 排障建议

1. 若启动阶段失败，先核对应用凭证、机器人能力和 WebSocket 长连接是否已开启。
2. 若没有出现 ready 日志，不要继续飞书后台事件配置，先解决连接问题。
3. 若能连接但不能回复，优先检查机器人是否在目标群内，以及 `FEISHU_CHAT_ID` 是否正确。

## 如何用 debug 日志定位单聊无响应

1. 先把运行配置中的 `log_level` 调整为 `debug`，然后重新启动 cc-connect。
2. 如果看不到 `Feishu SDK event received`，问题通常还停留在飞书事件订阅、权限或长连接层。
3. 如果出现 `Feishu event conversion failed`，说明事件已经到达，但转换成统一消息失败。
4. 如果出现 `Feishu message routing failed`，说明问题停留在路由或 handler，而不是飞书收消息。
5. 如果出现 `Sending thinking reply` 或 `Sending final reply` 后紧跟 `Feishu reply send failed`，说明应用处理到了回复阶段，但飞书出站失败。
6. 如果出现 `Claude Code request started` 后紧跟 `Claude Code invocation failed` 或 `Claude Code request timed out`，说明飞书入站正常，阻塞点在 Claude Code 调用。

联调时重点关注 `event_id`、`message_id`、`channel_id`、`chat_type`、`message_type` 和长度字段；默认实现不会把 `app_secret`、完整用户消息正文或完整回复正文写入日志。
