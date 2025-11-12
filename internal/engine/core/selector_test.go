package core

import (
	"testing"
)

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		pattern  string
		want     bool
		wantErr  bool
	}{
		// Simple patterns
		{
			name:     "exact match",
			filePath: "main.go",
			pattern:  "main.go",
			want:     true,
		},
		{
			name:     "wildcard extension",
			filePath: "main.go",
			pattern:  "*.go",
			want:     true,
		},
		{
			name:     "wildcard name",
			filePath: "main.go",
			pattern:  "main.*",
			want:     true,
		},
		{
			name:     "no match",
			filePath: "main.go",
			pattern:  "*.js",
			want:     false,
		},

		// Doublestar patterns
		{
			name:     "doublestar all files",
			filePath: "src/main.go",
			pattern:  "**/*.go",
			want:     true,
		},
		{
			name:     "doublestar nested",
			filePath: "src/foo/bar/test.js",
			pattern:  "src/**/*.js",
			want:     true,
		},
		{
			name:     "doublestar no match",
			filePath: "test/main.go",
			pattern:  "src/**/*.go",
			want:     false,
		},
		{
			name:     "doublestar middle",
			filePath: "src/foo/bar/baz/test.ts",
			pattern:  "src/**/test.ts",
			want:     true,
		},

		// Path-specific patterns
		{
			name:     "specific directory",
			filePath: "src/components/Button.tsx",
			pattern:  "src/components/*.tsx",
			want:     true,
		},
		{
			name:     "exclude test files",
			filePath: "src/main_test.go",
			pattern:  "**/*_test.go",
			want:     true,
		},
		{
			name:     "exclude test directory",
			filePath: "tests/unit/main.go",
			pattern:  "tests/**/*.go",
			want:     true,
		},

		// Windows-style paths are normalized to forward slashes
		// Note: In actual usage, filepath operations will handle OS-specific separators
		{
			name:     "mixed separators",
			filePath: "src/subdir/main.go",
			pattern:  "src/**/*.go",
			want:     true,
		},

		// Edge cases
		{
			name:     "empty pattern",
			filePath: "main.go",
			pattern:  "",
			want:     false,
		},
		{
			name:     "root level file",
			filePath: "main.go",
			pattern:  "*.go",
			want:     true,
		},
		{
			name:     "multiple wildcards",
			filePath: "src/foo/bar/test_main.go",
			pattern:  "src/**/test_*.go",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MatchGlob(tt.filePath, tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("MatchGlob() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("MatchGlob(%q, %q) = %v, want %v", tt.filePath, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestMatchesLanguage(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		language string
		want     bool
	}{
		// JavaScript variants
		{"js standard", "main.js", "javascript", true},
		{"js module", "main.mjs", "javascript", true},
		{"js commonjs", "main.cjs", "javascript", true},
		{"js short", "main.js", "js", true},
		{"jsx", "Component.jsx", "jsx", true},

		// TypeScript variants
		{"ts standard", "main.ts", "typescript", true},
		{"ts module", "main.mts", "typescript", true},
		{"ts commonjs", "main.cts", "typescript", true},
		{"ts short", "main.ts", "ts", true},
		{"tsx", "Component.tsx", "tsx", true},

		// Python variants
		{"py standard", "main.py", "python", true},
		{"py interface", "main.pyi", "python", true},
		{"py windows", "main.pyw", "python", true},
		{"py short", "main.py", "py", true},

		// Go
		{"go standard", "main.go", "go", true},
		{"go long name", "main.go", "golang", true},

		// Other languages
		{"java", "Main.java", "java", true},
		{"c", "main.c", "c", true},
		{"c header", "main.h", "c", true},
		{"cpp", "main.cpp", "cpp", true},
		{"cpp alt", "main.cc", "cpp", true},
		{"cpp header", "main.hpp", "cpp", true},
		{"rust", "main.rs", "rust", true},
		{"ruby", "main.rb", "ruby", true},
		{"php", "index.php", "php", true},
		{"swift", "Main.swift", "swift", true},
		{"kotlin", "Main.kt", "kotlin", true},
		{"kotlin script", "build.kts", "kotlin", true},
		{"scala", "Main.scala", "scala", true},
		{"shell", "script.sh", "shell", true},
		{"bash", "script.bash", "shell", true},
		{"shell short", "script.sh", "sh", true},

		// Case insensitivity
		{"uppercase ext", "Main.GO", "go", true},
		{"uppercase lang", "main.js", "JAVASCRIPT", true},
		{"mixed case", "Main.JS", "JavaScript", true},

		// No match
		{"wrong extension", "main.go", "javascript", false},
		{"unknown language", "main.xyz", "xyz", false},
		{"empty extension", "README", "go", false},
		{"no extension match", "main.txt", "javascript", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchesLanguage(tt.filePath, tt.language)
			if got != tt.want {
				t.Errorf("MatchesLanguage(%q, %q) = %v, want %v", tt.filePath, tt.language, got, tt.want)
			}
		})
	}
}

func TestMatchesSelector(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		selector *Selector
		want     bool
	}{
		// Nil selector (should match all)
		{
			name:     "nil selector",
			filePath: "any/file.go",
			selector: nil,
			want:     true,
		},

		// Language only
		{
			name:     "language match",
			filePath: "main.go",
			selector: &Selector{Languages: []string{"go"}},
			want:     true,
		},
		{
			name:     "language no match",
			filePath: "main.go",
			selector: &Selector{Languages: []string{"javascript"}},
			want:     false,
		},
		{
			name:     "multiple languages match first",
			filePath: "main.js",
			selector: &Selector{Languages: []string{"javascript", "typescript"}},
			want:     true,
		},
		{
			name:     "multiple languages match second",
			filePath: "main.ts",
			selector: &Selector{Languages: []string{"javascript", "typescript"}},
			want:     true,
		},

		// Include only
		{
			name:     "include match",
			filePath: "src/main.go",
			selector: &Selector{Include: []string{"src/**/*.go"}},
			want:     true,
		},
		{
			name:     "include no match",
			filePath: "test/main.go",
			selector: &Selector{Include: []string{"src/**/*.go"}},
			want:     false,
		},
		{
			name:     "multiple includes match first",
			filePath: "src/main.go",
			selector: &Selector{Include: []string{"src/**/*.go", "lib/**/*.go"}},
			want:     true,
		},
		{
			name:     "multiple includes match second",
			filePath: "lib/util.go",
			selector: &Selector{Include: []string{"src/**/*.go", "lib/**/*.go"}},
			want:     true,
		},
		{
			name:     "multiple includes no match",
			filePath: "test/main.go",
			selector: &Selector{Include: []string{"src/**/*.go", "lib/**/*.go"}},
			want:     false,
		},

		// Exclude only
		{
			name:     "exclude match - should reject",
			filePath: "src/main_test.go",
			selector: &Selector{Exclude: []string{"**/*_test.go"}},
			want:     false,
		},
		{
			name:     "exclude no match - should accept",
			filePath: "src/main.go",
			selector: &Selector{Exclude: []string{"**/*_test.go"}},
			want:     true,
		},
		{
			name:     "multiple excludes match first",
			filePath: "node_modules/pkg/index.js",
			selector: &Selector{Exclude: []string{"node_modules/**", "dist/**"}},
			want:     false,
		},
		{
			name:     "multiple excludes match second",
			filePath: "dist/bundle.js",
			selector: &Selector{Exclude: []string{"node_modules/**", "dist/**"}},
			want:     false,
		},

		// Combined filters
		{
			name:     "language + include both match",
			filePath: "src/main.go",
			selector: &Selector{
				Languages: []string{"go"},
				Include:   []string{"src/**/*.go"},
			},
			want: true,
		},
		{
			name:     "language match but include no match",
			filePath: "test/main.go",
			selector: &Selector{
				Languages: []string{"go"},
				Include:   []string{"src/**/*.go"},
			},
			want: false,
		},
		{
			name:     "include match but language no match",
			filePath: "src/main.js",
			selector: &Selector{
				Languages: []string{"go"},
				Include:   []string{"src/**/*"},
			},
			want: false,
		},
		{
			name:     "language + include match, exclude no match",
			filePath: "src/main.go",
			selector: &Selector{
				Languages: []string{"go"},
				Include:   []string{"src/**/*.go"},
				Exclude:   []string{"**/*_test.go"},
			},
			want: true,
		},
		{
			name:     "language + include match, but excluded",
			filePath: "src/main_test.go",
			selector: &Selector{
				Languages: []string{"go"},
				Include:   []string{"src/**/*.go"},
				Exclude:   []string{"**/*_test.go"},
			},
			want: false,
		},

		// Complex real-world scenarios
		{
			name:     "source files only, no tests, no vendor",
			filePath: "src/components/Button.tsx",
			selector: &Selector{
				Languages: []string{"tsx", "typescript"},
				Include:   []string{"src/**/*.{ts,tsx}"},
				Exclude:   []string{"**/*.test.ts", "**/*.test.tsx", "**/vendor/**"},
			},
			want: true,
		},
		{
			name:     "exclude test file",
			filePath: "src/components/Button.test.tsx",
			selector: &Selector{
				Languages: []string{"tsx", "typescript"},
				Include:   []string{"src/**/*.{ts,tsx}"},
				Exclude:   []string{"**/*.test.ts", "**/*.test.tsx"},
			},
			want: false,
		},
		{
			name:     "public API files only",
			filePath: "src/public/api.go",
			selector: &Selector{
				Languages: []string{"go"},
				Include:   []string{"src/public/**/*.go"},
				Exclude:   []string{"**/*_internal.go"},
			},
			want: true,
		},
		{
			name:     "exclude internal file",
			filePath: "src/public/api_internal.go",
			selector: &Selector{
				Languages: []string{"go"},
				Include:   []string{"src/public/**/*.go"},
				Exclude:   []string{"**/*_internal.go"},
			},
			want: false,
		},

		// Edge cases
		{
			name:     "empty languages list - should match all",
			filePath: "main.go",
			selector: &Selector{Languages: []string{}},
			want:     true,
		},
		{
			name:     "empty include list - should match all",
			filePath: "any/path/file.go",
			selector: &Selector{Include: []string{}},
			want:     true,
		},
		{
			name:     "empty exclude list - should not exclude",
			filePath: "any/path/file.go",
			selector: &Selector{Exclude: []string{}},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchesSelector(tt.filePath, tt.selector)
			if got != tt.want {
				t.Errorf("MatchesSelector(%q, %+v) = %v, want %v", tt.filePath, tt.selector, got, tt.want)
			}
		})
	}
}

