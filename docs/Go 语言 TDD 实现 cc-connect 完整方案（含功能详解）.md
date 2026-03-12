
# cc-connect Go 语言 TDD 实现完整方案

## 项目概述

基于原版 **cc-connect** 的核心功能，使用 Go 语言重新实现，提供 TUI（终端用户界面）操作界面，采用 TDD（测试驱动开发）流程，初期支持 Claude Code 和飞书平台。

### 原版 cc-connect 核心功能概览

**核心理念**：连接本地 AI 代理与聊天平台的桥梁工具，让你在任何聊天应用中控制本地 AI 代理。

**主要特性**：
1. **7 种 AI 代理支持**：Claude Code、Codex、Cursor Agent、Qoder CLI、Gemini CLI、OpenCode、iFlow CLI
2. **9 种聊天平台**：飞书、钉钉、Slack、Telegram、Discord、企业微信、LINE、QQ、QQ Bot
3. **多机器人中继**：在群聊中绑定多个机器人，让它们相互通信
4. **完整聊天控制**：通过斜杠命令切换模型、调整推理、更改权限模式、管理会话
5. **代理记忆**：无需接触终端即可读写代理指令文件
6. **定时任务**：用自然语言设置 cron 作业
7. **语音和图像**：发送语音消息或截图，cc-connect 处理 STT/TTS 和多模态转发
8. **多项目管理**：一个进程，多个项目，每个项目有自己的代理+平台组合

## 1. TDD 开发流程设计

### 1.1 基于功能优先级的 TDD 阶段划分

```
阶段 1: 核心消息系统 (第1-2周)
├── 统一消息结构 (支持文本、语音、图像、命令)
├── 消息路由引擎
├── 会话管理系统
└── 配置管理系统

阶段 2: 飞书平台适配器 (第3-4周)
├── WebSocket 连接 (无需公网IP)
├── 消息收发功能
├── 事件处理 (@提及、命令等)
└── 卡片消息支持

阶段 3: Claude Code 适配器 (第5-6周)
├── 子进程管理和通信
├── 流式输出解析
├── 工具调用处理 (Read, Grep, Glob等)
└── 权限模式切换 (default, yolo, plan, edit)

阶段 4: 斜杠命令系统 (第7周)
├── 基础命令 (/mode, /new, /list, /help)
├── 权限管理命令 (/allow, /stop)
├── 提供商管理命令 (/provider)
└── 定时任务命令 (/cron)

阶段 5: TUI 界面开发 (第8-10周)
├── 项目配置视图
├── 会话管理视图
├── 消息监控视图
└── 日志查看视图

阶段 6: 高级功能 (第11-12周)
├── 语音消息支持 (STT/TTS)
├── 多项目管理
├── 多机器人中继
└── 定时任务系统
```

### 1.2 TDD 循环（红-绿-重构）与功能对应

```
功能需求 → 编写测试 (红) → 最小实现 (绿) → 重构优化
    │          │            │            │
    ▼          ▼            ▼            ▼
飞书连接 → 连接测试 → WebSocket实现 → 错误处理优化
斜杠命令 → 命令解析测试 → 命令处理器 → 命令扩展
权限模式 → 模式切换测试 → 模式管理器 → 模式持久化
```

## 2. 核心功能模块的 TDD 实现

### 2.1 统一消息系统

**功能需求**：支持多种消息类型（文本、语音、图像、命令），跨平台统一格式。

