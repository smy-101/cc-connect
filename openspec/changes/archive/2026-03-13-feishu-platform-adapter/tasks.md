# 飞书平台适配器实现任务

## 1. 基础设施准备

- [x] 1.1 添加飞书 SDK 依赖
  - **影响**: `go.mod`, `go.sum`
  - **操作**: `go get github.com/larksuite/oapi-sdk-go/v3@latest`
  - **验证**: `go mod tidy` 成功，编译通过

- [x] 1.2 创建飞书适配器目录结构
  - **影响**: `internal/platform/feishu/`
  - **创建文件**: `client.go` (接口定义), `doc.go`
  - **验证**: 目录结构正确，包编译通过

## 2. 消息转换器实现 (TDD)

- [x] 2.1 编写消息转换测试：飞书文本消息 → 统一消息
  - **影响**: `internal/platform/feishu/message_converter_test.go`
  - **TDD**: 先写失败测试
  - **验证**: `go test -run TestTextMessageToUnified ./internal/platform/feishu/` 失败（红）

- [x] 2.2 实现飞书文本消息转统一消息
  - **影响**: `internal/platform/feishu/message_converter.go`
  - **TDD**: 最小实现使测试通过
  - **验证**: `go test -run TestTextMessageToUnified ./internal/platform/feishu/` 通过（绿）

- [x] 2.3 编写消息转换测试：@提及 提取
  - **影响**: `internal/platform/feishu/message_converter_test.go`
  - **测试场景**: 单个提及、多个提及、无提及
  - **验证**: `go test -run TestMentionExtraction ./internal/platform/feishu/` 失败（红）

- [x] 2.4 实现 @提及 信息提取
  - **影响**: `internal/platform/feishu/message_converter.go`
  - **TDD**: 最小实现
  - **验证**: `go test -run TestMentionExtraction ./internal/platform/feishu/` 通过（绿）

- [x] 2.5 编写消息转换测试：统一消息 → 飞书发送格式
  - **影响**: `internal/platform/feishu/message_converter_test.go`
  - **验证**: `go test -run TestUnifiedToFeishu ./internal/platform/feishu/` 失败（红）

- [x] 2.6 实现统一消息转飞书格式
  - **影响**: `internal/platform/feishu/message_converter.go`
  - **TDD**: 最小实现
  - **验证**: `go test -run TestUnifiedToFeishu ./internal/platform/feishu/` 通过（绿）

## 3. 事件解析器实现 (TDD)

- [x] 3.1 编写事件解析测试：完整事件结构解析
  - **影响**: `internal/platform/feishu/event_parser_test.go`
  - **测试数据**: 使用真实 `im.message.receive_v1` 事件 JSON
  - **验证**: `go test -run TestEventParsing ./internal/platform/feishu/` 失败（红）

- [x] 3.2 实现事件解析器
  - **影响**: `internal/platform/feishu/event_parser.go`
  - **TDD**: 最小实现
  - **验证**: `go test -run TestEventParsing ./internal/platform/feishu/` 通过（绿）

- [x] 3.3 编写事件解析测试：发送者信息提取
  - **影响**: `internal/platform/feishu/event_parser_test.go`
  - **验证**: `go test -run TestSenderExtraction ./internal/platform/feishu/` 失败（红）

- [x] 3.4 实现发送者信息提取
  - **影响**: `internal/platform/feishu/event_parser.go`
  - **TDD**: 最小实现
  - **验证**: `go test -run TestSenderExtraction ./internal/platform/feishu/` 通过（绿）

- [x] 3.5 编写事件解析测试：富文本消息解析
  - **影响**: `internal/platform/feishu/event_parser_test.go`
  - **验证**: `go test -run TestPostMessage ./internal/platform/feishu/` 失败（红）

- [x] 3.6 实现富文本消息解析
  - **影响**: `internal/platform/feishu/event_parser.go`
  - **TDD**: 最小实现
  - **验证**: `go test -run TestPostMessage ./internal/platform/feishu/` 通过（绿）

## 4. WebSocket 客户端接口实现 (TDD)

