# 统一消息结构

## Why

cc-connect 需要在不同聊天平台（飞书等）和 AI 代理（Claude Code 等）之间传递消息。当前项目刚起步，缺乏统一的消息模型，导致后续开发无法进行。统一消息结构是整个系统的基石，必须首先实现。

## What Changes

- 新增 `internal/core/message.go`：定义统一的消息结构体
- 新增 `internal/core/message_test.go`：TDD 测试文件
- 支持四种消息类型：`text`、`voice`、`image`、`command`
- 提供消息创建、序列化、反序列化能力
- 字段语义与 Python 版保持一致：`id`、`platform`、`user_id`、`content`、`type`、`timestamp`

## Capabilities

### New Capabilities

- `unified-message`: 统一消息模型，支持四种消息类型的创建、序列化与反序列化

### Modified Capabilities

无（这是项目的第一个能力）

## Impact

### 影响模块

- **core**: 新增 `internal/core/` 目录及消息相关代码

### 影响范围

- 属于 **阶段 1：核心消息系统** 的第一个切片
- 后续所有模块（飞书适配器、Claude Code 适配器、命令系统）都依赖此消息结构

### 技术约束

- 不引入外部依赖，使用 Go 标准库
- 序列化格式使用 JSON
- 字段命名使用 snake_case（与 Python 版语义一致，JSON tag 控制）

### 验收标准

- ✅ 支持创建 `text`、`voice`、`image`、`command` 四种类型的消息
- ✅ 每种类型有便捷构造函数（如 `NewTextMessage()`）
- ✅ 消息可序列化为 JSON，并可从 JSON 反序列化
- ✅ 反序列化时对未知字段具有容错性
- ✅ 测试覆盖率 > 85%
- ✅ 无外部依赖

### 风险与缓解

| 风险 | 缓解措施 |
|------|----------|
| 字段命名与 Go 惯例冲突 | 使用 JSON tag 映射到 snake_case |
| 消息 ID 生成策略不明确 | 使用 UUID v4 或时间戳+随机数 |
| 语音/图片内容格式未定 | 当前仅存储内容字符串，后续扩展为结构体 |
