package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/smy-101/cc-connect/internal/agent"
	"github.com/smy-101/cc-connect/internal/agent/claudecode"
	"github.com/smy-101/cc-connect/internal/core"
	"github.com/smy-101/cc-connect/internal/core/command"
	"github.com/smy-101/cc-connect/internal/platform/feishu"
)

// Application errors
var (
	// ErrNilConfig is returned when nil config is passed to New.
	ErrNilConfig = errors.New("config cannot be nil")
	// ErrNoProjects is returned when config has no projects.
	ErrNoProjects = errors.New("at least one project is required")
	// ErrFeishuNotEnabled is returned when Feishu is not enabled.
	ErrFeishuNotEnabled = errors.New("Feishu must be enabled for at least one project")
	// ErrClaudeCodeNotEnabled is returned when Claude Code is not enabled.
	ErrClaudeCodeNotEnabled = errors.New("Claude Code must be enabled for at least one project")
	// ErrMissingFeishuCredentials is returned when Feishu credentials are missing.
	ErrMissingFeishuCredentials = errors.New("Feishu AppID and AppSecret are required")
	// ErrAppAlreadyRunning is returned when Start is called on a running app.
	ErrAppAlreadyRunning = errors.New("app is already running")
	// ErrAppNotRunning is returned when operations require a running app.
	ErrAppNotRunning = errors.New("app is not running")
)

// AppStatus represents the current status of the application.
type AppStatus string

const (
	// AppStatusIdle indicates the app is idle (not started).
	AppStatusIdle AppStatus = "idle"
	// AppStatusRunning indicates the app is running.
	AppStatusRunning AppStatus = "running"
	// AppStatusStopping indicates the app is stopping.
	AppStatusStopping AppStatus = "stopping"
	// AppStatusStopped indicates the app has been stopped.
	AppStatusStopped AppStatus = "stopped"
)

// App is the main application struct that manages all components.
type App struct {
	// Configuration
	config   *core.AppConfig
	project  *core.ProjectConfig
	agentCfg *AgentConfig

	// Core components
	router              *core.Router
	agent               agent.Agent
	feishu              *feishu.Adapter
	executor            *command.Executor
	feishuClientFactory func(appID, appSecret string) feishu.FeishuClient

	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex
	status AppStatus

	// Configuration options
	agentTimeout time.Duration
}

// AgentConfig holds agent configuration options.
type AgentConfig struct {
	WorkingDir     string
	PermissionMode agent.PermissionMode
	AgentTimeout   time.Duration
	SessionConfig  core.SessionConfig
}

// New creates a new App instance with the given configuration.
// It validates the configuration and initializes all components.
func New(config *core.AppConfig) (*App, error) {
	// Validate config
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	// Get the default project or first project
	project, err := getActiveProject(config)
	if err != nil {
		return nil, err
	}

	// Create agent config
	agentCfg, err := createAgentConfig(project)
	if err != nil {
		return nil, fmt.Errorf("invalid agent config: %w", err)
	}

	// Create core components
	router := core.NewRouter()

	// Create agent
	agentConfig := &claudecode.Config{
		WorkingDir:     agentCfg.WorkingDir,
		PermissionMode: agentCfg.PermissionMode,
	}
	ag, err := claudecode.NewAgent(agentConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	// Create executor
	executor := command.NewExecutor(ag, router.Sessions())

	return &App{
		config:       config,
		project:      project,
		agentCfg:     agentCfg,
		router:       router,
		agent:        ag,
		executor:     executor,
		status:       AppStatusIdle,
		agentTimeout: agentCfg.AgentTimeout,
		feishuClientFactory: func(appID, appSecret string) feishu.FeishuClient {
			return feishu.NewSDKClient(appID, appSecret)
		},
	}, nil
}

// validateConfig validates the application configuration.
func validateConfig(config *core.AppConfig) error {
	if config == nil {
		return ErrNilConfig
	}

	if len(config.Projects) == 0 {
		return ErrNoProjects
	}

	// Use the config validator for more detailed validation
	validator := core.NewConfigValidator()
	if err := validator.Validate(config); err != nil {
		return err
	}

	// Check that at least one project has Feishu and Claude Code enabled
	var hasFeishu, hasClaudeCode bool
	for _, p := range config.Projects {
		if p.Feishu.Enabled {
			if p.Feishu.AppID == "" || p.Feishu.AppSecret == "" {
				return fmt.Errorf("%w: project %s", ErrMissingFeishuCredentials, p.Name)
			}
			hasFeishu = true
		}
		if p.ClaudeCode.Enabled {
			hasClaudeCode = true
		}
	}

	if !hasFeishu {
		return ErrFeishuNotEnabled
	}
	if !hasClaudeCode {
		return ErrClaudeCodeNotEnabled
	}

	return nil
}

// getActiveProject returns the default project or first enabled project.
func getActiveProject(config *core.AppConfig) (*core.ProjectConfig, error) {
	// Try default project first
	if config.DefaultProject != "" {
		project, ok := config.GetDefaultProject()
		if !ok {
			return nil, fmt.Errorf("default project %s not found", config.DefaultProject)
		}
		return project, nil
	}

	// Find first project with Feishu and Claude Code enabled
	for i := range config.Projects {
		p := &config.Projects[i]
		if p.Feishu.Enabled && p.ClaudeCode.Enabled {
			// Return a copy
			copy := *p
			return &copy, nil
		}
	}

	// Fall back to first project
	if len(config.Projects) > 0 {
		copy := config.Projects[0]
		return &copy, nil
	}

	return nil, ErrNoProjects
}

// createAgentConfig creates agent configuration from project config.
func createAgentConfig(project *core.ProjectConfig) (*AgentConfig, error) {
	// Parse permission mode
	mode := parsePermissionMode(project.ClaudeCode.DefaultPermissionMode)

	return &AgentConfig{
		WorkingDir:     project.WorkingDir,
		PermissionMode: mode,
		AgentTimeout:   5 * time.Minute, // Default timeout
		SessionConfig:  project.GetSessionConfig(),
	}, nil
}

// parsePermissionMode parses a permission mode string.
func parsePermissionMode(mode string) agent.PermissionMode {
	switch mode {
	case "edit":
		return agent.PermissionModeDefault // edit mode not directly supported
	case "acceptEdits":
		return agent.PermissionModeAcceptEdits
	case "plan":
		return agent.PermissionModePlan
	case "yolo", "bypassPermissions":
		return agent.PermissionModeBypassPermissions
	default:
		return agent.PermissionModeDefault
	}
}

// Status returns the current application status.
func (a *App) Status() AppStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

// Router returns the message router.
func (a *App) Router() *core.Router {
	return a.router
}

// Sessions returns the session manager (convenience method).
func (a *App) Sessions() *core.SessionManager {
	return a.router.Sessions()
}

// Agent returns the AI agent.
func (a *App) Agent() agent.Agent {
	return a.agent
}

// SetFeishuClientFactory overrides the Feishu client factory.
// It is primarily used by tests to avoid real network dependencies.
func (a *App) SetFeishuClientFactory(factory func(appID, appSecret string) feishu.FeishuClient) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if factory != nil {
		a.feishuClientFactory = factory
	}
}

