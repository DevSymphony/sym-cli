package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/DevSymphony/sym-cli/internal/config"
	"github.com/DevSymphony/sym-cli/internal/linter"
	"github.com/DevSymphony/sym-cli/internal/policy"
	"github.com/DevSymphony/sym-cli/internal/roles"
	"github.com/DevSymphony/sym-cli/internal/ui"
	"github.com/DevSymphony/sym-cli/pkg/schema"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Symphony for the current directory",
	Long: `Create a .sym directory with roles.json and user-policy.json files.

This command:
  1. Creates .sym/roles.json with default roles (admin, developer, viewer)
  2. Creates .sym/user-policy.json with default RBAC configuration
  3. Sets your role to admin (can be changed later via dashboard)
  4. Optionally registers MCP server for AI tools`,
	Run: runInit,
}

var (
	initForce       bool
	skipMCPRegister bool
	registerMCPOnly bool
	skipLLMSetup    bool
	setupLLMOnly    bool
)

func init() {
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Overwrite existing roles.json")
	initCmd.Flags().BoolVar(&skipMCPRegister, "skip-mcp", false, "Skip MCP server registration prompt")
	initCmd.Flags().BoolVar(&registerMCPOnly, "register-mcp", false, "Register MCP server only (skip roles/policy init)")
	initCmd.Flags().BoolVar(&skipLLMSetup, "skip-llm", false, "Skip LLM backend configuration prompt")
	initCmd.Flags().BoolVar(&setupLLMOnly, "setup-llm", false, "Setup LLM backend only (skip roles/policy init)")
}

func runInit(cmd *cobra.Command, args []string) {
	// MCP registration only mode
	if registerMCPOnly {
		ui.PrintTitle("MCP", "Registering Symphony MCP server")
		promptMCPRegistration()
		return
	}

	// LLM setup only mode
	if setupLLMOnly {
		ui.PrintTitle("LLM", "Setting up LLM backend")
		promptLLMBackendSetup()
		return
	}

	// Check if roles.json already exists
	exists, err := roles.RolesExists()
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to check roles.json: %v", err))
		os.Exit(1)
	}

	if exists && !initForce {
		ui.PrintWarn("roles.json already exists")
		fmt.Println("Use --force flag to overwrite")
		os.Exit(1)
	}

	// If force flag is set, remove existing code-policy.json
	if initForce {
		if err := removeExistingCodePolicy(); err != nil {
			ui.PrintWarn(fmt.Sprintf("Failed to remove existing code-policy.json: %v", err))
		}
	}

	// Create default roles (empty user lists - users select their own role)
	newRoles := roles.Roles{
		"admin":     []string{},
		"developer": []string{},
		"viewer":    []string{},
	}

	if err := roles.SaveRoles(newRoles); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to create roles.json: %v", err))
		os.Exit(1)
	}

	rolesPath, _ := roles.GetRolesPath()
	ui.PrintOK("roles.json created")
	fmt.Println(ui.Indent(fmt.Sprintf("Location: %s", rolesPath)))

	// Create default policy file with RBAC roles
	if err := createDefaultPolicy(); err != nil {
		ui.PrintWarn(fmt.Sprintf("Failed to create policy file: %v", err))
		fmt.Println(ui.Indent("You can manually create it later using the dashboard"))
	} else {
		ui.PrintOK("user-policy.json created with default RBAC roles")
	}

	// Create .sym/config.json with default settings
	if err := initializeConfigFile(); err != nil {
		ui.PrintWarn(fmt.Sprintf("Failed to create config.json: %v", err))
	} else {
		ui.PrintOK("config.json created")
	}

	// Set default role to admin during initialization
	if err := roles.SetCurrentRole("admin"); err != nil {
		ui.PrintWarn(fmt.Sprintf("Failed to save role selection: %v", err))
	} else {
		ui.PrintOK("Your role has been set to: admin")
	}

	// MCP registration prompt
	if !skipMCPRegister {
		promptMCPRegistration()
	}

	// LLM backend configuration prompt
	if !skipLLMSetup {
		promptLLMBackendSetup()
	}

	// Show completion message
	fmt.Println()
	ui.PrintDone("Initialization complete")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println(ui.Indent("Run 'sym dashboard' to manage roles and policies"))
	fmt.Println(ui.Indent("Commit .sym/ folder to share with your team"))
}

// createDefaultPolicy creates a default policy file with RBAC roles
func createDefaultPolicy() error {
	defaultPolicyPath := ".sym/user-policy.json"

	// Check if policy file already exists
	exists, err := policy.PolicyExists(defaultPolicyPath)
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

	return policy.SavePolicy(defaultPolicy, defaultPolicyPath)
}

// initializeConfigFile creates .sym/config.json with default settings
func initializeConfigFile() error {
	// Check if config.json already exists
	if config.ProjectConfigExists() {
		return nil
	}

	// Create default project config
	defaultConfig := &config.ProjectConfig{
		PolicyPath: ".sym/user-policy.json",
	}

	return config.SaveProjectConfig(defaultConfig)
}

// removeExistingCodePolicy removes generated linter config files when --force flag is used
func removeExistingCodePolicy() error {
	// Get list of generated files from registry
	convertGeneratedFiles := []string{"code-policy.json"}
	convertGeneratedFiles = append(convertGeneratedFiles, linter.Global().GetAllConfigFiles()...)

	// Check and remove from .sym directory
	symDir, err := getSymDir()
	if err == nil {
		for _, filename := range convertGeneratedFiles {
			filePath := filepath.Join(symDir, filename)
			if _, err := os.Stat(filePath); err == nil {
				if err := os.Remove(filePath); err != nil {
					ui.PrintWarn(fmt.Sprintf("Failed to remove %s: %v", filePath, err))
				} else {
					fmt.Println(ui.Indent(fmt.Sprintf("Removed existing %s", filePath)))
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
		fmt.Println(ui.Indent(fmt.Sprintf("Removed existing %s", legacyPath)))
	}

	return nil
}
