package claudecode

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/smy-101/cc-connect/internal/agent"
)

// SessionEventType represents the type of event from the Session
type SessionEventType int

const (
	// EventSystem indicates a system event (session init, etc.)
	EventSystem SessionEventType = iota
	// EventText indicates a text content event
	EventText
	// EventToolUse indicates a tool use event
	EventToolUse
	// EventToolResult indicates a tool result event
	EventToolResult
	// EventResult indicates the final result event
	EventResult
	// EventError indicates an error event
	EventError
	// EventPermissionRequest indicates a permission request event
	EventPermissionRequest
)

// Event represents an event from the Session event stream
type Event struct {
	// Type is the event type
	Type SessionEventType
	// SessionID is the current session ID
	SessionID string
	// Content is the text content (for EventText, EventResult, EventError)
	Content string
	// ToolName is the tool name (for EventToolUse, EventPermissionRequest)
	ToolName string
	// ToolInput is the tool input (for EventToolUse, EventPermissionRequest)
	ToolInput map[string]interface{}
	// RequestID is the permission request ID (for EventPermissionRequest)
	RequestID string
	// Error contains the error if any
	Error error
	// Done indicates if this is the final event
	Done bool
}

// ImageAttachment represents an image to be sent with a message
type ImageAttachment struct {
	// MimeType is the MIME type (e.g., "image/png", "image/jpeg")
	MimeType string
	// Data is the raw image data
	Data []byte
}

// FileAttachment represents a file to be sent with a message
type FileAttachment struct {
	// Path is the file path
	Path string
	// Data is the file content (optional, for in-memory files)
	Data []byte
}

// SessionConfig contains configuration for creating a Session
type SessionConfig struct {
	// WorkingDir is the working directory for the Claude process
	WorkingDir string
	// ClaudePath is the path to claude CLI (defaults to "claude")
	ClaudePath string
	// SessionID is the session ID to resume (optional)
	SessionID string
	// PermissionMode is the permission mode
	PermissionMode string
	// AutoApprove enables YOLO mode (auto-approve all permission requests)
	AutoApprove bool
	// AllowedTools is a list of pre-approved tools
	AllowedTools []string
	// Env is additional environment variables
	Env []string
}

// Session manages communication with Claude Code CLI
// It uses a persistent process with bidirectional communication via stdin/stdout
type Session struct {
	config *SessionConfig
	mu     sync.Mutex

	// Process management
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdinMu sync.Mutex
	// stdout is read in readLoop
	events chan Event

	// Current session ID (updated from system events)
	sessionID   atomic.Value // stores string
	autoApprove bool
	alive       atomic.Bool

	// Context for process lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// newSession creates a new Session with a persistent Claude process
func newSession(ctx context.Context, config *SessionConfig) (*Session, error) {
	if config.WorkingDir == "" {
		return nil, fmt.Errorf("working directory is required")
	}

	if config.ClaudePath == "" {
		config.ClaudePath = DefaultClaudePath()
	}

	// Check if claude CLI exists
	if _, err := exec.LookPath(config.ClaudePath); err != nil {
		return nil, fmt.Errorf("claude CLI not found: %w", err)
	}

	// Build command arguments for interactive stream mode
	args := []string{
		"--output-format", "stream-json",
		"--verbose",
		"--input-format", "stream-json",
		"--permission-prompt-tool", "stdio",
	}

	if config.PermissionMode != "" && config.PermissionMode != "default" {
		args = append(args, "--permission-mode", config.PermissionMode)
	}
	if config.SessionID != "" {
		args = append(args, "--resume", config.SessionID)
	}
	if len(config.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(config.AllowedTools, ","))
	}

	slog.Debug("Starting Claude session", "args", args, "dir", config.WorkingDir)

	// Create context for process lifecycle
	sessionCtx, cancel := context.WithCancel(ctx)

	cmd := exec.CommandContext(sessionCtx, config.ClaudePath, args...)
	cmd.Dir = config.WorkingDir

	// Filter out CLAUDECODE env var to prevent "nested session" detection
	env := filterEnv(os.Environ(), "CLAUDECODE")
	if len(config.Env) > 0 {
		env = mergeEnv(env, config.Env)
	}
	cmd.Env = env

	// Create stdin pipe
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Create stdout pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Capture stderr
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	// Start the process
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	session := &Session{
		config:      config,
		cmd:         cmd,
		stdin:       stdin,
		events:      make(chan Event, 64),
		autoApprove: config.AutoApprove,
		ctx:         sessionCtx,
		cancel:      cancel,
		done:        make(chan struct{}),
	}
	session.sessionID.Store(config.SessionID)
	session.alive.Store(true)

	// Start read loop in background
	go session.readLoop(stdout, &stderrBuf)

	slog.Debug("Session created", "session_id", config.SessionID)
	return session, nil
}

