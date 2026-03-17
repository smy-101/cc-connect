// Package claudecode provides a Claude Code CLI adapter for cc-connect.
//
// This package implements the agent.Agent interface for Claude Code CLI,
// providing persistent session management, bidirectional communication,
// streaming output parsing, and permission mode control.
//
// # Key Features
//
//   - Interactive stream mode: Uses --input-format stream-json --permission-prompt-tool stdio
//     for persistent bidirectional communication with Claude Code
//   - Session management: Maintains a long-running session process with stdin/stdout pipes
//   - Permission handling: Supports YOLO mode (auto-approve) and interactive permission prompts
//   - Streaming events: Parses JSONL format output from --output-format stream-json
//   - Permission modes: Supports default, acceptEdits, plan, bypassPermissions modes
//   - Session persistence: Leverages Claude Code's built-in session storage with --resume
//
// # Architecture
//
// The package consists of three main components:
//
//   - ClaudeCodeAgent: Implements the agent.Agent interface
//   - Session: Manages the persistent Claude Code process with bidirectional communication
//   - Event parsing: Handles stream-json events including control_request for permissions
//
// # Usage
//
//	config := &claudecode.Config{
//	    WorkingDir:      "/path/to/project",
//	    PermissionMode:  agent.PermissionModeBypassPermissions,
//	}
//	ag, err := claudecode.NewAgent(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	if err := ag.Start(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer ag.Stop()
//
//	resp, err := ag.SendMessage(ctx, "Fix the bug in main.go", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(resp.Content)
//
// # Permission Handling
//
// In bypassPermissions (YOLO) mode, all permission requests are automatically approved.
// In other modes, permission requests are automatically denied with a message directing
// users to use YOLO mode until interactive permission support is implemented.
package claudecode
