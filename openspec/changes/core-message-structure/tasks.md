# 统一消息结构 - 实现任务

## 1. 项目结构初始化

- [ ] 1.1 创建 `internal/core/` 目录结构
  - 影响目录：`internal/core/`
  - 验证：目录存在且包含 `.gitkeep` 或初始文件

## 2. TDD 第一轮：消息类型定义（🔴 红阶段）

- [ ] 2.1 编写 `MessageType` 常量测试
  - 文件：`internal/core/message_test.go`
  - 测试内容：验证四种类型常量存在且值正确
  - 验证：`go test ./internal/core/... -run TestMessageType -v`（预期失败）

- [ ] 2.2 编写 `Message` 结构体测试
  - 文件：`internal/core/message_test.go`
  - 测试内容：验证结构体字段存在、JSON tag 正确
  - 验证：`go test ./internal/core/... -run TestMessageStruct -v`（预期失败）

## 3. TDD 第一轮：最小实现（🟢 绿阶段）

- [ ] 3.1 实现 `MessageType` 常量
  - 文件：`internal/core/message.go`
  - 内容：定义 `MessageType` 类型和四个常量
  - 验证：`go test ./internal/core/... -run TestMessageType -v`（预期通过）

- [ ] 3.2 实现 `Message` 结构体
  - 文件：`internal/core/message.go`
  - 内容：定义结构体及 JSON tag
  - 验证：`go test ./internal/core/... -run TestMessageStruct -v`（预期通过）

## 4. TDD 第二轮：消息构造函数（🔴 红阶段）

- [ ] 4.1 编写通用构造函数测试 `NewMessage`
  - 文件：`internal/core/message_test.go`
  - 测试内容：验证字段设置、ID 和 Timestamp 自动生成
  - 验证：`go test ./internal/core/... -run TestNewMessage -v`（预期失败）

- [ ] 4.2 编写便捷构造函数测试
  - 文件：`internal/core/message_test.go`
  - 测试内容：`NewTextMessage`、`NewVoiceMessage`、`NewImageMessage`、`NewCommandMessage`
  - 验证：`go test ./internal/core/... -run TestNew.*Message -v`（预期失败）

- [ ] 4.3 编写 ID 唯一性测试
  - 文件：`internal/core/message_test.go`
  - 测试内容：连续创建 1000 个消息，验证 ID 唯一
  - 验证：`go test ./internal/core/... -run TestMessageIDUnique -v`（预期失败）

## 5. TDD 第二轮：构造函数实现（🟢 绿阶段）

- [ ] 5.1 实现 ID 生成函数 `GenerateMessageID`
  - 文件：`internal/core/message_id.go`
  - 内容：时间戳 + 随机数方案
  - 验证：`go test ./internal/core/... -run TestMessageIDUnique -v`（预期通过）

- [ ] 5.2 实现 `NewMessage` 通用构造函数
  - 文件：`internal/core/message.go`
  - 依赖：任务 5.1
  - 验证：`go test ./internal/core/... -run TestNewMessage -v`（预期通过）

- [ ] 5.3 实现四个便捷构造函数
  - 文件：`internal/core/message.go`
  - 依赖：任务 5.2
  - 验证：`go test ./internal/core/... -run TestNew.*Message -v`（预期通过）

## 6. TDD 第三轮：序列化（🔴 红阶段）

- [ ] 6.1 编写 JSON 序列化测试 `ToJSON`
  - 文件：`internal/core/message_test.go`
  - 测试内容：序列化结果正确、包含所有字段、snake_case 命名
  - 验证：`go test ./internal/core/... -run TestMessageToJSON -v`（预期失败）

- [ ] 6.2 编写 JSON 反序列化测试 `FromJSON`
  - 文件：`internal/core/message_test.go`
  - 测试内容：有效 JSON 反序列化、忽略未知字段、无效 JSON 错误处理
  - 验证：`go test ./internal/core/... -run TestMessageFromJSON -v`（预期失败）

- [ ] 6.3 编写往返一致性测试
  - 文件：`internal/core/message_test.go`
  - 测试内容：ToJSON → FromJSON 后数据一致
  - 验证：`go test ./internal/core/... -run TestMessageRoundTrip -v`（预期失败）

- [ ] 6.4 编写边界情况测试
  - 文件：`internal/core/message_test.go`
  - 测试内容：空内容、Unicode 字符、特殊字符
  - 验证：`go test ./internal/core/... -run TestMessageEdge -v`（预期失败）

## 7. TDD 第三轮：序列化实现（🟢 绿阶段）

- [ ] 7.1 实现 `ToJSON` 方法
  - 文件：`internal/core/message.go`
  - 验证：`go test ./internal/core/... -run TestMessageToJSON -v`（预期通过）

- [ ] 7.2 实现 `FromJSON` 函数
  - 文件：`internal/core/message.go`
  - 验证：`go test ./internal/core/... -run TestMessageFromJSON -v`（预期通过）

- [ ] 7.3 确保往返一致性
  - 验证：`go test ./internal/core/... -run TestMessageRoundTrip -v`（预期通过）

- [ ] 7.4 处理边界情况
  - 验证：`go test ./internal/core/... -run TestMessageEdge -v`（预期通过）

## 8. TDD 第四轮：重构与完善（🔵 重构阶段）

- [ ] 8.1 代码审查与重构
  - 检查代码风格、命名规范
  - 消除重复代码
  - 添加必要的注释

- [ ] 8.2 验证测试覆盖率
  - 命令：`go test ./internal/core/... -cover`
  - 目标：覆盖率 > 85%
  - 如不达标，补充测试用例

## 9. 验收

- [ ] 9.1 运行完整测试套件
  - 命令：`go test ./internal/core/... -v -cover`
  - 要求：所有测试通过，覆盖率 > 85%

- [ ] 9.2 代码格式化
  - 命令：`go fmt ./internal/core/...`

- [ ] 9.3 静态检查
  - 命令：`go vet ./internal/core/...`

---

## 任务依赖关系

```
1.1 ──▶ 2.1 ──▶ 3.1
    ──▶ 2.2 ──▶ 3.2
              │
              ▼
        4.1 ──▶ 5.2 ──▶ 5.3
        4.2 ────────────▶ 5.3
        4.3 ──▶ 5.1 ──▶ 5.2
              │
              ▼
        6.1 ──▶ 7.1
        6.2 ──▶ 7.2 ──▶ 7.3
        6.3 ────────────▶ 7.3
        6.4 ────────────▶ 7.4
              │
              ▼
             8.1 ──▶ 8.2
              │
              ▼
        9.1 ──▶ 9.2 ──▶ 9.3
```

**可并行任务**：
- 2.1 与 2.2 可并行
- 4.1、4.2、4.3 可并行
- 6.1、6.2、6.3、6.4 可并行
- 9.2 与 9.3 可并行
