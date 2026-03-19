package core

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/smy-101/cc-connect/internal/agent"
)

// mockAgent implements agent.Agent for testing
type mockAgent struct {
	status     atomic.Value // agent.AgentStatus
	started    atomic.Bool
	stopped    atomic.Bool
	startErr   error
	stopErr    error
	startCalls atomic.Int32
	stopCalls  atomic.Int32
}

func newMockAgent() *mockAgent {
	m := &mockAgent{}
	m.status.Store(agent.AgentStatusIdle)
	return m
}

func (m *mockAgent) SendMessage(ctx context.Context, content string, handler agent.EventHandler) (*agent.Response, error) {
	return &agent.Response{Content: "mock response"}, nil
}

func (m *mockAgent) SetPermissionMode(mode agent.PermissionMode) error {
	return nil
}

func (m *mockAgent) CurrentMode() agent.PermissionMode {
	return agent.PermissionModeDefault
}

func (m *mockAgent) SessionID() string {
	return "mock-session-id"
}

func (m *mockAgent) Start(ctx context.Context) error {
	m.startCalls.Add(1)
	if m.startErr != nil {
		return m.startErr
	}
	m.started.Store(true)
	m.status.Store(agent.AgentStatusRunning)
	return nil
}

func (m *mockAgent) Stop() error {
	m.stopCalls.Add(1)
	if m.stopErr != nil {
		return m.stopErr
	}
	m.stopped.Store(true)
	m.status.Store(agent.AgentStatusStopped)
	return nil
}

func (m *mockAgent) Status() agent.AgentStatus {
	return m.status.Load().(agent.AgentStatus)
}

func (m *mockAgent) Restart(ctx context.Context) error {
	return nil
}

func (m *mockAgent) RespondPermission(requestID, behavior string) error {
	return nil
}

// TestProjectStatus tests ProjectStatus constants
func TestProjectStatus(t *testing.T) {
	tests := []struct {
		name   string
		status ProjectStatus
		want   string
	}{
		{"idle", ProjectStatusIdle, "idle"},
		{"active", ProjectStatusActive, "active"},
		{"switching", ProjectStatusSwitching, "switching"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.want {
				t.Errorf("ProjectStatus%q = %q, want %q", tt.name, tt.status, tt.want)
			}
		})
	}
}

// TestNewProject tests creating a new Project
func TestNewProject(t *testing.T) {
	t.Run("creates project with config", func(t *testing.T) {
		config := &ProjectConfig{
			Name:       "test-project",
			WorkingDir: "/tmp/test",
		}

		project := NewProject(config)

		if project == nil {
			t.Fatal("NewProject should not return nil")
		}
		if project.Name != "test-project" {
			t.Errorf("project.Name = %q, want %q", project.Name, "test-project")
		}
		if project.Config != config {
			t.Error("project.Config should be the same as input config")
		}
		if project.Status() != ProjectStatusIdle {
			t.Errorf("new project status = %q, want %q", project.Status(), ProjectStatusIdle)
		}
		if project.Sessions() == nil {
			t.Error("project should have a SessionManager")
		}
	})

	t.Run("creates project with session config", func(t *testing.T) {
		sessionConfig := SessionConfig{
			ActiveTTL:       10 * time.Minute,
			ArchivedTTL:     1 * time.Hour,
			CleanupInterval: 2 * time.Minute,
		}
		config := &ProjectConfig{
			Name:       "test-project",
			WorkingDir: "/tmp/test",
		}

		project := NewProject(config, WithSessionConfig(sessionConfig))

		// Verify session manager was created (we can't directly check config)
		if project.Sessions() == nil {
			t.Error("project should have a SessionManager")
		}
	})
}

