package importer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SupportedFormats lists all supported file extensions
var SupportedFormats = map[string]bool{
	// Text documents
	".txt":      true,
	".md":       true,
	".markdown": true,
	// Code files
	".go":    true,
	".js":    true,
	".ts":    true,
	".jsx":   true,
	".tsx":   true,
	".py":    true,
	".java":  true,
	".rs":    true,
	".rb":    true,
	".php":   true,
	".c":     true,
	".cpp":   true,
	".h":     true,
	".hpp":   true,
	".cs":    true,
	".swift": true,
	".kt":    true,
	".scala": true,
	// Config/data files
	".yaml": true,
	".yml":  true,
	".json": true,
	".toml": true,
	".xml":  true,
	// Web files
	".html": true,
	".htm":  true,
	".css":  true,
	".scss": true,
	".less": true,
	// Other
	".rst":  true,
	".adoc": true,
}

// MaxFileSizeBytes is the maximum size for a single file (50KB for LLM context)
const MaxFileSizeBytes = 50 * 1024

// Reader handles file reading and format detection
type Reader struct {
	verbose bool
}

// NewReader creates a new Reader instance
func NewReader(verbose bool) *Reader {
	return &Reader{verbose: verbose}
}

// ReadFile reads a single file and extracts text content
func (r *Reader) ReadFile(ctx context.Context, filePath string) (*DocumentContent, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if path exists
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	// Ensure it's a file, not a directory
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file: %s", absPath)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(absPath))
	if !IsSupportedFormat(ext) {
		return nil, fmt.Errorf("unsupported format: %s", ext)
	}

	// Check file size
	if info.Size() == 0 {
		return nil, fmt.Errorf("empty file")
	}

	if info.Size() > MaxFileSizeBytes {
		return nil, fmt.Errorf("file too large (%d bytes, max %d bytes)", info.Size(), MaxFileSizeBytes)
	}

	// Read file content
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return &DocumentContent{
		Path:    absPath,
		Content: string(content),
		Format:  ext,
		Size:    info.Size(),
	}, nil
}

// IsSupportedFormat checks if a file extension is supported
func IsSupportedFormat(ext string) bool {
	return SupportedFormats[strings.ToLower(ext)]
}

// GetSupportedExtensions returns a list of supported extensions
func GetSupportedExtensions() []string {
	var exts []string
	for ext := range SupportedFormats {
		exts = append(exts, ext)
	}
	return exts
}
