package length

import (
	"testing"

	"github.com/DevSymphony/sym-cli/internal/engine/core"
)

func TestNewEngine(t *testing.T) {
	engine := NewEngine()
	if engine == nil {
		t.Fatal("NewEngine() returned nil")
	}
}

func TestGetCapabilities(t *testing.T) {
	engine := NewEngine()
	caps := engine.GetCapabilities()

	if caps.Name != "length" {
		t.Errorf("Name = %s, want length", caps.Name)
	}

	if !contains(caps.SupportedLanguages, "javascript") {
		t.Error("Expected javascript in supported languages")
	}

	if !contains(caps.SupportedCategories, "formatting") {
		t.Error("Expected formatting in supported categories")
	}

	if caps.SupportsAutofix {
		t.Error("Length engine should not support autofix")
	}
}

func TestFilterFiles(t *testing.T) {
	engine := &Engine{}

	files := []string{
		"src/main.js",
		"src/app.ts",
		"test/test.js",
		"README.md",
	}

	tests := []struct {
		name     string
		selector *core.Selector
		want     []string
	}{
		{
			name:     "nil selector - all files",
			selector: nil,
			want:     files,
		},
		{
			name: "with selector - filters JS/TS only",
			selector: &core.Selector{
				Languages: []string{"javascript"},
			},
			want: []string{"src/main.js", "src/app.ts", "test/test.js"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.filterFiles(files, tt.selector)
			if !equalSlices(got, tt.want) {
				t.Errorf("filterFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
