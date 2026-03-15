package claudecode

import (
	"context"
	"testing"

	"github.com/smy-101/cc-connect/internal/agent"
	"github.com/smy-101/cc-connect/internal/core"
)

// TestSessionIntegration tests the integration between Claude Code agent and core.Session
func TestSessionIntegration(t *testing.T) {
	t.Run("Session.AgentID stores Claude Code session-id", func(t *testing.T) {
		// Create a mock agent with a specific session ID
		config := &Config{
			SessionID:      "test-session-123",
			WorkingDir:     t.TempDir(),
			PermissionMode: agent.PermissionModeDefault,
		}
		mockAgent := NewMockAgent(config)

		// Start the agent
		err := mockAgent.Start(context.Background())
		if err != nil {
			t.Fatalf("Failed to start agent: %v", err)
		}

		// Create a session and bind the agent
		session := core.NewSession("feishu:user:ou_test")
		session.BindAgent(mockAgent.SessionID())

		// Verify the session ID is stored correctly
		if session.AgentID != "test-session-123" {
			t.Errorf("Expected AgentID to be 'test-session-123', got '%s'", session.AgentID)
		}
	})

	t.Run("Session.PermissionMode syncs with Agent", func(t *testing.T) {
		// Create a mock agent
		config := &Config{
			WorkingDir:     t.TempDir(),
			PermissionMode: agent.PermissionModeDefault,
		}
		mockAgent := NewMockAgent(config)

		// Start the agent
		err := mockAgent.Start(context.Background())
		if err != nil {
			t.Fatalf("Failed to start agent: %v", err)
		}

		// Create a session and set permission mode
		session := core.NewSession("feishu:user:ou_test")
		session.SetPermissionMode(string(mockAgent.CurrentMode()))

		// Verify initial mode
		if session.PermissionMode != string(agent.PermissionModeDefault) {
			t.Errorf("Expected PermissionMode to be 'default', got '%s'", session.PermissionMode)
		}

		// Change agent mode
		err = mockAgent.SetPermissionMode(agent.PermissionModeBypassPermissions)
		if err != nil {
			t.Fatalf("Failed to set permission mode: %v", err)
		}

		// Sync to session
		session.SetPermissionMode(string(mockAgent.CurrentMode()))

		// Verify mode sync
		if session.PermissionMode != string(agent.PermissionModeBypassPermissions) {
			t.Errorf("Expected PermissionMode to be 'bypassPermissions', got '%s'", session.PermissionMode)
		}
	})

	t.Run("Session.Metadata approved_tools accumulates", func(t *testing.T) {
		// Create a mock agent
		config := &Config{
			WorkingDir:     t.TempDir(),
			PermissionMode: agent.PermissionModeDefault,
		}
		mockAgent := NewMockAgent(config)

		// Start the agent
		err := mockAgent.Start(context.Background())
		if err != nil {
			t.Fatalf("Failed to start agent: %v", err)
		}

		// Create a session
		session := core.NewSession("feishu:user:ou_test")

		// Configure mock to return permission denied
		mockAgent.SetPermissionDenied([]agent.DeniedTool{
			{
				ToolName:  "Bash",
				ToolUseID: "toolu_1",
				ToolInput: map[string]interface{}{"command": "ls"},
			},
		})

		// Send message and get permission denied
		resp, err := mockAgent.SendMessage(context.Background(), "test", nil)
		if err != nil {
			t.Fatalf("SendMessage failed: %v", err)
		}

		if !resp.PermissionDenied {
			t.Fatal("Expected PermissionDenied to be true")
		}

		// Simulate user approval: add to approved tools
		if resp.PermissionDenied && len(resp.DeniedTools) > 0 {
			// Accumulate approved tools in session metadata
			existingTools := session.Metadata["approved_tools"]
			if existingTools != "" {
				session.SetMetadata("approved_tools", existingTools+",Bash")
			} else {
				session.SetMetadata("approved_tools", "Bash")
			}
		}

		// Verify approved tools accumulated
		if session.Metadata["approved_tools"] != "Bash" {
			t.Errorf("Expected approved_tools to be 'Bash', got '%s'", session.Metadata["approved_tools"])
		}

		// Second permission denial and approval
		mockAgent.Reset()
		mockAgent.SetPermissionDenied([]agent.DeniedTool{
			{
				ToolName:  "Write",
				ToolUseID: "toolu_2",
				ToolInput: map[string]interface{}{"file_path": "/tmp/test"},
			},
		})

		resp, err = mockAgent.SendMessage(context.Background(), "test2", nil)
		if err != nil {
			t.Fatalf("SendMessage failed: %v", err)
		}

		if !resp.PermissionDenied {
			t.Fatal("Expected PermissionDenied to be true")
		}

		// Accumulate second tool
		if resp.PermissionDenied && len(resp.DeniedTools) > 0 {
			existingTools := session.Metadata["approved_tools"]
			if existingTools != "" {
				session.SetMetadata("approved_tools", existingTools+","+resp.DeniedTools[0].ToolName)
			} else {
				session.SetMetadata("approved_tools", resp.DeniedTools[0].ToolName)
			}
		}

		// Verify both tools accumulated
		expected := "Bash,Write"
		if session.Metadata["approved_tools"] != expected {
			t.Errorf("Expected approved_tools to be '%s', got '%s'", expected, session.Metadata["approved_tools"])
		}
	})

	t.Run("Full session lifecycle with agent", func(t *testing.T) {
		// Create session manager
		sm := core.NewSessionManager(core.DefaultSessionConfig())

		// Create mock agent
		config := &Config{
			WorkingDir:     t.TempDir(),
			PermissionMode: agent.PermissionModeDefault,
		}
		mockAgent := NewMockAgent(config)

		// Start the agent
		err := mockAgent.Start(context.Background())
		if err != nil {
			t.Fatalf("Failed to start agent: %v", err)
		}

		// Create/get session
		sessionID := core.SessionID("feishu:user:ou_lifecycle")
		_ = sm.GetOrCreate(sessionID) // Create the session

		// Bind agent to session
		sm.Update(sessionID, func(s *core.Session) {
			s.BindAgent(mockAgent.SessionID())
			s.SetPermissionMode(string(mockAgent.CurrentMode()))
		})

		// Verify binding
		updated, ok := sm.Get(sessionID)
		if !ok {
			t.Fatal("Session not found")
		}

		if updated.AgentID != mockAgent.SessionID() {
			t.Errorf("Expected AgentID to be '%s', got '%s'", mockAgent.SessionID(), updated.AgentID)
		}

		if updated.PermissionMode != string(agent.PermissionModeDefault) {
			t.Errorf("Expected PermissionMode to be 'default', got '%s'", updated.PermissionMode)
		}
	})
}

