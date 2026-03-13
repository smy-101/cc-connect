# 消息路由 - 实现任务

## 1. TDD 第一轮：错误定义和基础结构（🔴 红阶段）

- [x] 1.1 编写错误定义测试
  - 文件：`internal/core/router_test.go`
  - 测试内容：验证 `ErrNoHandler`、`ErrNilHandler`、`ErrHandlerPanic` 错误存在
  - 验证：`go test ./internal/core/... -run TestRouterErrors -v`（预期失败）

- [x] 1.2 编写 `NewRouter` 测试
  - 文件：`internal/core/router_test.go`
  - 测试内容：验证创建新路由器、初始状态为空
  - 验证：`go test ./internal/core/... -run TestNewRouter -v`（预期失败）

## 2. TDD 第一轮：最小实现（🟢 绿阶段）

- [x] 2.1 实现错误常量
  - 文件：`internal/core/router.go`
  - 内容：定义 `ErrNoHandler`、`ErrNilHandler`、`ErrHandlerPanic`
  - 验证：`go test ./internal/core/... -run TestRouterErrors -v`（预期通过）

- [x] 2.2 实现 `Router` 结构体和 `NewRouter`
  - 文件：`internal/core/router.go`
  - 内容：定义 `Router` 结构体、`Handler` 类型、`NewRouter` 函数
  - 验证：`go test ./internal/core/... -run TestNewRouter -v`（预期通过）

## 3. TDD 第二轮：处理器注册（🔴 红阶段）

- [x] 3.1 编写 `Register` 测试
  - 文件：`internal/core/router_test.go`
  - 测试内容：注册处理器、`HasHandler` 返回正确结果
  - 验证：`go test ./internal/core/... -run TestRouterRegister -v`（预期失败）

- [x] 3.2 编写 `Register` nil 处理器测试
  - 文件：`internal/core/router_test.go`
  - 测试内容：注册 nil 处理器返回 `ErrNilHandler`
  - 验证：`go test ./internal/core/... -run TestRouterRegisterNil -v`（预期失败）

- [x] 3.3 编写 `Unregister` 测试
  - 文件：`internal/core/router_test.go`
  - 测试内容：注销处理器、注销不存在的处理器
  - 验证：`go test ./internal/core/... -run TestRouterUnregister -v`（预期失败）

## 4. TDD 第二轮：注册实现（🟢 绿阶段）

- [x] 4.1 实现 `Register` 方法
  - 文件：`internal/core/router.go`
  - 内容：实现处理器注册，nil 检查
  - 验证：`go test ./internal/core/... -run TestRouterRegister -v`（预期通过）

- [x] 4.2 实现 `HasHandler` 方法
  - 文件：`internal/core/router.go`
  - 内容：检查处理器是否存在
  - 验证：`go test ./internal/core/... -run TestRouterRegister -v`（预期通过）

- [x] 4.3 实现 `Unregister` 方法
  - 文件：`internal/core/router.go`
  - 内容：注销处理器
  - 验证：`go test ./internal/core/... -run TestRouterUnregister -v`（预期通过）

## 5. TDD 第三轮：消息路由（🔴 红阶段）

- [x] 5.1 编写 `Route` 基本测试
  - 文件：`internal/core/router_test.go`
  - 测试内容：路由消息到正确处理器、处理器被调用
  - 验证：`go test ./internal/core/... -run TestRouterRoute -v`（预期失败）

- [x] 5.2 编写 `Route` 无处理器测试
  - 文件：`internal/core/router_test.go`
  - 测试内容：未注册类型的消息返回 `ErrNoHandler`
  - 验证：`go test ./internal/core/... -run TestRouterRouteNoHandler -v`（预期失败）

- [x] 5.3 编写 `Route` 处理器错误测试
  - 文件：`internal/core/router_test.go`
  - 测试内容：处理器返回错误时，`Route` 返回该错误
  - 验证：`go test ./internal/core/... -run TestRouterRouteError -v`（预期失败）

- [x] 5.4 编写 `Route` 处理器 panic 测试
  - 文件：`internal/core/router_test.go`
  - 测试内容：处理器 panic 时，返回 `ErrHandlerPanic`
  - 验证：`go test ./internal/core/... -run TestRouterRoutePanic -v`（预期失败）

## 6. TDD 第三轮：路由实现（🟢 绿阶段）

