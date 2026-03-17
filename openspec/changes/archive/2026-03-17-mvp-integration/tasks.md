## 1. 基础结构和接口定义

- [x] 1.1 创建 `internal/app/doc.go` 包文档，说明整合层职责
- [x] 1.2 编写 `ReplySender` 接口定义的测试（`internal/app/app_test.go`）
- [x] 1.3 实现 `ReplySender` 接口和 `replySender` 适配器
- [x] 1.4 编写 `HandlerContext` 结构体定义的测试
- [x] 1.5 实现 `HandlerContext` 结构体

**验证**: `go test ./internal/app/... -v`

## 2. App 核心结构

- [x] 2.1 编写 `App` 结构体和 `New` 函数的失败测试（无效配置、缺失字段）
- [x] 2.2 实现 `App` 结构体定义和 `AppStatus` 常量
- [x] 2.3 实现 `New` 函数（配置验证、组件初始化）
- [x] 2.4 编写 `App.Start` 的失败测试（Feishu 连接失败、Agent 启动失败）
- [x] 2.5 实现 `App.Start` 方法（启动 Agent、连接 Feishu、注册处理器）
- [x] 2.6 编写 `App.Stop` 的测试（优雅关闭、资源清理）
- [x] 2.7 实现 `App.Stop` 方法
- [x] 2.8 编写 `App.WaitForShutdown` 的测试
- [x] 2.9 实现 `App.WaitForShutdown` 方法

**验证**: `go test ./internal/app/... -v -cover`

## 3. 消息处理器

- [x] 3.1 编写 `wrapHandler` 适配器的测试（HandlerContext 注入、panic 恢复）
- [x] 3.2 实现 `wrapHandler` 适配器函数
- [x] 3.3 编写文本消息处理器的测试（成功处理、超时、错误）
- [x] 3.4 实现文本消息处理器（状态提示、Agent 调用、响应发送）
- [x] 3.5 编写命令消息处理器的测试（成功执行、未知命令）
- [x] 3.6 实现命令消息处理器
- [x] 3.7 编写 `registerHandlers` 方法的测试
- [x] 3.8 实现 `registerHandlers` 方法

**依赖**: 任务 2 完成后开始
**验证**: `go test ./internal/app/... -v -cover`

## 4. CLI 入口点

- [x] 4.1 创建 `cmd/cc-connect/main.go` 骨架
- [x] 4.2 实现配置文件加载逻辑（支持 `-config` 参数）
- [x] 4.3 实现应用初始化和启动
- [x] 4.4 实现信号处理（SIGINT、SIGTERM 触发优雅关闭）
- [x] 4.5 实现版本信息显示（`--version` 参数）
- [x] 4.6 添加启动失败的错误处理和友好提示

**依赖**: 任务 2、3 完成后开始
**验证**: `go build ./cmd/cc-connect && ./cc-connect --help`

## 5. E2E 测试基础设施

- [x] 5.1 创建 `test/e2e/` 目录结构
- [x] 5.2 编写 `MockAgent` 实现（支持预设响应、延迟、错误）
- [x] 5.3 编写 `MockFeishuServer` 实现（事件注入、响应捕获）
- [x] 5.4 编写 `MockReplySender` 实现（用于单元测试）

**依赖**: 无依赖，可与任务 2、3 并行
**验证**: `go test ./test/e2e/... -v`

## 6. E2E 测试用例

- [x] 6.1 编写 E2E 测试：完整文本消息流程（成功路径）
- [x] 6.2 编写 E2E 测试：命令消息流程（/mode、/help、/new）
- [x] 6.3 编写 E2E 测试：Agent 超时场景
- [x] 6.4 编写 E2E 测试：Agent 错误场景
- [x] 6.5 编写 E2E 测试：并发消息处理
- [x] 6.6 编写 E2E 测试：优雅关闭流程

**依赖**: 任务 5 完成后开始
**验证**: `go test ./test/e2e/... -v -tags=e2e`

## 7. 集成和验收

- [x] 7.1 运行所有单元测试确保通过
- [x] 7.2 运行 E2E 测试确保通过
- [x] 7.3 检查 `internal/app` 测试覆盖率 ≥ 80%
- [ ] 7.4 手动测试：使用真实配置启动应用
- [ ] 7.5 手动测试：发送文本消息并获得响应
- [ ] 7.6 手动测试：发送斜杠命令并获得响应
- [ ] 7.7 手动测试：优雅关闭（Ctrl+C）

**依赖**: 任务 1-6 全部完成
**验证**: `go test ./... -cover`
- `go build ./cmd/cc-connect`
- README.md 已创建

## 任务依赖关系

```
任务 1 (基础结构) ──┬──▶ 任务 2 (App 核心) ──┬──▶ 任务 3 (处理器) ──┐
                   │                        │                      │
                   │                        ▼                      ▼
                   │                   任务 4 (CLI) ◀──────── 任务 7 (集成)
                   │
                   └──▶ 任务 5 (E2E 基础) ──▶ 任务 6 (E2E 用例) ──▶ 任务 7 (集成)

可并行: 任务 2 与任务 5
```

## 预计工时

| 任务组 | 预计时间 |
|--------|----------|
| 1. 基础结构 | 0.5 天 |
| 2. App 核心 | 1 天 |
| 3. 消息处理器 | 1 天 |
| 4. CLI 入口点 | 0.5 天 |
| 5. E2E 基础设施 | 0.5 天 |
| 6. E2E 测试用例 | 1 天 |
| 7. 集成和验收 | 0.5 天 |
| **总计** | **5 天** |