**测试先行 (RED):**
```go
// internal/core/message_test.go
func TestMessageSystem(t *testing.T) {
    t.Run("支持多种消息类型", func(t *testing.T) {
        testCases := []struct {
            msgType    MessageType
            content    string
            shouldPass bool
        }{
            {MessageTypeText, "Hello World", true},
            {MessageTypeVoice, "audio.mp3", true},
            {MessageTypeImage, "screenshot.png", true},
            {MessageTypeCommand, "/mode yolo", true},
            {"invalid", "test", false},
        }
        
        for _, tc := range testCases {
            msg, err := NewMessage("test", "user", tc.content, tc.msgType)
            if tc.shouldPass {
                assert.NoError(t, err)
                assert.Equal(t, tc.msgType, msg.Type)
            } else {
                assert.Error(t, err)
            }
        }
    })
    
    t.Run("消息序列化兼容性", func(t *testing.T) {
        // 确保与Python版本的序列化兼容
        original := NewTextMessage("feishu", "user123", "测试消息")
        data, err := original.Serialize()
        assert.NoError(t, err)
        
        // 模拟Python版本的消息格式
        pythonFormat := map[string]interface{}{
            "id":        original.ID,
            "platform":  "feishu",
            "user_id":   "user123",
            "content":   "测试消息",
            "type":      "text",
            "timestamp": original.Timestamp.Format(time.RFC3339),
        }
        
        pythonData, _ := json.Marshal(pythonFormat)
        
        // 两种格式应该都能正确解析
        msg1, err1 := DeserializeMessage(data)
        msg2, err2 := DeserializeMessage(pythonData)
        
        assert.NoError(t, err1)
        assert.NoError(t, err2)
        assert.Equal(t, msg1.Content, msg2.Content)
    })
}
```

### 2.2 飞书 WebSocket 适配器

**功能需求**：无需公网IP的 WebSocket 连接，支持心跳、重连、事件处理。

**测试先行 (RED):**
```go
// internal/platform/feishu/client_test.go
func TestFeishuWebSocketFeatures(t *testing.T) {
    t.Run("无需公网IP的连接", func(t *testing.T) {
        // 模拟内网环境
        client := NewFeishuClient("app-id", "app-secret")
        client.SetTestMode(true) // 启用测试模式，不依赖外部网络
        
        ctx := context.Background()
        err := client.Connect(ctx)
        
        assert.NoError(t, err)
        assert.True(t, client.IsConnected())
        assert.False(t, client.RequiresPublicIP()) // 飞书不需要公网IP
    })
    
    t.Run("自动重连机制", func(t *testing.T) {
        server := NewMockWebSocketServer()
        server.SetDisconnectAfter(3) // 3次消息后断开
        
        client := NewFeishuClient("app-id", "app-secret")
        client.SetBaseURL(server.URL)
        client.SetReconnectInterval(100 * time.Millisecond)
        
        var disconnectCount int
        client.OnDisconnect(func() {
            disconnectCount++
        })
        
        var reconnectCount int
        client.OnReconnect(func() {
            reconnectCount++
        })
        
        ctx := context.Background()
        client.Connect(ctx)
        
        // 发送消息触发断开
        for i := 0; i < 5; i++ {
            client.SendMessage(NewTextMessage("feishu", "user", fmt.Sprintf("msg %d", i)))
            time.Sleep(50 * time.Millisecond)
        }
        
        assert.Eventually(t, func() bool {
            return reconnectCount > 0
        }, 2*time.Second, 100*time.Millisecond)
    })
    
    t.Run("@提及消息处理", func(t *testing.T) {
        handler := NewFeishuEventHandler()
        
        // 模拟飞书的@提及事件
        mentionEvent := `{
            "type": "im.message.receive_v1",
            "event": {
                "message": {
                    "content": "{\"text\":\"@_user_1 帮我修复bug\"}",
                    "mentions": [{"key": "@_user_1", "id": {"open_id": "user123"}}]
                }
            }
        }`
        
        msg, err := handler.HandleEvent([]byte(mentionEvent))
        assert.NoError(t, err)
        assert.Equal(t, MessageTypeCommand, msg.Type)
        assert.Contains(t, msg.Content, "帮我修复bug")
        assert.Equal(t, "user123", msg.UserID)
    })
}
```

### 2.3 Claude Code 权限模式管理

**功能需求**：支持多种权限模式，运行时切换。

