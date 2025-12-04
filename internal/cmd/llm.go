package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var llmCmd = &cobra.Command{
	Use:   "llm",
	Short: "Manage LLM engine configuration",
	Long: `Configure and manage LLM engines for Symphony.

Symphony supports multiple LLM engines:
  - MCP Sampling: Uses the host LLM when running as MCP server
  - CLI: Uses local CLI tools (claude, gemini)
  - API: Uses OpenAI API directly

The default mode is 'auto' which tries engines in this order:
MCP Sampling → CLI → API`,
}

var llmSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive LLM engine setup",
	Long:  `Interactively configure which LLM engine to use.`,
	Run:   runLLMSetup,
}

var llmStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current LLM engine status",
	Long:  `Display the current LLM engine configuration and availability.`,
	Run:   runLLMStatus,
}

var llmTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test LLM engine connection",
	Long:  `Send a test request to verify LLM engine is working.`,
	Run:   runLLMTest,
}

func init() {
	rootCmd.AddCommand(llmCmd)
	llmCmd.AddCommand(llmSetupCmd)
	llmCmd.AddCommand(llmStatusCmd)
	llmCmd.AddCommand(llmTestCmd)
}

func runLLMSetup(_ *cobra.Command, _ []string) {
	fmt.Println("🤖 LLM Engine Configuration")
	fmt.Println()

	// Load current config
	cfg := llm.LoadLLMConfig()

	// Show current settings
	fmt.Println("Current settings:")
	fmt.Printf("  Engine mode: %s\n", cfg.Backend)
	if cfg.CLI != "" {
		fmt.Printf("  CLI: %s\n", cfg.CLI)
	}
	if cfg.Model != "" {
		fmt.Printf("  Model: %s\n", cfg.Model)
	}
	if cfg.HasAPIKey() {
		fmt.Println("  API Key: configured")
	} else {
		fmt.Println("  API Key: not set")
	}
	fmt.Println()

	// Show menu
	items := []string{
		"Configure CLI tool",
		"Set OpenAI API key",
		"Change engine mode",
		"Test current configuration",
		"Reset to defaults",
		"Exit",
	}

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   "▸ {{ . | cyan }}",
		Inactive: "  {{ . }}",
		Selected: "✓ {{ . | green }}",
	}

	selectPrompt := promptui.Select{
		Label:     "What would you like to configure",
		Items:     items,
		Templates: templates,
		Size:      6,
	}

	index, _, err := selectPrompt.Run()
	if err != nil {
		fmt.Println("\nSetup cancelled")
		return
	}

	switch index {
	case 0:
		configureCLI(cfg)
	case 1:
		promptAPIKeySetup()
	case 2:
		configureEngineMode(cfg)
	case 3:
		runLLMTest(nil, nil)
	case 4:
		resetLLMConfig()
	case 5:
		fmt.Println("\nExiting setup")
	}
}

func configureCLI(cfg *llm.LLMConfig) {
	fmt.Println("\n🔧 CLI Tool Configuration")
	fmt.Println()

	// Detect available CLIs
	clis := llm.DetectAvailableCLIs()

	// Build selection items
	var items []string
	var availableCLIs []llm.CLIInfo

	for _, cli := range clis {
		status := "✗ not found"
		if cli.Available {
			status = "✓ available"
			if cli.Version != "" {
				status = fmt.Sprintf("✓ %s", cli.Version)
			}
		}
		items = append(items, fmt.Sprintf("%s (%s)", cli.Name, status))
		availableCLIs = append(availableCLIs, cli)
	}

	items = append(items, "Skip CLI configuration")

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   "▸ {{ . | cyan }}",
		Inactive: "  {{ . }}",
		Selected: "✓ {{ . | green }}",
	}

	selectPrompt := promptui.Select{
		Label:     "Select CLI tool to use",
		Items:     items,
		Templates: templates,
		Size:      len(items),
	}

	index, _, err := selectPrompt.Run()
	if err != nil || index >= len(availableCLIs) {
		fmt.Println("\nCLI configuration skipped")
		return
	}

	selectedCLI := availableCLIs[index]

	if !selectedCLI.Available {
		fmt.Printf("\n⚠️  %s is not installed or not in PATH\n", selectedCLI.Name)
		fmt.Println("Please install it first and try again")
		return
	}

	// Update config
	cfg.CLI = selectedCLI.Provider

	// Get provider for default model
	providerInfo := llm.GetCLIProviderInfo(selectedCLI.Provider)
	if providerInfo != nil {
		cfg.Model = providerInfo.DefaultModel
		cfg.LargeModel = providerInfo.LargeModel
	}

	// Save config
	if err := llm.SaveLLMConfig(cfg); err != nil {
		fmt.Printf("\n❌ Failed to save configuration: %v\n", err)
		return
	}

	fmt.Printf("\n✓ CLI engine configured: %s\n", selectedCLI.Name)
	if cfg.Model != "" {
		fmt.Printf("  Default model: %s\n", cfg.Model)
	}
	if cfg.LargeModel != "" {
		fmt.Printf("  Large model: %s\n", cfg.LargeModel)
	}
	fmt.Println("  Configuration saved to .sym/.env")
}

