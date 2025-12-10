package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/DevSymphony/sym-cli/internal/linter"
	"github.com/DevSymphony/sym-cli/internal/policy"
	"github.com/DevSymphony/sym-cli/internal/roles"
	"github.com/DevSymphony/sym-cli/internal/util/config"
	"github.com/DevSymphony/sym-cli/internal/util/git"
	"github.com/DevSymphony/sym-cli/pkg/schema"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Symphony for the current directory",
	Long: `Initialize Symphony for the current project.

This command:
  1. Creates .sym/roles.json with default roles (admin, developer, viewer)
  2. Creates .sym/user-policy.json with default RBAC configuration
  3. Creates .sym/config.json with default settings
  4. Sets your default role (admin for new projects, non-privileged for existing)
  5. Optionally registers MCP server for AI tools
  6. Optionally configures LLM backend

If .sym/ already exists (e.g., synced via git), existing configuration files
are preserved and only missing files are created. This allows team members
to share roles and policies through version control.

Use --force to reinitialize and overwrite existing configuration.`,
	Run: runInit,
}

var (
	initForce       bool
	skipMCPRegister bool
	skipLLMSetup    bool
)

func init() {
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Overwrite existing Symphony configuration")
	initCmd.Flags().BoolVar(&skipMCPRegister, "skip-mcp", false, "Skip MCP server registration prompt")
	initCmd.Flags().BoolVar(&skipLLMSetup, "skip-llm", false, "Skip LLM backend configuration prompt")
}

func runInit(cmd *cobra.Command, args []string) {
	// Check if .sym directory already exists
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		printError(fmt.Sprintf("Failed to find git repository: %v", err))
		os.Exit(1)
	}
	symDir := filepath.Join(repoRoot, ".sym")

	symDirExists := false
	if _, err := os.Stat(symDir); err == nil {
		symDirExists = true
	} else if !os.IsNotExist(err) {
		printError(fmt.Sprintf("Failed to check .sym directory: %v", err))
		os.Exit(1)
	}

	// When .sym exists without --force, preserve existing files and only create missing ones
	if symDirExists && !initForce {
		fmt.Println("Found existing .sym directory, preserving existing configuration files...")
	}

	// Create roles.json only if it doesn't exist or --force is set
	rolesExist, err := roles.RolesExists()
	if err != nil {
		printError(fmt.Sprintf("Failed to check roles.json: %v", err))
		os.Exit(1)
	}

	rolesPath, _ := roles.GetRolesPath()
	if rolesExist && !initForce {
		printOK("roles.json already exists (preserved)")
		fmt.Println(indent(fmt.Sprintf("Location: %s", rolesPath)))
	} else {
		// Create default roles (empty user lists - users select their own role)
		newRoles := roles.Roles{
			"admin":     []string{},
			"developer": []string{},
			"viewer":    []string{},
		}

		if err := roles.SaveRoles(newRoles); err != nil {
			printError(fmt.Sprintf("Failed to create roles.json: %v", err))
			os.Exit(1)
		}

		printOK("roles.json created")
		fmt.Println(indent(fmt.Sprintf("Location: %s", rolesPath)))
	}

	// Create default policy file with RBAC roles (only if not exists or --force)
	policyCreated, err := createDefaultPolicy()
	if err != nil {
		printWarn(fmt.Sprintf("Failed to create policy file: %v", err))
		fmt.Println(indent("You can manually create it later using the dashboard"))
	} else if policyCreated {
		printOK("user-policy.json created with default RBAC roles")
	} else {
		printOK("user-policy.json already exists (preserved)")
	}

	// Create .sym/config.json with default settings (only if not exists or --force)
	configCreated, err := initializeConfigFile()
	if err != nil {
		printWarn(fmt.Sprintf("Failed to create config.json: %v", err))
	} else if configCreated {
		printOK("config.json created")
	} else {
		printOK("config.json already exists (preserved)")
	}

	// Select appropriate default role based on existing configuration
	// Note: role selection is stored in .sym/.env which is gitignored, so each user sets their own role
	defaultRole := selectDefaultRole(policyCreated)
	if err := roles.SetCurrentRole(defaultRole); err != nil {
		printWarn(fmt.Sprintf("Failed to save role selection: %v", err))
	} else {
		printOK(fmt.Sprintf("Your role has been set to: %s", defaultRole))
	}

	// MCP registration prompt
	if !skipMCPRegister {
		promptMCPRegistration()
	}

	// LLM backend configuration prompt
	if !skipLLMSetup {
		promptLLMBackendSetup()
	}

	// Clean up generated files at the end (only when --force is set)
	if initForce {
		if err := removeExistingCodePolicy(); err != nil {
			printWarn(fmt.Sprintf("Failed to remove generated files: %v", err))
		}
	}

	// Show completion message
	fmt.Println()
	printDone("Initialization complete")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println(indent("Run 'sym dashboard' to manage roles and policies"))
	fmt.Println(indent("Commit .sym/ folder to share with your team"))
}