- [x] 4.1 编写客户端接口定义
  - **影响**: `internal/platform/feishu/client.go`
  - **定义**: `FeishuClient` 接口，`Connect`, `Disconnect`, `IsConnected`, `SendText`, `OnEvent` 方法
  - **验证**: 编译通过

- [x] 4.2 编写 Mock 客户端实现
  - **影响**: `internal/platform/feishu/mock_client.go`
  - **用途**: 供其他模块测试使用
  - **验证**: `go test ./internal/platform/feishu/` 通过

- [x] 4.3 编写 SDK 客户端集成测试
  - **影响**: `internal/platform/feishu/client_impl_test.go`
  - **注意**: 使用 build tag `integration`，不依赖真实网络
  - **验证**: `go test -tags=integration ./internal/platform/feishu/` 通过

- [x] 4.4 实现 SDK 客户端封装
  - **影响**: `internal/platform/feishu/client_impl.go`
  - **实现**: 使用 `larkws.NewClient` 创建客户端
  - **验证**: 编译通过，mock 测试通过

- [x] 4.5 实现异步事件处理
  - **影响**: `internal/platform/feishu/client_impl.go`
  - **实现**: 使用 channel 缓冲事件，worker goroutine 处理
  - **验证**: `go test -run TestAsyncEventProcessing ./internal/platform/feishu/` 通过

## 5. 消息发送器实现 (TDD)

- [x] 5.1 编写发送器测试：发送文本消息
  - **影响**: `internal/platform/feishu/sender_test.go`
  - **使用 Mock**: Mock SDK API 客户端
  - **验证**: `go test -run TestSendTextMessage ./internal/platform/feishu/` 失败（红）

- [x] 5.2 实现文本消息发送
  - **影响**: `internal/platform/feishu/sender.go`
  - **TDD**: 最小实现
  - **验证**: `go test -run TestSendTextMessage ./internal/platform/feishu/` 通过（绿）

- [x] 5.3 编写发送器测试：错误处理
  - **影响**: `internal/platform/feishu/sender_test.go`
  - **测试场景**: 无效 chat_id、网络错误、权限不足
  - **验证**: `go test -run TestSendErrors ./internal/platform/feishu/` 失败（红）

- [x] 5.4 实现发送错误处理
  - **影响**: `internal/platform/feishu/sender.go`
  - **TDD**: 最小实现
  - **验证**: `go test -run TestSendErrors ./internal/platform/feishu/` 通过（绿）

## 6. 路由集成

- [x] 6.1 编写集成测试：飞书事件 → 消息路由
  - **影响**: `internal/platform/feishu/integration_test.go`
  - **测试**: 模拟飞书事件，验证消息被正确路由
  - **验证**: `go test -run TestFeishuToRouter ./internal/platform/feishu/` 失败（红）

- [x] 6.2 实现飞书适配器与路由器集成
  - **影响**: `internal/platform/feishu/adapter.go`
  - **实现**: 创建 `FeishuAdapter` 结构，将事件转换为消息并调用 `router.Route`
  - **验证**: `go test -run TestFeishuToRouter ./internal/platform/feishu/` 通过（绿）

## 7. 最终验证

- [x] 7.1 运行完整测试套件
  - **验证**: `go test ./internal/platform/feishu/... -v -cover` 覆盖率 ≥ 85%

- [x] 7.2 运行竞态检测
  - **验证**: `go test ./internal/platform/feishu/... -race` 通过

- [x] 7.3 运行全部测试
  - **验证**: `go test ./...` 通过

- [x] 7.4 更新 CLAUDE.md 文档
  - **影响**: `CLAUDE.md`
  - **内容**: 添加飞书适配器相关说明

---

## 任务依赖关系

```
1.x (基础设施)
    │
    ├──► 2.x (消息转换器) ──┬──► 6.x (路由集成) ──► 7.x (验证)
    │                       │
    ├──► 3.x (事件解析器) ──┤
    │                       │
    └──► 4.x (客户端接口) ──┤
            │               │
            └──► 5.x (发送器)┘
```

**可并行任务**:
- 2.x、3.x、4.x 可以并行开始
- 5.x 依赖 4.x（需要接口定义）

**关键路径**: 1.x → 4.x → 5.x → 6.x → 7.x
