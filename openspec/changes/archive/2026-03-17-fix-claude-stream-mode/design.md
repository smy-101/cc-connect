## Context

### 当前状态

现有 `internal/agent/claudecode/` 实现使用 `-p` 单次执行模式：

```
claude -p --output-format stream-json --permission-mode <mode> <message>
```

问题：
1. 每条消息启动新进程，进程立即退出
2. 无法接收和处理 `control_request` 权限请求
3. stdin 在启动后立即关闭，无法进行双向通信

### 参考实现

chenhg5/cc-connect 使用交互式流模式：

```
claude --output-format stream-json --input-format stream-json --permission-prompt-tool stdio
```

进程持久运行，通过 stdin/stdout 进行双向 JSON 消息通信。

### 约束

- 保持现有 `agent.Agent` 接口不变
- 支持现有权限模式：default, acceptEdits, plan, bypassPermissions
- 必须可测试（不依赖真实 Claude CLI）

## Goals / Non-Goals

**Goals:**
- 实现持久会话进程，支持多条消息
- 实现双向通信：通过 stdin 发送消息，从 stdout 读取事件
- 实现 `control_request` / `control_response` 权限处理
- 支持 YOLO 模式自动批准权限请求
- 支持图片和文件附件

**Non-Goals:**
- 不实现用户交互式权限确认（后续阶段）
- 不修改 `agent.Agent` 接口
- 不支持多模型切换（后续阶段）
- 不实现 provider 代理功能

## Decisions

### 1. 会话进程管理架构

**决策**: 引入 `Session` 结构体管理持久进程

**理由**:
- 参考项目使用 `claudeSession` 结构体封装进程生命周期
- 将进程管理与 Agent 接口解耦
- 便于测试时 mock

```
┌─────────────────────────────────────────────────────────────┐
│                      ClaudeCodeAgent                        │
│  (implements agent.Agent)                                   │
├─────────────────────────────────────────────────────────────┤
│  - config *Config                                           │
│  - session *Session  ◄─── 持久会话                          │
│  - mu sync.RWMutex                                          │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                        Session                              │
├─────────────────────────────────────────────────────────────┤
│  - cmd *exec.Cmd           // Claude 进程                   │
│  - stdin io.WriteCloser    // 发送消息                      │
│  - stdout io.Reader        // 接收事件                      │
│  - events chan Event       // 事件通道                      │
│  - autoApprove bool        // YOLO 模式                     │
└─────────────────────────────────────────────────────────────┘
```

**替代方案**:
- 直接在 Agent 中管理进程 → 职责混乱，难以测试
- 使用外部进程池 → 过度设计

### 2. CLI 参数构建

**决策**: 使用以下参数启动 Claude 进程

```go
args := []string{
    "claude",
    "--output-format", "stream-json",
    "--input-format", "stream-json",
    "--permission-prompt-tool", "stdio",
    "--permission-mode", mode,  // 可选
    "--resume", sessionID,      // 可选，恢复会话
}
```

**理由**:
- `--input-format stream-json`: 启用 stdin JSON 输入
- `--permission-prompt-tool stdio`: 权限请求通过 stdin/stdout
- `--resume`: 保持会话连续性

### 3. 事件类型与处理

**决策**: 定义事件类型枚举和处理器

```go
type EventType string

const (
    EventTypeSystem            EventType = "system"
    EventTypeAssistant         EventType = "assistant"
    EventTypeUser              EventType = "user"
    EventTypeResult            EventType = "result"
    EventTypeControlRequest    EventType = "control_request"
    EventTypeControlCancel     EventType = "control_cancel_request"
)

type Event struct {
    Type       EventType
    SessionID  string
    Content    string
    ToolName   string
    ToolInput  string
    RequestID  string  // for control_request
    // ...
}
```

**处理流程**:

```
stdout ──► scanner ──► parse JSON ──► handle by type
                                        │
            ┌───────────────────────────┼───────────────────────────┐
            ▼                           ▼                           ▼
     system/init              assistant/text              control_request
     (store sessionID)        (emit to events)            (respond permission)
```

### 4. 权限请求处理

**决策**:
- YOLO 模式：自动批准所有权限请求
- 非 YOLO 模式：暂时自动拒绝（用户交互在后续阶段）