// TestProjectStatusManagement tests Project status management
func TestProjectStatusManagement(t *testing.T) {
	t.Run("initial status is idle", func(t *testing.T) {
		config := &ProjectConfig{Name: "test", WorkingDir: "/tmp"}
		project := NewProject(config)

		if project.Status() != ProjectStatusIdle {
			t.Errorf("initial status = %q, want %q", project.Status(), ProjectStatusIdle)
		}
	})

	t.Run("set status to active", func(t *testing.T) {
		config := &ProjectConfig{Name: "test", WorkingDir: "/tmp"}
		project := NewProject(config)

		project.SetStatus(ProjectStatusActive)

		if project.Status() != ProjectStatusActive {
			t.Errorf("status = %q, want %q", project.Status(), ProjectStatusActive)
		}
	})

	t.Run("set status to switching", func(t *testing.T) {
		config := &ProjectConfig{Name: "test", WorkingDir: "/tmp"}
		project := NewProject(config)

		project.SetStatus(ProjectStatusSwitching)

		if project.Status() != ProjectStatusSwitching {
			t.Errorf("status = %q, want %q", project.Status(), ProjectStatusSwitching)
		}
	})
}

// TestProjectGetOrCreateAgent tests lazy agent creation
func TestProjectGetOrCreateAgent(t *testing.T) {
	t.Run("creates agent on first call", func(t *testing.T) {
		config := &ProjectConfig{Name: "test", WorkingDir: "/tmp"}
		project := NewProject(config)

		factory := func(cfg *ProjectConfig) (agent.Agent, error) {
			return newMockAgent(), nil
		}

		ag, err := project.GetOrCreateAgent(context.Background(), factory)
		if err != nil {
			t.Fatalf("GetOrCreateAgent failed: %v", err)
		}
		if ag == nil {
			t.Error("agent should not be nil")
		}
	})

	t.Run("returns same agent on subsequent calls", func(t *testing.T) {
		config := &ProjectConfig{Name: "test", WorkingDir: "/tmp"}
		project := NewProject(config)

		callCount := 0
		factory := func(cfg *ProjectConfig) (agent.Agent, error) {
			callCount++
			return newMockAgent(), nil
		}

		ag1, _ := project.GetOrCreateAgent(context.Background(), factory)
		ag2, _ := project.GetOrCreateAgent(context.Background(), factory)

		if callCount != 1 {
			t.Errorf("factory called %d times, want 1", callCount)
		}
		if ag1 != ag2 {
			t.Error("should return same agent instance")
		}
	})

	t.Run("returns error if factory fails", func(t *testing.T) {
		config := &ProjectConfig{Name: "test", WorkingDir: "/tmp"}
		project := NewProject(config)

		expectedErr := errors.New("factory error")
		factory := func(cfg *ProjectConfig) (agent.Agent, error) {
			return nil, expectedErr
		}

		ag, err := project.GetOrCreateAgent(context.Background(), factory)

		if ag != nil {
			t.Error("agent should be nil on error")
		}
		if !errors.Is(err, expectedErr) {
			t.Errorf("err = %v, want %v", err, expectedErr)
		}
	})

	t.Run("starts agent after creation", func(t *testing.T) {
		config := &ProjectConfig{Name: "test", WorkingDir: "/tmp"}
		project := NewProject(config)

		mockAg := newMockAgent()
		factory := func(cfg *ProjectConfig) (agent.Agent, error) {
			return mockAg, nil
		}

		_, err := project.GetOrCreateAgent(context.Background(), factory)
		if err != nil {
			t.Fatalf("GetOrCreateAgent failed: %v", err)
		}

		if mockAg.startCalls.Load() != 1 {
			t.Errorf("agent.Start called %d times, want 1", mockAg.startCalls.Load())
		}
	})

	t.Run("returns error if agent start fails", func(t *testing.T) {
		config := &ProjectConfig{Name: "test", WorkingDir: "/tmp"}
		project := NewProject(config)

		mockAg := newMockAgent()
		mockAg.startErr = errors.New("start error")
		factory := func(cfg *ProjectConfig) (agent.Agent, error) {
			return mockAg, nil
		}

		_, err := project.GetOrCreateAgent(context.Background(), factory)

		if err == nil {
			t.Error("should return error when agent start fails")
		}
	})
}