func configureEngineMode(cfg *llm.LLMConfig) {
	fmt.Println("\n⚙️  Engine Mode Configuration")
	fmt.Println()

	items := []string{
		"auto - Automatically select best available engine",
		"cli - Always use CLI tool",
		"api - Always use OpenAI API",
	}

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   "▸ {{ . | cyan }}",
		Inactive: "  {{ . }}",
		Selected: "✓ {{ . | green }}",
	}

	selectPrompt := promptui.Select{
		Label:     "Select engine mode",
		Items:     items,
		Templates: templates,
		Size:      3,
	}

	index, _, err := selectPrompt.Run()
	if err != nil {
		fmt.Println("\nEngine mode configuration cancelled")
		return
	}

	modes := []llm.Mode{
		llm.ModeAuto,
		llm.ModeCLI,
		llm.ModeAPI,
	}

	cfg.Backend = modes[index]

	// Save config
	if err := llm.SaveLLMConfig(cfg); err != nil {
		fmt.Printf("\n❌ Failed to save configuration: %v\n", err)
		return
	}

	fmt.Printf("\n✓ Engine mode set to: %s\n", cfg.Backend)
}

func resetLLMConfig() {
	fmt.Println("\n🔄 Resetting LLM Configuration")

	// Confirm
	prompt := promptui.Prompt{
		Label:     "Are you sure you want to reset LLM configuration",
		IsConfirm: true,
	}

	result, err := prompt.Run()
	if err != nil || strings.ToLower(result) != "y" {
		fmt.Println("\nReset cancelled")
		return
	}

	// Save default config
	cfg := llm.DefaultLLMConfig()
	if err := llm.SaveLLMConfig(cfg); err != nil {
		fmt.Printf("\n❌ Failed to reset configuration: %v\n", err)
		return
	}

	fmt.Println("\n✓ LLM configuration reset to defaults")
}

func runLLMStatus(_ *cobra.Command, _ []string) {
	fmt.Println("🤖 LLM Engine Status")
	fmt.Println()

	// Load config
	cfg := llm.LoadLLMConfig()

	// Create client to check engines
	client := llm.NewClient(llm.WithConfig(cfg), llm.WithVerbose(false))

	fmt.Println("Configuration:")
	fmt.Printf("  Engine mode: %s\n", cfg.Backend)
	if cfg.CLI != "" {
		fmt.Printf("  CLI provider: %s\n", cfg.CLI)
	}
	if cfg.Model != "" {
		fmt.Printf("  Model: %s\n", cfg.Model)
	}
	fmt.Println()

	// Show engine availability
	fmt.Println("Engine availability:")

	engines := client.GetEngines()
	if len(engines) == 0 {
		fmt.Println("  ⚠️  No engines configured")
	} else {
		for _, e := range engines {
			status := "✗ unavailable"
			if e.IsAvailable() {
				status = "✓ available"
			}
			fmt.Printf("  %s: %s\n", e.Name(), status)
		}
	}

	fmt.Println()

	// Show active engine
	active := client.GetActiveEngine()
	if active != nil {
		fmt.Printf("Active engine: %s\n", active.Name())

		caps := active.Capabilities()
		fmt.Println("Capabilities:")
		fmt.Printf("  Temperature: %v\n", caps.SupportsTemperature)
		fmt.Printf("  Max tokens: %v\n", caps.SupportsMaxTokens)
		fmt.Printf("  Complexity hint: %v\n", caps.SupportsComplexity)
	} else {
		fmt.Println("⚠️  No active engine available")
	}

	fmt.Println()
	fmt.Println("💡 Run 'sym llm setup' to configure engines")
	fmt.Println("💡 Run 'sym llm test' to verify connection")
}

