package ast

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

	if caps.Name != "ast" {
		t.Errorf("Name = %s, want ast", caps.Name)
	}

	if !contains(caps.SupportedLanguages, "javascript") {
		t.Error("Expected javascript in supported languages")
	}

	if !contains(caps.SupportedCategories, "error_handling") {
		t.Error("Expected error_handling in supported categories")
	}

	if caps.SupportsAutofix {
		t.Error("AST engine should not support autofix")
	}
}

func TestMatchesSelector(t *testing.T) {
	engine := &Engine{}

	tests := []struct {
		name     string
		file     string
		selector *core.Selector
		want     bool
	}{
		{
			name:     "nil selector",
			file:     "src/main.js",
			selector: nil,
			want:     true,
		},
		{
			name: "matches javascript",
			file: "src/main.js",
			selector: &core.Selector{
				Languages: []string{"javascript"},
			},
			want: true,
		},
		{
			name: "doesn't match typescript",
			file: "src/main.js",
			selector: &core.Selector{
				Languages: []string{"typescript"},
			},
			want: false,
		},
		{
			name: "matches include pattern",
			file: "src/main.js",
			selector: &core.Selector{
				Include: []string{"src/*"},
			},
			want: true,
		},
		{
			name: "excluded by pattern",
			file: "test/main.js",
			selector: &core.Selector{
				Exclude: []string{"test/*"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.matchesSelector(tt.file, tt.selector)
			if got != tt.want {
				t.Errorf("matchesSelector() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchesLanguage(t *testing.T) {
	tests := []struct {
		ext  string
		lang string
		want bool
	}{
		{".js", "javascript", true},
		{".jsx", "jsx", true},
		{".ts", "typescript", true},
		{".tsx", "tsx", true},
		{".mjs", "javascript", true},
		{".py", "javascript", false},
	}

	for _, tt := range tests {
		got := matchesLanguage(tt.ext, tt.lang)
		if got != tt.want {
			t.Errorf("matchesLanguage(%q, %q) = %v, want %v", tt.ext, tt.lang, got, tt.want)
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