// TestProjectClearSessions tests session clearing
func TestProjectClearSessions(t *testing.T) {
	t.Run("clears all sessions", func(t *testing.T) {
		config := &ProjectConfig{Name: "test", WorkingDir: "/tmp"}
		project := NewProject(config)

		// Create some sessions
		sessions := project.Sessions()
		sessions.GetOrCreate("session1")
		sessions.GetOrCreate("session2")
		sessions.GetOrCreate("session3")

		// Verify sessions exist
		if len(sessions.List()) != 3 {
			t.Fatalf("expected 3 sessions, got %d", len(sessions.List()))
		}

		// Clear sessions
		project.ClearSessions()

		// Get the new session manager after clearing
		newSessions := project.Sessions()
		if len(newSessions.List()) != 0 {
			t.Errorf("expected 0 sessions after clear, got %d", len(newSessions.List()))
		}
	})

	t.Run("clears sessions on empty project", func(t *testing.T) {
		config := &ProjectConfig{Name: "test", WorkingDir: "/tmp"}
		project := NewProject(config)

		// Should not panic
		project.ClearSessions()

		if len(project.Sessions().List()) != 0 {
			t.Error("sessions should be empty")
		}
	})
}

// TestProjectAgent tests agent getter
func TestProjectAgent(t *testing.T) {
	t.Run("returns nil when not created", func(t *testing.T) {
		config := &ProjectConfig{Name: "test", WorkingDir: "/tmp"}
		project := NewProject(config)

		if project.Agent() != nil {
			t.Error("agent should be nil before GetOrCreateAgent")
		}
	})

	t.Run("returns agent after creation", func(t *testing.T) {
		config := &ProjectConfig{Name: "test", WorkingDir: "/tmp"}
		project := NewProject(config)

		factory := func(cfg *ProjectConfig) (agent.Agent, error) {
			return newMockAgent(), nil
		}

		project.GetOrCreateAgent(context.Background(), factory)

		if project.Agent() == nil {
			t.Error("agent should not be nil after GetOrCreateAgent")
		}
	})
}

// TestProjectStopAgent tests stopping the agent
func TestProjectStopAgent(t *testing.T) {
	t.Run("stops running agent", func(t *testing.T) {
		config := &ProjectConfig{Name: "test", WorkingDir: "/tmp"}
		project := NewProject(config)

		mockAg := newMockAgent()
		factory := func(cfg *ProjectConfig) (agent.Agent, error) {
			return mockAg, nil
		}

		project.GetOrCreateAgent(context.Background(), factory)
		err := project.StopAgent()

		if err != nil {
			t.Errorf("StopAgent failed: %v", err)
		}
		if mockAg.stopCalls.Load() != 1 {
			t.Errorf("agent.Stop called %d times, want 1", mockAg.stopCalls.Load())
		}
	})

	t.Run("returns nil when no agent", func(t *testing.T) {
		config := &ProjectConfig{Name: "test", WorkingDir: "/tmp"}
		project := NewProject(config)

		err := project.StopAgent()

		if err != nil {
			t.Errorf("StopAgent should return nil when no agent, got: %v", err)
		}
	})

	t.Run("returns error if stop fails", func(t *testing.T) {
		config := &ProjectConfig{Name: "test", WorkingDir: "/tmp"}
		project := NewProject(config)

		mockAg := newMockAgent()
		mockAg.stopErr = errors.New("stop error")
		factory := func(cfg *ProjectConfig) (agent.Agent, error) {
			return mockAg, nil
		}

		project.GetOrCreateAgent(context.Background(), factory)
		err := project.StopAgent()

		if err == nil {
			t.Error("should return error when stop fails")
		}
	})
}

