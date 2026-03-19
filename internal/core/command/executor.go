package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/smy-101/cc-connect/internal/agent"
	"github.com/smy-101/cc-connect/internal/core"
)

// Executor executes slash commands and returns results.
// It holds references to the Agent and SessionManager for command operations.
type Executor struct {
	agent    agent.Agent
	sessions *core.SessionManager
	router   *core.ProjectRouter // Optional: for project switching
}

// NewExecutor creates a new command executor.
func NewExecutor(ag agent.Agent, sessions *core.SessionManager) *Executor {
	return &Executor{
		agent:    ag,
		sessions: sessions,
	}
}

// NewExecutorWithRouter creates a new command executor with a ProjectRouter.
// This enables the /project command functionality.
func NewExecutorWithRouter(router *core.ProjectRouter) *Executor {
	return &Executor{
		router: router,
	}
}

// SetAgent sets the agent for the executor.
func (e *Executor) SetAgent(ag agent.Agent) {
	e.agent = ag
}

// SetSessions sets the session manager for the executor.
func (e *Executor) SetSessions(sessions *core.SessionManager) {
	e.sessions = sessions
}

// Execute executes a command and returns the result.
// It routes commands to the appropriate handler based on the command name.
func (e *Executor) Execute(ctx context.Context, cmd Command, msg *core.Message) CommandResult {
	// Handle empty command
	if cmd.IsEmpty() {
		return CommandResult{
			Message: "无效的命令",
			Error:   ErrEmptyCommand,
		}
	}

	// Route to handlers
	switch cmd.Name {
	case "help":
		return e.handleHelp(ctx, cmd, msg)
	case "mode":
		return e.handleMode(ctx, cmd, msg)
	case "new":
		return e.handleNew(ctx, cmd, msg)
	case "list":
		return e.handleList(ctx, cmd, msg)
	case "stop":
		return e.handleStop(ctx, cmd, msg)
	case "project":
		return e.handleProject(ctx, cmd, msg)
	case "allow":
		return e.handleAllow(ctx, cmd, msg)
	case "deny":
		return e.handleDeny(ctx, cmd, msg)
	case "answer":
		return e.handleAnswer(ctx, cmd, msg)
	default:
		return CommandResult{
			Message: "未知命令: /" + cmd.Name + "\n输入 /help 查看可用命令",
			Error:   ErrUnknownCommand,
		}
	}
}

// handleProject handles the /project command for project switching
func (e *Executor) handleProject(ctx context.Context, cmd Command, msg *core.Message) CommandResult {
	// Check if router is available
	if e.router == nil {
		return CommandResult{
			Message: "❌ 多项目功能未启用",
			Error:   fmt.Errorf("project router not configured"),
		}
	}

	// No args - show current project and list
	if len(cmd.Args) == 0 {
		return e.showProjectList()
	}

	// Get target project name
	targetName := cmd.Args[0]

	// Check if --keep or -k flag is set
	keepSession := cmd.HasFlag("keep") || cmd.HasFlag("k")

	// Switch project
	return e.switchProject(ctx, targetName, keepSession)
}

// showProjectList returns a list of all projects with current project marked
func (e *Executor) showProjectList() CommandResult {
	projects := e.router.ListProjects()

	var sb strings.Builder
	sb.WriteString("📋 项目列表:\n")

	for _, p := range projects {
		if p.IsActive {
			sb.WriteString(fmt.Sprintf("  • %s (当前)\n", p.Name))
		} else {
			sb.WriteString(fmt.Sprintf("  • %s\n", p.Name))
		}
	}

	sb.WriteString("\n用法: /project <项目名> [--keep|-k]")

	return CommandResult{Message: sb.String()}
}

// switchProject switches to the specified project
func (e *Executor) switchProject(ctx context.Context, name string, keepSession bool) CommandResult {
	// Check if project exists
	project, exists := e.router.GetProject(name)
	if !exists {
		// Return error with available projects
		projectNames := e.router.ProjectNames()
		return CommandResult{
			Message: fmt.Sprintf("❌ 项目 %q 不存在\n\n可用项目: %s", name, strings.Join(projectNames, ", ")),
			Error:   core.ErrProjectNotFound,
		}
	}

	// Check if already active
	active := e.router.ActiveProject()
	if active != nil && active.Name == name {
		return CommandResult{
			Message: fmt.Sprintf("✅ 已是当前项目: %s", name),
		}
	}

	// Switch project
	if err := e.router.SwitchProject(ctx, name, keepSession); err != nil {
		return CommandResult{
			Message: fmt.Sprintf("❌ 切换项目失败: %v", err),
			Error:   err,
		}
	}

	// Update executor's agent and sessions to the new project's
	newProject, _ := e.router.GetProject(name)
	if newProject != nil {
		e.agent = newProject.Agent()
		e.sessions = newProject.Sessions()
	}

	sessionInfo := ""
	if keepSession {
		sessionInfo = " (保留会话)"
	}

	return CommandResult{
		Message: fmt.Sprintf("✅ 已切换到项目: %s%s\n工作目录: %s", name, sessionInfo, project.Config.WorkingDir),
	}
}
