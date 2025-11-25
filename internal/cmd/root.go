package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// verbose is a global flag for verbose output
// Used by convert and validate commands
var verbose bool

// symphonyclient integration: Updated root command from symphony to sym
var rootCmd = &cobra.Command{
	Use:   "sym",
	Short: "sym - Code Convention Management Tool with RBAC",
	Long: `sym is a unified CLI tool for code convention validation and role-based access control.

Features:
  - Natural language policy definition (A → B schema conversion)
  - Multi-engine code validation (Pattern, Length, Style, AST)
  - Role-based file access control with GitHub OAuth
  - Web dashboard for policy and role management
  - Template system for popular frameworks`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")

	// symphonyclient integration: Added symphonyclient commands
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(dashboardCmd)
	rootCmd.AddCommand(myRoleCmd)
	rootCmd.AddCommand(whoamiCmd)
	rootCmd.AddCommand(policyCmd)
	// Note: mcpCmd is registered in mcp.go's init()

	// sym-cli core commands
	rootCmd.AddCommand(convertCmd)
	rootCmd.AddCommand(validateCmd)
}
