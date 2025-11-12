package cmd

import (
	"fmt"
	"os"
	"github.com/DevSymphony/sym-cli/internal/config"
	"github.com/DevSymphony/sym-cli/internal/git"
	"github.com/DevSymphony/sym-cli/internal/roles"
	"github.com/DevSymphony/sym-cli/internal/server"

	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:     "dashboard",
	Aliases: []string{"dash"},
	Short:   "Start the web dashboard",
	Long: `Start a local web server to manage roles through a browser interface.

The dashboard provides a visual interface for:
  - Viewing all users and their roles
  - Adding/removing users from roles (admin only)
  - Viewing current repository information`,
	Run: runDashboard,
}

var dashboardPort int

func init() {
	// symphonyclient integration: default port 3000 → 8787
	dashboardCmd.Flags().IntVarP(&dashboardPort, "port", "p", 8787, "Port to run the dashboard on")
}

func runDashboard(cmd *cobra.Command, args []string) {
	// Check if logged in
	if !config.IsLoggedIn() {
		fmt.Println("❌ Not logged in")
		// symphonyclient integration: symphony → sym command
		fmt.Println("Run 'sym login' first")
		os.Exit(1)
	}

	// Check if in git repository
	if !git.IsGitRepo() {
		fmt.Println("❌ Not a git repository")
		fmt.Println("Navigate to a git repository before running this command")
		os.Exit(1)
	}

	// Check if roles.json exists
	exists, err := roles.RolesExists()
	if err != nil || !exists {
		fmt.Println("❌ roles.json not found")
		// symphonyclient integration: symphony → sym command
		fmt.Println("Run 'sym init' to create it")
		os.Exit(1)
	}

	// Start server
	srv, err := server.NewServer(dashboardPort)
	if err != nil {
		fmt.Printf("❌ Failed to create server: %v\n", err)
		os.Exit(1)
	}

	if err := srv.Start(); err != nil {
		fmt.Printf("❌ Failed to start server: %v\n", err)
		os.Exit(1)
	}
}
