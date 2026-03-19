## 1. 核心状态机实现

> 阶段目标：实现 pendingPermission 状态机，支持阻塞/唤醒机制
> 影响模块：`internal/agent/claudecode`

- [x] 1.1 定义 pendingPermission 结构和 PermissionResult 结构
  - 新增 `pendingPermission` 结构体（requestID, toolName, toolInput, resolved channel, result）
  - 新增 `PermissionResult` 结构体（Behavior, UpdatedInput, Message）
  - 新增 `UserQuestion` 和 `UserQuestionOption` 结构体
  - 验证：`go build ./internal/agent/claudecode/...`

- [x] 1.2 实现 RespondPermission 方法（TDD）
  - 先写失败测试：调用 RespondPermission 时无 pending 返回错误
  - 先写失败测试：重复响应同一 requestID 返回错误
  - 实现最小逻辑：存储 result 并关闭 resolved channel
  - 验证：`go test ./internal/agent/claudecode/... -run TestRespondPermission`

- [x] 1.3 修改 handleControlRequest 创建 pending 状态（TDD）
  - 先写测试：autoApprove=true 时不创建 pending
  - 先写测试：autoApprove=false 时创建 pending 并发送事件
  - 实现：创建 pendingPermission，发送 EventPermissionRequest
  - 验证：`go test ./internal/agent/claudecode/... -run TestControlRequest`

- [x] 1.4 实现阻塞等待机制（TDD）
  - 先写测试：模拟用户响应后唤醒
  - 先写测试：超时后自动拒绝
  - 实现：使用 `<-pending.resolved` 阻塞，带 context 和 timeout
  - 验证：`go test ./internal/agent/claudecode/... -run TestPermissionBlocking`

- [x] 1.5 扩展 Event 结构支持权限请求字段
  - 新增 Event.RequestID, Event.ToolInputRaw, Event.Questions 字段
  - 修改 handleControlRequest 填充这些字段
  - 验证：`go test ./internal/agent/claudecode/... -run TestEvent`

## 2. Card 结构与 Builder

> 阶段目标：定义平台无关的 Card 结构，便于构建交互式消息
> 影响模块：`internal/core`

- [x] 2.1 定义 Card 核心结构
  - 新增 `Card`, `CardHeader`, `CardElement` 接口
  - 实现 `CardMarkdown`, `CardActions`, `CardButton`, `CardDivider`, `CardNote`
  - 验证：`go build ./internal/core/...`

- [x] 2.2 实现 CardBuilder（TDD）
  - 先写测试：链式构建卡片
  - 实现 `NewCard().Title().Markdown().Buttons().Build()`
  - 验证：`go test ./internal/core/... -run TestCardBuilder`

- [x] 2.3 实现 Card.RenderText 降级方法（TDD）
  - 先写测试：卡片渲染为文本消息
  - 实现：将按钮渲染为 `[按钮名]` 格式
  - 验证：`go test ./internal/core/... -run TestCardRender`

- [x] 2.4 定义 CardSender 接口
  - 新增 `CardSender` 接口（SendCard, ReplyCard）
  - 验证：`go build ./internal/core/...`

## 3. 飞书卡片适配器

> 阶段目标：实现飞书交互式卡片的发送和回调处理
> 影响模块：`internal/platform/feishu`
> 依赖：阶段 2 完成

- [x] 3.1 实现 renderCardMap 函数（TDD）
  - 先写测试：Card 转换为飞书卡片 JSON
  - 实现：渲染 header, markdown, buttons, divider, note
  - 验证：`go test ./internal/platform/feishu/... -run TestRenderCard`

- [x] 3.2 实现 ReplyCard 和 SendCard 方法（TDD）
  - 先写测试：mock client 验证 API 调用
  - 实现：调用飞书 Message.Reply API
  - 验证：`go test ./internal/platform/feishu/... -run TestCardSender`

- [x] 3.3 实现卡片回调解析（TDD）
  - 先写测试：解析按钮点击回调
  - 实现：提取 action 和 session_key
  - 转换为 `Message{Type: "command", Content: "/allow req123"}` 格式
  - 验证：`go test ./internal/platform/feishu/... -run TestCardCallback`

- [x] 3.4 注册卡片回调路由
  - 在 feishu adapter 的 webhook handler 中添加卡片回调处理
  - 验证：集成测试覆盖完整回调流程

## 4. Router 权限处理集成

> 阶段目标：将权限状态机与消息路由集成
> 影响模块：`internal/core`
> 依赖：阶段 1、阶段 3 完成

- [x] 4.1 处理 EventPermissionRequest 事件
  - 在 Router 中监听 EventPermissionRequest
  - 检查平台是否实现 CardSender
  - 发送权限请求卡片或降级文本
  - 验证：`go test ./internal/core/... -run TestPermissionRouting`

- [x] 4.2 处理 /allow 和 /deny 命令（TDD）
  - 先写测试：/allow 命令批准权限
  - 先写测试：/deny 命令拒绝权限
  - 实现：解析命令，调用 session.RespondPermission
  - 验证：`go test ./internal/core/... -run TestAllowDeny`

- [x] 4.3 处理 /answer 命令（TDD）
  - 先写测试：/answer 命令回答 AskUserQuestion
  - 实现：解析答案，构造 PermissionResult
  - 验证：`go test ./internal/core/... -run TestAnswer`

- [x] 4.4 实现 Session busy 状态
  - 在等待权限时标记 session 为 busy
  - busy 时拒绝新消息或排队
  - 验证：`go test ./internal/core/... -run TestSessionBusy`

## 5. AskUserQuestion 支持

> 阶段目标：完整支持 Claude 的 AskUserQuestion 工具
> 影响模块：`internal/agent/claudecode`, `internal/core`
> 依赖：阶段 4 完成

- [x] 5.1 解析 AskUserQuestion 输入（TDD）
  - 先写测试：解析 questions 数组
  - 实现：parseUserQuestions 函数
  - 验证：`go test ./internal/agent/claudecode/... -run TestParseQuestions`

- [x] 5.2 生成问答卡片
  - 单选：按钮组
  - 多选：提示 + 多选按钮
  - 开放：提示文本回复
  - 验证：`go test ./internal/core/... -run TestQuestionCard`

- [x] 5.3 处理答案按钮回调
  - 解析 `ans:req123:PostgreSQL` 格式
  - 构造 UpdatedInput: {"answers": ["PostgreSQL"]}
  - 验证：`go test ./internal/core/... -run TestAnswerCallback`

## 6. 集成测试与验收

> 阶段目标：端到端测试确保完整流程可用
> 影响模块：`test/e2e`

- [x] 6.1 编写权限请求端到端测试
  - 场景：Claude 请求 Bash 权限 → 飞书卡片 → 用户批准 → Claude 继续
  - 场景：用户拒绝 → Claude 收到拒绝消息
  - 验证：`go test ./test/e2e/... -run TestPermissionE2E -tags=integration`

- [x] 6.2 编写 AskUserQuestion 端到端测试
  - 场景：Claude 问用户 → 飞书选项卡片 → 用户选择 → Claude 收到答案
  - 验证：`go test ./test/e2e/... -run TestAskUserQuestionE2E -tags=integration`

- [x] 6.3 更新 CLAUDE.md 文档
  - 添加权限处理机制说明
  - 添加 /allow、/deny、/answer 命令文档

- [x] 6.4 确保测试覆盖率 > 80%
  - 运行 `go test ./... -coverprofile=coverage.out`
  - 补充缺失的测试用例
  - 结果：83.8% 覆盖率 ✓
