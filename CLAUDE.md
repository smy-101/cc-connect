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
- **Slash Commands**: `/mode`, `/new`, `/list`, `/help`, `/allow`, `/stop`, `/provider`, `/cron`

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