**测试先行 (RED):**
```go
// internal/agent/claudecode/mode_test.go
func TestClaudeCodePermissionModes(t *testing.T) {
    t.Run("权限模式切换", func(t *testing.T) {
        agent := NewClaudeCodeAgent("/tmp/test")
        
        testModes := []struct {
            mode       string
            alias      []string
            expected   string
            shouldWork bool
        }{
            {"default", nil, "default", true},
            {"acceptEdits", []string{"edit"}, "acceptEdits", true},
            {"plan", nil, "plan", true},
            {"bypassPermissions", []string{"yolo"}, "bypassPermissions", true},
            {"invalid", nil, "", false},
        }
        
        for _, tm := range testModes {
            err := agent.SetPermissionMode(tm.mode)
            if tm.shouldWork {
                assert.NoError(t, err)
                assert.Equal(t, tm.expected, agent.CurrentMode())
            } else {
                assert.Error(t, err)
            }
        }
    })
    
    t.Run("工具调用权限控制", func(t *testing.T) {
        agent := NewClaudeCodeAgent("/tmp/test")
        
        testCases := []struct {
            mode     string
            tool     string
            autoAllow bool
        }{
            {"default", "Read", false},      // 默认模式所有工具都需要批准
            {"default", "WriteFile", false},
            {"edit", "Read", false},         // 编辑模式：编辑工具自动批准
            {"edit", "WriteFile", true},     // WriteFile 应该自动批准
            {"edit", "RunCommand", false},   // 运行命令仍需批准
            {"yolo", "Read", true},          // YOLO模式全部自动批准
            {"yolo", "WriteFile", true},
            {"yolo", "RunCommand", true},
            {"plan", "Read", true},          // 计划模式只读工具自动批准
            {"plan", "WriteFile", false},    // 写入工具需要批准
        }
        
        for _, tc := range testCases {
            agent.SetPermissionMode(tc.mode)
            requiresApproval := agent.RequiresToolApproval(tc.tool)
            assert.Equal(t, !tc.autoAllow, requiresApproval,
                "模式 %s, 工具 %s 的权限判断错误", tc.mode, tc.tool)
        }
    })
    
    t.Run("运行时模式切换", func(t *testing.T) {
        agent := NewClaudeCodeAgent("/tmp/test")
        agent.Start(context.Background(), nil)
        
        // 通过斜杠命令切换模式
        commandHandler := NewCommandHandler()
        
        // 模拟用户发送 /mode yolo
        msg := NewCommandMessage("feishu", "user123", "/mode yolo")
        response, err := commandHandler.Handle(msg, agent)
        
        assert.NoError(t, err)
        assert.Contains(t, response, "切换到 YOLO 模式")
        assert.Equal(t, "yolo", agent.CurrentMode())
        
        // 验证工具调用行为已改变
        assert.False(t, agent.RequiresToolApproval("WriteFile"))
    })
}
```

### 2.4 斜杠命令系统

**功能需求**：完整的斜杠命令支持，包括模式切换、会话管理、提供商管理等。

