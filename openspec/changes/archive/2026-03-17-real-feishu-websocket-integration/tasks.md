## 1. SDK 接入与契约测试

- [x] 1.1 为 `internal/platform/feishu` 补充围绕真实 SDK facade 的失败测试，覆盖连接成功、认证失败、context 取消和假阳性连接状态
- [x] 1.2 在 `go.mod` / `go.sum` 中引入飞书官方 Go SDK，并建立可替换的 SDK facade 接口，避免测试直接依赖真实 SDK 对象
- [x] 1.3 重构 `internal/platform/feishu/client_impl.go` 的基础结构，使 `FeishuClient` 继续保持稳定，对内切换到 facade 驱动的真实实现

## 2. 真实长连接与事件接收

- [x] 2.1 为真实连接状态机补充失败测试，覆盖 `Connect`、`Disconnect`、`IsConnected`、异常断开和恢复前后的状态变化
- [x] 2.2 实现基于官方 SDK 的真实长连接建立、关闭、连接状态维护和恢复逻辑
- [x] 2.3 为事件接收链路补充测试，覆盖真实 SDK 回调注册、事件入队、处理失败不破坏连接
- [x] 2.4 实现 SDK 事件回调到现有 `Adapter.HandleEvent` 的桥接，并保持异步处理边界

## 3. 真实消息发送与应用启动集成

- [x] 3.1 为 `SendText` 和应用启动失败路径补充失败测试，覆盖未连接发送、发送成功、连接失败向 `app.Start()` 透传
- [x] 3.2 实现基于真实飞书 API 的文本消息发送逻辑，并与现有 `Sender` 适配
- [x] 3.3 调整 `internal/app` 和 `cmd/cc-connect` 的启动反馈，明确区分“进程启动”与“飞书长连接就绪”

## 4. 文档与联调验收

- [x] 4.1 更新 `README.md` 的飞书接入说明，明确“先启动 cc-connect 建立长连接，再到飞书后台继续事件配置”的顺序
- [x] 4.2 增加真实联调说明或验收清单，覆盖凭证准备、启动成功标志、事件订阅、文本消息收发和断开恢复
- [x] 4.3 补充 build tag 或环境变量门控的飞书集成验证入口，确保默认测试不依赖真实外网环境
- [ ] 4.4 使用真实配置完成一次最小联调验收，并记录结果以关闭 `mvp-integration` 中相关手测阻塞项

当前阻塞：本地环境未检测到 `FEISHU_APP_ID` / `FEISHU_APP_SECRET` / `FEISHU_CHAT_ID`，无法完成真实联调验收。

## 5. 最终验证

- [x] 5.1 运行 `go test ./internal/platform/feishu/... -v -cover`，确认飞书适配器核心测试通过且覆盖率达标
- [x] 5.2 运行相关应用层测试，如 `go test ./internal/app/... -v`，确认启动与错误传播链路未回归
- [x] 5.3 运行受影响范围内的完整测试集，并确认 OpenSpec artifacts 与实现结果一致，可进入 `/opsx:apply`
