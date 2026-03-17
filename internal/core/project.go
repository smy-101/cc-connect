package core

import (
	"context"
	"errors"
	"sync"

	"github.com/smy-101/cc-connect/internal/agent"
)

// Project status errors
var (
	// ErrProjectAlreadyActive is returned when switching to the currently active project
	ErrProjectAlreadyActive = errors.New("project is already active")
	// ErrProjectNotFound is returned when a project is not found
	ErrProjectNotFound = errors.New("project not found")
	// ErrAgentStartFailed is returned when agent fails to start
	ErrAgentStartFailed = errors.New("failed to start agent")
)

// ProjectStatus represents the current status of a project
type ProjectStatus string

const (
	// ProjectStatusIdle indicates the project is idle (not active)
	ProjectStatusIdle ProjectStatus = "idle"
	// ProjectStatusActive indicates the project is active
	ProjectStatusActive ProjectStatus = "active"
	// ProjectStatusSwitching indicates the project is being switched to
	ProjectStatusSwitching ProjectStatus = "switching"
)

// Project represents a single project instance with its own Agent and SessionManager
type Project struct {
	// Name is the project name
	Name string
	// Config is the project configuration
	Config *ProjectConfig
	// IsActive indicates if this is the currently active project
	IsActive bool

	// Internal state
	agent       agent.Agent
	sessions    *SessionManager
	status      ProjectStatus
	agentMu     sync.RWMutex
	statusMu    sync.RWMutex
	initialized bool
	initMu      sync.Mutex
}

// ProjectOption is a functional option for creating a Project
type ProjectOption func(*Project)

// WithSessionConfig sets the session configuration for the project
func WithSessionConfig(config SessionConfig) ProjectOption {
	return func(p *Project) {
		p.sessions = NewSessionManager(config)
	}
}