```go
func (s *Session) handleControlRequest(raw map[string]any) {
    requestID := raw["request_id"].(string)

    if s.autoApprove {
        // 自动批准
        s.respondPermission(requestID, PermissionResult{
            Behavior: "allow",
        })
    } else {
        // 暂时自动拒绝
        s.respondPermission(requestID, PermissionResult{
            Behavior: "deny",
            Message:  "请使用 YOLO 模式或等待权限交互功能上线",
        })
    }
}
```

**响应格式**:
```json
{
  "type": "control_response",
  "response": {
    "subtype": "success",
    "request_id": "<request_id>",
    "response": {
      "behavior": "allow",
      "updatedInput": {}
    }
  }
}
```

### 5. 消息发送格式

**决策**: 使用 JSON 格式发送用户消息

```go
func (s *Session) Send(prompt string, images []ImageAttachment, files []FileAttachment) error {
    if len(images) == 0 && len(files) == 0 {
        return s.writeJSON(map[string]any{
            "type": "user",
            "message": map[string]any{
                "role": "user",
                "content": prompt,
            },
        })
    }

    // 多模态消息
    var parts []map[string]any
    for _, img := range images {
        parts = append(parts, map[string]any{
            "type": "image",
            "source": map[string]any{
                "type": "base64",
                "media_type": img.MimeType,
                "data": base64.StdEncoding.EncodeToString(img.Data),
            },
        })
    }
    parts = append(parts, map[string]any{
        "type": "text",
        "text": prompt,
    })

    return s.writeJSON(map[string]any{
        "type": "user",
        "message": map[string]any{
            "role": "user",
            "content": parts,
        },
    })
}
```

### 6. 包结构与文件划分

**决策**: 重构现有文件结构

```
internal/agent/claudecode/
├── agent.go           # Agent 接口实现（修改）
├── session.go         # 新增：Session 会话管理
├── session_test.go    # 新增：Session 测试
├── events.go          # 修改：事件类型定义
├── parser.go          # 修改：事件解析
├── process.go         # 删除或重构：进程管理逻辑移入 session.go
├── permission.go      # 保留：权限模式映射
├── mock_agent.go      # 修改：更新 mock
└── doc.go             # 保留
```

## Risks / Trade-offs

### 风险 1: 进程泄漏

**风险**: 如果 Session.Close() 未正确调用，Claude 进程可能泄漏

**缓解**:
- 在 Agent.Stop() 中确保调用 Session.Close()
- 使用 context 管理进程生命周期
- 添加进程健康检查和超时清理

### 风险 2: 并发写入 stdin

**风险**: 多个 goroutine 同时写入 stdin 可能导致数据竞争

**缓解**:
- 使用 sync.Mutex 保护 stdin 写入
- 所有写入通过 writeJSON 方法

### 风险 3: 事件解析失败

**风险**: Claude 输出格式变化导致解析失败

**缓解**:
- 解析时记录原始行，便于调试
- 对未知事件类型记录日志但不中断
- 添加集成测试验证真实 Claude 输出

### 风险 4: 权限请求阻塞

**风险**: 如果 control_response 未及时发送，Claude 可能阻塞

**缓解**:
- 在 readLoop 中立即处理 control_request
- 设置响应超时
- YOLO 模式下自动批准避免阻塞

### Trade-off: 用户交互延迟

**权衡**: 本次实现暂不支持用户交互式权限确认

**理由**:
- 需要与 Feishu 平台深度集成
- 增加复杂度，影响 MVP 交付
- YOLO 模式可满足大部分使用场景
- 用户交互功能可在后续迭代中添加

## Migration Plan

### 阶段 1: 添加 Session 实现（不破坏现有功能）

1. 创建 `session.go`，实现 Session 结构体
2. 添加 Session 测试
3. 更新 `events.go`，添加新事件类型

### 阶段 2: 切换 Agent 到使用 Session

1. 修改 `agent.go`，使用 Session 替代 ProcessManager
2. 更新 Agent 测试
3. 更新 mock_agent.go

### 阶段 3: 清理

1. 删除或重构 `process.go`
2. 更新集成测试
3. 验证端到端流程

### 回滚策略

如果新实现出现问题：
1. 恢复 `agent.go` 使用 ProcessManager
2. Session 相关代码可保留但不使用
3. 通过配置开关选择使用哪种模式（可选）

## Open Questions

1. **会话恢复策略**: 当进程意外退出时，如何恢复会话？
   - 当前设计：使用 `--resume` 恢复
   - 需要验证：Claude CLI 的 resume 行为

2. **多会话支持**: 是否需要支持多个并发会话？
   - 当前设计：每个 Agent 一个会话
   - 后续可能需要：会话池或按 channel 隔离
