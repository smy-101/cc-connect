## Context

cc-connect 已完成阶段 1 核心消息系统的实现，包括：
- `internal/core/message.go`：统一消息模型
- `internal/core/router.go`：消息路由器
- `internal/core/session.go`：会话管理
- `internal/core/config.go`：配置管理（已定义 FeishuConfig 结构）

本设计为阶段 2 飞书平台适配器，目标是建立飞书与 cc-connect 之间的消息桥梁。

**技术背景**：
- 飞书开放平台提供 WebSocket 长连接模式，无需公网 IP
- 官方 Go SDK：`github.com/larksuite/oapi-sdk-go/v3`
- 事件类型：`im.message.receive_v1`（v2.0 版本事件结构）

## Goals / Non-Goals

**Goals:**
1. 建立与飞书开放平台的 WebSocket 长连接
2. 接收并解析飞书消息事件（单聊 + 群聊@提及）
3. 将飞书消息转换为统一消息模型并注入消息路由
4. 支持通过飞书 API 发送回复消息
5. 支持断线自动重连

**Non-Goals:**
1. 暂不支持卡片消息发送（仅预留接口）
2. 暂不支持图片、语音消息接收（MVP 仅文本）
3. 暂不支持多实例集群部署（消息随机分发问题）
4. 暂不支持飞书商店应用（仅企业自建应用）

## Decisions

### D1: 使用官方 SDK 而非自实现 WebSocket

**选择**：使用飞书官方 Go SDK (`larksuite/oapi-sdk-go/v3`)

**理由**：
- SDK 内置认证、加密、心跳、重连逻辑
- 提供 typed 事件结构，减少解析错误
- 社区维护，API 变更有升级支持

**替代方案**：
- 自实现 WebSocket：工作量大，需要处理飞书特有协议
- 使用第三方库：缺乏飞书特定支持

### D2: 接口抽象层隔离 SDK

**选择**：定义 `FeishuClient` 接口，SDK 实现作为私有依赖

```
┌─────────────────┐     ┌──────────────────────┐
│  FeishuClient   │     │  feishuSDKClient     │
│  (interface)    │◄────│  (private impl)      │
└─────────────────┘     └──────────────────────┘
        │                        │
        ▼                        ▼
┌─────────────────┐     ┌──────────────────────┐
│  core.Router    │     │  larksuite SDK       │
└─────────────────┘     └──────────────────────┘
```

**理由**：
- 便于单元测试，可 mock SDK 行为
- SDK 升级时变更范围可控
- 未来可支持其他平台（钉钉、Slack）复用架构

### D3: 消息处理异步化

**选择**：事件接收后立即返回，业务逻辑通过 channel 异步处理

```
飞书事件 ──► OnMessage() ──► eventChan ──► worker ──► Router.Route()
                │
                ▼
           立即返回 nil（ACK）
```

**理由**：
- 飞书要求 3 秒内响应，否则触发重推
- AI 代理响应可能耗时较长
- 异步处理允许失败重试而不影响 ACK

### D4: MVP 仅支持文本消息

**选择**：MVP 阶段仅实现 `text` 类型消息处理

**理由**：
- 文本消息覆盖 90% 的 AI 交互场景
- 图片、语音需要额外处理流程（下载、STT 等）
- 符合垂直切片原则，快速验证主链路

### D5: 群聊仅处理 @提及消息

**选择**：使用 `im:message.group_at_msg` 权限，仅接收 @机器人消息

**理由**：
- `im:message.group_msg` 是敏感权限，需要飞书审核
- @提及 模式符合机器人交互习惯
- MVP 阶段避免权限审核延迟

## Architecture

### 包结构

```
internal/platform/feishu/
├── client.go           # FeishuClient 接口定义
├── client_impl.go      # SDK 客户端实现
├── client_impl_test.go # 测试（使用 mock）
├── event_parser.go     # 事件解析器
├── event_parser_test.go
├── message_converter.go # 消息转换器
├── message_converter_test.go
├── sender.go           # 消息发送器
├── sender_test.go
└── mock_client.go      # Mock 实现（供其他包测试使用）
```

