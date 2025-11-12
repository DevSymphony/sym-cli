package validator

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// FileSelector handles file discovery and filtering based on rule selectors
type FileSelector struct {
	basePath string
}

// NewFileSelector creates a new file selector
func NewFileSelector(basePath string) *FileSelector {
	return &FileSelector{
		basePath: basePath,
	}
}

// SelectFiles finds files that match the given selector
func (fs *FileSelector) SelectFiles(selector *schema.Selector) ([]string, error) {
	if selector == nil {
		// No selector means match all files
		return fs.findAllFiles()
	}

	// Start with all files
	allFiles, err := fs.findAllFiles()
	if err != nil {
		return nil, err
	}

	// Filter by include patterns
	files := allFiles
	if len(selector.Include) > 0 {
		files = fs.filterByPatterns(allFiles, selector.Include, true)
	}

	// Filter out excluded files
	if len(selector.Exclude) > 0 {
		files = fs.filterByPatterns(files, selector.Exclude, false)
	}

	// Filter by language (based on file extension)
	if len(selector.Languages) > 0 {
		files = fs.filterByLanguages(files, selector.Languages)
	}

	return files, nil
}

// findAllFiles recursively finds all files under basePath
func (fs *FileSelector) findAllFiles() ([]string, error) {
	var files []string

	err := filepath.Walk(fs.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Skip common directories to ignore
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" || name == ".sym" {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path from base
		relPath, err := filepath.Rel(fs.basePath, path)
		if err != nil {
			return err
		}

		files = append(files, relPath)
		return nil
	})

	return files, err
}

// filterByPatterns filters files by glob patterns
func (fs *FileSelector) filterByPatterns(files []string, patterns []string, include bool) []string {
	var result []string

	for _, file := range files {
		matched := false
		for _, pattern := range patterns {
			if match, _ := filepath.Match(pattern, file); match {
				matched = true
				break
			}
			// Also try glob pattern matching (e.g., **/*.js)
			if matchGlob(file, pattern) {
				matched = true
				break
			}
		}

		// Include mode: keep if matched
		// Exclude mode: keep if NOT matched
		if (include && matched) || (!include && !matched) {
			result = append(result, file)
		}
	}

	return result
}

// filterByLanguages filters files by programming language
func (fs *FileSelector) filterByLanguages(files []string, languages []string) []string {
	var result []string

	extMap := buildLanguageExtensionMap(languages)

	for _, file := range files {
		ext := filepath.Ext(file)
		if _, ok := extMap[ext]; ok {
			result = append(result, file)
		}
	}

	return result
}

// buildLanguageExtensionMap creates a map of file extensions for given languages
func buildLanguageExtensionMap(languages []string) map[string]bool {
	extMap := make(map[string]bool)

	for _, lang := range languages {
		exts := getExtensionsForLanguage(lang)
		for _, ext := range exts {
			extMap[ext] = true
		}
	}

	return extMap
}

// getExtensionsForLanguage returns file extensions for a language
func getExtensionsForLanguage(language string) []string {
	language = strings.ToLower(language)

	switch language {
	case "javascript", "js":
		return []string{".js", ".mjs", ".cjs"}
	case "typescript", "ts":
		return []string{".ts", ".mts", ".cts"}
	case "jsx":
		return []string{".jsx"}
	case "tsx":
		return []string{".tsx"}
	case "go", "golang":
		return []string{".go"}
	case "python", "py":
		return []string{".py"}
	case "java":
		return []string{".java"}
	case "c":
		return []string{".c", ".h"}
	case "cpp", "c++", "cxx":
		return []string{".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx"}
	case "rust", "rs":
		return []string{".rs"}
	case "ruby", "rb":
		return []string{".rb"}
	case "php":
		return []string{".php"}
	case "shell", "bash", "sh":
		return []string{".sh", ".bash"}
	case "yaml", "yml":
		return []string{".yaml", ".yml"}
	case "json":
		return []string{".json"}
	case "xml":
		return []string{".xml"}
	case "html":
		return []string{".html", ".htm"}
	case "css":
		return []string{".css"}
	default:
		return []string{}
	}
}

// matchGlob performs glob pattern matching with ** support
func matchGlob(path, pattern string) bool {
	// Convert glob pattern to simple matching
	// This is a simplified implementation - for production, use a proper glob library

	// Handle **/ prefix (matches any directory depth)
	if strings.HasPrefix(pattern, "**/") {
		suffix := strings.TrimPrefix(pattern, "**/")
		// Check if path ends with the suffix or any subdirectory contains it
		if strings.HasSuffix(path, suffix) {
			return true
		}
		// Check if any part matches
		parts := strings.Split(path, string(filepath.Separator))
		for i := range parts {
			subPath := strings.Join(parts[i:], string(filepath.Separator))
			if match, _ := filepath.Match(suffix, subPath); match {
				return true
			}
		}
	}

	// Handle exact match
	if match, _ := filepath.Match(pattern, path); match {
		return true
	}

	// Handle directory prefix patterns (e.g., "src/**")
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		if strings.HasPrefix(path, prefix+string(filepath.Separator)) || path == prefix {
			return true
		}
	}

	return false
}

// GetLanguageFromFile determines the programming language from a file path
func GetLanguageFromFile(filePath string) string {
	ext := filepath.Ext(filePath)

	switch ext {
	case ".js", ".mjs", ".cjs":
		return "javascript"
	case ".ts", ".mts", ".cts":
		return "typescript"
	case ".jsx":
		return "jsx"
	case ".tsx":
		return "tsx"
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".java":
		return "java"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx":
		return "cpp"
	case ".rs":
		return "rust"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".sh", ".bash":
		return "shell"
	default:
		return ""
	}
}