// TestProjectConcurrentAccess tests concurrent access to project
func TestProjectConcurrentAccess(t *testing.T) {
	t.Run("concurrent GetOrCreateAgent", func(t *testing.T) {
		config := &ProjectConfig{Name: "test", WorkingDir: "/tmp"}
		project := NewProject(config)

		var factoryCalls atomic.Int32
		factory := func(cfg *ProjectConfig) (agent.Agent, error) {
			factoryCalls.Add(1)
			time.Sleep(10 * time.Millisecond) // Simulate slow creation
			return newMockAgent(), nil
		}

		const goroutines = 10
		done := make(chan agent.Agent, goroutines)

		for i := 0; i < goroutines; i++ {
			go func() {
				ag, _ := project.GetOrCreateAgent(context.Background(), factory)
				done <- ag
			}()
		}

		var firstAgent agent.Agent
		for i := 0; i < goroutines; i++ {
			ag := <-done
			if firstAgent == nil {
				firstAgent = ag
			} else if ag != firstAgent {
				t.Error("all goroutines should get same agent")
			}
		}

		// Factory should only be called once due to lazy initialization
		if factoryCalls.Load() > 1 {
			t.Errorf("factory called %d times, should be at most 1", factoryCalls.Load())
		}
	})

	t.Run("concurrent status changes", func(t *testing.T) {
		config := &ProjectConfig{Name: "test", WorkingDir: "/tmp"}
		project := NewProject(config)

		const goroutines = 100
		done := make(chan bool, goroutines)

		for i := 0; i < goroutines; i++ {
			go func(idx int) {
				if idx%3 == 0 {
					project.SetStatus(ProjectStatusActive)
				} else if idx%3 == 1 {
					project.SetStatus(ProjectStatusSwitching)
				} else {
					_ = project.Status()
				}
				done <- true
			}(i)
		}

		for i := 0; i < goroutines; i++ {
			<-done
		}
	})
}

// TestNewProjectRouter tests creating a new ProjectRouter
func TestNewProjectRouter(t *testing.T) {
	t.Run("creates router with configs", func(t *testing.T) {
		configs := []ProjectConfig{
			{Name: "project1", WorkingDir: "/tmp/p1"},
			{Name: "project2", WorkingDir: "/tmp/p2"},
		}

		factory := func(cfg *ProjectConfig) (agent.Agent, error) {
			return newMockAgent(), nil
		}

		router, err := NewProjectRouter(configs, factory)

		if err != nil {
			t.Fatalf("NewProjectRouter failed: %v", err)
		}
		if router == nil {
			t.Fatal("router should not be nil")
		}

		names := router.ProjectNames()
		if len(names) != 2 {
			t.Errorf("expected 2 projects, got %d", len(names))
		}
	})

	t.Run("returns error with empty configs", func(t *testing.T) {
		factory := func(cfg *ProjectConfig) (agent.Agent, error) {
			return newMockAgent(), nil
		}

		_, err := NewProjectRouter([]ProjectConfig{}, factory)

		if err == nil {
			t.Error("should return error with empty configs")
		}
	})

	t.Run("returns error with nil factory", func(t *testing.T) {
		configs := []ProjectConfig{{Name: "test", WorkingDir: "/tmp"}}

		_, err := NewProjectRouter(configs, nil)

		if err == nil {
			t.Error("should return error with nil factory")
		}
	})
}

// TestProjectRouterGetProject tests GetProject
func TestProjectRouterGetProject(t *testing.T) {
	configs := []ProjectConfig{
		{Name: "frontend", WorkingDir: "/tmp/frontend"},
		{Name: "backend", WorkingDir: "/tmp/backend"},
	}

	factory := func(cfg *ProjectConfig) (agent.Agent, error) {
		return newMockAgent(), nil
	}

	router, _ := NewProjectRouter(configs, factory)

	t.Run("returns existing project", func(t *testing.T) {
		p, ok := router.GetProject("frontend")

		if !ok {
			t.Error("should find frontend project")
		}
		if p == nil {
			t.Fatal("project should not be nil")
		}
		if p.Name != "frontend" {
			t.Errorf("project.Name = %q, want %q", p.Name, "frontend")
		}
	})

	t.Run("returns false for non-existing project", func(t *testing.T) {
		p, ok := router.GetProject("unknown")

		if ok {
			t.Error("should not find unknown project")
		}
		if p != nil {
			t.Error("project should be nil")
		}
	})
}

