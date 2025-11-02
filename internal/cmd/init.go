package cmd

import (
	"fmt"
	"os"
	"github.com/DevSymphony/sym-cli/internal/config"
	"github.com/DevSymphony/sym-cli/internal/git"
	"github.com/DevSymphony/sym-cli/internal/github"
	"github.com/DevSymphony/sym-cli/internal/policy"
	"github.com/DevSymphony/sym-cli/internal/roles"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize roles.json for the current repository",
	Long: `Create a .github/roles.json file in the current repository and
automatically set the current user as the first admin.

This command:
  1. Checks if you're logged in
  2. Gets your GitHub username
  3. Verifies the current directory is a git repository
  4. Creates .github/roles.json with you as admin
  5. Prompts you to commit and push the changes`,
	Run: runInit,
}

var initForce bool

func init() {
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Overwrite existing roles.json")
}

func runInit(cmd *cobra.Command, args []string) {
	// Check if logged in
	if !config.IsLoggedIn() {
		fmt.Println("❌ Not logged in")
		fmt.Println("Run 'sym login' first")
		os.Exit(1)
	}

	// Check if in git repository
	if !git.IsGitRepo() {
		fmt.Println("❌ Not a git repository")
		fmt.Println("Navigate to a git repository before running this command")
		os.Exit(1)
	}

	// Get current user
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("❌ Failed to load config: %v\n", err)
		os.Exit(1)
	}

	token, err := config.LoadToken()
	if err != nil {
		fmt.Printf("❌ Failed to load token: %v\n", err)
		os.Exit(1)
	}

	client := github.NewClient(cfg.GetGitHubHost(), token.AccessToken)
	user, err := client.GetCurrentUser()
	if err != nil {
		fmt.Printf("❌ Failed to get current user: %v\n", err)
		os.Exit(1)
	}

	// Check if roles.json already exists
	exists, err := roles.RolesExists()
	if err != nil {
		fmt.Printf("❌ Failed to check roles.json: %v\n", err)
		os.Exit(1)
	}

	if exists && !initForce {
		fmt.Println("⚠ roles.json already exists")
		fmt.Println("Use --force flag to overwrite")
		os.Exit(1)
	}

	// Create roles with current user as admin
	newRoles := roles.Roles{
		"admin":     []string{user.Login},
		"developer": []string{},
		"viewer":    []string{},
	}

	if err := roles.SaveRoles(newRoles); err != nil {
		fmt.Printf("❌ Failed to create roles.json: %v\n", err)
		os.Exit(1)
	}

	rolesPath, _ := roles.GetRolesPath()
	fmt.Println("✓ roles.json created successfully!")
	fmt.Printf("  Location: %s\n", rolesPath)
	fmt.Printf("  You (%s) have been set as admin\n", user.Login)

	// Create default policy file with RBAC roles
	fmt.Println("\nCreating default policy file...")
	if err := createDefaultPolicy(cfg); err != nil {
		fmt.Printf("⚠ Warning: Failed to create policy file: %v\n", err)
		fmt.Println("You can manually create it later using the dashboard")
	} else {
		fmt.Println("✓ user-policy.json created with default RBAC roles")
	}

	fmt.Println("\nNext steps:")
	fmt.Println("  1. Review the files:")
	fmt.Println("     cat .github/roles.json")
	fmt.Println("     cat .github/user-policy.json")
	fmt.Println("  2. Commit: git add .github/ && git commit -m 'Initialize Symphony roles and policy'")
	fmt.Println("  3. Push: git push")
	fmt.Println("\nAfter pushing, team members can clone and use 'sym my-role' to check their access.")
}

// createDefaultPolicy creates a default policy file with RBAC roles
func createDefaultPolicy(cfg *config.Config) error {
	// Check if policy file already exists
	exists, err := policy.PolicyExists(cfg.PolicyPath)
	if err != nil {
		return err
	}

	if exists && !initForce {
		// Policy already exists, skip creation
		return nil
	}

	// Create default policy with admin, developer, viewer RBAC roles
	defaultPolicy := &policy.Policy{
		Version: "1.0.0",
		RBAC: &policy.RBAC{
			Roles: map[string]policy.RBACRole{
				"admin": {
					AllowWrite:    []string{"**/*"},
					DenyWrite:     []string{},
					CanEditPolicy: true,
					CanEditRoles:  true,
				},
				"developer": {
					AllowWrite:    []string{"src/**", "tests/**", "docs/**"},
					DenyWrite:     []string{".github/**", "config/**", "*.config.js", "*.config.ts"},
					CanEditPolicy: false,
					CanEditRoles:  false,
				},
				"viewer": {
					AllowWrite:    []string{},
					DenyWrite:     []string{"**/*"},
					CanEditPolicy: false,
					CanEditRoles:  false,
				},
			},
		},
		Defaults: &policy.Defaults{
			Languages: []string{"javascript", "typescript"},
			Severity:  "error",
			Autofix:   true,
		},
		Rules: []policy.Rule{},
	}

	return policy.SavePolicy(defaultPolicy, cfg.PolicyPath)
}
