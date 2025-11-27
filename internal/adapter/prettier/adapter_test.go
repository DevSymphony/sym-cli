package prettier

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/adapter"
)

func TestNewAdapter(t *testing.T) {
	a := NewAdapter("")
	if a == nil {
		t.Fatal("NewAdapter() returned nil")
	}

	if a.ToolsDir == "" {
		t.Error("ToolsDir should not be empty")
	}
}

func TestNewAdapter_CustomToolsDir(t *testing.T) {
	toolsDir := "/custom/tools"

	a := NewAdapter(toolsDir)

	if a.ToolsDir != toolsDir {
		t.Errorf("ToolsDir = %q, want %q", a.ToolsDir, toolsDir)
	}
}

func TestName(t *testing.T) {
	a := NewAdapter("")
	if a.Name() != "prettier" {
		t.Errorf("Name() = %q, want %q", a.Name(), "prettier")
	}
}

func TestGetPrettierPath(t *testing.T) {
	a := NewAdapter("/test/tools")
	expected := filepath.Join("/test/tools", "node_modules", ".bin", "prettier")

	got := a.getPrettierPath()
	if got != expected {
		t.Errorf("getPrettierPath() = %q, want %q", got, expected)
	}
}

func TestInitPackageJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "prettier-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	a := NewAdapter(tmpDir)

	if err := a.initPackageJSON(); err != nil {
		t.Fatalf("initPackageJSON() error = %v", err)
	}

	packagePath := filepath.Join(tmpDir, "package.json")
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		t.Error("package.json was not created")
	}
}

func TestCheckAvailability_NotFound(t *testing.T) {
	a := NewAdapter("/nonexistent/path")

	ctx := context.Background()
	err := a.CheckAvailability(ctx)

	if err == nil {
		t.Log("Prettier found globally, test skipped")
	}
}

func TestInstall(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "prettier-install-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	a := NewAdapter(tmpDir)

	ctx := context.Background()
	config := adapter.InstallConfig{
		ToolsDir: tmpDir,
	}

	err = a.Install(ctx, config)
	if err != nil {
		t.Logf("Install failed (expected if npm unavailable): %v", err)
	}
}

func TestExecute(t *testing.T) {
	a := NewAdapter(t.TempDir())

	ctx := context.Background()
	config := []byte(`{"semi": true}`)
	files := []string{"test.js"}

	_, err := a.Execute(ctx, config, files)
	if err == nil {
		t.Log("Execute succeeded (Prettier may be available)")
	}
}

func TestExecuteWithMode(t *testing.T) {
	a := NewAdapter(t.TempDir())

	ctx := context.Background()
	config := []byte(`{"semi": true}`)
	files := []string{"test.js"}

	for _, mode := range []string{"check", "write"} {
		t.Run(mode, func(t *testing.T) {
			_, err := a.ExecuteWithMode(ctx, config, files, mode)
			if err == nil {
				t.Logf("ExecuteWithMode(%s) succeeded", mode)
			}
		})
	}
}

func TestParseOutput(t *testing.T) {
	a := NewAdapter("")

	tests := []struct {
		name    string
		output  *adapter.ToolOutput
		wantLen int
		wantErr bool
	}{
		{
			name: "no violations",
			output: &adapter.ToolOutput{
				Stdout:   "",
				Stderr:   "",
				ExitCode: 0,
			},
			wantLen: 0,
			wantErr: false,
		},
		{
			name: "with violations",
			output: &adapter.ToolOutput{
				Stdout:   "file1.js\nfile2.js\n",
				Stderr:   "",
				ExitCode: 1,
			},
			wantLen: 2,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations, err := a.ParseOutput(tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(violations) != tt.wantLen {
				t.Errorf("ParseOutput() returned %d violations, want %d", len(violations), tt.wantLen)
			}
		})
	}
}
