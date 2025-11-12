package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
)

// promptAPIKeySetup prompts user to setup API key (without checking if it exists)
func promptAPIKeySetup() {
	promptAPIKeyConfiguration(false)
}

// promptAPIKeyIfNeeded checks if OpenAI API key is configured and prompts if not
func promptAPIKeyIfNeeded() {
	promptAPIKeyConfiguration(true)
}

// promptAPIKeyConfiguration handles API key configuration with optional existence check
func promptAPIKeyConfiguration(checkExisting bool) {
	envPath := filepath.Join(".sym", ".env")

	if checkExisting {
		// 1. Check environment variable
		if os.Getenv("OPENAI_API_KEY") != "" {
			fmt.Println("\n✓ OpenAI API key detected from environment")
			return
		}

		// 2. Check .sym/.env file
		if hasAPIKeyInEnvFile(envPath) {
			fmt.Println("\n✓ OpenAI API key found in .sym/.env")
			return
		}

		// Neither found - show warning
		fmt.Println("\n⚠ OpenAI API key not found")
		fmt.Println("  (Required for convert, validate commands and MCP auto-conversion)")
		fmt.Println()
	}

	// Create selection prompt
	items := []string{
		"Enter API key",
		"Skip (set manually later)",
	}

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   "▸ {{ . | cyan }}",
		Inactive: "  {{ . }}",
		Selected: "✓ {{ . | green }}",
	}

	selectPrompt := promptui.Select{
		Label:     "Would you like to configure it now",
		Items:     items,
		Templates: templates,
		Size:      2,
	}

	index, _, err := selectPrompt.Run()
	if err != nil {
		fmt.Println("\nSkipped API key configuration")
		return
	}

	switch index {
	case 0: // Enter API key
		apiKey, err := promptForAPIKey()
		if err != nil {
			fmt.Printf("\n❌ Failed to read API key: %v\n", err)
			return
		}

		// Validate API key format
		if err := validateAPIKey(apiKey); err != nil {
			fmt.Printf("\n⚠ Warning: %v\n", err)
			fmt.Println("  API key was saved anyway. Make sure it's correct.")
		}

		// Save to .sym/.env
		if err := saveToEnvFile(envPath, "OPENAI_API_KEY", apiKey); err != nil {
			fmt.Printf("\n❌ Failed to save API key: %v\n", err)
			return
		}

		fmt.Println("\n✓ API key saved to .sym/.env")

		// Add to .gitignore
		if err := ensureGitignore(".sym/.env"); err != nil {
			fmt.Printf("⚠ Warning: Failed to update .gitignore: %v\n", err)
			fmt.Println("  Please manually add '.sym/.env' to .gitignore")
		} else {
			fmt.Println("✓ Added .sym/.env to .gitignore")
		}

	case 1: // Skip
		fmt.Println("\nSkipped API key configuration")
		fmt.Println("\n💡 Tip: You can set OPENAI_API_KEY in:")
		fmt.Println("  - .sym/.env file")
		fmt.Println("  - System environment variable")
	}
}

// promptForAPIKey prompts user to enter API key
func promptForAPIKey() (string, error) {
	fmt.Print("Enter your OpenAI API key: ")

	// Use bufio reader for better paste support
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read API key: %w", err)
	}

	// Clean the input: remove all whitespace, control characters, and non-printable characters
	apiKey := cleanAPIKey(input)

	if len(apiKey) == 0 {
		return "", fmt.Errorf("API key cannot be empty")
	}

	return apiKey, nil
}

// cleanAPIKey removes whitespace, control characters, and non-printable characters from API key
func cleanAPIKey(input string) string {
	var result strings.Builder
	for _, r := range input {
		// Only keep printable ASCII characters (excluding space)
		if r >= 33 && r <= 126 {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// validateAPIKey performs basic validation on API key format
func validateAPIKey(key string) error {
	if !strings.HasPrefix(key, "sk-") {
		return fmt.Errorf("API key should start with 'sk-'")
	}
	if len(key) < 20 {
		return fmt.Errorf("API key seems too short")
	}
	return nil
}

// hasAPIKeyInEnvFile checks if OPENAI_API_KEY exists in .env file
func hasAPIKeyInEnvFile(envPath string) bool {
	file, err := os.Open(envPath)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "OPENAI_API_KEY=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
				return true
			}
		}
	}

	return false
}

// saveToEnvFile saves a key-value pair to .env file
func saveToEnvFile(envPath, key, value string) error {
	// Create .sym directory if it doesn't exist
	symDir := filepath.Dir(envPath)
	if err := os.MkdirAll(symDir, 0755); err != nil {
		return fmt.Errorf("failed to create .sym directory: %w", err)
	}

	// Read existing content
	var lines []string
	existingFile, err := os.Open(envPath)
	if err == nil {
		scanner := bufio.NewScanner(existingFile)
		for scanner.Scan() {
			line := scanner.Text()
			// Skip existing OPENAI_API_KEY lines
			if !strings.HasPrefix(strings.TrimSpace(line), key+"=") {
				lines = append(lines, line)
			}
		}
		_ = existingFile.Close()
	}

	// Add new key
	lines = append(lines, fmt.Sprintf("%s=%s", key, value))

	// Write to file with restrictive permissions (owner read/write only)
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write .env file: %w", err)
	}

	return nil
}

// ensureGitignore ensures that the given path is in .gitignore
func ensureGitignore(path string) error {
	gitignorePath := ".gitignore"

	// Read existing .gitignore
	var lines []string
	existingFile, err := os.Open(gitignorePath)
	if err == nil {
		scanner := bufio.NewScanner(existingFile)
		for scanner.Scan() {
			line := scanner.Text()
			lines = append(lines, line)
			// Check if already exists
			if strings.TrimSpace(line) == path {
				_ = existingFile.Close()
				return nil // Already in .gitignore
			}
		}
		_ = existingFile.Close()
	}

	// Add to .gitignore
	lines = append(lines, "", "# Symphony API key configuration", path)
	content := strings.Join(lines, "\n") + "\n"

	if err := os.WriteFile(gitignorePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to update .gitignore: %w", err)
	}

	return nil
}

// getAPIKey retrieves OpenAI API key from environment or .env file
// Priority: 1) System environment variable 2) .sym/.env file
func getAPIKey() (string, error) {
	// 1. Check system environment variable first
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		return key, nil
	}

	// 2. Check .sym/.env file
	envPath := filepath.Join(".sym", ".env")
	key, err := loadFromEnvFile(envPath, "OPENAI_API_KEY")
	if err == nil && key != "" {
		return key, nil
	}

	return "", fmt.Errorf("OPENAI_API_KEY not found in environment or .sym/.env")
}

// loadFromEnvFile loads a specific key from .env file
func loadFromEnvFile(envPath, key string) (string, error) {
	file, err := os.Open(envPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, key+"=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}

	return "", fmt.Errorf("key %s not found in %s", key, envPath)
}
