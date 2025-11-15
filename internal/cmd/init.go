package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"github.com/DevSymphony/sym-cli/internal/config"
	"github.com/DevSymphony/sym-cli/internal/envutil"
	"github.com/DevSymphony/sym-cli/internal/git"
	"github.com/DevSymphony/sym-cli/internal/github"
	"github.com/DevSymphony/sym-cli/internal/policy"
	"github.com/DevSymphony/sym-cli/internal/roles"
	"github.com/DevSymphony/sym-cli/pkg/schema" // symphonyclient integration

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize roles.json for the current repository",
	Long: `Create a .sym/roles.json file in the current repository and
automatically set the current user as the first admin.

This command:
  1. Checks if you're logged in
  2. Gets your GitHub username
  3. Verifies the current directory is a git repository
  4. Creates .sym/roles.json with you as admin
  5. Prompts you to commit and push the changes`,
	Run: runInit,
}

var (
	initForce        bool
	skipMCPRegister  bool
	registerMCPOnly  bool
	skipAPIKey       bool
	setupAPIKeyOnly  bool
)

func init() {
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Overwrite existing roles.json")
	initCmd.Flags().BoolVar(&skipMCPRegister, "skip-mcp", false, "Skip MCP server registration prompt")
	initCmd.Flags().BoolVar(&registerMCPOnly, "register-mcp", false, "Register MCP server only (skip roles/policy init)")
	initCmd.Flags().BoolVar(&skipAPIKey, "skip-api-key", false, "Skip OpenAI API key configuration prompt")
	initCmd.Flags().BoolVar(&setupAPIKeyOnly, "setup-api-key", false, "Setup OpenAI API key only (skip roles/policy init)")
}

func runInit(cmd *cobra.Command, args []string) {
	// MCP registration only mode
	if registerMCPOnly {
		fmt.Println("🔧 Registering Symphony MCP server...")
		promptMCPRegistration()
		return
	}

	// API key setup only mode
	if setupAPIKeyOnly {
		fmt.Println("🔑 Setting up OpenAI API key...")
		promptAPIKeySetup()
		return
	}

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

	// If force flag is set, remove existing code-policy.json
	if initForce {
		if err := removeExistingCodePolicy(); err != nil {
			fmt.Printf("⚠ Warning: Failed to remove existing code-policy.json: %v\n", err)
		}
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

	// Create .sym/.env with default POLICY_PATH
	fmt.Println("\nSetting up environment configuration...")
	if err := initializeEnvFile(); err != nil {
		fmt.Printf("⚠ Warning: Failed to create .sym/.env: %v\n", err)
	} else {
		fmt.Println("✓ .sym/.env created with default policy path")
	}

	fmt.Println("\nNext steps:")
	fmt.Println("  1. Review the files:")
	fmt.Println("     cat .sym/roles.json")
	fmt.Println("     cat .sym/user-policy.json")
	fmt.Println("  2. Commit: git add .sym/ && git commit -m 'Initialize Symphony roles and policy'")
	fmt.Println("  3. Push: git push")
	fmt.Println("\nAfter pushing, team members can clone and use 'sym my-role' to check their access.")

	// MCP registration prompt
	if !skipMCPRegister {
		promptMCPRegistration()
	}

	// API key configuration prompt
	if !skipAPIKey {
		promptAPIKeyIfNeeded()
	}
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
	defaultPolicy := &schema.UserPolicy{
		Version: "1.0.0",
		RBAC: &schema.UserRBAC{
			Roles: map[string]schema.UserRole{
				"admin": {
					AllowWrite:    []string{"**/*"},
					DenyWrite:     []string{},
					CanEditPolicy: true,
					CanEditRoles:  true,
				},
				"developer": {
					AllowWrite:    []string{"src/**", "tests/**", "docs/**"},
					DenyWrite:     []string{".sym/**", "config/**", "*.config.js", "*.config.ts"},
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
		Defaults: &schema.UserDefaults{
			Languages: []string{"javascript", "typescript"},
			Severity:  "error",
			Autofix:   true,
		},
		Rules: []schema.UserRule{},
	}

	return policy.SavePolicy(defaultPolicy, cfg.PolicyPath)
}

// removeExistingCodePolicy removes all files generated by convert command
// including linter configurations and code-policy.json
// initializeEnvFile creates .sym/.env with default POLICY_PATH if not exists
func initializeEnvFile() error {
	envPath := filepath.Join(".sym", ".env")
	defaultPolicyPath := ".sym/user-policy.json"

	// Check if .env already exists
	if _, err := os.Stat(envPath); err == nil {
		// File exists, check if POLICY_PATH is already set
		existingPath := envutil.LoadKeyFromEnvFile(envPath, "POLICY_PATH")
		if existingPath != "" {
			// POLICY_PATH already set, nothing to do
			return nil
		}
		// POLICY_PATH not set, add it
		return envutil.SaveKeyToEnvFile(envPath, "POLICY_PATH", defaultPolicyPath)
	}

	// .env doesn't exist, create it with default POLICY_PATH
	content := fmt.Sprintf("# Policy configuration\nPOLICY_PATH=%s\n", defaultPolicyPath)
	return os.WriteFile(envPath, []byte(content), 0644)
}

func removeExistingCodePolicy() error {
	// Files generated by convert command
	convertGeneratedFiles := []string{
		"code-policy.json",
		".eslintrc.json",
		"checkstyle.xml",
		"pmd-ruleset.xml",
	}

	// Check and remove from .sym directory
	symDir, err := getSymDir()
	if err == nil {
		for _, filename := range convertGeneratedFiles {
			filePath := filepath.Join(symDir, filename)
			if _, err := os.Stat(filePath); err == nil {
				if err := os.Remove(filePath); err != nil {
					fmt.Printf("⚠ Warning: Failed to remove %s: %v\n", filePath, err)
				} else {
					fmt.Printf("✓ Removed existing %s\n", filePath)
				}
			}
		}
	}

	// Check and remove from current directory (legacy mode for code-policy.json)
	legacyPath := "code-policy.json"
	if _, err := os.Stat(legacyPath); err == nil {
		if err := os.Remove(legacyPath); err != nil {
			return fmt.Errorf("failed to remove %s: %w", legacyPath, err)
		}
		fmt.Printf("✓ Removed existing %s\n", legacyPath)
	}

	return nil
}
