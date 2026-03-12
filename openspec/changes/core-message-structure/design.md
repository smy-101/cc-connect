# 统一消息结构 - 技术设计

## Context

cc-connect 是一个从零开始的 Go 项目，需要在聊天平台和 AI 代理之间传递消息。当前没有任何代码实现，统一消息结构是整个系统的第一个组件，位于核心域（core domain）。

**约束条件**：
- 语言版本：Go 1.25.x
- 不引入外部依赖
- 字段语义与 Python 版一致（snake_case）
- 测试覆盖率 > 85%

## Goals / Non-Goals

**Goals:**
- 定义统一的消息结构体，支持四种类型
- 提供便捷的消息创建方法
- 实现 JSON 序列化/反序列化
- 完全可测试，无需外部依赖

**Non-Goals:**
- 不实现消息路由（后续切片）
- 不实现会话管理（后续切片）
- 不实现消息持久化
- 不引入 gRPC/Protobuf 等复杂序列化方案

## Decisions

### 1. 包结构

```
internal/core/
├── message.go          # Message 结构体及核心方法
├── message_test.go     # TDD 测试文件
└── message_id.go       # ID 生成逻辑
```

**理由**：保持简单，按职责拆分文件。`message.go` 包含核心结构，`message_id.go` 处理 ID 生成逻辑，便于独立测试。

### 2. 核心类型定义

```go
// MessageType 消息类型枚举
type MessageType string

const (
    MessageTypeText    MessageType = "text"
    MessageTypeVoice   MessageType = "voice"
    MessageTypeImage   MessageType = "image"
    MessageTypeCommand MessageType = "command"
)

// Message 统一消息结构
type Message struct {
    ID        string      `json:"id"`
    Platform  string      `json:"platform"`
    UserID    string      `json:"user_id"`
    Content   string      `json:"content"`
    Type      MessageType `json:"type"`
    Timestamp time.Time   `json:"timestamp"`
}
```

**理由**：
- 使用 `string` 类型别名定义 `MessageType`，简单且 JSON 友好
- JSON tag 使用 snake_case，与 Python 版字段语义一致
- 不使用指针类型，保持结构体简单

**替代方案**：
- 使用 `int` 枚举 → JSON 序列化不友好，放弃
- 使用 `iota` → 同上，放弃

### 3. 消息 ID 生成

```go
// GenerateMessageID 生成唯一消息 ID
// 格式: <unix_nano>_<random_8chars>
// 示例: 1709234567890123456_a1b2c3d4
func GenerateMessageID() string {
    ts := time.Now().UnixNano()
    randBytes := make([]byte, 4)
    rand.Read(randBytes)
    return fmt.Sprintf("%d_%x", ts, randBytes)
}
```

**理由**：
- 不引入外部 UUID 库，保持零依赖
- 时间戳 + 随机数组合，碰撞概率极低
- 可读性好，便于调试

**替代方案**：
- UUID v4 → 需要引入 `github.com/google/uuid`，增加依赖
- 纯随机 → 无序，不利于排序和调试
- 纯时间戳 → 高并发下可能冲突

### 4. 构造函数设计

```go
// NewMessage 创建新消息（通用）
func NewMessage(platform, userID, content string, msgType MessageType) *Message

// NewTextMessage 创建文本消息（便捷方法）
func NewTextMessage(platform, userID, content string) *Message

// NewVoiceMessage 创建语音消息
func NewVoiceMessage(platform, userID, content string) *Message

// NewImageMessage 创建图片消息
func NewImageMessage(platform, userID, content string) *Message

// NewCommandMessage 创建命令消息
func NewCommandMessage(platform, userID, content string) *Message
```

**理由**：
- 提供通用构造函数和类型特定的便捷方法
- 返回指针避免结构体复制
- ID 和 Timestamp 在构造时自动生成

### 5. 序列化设计

```go
// ToJSON 序列化为 JSON 字节流
func (m *Message) ToJSON() ([]byte, error)

// FromJSON 从 JSON 字节流反序列化
func FromJSON(data []byte) (*Message, error)
```

**理由**：
- 使用 Go 标准库 `encoding/json`
- `FromJSON` 为包级函数，符合 Go 惯例
- 反序列化时忽略未知字段（JSON decoder 默认行为）

## Risks / Trade-offs

| 风险 | 缓解措施 |
|------|----------|
| ID 生成在高并发下可能冲突 | 时间戳精度为纳秒级 + 4 字节随机数，碰撞概率 < 10^-15 |
| 语音/图片内容当前仅为字符串 | 设计为后续扩展预留：Content 可改为 interface{} 或增加 MediaURL 字段 |
| 无消息校验 | 在后续路由层添加校验逻辑，当前保持核心简单 |
| JSON 序列化性能 | 当前消息量不大，标准库足够；后续可替换为 json-iterator |

## Package 边界

```
┌─────────────────────────────────────────────────────┐
│                    internal/core                     │
│  ┌─────────────────────────────────────────────┐   │
│  │              Message (核心域)                │   │
│  │  - 类型定义                                  │   │
│  │  - 构造函数                                  │   │
│  │  - 序列化                                    │   │
│  └─────────────────────────────────────────────┘   │
│                      │                              │
│                      ▼                              │
│  ┌─────────────────────────────────────────────┐   │
│  │          未来扩展（不在本次范围）            │   │
│  │  - Router (消息路由)                         │   │
│  │  - Session (会话管理)                        │   │
│  │  - Config (配置管理)                         │   │
│  └─────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
          │                              │
          ▼                              ▼
┌──────────────────┐          ┌──────────────────┐
│ platform/feishu  │          │ agent/claudecode │
│ (适配层)         │          │ (适配层)         │
└──────────────────┘          └──────────────────┘
```

## 测试策略

**TDD 循环**：
1. 🔴 编写测试用例（`message_test.go`）
2. 🟢 实现最小代码使测试通过
3. 🔵 重构优化

**测试覆盖**：
- 消息创建：四种类型各一个测试
- ID 生成：唯一性、格式验证
- 序列化：正向序列化、反序列化、往返一致性
- 边界情况：空内容、特殊字符、未知类型处理

**命令**：
```bash
go test ./internal/core/... -v -cover
```
