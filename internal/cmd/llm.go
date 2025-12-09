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

	// Dynamically generate model aliases from registry
	fmt.Println("Supported model aliases:")
	for _, p := range providers {
		if len(p.Models) > 0 {
			modelIDs := make([]string, 0, len(p.Models))
			for _, m := range p.Models {
				modelIDs = append(modelIDs, m.ID)
			}
			fmt.Printf("  %s: %s\n", p.DisplayName, strings.Join(modelIDs, ", "))
		}
	}
	fmt.Println()

	// Show API key instructions for providers that require them
	for _, p := range providers {
		if p.APIKey.Required && p.APIKey.EnvVarName != "" {
			fmt.Printf("For %s, also add to .sym/.env:\n", p.DisplayName)
			fmt.Printf("  %s=%s...\n", p.APIKey.EnvVarName, p.APIKey.Prefix)
			fmt.Println()
		}
	}

	fmt.Println("After configuration, run 'sym llm test' to verify.")
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

	// Get provider options dynamically from registry
	providerOptions := llm.GetProviderOptions(true) // includes "Skip"

	// Select provider
	var selectedDisplayName string
	providerPrompt := &survey.Select{
		Message: "Select LLM provider:",
		Options: providerOptions,
	}

	if err := survey.AskOne(providerPrompt, &selectedDisplayName); err != nil {
		fmt.Println("Skipped LLM configuration")
		return
	}

	if selectedDisplayName == "Skip" {
		fmt.Println("Skipped LLM configuration")
		fmt.Println(ui.Indent("Tip: Run 'sym llm setup' to configure later"))
		return
	}

	// Get provider info from registry
	providerInfo := llm.GetProviderByDisplayName(selectedDisplayName)
	if providerInfo == nil {
		ui.PrintError(fmt.Sprintf("Unknown provider: %s", selectedDisplayName))
		return
	}

	providerName := providerInfo.Name
	var modelID string

	// Handle API key if required
	if llm.RequiresAPIKey(providerName) {
		if err := promptAndSaveAPIKey(providerName); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to save API key: %v", err))
			return
		}
	}

	// Select model (common for all providers)
	modelOptions := llm.GetModelOptions(providerName)
	if len(modelOptions) > 0 {
		var selectedOption string
		modelPrompt := &survey.Select{
			Message: fmt.Sprintf("Select %s model:", providerInfo.DisplayName),
			Options: modelOptions,
			Default: llm.GetDefaultModelOption(providerName),
		}
		if err := survey.AskOne(modelPrompt, &selectedOption); err != nil {
			fmt.Println("Skipped model selection, using default")
			modelID = providerInfo.DefaultModel
		} else {
			modelID = llm.GetModelIDFromOption(providerName, selectedOption)
		}
	} else {
		modelID = providerInfo.DefaultModel
	}

	// Save to config.json
	if err := config.UpdateProjectConfigLLM(providerName, modelID); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to save config: %v", err))
		return
	}

	ui.PrintOK(fmt.Sprintf("LLM provider saved: %s (%s)", selectedDisplayName, modelID))
}

// promptAndSaveAPIKey prompts for API key and saves to .env
func promptAndSaveAPIKey(providerName string) error {
	envVarName := llm.GetAPIKeyEnvVar(providerName)
	if envVarName == "" {
		return fmt.Errorf("provider %s does not have API key configuration", providerName)
	}

	var apiKey string
	prompt := &survey.Password{
		Message: fmt.Sprintf("Enter your %s:", envVarName),
	}

	if err := survey.AskOne(prompt, &apiKey); err != nil {
		return err
	}

	// Validate API key using registry
	if err := llm.ValidateAPIKey(providerName, apiKey); err != nil {
		ui.PrintWarn(err.Error())
		// Continue anyway - it's a warning, not a blocking error
		// But if the key is empty, we should return the error
		if apiKey == "" {
			return err
		}
	}

	// Save to .env file
	envPath := config.GetProjectEnvPath()
	if err := saveAPIKeyToEnv(envPath, envVarName, apiKey); err != nil {
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
func saveAPIKeyToEnv(envPath, envVarName, apiKey string) error {
	return envutil.SaveKeyToEnvFile(envPath, envVarName, apiKey)
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