**测试先行 (RED):**
```go
// internal/core/command_test.go
func TestSlashCommandSystem(t *testing.T) {
    t.Run("基础命令解析", func(t *testing.T) {
        parser := NewCommandParser()
        
        testCases := []struct {
            input    string
            expected Command
        }{
            {"/mode", Command{Name: "mode", Args: []string{}}},
            {"/mode yolo", Command{Name: "mode", Args: []string{"yolo"}}},
            {"/new session1", Command{Name: "new", Args: []string{"session1"}}},
            {"/list", Command{Name: "list", Args: []string{}}},
            {"/help", Command{Name: "help", Args: []string{}}},
            {"/provider list", Command{Name: "provider", Args: []string{"list"}}},
            {"/provider add relay sk-xxx", Command{Name: "provider", Args: []string{"add", "relay", "sk-xxx"}}},
            {"/cron add 0 6 * * * 总结GitHub趋势", Command{
                Name: "cron",
                Args: []string{"add", "0", "6", "*", "*", "*", "总结GitHub趋势"},
            }},
        }
        
        for _, tc := range testCases {
            cmd, err := parser.Parse(tc.input)
            assert.NoError(t, err)
            assert.Equal(t, tc.expected.Name, cmd.Name)
            assert.Equal(t, tc.expected.Args, cmd.Args)
        }
    })
    
    t.Run("提供商管理命令", func(t *testing.T) {
        manager := NewProviderManager()
        commandHandler := NewCommandHandler()
        
        // 添加提供商
        msg := NewCommandMessage("feishu", "user123", 
            "/provider add relay sk-xxx https://api.relay.com claude-sonnet-4")
        
        response, err := commandHandler.Handle(msg, manager)
        assert.NoError(t, err)
        assert.Contains(t, response, "提供商 relay 添加成功")
        
        // 列出提供商
        msg = NewCommandMessage("feishu", "user123", "/provider list")
        response, err = commandHandler.Handle(msg, manager)
        assert.NoError(t, err)
        assert.Contains(t, response, "relay")
        assert.Contains(t, response, "claude-sonnet-4")
        
        // 切换提供商
        msg = NewCommandMessage("feishu", "user123", "/provider switch relay")
        response, err = commandHandler.Handle(msg, manager)
        assert.NoError(t, err)
        assert.Contains(t, response, "已切换到 relay")
        
        // 验证环境变量已设置
        assert.Equal(t, "sk-xxx", os.Getenv("ANTHROPIC_API_KEY"))
        assert.Equal(t, "https://api.relay.com", os.Getenv("ANTHROPIC_BASE_URL"))
    })
    
    t.Run("定时任务命令", func(t *testing.T) {
        scheduler := NewCronScheduler()
        commandHandler := NewCommandHandler()
        
        // 添加定时任务
        msg := NewCommandMessage("feishu", "user123",
            "/cron add 0 6 * * * 每天早上6点总结GitHub趋势")
        
        response, err := commandHandler.Handle(msg, scheduler)
        assert.NoError(t, err)
        assert.Contains(t, response, "定时任务添加成功")
        
        // 列出定时任务
        msg = NewCommandMessage("feishu", "user123", "/cron")
        response, err = commandHandler.Handle(msg, scheduler)
        assert.NoError(t, err)
        assert.Contains(t, response, "0 6 * * *")
        assert.Contains(t, response, "GitHub趋势")
        
        // 模拟定时任务触发
        go scheduler.Start()
        time.Sleep(100 * time.Millisecond)
        
        // 验证任务已添加到cron
        assert.Equal(t, 1, scheduler.JobCount())
    })
}
```

### 2.5 多机器人中继功能

**功能需求**：在群聊中绑定多个机器人，实现跨AI代理的协作。

**测试先行 (RED):**
```go
// internal/core/relay_test.go
func TestMultiBotRelay(t *testing.T) {
    t.Run("机器人绑定功能", func(t *testing.T) {
        relay := NewBotRelay()
        
        // 创建两个代理
        claude := NewMockAgent("claudecode")
        gemini := NewMockAgent("gemini")
        
        // 绑定到同一个聊天会话
        sessionID := "group-chat-123"
        relay.BindAgent(sessionID, "claudecode", claude)
        relay.BindAgent(sessionID, "gemini", gemini)
        
        bindings := relay.GetBindings(sessionID)
        assert.Equal(t, 2, len(bindings))
        assert.Contains(t, bindings, "claudecode")
        assert.Contains(t, bindings, "gemini")
    })
    
    t.Run("跨代理消息转发", func(t *testing.T) {
        relay := NewBotRelay()
        
        claude := NewMockAgent("claudecode")
        gemini := NewMockAgent("gemini")
        
        sessionID := "group-chat-123"
        relay.BindAgent(sessionID, "claudecode", claude)
        relay.BindAgent(sessionID, "gemini", gemini)
        
        // 用户向Claude提问
        userMsg := NewTextMessage("feishu", "user123", "@Claude 帮我review这段代码")
        relay.RouteMessage(sessionID, userMsg)
        
        // Claude应该收到消息
        assert.Equal(t, 1, len(claude.ReceivedMessages))
        assert.Contains(t, claude.ReceivedMessages[0].Content, "review这段代码")
        
        // Claude咨询Gemini
        claudeToGemini := NewTextMessage("internal", "claudecode", 
            "请问Gemini对这个架构有什么看法？")
        relay.SendToAgent(sessionID, "gemini", claudeToGemini)
        
        // Gemini应该收到消息
        assert.Equal(t, 1, len(gemini.ReceivedMessages))
        assert.Contains(t, gemini.ReceivedMessages[0].Content, "对这个架构有什么看法")
        
        // Gemini回复
        geminiResponse := NewTextMessage("internal", "gemini",
            "我认为这个架构在可扩展性方面可以改进...")
        relay.RouteMessage(sessionID, geminiResponse)
        
        // 回复应该转发回群聊
        // 这里可以验证消息被正确转发
    })
    
    t.Run("@提及特定机器人", func(t *testing.T) {
        relay := NewBotRelay()
        
        claude := NewMockAgent("claudecode")
        gemini := NewMockAgent("gemini")
        
        sessionID := "group-chat-123"
        relay.BindAgent(sessionID, "claudecode", claude)
        relay.BindAgent(sessionID, "gemini", gemini)
        
        // 用户@提及特定机器人
        testCases := []struct {
            message    string
            targetBot  string
            shouldReceive bool
        }{
            {"@Claude 前端代码怎么优化？", "claudecode", true},
            {"@Gemini 这个算法复杂度如何？", "gemini", true},
            {"大家怎么看这个问题？", "", false}, // 没有@提及，所有机器人都可能回复
            {"@Claude @Gemini 你们合作分析一下", "", true}, // 多机器人提及
        }
        
        for _, tc := range testCases {
            claude.Reset()
            gemini.Reset()
            
            msg := NewTextMessage("feishu", "user123", tc.message)
            relay.RouteMessage(sessionID, msg)
            
            if tc.targetBot == "claudecode" {
                assert.Equal(t, tc.shouldReceive, len(claude.ReceivedMessages) > 0)
            } else if tc.targetBot == "gemini" {
                assert.Equal(t, tc.shouldReceive, len(gemini.ReceivedMessages) > 0)
            } else {
                // 没有指定目标，可能所有机器人都收到
                // 这里测试路由逻辑
            }
        }
    })
}
```