// Start starts the application by starting the Agent, connecting to Feishu,
// and registering message handlers.
func (a *App) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.status == AppStatusRunning {
		return ErrAppAlreadyRunning
	}

	// Create app context
	a.ctx, a.cancel = context.WithCancel(context.Background())

	// Start the agent
	if err := a.agent.Start(ctx); err != nil {
		return fmt.Errorf("failed to start agent: %w", err)
	}

	// Create Feishu client and adapter
	feishuClient := a.feishuClientFactory(a.project.Feishu.AppID, a.project.Feishu.AppSecret)
	a.feishu = feishu.NewAdapter(feishuClient, a.router)

	// Register handlers before connecting
	if err := a.registerHandlers(); err != nil {
		_ = a.agent.Stop()
		return fmt.Errorf("failed to register handlers: %w", err)
	}

	// Connect to Feishu
	if err := a.feishu.Start(a.ctx); err != nil {
		_ = a.agent.Stop()
		return fmt.Errorf("failed to connect to Feishu: %w", err)
	}

	a.status = AppStatusRunning
	return nil
}

// Stop stops the application gracefully.
// It stops accepting new messages, waits for in-flight messages, and cleans up resources.
func (a *App) Stop() error {
	a.mu.Lock()
	if a.status != AppStatusRunning {
		a.mu.Unlock()
		return nil
	}
	a.status = AppStatusStopping
	a.mu.Unlock()

	// Cancel context to signal shutdown
	if a.cancel != nil {
		a.cancel()
	}

	// Stop Feishu adapter
	if a.feishu != nil {
		if err := a.feishu.Stop(); err != nil {
			// Log but continue cleanup
		}
	}

	// Stop agent
	if a.agent != nil {
		if err := a.agent.Stop(); err != nil {
			// Log but continue cleanup
		}
	}

	// Wait for goroutines with timeout
	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines finished
	case <-time.After(30 * time.Second):
		// Timeout waiting for goroutines
	}

	a.mu.Lock()
	a.status = AppStatusStopped
	a.mu.Unlock()

	return nil
}

// WaitForShutdown blocks until the application has fully stopped.
func (a *App) WaitForShutdown() {
	a.wg.Wait()
}

// registerHandlers registers all message handlers with the router.
func (a *App) registerHandlers() error {
	// Register text message handler
	if err := a.router.Register(core.MessageTypeText, a.wrapHandler(a.handleText)); err != nil {
		return fmt.Errorf("failed to register text handler: %w", err)
	}

	// Register command message handler
	if err := a.router.Register(core.MessageTypeCommand, a.wrapHandler(a.handleCommand)); err != nil {
		return fmt.Errorf("failed to register command handler: %w", err)
	}

	return nil
}