// readLoop continuously reads events from stdout
func (s *Session) readLoop(stdout io.ReadCloser, stderrBuf *bytes.Buffer) {
	defer func() {
		s.alive.Store(false)
		if err := s.cmd.Wait(); err != nil {
			stderrMsg := strings.TrimSpace(stderrBuf.String())
			if stderrMsg != "" {
				slog.Error("Claude process failed", "error", err, "stderr", stderrMsg)
				evt := Event{Type: EventError, Error: fmt.Errorf("%s", stderrMsg)}
				select {
				case s.events <- evt:
				case <-s.ctx.Done():
					return
				}
			}
		}
		close(s.events)
		close(s.done)
	}()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			slog.Debug("Non-JSON line from stdout", "line", line)
			continue
		}

		eventType, _ := raw["type"].(string)
		slog.Debug("Claude event received", "type", eventType)

		switch eventType {
		case "system":
			s.handleSystem(raw)
		case "assistant":
			s.handleAssistant(raw)
		case "user":
			// User events are echoes, not typically needed
		case "result":
			s.handleResult(raw)
		case "control_request":
			s.handleControlRequest(raw)
		case "control_cancel_request":
			requestID, _ := raw["request_id"].(string)
			slog.Debug("Permission request cancelled", "request_id", requestID)
		}
	}

	if err := scanner.Err(); err != nil {
		slog.Error("Scanner error", "error", err)
		evt := Event{Type: EventError, Error: fmt.Errorf("read stdout: %w", err)}
		select {
		case s.events <- evt:
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Session) handleSystem(raw map[string]interface{}) {
	if sid, ok := raw["session_id"].(string); ok && sid != "" {
		s.sessionID.Store(sid)
		slog.Debug("Session ID updated", "session_id", sid)
	}
}

func (s *Session) handleAssistant(raw map[string]interface{}) {
	msg, ok := raw["message"].(map[string]interface{})
	if !ok {
		return
	}
	contentArr, ok := msg["content"].([]interface{})
	if !ok {
		return
	}
	for _, contentItem := range contentArr {
		item, ok := contentItem.(map[string]interface{})
		if !ok {
			continue
		}
		contentType, _ := item["type"].(string)
		switch contentType {
		case "tool_use":
			toolName, _ := item["name"].(string)
			inputSummary := summarizeInput(toolName, item["input"])
			evt := Event{Type: EventToolUse, ToolName: toolName, ToolInput: item["input"].(map[string]interface{})}
			_ = inputSummary // Keep for logging if needed
			select {
			case s.events <- evt:
			case <-s.ctx.Done():
				return
			}
		case "thinking":
			if thinking, ok := item["thinking"].(string); ok && thinking != "" {
				evt := Event{Type: EventText, Content: thinking}
				select {
				case s.events <- evt:
				case <-s.ctx.Done():
					return
				}
			}
		case "text":
			if text, ok := item["text"].(string); ok && text != "" {
				evt := Event{Type: EventText, Content: text}
				select {
				case s.events <- evt:
				case <-s.ctx.Done():
					return
				}
			}
		}
	}
}

func (s *Session) handleResult(raw map[string]interface{}) {
	var content string
	if result, ok := raw["result"].(string); ok {
		content = result
	}
	if sid, ok := raw["session_id"].(string); ok && sid != "" {
		s.sessionID.Store(sid)
	}
	evt := Event{Type: EventResult, Content: content, SessionID: s.CurrentSessionID(), Done: true}
	select {
	case s.events <- evt:
	case <-s.ctx.Done():
		return
	}
}

func (s *Session) handleControlRequest(raw map[string]interface{}) {
	requestID, _ := raw["request_id"].(string)
	request, _ := raw["request"].(map[string]interface{})
	if request == nil {
		return
	}
	subtype, _ := request["subtype"].(string)
	if subtype != "can_use_tool" {
		slog.Debug("Unknown control request subtype", "subtype", subtype)
		return
	}

	toolName, _ := request["tool_name"].(string)
	input, _ := request["input"].(map[string]interface{})

	// Auto mode: approve immediately without asking the user
	if s.autoApprove {
		slog.Debug("Auto-approving permission request", "request_id", requestID, "tool", toolName)
		_ = s.RespondPermission(requestID, PermissionResult{
			Behavior:     "allow",
			UpdatedInput: input,
		})
		return
	}

	slog.Info("Permission request received", "request_id", requestID, "tool", toolName)
	evt := Event{
		Type:      EventPermissionRequest,
		RequestID: requestID,
		ToolName:  toolName,
		ToolInput: input,
	}

	select {
	case s.events <- evt:
	case <-s.ctx.Done():
		return
	}
}

// summarizeInput produces a short human-readable description of tool input
func summarizeInput(tool string, input interface{}) string {
	m, ok := input.(map[string]interface{})
	if !ok {
		return ""
	}

	switch tool {
	case "Read", "Edit", "Write":
		if fp, ok := m["file_path"].(string); ok {
			return fp
		}
	case "Bash":
		if cmd, ok := m["command"].(string); ok {
			return cmd
		}
	case "Grep":
		if p, ok := m["pattern"].(string); ok {
			return p
		}
	case "Glob":
		if p, ok := m["pattern"].(string); ok {
			return p
		}
		if p, ok := m["glob_pattern"].(string); ok {
			return p
		}
	}

	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(b)
}

// Events returns the event channel
func (s *Session) Events() <-chan Event {
	return s.events
}

// Send sends a user message to the Claude process stdin
func (s *Session) Send(prompt string, images []ImageAttachment, files []FileAttachment) error {
	if !s.alive.Load() {
		return fmt.Errorf("session process is not running")
	}

	if len(images) == 0 && len(files) == 0 {
		return s.writeJSON(map[string]interface{}{
			"type": "user",
			"message": map[string]interface{}{
				"role":    "user",
				"content": prompt,
			},
		})
	}

	// Multi-modal message
	var parts []map[string]interface{}

	// Encode images as base64
	for _, img := range images {
		mimeType := img.MimeType
		if mimeType == "" {
			mimeType = "image/png"
		}
		parts = append(parts, map[string]interface{}{
			"type": "image",
			"source": map[string]interface{}{
				"type":       "base64",
				"media_type": mimeType,
				"data":       base64.StdEncoding.EncodeToString(img.Data),
			},
		})
	}

	// Add text part
	textPart := prompt
	if textPart == "" && (len(images) > 0 || len(files) > 0) {
		textPart = "Please analyze the attached content."
	}
	parts = append(parts, map[string]interface{}{
		"type": "text",
		"text": textPart,
	})

	return s.writeJSON(map[string]interface{}{
		"type": "user",
		"message": map[string]interface{}{
			"role":    "user",
			"content": parts,
		},
	})
}

// SendMessage sends a message and collects the response via event callback
func (s *Session) SendMessage(ctx context.Context, prompt string, images []ImageAttachment, files []FileAttachment, eventHandler func(Event)) (*Event, error) {
	if !s.alive.Load() {
		return nil, fmt.Errorf("session process is not running")
	}

	// Send the message
	if err := s.Send(prompt, images, files); err != nil {
		return nil, fmt.Errorf("send failed: %w", err)
	}

	// Collect events until we get a result or error
	var finalEvent *Event
	var textBuilder strings.Builder

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-s.ctx.Done():
			return nil, fmt.Errorf("session closed")
		case event, ok := <-s.events:
			if !ok {
				// Channel closed, session ended
				if finalEvent != nil {
					return finalEvent, nil
				}
				return nil, fmt.Errorf("session ended unexpectedly")
			}

			// Call event handler if provided
			if eventHandler != nil && event.Type != EventResult {
				eventHandler(event)
			}

			// Accumulate text
			if event.Type == EventText {
				textBuilder.WriteString(event.Content)
			}

			// Capture session ID updates
			if event.SessionID != "" {
				s.sessionID.Store(event.SessionID)
			}

			// Check for final events
			if event.Type == EventResult || event.Type == EventError {
				// If result is empty but we have accumulated text, use that
				if event.Type == EventResult && event.Content == "" && textBuilder.Len() > 0 {
					event.Content = textBuilder.String()
				}
				finalEvent = &event
				return finalEvent, nil
			}
		}
	}
}

