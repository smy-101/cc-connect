# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

cc-connect is a Go-based bridge tool that connects local AI agents (Claude Code, etc.) with chat platforms (Feishu/飞书, etc.), enabling control of AI agents from any chat application. The project follows strict TDD methodology.

## Development Commands

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run tests for a specific package
go test ./internal/core/... -v

# Run a single test
go test -run TestMessageSystem ./internal/core/...

# Run integration tests (requires build tag)
go test ./internal/platform/... -tags=integration -v

# Run benchmarks
go test ./internal/... -bench=. -benchtime=5s
```

## Architecture

```
internal/
├── core/           # Core domain: unified message structure, routing, session management, config
├── platform/       # Platform adapters: Feishu WebSocket, etc.
├── agent/          # AI agent adapters: Claude Code subprocess management
└── tui/            # Terminal UI (planned)
test/
└── e2e/            # End-to-end tests
```

### Key Domain Concepts

- **Unified Message Model**: Supports `text`, `voice`, `image`, `command` types
- **Message Serialization**: Must be compatible with Python version (fields: id, platform, user_id, content, type, timestamp)
- **Permission Modes**: `default`, `edit`/`acceptEdits`, `plan`, `yolo`/`bypassPermissions`
- **Slash Commands**: `/mode`, `/new`, `/list`, `/help`, `/allow`, `/deny`, `/answer`, `/stop`, `/provider`, `/cron`

## Interactive Permission Handling

The system supports interactive permission requests from Claude Code, allowing users to approve/deny tool usage and answer questions via chat cards.

### Permission Flow

1. **Tool Permission Request**: Claude requests to use a tool (e.g., Bash, Read)
2. **Card Display**: System sends an interactive card to Feishu with Allow/Deny buttons
3. **User Response**: User clicks button or uses `/allow` or `/deny` command
4. **Execution Continues**: Claude receives the response and continues

### AskUserQuestion Flow

1. **Question**: Claude asks a question with options
2. **Option Card**: System displays buttons for each option
3. **Answer**: User clicks an option or uses `/answer` command
4. **Response**: Claude receives the answer and continues

### Slash Commands for Permissions

| Command | Description |
|---------|-------------|
| `/allow <request_id>` | Approve a pending permission request |
| `/deny <request_id>` | Deny a pending permission request |
| `/answer <request_id> <answer>` | Answer an AskUserQuestion request |

### Permission Modes

| Mode | Description |
|------|-------------|
| `default` | Interactive mode - prompts for permission |
| `plan` | Plan mode - shows plan before execution |
| `edit` | Edit mode - allows file edits |
| `acceptEdits` | Auto-accept edit mode |
| `yolo` / `bypassPermissions` | Auto-approve all requests |

### Key Files

| File | Purpose |
|------|---------|
| `internal/agent/claudecode/session.go` | Permission state machine, pending permission handling |
| `internal/core/permission_handler.go` | Permission request event handling |
| `internal/core/card.go` | Platform-agnostic card structure |
| `internal/core/question_card.go` | AskUserQuestion card generation |
| `internal/platform/feishu/card_callback.go` | Card callback parsing |
| `internal/platform/feishu/card_renderer.go` | Card JSON rendering for Feishu |

## OpenSpec Workflow

This project uses OpenSpec for structured change management. Use the opsx commands:

| Command | Purpose |
|---------|---------|
| `/opsx:new` | Start a new change (creates scaffold) |
| `/opsx:explore` | Think through ideas before creating a change |
| `/opsx:continue` | Create the next artifact in a change |
| `/opsx:apply` | Implement tasks from a change |
| `/opsx:verify` | Verify implementation matches artifacts |
| `/opsx:archive` | Archive a completed change |
| `/opsx:propose` | Create change + all artifacts in one step |

**Typical workflow**: `/opsx:new` → `/opsx:continue` (for each artifact) → `/opsx:apply` → `/opsx:verify` → `/opsx:archive`

## TDD Requirements

- **Strict red-green-refactor cycle**: Write failing test first, minimal implementation, then refactor
- **Coverage target**: > 85% for core functionality
- **Test isolation**: All tests involving WebSocket, subprocess, scheduler, or network must use mocks/stubs
- **No public IP dependency**: Feishu adapter uses WebSocket (no public IP required)

## Configuration Reference

The configuration system uses TOML format with environment variable expansion support.

### Configuration File Structure

```toml
# Application-level config
log_level = "info"              # debug, info, warn, error
default_project = "my-project"  # Default project name

# Project configurations
[[projects]]
name = "my-project"
description = "My project description"
working_dir = "/home/user/workspace"

[projects.feishu]
app_id = "${FEISHU_APP_ID}"      # Environment variable expansion
app_secret = "${FEISHU_APP_SECRET}"
enabled = true

[projects.claude_code]
default_permission_mode = "default"  # default, edit, acceptEdits, plan, yolo, bypassPermissions
enabled = true
```

### Key Types

- `AppConfig` - Application-level configuration
- `ProjectConfig` - Per-project configuration
- `FeishuConfig` - Feishu platform settings
- `ClaudeCodeConfig` - Claude Code agent settings
- `SessionConfigOpt` - Optional session config overrides

### Environment Variables

Use `${VAR}` syntax for environment variable expansion. Missing variables become empty strings with validation warnings.

### Example Configuration

See `config.example.toml` for a complete example.

For OpenSpec configuration, see `openspec/config.yaml`.

## Development Phases

1. **Core message system** - Unified message structure, routing, sessions, config
2. **Feishu adapter** - WebSocket connection, message handling, @mentions
3. **Claude Code adapter** - Subprocess management, streaming output, permission modes
4. **Slash commands** - Mode switching, session management, provider/cron commands
5. **TUI** - Message monitor, session management, config view
6. **Advanced** - Multi-project, multi-bot relay, voice, full cron system

Current focus: MVP with Feishu + Claude Code only.

## Feishu Platform Adapter

The Feishu adapter (`internal/platform/feishu/`) provides integration with Feishu's WebSocket long connection API.

### Key Components

| File | Purpose |
|------|---------|
| `client.go` | FeishuClient interface definition |
| `client_impl.go` | SDK-based client implementation |
| `mock_client.go` | Mock client for testing |
| `message_converter.go` | Message format conversion (Feishu ↔ Unified) |
| `event_parser.go` | Parse `im.message.receive_v1` events |
| `sender.go` | Send messages to Feishu API |
| `adapter.go` | Integrate with core.Router |
| `async_processor.go` | Async event processing for fast ACK |

### Usage Example

```go
// Create client
client := feishu.NewSDKClient(appID, appSecret)

// Create adapter with router
router := core.NewRouter()
adapter := feishu.NewAdapter(client, router)

// Register handler for text messages
router.Register(core.MessageTypeText, func(ctx context.Context, msg *core.Message) error {
    // Handle the message
    return adapter.SendReply(ctx, msg.ChannelID, "Echo: "+msg.Content)
})

// Start the connection
if err := adapter.Start(ctx); err != nil {
    log.Fatal(err)
}
```

### Testing

```bash
# Run feishu package tests
go test ./internal/platform/feishu/... -v

# Run with race detection
go test ./internal/platform/feishu/... -race

# Run integration tests (requires build tag)
go test ./internal/platform/feishu/... -tags=integration -v
```