### 核心接口

```go
// FeishuClient 飞书客户端接口
type FeishuClient interface {
    // Connect 建立长连接（阻塞）
    Connect(ctx context.Context) error

    // Disconnect 断开连接
    Disconnect() error

    // IsConnected 检查连接状态
    IsConnected() bool

    // SendText 发送文本消息
    SendText(ctx context.Context, chatID, content string) error

    // OnEvent 注册事件回调
    OnEvent(handler EventHandler)
}

// EventHandler 事件处理器
type EventHandler func(ctx context.Context, event *MessageReceiveEvent) error

// MessageReceiveEvent 解析后的事件结构
type MessageReceiveEvent struct {
    EventID     string
    MessageType string    // "text", "post", "image", etc.
    Content     string    // JSON 字符串
    ChatID      string
    ChatType    string    // "p2p", "group"
    Sender      SenderInfo
    Mentions    []MentionInfo
    CreateTime  time.Time
    RawEvent    any       // 原始 SDK 事件（用于调试）
}
```

### 状态流转

```
         ┌───────────────┐
         │   Disconnected │
         └───────┬───────┘
                 │ Connect()
                 ▼
         ┌───────────────┐
    ┌───►│  Connecting   │───┐
    │    └───────┬───────┘   │ 认证失败
    │            │ 认证成功  │
    │            ▼           ▼
    │    ┌───────────────┐ ┌───────────────┐
    │    │  Connected    │ │     Error     │
    │    └───────┬───────┘ └───────────────┘
    │            │ 断开
    │            ▼
    │    ┌───────────────┐
    │    │ Reconnecting  │
    └────┴───────────────┘
```

### 消息流程

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  飞书服务器  │────►│ FeishuClient│────►│EventParser  │────►│   Router    │
│  (WebSocket)│     │  (SDK)      │     │ (转换)       │     │  (路由)     │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
                                                                  │
                          ┌───────────────────────────────────────┘
                          ▼
                   ┌─────────────┐     ┌─────────────┐
                   │   Agent     │────►│  Sender     │
                   │ (AI 代理)   │     │ (回复飞书)   │
                   └─────────────┘     └─────────────┘
```

## Risks / Trade-offs

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| SDK 版本 API 变更 | 编译失败 | 接口抽象层隔离；锁定 SDK 版本 |
| 3秒超时限制 | 消息重推 | 异步处理 + 快速 ACK |
| 集群模式消息随机 | 多实例消息丢失 | MVP 单实例；后续加消息队列 |
| 群消息权限审核 | 功能受限 | 使用 @提及 权限规避 |
| WebSocket 不稳定 | 断线期间消息丢失 | SDK 内置重连；关键消息持久化（后续） |

## Migration Plan

### 部署步骤

1. 确保飞书应用已创建，获取 APP_ID 和 APP_SECRET
2. 配置 `config.toml` 中的 `[projects.feishu]` 部分
3. 启动 cc-connect，验证 WebSocket 连接成功
4. 在飞书开放平台配置事件订阅（选择"使用长连接接收事件"）
5. 添加 `im.message.receive_v1` 事件订阅

### 回滚策略

1. 在飞书开放平台取消事件订阅
2. 停止 cc-connect 进程
3. 配置 `enabled = false` 禁用飞书平台

## Open Questions

1. **消息持久化**：是否需要在断线期间持久化消息？
   - 当前决策：MVP 不实现，后续阶段考虑

2. **多机器人支持**：是否支持同一应用多机器人？
   - 当前决策：MVP 单机器人，架构预留扩展点

3. **卡片消息**：何时支持交互式卡片？
   - 当前决策：阶段 4 斜杠命令系统时考虑
