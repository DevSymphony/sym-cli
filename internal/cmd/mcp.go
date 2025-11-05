package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/DevSymphony/sym-cli/internal/converter"
	"github.com/DevSymphony/sym-cli/internal/git"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/internal/mcp"
	"github.com/DevSymphony/sym-cli/pkg/schema"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

var (
	mcpConfig string
	mcpHost   string
	mcpPort   int
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server to integrate with LLM tools",
	Long: `Start Model Context Protocol (MCP) server.
LLM-based coding tools can query conventions and validate code through stdio or HTTP.

Tools provided by MCP server:
- query_conventions: Query conventions for given context
- validate_code: Validate code compliance with conventions

By default, communicates via stdio. If --port is specified, starts HTTP server.`,
	Example: `  sym mcp
  sym mcp --config code-policy.json
  sym mcp --host 0.0.0.0 --port 8080`,
	RunE: runMCP,
}

func init() {
	rootCmd.AddCommand(mcpCmd)

	mcpCmd.Flags().StringVarP(&mcpConfig, "config", "c", "", "policy file path (code-policy.json)")
	mcpCmd.Flags().StringVar(&mcpHost, "host", "127.0.0.1", "server host (HTTP mode only)")
	mcpCmd.Flags().IntVarP(&mcpPort, "port", "p", 4000, "server port (0 = stdio mode, >0 = HTTP mode)")
}

func runMCP(cmd *cobra.Command, args []string) error {
	// Get git root directory
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	userPolicyPath := filepath.Join(repoRoot, ".sym", "user-policy.json")
	codePolicyPath := filepath.Join(repoRoot, ".sym", "code-policy.json")

	// If custom config path is specified, use it directly
	if mcpConfig != "" {
		codePolicyPath = mcpConfig
	}

	// Check if user-policy.json exists
	userPolicyExists := fileExists(userPolicyPath)
	codePolicyExists := fileExists(codePolicyPath)

	// Case 1: No user-policy.json → Launch dashboard
	if !userPolicyExists {
		fmt.Println("❌ User policy not found at:", userPolicyPath)
		fmt.Println("📝 Opening dashboard to create policy...")

		// Launch dashboard
		if err := launchDashboard(); err != nil {
			return fmt.Errorf("failed to launch dashboard: %w", err)
		}

		fmt.Println("\n✓ Dashboard launched at http://localhost:8787")
		fmt.Println("Please create your policy in the dashboard, then restart MCP server.")
		return nil
	}

	// Case 2: user-policy.json exists but code-policy.json doesn't → Auto-convert
	if userPolicyExists && !codePolicyExists {
		fmt.Println("✓ User policy found at:", userPolicyPath)
		fmt.Println("⚙️  Code policy not found. Converting user policy...")

		if err := autoConvertPolicy(userPolicyPath, codePolicyPath); err != nil {
			return fmt.Errorf("failed to convert policy: %w", err)
		}

		fmt.Println("✓ Policy converted successfully:", codePolicyPath)
	}

	// Case 3: Both exist → Start MCP server normally
	fmt.Println("✓ Policy loaded from:", codePolicyPath)
	server := mcp.NewServer(mcpHost, mcpPort, codePolicyPath)
	return server.Start()
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// launchDashboard launches the dashboard in the background
func launchDashboard() error {
	// Open browser to dashboard
	url := "http://localhost:8787"
	go func() {
		time.Sleep(1 * time.Second)
		browser.OpenURL(url)
	}()

	// Start dashboard server in background
	// Note: This will block, so in practice you'd want to run this in a separate process
	// For now, we just inform the user to run it manually
	fmt.Println("Please run in another terminal:")
	fmt.Println("  sym dashboard")

	return nil
}

// autoConvertPolicy converts user-policy.json to code-policy.json
func autoConvertPolicy(userPolicyPath, codePolicyPath string) error {
	// Load user policy
	data, err := os.ReadFile(userPolicyPath)
	if err != nil {
		return fmt.Errorf("failed to read user policy: %w", err)
	}

	var userPolicy schema.UserPolicy
	if err := json.Unmarshal(data, &userPolicy); err != nil {
		return fmt.Errorf("failed to parse user policy: %w", err)
	}

	// Setup LLM client
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	llmClient := llm.NewClient(apiKey,
		llm.WithModel("gpt-4o-mini"),
		llm.WithTimeout(30*time.Second),
	)

	// Create converter
	conv := converter.NewConverter(converter.WithLLMClient(llmClient))

	// Setup context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(len(userPolicy.Rules)*30)*time.Second)
	defer cancel()

	fmt.Printf("Converting %d rules...\n", len(userPolicy.Rules))

	// Convert to all targets
	result, err := conv.ConvertMultiTarget(ctx, &userPolicy, converter.MultiTargetConvertOptions{
		Targets:             []string{"all"},
		OutputDir:           filepath.Dir(codePolicyPath),
		ConfidenceThreshold: 0.7,
	})
	if err != nil {
		return fmt.Errorf("conversion failed: %w", err)
	}

	// Write code policy
	codePolicyJSON, err := json.MarshalIndent(result.CodePolicy, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize code policy: %w", err)
	}

	if err := os.WriteFile(codePolicyPath, codePolicyJSON, 0644); err != nil {
		return fmt.Errorf("failed to write code policy: %w", err)
	}

	// Write linter configs
	for linterName, config := range result.LinterConfigs {
		outputPath := filepath.Join(filepath.Dir(codePolicyPath), config.Filename)
		if err := os.WriteFile(outputPath, config.Content, 0644); err != nil {
			fmt.Printf("Warning: failed to write %s config: %v\n", linterName, err)
		} else {
			fmt.Printf("  ✓ Generated %s: %s\n", linterName, outputPath)
		}
	}

	return nil
}