// TestProjectRouterSetActiveProject tests SetActiveProject
func TestProjectRouterSetActiveProject(t *testing.T) {
	configs := []ProjectConfig{
		{Name: "frontend", WorkingDir: "/tmp/frontend"},
		{Name: "backend", WorkingDir: "/tmp/backend"},
	}

	factory := func(cfg *ProjectConfig) (agent.Agent, error) {
		return newMockAgent(), nil
	}

	router, _ := NewProjectRouter(configs, factory)

	t.Run("sets active project", func(t *testing.T) {
		err := router.SetActiveProject("backend")

		if err != nil {
			t.Fatalf("SetActiveProject failed: %v", err)
		}

		active := router.ActiveProject()
		if active == nil {
			t.Fatal("active project should not be nil")
		}
		if active.Name != "backend" {
			t.Errorf("active.Name = %q, want %q", active.Name, "backend")
		}
	})

	t.Run("returns error for non-existing project", func(t *testing.T) {
		err := router.SetActiveProject("unknown")

		if err == nil {
			t.Error("should return error for non-existing project")
		}
	})
}

// TestProjectRouterSwitchProject tests SwitchProject
func TestProjectRouterSwitchProject(t *testing.T) {
	t.Run("switches to new project", func(t *testing.T) {
		configs := []ProjectConfig{
			{Name: "frontend", WorkingDir: "/tmp/frontend"},
			{Name: "backend", WorkingDir: "/tmp/backend"},
		}

		var factoryCalls atomic.Int32
		factory := func(cfg *ProjectConfig) (agent.Agent, error) {
			factoryCalls.Add(1)
			return newMockAgent(), nil
		}

		router, _ := NewProjectRouter(configs, factory)
		router.SetActiveProject("frontend")

		// Start the initial project
		err := router.StartActiveProject(context.Background())
		if err != nil {
			t.Fatalf("StartActiveProject failed: %v", err)
		}

		// Switch to backend
		err = router.SwitchProject(context.Background(), "backend", false)

		if err != nil {
			t.Fatalf("SwitchProject failed: %v", err)
		}

		active := router.ActiveProject()
		if active.Name != "backend" {
			t.Errorf("active.Name = %q, want %q", active.Name, "backend")
		}
	})

	t.Run("returns error for non-existing project", func(t *testing.T) {
		configs := []ProjectConfig{
			{Name: "frontend", WorkingDir: "/tmp/frontend"},
		}

		factory := func(cfg *ProjectConfig) (agent.Agent, error) {
			return newMockAgent(), nil
		}

		router, _ := NewProjectRouter(configs, factory)
		router.SetActiveProject("frontend")

		err := router.SwitchProject(context.Background(), "unknown", false)

		if !errors.Is(err, ErrProjectNotFound) {
			t.Errorf("err = %v, want ErrProjectNotFound", err)
		}
	})

	t.Run("returns error when switching to current project", func(t *testing.T) {
		configs := []ProjectConfig{
			{Name: "frontend", WorkingDir: "/tmp/frontend"},
		}

		factory := func(cfg *ProjectConfig) (agent.Agent, error) {
			return newMockAgent(), nil
		}

		router, _ := NewProjectRouter(configs, factory)
		router.SetActiveProject("frontend")

		err := router.SwitchProject(context.Background(), "frontend", false)

		if !errors.Is(err, ErrProjectAlreadyActive) {
			t.Errorf("err = %v, want ErrProjectAlreadyActive", err)
		}
	})

	t.Run("clears sessions when keepSession is false", func(t *testing.T) {
		configs := []ProjectConfig{
			{Name: "frontend", WorkingDir: "/tmp/frontend"},
			{Name: "backend", WorkingDir: "/tmp/backend"},
		}

		factory := func(cfg *ProjectConfig) (agent.Agent, error) {
			return newMockAgent(), nil
		}

		router, _ := NewProjectRouter(configs, factory)
		router.SetActiveProject("frontend")

		// Create sessions in frontend project
		frontend, _ := router.GetProject("frontend")
		frontend.Sessions().GetOrCreate("session1")
		frontend.Sessions().GetOrCreate("session2")

		if len(frontend.Sessions().List()) != 2 {
			t.Fatalf("expected 2 sessions, got %d", len(frontend.Sessions().List()))
		}

		// Switch without keeping sessions
		router.SwitchProject(context.Background(), "backend", false)

		// Frontend sessions should be cleared
		if len(frontend.Sessions().List()) != 0 {
			t.Errorf("expected 0 sessions after switch, got %d", len(frontend.Sessions().List()))
		}
	})

	t.Run("keeps sessions when keepSession is true", func(t *testing.T) {
		configs := []ProjectConfig{
			{Name: "frontend", WorkingDir: "/tmp/frontend"},
			{Name: "backend", WorkingDir: "/tmp/backend"},
		}

		factory := func(cfg *ProjectConfig) (agent.Agent, error) {
			return newMockAgent(), nil
		}

		router, _ := NewProjectRouter(configs, factory)
		router.SetActiveProject("frontend")

		// Create sessions in frontend project
		frontend, _ := router.GetProject("frontend")
		frontend.Sessions().GetOrCreate("session1")
		frontend.Sessions().GetOrCreate("session2")

		// Switch keeping sessions
		router.SwitchProject(context.Background(), "backend", true)

		// Frontend sessions should still exist
		if len(frontend.Sessions().List()) != 2 {
			t.Errorf("expected 2 sessions after switch, got %d", len(frontend.Sessions().List()))
		}
	})
}

