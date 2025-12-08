package linter

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ===== Path/Directory Helpers =====

// DefaultToolsDir returns the standard tools directory (~/.sym/tools).
// Used by all linters for consistent tool installation location.
func DefaultToolsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".sym", "tools")
}

// EnsureDir creates directory if it doesn't exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// FindTool locates a tool binary, checking local path first, then global PATH.
// Returns empty string if not found.
func FindTool(localPath, globalName string) string {
	if localPath != "" {
		if _, err := os.Stat(localPath); err == nil {
			return localPath
		}
	}
	if path, err := exec.LookPath(globalName); err == nil {
		return path
	}
	return ""
}

// ===== Config File Helpers =====

// WriteTempConfig writes config content to a temp file in the tools directory.
// Returns the path to the created temp file.
func WriteTempConfig(toolsDir string, prefix string, content []byte) (string, error) {
	tmpDir := filepath.Join(toolsDir, ".tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", err
	}

	tmpFile, err := os.CreateTemp(tmpDir, prefix+"-*.json")
	if err != nil {
		return "", err
	}
	defer func() { _ = tmpFile.Close() }()

	if _, err := tmpFile.Write(content); err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

// ===== Severity Helpers =====

// MapSeverity normalizes severity strings to standard values.
// Returns "error", "warning", or "info".
func MapSeverity(s string) string {
	switch strings.ToLower(s) {
	case "error", "err", "fatal", "critical":
		return "error"
	case "warning", "warn":
		return "warning"
	default:
		return "info"
	}
}

// MapPriority converts numeric priority to severity string.
// Priority 1 = error, 2-3 = warning, 4+ = info.
func MapPriority(priority int) string {
	switch priority {
	case 1:
		return "error"
	case 2, 3:
		return "warning"
	default:
		return "info"
	}
}

// ===== Response Parsing Helpers =====

// CleanJSONResponse removes markdown code block markers from LLM responses.
// This is commonly needed when parsing JSON responses from language models.
func CleanJSONResponse(response string) string {
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	return strings.TrimSpace(response)
}