// createDefaultPolicy creates a default policy file with RBAC roles.
// Returns (true, nil) if a new file was created, (false, nil) if existing file was preserved.
func createDefaultPolicy() (bool, error) {
	defaultPolicyPath := ".sym/user-policy.json"

	// Check if policy file already exists
	exists, err := policy.PolicyExists(defaultPolicyPath)
	if err != nil {
		return false, err
	}

	if exists && !initForce {
		// Policy already exists, preserve it
		return false, nil
	}

	// Create default policy with categories and RBAC roles
	defaultPolicy := &schema.UserPolicy{
		Version: "1.0.0",
		Category: []schema.CategoryDef{
			{Name: "security", Description: "Security rules (authentication, authorization, vulnerability prevention, etc.)"},
			{Name: "style", Description: "Code style and formatting rules"},
			{Name: "documentation", Description: "Documentation rules (comments, docstrings, etc.)"},
			{Name: "error_handling", Description: "Error handling and exception management rules"},
			{Name: "architecture", Description: "Code structure and architecture rules"},
			{Name: "performance", Description: "Performance optimization rules"},
			{Name: "testing", Description: "Testing rules (coverage, test patterns, etc.)"},
		},
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

	if err := policy.SavePolicy(defaultPolicy, defaultPolicyPath); err != nil {
		return false, err
	}
	return true, nil
}

// initializeConfigFile creates .sym/config.json with default settings.
// Returns (true, nil) if a new file was created, (false, nil) if existing file was preserved.
func initializeConfigFile() (bool, error) {
	// Check if config.json already exists (skip unless force is set)
	if config.ProjectConfigExists() && !initForce {
		return false, nil
	}

	// Create default project config
	defaultConfig := &config.ProjectConfig{
		PolicyPath: ".sym/user-policy.json",
	}

	if err := config.SaveProjectConfig(defaultConfig); err != nil {
		return false, err
	}
	return true, nil
}

// selectDefaultRole determines the appropriate default role for a new user.
// For existing projects, selects a role without policy/role editing permissions.
// For new projects, returns the first available role (typically "admin").
func selectDefaultRole(isNewProject bool) string {
	// For new projects, use admin role
	if isNewProject {
		return "admin"
	}

	// Try to load existing policy to find a non-privileged role
	loader := policy.NewLoader(false)
	userPolicy, err := loader.LoadUserPolicy(".sym/user-policy.json")
	if err != nil || userPolicy == nil || userPolicy.RBAC == nil || len(userPolicy.RBAC.Roles) == 0 {
		// Fallback: try to get first available role from roles.json
		availableRoles, err := roles.GetAvailableRoles()
		if err != nil || len(availableRoles) == 0 {
			return "admin" // Ultimate fallback
		}
		return availableRoles[0]
	}

	// Find a role without editing permissions (non-privileged)
	var firstRole string
	for roleName, role := range userPolicy.RBAC.Roles {
		if firstRole == "" {
			firstRole = roleName
		}
		if !role.CanEditPolicy && !role.CanEditRoles {
			return roleName
		}
	}

	// No non-privileged role found, return first available role
	if firstRole != "" {
		return firstRole
	}
	return "admin"
}

// removeExistingCodePolicy removes generated linter config files when --force flag is used
func removeExistingCodePolicy() error {
	// Get list of generated files from registry
	convertGeneratedFiles := []string{"code-policy.json"}
	convertGeneratedFiles = append(convertGeneratedFiles, linter.Global().GetAllConfigFiles()...)

	// Check and remove from .sym directory
	repoRoot, err := git.GetRepoRoot()
	if err == nil {
		symDir := filepath.Join(repoRoot, ".sym")
		for _, filename := range convertGeneratedFiles {
			filePath := filepath.Join(symDir, filename)
			if _, err := os.Stat(filePath); err == nil {
				if err := os.Remove(filePath); err != nil {
					printWarn(fmt.Sprintf("Failed to remove %s: %v", filePath, err))
				} else {
					fmt.Println(indent(fmt.Sprintf("Removed existing %s", filePath)))
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
		fmt.Println(indent(fmt.Sprintf("Removed existing %s", legacyPath)))
	}

	return nil
}