// RespondPermission sends a control_response to the Claude process stdin
func (s *Session) RespondPermission(requestID string, result PermissionResult) error {
	if !s.alive.Load() {
		return fmt.Errorf("session process is not running")
	}

	var permResponse map[string]interface{}
	if result.Behavior == "allow" {
		updatedInput := result.UpdatedInput
		if updatedInput == nil {
			updatedInput = make(map[string]interface{})
		}
		permResponse = map[string]interface{}{
			"behavior":     "allow",
			"updatedInput": updatedInput,
		}
	} else {
		msg := result.Message
		if msg == "" {
			msg = "The user denied this tool use. Stop and wait for the user's instructions."
		}
		permResponse = map[string]interface{}{
			"behavior": "deny",
			"message":  msg,
		}
	}

	controlResponse := map[string]interface{}{
		"type": "control_response",
		"response": map[string]interface{}{
			"subtype":    "success",
			"request_id": requestID,
			"response":   permResponse,
		},
	}

	slog.Debug("Sending permission response", "request_id", requestID, "behavior", result.Behavior)
	return s.writeJSON(controlResponse)
}

func (s *Session) writeJSON(v interface{}) error {
	s.stdinMu.Lock()
	defer s.stdinMu.Unlock()

	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if _, err := s.stdin.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write stdin: %w", err)
	}
	return nil
}

