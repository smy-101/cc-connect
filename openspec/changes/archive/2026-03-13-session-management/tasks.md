# 会话管理 - 实现任务

## 1. TDD 第一轮：SessionID 派生（🔴 红阶段）

- [x] 1.1 编写 `DeriveSessionID` 测试
  - 文件：`internal/core/session_test.go`
  - 测试内容：私聊、群聊、群聊优先的会话 ID 派生
  - 验证：`go test ./internal/core/... -run TestDeriveSessionID -v`（预期失败）

## 2. TDD 第一轮：Message 结构扩展（🔴 红阶段）

- [x] 2.1 编写 Message ChannelID 字段测试
  - 文件：`internal/core/message_test.go`
  - 测试内容：ChannelID 字段存在、JSON 序列化/反序列化、可选性
  - 验证：`go test ./internal/core/... -run TestMessageChannelID -v`（预期失败）

## 3. TDD 第一轮：最小实现（🟢 绿阶段）

- [x] 3.1 实现 Message.ChannelID 字段
  - 文件：`internal/core/message.go`
  - 内容：添加 `ChannelID string` 字段，更新 JSON tag
  - 验证：`go test ./internal/core/... -run TestMessageChannelID -v`（预期通过）

- [x] 3.2 实现 `DeriveSessionID` 函数
  - 文件：`internal/core/session.go`
  - 内容：实现会话 ID 派生逻辑（platform:type:identifier 格式）
  - 验证：`go test ./internal/core/... -run TestDeriveSessionID -v`（预期通过）

## 4. TDD 第二轮：Session 结构和状态（🔴 红阶段）

- [x] 4.1 编写 Session 结构测试
  - 文件：`internal/core/session_test.go`
  - 测试内容：Session 字段存在、SessionStatus 枚举、时间字段
  - 验证：`go test ./internal/core/... -run TestSession -v`（预期失败）

- [x] 4.2 编写 Session 方法测试
  - 文件：`internal/core/session_test.go`
  - 测试内容：`BindAgent`、`SetPermissionMode`、`SetMetadata`、`Touch`、`Clone`
  - 验证：`go test ./internal/core/... -run TestSessionMethods -v`（预期失败）

## 5. TDD 第二轮：Session 实现（🟢 绿阶段）

- [x] 5.1 实现 Session 结构和常量
  - 文件：`internal/core/session.go`
  - 内容：`SessionID`、`SessionStatus`、`Session` 结构体、`NewSession` 构造函数
  - 验证：`go test ./internal/core/... -run TestSession -v`（预期通过）

- [x] 5.2 实现 Session 方法
  - 文件：`internal/core/session.go`
  - 内容：`BindAgent`、`SetPermissionMode`、`SetMetadata`、`Touch`、`Clone`
  - 验证：`go test ./internal/core/... -run TestSessionMethods -v`（预期通过）

## 6. TDD 第三轮：SessionManager 基础（🔴 红阶段）

- [x] 6.1 编写 SessionManager 创建测试
  - 文件：`internal/core/session_test.go`
  - 测试内容：`NewSessionManager`、初始状态为空、配置默认值
  - 验证：`go test ./internal/core/... -run TestNewSessionManager -v`（预期失败）

- [x] 6.2 编写 `GetOrCreate` 测试
  - 文件：`internal/core/session_test.go`
  - 测试内容：自动创建、返回已存在会话、返回副本
  - 验证：`go test ./internal/core/... -run TestGetOrCreate -v`（预期失败）

- [x] 6.3 编写 `Get` 测试
  - 文件：`internal/core/session_test.go`
  - 测试内容：获取存在/不存在的会话、返回副本
  - 验证：`go test ./internal/core/... -run TestSessionManagerGet -v`（预期失败）

## 7. TDD 第三轮：SessionManager 实现（🟢 绿阶段）

- [x] 7.1 实现 `SessionConfig` 和 `NewSessionManager`
  - 文件：`internal/core/session.go`
  - 内容：`SessionConfig` 结构、默认配置、`NewSessionManager` 构造函数
  - 验证：`go test ./internal/core/... -run TestNewSessionManager -v`（预期通过）

- [x] 7.2 实现 `GetOrCreate` 方法
  - 文件：`internal/core/session.go`
  - 内容：检查存在、自动创建、返回副本
  - 验证：`go test ./internal/core/... -run TestGetOrCreate -v`（预期通过）

- [x] 7.3 实现 `Get` 方法
  - 文件：`internal/core/session.go`
  - 内容：获取会话、返回副本、不存在返回 nil
  - 验证：`go test ./internal/core/... -run TestSessionManagerGet -v`（预期通过）

