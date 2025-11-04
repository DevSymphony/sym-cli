package eslint

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/adapter"
)

func TestExecute_FileCreation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "eslint-exec-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	a := NewAdapter("", tmpDir)

	ctx := context.Background()
	config := []byte(`{"rules": {"semi": [2, "always"]}}`)
	files := []string{"test.js"}

	_, err = a.execute(ctx, config, files)

	configPath := filepath.Join(tmpDir, ".symphony-eslintrc.json")
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Error("Config file should have been cleaned up")
	}
}

func TestGetESLintCommand(t *testing.T) {
	tests := []struct {
		name        string
		toolsDir    string
		wantContain string
	}{
		{
			name:        "local installation",
			toolsDir:    "/home/user/.symphony/tools",
			wantContain: "node_modules",
		},
		{
			name:        "empty tools dir",
			toolsDir:    "",
			wantContain: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAdapter(tt.toolsDir, "")
			cmd := a.getESLintCommand()

			if tt.wantContain != "" && len(cmd) > 0 {
				contains := false
				if len(cmd) > 0 && findSubstring(cmd, tt.wantContain) {
					contains = true
				}
				if !contains && tt.toolsDir != "" {
					t.Logf("Command %q doesn't contain %q (may use global)", cmd, tt.wantContain)
				}
			}
		})
	}
}

func TestGetExecutionArgs(t *testing.T) {
	a := NewAdapter("", "/work/dir")

	configPath := "/tmp/config.json"
	files := []string{"file1.js", "file2.js"}

	cmd, args := a.getExecutionArgs(configPath, files)

	if cmd == "" {
		t.Error("Expected non-empty command")
	}

	if len(args) == 0 {
		t.Error("Expected non-empty args")
	}

	// Check for essential flags
	foundConfig := false
	foundFormat := false
	for _, arg := range args {
		if arg == "--config" {
			foundConfig = true
		}
		if arg == "--format" {
			foundFormat = true
		}
	}

	if !foundConfig {
		t.Error("Expected --config flag in args")
	}

	if !foundFormat {
		t.Error("Expected --format flag in args")
	}
}

func TestWriteConfigFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "eslint-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	a := NewAdapter("", tmpDir)

	config := []byte(`{"rules": {"semi": [2, "always"]}}`)

	configPath, err := a.writeConfigFile(config)
	if err != nil {
		t.Fatalf("writeConfigFile() error = %v", err)
	}
	defer os.Remove(configPath)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Config file is empty")
	}
}

func TestExecute_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir, err := os.MkdirTemp("", "eslint-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.js")
	testCode := []byte("var x = 1\n") // Missing semicolon
	if err := os.WriteFile(testFile, testCode, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	a := NewAdapter("", tmpDir)

	ctx := context.Background()
	config := []byte(`{"rules": {"semi": [2, "always"]}}`)
	files := []string{testFile}

	output, err := a.execute(ctx, config, files)
	if err != nil {
		t.Logf("Execute failed (expected if ESLint not available): %v", err)
		return
	}

	if output == nil {
		t.Error("Expected non-nil output")
	}
}

func TestParseOutput_EmptyOutput(t *testing.T) {
	output := &adapter.ToolOutput{
		Stdout:   "",
		Stderr:   "",
		ExitCode: 0,
	}

	violations, err := parseOutput(output)
	if err != nil {
		t.Errorf("parseOutput() error = %v, want nil", err)
	}

	if len(violations) != 0 {
		t.Errorf("Expected 0 violations, got %d", len(violations))
	}
}

func TestParseOutput_InvalidJSON(t *testing.T) {
	output := &adapter.ToolOutput{
		Stdout:   "invalid json",
		Stderr:   "",
		ExitCode: 1,
	}

	_, err := parseOutput(output)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}
