// Package claudecode provides a Claude Code CLI adapter for cc-connect.
//
// This package implements the agent.Agent interface for Claude Code CLI,
// providing persistent process management, streaming output parsing,
// and permission mode control.
//
// # Key Features
//
//   - Persistent process: Each session maintains a long-running Claude Code process
//   - Streaming output: Parses JSONL format output from --output-format stream-json
//   - Permission modes: Supports default, acceptEdits, plan, bypassPermissions modes
//   - Session persistence: Leverages Claude Code's built-in session storage
//   - Crash recovery: Automatically restarts crashed processes with --resume
//
// # Usage
//
//	config := &claudecode.Config{
//	    WorkingDir:      "/path/to/project",
//	    PermissionMode:  agent.PermissionModeDefault,
//	}
//	agent := claudecode.NewAgent(config)
//
//	if err := agent.Start(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer agent.Stop()
//
//	resp, err := agent.SendMessage(ctx, "Fix the bug in main.go", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(resp.Content)
package claudecode