## 8. TDD 第四轮：生命周期管理（🔴 红阶段）

- [x] 8.1 编写 `Archive` 测试
  - 文件：`internal/core/session_test.go`
  - 测试内容：归档 active 会话、归档不存在会话、状态转换、ArchivedAt 时间
  - 验证：`go test ./internal/core/... -run TestSessionArchive -v`（预期失败）

- [x] 8.2 编写 `Destroy` 测试
  - 文件：`internal/core/session_test.go`
  - 测试内容：销毁存在/不存在会话、资源释放
  - 验证：`go test ./internal/core/... -run TestSessionDestroy -v`（预期失败）

- [x] 8.3 编写状态不可逆测试
  - 文件：`internal/core/session_test.go`
  - 测试内容：archived 状态无法转换回 active
  - 验证：`go test ./internal/core/... -run TestSessionStateTransition -v`（预期失败）

## 9. TDD 第四轮：生命周期实现（🟢 绿阶段）

- [x] 9.1 实现 `Archive` 方法
  - 文件：`internal/core/session.go`
  - 内容：状态检查、状态转换、设置 ArchivedAt、返回错误
  - 验证：`go test ./internal/core/... -run TestSessionArchive -v`（预期通过）

- [x] 9.2 实现 `Destroy` 方法
  - 文件：`internal/core/session.go`
  - 内容：从 map 中删除会话
  - 验证：`go test ./internal/core/... -run TestSessionDestroy -v`（预期通过）

- [x] 9.3 实现状态转换保护
  - 文件：`internal/core/session.go`
  - 内容：在 Archive 中检查当前状态，拒绝非法转换
  - 验证：`go test ./internal/core/... -run TestSessionStateTransition -v`（预期通过）

## 10. TDD 第五轮：自动清理（🔴 红阶段）

- [x] 10.1 编写 `cleanup` 测试
  - 文件：`internal/core/session_test.go`
  - 测试内容：active 超时归档、archived 超时销毁、使用 mock 时间
  - 验证：`go test ./internal/core/... -run TestSessionCleanup -v`（预期失败）

- [x] 10.2 编写 `StartCleanup` 测试
  - 文件：`internal/core/session_test.go`
  - 测试内容：定时执行、context 取消退出
  - 验证：`go test ./internal/core/... -run TestStartCleanup -v`（预期失败）

## 11. TDD 第五轮：自动清理实现（🟢 绿阶段）

- [x] 11.1 实现 `cleanup` 私有方法
  - 文件：`internal/core/session.go`
  - 内容：遍历会话、检查超时、执行归档/销毁
  - 验证：`go test ./internal/core/... -run TestSessionCleanup -v`（预期通过）

- [x] 11.2 实现 `StartCleanup` 方法
  - 文件：`internal/core/session.go`
  - 内容：启动 goroutine、ticker 定时执行、context 取消退出
  - 验证：`go test ./internal/core/... -run TestStartCleanup -v`（预期通过）

## 12. TDD 第六轮：并发安全（🔴 红阶段）

- [x] 12.1 编写并发 GetOrCreate 测试
  - 文件：`internal/core/session_test.go`
  - 测试内容：多个 goroutine 同时获取同一会话，只创建一个
  - 验证：`go test ./internal/core/... -run TestConcurrentGetOrCreate -race -v`（预期失败）

- [x] 12.2 编写并发读写测试
  - 文件：`internal/core/session_test.go`
  - 测试内容：同时修改和读取会话，无竞态条件
  - 验证：`go test ./internal/core/... -run TestConcurrentReadWrite -race -v`（预期失败）

- [x] 12.3 编写并发清理和访问测试
  - 文件：`internal/core/session_test.go`
  - 测试内容：清理时访问会话，无竞态条件
  - 验证：`go test ./internal/core/... -run TestConcurrentCleanup -race -v`（预期失败）

## 13. TDD 第六轮：并发安全实现（🟢 绿阶段）

- [x] 13.1 添加 sync.RWMutex 保护
  - 文件：`internal/core/session.go`
  - 内容：在 Get、GetOrCreate、Archive、Destroy、cleanup 中使用锁
  - 验证：`go test ./internal/core/... -race -v`（预期通过，无竞态条件）

## 14. TDD 第七轮：Router 集成（🔴 红阶段）

- [x] 14.1 编写 `RouteWithSession` 测试
  - 文件：`internal/core/router_test.go`
  - 测试内容：自动派生 SessionID、自动获取/创建会话、传递会话给处理器
  - 验证：`go test ./internal/core/... -run TestRouteWithSession -v`（预期失败）

