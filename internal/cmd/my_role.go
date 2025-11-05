package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"github.com/DevSymphony/sym-cli/internal/config"
	"github.com/DevSymphony/sym-cli/internal/git"
	"github.com/DevSymphony/sym-cli/internal/github"
	"github.com/DevSymphony/sym-cli/internal/roles"

	"github.com/spf13/cobra"
)

var myRoleCmd = &cobra.Command{
	Use:   "my-role",
	Short: "Check your role in the current repository",
	Long: `Display your role in the current repository based on roles.json.

Output can be formatted as JSON using --json flag for scripting purposes.`,
	Run: runMyRole,
}

var myRoleJSON bool

func init() {
	myRoleCmd.Flags().BoolVar(&myRoleJSON, "json", false, "Output in JSON format")
}

func runMyRole(cmd *cobra.Command, args []string) {
	// Check if logged in
	if !config.IsLoggedIn() {
		if myRoleJSON {
			output := map[string]string{"error": "not logged in"}
			_ = json.NewEncoder(os.Stdout).Encode(output)
		} else {
			fmt.Println("❌ Not logged in")
			fmt.Println("Run 'sym login' first")
		}
		os.Exit(1)
	}

	// Check if in git repository
	if !git.IsGitRepo() {
		if myRoleJSON {
			output := map[string]string{"error": "not a git repository"}
			_ = json.NewEncoder(os.Stdout).Encode(output)
		} else {
			fmt.Println("❌ Not a git repository")
			fmt.Println("Navigate to a git repository before running this command")
		}
		os.Exit(1)
	}

	// Get current user
	cfg, err := config.LoadConfig()
	if err != nil {
		handleError("Failed to load config", err, myRoleJSON)
		os.Exit(1)
	}

	token, err := config.LoadToken()
	if err != nil {
		handleError("Failed to load token", err, myRoleJSON)
		os.Exit(1)
	}

	client := github.NewClient(cfg.GetGitHubHost(), token.AccessToken)
	user, err := client.GetCurrentUser()
	if err != nil {
		handleError("Failed to get current user", err, myRoleJSON)
		os.Exit(1)
	}

	// Get user role
	role, err := roles.GetUserRole(user.Login)
	if err != nil {
		handleError("Failed to get role", err, myRoleJSON)
		os.Exit(1)
	}

	// Get repo info
	owner, repo, err := git.GetRepoInfo()
	if err != nil {
		handleError("Failed to get repository info", err, myRoleJSON)
		os.Exit(1)
	}

	if myRoleJSON {
		output := map[string]string{
			"username": user.Login,
			"role":     role,
			"owner":    owner,
			"repo":     repo,
		}
		_ = json.NewEncoder(os.Stdout).Encode(output)
	} else {
		fmt.Printf("Repository: %s/%s\n", owner, repo)
		fmt.Printf("User: %s\n", user.Login)
		fmt.Printf("Role: %s\n", role)

		if role == "none" {
			fmt.Println("\n⚠ You don't have any role assigned in this repository")
			fmt.Println("Contact an admin to get access")
		}
	}
}

func handleError(msg string, err error, jsonMode bool) {
	if jsonMode {
		output := map[string]string{"error": fmt.Sprintf("%s: %v", msg, err)}
		_ = json.NewEncoder(os.Stdout).Encode(output)
	} else {
		fmt.Printf("❌ %s: %v\n", msg, err)
	}
}