// TestProjectRouterListProjects tests ListProjects
func TestProjectRouterListProjects(t *testing.T) {
	configs := []ProjectConfig{
		{Name: "frontend", WorkingDir: "/tmp/frontend"},
		{Name: "backend", WorkingDir: "/tmp/backend"},
		{Name: "devops", WorkingDir: "/tmp/devops"},
	}

	factory := func(cfg *ProjectConfig) (agent.Agent, error) {
		return newMockAgent(), nil
	}

	router, _ := NewProjectRouter(configs, factory)
	router.SetActiveProject("backend")

	t.Run("lists all projects", func(t *testing.T) {
		list := router.ListProjects()

		if len(list) != 3 {
			t.Errorf("expected 3 projects, got %d", len(list))
		}
	})

	t.Run("marks active project", func(t *testing.T) {
		list := router.ListProjects()

		for _, info := range list {
			if info.Name == "backend" && !info.IsActive {
				t.Error("backend should be marked as active")
			}
			if info.Name != "backend" && info.IsActive {
				t.Errorf("%s should not be marked as active", info.Name)
			}
		}
	})
}

// TestProjectRouterStopAllAgents tests StopAllAgents
func TestProjectRouterStopAllAgents(t *testing.T) {
	t.Run("stops all agents", func(t *testing.T) {
		configs := []ProjectConfig{
			{Name: "frontend", WorkingDir: "/tmp/frontend"},
			{Name: "backend", WorkingDir: "/tmp/backend"},
		}

		factory := func(cfg *ProjectConfig) (agent.Agent, error) {
			return newMockAgent(), nil
		}

		router, _ := NewProjectRouter(configs, factory)
		router.SetActiveProject("frontend")

		// Start agents for both projects
		router.StartActiveProject(context.Background())
		router.SwitchProject(context.Background(), "backend", true)

		// Stop all agents
		err := router.StopAllAgents()

		if err != nil {
			t.Errorf("StopAllAgents failed: %v", err)
		}

		// Verify agents are stopped
		frontend, _ := router.GetProject("frontend")
		backend, _ := router.GetProject("backend")

		if frontend.Agent() != nil {
			t.Error("frontend agent should be nil after stop")
		}
		if backend.Agent() != nil {
			t.Error("backend agent should be nil after stop")
		}
	})
}

// TestProjectRouterConcurrentAccess tests concurrent access to router
func TestProjectRouterConcurrentAccess(t *testing.T) {
	t.Run("concurrent ListProjects", func(t *testing.T) {
		configs := []ProjectConfig{
			{Name: "frontend", WorkingDir: "/tmp/frontend"},
			{Name: "backend", WorkingDir: "/tmp/backend"},
		}

		factory := func(cfg *ProjectConfig) (agent.Agent, error) {
			return newMockAgent(), nil
		}

		router, _ := NewProjectRouter(configs, factory)
		router.SetActiveProject("frontend")

		const goroutines = 100
		done := make(chan bool, goroutines)

		for i := 0; i < goroutines; i++ {
			go func() {
				router.ListProjects()
				router.ActiveProject()
				router.GetProject("frontend")
				done <- true
			}()
		}

		for i := 0; i < goroutines; i++ {
			<-done
		}
	})
}
