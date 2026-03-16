## 1. 命令类型和解析器（TDD: 先测试后实现）

- [x] 1.1 创建 `internal/core/command/types.go`，定义 `Command` 和 `CommandResult` 结构体
- [x] 1.2 编写 `parser_test.go` 测试用例：`IsCommand` 函数的各种输入场景
- [x] 1.3 实现 `parser.go` 中的 `IsCommand` 函数，使测试通过
- [x] 1.4 编写 `Parse` 函数的测试用例：无参数、单参数、多参数、多空格场景
- [x] 1.5 实现 `Parse` 函数，使测试通过
- [x] 1.6 运行 `go test ./internal/core/command/... -v` 验证覆盖率 > 85%

**验证命令**: `go test ./internal/core/command/... -v -cover`

---

## 2. 命令执行器基础（TDD）

- [x] 2.1 创建 `executor.go`，定义 `Executor` 结构体（持有 Agent 和 SessionManager）
- [x] 2.2 编写 `executor_test.go`：使用 MockAgent 测试 `/help` 命令
- [x] 2.3 实现 `handleHelp` 处理器，使测试通过
- [x] 2.4 编写测试：未知命令返回错误提示
- [x] 2.5 实现默认处理器（switch default 分支）

**验证命令**: `go test ./internal/core/command/... -v -cover`

---

## 3. /mode 命令实现（TDD）

- [x] 3.1 编写测试：`/mode yolo` 调用 `Agent.SetPermissionMode(bypassPermissions)`
- [x] 3.2 编写测试：`/mode edit` 调用 `Agent.SetPermissionMode(acceptEdits)`
- [x] 3.3 编写测试：`/mode plan` 调用 `Agent.SetPermissionMode(plan)`
- [x] 3.4 编写测试：`/mode default` 调用 `Agent.SetPermissionMode(default)`
- [x] 3.5 编写测试：`/mode` 无参数返回当前模式
- [x] 3.6 编写测试：`/mode invalid` 返回错误消息
- [x] 3.7 实现 `handleMode` 处理器，使所有测试通过

**验证命令**: `go test ./internal/core/command/... -v -run TestMode -cover`

---

## 4. /new 命令实现（TDD）

- [x] 4.1 编写测试：`/new` 创建新会话
- [x] 4.2 编写测试：`/new my-session` 创建命名会话
- [x] 4.3 实现 `handleNew` 处理器

**验证命令**: `go test ./internal/core/command/... -v -run TestNew -cover`

---

## 5. /list 命令实现（TDD）

- [x] 5.1 编写测试：多个会话时 `/list` 返回会话列表
- [x] 5.2 编写测试：无会话时 `/list` 返回空列表消息
- [x] 5.3 实现 `handleList` 处理器

**验证命令**: `go test ./internal/core/command/... -v -run TestList -cover`

---

## 6. /stop 命令实现（TDD）

- [x] 6.1 编写测试：Agent 运行时 `/stop` 调用 `Agent.Stop()`
- [x] 6.2 编写测试：Agent 未运行时 `/stop` 返回提示消息
- [x] 6.3 实现 `handleStop` 处理器

**验证命令**: `go test ./internal/core/command/... -v -run TestStop -cover`

---

## 7. Feishu 适配器集成

- [x] 7.1 编写 `adapter_test.go`：测试 `/mode yolo` 文本被转为 `MessageTypeCommand`
- [x] 7.2 编写测试：普通文本消息保持为 `MessageTypeText`
- [x] 7.3 修改 `adapter.go`，在消息处理中添加命令检测逻辑
- [x] 7.4 编写集成测试：命令执行结果通过 Router 返回

**验证命令**: `go test ./internal/platform/feishu/... -v -cover`

---

## 8. 覆盖率验证和重构

- [x] 8.1 运行完整测试套件：`go test ./... -coverprofile=coverage.out`
- [x] 8.2 检查覆盖率报告，确保 `internal/core/command/` 覆盖率 > 85%
- [x] 8.3 如有必要，补充边界测试用例
- [x] 8.4 代码重构：提取公共逻辑，消除重复

**验证命令**: `go tool cover -html=coverage.out`

---

## 9. 文档和验收

- [x] 9.1 更新 `CLAUDE.md` 中的斜杠命令说明（如有必要）
- [x] 9.2 运行完整测试验证所有功能：`go test ./...`
- [x] 9.3 手动验收：确认 5 个命令的行为符合 spec

---

## 任务依赖关系

```
1.1 ──▶ 1.2 ──▶ 1.3 ──▶ 1.4 ──▶ 1.5 ──▶ 1.6
                                        │
                                        ▼
2.1 ──▶ 2.2 ──▶ 2.3 ──▶ 2.4 ──▶ 2.5
        │
        ▼
3.1 ──▶ 3.2 ──▶ 3.3 ──▶ 3.4 ──▶ 3.5 ──▶ 3.6 ──▶ 3.7
                                        │
        ┌───────────────────────────────┤
        ▼                               ▼
4.1 ──▶ 4.2 ──▶ 4.3            6.1 ──▶ 6.2 ──▶ 6.3
        │                               │
        ▼                               │
5.1 ──▶ 5.2 ──▶ 5.3                     │
        │                               │
        └───────────────┬───────────────┘
                        ▼
                    7.1 ──▶ 7.2 ──▶ 7.3 ──▶ 7.4
                                                │
                                                ▼
                                            8.1 ──▶ 8.2 ──▶ 8.3 ──▶ 8.4
                                                                        │
                                                                        ▼
                                                                    9.1 ──▶ 9.2 ──▶ 9.3
```

**可并行任务**：
- 任务 3、4、5、6 可以在任务 2 完成后并行进行（各自独立的命令处理器）
