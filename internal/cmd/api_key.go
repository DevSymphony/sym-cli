package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/DevSymphony/sym-cli/internal/envutil"
	"github.com/DevSymphony/sym-cli/internal/ui"
)

// promptAPIKeySetup prompts user to setup API key (without checking if it exists)
func promptAPIKeySetup() {
	promptAPIKeyConfiguration(false)
}

// promptAPIKeyConfiguration handles API key configuration with optional existence check
func promptAPIKeyConfiguration(checkExisting bool) {
	envPath := filepath.Join(".sym", ".env")

	if checkExisting {
		// 1. Check environment variable or .env file
		if envutil.GetAPIKey("OPENAI_API_KEY") != "" {
			ui.PrintOK("OpenAI API key detected from environment or .sym/.env")
			return
		}

		// 2. Check .sym/.env file
		if hasAPIKeyInEnvFile(envPath) {
			ui.PrintOK("OpenAI API key found in .sym/.env")
			return
		}

		// Neither found - show warning
		ui.PrintWarn("OpenAI API key not found")
		fmt.Println(ui.Indent("Required for convert, validate commands and MCP auto-conversion"))
		fmt.Println()
	}

	// Create selection prompt
	options := []string{
		"Enter API key",
		"Skip (set manually later)",
	}

	var selected string
	prompt := &survey.Select{
		Message: "Would you like to configure it now?",
		Options: options,
	}

	if err := survey.AskOne(prompt, &selected); err != nil {
		fmt.Println("Skipped API key configuration")
		return
	}

	switch selected {
	case "Enter API key":
		apiKey, err := promptForAPIKeyWithSurvey()
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to read API key: %v", err))
			return
		}

		// Validate API key format
		if err := validateAPIKey(apiKey); err != nil {
			ui.PrintWarn(fmt.Sprintf("%v", err))
			fmt.Println(ui.Indent("API key was saved anyway. Make sure it's correct."))
		}

		// Save to .sym/.env
		if err := envutil.SaveKeyToEnvFile(envPath, "OPENAI_API_KEY", apiKey); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to save API key: %v", err))
			return
		}

		ui.PrintOK("API key saved to .sym/.env")

		// Add to .gitignore
		if err := ensureGitignore(".sym/.env"); err != nil {
			ui.PrintWarn(fmt.Sprintf("Failed to update .gitignore: %v", err))
			fmt.Println(ui.Indent("Please manually add '.sym/.env' to .gitignore"))
		} else {
			ui.PrintOK("Added .sym/.env to .gitignore")
		}

	case "Skip (set manually later)":
		fmt.Println("Skipped API key configuration")
		fmt.Println()
		fmt.Println("Tip: You can set OPENAI_API_KEY in:")
		fmt.Println(ui.Indent(".sym/.env file"))
		fmt.Println(ui.Indent("System environment variable"))
	}
}

// promptForAPIKeyWithSurvey prompts user to enter API key using survey
func promptForAPIKeyWithSurvey() (string, error) {
	var apiKey string
	prompt := &survey.Password{
		Message: "Enter your OpenAI API key:",
	}

	if err := survey.AskOne(prompt, &apiKey); err != nil {
		return "", err
	}

	// Clean the input
	apiKey = cleanAPIKey(apiKey)

	if len(apiKey) == 0 {
		return "", fmt.Errorf("API key cannot be empty")
	}

	return apiKey, nil
}

// promptForAPIKey prompts user to enter API key (legacy, kept for compatibility)
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
// Returns error if not found
func getAPIKey() (string, error) {
	key := envutil.GetAPIKey("OPENAI_API_KEY")
	if key == "" {
		return "", fmt.Errorf("OPENAI_API_KEY not found in environment or .sym/.env")
	}
	return key, nil
}