// Close closes the session gracefully
func (s *Session) Close() error {
	s.cancel()

	select {
	case <-s.done:
		return nil
	case <-time.After(8 * time.Second):
		slog.Warn("Graceful close timed out, killing process")
		if s.cmd != nil && s.cmd.Process != nil {
			_ = s.cmd.Process.Kill()
		}
		<-s.done
		return nil
	}
}

// Alive returns true if the session is active
func (s *Session) Alive() bool {
	return s.alive.Load()
}

// CurrentSessionID returns the current session ID
func (s *Session) CurrentSessionID() string {
	v, _ := s.sessionID.Load().(string)
	return v
}

// PermissionResult represents the result of a permission request
type PermissionResult struct {
	Behavior     string                 `json:"behavior"`
	Message      string                 `json:"message,omitempty"`
	UpdatedInput map[string]interface{} `json:"updatedInput,omitempty"`
}

// filterEnv returns a copy of env with entries matching the given key removed
func filterEnv(env []string, key string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, prefix) {
			out = append(out, e)
		}
	}
	return out
}

// mergeEnv merges extra environment variables into env
func mergeEnv(env []string, extra []string) []string {
	// Build a map of extra env vars
	extraMap := make(map[string]string)
	for _, e := range extra {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			extraMap[parts[0]] = parts[1]
		}
	}

	// Update existing or append new
	result := make([]string, 0, len(env)+len(extra))
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) >= 1 {
			if newVal, ok := extraMap[parts[0]]; ok {
				// Replace with new value
				result = append(result, parts[0]+"="+newVal)
				delete(extraMap, parts[0])
				continue
			}
		}
		result = append(result, e)
	}

	// Append remaining extra vars
	for k, v := range extraMap {
		result = append(result, k+"="+v)
	}

	return result
}

// convertToAgentStreamEvent converts a Session Event to agent.StreamEvent
func convertToAgentStreamEvent(event Event) agent.StreamEvent {
	ae := agent.StreamEvent{}

	switch event.Type {
	case EventSystem:
		ae.Type = agent.StreamEventTypeSystem
	case EventText:
		ae.Type = agent.StreamEventTypeText
		ae.Content = event.Content
	case EventToolUse:
		ae.Type = agent.StreamEventTypeToolUse
		ae.Tool = &agent.ToolInfo{
			Name:  event.ToolName,
			Input: event.ToolInput,
		}
	case EventToolResult:
		ae.Type = agent.StreamEventTypeToolResult
	case EventResult:
		ae.Type = agent.StreamEventTypeResult
		ae.Content = event.Content
	case EventError:
		ae.Type = agent.StreamEventTypeError
		ae.Content = event.Content
	}

	return ae
}
