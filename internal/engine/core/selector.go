package core

import (
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// MatchGlob checks if a file path matches a glob pattern.
// Supports doublestar patterns (e.g., "**/*.js", "src/**/test_*.go").
func MatchGlob(filePath, pattern string) (bool, error) {
	// Normalize paths for consistent matching
	filePath = filepath.ToSlash(filePath)
	pattern = filepath.ToSlash(pattern)

	return doublestar.Match(pattern, filePath)
}

// MatchesLanguage checks if a file extension matches a language.
func MatchesLanguage(filePath, language string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))

	// Language to extension mapping
	langExtMap := map[string][]string{
		"javascript": {".js", ".mjs", ".cjs"},
		"js":         {".js", ".mjs", ".cjs"},
		"typescript": {".ts", ".mts", ".cts"},
		"ts":         {".ts", ".mts", ".cts"},
		"jsx":        {".jsx"},
		"tsx":        {".tsx"},
		"python":     {".py", ".pyi", ".pyw"},
		"py":         {".py", ".pyi", ".pyw"},
		"go":         {".go"},
		"golang":     {".go"},
		"java":       {".java"},
		"c":          {".c", ".h"},
		"cpp":        {".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx"},
		"c++":        {".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx"},
		"rust":       {".rs"},
		"ruby":       {".rb"},
		"php":        {".php"},
		"swift":      {".swift"},
		"kotlin":     {".kt", ".kts"},
		"scala":      {".scala"},
		"shell":      {".sh", ".bash", ".zsh"},
		"sh":         {".sh", ".bash", ".zsh"},
	}

	exts, ok := langExtMap[strings.ToLower(language)]
	if !ok {
		return false
	}

	for _, e := range exts {
		if ext == e {
			return true
		}
	}
	return false
}

// MatchesSelector checks if a file matches the selector criteria.
// Returns true if the file passes all selector filters.
func MatchesSelector(filePath string, selector *Selector) bool {
	if selector == nil {
		return true
	}

	// Normalize file path
	filePath = filepath.ToSlash(filePath)

	// Language filter
	if len(selector.Languages) > 0 {
		matched := false
		for _, lang := range selector.Languages {
			if MatchesLanguage(filePath, lang) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Include filter (if specified, file must match at least one pattern)
	if len(selector.Include) > 0 {
		matched := false
		for _, pattern := range selector.Include {
			if m, err := MatchGlob(filePath, pattern); err == nil && m {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Exclude filter (if file matches any pattern, exclude it)
	if len(selector.Exclude) > 0 {
		for _, pattern := range selector.Exclude {
			if m, err := MatchGlob(filePath, pattern); err == nil && m {
				return false
			}
		}
	}

	return true
}

// FilterFiles filters a list of files based on the selector criteria.
// Returns a new slice containing only the files that match.
func FilterFiles(files []string, selector *Selector) []string {
	if selector == nil {
		return files
	}

	filtered := make([]string, 0, len(files))
	for _, file := range files {
		if MatchesSelector(file, selector) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}
