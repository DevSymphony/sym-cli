package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/DevSymphony/sym-cli/internal/adapter/registry"
	"github.com/DevSymphony/sym-cli/internal/envutil"
	"github.com/DevSymphony/sym-cli/internal/policy"
	"github.com/DevSymphony/sym-cli/internal/roles"
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
	skipAPIKey      bool
	setupAPIKeyOnly bool
	skipLLMSetup    bool
	setupLLMOnly    bool
)

func init() {
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Overwrite existing roles.json")
	initCmd.Flags().BoolVar(&skipMCPRegister, "skip-mcp", false, "Skip MCP server registration prompt")
	initCmd.Flags().BoolVar(&registerMCPOnly, "register-mcp", false, "Register MCP server only (skip roles/policy init)")
	initCmd.Flags().BoolVar(&skipAPIKey, "skip-api-key", false, "Skip OpenAI API key configuration prompt (deprecated, use --skip-llm)")
	initCmd.Flags().BoolVar(&setupAPIKeyOnly, "setup-api-key", false, "Setup OpenAI API key only (deprecated, use --setup-llm)")
	initCmd.Flags().BoolVar(&skipLLMSetup, "skip-llm", false, "Skip LLM backend configuration prompt")
	initCmd.Flags().BoolVar(&setupLLMOnly, "setup-llm", false, "Setup LLM backend only (skip roles/policy init)")
}

func runInit(cmd *cobra.Command, args []string) {
	// MCP registration only mode
	if registerMCPOnly {
		fmt.Println("🔧 Registering Symphony MCP server...")
		promptMCPRegistration()
		return
	}

	// API key setup only mode (deprecated)
	if setupAPIKeyOnly {
		fmt.Println("🔑 Setting up OpenAI API key...")
		promptAPIKeySetup()
		return
	}

	// LLM setup only mode
	if setupLLMOnly {
		fmt.Println("🤖 Setting up LLM backend...")
		promptLLMBackendSetup()
		return
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

	// Create default roles (empty user lists - users select their own role)
	newRoles := roles.Roles{
		"admin":     []string{},
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

	// Create default policy file with RBAC roles
	fmt.Println("\nCreating default policy file...")
	if err := createDefaultPolicy(); err != nil {
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

	// Automatically set role to admin during initialization
	// (Since this is initialization, the user should have full permissions)
	// This also handles the case where .env exists but CURRENT_ROLE is not set
	if err := roles.SetCurrentRole("admin"); err != nil {
		fmt.Printf("⚠ Warning: Failed to save role selection: %v\n", err)
	} else {
		fmt.Println("✓ Your role has been set to: admin (default for initialization)")
	}

	// MCP registration prompt
	if !skipMCPRegister {
		promptMCPRegistration()
	}

	// LLM backend configuration prompt
	if !skipLLMSetup && !skipAPIKey {
		promptLLMBackendSetup()
	}

	// Show dashboard guide after all initialization is complete
	fmt.Println("\n🎯 What's Next: Use Symphony Dashboard")
	fmt.Println()
	fmt.Println("Start the web dashboard:")
	fmt.Println("  sym dashboard")
	fmt.Println()
	fmt.Println("Dashboard features:")
	fmt.Println("  📋 Manage roles - Configure permissions for each role")
	fmt.Println("  📝 Edit policies - Create and modify coding conventions")
	fmt.Println("  🎭 Change role - Select a different role anytime")
	fmt.Println("  ✅ Test validation - Check rules against your code in real-time")
	fmt.Println()
	fmt.Println("After setup, commit and push .sym/roles.json and .sym/user-policy.json to share with your team.")
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

	// .env doesn't exist, create it with default settings
	content := fmt.Sprintf("# Symphony local configuration\nPOLICY_PATH=%s\nCURRENT_ROLE=admin\n", defaultPolicyPath)
	return os.WriteFile(envPath, []byte(content), 0644)
}

func removeExistingCodePolicy() error {
	// Files generated by convert command (dynamically retrieved from registry)
	convertGeneratedFiles := []string{"code-policy.json"}
	convertGeneratedFiles = append(convertGeneratedFiles, registry.Global().GetAllConfigFiles()...)

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
