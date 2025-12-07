package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/DevSymphony/sym-cli/internal/config"
	"github.com/DevSymphony/sym-cli/internal/envutil"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/internal/ui"
	"github.com/spf13/cobra"
)

var llmCmd = &cobra.Command{
	Use:   "llm",
	Short: "Manage LLM provider configuration",
	Long: `Configure and manage LLM providers for Symphony.

Symphony supports multiple LLM providers:
  - claudecode: Claude Code CLI (requires 'claude' in PATH)
  - geminicli: Gemini CLI (requires 'gemini' in PATH)
  - openaiapi: OpenAI API (requires OPENAI_API_KEY)

Configuration is stored in:
  - .sym/config.json: Provider and model settings (safe to commit)
  - .sym/.env: API keys (gitignored)`,
}

var llmStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current LLM provider status",
	Long:  `Display the current LLM provider configuration and availability.`,
	Run:   runLLMStatus,
}

var llmTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test LLM provider connection",
	Long:  `Send a test request to verify LLM provider is working.`,
	Run:   runLLMTest,
}

var llmSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Show LLM setup instructions",
	Long:  `Display instructions for configuring LLM providers.`,
	Run:   runLLMSetup,
}

func init() {
	rootCmd.AddCommand(llmCmd)
	llmCmd.AddCommand(llmStatusCmd)
	llmCmd.AddCommand(llmTestCmd)
	llmCmd.AddCommand(llmSetupCmd)
}

func runLLMStatus(_ *cobra.Command, _ []string) {
	ui.PrintTitle("LLM", "Provider Status")
	fmt.Println()

	// Load config
	cfg := llm.LoadConfig()

	fmt.Println("Configuration:")
	if cfg.Provider != "" {
		fmt.Printf("  Provider: %s\n", cfg.Provider)
	} else {
		fmt.Println("  Provider: (not configured)")
	}
	if cfg.Model != "" {
		fmt.Printf("  Model: %s\n", cfg.Model)
	}
	fmt.Println()

	// Show available providers
	fmt.Println("Available providers:")
	providers := llm.ListProviders()
	for _, p := range providers {
		status := "not available"
		if p.Available {
			status = "available"
			if p.Path != "" {
				status = fmt.Sprintf("available (%s)", p.Path)
			}
		}
		fmt.Printf("  %s: %s\n", p.DisplayName, status)
	}
	fmt.Println()

	// Try to create provider
	provider, err := llm.New(cfg)
	if err != nil {
		ui.PrintWarn(fmt.Sprintf("Configuration error: %v", err))
	} else {
		ui.PrintOK(fmt.Sprintf("Active provider: %s", provider.Name()))
	}

	fmt.Println()
	fmt.Println("Run 'sym llm setup' for configuration instructions")
	fmt.Println("Run 'sym llm test' to verify connection")
}

func runLLMTest(_ *cobra.Command, _ []string) {
	ui.PrintTitle("LLM", "Testing Provider Connection")
	fmt.Println()

	// Load config
	cfg := llm.LoadConfig()

	// Create provider
	provider, err := llm.New(cfg)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to create provider: %v", err))
		fmt.Println()
		fmt.Println("Please configure a provider:")
		fmt.Println("  sym llm setup")
		return
	}

	fmt.Printf("Testing provider: %s\n\n", provider.Name())

	// Create test request
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	prompt := "You are a helpful assistant. Respond with exactly one word.\n\nSay 'OK' to confirm you are working."
	response, err := provider.Execute(ctx, prompt, llm.Text)

	if err != nil {
		ui.PrintError(fmt.Sprintf("Test failed: %v", err))
		return
	}

	ui.PrintOK("Test successful!")
	fmt.Printf("  Response: %s\n", strings.TrimSpace(response))
}

func runLLMSetup(_ *cobra.Command, _ []string) {
	ui.PrintTitle("LLM", "Provider Setup Instructions")
	fmt.Println()

	// Show available providers
	fmt.Println("Available providers:")
	providers := llm.ListProviders()
	for _, p := range providers {
		status := "not installed"
		if p.Available {
			status = "ready"
		}
		fmt.Printf("  %s (%s): %s\n", p.Name, p.DisplayName, status)
	}
	fmt.Println()

	fmt.Println("Configuration files:")
	fmt.Println("  .sym/config.json - Provider and model settings (safe to commit)")
	fmt.Println("  .sym/.env - API keys only (gitignored)")
	fmt.Println()

	fmt.Println("Example .sym/config.json:")
	fmt.Println(`  {
    "llm": {
      "provider": "claudecode",
      "model": "sonnet"
    }
  }`)
	fmt.Println()
	fmt.Println("Supported model aliases:")
	fmt.Println("  Claude: sonnet, opus, haiku")
	fmt.Println("  Gemini: gemini-2.5-flash, gemini-2.5-pro")
	fmt.Println()

	fmt.Println("For OpenAI API, also add to .sym/.env:")
	fmt.Println("  OPENAI_API_KEY=sk-...")
	fmt.Println()

	fmt.Println("After configuration, run 'sym llm test' to verify.")
}

// LLM provider options and model mappings
var llmProviderOptions = []string{
	"OpenAI API",
	"Claude Code",
	"Gemini CLI",
	"Skip",
}

var llmProviderToName = map[string]string{
	"OpenAI API":  "openaiapi",
	"Claude Code": "claudecode",
	"Gemini CLI":  "geminicli",
}

