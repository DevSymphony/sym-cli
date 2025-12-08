package cmd

import (
	"fmt"
	"os"

	"github.com/DevSymphony/sym-cli/internal/roles"
	"github.com/DevSymphony/sym-cli/internal/server"

	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:     "dashboard",
	Aliases: []string{"dash"},
	Short:   "Start the web dashboard",
	Long: `Start a local web server to manage roles and policies through a browser interface.

The dashboard provides a visual interface for:
  - Selecting your role
  - Managing role permissions
  - Editing coding policies and rules`,
	Run: runDashboard,
}

var dashboardPort int

func init() {
	dashboardCmd.Flags().IntVarP(&dashboardPort, "port", "p", 8787, "Port to run the dashboard on")
}

func runDashboard(cmd *cobra.Command, args []string) {
	// Check if roles.json exists
	exists, err := roles.RolesExists()
	if err != nil || !exists {
		fmt.Println("❌ roles.json not found")
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
