# cc-connect

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

cc-connect 是一个基于 Go 语言开发的桥接工具，用于连接本地 AI 代理（如 Claude Code）与聊天平台（如飞书），实现从任意聊天应用控制 AI 代理的能力。

## 功能特性

- 🤖 **AI 代理集成**：支持 Claude Code CLI，可扩展支持其他 AI 代理
- 💬 **多平台支持**：目前支持飞书 WebSocket 长连接，易于扩展
- 🔄 **会话管理**：自动管理对话上下文，支持多用户隔离
- ⚡ **斜杠命令**：内置 `/mode`、`/help`、`/new`、`/list`、`/stop` 等命令
- 🔐 **权限模式**：支持多种权限模式（default、plan、yolo 等）
- 🛡️ **优雅关闭**：支持 SIGINT/SIGTERM 信号的优雅关闭

## 架构概览

```
┌─────────────────────────────────────────────────────────────┐
│                      cmd/cc-connect                         │
│                     (CLI 入口点)                             │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     internal/app                            │
│              (应用整合层 - 生命周期管理)                      │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────────┐    │
│  │ Router  │  │  Agent  │  │ Feishu  │  │  Executor   │    │
│  │         │  │         │  │ Adapter │  │  (command)  │    │
│  └────┬────┘  └────┬────┘  └────┬────┘  └──────┬──────┘    │
└───────┼────────────┼────────────┼──────────────┼───────────┘
        │            │            │              │
        ▼            ▼            ▼              ▼
┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────┐
│   core    │ │   agent   │ │  feishu   │ │  command  │
│ (消息路由) │ │(AI 代理)  │ │ (飞书适配)│ │ (命令执行)│
└───────────┘ └───────────┘ └───────────┘ └───────────┘
```

## 项目结构

```
cc-connect/
├── cmd/
│   └── cc-connect/          # CLI 入口点
│       └── main.go
├── internal/
│   ├── app/                 # 应用整合层
│   │   ├── app.go           # App 结构和生命周期
│   │   ├── handlers.go      # 消息处理器
│   │   ├── context.go       # HandlerContext
│   │   └── reply.go         # ReplySender 接口
│   ├── core/                # 核心域
│   │   ├── message.go       # 统一消息结构
│   │   ├── router.go        # 消息路由器
│   │   ├── session.go       # 会话管理
│   │   ├── config.go        # 配置管理
│   │   └── command/         # 斜杠命令
│   ├── platform/            # 平台适配器
│   │   └── feishu/          # 飞书适配器
│   └── agent/               # AI 代理适配器
│       └── claudecode/      # Claude Code 适配器
├── test/
│   └── e2e/                 # 端到端测试
├── config.example.toml      # 示例配置文件
└── README.md
```

## 快速开始

### 前置要求