// NewProject creates a new Project instance
func NewProject(config *ProjectConfig, opts ...ProjectOption) *Project {
	p := &Project{
		Name:    config.Name,
		Config:  config,
		status:  ProjectStatusIdle,
		sessions: NewSessionManager(DefaultSessionConfig()),
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Status returns the current project status
func (p *Project) Status() ProjectStatus {
	p.statusMu.RLock()
	defer p.statusMu.RUnlock()
	return p.status
}

// SetStatus sets the project status
func (p *Project) SetStatus(status ProjectStatus) {
	p.statusMu.Lock()
	defer p.statusMu.Unlock()
	p.status = status
}

// Sessions returns the session manager for this project
func (p *Project) Sessions() *SessionManager {
	p.agentMu.RLock()
	defer p.agentMu.RUnlock()
	return p.sessions
}

// ClearSessions clears all sessions for this project
func (p *Project) ClearSessions() {
	p.sessions = NewSessionManager(DefaultSessionConfig())
}

// Agent returns the current agent (may be nil if not yet created)
func (p *Project) Agent() agent.Agent {
	p.agentMu.RLock()
	defer p.agentMu.RUnlock()
	return p.agent
}

// GetOrCreateAgent lazily creates and returns the agent
// It uses double-checked locking to ensure only one agent is created
func (p *Project) GetOrCreateAgent(ctx context.Context, factory AgentFactory) (agent.Agent, error) {
	// First check without lock (fast path)
	p.agentMu.RLock()
	if p.agent != nil {
		p.agentMu.RUnlock()
		return p.agent, nil
	}
	p.agentMu.RUnlock()

	// Use init mutex to ensure only one goroutine creates the agent
	p.initMu.Lock()
	defer p.initMu.Unlock()

	// Double-check after acquiring lock
	p.agentMu.RLock()
	if p.agent != nil {
		p.agentMu.RUnlock()
		return p.agent, nil
	}
	p.agentMu.RUnlock()

	// Create new agent
	ag, err := factory(p.Config)
	if err != nil {
		return nil, err
	}

	// Start the agent
	if err := ag.Start(ctx); err != nil {
		return nil, err
	}

	// Store the agent
	p.agentMu.Lock()
	p.agent = ag
	p.agentMu.Unlock()

	return ag, nil
}

// StopAgent stops the agent if it exists
func (p *Project) StopAgent() error {
	p.agentMu.Lock()
	defer p.agentMu.Unlock()

	if p.agent == nil {
		return nil
	}

	err := p.agent.Stop()
	p.agent = nil
	return err
}

// AgentFactory is a function that creates an Agent for a given project config
type AgentFactory func(config *ProjectConfig) (agent.Agent, error)

// ProjectInfo contains information about a project for listing
type ProjectInfo struct {
	Name       string
	WorkingDir string
	IsActive   bool
	Status     ProjectStatus
	Config     *ProjectConfig
}

// ProjectRouter manages multiple projects and handles switching between them
type ProjectRouter struct {
	projects map[string]*Project
	active   *Project
	mu       sync.RWMutex

	// Factory for creating agents
	agentFactory AgentFactory
}

// NewProjectRouter creates a new ProjectRouter
func NewProjectRouter(configs []ProjectConfig, factory AgentFactory) (*ProjectRouter, error) {
	if len(configs) == 0 {
		return nil, errors.New("at least one project config is required")
	}

	if factory == nil {
		return nil, errors.New("agent factory is required")
	}

	projects := make(map[string]*Project)
	for i := range configs {
		cfg := &configs[i]
		projects[cfg.Name] = NewProject(cfg)
	}

	router := &ProjectRouter{
		projects:     projects,
		agentFactory: factory,
	}

	return router, nil
}

// SwitchProject switches to the specified project
// If keepSession is false, the old project's sessions are cleared
func (r *ProjectRouter) SwitchProject(ctx context.Context, name string, keepSession bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Find target project
	targetProject, ok := r.projects[name]
	if !ok {
		return ErrProjectNotFound
	}

	// Check if already active
	if r.active != nil && r.active.Name == name {
		return ErrProjectAlreadyActive
	}

	// Store old active for rollback
	oldActive := r.active

	// Set switching status on target
	targetProject.SetStatus(ProjectStatusSwitching)

	// Stop old agent if exists
	if oldActive != nil {
		if err := oldActive.StopAgent(); err != nil {
			// Log but continue - we still want to switch
		}

		// Clear sessions if not keeping
		if !keepSession {
			oldActive.ClearSessions()
		}

		oldActive.SetStatus(ProjectStatusIdle)
	}

	// Create/get agent for new project
	_, err := targetProject.GetOrCreateAgent(ctx, r.agentFactory)
	if err != nil {
		// Try to rollback
		if oldActive != nil {
			oldActive.GetOrCreateAgent(ctx, r.agentFactory)
			oldActive.SetStatus(ProjectStatusActive)
			r.active = oldActive
		}
		targetProject.SetStatus(ProjectStatusIdle)
		return err
	}

	// Update active project
	targetProject.SetStatus(ProjectStatusActive)
	r.active = targetProject

	return nil
}

// ActiveProject returns the currently active project
func (r *ProjectRouter) ActiveProject() *Project {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.active
}

// GetProject returns a project by name
func (r *ProjectRouter) GetProject(name string) (*Project, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.projects[name]
	return p, ok
}

// ListProjects returns a list of all projects with their status
func (r *ProjectRouter) ListProjects() []ProjectInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ProjectInfo, 0, len(r.projects))
	for name, p := range r.projects {
		info := ProjectInfo{
			Name:       name,
			WorkingDir: p.Config.WorkingDir,
			Status:     p.Status(),
			IsActive:   r.active != nil && r.active.Name == name,
		}
		result = append(result, info)
	}

	return result
}

// SetActiveProject sets the initial active project without stopping any agents
// This is used during initialization
func (r *ProjectRouter) SetActiveProject(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.projects[name]
	if !ok {
		return ErrProjectNotFound
	}

	r.active = p
	p.SetStatus(ProjectStatusActive)
	return nil
}

// StartActiveProject starts the agent for the active project
func (r *ProjectRouter) StartActiveProject(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.active == nil {
		return errors.New("no active project")
	}

	_, err := r.active.GetOrCreateAgent(ctx, r.agentFactory)
	return err
}

// StopAllAgents stops all agents in all projects
func (r *ProjectRouter) StopAllAgents() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var lastErr error
	for _, p := range r.projects {
		if err := p.StopAgent(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// ProjectNames returns a list of all project names
func (r *ProjectRouter) ProjectNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.projects))
	for name := range r.projects {
		names = append(names, name)
	}
	return names
}
