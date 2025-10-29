package cmd

import (
	"github.com/DevSymphony/sym-cli/internal/mcp"
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
	server := mcp.NewServer(mcpHost, mcpPort, mcpConfig)
	return server.Start()
}