- [x] 14.2 编写 SessionHandler 类型测试
  - 文件：`internal/core/router_test.go`
  - 测试内容：新签名处理器、会话上下文传递
  - 验证：`go test ./internal/core/... -run TestSessionHandler -v`（预期失败）

## 15. TDD 第七轮：Router 集成实现（🟢 绿阶段）

- [x] 15.1 实现 `SessionHandler` 类型
  - 文件：`internal/core/router.go`
  - 内容：定义 `SessionHandler func(ctx context.Context, msg *Message, session *Session) error`
  - 验证：`go test ./internal/core/... -run TestSessionHandler -v`（预期通过）

- [x] 15.2 实现 `RouteWithSession` 方法
  - 文件：`internal/core/router.go`
  - 内容：派生 SessionID、获取会话、更新活跃时间、调用处理器
  - 验证：`go test ./internal/core/... -run TestRouteWithSession -v`（预期通过）

- [x] 15.3 扩展 Router 持有 SessionManager
  - 文件：`internal/core/router.go`
  - 内容：Router 结构体添加 sessions 字段、NewRouter 初始化
  - 验证：`go test ./internal/core/... -run TestRouteWithSession -v`（预期通过）

## 16. TDD 第八轮：可配置时间函数（🔴 红阶段 → 🟢 绿阶段）

- [x] 16.1 编写时间函数注入测试
  - 文件：`internal/core/session_test.go`
  - 测试内容：使用 mock 时间、验证时间相关操作使用注入函数
  - 验证：`go test ./internal/core/... -run TestMockTime -v`

- [x] 16.2 实现时间函数注入
  - 文件：`internal/core/session.go`
  - 内容：SessionManager 添加 `now func() time.Time` 字段、默认使用 time.Now
  - 验证：`go test ./internal/core/... -run TestMockTime -v`（预期通过）

## 17. TDD 第九轮：重构与完善（🔵 重构阶段）

- [x] 17.1 代码审查与重构
  - 检查代码风格、命名规范
  - 消除重复代码
  - 添加必要的注释和文档注释

- [x] 17.2 验证测试覆盖率
  - 命令：`go test ./internal/core/... -cover`
  - 目标：覆盖率 > 85%
  - 结果：覆盖率 96.6%

## 18. 验收

- [x] 18.1 运行完整测试套件
  - 命令：`go test ./internal/core/... -v -cover`
  - 要求：所有测试通过，覆盖率 > 85%
  - 结果：✓ 所有测试通过，覆盖率 96.6%

- [x] 18.2 运行竞态检测
  - 命令：`go test ./internal/core/... -race`
  - 要求：无竞态条件
  - 结果：✓ 无竞态条件

- [x] 18.3 代码格式化
  - 命令：`go fmt ./internal/core/...`
  - 结果：✓ 已格式化

- [x] 18.4 静态检查
  - 命令：`go vet ./internal/core/...`
  - 结果：✓ 无问题

---

## 任务依赖关系

```
1.1 ──▶ 3.2
          │
2.1 ──▶ 3.1 ──▶ 3.2
                    │
                    ▼
4.1 ──▶ 5.1        │
4.2 ──▶ 5.2        │
                    │
                    ▼
6.1 ──▶ 7.1 ◀──────┘
6.2 ──▶ 7.2
6.3 ──▶ 7.3
          │
          ▼
8.1 ──▶ 9.1
8.2 ──▶ 9.2
8.3 ──▶ 9.3
          │
          ▼
10.1 ──▶ 11.1
10.2 ──▶ 11.2
            │
            ▼
12.1 ──▶ 13.1
12.2 ──▶ 13.1
12.3 ──▶ 13.1
          │
          ▼
14.1 ──▶ 15.1 ──▶ 15.2 ──▶ 15.3
14.2 ──▶ 15.1
          │
          ▼
16.1 ──▶ 16.2
          │
          ▼
17.1 ──▶ 17.2
          │
          ▼
18.1 ──▶ 18.2 ──▶ 18.3 ──▶ 18.4
```

**可并行任务**：
- 1.1 与 2.1 可并行（测试编写）
- 4.1 与 4.2 可并行
- 6.1、6.2、6.3 可并行
- 8.1、8.2、8.3 可并行
- 10.1 与 10.2 可并行
- 12.1、12.2、12.3 可并行
- 14.1 与 14.2 可并行
- 18.3 与 18.4 可并行