## 3. TUI 界面开发策略

### 3.1 基于功能优先级的视图开发

```go
// 第一阶段：基础监控视图（第8周）
type MonitorView struct {
    messages []*Message      // 消息列表
    sessions map[string]*Session // 会话状态
    agents   map[string]*Agent   // 代理状态
    platforms map[string]*Platform // 平台状态
}

// 第二阶段：交互式配置视图（第9周）
type ConfigView struct {
    projects []ProjectConfig    // 项目配置
    currentProject *ProjectConfig // 当前项目
    editingField string         // 正在编辑的字段
}

// 第三阶段：高级管理视图（第10周）
type ManagementView struct {
    cronJobs []CronJob         // 定时任务
    providers []Provider       // API提供商
    relayBindings []RelayBinding // 中继绑定
}
```

### 3.2 TUI 测试策略

```go
// internal/tui/views/monitor_test.go
func TestTUIViews(t *testing.T) {
    t.Run("消息实时更新", func(t *testing.T) {
        view := NewMonitorView(80, 24)
        
        // 模拟消息流
        go func() {
            for i := 0; i < 10; i++ {
                msg := NewTextMessage("feishu", fmt.Sprintf("user%d", i),
                    fmt.Sprintf("消息 %d", i))
                view.AddMessage(msg)
                time.Sleep(50 * time.Millisecond)
            }
        }()
        
        // 验证视图能正确处理实时更新
        time.Sleep(300 * time.Millisecond)
        rendered := view.Render()
        
        assert.Contains(t, rendered, "消息 5")
        assert.Contains(t, rendered, "消息 9")
    })
    
    t.Run("键盘快捷键", func(t *testing.T) {
        app := NewTUIApp()
        
        // 测试快捷键
        testCases := []struct {
            key      tea.KeyMsg
            expectedView string
        }{
            {tea.KeyMsg{Type: tea.KeyF1}, "help"},
            {tea.KeyMsg{Type: tea.KeyF2}, "config"},
            {tea.KeyMsg{Type: tea.KeyTab}, "sessions"},
            {tea.KeyMsg{Type: tea.KeyCtrlC}, "exit"},
        }
        
        for _, tc := range testCases {
            app.HandleKey(tc.key)
            assert.Equal(t, tc.expectedView, app.CurrentView())
        }
    })
}
```

