package claudecode

import (
	"context"
	"fmt"
	"io"
	"log/slog"
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
	config *ProcessConfig
	cmd    *exec.Cmd
	cancel context.CancelFunc
	mu     sync.Mutex
	done   chan error
	stdout io.Reader
	stdin  io.Writer
	stderr io.Reader
}

func (pm *ProcessManager) buildArgs() []string {
	args := []string{
		pm.config.ClaudePath,
		"-p",
		"--verbose",
		"--output-format", "stream-json",
		"--permission-mode", PermissionModeToCLIArg(pm.config.PermissionMode),
	}

	if pm.config.Resume {
		args = append(args, "--resume", pm.config.SessionID)
	} else if pm.config.SessionID != "" {
		args = append(args, "--session-id", pm.config.SessionID)
	}

	if pm.config.Message != "" {
		args = append(args, pm.config.Message)
	}

	return args
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
	args := pm.buildArgs()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = pm.config.WorkingDir

	return cmd
}

func (pm *ProcessManager) buildCommandContext(ctx context.Context) *exec.Cmd {
	args := pm.buildArgs()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
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

	processCtx, cancel := context.WithCancel(ctx)
	cmd := pm.buildCommandContext(processCtx)

	slog.Debug("Building Claude command", "args", cmd.Args, "working_dir", cmd.Dir)

	// Set up environment variables
	cmd.Env = os.Environ()
	for _, env := range pm.config.Env {
		cmd.Env = append(cmd.Env, env)
	}

	// Get pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("failed to get stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("failed to get stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("failed to get stderr: %w", err)
	}

	pm.stdin = stdin
	pm.stdout = stdout
	pm.stderr = stderr
	pm.cmd = cmd

	pm.cancel = cancel

	// Start the process
	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("failed to start process: %w", err)
	}

	slog.Debug("Claude process started", "pid", cmd.Process.Pid, "args", cmd.Args)

	// Monitor process exit in background
	go func() {
		err := cmd.Wait()
		slog.Debug("Claude process exited", "pid", cmd.Process.Pid, "error", err)
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
		if pm.cancel != nil {
			pm.cancel()
		}
		pm.cmd = nil
		pm.stdin = nil
		pm.stdout = nil
		pm.stderr = nil
		pm.cancel = nil
		return nil
	case <-time.After(2 * time.Second):
		// Force kill after timeout
		if pm.cancel != nil {
			pm.cancel()
		}
		if err := pm.cmd.Process.Signal(syscall.SIGKILL); err != nil {
			// Ignore errors from kill
		}
		<-pm.done
		pm.cmd = nil
		pm.stdin = nil
		pm.stdout = nil
		pm.stderr = nil
		pm.cancel = nil
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
	err := <-pm.done

	pm.mu.Lock()
	pm.cmd = nil
	pm.stdin = nil
	pm.stdout = nil
	pm.stderr = nil
	pm.cancel = nil
	pm.mu.Unlock()

	return err
}

// CloseStdin closes the stdin pipe, signaling EOF to the subprocess.
// This is necessary when using -p mode with a message argument,
// since Claude CLI may wait for stdin EOF before processing.
func (pm *ProcessManager) CloseStdin() error {
	if closer, ok := pm.stdin.(io.Closer); ok {
		return closer.Close()
	}
	return nil
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