- [x] 6.1 实现 `Route` 方法（基本功能）
  - 文件：`internal/core/router.go`
  - 内容：查找处理器并调用
  - 验证：`go test ./internal/core/... -run TestRouterRoute -v`（预期通过）

- [x] 6.2 实现无处理器错误处理
  - 文件：`internal/core/router.go`
  - 内容：返回 `ErrNoHandler`
  - 验证：`go test ./internal/core/... -run TestRouterRouteNoHandler -v`（预期通过）

- [x] 6.3 实现 panic 恢复
  - 文件：`internal/core/router.go`
  - 内容：`defer recover` 捕获 panic，返回 `ErrHandlerPanic`
  - 验证：`go test ./internal/core/... -run TestRouterRoutePanic -v`（预期通过）

## 7. TDD 第四轮：并发安全（🔴 红阶段）

- [x] 7.1 编写并发路由测试
  - 文件：`internal/core/router_test.go`
  - 测试内容：多个 goroutine 同时调用 `Route`
  - 验证：`go test ./internal/core/... -run TestRouterConcurrentRoute -race -v`（预期失败）

- [x] 7.2 编写并发注册和路由测试
  - 文件：`internal/core/router_test.go`
  - 测试内容：同时 `Register` 和 `Route`
  - 验证：`go test ./internal/core/... -run TestRouterConcurrentRegister -race -v`（预期失败）

## 8. TDD 第四轮：并发实现（🟢 绿阶段）

- [x] 8.1 添加 `sync.RWMutex` 保护
  - 文件：`internal/core/router.go`
  - 内容：在 `Register`、`Unregister`、`HasHandler`、`Route` 中使用锁
  - 验证：`go test ./internal/core/... -race -v`（预期通过，无竞态条件）

## 9. TDD 第五轮：Context 支持（🔴 红阶段）

- [x] 9.1 编写 Context 取消测试
  - 文件：`internal/core/router_test.go`
  - 测试内容：传入已取消的 context，处理器可感知
  - 验证：`go test ./internal/core/... -run TestRouterContextCancel -v`（预期失败）

## 10. TDD 第五轮：Context 实现（🟢 绿阶段）

- [x] 10.1 确保 Context 正确传递
  - 文件：`internal/core/router.go`
  - 内容：`Route` 方法将 context 传递给处理器
  - 验证：`go test ./internal/core/... -run TestRouterContext -v`（预期通过）

## 11. TDD 第六轮：重构与完善（🔵 重构阶段）

- [x] 11.1 代码审查与重构
  - 检查代码风格、命名规范
  - 消除重复代码
  - 添加必要的注释

- [x] 11.2 验证测试覆盖率
  - 命令：`go test ./internal/core/... -cover`
  - 目标：覆盖率 > 85%
  - 如不达标，补充测试用例

## 12. 验收

- [x] 12.1 运行完整测试套件
  - 命令：`go test ./internal/core/... -v -cover`
  - 要求：所有测试通过，覆盖率 > 85%

- [x] 12.2 运行竞态检测
  - 命令：`go test ./internal/core/... -race`
  - 要求：无竞态条件

- [x] 12.3 代码格式化
  - 命令：`go fmt ./internal/core/...`

- [x] 12.4 静态检查
  - 命令：`go vet ./internal/core/...`

---

## 任务依赖关系

```
1.1 ──▶ 2.1
1.2 ──▶ 2.2

2.1 ──▶ 3.1 ──▶ 4.1 ──▶ 4.2
    ──▶ 3.2 ──▶ 4.1
    ──▶ 3.3 ──▶ 4.3

4.1/4.2/4.3 ──▶ 5.1 ──▶ 6.1
            ──▶ 5.2 ──▶ 6.2
            ──▶ 5.3 ──▶ 6.1
            ──▶ 5.4 ──▶ 6.3

6.1/6.2/6.3 ──▶ 7.1 ──▶ 8.1
            ──▶ 7.2 ──▶ 8.1

8.1 ──▶ 9.1 ──▶ 10.1
              │
              ▼
            11.1 ──▶ 11.2
              │
              ▼
        12.1 ──▶ 12.2 ──▶ 12.3 ──▶ 12.4
```

**可并行任务**：
- 1.1 与 1.2 可并行
- 3.1、3.2、3.3 可并行（测试编写）
- 5.1、5.2、5.3、5.4 可并行（测试编写）
- 7.1 与 7.2 可并行
- 12.3 与 12.4 可并行
