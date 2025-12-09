package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/DevSymphony/sym-cli/internal/mcp"
	"github.com/DevSymphony/sym-cli/internal/util/git"
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
	if _, err := os.Stat(userPolicyPath); os.IsNotExist(err) {
		return fmt.Errorf("user policy not found: %s\nRun 'sym init' first or 'sym dashboard' to create policy", userPolicyPath)
	}

	// Start MCP server - it will handle conversion automatically if needed
	server := mcp.NewServer(configPath)
	return server.Start()
}
