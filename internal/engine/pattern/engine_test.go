package pattern

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

	if caps.Name != "pattern" {
		t.Errorf("Name = %s, want pattern", caps.Name)
	}

	if !contains(caps.SupportedLanguages, "javascript") {
		t.Error("Expected javascript in supported languages")
	}

	if !contains(caps.SupportedCategories, "naming") {
		t.Error("Expected naming in supported categories")
	}

	if caps.SupportsAutofix {
		t.Error("Pattern engine should not support autofix")
	}
}

func TestMatchesLanguage(t *testing.T) {
	engine := &Engine{}

	tests := []struct {
		file  string
		langs []string
		want  bool
	}{
		{"main.js", []string{"javascript"}, true},
		{"app.jsx", []string{"jsx"}, true},
		{"server.ts", []string{"typescript"}, true},
		{"component.tsx", []string{"tsx"}, true},
		{"main.js", []string{"typescript"}, false},
		{"app.py", []string{"javascript"}, false},
		{"main.js", []string{"javascript", "typescript"}, true},
	}

	for _, tt := range tests {
		got := engine.matchesLanguage(tt.file, tt.langs)
		if got != tt.want {
			t.Errorf("matchesLanguage(%q, %v) = %v, want %v", tt.file, tt.langs, got, tt.want)
		}
	}
}

func TestFilterFiles(t *testing.T) {
	engine := &Engine{}

	files := []string{
		"src/main.js",
		"src/app.ts",
		"test/test.js",
		"README.md",
		"src/styles.css",
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
			name: "javascript only",
			selector: &core.Selector{
				Languages: []string{"javascript"},
			},
			want: []string{"src/main.js", "test/test.js"},
		},
		{
			name: "typescript only",
			selector: &core.Selector{
				Languages: []string{"typescript"},
			},
			want: []string{"src/app.ts"},
		},
		{
			name: "multiple languages",
			selector: &core.Selector{
				Languages: []string{"javascript", "typescript"},
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

func TestDetectLanguage(t *testing.T) {
	engine := &Engine{}

	tests := []struct {
		files []string
		want  string
	}{
		{[]string{"main.js"}, "javascript"},
		{[]string{"app.jsx"}, "jsx"},
		{[]string{"server.ts"}, "typescript"},
		{[]string{"component.tsx"}, "tsx"},
		{[]string{}, "javascript"},
	}

	for _, tt := range tests {
		got := engine.detectLanguage(tt.files)
		if got != tt.want {
			t.Errorf("detectLanguage(%v) = %q, want %q", tt.files, got, tt.want)
		}
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
