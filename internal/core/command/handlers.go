package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/smy-101/cc-connect/internal/agent"
	"github.com/smy-101/cc-connect/internal/agent/claudecode"
	"github.com/smy-101/cc-connect/internal/core"
)

// handleHelp returns help information about available commands
func (e *Executor) handleHelp(ctx context.Context, cmd Command, msg *core.Message) CommandResult {
	helpText := `可用命令:

/mode [mode]  - 切换权限模式
  可用模式: default, edit, plan, yolo
  无参数时显示当前模式

/new [name]   - 创建新会话
  清除当前上下文，开始新对话
  可选参数: 会话名称

/list         - 列出所有活跃会话

/help         - 显示此帮助信息

/stop         - 停止当前 Agent

/project [name] [--keep|-k] - 项目管理
  无参数时显示项目列表
  /project <name> - 切换到指定项目
  --keep / -k - 切换时保留会话

Claude Code 命令 (双斜杠 //):
//cost        显示 token 使用统计
//compact     压缩对话历史
//review      请求代码审查
//init        初始化项目 CLAUDE.md
//pr-comments 查看 PR 评论
//security-review 安全审查

提示: 使用 // 前缀调用 Claude Code 的原生命令
`
	return CommandResult{
		Message: fmt.Sprintf("```\n%s```", helpText),
	}
}

// handleMode handles the /mode command for switching permission modes
func (e *Executor) handleMode(ctx context.Context, cmd Command, msg *core.Message) CommandResult {
	// No args - show current mode
	if len(cmd.Args) == 0 {
		currentMode := e.agent.CurrentMode()
		return CommandResult{
			Message: fmt.Sprintf("当前权限模式: %s", currentMode),
		}
	}

	modeArg := cmd.Args[0]
	mode, err := claudecode.ParsePermissionMode(modeArg)
	if err != nil {
		return CommandResult{
			Message: fmt.Sprintf("无效的权限模式: %s\n可用模式: default, edit, plan, yolo", modeArg),
			Error:   err,
		}
	}

	if err := e.agent.SetPermissionMode(mode); err != nil {
		return CommandResult{
			Message: fmt.Sprintf("切换模式失败: %v", err),
			Error:   err,
		}
	}

	desc := claudecode.PermissionModeDescription(mode)
	return CommandResult{
		Message: fmt.Sprintf("✅ 已切换到 %s 模式\n%s", modeArg, desc),
	}
}

// handleNew handles the /new command for creating new sessions
func (e *Executor) handleNew(ctx context.Context, cmd Command, msg *core.Message) CommandResult {
	sessionID := core.DeriveSessionID(msg)

	// Destroy old session if exists (ignore error if not found)
	_ = e.sessions.Destroy(sessionID)

	// Create new session
	newSession := e.sessions.GetOrCreate(sessionID)

	// Set name if provided
	var sessionName string
	if len(cmd.Args) > 0 {
		sessionName = cmd.Args[0]
		newSession.SetMetadata("name", sessionName)
		// Update the session in the manager
		e.sessions.Update(sessionID, func(s *core.Session) {
			s.SetMetadata("name", sessionName)
		})
	}

	if sessionName != "" {
		return CommandResult{Message: fmt.Sprintf("✅ 已创建新会话: %s", sessionName)}
	}
	return CommandResult{Message: "✅ 已创建新会话"}
}

// handleList handles the /list command for listing sessions
func (e *Executor) handleList(ctx context.Context, cmd Command, msg *core.Message) CommandResult {
	sessions := e.sessions.List()

	if len(sessions) == 0 {
		return CommandResult{Message: "当前没有活跃会话"}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("活跃会话 (%d):\n", len(sessions)))
	for _, s := range sessions {
		name := s.Metadata["name"]
		if name == "" {
			name = string(s.ID)
		}
		sb.WriteString(fmt.Sprintf("  - %s\n", name))
	}

	return CommandResult{Message: sb.String()}
}

// handleStop handles the /stop command for stopping the agent
func (e *Executor) handleStop(ctx context.Context, cmd Command, msg *core.Message) CommandResult {
	// Check if agent is running
	if e.agent.Status() != agent.AgentStatusRunning {
		return CommandResult{Message: "Agent 未运行"}
	}

	// Stop the agent
	if err := e.agent.Stop(); err != nil {
		return CommandResult{
			Message: fmt.Sprintf("停止 Agent 失败: %v", err),
			Error:   err,
		}
	}

	return CommandResult{Message: "✅ Agent 已停止"}
}

// handleAllow handles the /allow command for approving permission requests
func (e *Executor) handleAllow(ctx context.Context, cmd Command, msg *core.Message) CommandResult {
	if len(cmd.Args) < 1 {
		return CommandResult{
			Message: "❌ 请提供请求 ID\n用法: /allow <request_id>",
			Error:   fmt.Errorf("missing request ID"),
		}
	}

	requestID := cmd.Args[0]
	if err := e.agent.RespondPermission(requestID, "allow"); err != nil {
		return CommandResult{
			Message: fmt.Sprintf("❌ 批准失败: %v", err),
			Error:   err,
		}
	}

	return CommandResult{Message: fmt.Sprintf("✅ 已批准请求: %s", requestID)}
}

// handleDeny handles the /deny command for rejecting permission requests
func (e *Executor) handleDeny(ctx context.Context, cmd Command, msg *core.Message) CommandResult {
	if len(cmd.Args) < 1 {
		return CommandResult{
			Message: "❌ 请提供请求 ID\n用法: /deny <request_id>",
			Error:   fmt.Errorf("missing request ID"),
		}
	}

	requestID := cmd.Args[0]
	if err := e.agent.RespondPermission(requestID, "deny"); err != nil {
		return CommandResult{
			Message: fmt.Sprintf("❌ 拒绝失败: %v", err),
			Error:   err,
		}
	}

	return CommandResult{Message: fmt.Sprintf("✅ 已拒绝请求: %s", requestID)}
}

// handleAnswer handles the /answer command for responding to AskUserQuestion
func (e *Executor) handleAnswer(ctx context.Context, cmd Command, msg *core.Message) CommandResult {
	if len(cmd.Args) < 2 {
		return CommandResult{
			Message: "❌ 请提供请求 ID 和答案\n用法: /answer <request_id> <answer>",
			Error:   fmt.Errorf("missing request ID or answer"),
		}
	}

	requestID := cmd.Args[0]
	answer := strings.Join(cmd.Args[1:], " ")

	if err := e.agent.RespondPermission(requestID, "answer:"+answer); err != nil {
		return CommandResult{
			Message: fmt.Sprintf("❌ 回答失败: %v", err),
			Error:   err,
		}
	}

	return CommandResult{Message: fmt.Sprintf("✅ 已回答: %s", answer)}
}
