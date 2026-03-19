## ADDED Requirements

### Requirement: CardSender 接口

系统必须定义平台无关的 `CardSender` 接口，用于发送交互式卡片。

- 必须定义 `SendCard(ctx, replyCtx, card)` 方法
- 必须定义 `ReplyCard(ctx, replyCtx, card)` 方法
- 平台 adapter 可选择是否实现此接口

#### Scenario: 检查平台是否支持卡片

- **WHEN** Router 需要发送权限请求卡片
- **THEN** 先检查 `CardSender` 接口是否实现
- **THEN** 未实现则降级为文本消息

### Requirement: Card 结构定义

系统必须定义平台无关的 `Card` 结构。

- 必须支持 `Header`（标题和颜色）
- 必须支持 `Markdown` 元素
- 必须支持 `Actions` 元素（按钮行）
- 必须支持 `Divider` 元素
- 必须支持 `Note` 元素（脚注）
- 按钮必须支持 text、type、value 属性

#### Scenario: 构建权限请求卡片

- **WHEN** 构建权限请求卡片
- **THEN** 使用 CardBuilder 创建：
  ```go
  NewCard().
    Title("🤖 Claude 需要您的确认", "blue").
    Markdown("**工具**: Bash\n**命令**: `npm install`").
    ButtonsEqual(
      PrimaryBtn("✅ 允许", "perm:allow:req123"),
      DangerBtn("❌ 拒绝", "perm:deny:req123"),
    ).
    Note("回复 A 允许，D 拒绝")
  ```

#### Scenario: 构建 AskUserQuestion 卡片

- **WHEN** 构建问答卡片
- **THEN** 使用 CardBuilder 创建：
  ```go
  NewCard().
    Title("🤖 Claude 问您", "blue").
    Markdown("您希望使用哪种数据库？").
    ButtonsEqual(
      DefaultBtn("PostgreSQL", "ans:req123:PostgreSQL"),
      DefaultBtn("MySQL", "ans:req123:MySQL"),
      DefaultBtn("SQLite", "ans:req123:SQLite"),
    )
  ```

### Requirement: 飞书卡片渲染

飞书 adapter 必须将 `Card` 渲染为飞书交互式卡片 JSON。

- 必须使用飞书卡片 v1 格式
- 按钮必须设置正确的 `value` 字段
- `value` 必须包含 `session_key` 用于回调路由
- 按钮点击必须触发飞书回调事件

#### Scenario: 渲染卡片为飞书 JSON

- **WHEN** 调用 `feishuAdapter.ReplyCard(ctx, replyCtx, card)`
- **THEN** 生成飞书卡片 JSON，包含：
  - `config.wide_screen_mode: true`
  - `header.title` 为卡片标题
  - `elements` 包含 markdown 和 button
  - 按钮 `value` 包含 action 和 session_key

#### Scenario: 发送卡片成功

- **WHEN** 调用 `feishuAdapter.ReplyCard(ctx, replyCtx, card)`
- **THEN** 调用飞书 API 发送消息
- **THEN** 返回 nil 表示成功

#### Scenario: 发送卡片失败

- **WHEN** 飞书 API 返回错误
- **THEN** 返回包装后的错误信息

### Requirement: 飞书卡片回调处理

飞书 adapter 必须处理卡片按钮点击回调。

- 必须验证回调签名（使用 encryptKey）
- 必须解析按钮 `value` 中的 action 和 session_key
- 必须将回调转换为统一消息格式
- action 格式必须为 `perm:allow:<requestID>` 或 `perm:deny:<requestID>`

#### Scenario: 解析允许按钮回调

- **WHEN** 用户点击"允许"按钮
- **THEN** 飞书发送卡片回调到 webhook
- **THEN** adapter 解析 action = "perm:allow:req123"
- **THEN** adapter 生成 `Message{Type: "command", Content: "/allow req123"}`

#### Scenario: 解析拒绝按钮回调

- **WHEN** 用户点击"拒绝"按钮
- **THEN** adapter 解析 action = "perm:deny:req123"
- **THEN** adapter 生成 `Message{Type: "command", Content: "/deny req123"}`

#### Scenario: 解析答案按钮回调

- **WHEN** 用户点击答案选项按钮
- **THEN** adapter 解析 action = "ans:req123:PostgreSQL"
- **THEN** adapter 生成 `Message{Type: "command", Content: "/answer req123 PostgreSQL"}`

#### Scenario: 验证回调签名

- **WHEN** 收到飞书卡片回调
- **THEN** 使用 encryptKey 验证签名
- **THEN** 签名无效则返回 403 错误

### Requirement: 文本降级

不支持卡片的平台必须降级为文本消息。

- 文本消息必须包含关键信息
- 必须提示用户使用命令响应

#### Scenario: 降级为文本消息

- **GIVEN** 平台未实现 `CardSender` 接口
- **WHEN** 需要发送权限请求
- **THEN** 发送文本消息：
  ```
  🤖 Claude 需要您的确认

  工具: Bash
  命令: npm install

  回复 /allow 允许, /deny 拒绝
  请求ID: req123
  ```
