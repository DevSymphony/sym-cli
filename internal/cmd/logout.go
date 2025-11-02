package cmd

import (
	"fmt"
	"os"
	"github.com/DevSymphony/sym-cli/internal/config"

	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and remove stored credentials",
	Long:  `Remove the stored access token and log out of Symphony.`,
	Run:   runLogout,
}

func runLogout(cmd *cobra.Command, args []string) {
	if !config.IsLoggedIn() {
		fmt.Println("⚠ Not logged in")
		os.Exit(0)
	}

	if err := config.DeleteToken(); err != nil {
		fmt.Printf("❌ Failed to log out: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Successfully logged out")
}
