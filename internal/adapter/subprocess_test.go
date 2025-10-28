package adapter

import (
	"context"
	"testing"
	"time"
)

func TestNewSubprocessExecutor(t *testing.T) {
	executor := NewSubprocessExecutor()
	if executor == nil {
		t.Fatal("NewSubprocessExecutor() returned nil")
	}

	if executor.Timeout != 2*time.Minute {
		t.Errorf("Default timeout = %v, want 2m", executor.Timeout)
	}

	if executor.Env == nil {
		t.Error("Env map should be initialized")
	}
}

func TestExecute_Success(t *testing.T) {
	executor := NewSubprocessExecutor()
	ctx := context.Background()

	output, err := executor.Execute(ctx, "echo", "hello")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if output.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", output.ExitCode)
	}

	if output.Stdout == "" {
		t.Error("Expected stdout output")
	}
}

func TestExecute_WithWorkDir(t *testing.T) {
	executor := NewSubprocessExecutor()
	executor.WorkDir = "/tmp"
	ctx := context.Background()

	output, err := executor.Execute(ctx, "pwd")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if output.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", output.ExitCode)
	}
}

func TestExecute_WithEnv(t *testing.T) {
	executor := NewSubprocessExecutor()
	executor.Env = map[string]string{
		"TEST_VAR": "test_value",
	}
	ctx := context.Background()

	output, err := executor.Execute(ctx, "sh", "-c", "echo $TEST_VAR")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if output.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", output.ExitCode)
	}
}

func TestExecute_NonZeroExit(t *testing.T) {
	executor := NewSubprocessExecutor()
	ctx := context.Background()

	output, err := executor.Execute(ctx, "sh", "-c", "exit 1")
	if err != nil {
		t.Fatalf("Execute should not return error for non-zero exit: %v", err)
	}

	if output.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", output.ExitCode)
	}
}

func TestExecute_Timeout(t *testing.T) {
	executor := NewSubprocessExecutor()
	executor.Timeout = 10 * time.Millisecond
	ctx := context.Background()

	output, err := executor.Execute(ctx, "sleep", "1")
	// Timeout can result in either error or killed exit code
	if err == nil && (output == nil || output.ExitCode == 0) {
		t.Error("Expected timeout error or non-zero exit code")
	}
}

func TestEnvSlice(t *testing.T) {
	executor := &SubprocessExecutor{
		Env: map[string]string{
			"KEY1": "value1",
			"KEY2": "value2",
		},
	}

	slice := executor.envSlice()
	if len(slice) != 2 {
		t.Errorf("envSlice() length = %d, want 2", len(slice))
	}

	// Check that both key-value pairs are present
	found := make(map[string]bool)
	for _, env := range slice {
		if env == "KEY1=value1" {
			found["KEY1"] = true
		}
		if env == "KEY2=value2" {
			found["KEY2"] = true
		}
	}

	if !found["KEY1"] || !found["KEY2"] {
		t.Errorf("envSlice() = %v, missing expected env vars", slice)
	}
}