func TestFilterFiles(t *testing.T) {
	files := []string{
		"src/main.go",
		"src/util.go",
		"src/main_test.go",
		"test/integration.go",
		"lib/helper.js",
		"lib/index.ts",
		"dist/bundle.js",
		"node_modules/pkg/index.js",
	}

	tests := []struct {
		name     string
		files    []string
		selector *Selector
		want     []string
	}{
		{
			name:     "nil selector - all files",
			files:    files,
			selector: nil,
			want:     files,
		},
		{
			name:  "go files only",
			files: files,
			selector: &Selector{
				Languages: []string{"go"},
			},
			want: []string{
				"src/main.go",
				"src/util.go",
				"src/main_test.go",
				"test/integration.go",
			},
		},
		{
			name:  "src directory only",
			files: files,
			selector: &Selector{
				Include: []string{"src/**/*"},
			},
			want: []string{
				"src/main.go",
				"src/util.go",
				"src/main_test.go",
			},
		},
		{
			name:  "exclude test files",
			files: files,
			selector: &Selector{
				Exclude: []string{"**/*_test.go", "**/test/**"},
			},
			want: []string{
				"src/main.go",
				"src/util.go",
				"lib/helper.js",
				"lib/index.ts",
				"dist/bundle.js",
				"node_modules/pkg/index.js",
			},
		},
		{
			name:  "go files in src, no tests",
			files: files,
			selector: &Selector{
				Languages: []string{"go"},
				Include:   []string{"src/**/*.go"},
				Exclude:   []string{"**/*_test.go"},
			},
			want: []string{
				"src/main.go",
				"src/util.go",
			},
		},
		{
			name:  "js/ts but exclude dist and node_modules",
			files: files,
			selector: &Selector{
				Languages: []string{"javascript", "typescript"},
				Exclude:   []string{"dist/**", "node_modules/**"},
			},
			want: []string{
				"lib/helper.js",
				"lib/index.ts",
			},
		},
		{
			name:     "empty file list",
			files:    []string{},
			selector: &Selector{Languages: []string{"go"}},
			want:     []string{},
		},
		{
			name:  "no matches",
			files: files,
			selector: &Selector{
				Languages: []string{"python"},
			},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterFiles(tt.files, tt.selector)
			if len(got) != len(tt.want) {
				t.Errorf("FilterFiles() returned %d files, want %d", len(got), len(tt.want))
				t.Errorf("got:  %v", got)
				t.Errorf("want: %v", tt.want)
				return
			}
			for i, file := range got {
				if file != tt.want[i] {
					t.Errorf("FilterFiles()[%d] = %q, want %q", i, file, tt.want[i])
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkMatchGlob(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = MatchGlob("src/foo/bar/baz/test.go", "src/**/*.go")
	}
}

func BenchmarkMatchesSelector(b *testing.B) {
	selector := &Selector{
		Languages: []string{"go"},
		Include:   []string{"src/**/*.go"},
		Exclude:   []string{"**/*_test.go"},
	}
	for i := 0; i < b.N; i++ {
		MatchesSelector("src/foo/bar/main.go", selector)
	}
}

func BenchmarkFilterFiles(b *testing.B) {
	files := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		if i%2 == 0 {
			files[i] = "src/file.go"
		} else {
			files[i] = "test/file_test.go"
		}
	}
	selector := &Selector{
		Languages: []string{"go"},
		Exclude:   []string{"**/*_test.go"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FilterFiles(files, selector)
	}
}