## 4. 持续集成与质量保证

### 4.1 基于功能的测试覆盖率要求

```yaml
# .github/workflows/ci.yml
jobs:
  test:
    strategy:
      matrix:
        test-type: [unit, integration, e2e]
    
    steps:
    - name: 运行单元测试 (核心功能)
      run: |
        go test ./internal/core/... -v -coverprofile=core.coverage
        # 要求核心功能覆盖率 > 85%
    
    - name: 运行集成测试 (平台适配器)
      run: |
        go test ./internal/platform/... -v -tags=integration
        # 飞书适配器必须通过所有集成测试
    
    - name: 运行端到端测试 (完整流程)
      run: |
        go test ./test/e2e/... -v -tags=e2e
        # 验证从消息接收到AI响应的完整流程
    
    - name: 性能测试
      run: |
        go test ./internal/... -bench=. -benchtime=5s
        # 消息处理延迟 < 100ms
        # 内存使用 < 100MB
```

### 4.2 功能验收标准

```go
// test/e2e/complete_workflow_test.go
func TestCompleteWorkflow(t *testing.T) {
    t.Run("完整消息流程: 用户 → 飞书 → Claude Code → 响应", func(t *testing.T) {
        // 1. 启动cc-connect
        app := StartCCConnect("test/config.toml")
        defer app.Stop()
        
        // 2. 模拟用户发送消息到飞书
        feishuMock := NewMockFeishuServer()
        userMsg := "帮我修复main.go中的bug"
        feishuMock.SendMessage(userMsg)
        
        // 3. 验证消息被正确接收
        assert.Eventually(t, func() bool {
            return app.HasReceivedMessage(userMsg)
        }, 5*time.Second, 100*time.Millisecond)
        
        // 4. 验证Claude Code被调用
        assert.Eventually(t, func() bool {
            return app.AgentWasCalled("claudecode")
        }, 10*time.Second, 500*time.Millisecond)
        
        // 5. 验证响应发送回飞书
        assert.Eventually(t, func() bool {
            return feishuMock.HasReceivedResponse()
        }, 15*time.Second, 1*time.Second)
        
        // 6. 验证响应内容合理
        response := feishuMock.LastResponse()
        assert.Contains(t, response, "bug")
        assert.Contains(t, response, "main.go")
    })
    
    t.Run("斜杠命令工作流", func(t *testing.T) {
        app := StartCCConnect("test/config.toml")
        defer app.Stop()
        
        // 发送 /mode yolo 命令
        feishuMock := NewMockFeishuServer()
        feishuMock.SendMessage("/mode yolo")
        
        // 验证模式已切换
        assert.Eventually(t, func() bool {
            return app.CurrentMode() == "yolo"
        }, 3*time.Second, 100*time.Millisecond)
        
        // 验证响应消息
        response := feishuMock.LastResponse()
        assert.Contains(t, response, "YOLO")
        assert.Contains(t, response, "切换成功")
    })
    
    t.Run("多机器人协作", func(t *testing.T) {
        app := StartCCConnect("test/multi-bot-config.toml")
        defer app.Stop()
        
        // 绑定两个机器人
        feishuMock := NewMockFeishuServer()
        feishuMock.SendMessage("/bind claudecode")
        feishuMock.SendMessage("/bind gemini")
        
        // 发送需要协作的问题
        feishuMock.SendMessage("@Claude @Gemini 你们合作分析这个架构设计")
        
        // 验证两个代理都被调用
        assert.Eventually(t, func() bool {
            return app.AgentWasCalled("claudecode") && 
                   app.AgentWasCalled("gemini")
        }, 10*time.Second, 500*time.Millisecond)
        
        // 验证有协作响应
        responses := feishuMock.AllResponses()
        assert.Greater(t, len(responses), 1)
        
        // 验证响应显示协作结果
        collaborationFound := false
        for _, resp := range responses {
            if strings.Contains(resp, "Claude") && strings.Contains(resp, "Gemini") {
                collaborationFound = true
                break
            }
        }
        assert.True(t, collaborationFound, "应该显示两个机器人的协作结果")
    })
}
```

## 5. 开发路线图与里程碑

