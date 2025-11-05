package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"github.com/DevSymphony/sym-cli/internal/config"
	"github.com/DevSymphony/sym-cli/internal/github"

	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Display the current authenticated user",
	Long: `Show information about the currently authenticated GitHub user.

Output can be formatted as JSON using --json flag for scripting purposes.`,
	Run: runWhoami,
}

var whoamiJSON bool

func init() {
	whoamiCmd.Flags().BoolVar(&whoamiJSON, "json", false, "Output in JSON format")
}

func runWhoami(cmd *cobra.Command, args []string) {
	// Check if logged in
	if !config.IsLoggedIn() {
		if whoamiJSON {
			output := map[string]string{"error": "not logged in"}
			_ = json.NewEncoder(os.Stdout).Encode(output)
		} else {
			fmt.Println("❌ Not logged in")
			fmt.Println("Run 'sym login' first")
		}
		os.Exit(1)
	}

	// Get current user
	cfg, err := config.LoadConfig()
	if err != nil {
		handleWhoamiError("Failed to load config", err, whoamiJSON)
		os.Exit(1)
	}

	token, err := config.LoadToken()
	if err != nil {
		handleWhoamiError("Failed to load token", err, whoamiJSON)
		os.Exit(1)
	}

	client := github.NewClient(cfg.GetGitHubHost(), token.AccessToken)
	user, err := client.GetCurrentUser()
	if err != nil {
		handleWhoamiError("Failed to get current user", err, whoamiJSON)
		os.Exit(1)
	}

	if whoamiJSON {
		output := map[string]interface{}{
			"username": user.Login,
			"name":     user.Name,
			"email":    user.Email,
			"id":       user.ID,
		}
		_ = json.NewEncoder(os.Stdout).Encode(output)
	} else {
		fmt.Printf("Username: %s\n", user.Login)
		if user.Name != "" {
			fmt.Printf("Name: %s\n", user.Name)
		}
		if user.Email != "" {
			fmt.Printf("Email: %s\n", user.Email)
		}
		fmt.Printf("GitHub ID: %d\n", user.ID)
		fmt.Printf("Host: %s\n", cfg.GetGitHubHost())
	}
}

func handleWhoamiError(msg string, err error, jsonMode bool) {
	if jsonMode {
		output := map[string]string{"error": fmt.Sprintf("%s: %v", msg, err)}
		_ = json.NewEncoder(os.Stdout).Encode(output)
	} else {
		fmt.Printf("❌ %s: %v\n", msg, err)
	}
}
