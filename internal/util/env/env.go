package env

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// GetAPIKey retrieves an API key from environment or .sym/.env
// It checks system environment variable first, then .sym/.env file
func GetAPIKey(keyName string) string {
	// 1. Check system environment variable first
	if key := os.Getenv(keyName); key != "" {
		return key
	}

	// 2. Check .sym/.env file
	return LoadKeyFromEnvFile(filepath.Join(".sym", ".env"), keyName)
}

// LoadKeyFromEnvFile reads a specific key from .env file
func LoadKeyFromEnvFile(envPath, key string) string {
	file, err := os.Open(envPath)
	if err != nil {
		return ""
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	prefix := key + "="

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip comments and empty lines
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		// Check if line starts with our key
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(line[len(prefix):])
		}
	}

	return ""
}

// SaveKeyToEnvFile saves a key-value pair to .env file
// It preserves existing lines, comments, and blank lines
func SaveKeyToEnvFile(envPath, key, value string) error {
	// Create .sym directory if it doesn't exist
	symDir := filepath.Dir(envPath)
	if err := os.MkdirAll(symDir, 0755); err != nil {
		return err
	}

	// Read existing content
	var lines []string
	keyFound := false
	existingFile, err := os.Open(envPath)
	if err == nil {
		scanner := bufio.NewScanner(existingFile)
		for scanner.Scan() {
			line := scanner.Text()
			trimmed := strings.TrimSpace(line)

			// Check if this line defines the key we're updating
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") && strings.HasPrefix(trimmed, key+"=") {
				// Replace existing key with new value
				lines = append(lines, key+"="+value)
				keyFound = true
			} else {
				// Keep all other lines (including comments and blank lines)
				lines = append(lines, line)
			}
		}
		_ = existingFile.Close()
	} else if !os.IsNotExist(err) {
		return err
	}

	// If key not found, add it at the end
	if !keyFound {
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "") // Add blank line before new key
		}
		lines = append(lines, key+"="+value)
	}

	// Write to file
	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(envPath, []byte(content), 0600)
}