### 里程碑 1: 核心消息系统 (第2周末)
- [x] 统一消息结构 (文本、语音、图像、命令)
- [x] 消息序列化/反序列化
- [x] 会话管理基础
- [x] 配置管理系统
- [ ] **测试覆盖率**: > 80%

### 里程碑 2: 飞书平台完整支持 (第4周末)
- [x] WebSocket 连接 (无需公网IP)
- [x] 消息收发功能
- [x] @提及和命令识别
- [x] 自动重连和心跳
- [ ] **集成测试**: 所有飞书事件类型

### 里程碑 3: Claude Code 完整集成 (第6周末)
- [x] 子进程管理
- [x] 流式输出解析
- [x] 工具调用处理
- [x] 权限模式切换 (default, edit, plan, yolo)
- [ ] **端到端测试**: 完整AI交互流程

### 里程碑 4: 斜杠命令系统 (第7周末)
- [x] 基础命令 (/mode, /new, /list, /help)
- [x] 权限管理命令 (/allow, /stop)
- [x] 提供商管理命令 (/provider)
- [x] 定时任务命令 (/cron)
- [ ] **用户体验测试**: 命令响应时间 < 500ms

### 里程碑 5: TUI 基础界面 (第10周末)
- [x] 消息监控视图
- [x] 会话管理视图
- [x] 项目配置视图
- [x] 实时状态显示
- [ ] **性能指标**: 界面响应 < 100ms

### 里程碑 6: 高级功能 (第12周末)
- [ ] 多项目管理
- [ ] 多机器人中继
- [ ] 语音消息支持 (STT/TTS)
- [ ] 定时任务系统
- [ ] **生产就绪**: 所有功能稳定可用

## 6. 风险管理与缓解策略

### 技术风险
1. **WebSocket 连接稳定性**
   - 风险：飞书WebSocket连接可能不稳定
   - 缓解：实现自动重连、指数退避、心跳检测
   - 测试：模拟网络中断、服务器重启场景

2. **Claude Code 进程管理**
   - 风险：子进程可能挂起或资源泄漏
   - 缓解：使用context控制超时，监控资源使用
   - 测试：长时间运行测试，内存泄漏检测

3. **TUI 性能问题**
   - 风险：消息量大时界面卡顿
   - 缓解：异步渲染、消息分页、虚拟滚动
   - 测试：性能基准测试，压力测试

### 功能风险
1. **权限模式切换的副作用**
   - 风险：模式切换可能导致未预期的工具调用
   - 缓解：添加确认提示，记录模式切换日志
   - 测试：所有模式组合的边界测试

2. **多机器人协作的复杂性**
   - 风险：机器人间消息循环或竞争条件
   - 缓解：添加消息去重，设置超时机制
   - 测试：并发消息处理测试

## 7. 部署与维护策略

### 7.1 渐进式部署
```yaml
# 第一阶段：内部测试
audience: 开发团队
features: 核心消息 + 飞书 + Claude Code
duration: 2周

# 第二阶段：有限用户测试
audience: 技术爱好者
features: 添加斜杠命令 + TUI基础
duration: 3周

# 第三阶段：公开测试
audience: 所有用户
features: 完整功能集
duration: 4周

# 第四阶段：正式发布
audience: 生产用户
features: 稳定版本 + 文档完善
```

### 7.2 监控与告警
```go
// 关键指标监控
type Metrics struct {
    MessageLatency   prometheus.Histogram // 消息处理延迟
    ConnectionStatus prometheus.Gauge     // 连接状态
    AgentHealth      prometheus.Gauge     // 代理健康状态
    ErrorRate        prometheus.Counter   // 错误率
    MemoryUsage      prometheus.Gauge     // 内存使用
}

// 告警规则
// - 消息延迟 > 500ms 持续5分钟
// - 连接断开 > 3次/小时  
// - 代理无响应 > 2分钟
// - 内存使用 > 80% 持续10分钟
```

这个完整的 TDD 方案将 cc-connect 的所有核心功能都融入了开发流程，确保每个功能都有对应的测试覆盖，同时保持了 Go 语言的最佳实践和代码质量。