// TestRouterIntegration tests the integration between agent and core.Router
func TestRouterIntegration(t *testing.T) {
	t.Run("Router registers text message handler with agent", func(t *testing.T) {
		// Create router
		router := core.NewRouter()

		// Create mock agent
		config := &Config{
			WorkingDir:     t.TempDir(),
			PermissionMode: agent.PermissionModeDefault,
		}
		mockAgent := NewMockAgent(config)

		// Start the agent
		err := mockAgent.Start(context.Background())
		if err != nil {
			t.Fatalf("Failed to start agent: %v", err)
		}

		// Register handler that uses the agent
		err = router.RegisterSessionHandler(core.MessageTypeText, func(ctx context.Context, msg *core.Message, session *core.Session) error {
			// Bind agent to session
			session.BindAgent(mockAgent.SessionID())
			session.SetPermissionMode(string(mockAgent.CurrentMode()))

			// Send message to agent
			_, err := mockAgent.SendMessage(ctx, msg.Content, nil)
			return err
		})
		if err != nil {
			t.Fatalf("Failed to register handler: %v", err)
		}

		// Verify handler works by routing a message
		msg := &core.Message{
			ID:       "msg-test",
			Platform: "feishu",
			UserID:   "ou_test",
			Content:  "test",
			Type:     core.MessageTypeText,
		}

		// This should succeed without error (no ErrNoHandler)
		err = router.RouteWithSession(context.Background(), msg)
		if err != nil {
			t.Errorf("Expected handler to be registered and work, got error: %v", err)
		}
	})

	t.Run("Message flow: Router → Agent → Response", func(t *testing.T) {
		// Create router
		router := core.NewRouter()

		// Create mock agent with preset response
		config := &Config{
			WorkingDir:     t.TempDir(),
			PermissionMode: agent.PermissionModeDefault,
		}
		mockAgent := NewMockAgent(config)
		mockAgent.SetResponse(&agent.Response{
			Content: "Hello from Claude!",
		})

		// Start the agent
		err := mockAgent.Start(context.Background())
		if err != nil {
			t.Fatalf("Failed to start agent: %v", err)
		}

		// Track response
		var capturedResponse string
		var capturedSessionID string

		// Register handler that uses the agent
		err = router.RegisterSessionHandler(core.MessageTypeText, func(ctx context.Context, msg *core.Message, session *core.Session) error {
			// Bind agent to session
			session.BindAgent(mockAgent.SessionID())

			// Send message to agent
			resp, err := mockAgent.SendMessage(ctx, msg.Content, nil)
			if err != nil {
				return err
			}

			capturedResponse = resp.Content
			capturedSessionID = session.AgentID
			return nil
		})
		if err != nil {
			t.Fatalf("Failed to register handler: %v", err)
		}

		// Create a message
		msg := &core.Message{
			ID:        "msg-1",
			Platform:  "feishu",
			UserID:    "ou_user1",
			ChannelID: "",
			Content:   "Hello",
			Type:      core.MessageTypeText,
		}

		// Route the message
		err = router.RouteWithSession(context.Background(), msg)
		if err != nil {
			t.Fatalf("Failed to route message: %v", err)
		}

		// Verify response
		if capturedResponse != "Hello from Claude!" {
			t.Errorf("Expected response 'Hello from Claude!', got '%s'", capturedResponse)
		}

		// Verify session was bound to agent
		if capturedSessionID != mockAgent.SessionID() {
			t.Errorf("Expected session AgentID to be '%s', got '%s'", mockAgent.SessionID(), capturedSessionID)
		}
	})

	t.Run("Router handles agent errors gracefully", func(t *testing.T) {
		// Create router
		router := core.NewRouter()

		// Create mock agent that returns errors
		config := &Config{
			WorkingDir:     t.TempDir(),
			PermissionMode: agent.PermissionModeDefault,
		}
		mockAgent := NewMockAgent(config)
		mockAgent.SetError(agent.ErrAgentBusy)

		// Start the agent
		err := mockAgent.Start(context.Background())
		if err != nil {
			t.Fatalf("Failed to start agent: %v", err)
		}

		// Register handler
		err = router.RegisterSessionHandler(core.MessageTypeText, func(ctx context.Context, msg *core.Message, session *core.Session) error {
			_, err := mockAgent.SendMessage(ctx, msg.Content, nil)
			return err
		})
		if err != nil {
			t.Fatalf("Failed to register handler: %v", err)
		}

		// Create a message
		msg := &core.Message{
			ID:       "msg-1",
			Platform: "feishu",
			UserID:   "ou_user1",
			Content:  "Hello",
			Type:     core.MessageTypeText,
		}

		// Route the message - should return error
		err = router.RouteWithSession(context.Background(), msg)
		if err == nil {
			t.Fatal("Expected error from agent, got nil")
		}

		if err != agent.ErrAgentBusy {
			t.Errorf("Expected ErrAgentBusy, got %v", err)
		}
	})

	t.Run("Router handles permission denied response", func(t *testing.T) {
		// Create router
		router := core.NewRouter()

		// Create mock agent that returns permission denied
		config := &Config{
			WorkingDir:     t.TempDir(),
			PermissionMode: agent.PermissionModeDefault,
		}
		mockAgent := NewMockAgent(config)
		mockAgent.SetPermissionDenied([]agent.DeniedTool{
			{
				ToolName:  "Bash",
				ToolUseID: "toolu_1",
				ToolInput: map[string]interface{}{"command": "rm -rf /"},
			},
		})

		// Start the agent
		err := mockAgent.Start(context.Background())
		if err != nil {
			t.Fatalf("Failed to start agent: %v", err)
		}

		var deniedTools []agent.DeniedTool

		// Register handler that processes permission denied
		err = router.RegisterSessionHandler(core.MessageTypeText, func(ctx context.Context, msg *core.Message, session *core.Session) error {
			resp, err := mockAgent.SendMessage(ctx, msg.Content, nil)
			if err != nil {
				return err
			}

			if resp.PermissionDenied {
				deniedTools = resp.DeniedTools
				// In a real scenario, we would ask the user for approval here
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Failed to register handler: %v", err)
		}

		// Create a message
		msg := &core.Message{
			ID:       "msg-1",
			Platform: "feishu",
			UserID:   "ou_user1",
			Content:  "Delete everything",
			Type:     core.MessageTypeText,
		}

		// Route the message
		err = router.RouteWithSession(context.Background(), msg)
		if err != nil {
			t.Fatalf("Failed to route message: %v", err)
		}

		// Verify permission denied was captured
		if len(deniedTools) != 1 {
			t.Fatalf("Expected 1 denied tool, got %d", len(deniedTools))
		}

		if deniedTools[0].ToolName != "Bash" {
			t.Errorf("Expected denied tool 'Bash', got '%s'", deniedTools[0].ToolName)
		}
	})

	t.Run("Streaming events through router", func(t *testing.T) {
		// Create router
		router := core.NewRouter()

		// Create mock agent with streaming events
		config := &Config{
			WorkingDir:     t.TempDir(),
			PermissionMode: agent.PermissionModeDefault,
		}
		mockAgent := NewMockAgent(config)
		mockAgent.SetStreamEvents([]agent.StreamEvent{
			{Type: agent.StreamEventTypeText, Content: "Thinking..."},
			{Type: agent.StreamEventTypeToolUse, Tool: &agent.ToolInfo{Name: "Read", ID: "toolu_1"}},
			{Type: agent.StreamEventTypeResult, Content: "Done!"},
		})
		mockAgent.SetResponse(&agent.Response{
			Content: "Final response",
		})

		// Start the agent
		err := mockAgent.Start(context.Background())
		if err != nil {
			t.Fatalf("Failed to start agent: %v", err)
		}

		var capturedEvents []agent.StreamEvent

		// Register handler with streaming
		err = router.RegisterSessionHandler(core.MessageTypeText, func(ctx context.Context, msg *core.Message, session *core.Session) error {
			_, err := mockAgent.SendMessage(ctx, msg.Content, func(event agent.StreamEvent) {
				capturedEvents = append(capturedEvents, event)
			})
			return err
		})
		if err != nil {
			t.Fatalf("Failed to register handler: %v", err)
		}

		// Create a message
		msg := &core.Message{
			ID:       "msg-1",
			Platform: "feishu",
			UserID:   "ou_user1",
			Content:  "Analyze this",
			Type:     core.MessageTypeText,
		}

		// Route the message
		err = router.RouteWithSession(context.Background(), msg)
		if err != nil {
			t.Fatalf("Failed to route message: %v", err)
		}

		// Verify events were captured
		if len(capturedEvents) != 3 {
			t.Fatalf("Expected 3 events, got %d", len(capturedEvents))
		}

		if capturedEvents[0].Type != agent.StreamEventTypeText {
			t.Errorf("Expected first event to be text, got %s", capturedEvents[0].Type)
		}

		if capturedEvents[1].Type != agent.StreamEventTypeToolUse {
			t.Errorf("Expected second event to be tool_use, got %s", capturedEvents[1].Type)
		}

		if capturedEvents[2].Type != agent.StreamEventTypeResult {
			t.Errorf("Expected third event to be result, got %s", capturedEvents[2].Type)
		}
	})
}
