package claudecode

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/smy-101/cc-connect/internal/agent"
)

// ProcessConfig contains configuration for the Claude Code process
type ProcessConfig struct {
	SessionID      string
	WorkingDir     string
	PermissionMode agent.PermissionMode
	Resume         bool
	ClaudePath     string   // Path to claude CLI (defaults to "claude")
	AllowedTools   []string // Tools to pre-approve
	Env            []string // Additional environment variables
	Message        string   // Message to send (used for single-shot mode)
}

// DefaultClaudePath returns the default path to claude CLI
func DefaultClaudePath() string {
	return "claude"
}

// ProcessManager manages a Claude Code subprocess
type ProcessManager struct {
	config  *ProcessConfig
	cmd     *exec.Cmd
	cancel  context.CancelFunc
	mu      sync.Mutex
	done    chan error
	stdout io.Reader
	stdin  io.Writer
	stderr io.Reader
}

// NewProcessManager creates a new ProcessManager
func NewProcessManager(config *ProcessConfig) *ProcessManager {
	if config.ClaudePath == "" {
		config.ClaudePath = DefaultClaudePath()
	}
	return &ProcessManager{
		config: config,
		done:   make(chan error, 1),
	}
}

// buildCommand constructs the exec.Cmd for the process
func (pm *ProcessManager) buildCommand() *exec.Cmd {
	args := []string{
		pm.config.ClaudePath, // First arg is the command name
		"-p",
		"--output-format", "stream-json",
		"--session-id", pm.config.SessionID,
		"--permission-mode", PermissionModeToCLIArg(pm.config.PermissionMode),
	}

	if pm.config.Resume {
		args = append(args, "--resume")
	}

	// Add message as the last argument if provided
	if pm.config.Message != "" {
		args = append(args, pm.config.Message)
	}

	// Build the command using exec.Command function
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = pm.config.WorkingDir

	return cmd
}

// Start starts the process
func (pm *ProcessManager) Start(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.cmd != nil && pm.cmd.Process != nil {
		return agent.ErrAgentAlreadyRunning
	}

	cmd := pm.buildCommand()

	// Set up environment variables
	cmd.Env = os.Environ()
	for _, env := range pm.config.Env {
		cmd.Env = append(cmd.Env, env)
	}

	// Get pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr: %w", err)
	}

	pm.stdin = stdin
	pm.stdout = stdout
	pm.stderr = stderr
	pm.cmd = cmd

	// Create cancel context for timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	pm.cancel = cancel

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	// Monitor process exit in background
	go func() {
		err := cmd.Wait()
		pm.done <- err
	}()

	return nil
}

// Stop gracefully stops the process
func (pm *ProcessManager) Stop() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.cmd == nil || pm.cmd.Process == nil {
		return nil
	}

	// Cancel the context
	if pm.cancel != nil {
		pm.cancel()
	}

	// Send SIGTERM
	if err := pm.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// Process might already be dead
		if err.Error() == "os: signal: process not initialized" || err.Error() == "os: ErrNoProcess" {
			pm.cmd = nil
			pm.stdin = nil
			pm.stdout = nil
			pm.stderr = nil
			return nil
		}
	}

	// Wait for process to exit with timeout
	select {
	case <-pm.done:
		pm.cmd = nil
		pm.stdin = nil
		pm.stdout = nil
		pm.stderr = nil
		return nil
	case <-time.After(2 * time.Second):
		// Force kill after timeout
		if err := pm.cmd.Process.Signal(syscall.SIGKILL); err != nil {
			// Ignore errors from kill
		}
		<-pm.done
		pm.cmd.Wait()
		pm.cmd = nil
		pm.stdin = nil
		pm.stdout = nil
		pm.stderr = nil
		return fmt.Errorf("process did not exit after SIGTERM, killed")
	}
}

// Restart restarts the process with optional mode change
func (pm *ProcessManager) Restart(ctx context.Context, newMode *agent.PermissionMode) error {
	pm.mu.Lock()

	if pm.cmd == nil || pm.cmd.Process == nil {
		pm.mu.Unlock()
		return agent.ErrAgentNotRunning
	}

	// Update config
	if newMode != nil && *newMode != "" {
		pm.config.PermissionMode = *newMode
	}
	pm.config.Resume = true // Always resume on restart
	pm.mu.Unlock()

	// Stop current process
	if err := pm.Stop(); err != nil {
		return fmt.Errorf("failed to stop process for restart: %w", err)
	}

	// Start new process
	return pm.Start(ctx)
}

// IsRunning returns true if the process is running
func (pm *ProcessManager) IsRunning() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.cmd == nil || pm.cmd.Process == nil {
		return false
	}

	// Try to check if process is still running by sending signal 0
	err := pm.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

// Wait blocks until the process exits
func (pm *ProcessManager) Wait() error {
	select {
	case <-pm.done:
		return nil
	case err := <-pm.done:
		return err
	}
}

// Stdin returns the stdin writer
func (pm *ProcessManager) Stdin() io.Writer {
	return pm.stdin
}

// Stdout returns the stdout reader
func (pm *ProcessManager) Stdout() io.Reader {
	return pm.stdout
}

// Stderr returns the stderr reader
func (pm *ProcessManager) Stderr() io.Reader {
	return pm.stderr
}
