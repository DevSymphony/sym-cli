package prettier

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestExecute_FileCreation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "prettier-exec-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	a := NewAdapter("", tmpDir)

	ctx := context.Background()
	config := []byte(`{"semi": true}`)
	files := []string{"test.js"}

	_, _ = a.execute(ctx, config, files, "check")

	configPath := filepath.Join(tmpDir, ".symphony-prettierrc.json")
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Error("Config file should have been cleaned up")
	}
}

func TestGetPrettierCommand(t *testing.T) {
	tests := []struct {
		name     string
		toolsDir string
	}{
		{
			name:     "local installation",
			toolsDir: "/home/user/.symphony/tools",
		},
		{
			name:     "empty tools dir",
			toolsDir: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAdapter(tt.toolsDir, "")
			cmd := a.getPrettierCommand()

			if cmd == "" {
				t.Error("Expected non-empty command")
			}
		})
	}
}

func TestWriteConfigFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "prettier-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	a := NewAdapter("", tmpDir)

	config := []byte(`{"semi": true, "singleQuote": true}`)

	configPath, err := a.writeConfigFile(config)
	if err != nil {
		t.Fatalf("writeConfigFile() error = %v", err)
	}
	defer func() { _ = os.Remove(configPath) }()

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

	tmpDir, err := os.MkdirTemp("", "prettier-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a test file with bad formatting
	testFile := filepath.Join(tmpDir, "test.js")
	testCode := []byte("const x=1;const y=2;") // Bad formatting
	if err := os.WriteFile(testFile, testCode, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	a := NewAdapter("", tmpDir)

	ctx := context.Background()
	config := []byte(`{"semi": true, "singleQuote": true}`)
	files := []string{testFile}

	output, err := a.execute(ctx, config, files, "check")
	if err != nil {
		t.Logf("Execute failed (expected if Prettier not available): %v", err)
		return
	}

	if output == nil {
		t.Skip("Prettier not available in test environment")
		return
	}

	// If we got here, Prettier is available and returned output
	t.Logf("Prettier executed successfully, exit code: %d", output.ExitCode)
}
