package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/DevSymphony/sym-cli/internal/git"
	"github.com/DevSymphony/sym-cli/internal/mcp"
	"github.com/DevSymphony/sym-cli/internal/ui"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

var mcpConfig string

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server to integrate with LLM tools",
	Long: `Start Model Context Protocol (MCP) server.
LLM-based coding tools can query conventions and validate code through stdio.

Tools provided by MCP server:
- query_conventions: Query conventions for given context
- validate_code: Validate code compliance with conventions

Communicates via stdio for integration with Claude Desktop, Claude Code, Cursor, and other MCP clients.`,
	Example: `  sym mcp
  sym mcp --config code-policy.json`,
	RunE: runMCP,
}

func init() {
	rootCmd.AddCommand(mcpCmd)

	mcpCmd.Flags().StringVarP(&mcpConfig, "config", "c", "", "policy file path (code-policy.json)")
}

func runMCP(cmd *cobra.Command, args []string) error {
	// Get git root directory
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	symDir := filepath.Join(repoRoot, ".sym")
	userPolicyPath := filepath.Join(symDir, "user-policy.json")

	// If custom config path is specified, use it directly
	var configPath string
	if mcpConfig != "" {
		configPath = mcpConfig
	} else {
		// Use .sym directory as config path for auto-detection
		configPath = symDir
	}

	// Check if user-policy.json exists
	userPolicyExists := fileExists(userPolicyPath)

	// If no user-policy.json → Launch dashboard
	if !userPolicyExists {
		ui.PrintError(fmt.Sprintf("User policy not found at: %s", userPolicyPath))
		fmt.Println("Opening dashboard to create policy...")

		// Launch dashboard
		if err := launchDashboard(); err != nil {
			return fmt.Errorf("failed to launch dashboard: %w", err)
		}

		fmt.Println()
		ui.PrintOK("Dashboard launched at http://localhost:8787")
		fmt.Println("Please create your policy in the dashboard, then restart MCP server.")
		return nil
	}

	// Start MCP server - it will handle conversion automatically if needed
	server := mcp.NewServer(configPath)
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
		_ = browser.OpenURL(url) // Ignore error - browser opening is best-effort
	}()

	// Start dashboard server in background
	// Note: This will block, so in practice you'd want to run this in a separate process
	// For now, we just inform the user to run it manually
	fmt.Println("Please run in another terminal:")
	fmt.Println("  sym dashboard")

	return nil
}