- Go 1.21 或更高版本
- [Claude Code CLI](https://github.com/anthropics/claude-code) 已安装
- 飞书开放平台应用（启用 WebSocket 长连接）

### 安装

```bash
# 克隆仓库
git clone https://github.com/smy-101/cc-connect.git
cd cc-connect

# 安装依赖
go mod download

# 编译
go build -o cc-connect ./cmd/cc-connect
```

### 配置

1. 复制示例配置文件：

```bash
cp config.example.toml config.toml
```

2. 编辑配置文件，填入飞书应用凭证：

```toml
log_level = "info"
default_project = "my-project"

[[projects]]
name = "my-project"
description = "我的项目"
working_dir = "/path/to/your/project"

[projects.feishu]
app_id = "${FEISHU_APP_ID}"      # 支持环境变量
app_secret = "${FEISHU_APP_SECRET}"
enabled = true

[projects.claude_code]
default_permission_mode = "default"  # default, plan, yolo, bypassPermissions
enabled = true
```

3. 设置环境变量：

```bash
export FEISHU_APP_ID="your-app-id"
export FEISHU_APP_SECRET="your-app-secret"
```

### 运行

```bash
# 使用默认配置文件 ./config.toml
./cc-connect

# 指定配置文件路径
./cc-connect -config /path/to/config.toml

# 查看版本信息
./cc-connect --version
```

## 斜杠命令

| 命令 | 说明 | 示例 |
|------|------|------|
| `/help` | 显示可用命令列表 | `/help` |
| `/mode <mode>` | 切换权限模式 | `/mode yolo` |
| `/new` | 创建新会话 | `/new` |
| `/list` | 列出所有会话 | `/list` |
| `/stop` | 停止当前处理 | `/stop` |

### 权限模式说明

| 模式 | 说明 |
|------|------|
| `default` | 所有工具操作需要确认 |
| `plan` | 自动批准只读工具 |
| `acceptEdits` | 自动批准编辑工具 |
| `yolo` / `bypassPermissions` | 自动批准所有操作 |

## 开发指南

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行带覆盖率的测试
go test ./... -cover

# 运行特定包的测试
go test ./internal/app/... -v

# 运行 E2E 测试
go test ./test/e2e/... -v
```

### 测试覆盖率

| 包 | 覆盖率 |
|----|--------|
| internal/agent | 90.9% |
| internal/app | 81.4% |
| internal/core | 93.0% |
| internal/core/command | 93.3% |
| internal/platform/feishu | 92.0% |

### 代码规范

本项目遵循 TDD（测试驱动开发）方法论：

1. **红-绿-重构循环**：先写失败的测试，再实现最小代码，最后重构
2. **覆盖率目标**：核心功能 > 80%
3. **测试隔离**：所有涉及 WebSocket、子进程、网络调用的测试使用 Mock

### 添加新的平台适配器

1. 在 `internal/platform/` 下创建新目录
2. 实现 `FeishuClient` 接口（参考 `feishu/` 目录）
3. 在 `app.go` 中集成新适配器

### 添加新的 AI 代理

1. 在 `internal/agent/` 下创建新目录
2. 实现 `agent.Agent` 接口
3. 在 `app.go` 中使用新代理

## 配置参考

### 应用级配置

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `log_level` | string | 否 | `info` | 日志级别：debug, info, warn, error |
| `default_project` | string | 否 | - | 默认项目名称 |

### 项目配置

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 项目名称 |
| `description` | string | 否 | 项目描述 |
| `working_dir` | string | 是 | 工作目录路径 |
| `feishu` | FeishuConfig | 是 | 飞书配置 |
| `claude_code` | ClaudeCodeConfig | 是 | Claude Code 配置 |

### 飞书配置

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `app_id` | string | 是 | 飞书应用 ID |
| `app_secret` | string | 是 | 飞书应用密钥 |
| `enabled` | bool | 否 | 是否启用（默认 false） |

### Claude Code 配置

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `default_permission_mode` | string | 否 | `default` | 默认权限模式 |
| `enabled` | bool | 否 | false | 是否启用 |

## 常见问题

### Q: 飞书连接失败怎么办？

1. 检查 `app_id` 和 `app_secret` 是否正确
2. 确认飞书应用已启用 WebSocket 长连接
3. 检查网络连接是否正常

### Q: Claude Code 启动失败？

1. 确认 Claude Code CLI 已正确安装
2. 检查 `working_dir` 路径是否存在
3. 确认有足够的权限访问工作目录

### Q: 消息没有响应？

1. 检查日志输出，确认消息是否被接收
2. 确认 Agent 是否正常运行
3. 检查是否触发了权限确认（需要用户批准）

## 路线图

- [x] 核心消息系统
- [x] 飞书适配器
- [x] Claude Code 适配器
- [x] 斜杠命令系统
- [x] 应用整合层
- [ ] TUI 终端界面
- [ ] 多项目支持
- [ ] 多机器人中继
- [ ] 语音消息支持
- [ ] 完整的 Cron 任务系统

## 贡献指南

欢迎贡献代码！请遵循以下步骤：

1. Fork 本仓库
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。

## 致谢

- [Claude Code](https://github.com/anthropics/claude-code) - Anthropic 的 AI 编程助手
- [飞书开放平台](https://open.feishu.cn/) - 提供强大的即时通讯能力