func runLLMTest(_ *cobra.Command, _ []string) {
	fmt.Println("🧪 Testing LLM Engine Connection")
	fmt.Println()

	// Load config
	cfg := llm.LoadLLMConfig()

	// Create client
	client := llm.NewClient(llm.WithConfig(cfg), llm.WithVerbose(true))

	active := client.GetActiveEngine()
	if active == nil {
		fmt.Println("❌ No LLM engine available")
		fmt.Println()
		fmt.Println("Please configure an engine:")
		fmt.Println("  sym llm setup")
		return
	}

	fmt.Printf("Testing engine: %s\n\n", active.Name())

	// Create test request
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := client.Request(
		"You are a helpful assistant. Respond with exactly one word.",
		"Say 'OK' to confirm you are working.",
	).Execute(ctx)

	if err != nil {
		fmt.Printf("\n❌ Test failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Test successful!\n")
	fmt.Printf("  Response: %s\n", strings.TrimSpace(response))
}

// promptLLMBackendSetup is called from init command to setup LLM engine.
func promptLLMBackendSetup() {
	fmt.Println("\n🤖 LLM Engine Configuration")
	fmt.Println("  Symphony uses LLM for policy conversion and code validation.")
	fmt.Println()

	// Detect available CLIs
	clis := llm.DetectAvailableCLIs()

	// Check API key
	cfg := llm.LoadLLMConfig()
	hasAPIKey := cfg.HasAPIKey()

	// Show detected tools
	fmt.Println("  Detected LLM tools:")
	hasAnyCLI := false
	for _, cli := range clis {
		status := "✗"
		if cli.Available {
			status = "✓"
			hasAnyCLI = true
		}
		version := ""
		if cli.Version != "" {
			version = fmt.Sprintf(" (%s)", cli.Version)
		}
		fmt.Printf("    %s %s%s\n", status, cli.Name, version)
	}

	if hasAPIKey {
		fmt.Println("    ✓ OpenAI API key (configured)")
	} else {
		fmt.Println("    ✗ OpenAI API key (not set)")
	}
	fmt.Println()

	// If nothing available, skip
	if !hasAnyCLI && !hasAPIKey {
		fmt.Println("  ⚠️  No LLM engine available")
		fmt.Println("  You can configure one later with: sym llm setup")
		return
	}

	// Build selection items
	var items []string
	var modes []llm.Mode

	items = append(items, "Auto (recommended) - Use best available engine")
	modes = append(modes, llm.ModeAuto)

	for _, cli := range clis {
		if cli.Available {
			items = append(items, fmt.Sprintf("%s CLI", cli.Name))
			modes = append(modes, llm.ModeCLI)
		}
	}

	if hasAPIKey {
		items = append(items, "OpenAI API")
		modes = append(modes, llm.ModeAPI)
	}

	items = append(items, "Skip (configure later)")
	modes = append(modes, "")

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   "▸ {{ . | cyan }}",
		Inactive: "  {{ . }}",
		Selected: "✓ {{ . | green }}",
	}

	selectPrompt := promptui.Select{
		Label:     "Select your preferred LLM engine",
		Items:     items,
		Templates: templates,
		Size:      len(items),
		Stdout:    &bellSkipper{},
	}

	index, _, err := selectPrompt.Run()
	if err != nil || modes[index] == "" {
		fmt.Println("\n  LLM engine configuration skipped")
		fmt.Println("  Run 'sym llm setup' to configure later")
		return
	}

	// Update config
	cfg.Backend = modes[index]

	// If CLI selected, set the specific CLI provider
	if modes[index] == llm.ModeCLI {
		// Find which CLI was selected
		cliIndex := index - 1 // Account for "Auto" option
		cliCount := 0
		for _, cli := range clis {
			if cli.Available {
				if cliCount == cliIndex {
					cfg.CLI = cli.Provider
					providerInfo := llm.GetCLIProviderInfo(cli.Provider)
					if providerInfo != nil {
						cfg.Model = providerInfo.DefaultModel
						cfg.LargeModel = providerInfo.LargeModel
					}
					break
				}
				cliCount++
			}
		}
	}

	// Save config
	if err := llm.SaveLLMConfig(cfg); err != nil {
		fmt.Printf("\n  ⚠️  Failed to save LLM configuration: %v\n", err)
		return
	}

	fmt.Printf("\n  ✓ LLM engine set to: %s\n", cfg.Backend)
	if cfg.CLI != "" {
		fmt.Printf("    CLI: %s\n", cfg.CLI)
	}
	fmt.Println("    Configuration saved to .sym/.env")
}