// Claude model options with descriptions
// Note: Claude CLI accepts short aliases like "sonnet", "opus", "haiku"
var claudeModelOptions = []string{
	"sonnet - Balanced performance and speed (recommended)",
	"opus   - Highest capability",
	"haiku  - Fast and efficient",
}

var claudeModelToShortName = map[string]string{
	"sonnet - Balanced performance and speed (recommended)": "sonnet",
	"opus   - Highest capability":                           "opus",
	"haiku  - Fast and efficient":                           "haiku",
}

// Gemini model options with descriptions
var geminiModelOptions = []string{
	"2.5 flash - Fast and efficient (recommended)",
	"2.5 pro   - Higher capability",
}

var geminiModelToShortName = map[string]string{
	"2.5 flash - Fast and efficient (recommended)": "gemini-2.5-flash",
	"2.5 pro   - Higher capability":                "gemini-2.5-pro",
}

// OpenAI model options with descriptions
var openaiModelOptions = []string{
	"gpt-4o-mini - Fast and efficient (recommended)",
	"gpt-5-mini  - Next generation model",
}

var openaiModelToID = map[string]string{
	"gpt-4o-mini - Fast and efficient (recommended)": "gpt-4o-mini",
	"gpt-5-mini  - Next generation model":            "gpt-5-mini",
}

// promptLLMBackendSetup is called from init command to setup LLM provider.
func promptLLMBackendSetup() {
	// Use custom template to hide "type to filter" and typed characters
	restore := useSelectTemplateNoFilter()
	defer restore()

	fmt.Println()
	ui.PrintTitle("LLM", "Configure LLM Provider")
	fmt.Println(ui.Indent("Symphony uses LLM for policy conversion and code validation"))
	fmt.Println()

	// Select provider
	var selectedProvider string
	providerPrompt := &survey.Select{
		Message: "Select LLM provider:",
		Options: llmProviderOptions,
	}

	if err := survey.AskOne(providerPrompt, &selectedProvider); err != nil {
		fmt.Println("Skipped LLM configuration")
		return
	}

	if selectedProvider == "Skip" {
		fmt.Println("Skipped LLM configuration")
		fmt.Println(ui.Indent("Tip: Run 'sym init --setup-llm' to configure later"))
		return
	}

	providerName := llmProviderToName[selectedProvider]
	var modelID string

	// Handle provider-specific configuration
	switch selectedProvider {
	case "OpenAI API":
		// Prompt for API key
		if err := promptAndSaveAPIKey(); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to save API key: %v", err))
			return
		}
		// Select OpenAI model
		var selectedOption string
		modelPrompt := &survey.Select{
			Message: "Select OpenAI model:",
			Options: openaiModelOptions,
			Default: openaiModelOptions[0], // gpt-4o-mini (recommended)
		}
		if err := survey.AskOne(modelPrompt, &selectedOption); err != nil {
			fmt.Println("Skipped model selection, using default")
			modelID = "gpt-4o-mini"
		} else {
			modelID = openaiModelToID[selectedOption]
		}

	case "Claude Code":
		// Select Claude model
		var selectedOption string
		modelPrompt := &survey.Select{
			Message: "Select Claude model:",
			Options: claudeModelOptions,
			Default: claudeModelOptions[0], // sonnet (recommended)
		}
		if err := survey.AskOne(modelPrompt, &selectedOption); err != nil {
			fmt.Println("Skipped model selection, using default")
			modelID = "sonnet"
		} else {
			modelID = claudeModelToShortName[selectedOption]
		}

	case "Gemini CLI":
		// Select Gemini model
		var selectedOption string
		modelPrompt := &survey.Select{
			Message: "Select Gemini model:",
			Options: geminiModelOptions,
			Default: geminiModelOptions[0], // 2.5 flash (recommended)
		}
		if err := survey.AskOne(modelPrompt, &selectedOption); err != nil {
			fmt.Println("Skipped model selection, using default")
			modelID = "gemini-2.5-flash"
		} else {
			modelID = geminiModelToShortName[selectedOption]
		}
	}

	// Save to config.json
	if err := config.UpdateProjectConfigLLM(providerName, modelID); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to save config: %v", err))
		return
	}

	ui.PrintOK(fmt.Sprintf("LLM provider saved: %s (%s)", selectedProvider, modelID))
}

// promptAndSaveAPIKey prompts for OpenAI API key and saves to .env
func promptAndSaveAPIKey() error {
	var apiKey string
	prompt := &survey.Password{
		Message: "Enter your OpenAI API key:",
	}

	if err := survey.AskOne(prompt, &apiKey); err != nil {
		return err
	}

	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	// Validate API key format
	if !strings.HasPrefix(apiKey, "sk-") {
		ui.PrintWarn("API key should start with 'sk-'")
	}

	// Save to .env file
	envPath := config.GetProjectEnvPath()
	if err := saveAPIKeyToEnv(envPath, apiKey); err != nil {
		return err
	}

	ui.PrintOK("API key saved to .sym/.env (gitignored)")

	// Ensure .env is in .gitignore
	if err := ensureGitignore(".sym/.env"); err != nil {
		ui.PrintWarn(fmt.Sprintf("Failed to update .gitignore: %v", err))
	}

	return nil
}

// saveAPIKeyToEnv saves the API key to the .env file
func saveAPIKeyToEnv(envPath, apiKey string) error {
	// Use existing envutil package
	return envutil.SaveKeyToEnvFile(envPath, "OPENAI_API_KEY", apiKey)
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

