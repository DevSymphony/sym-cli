package linter

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// SubprocessExecutor runs external tools as subprocesses.
type SubprocessExecutor struct {
	// Timeout is the max execution time.
	// Default: 2 minutes
	Timeout time.Duration

	// WorkDir is the working directory.
	WorkDir string

	// Env is additional environment variables.
	Env map[string]string
}

// NewSubprocessExecutor creates a new executor.
func NewSubprocessExecutor() *SubprocessExecutor {
	return &SubprocessExecutor{
		Timeout: 2 * time.Minute,
		Env:     make(map[string]string),
	}
}

// Execute runs a command and returns its output.
func (e *SubprocessExecutor) Execute(ctx context.Context, name string, args ...string) (*ToolOutput, error) {
	// Apply timeout
	if e.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.Timeout)
		defer cancel()
	}

	// Create command
	cmd := exec.CommandContext(ctx, name, args...)

	if e.WorkDir != "" {
		cmd.Dir = e.WorkDir
	}

	// Add environment variables
	if len(e.Env) > 0 {
		cmd.Env = append(os.Environ(), e.envSlice()...)
	}

	// Capture output
	start := time.Now()
	stdout, err := cmd.Output()
	duration := time.Since(start)

	output := &ToolOutput{
		Stdout:   string(stdout),
		Duration: duration.String(),
	}

	if err != nil {
		// Check if it's an ExitError (non-zero exit code)
		if exitErr, ok := err.(*exec.ExitError); ok {
			output.Stderr = string(exitErr.Stderr)
			output.ExitCode = exitErr.ExitCode()
			return output, nil // Return output even on non-zero exit
		}
		return nil, fmt.Errorf("failed to execute %s: %w", name, err)
	}

	output.ExitCode = 0
	return output, nil
}

func (e *SubprocessExecutor) envSlice() []string {
	result := make([]string, 0, len(e.Env))
	for k, v := range e.Env {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}
