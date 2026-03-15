// Package agent provides AI agent adapters for cc-connect.
//
// The agent package defines the Agent interface that all AI agent implementations
// must satisfy. It provides a unified abstraction for sending messages to AI agents
// and receiving responses, supporting both streaming and blocking modes.
//
// The package also provides AgentManager for managing multiple agent instances
// across different sessions.
//
// Currently supported agents:
//   - Claude Code CLI (internal/agent/claudecode)
package agent
