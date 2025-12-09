package cmd

import (
	"fmt"
	"os"

	"github.com/DevSymphony/sym-cli/internal/policy"
	"github.com/DevSymphony/sym-cli/internal/util/config"

	"github.com/spf13/cobra"
)

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage coding policy",
	Long: `Manage coding conventions and policy settings.

Commands:
  sym policy path           # Show current policy file path
  sym policy path --set PATH  # Set policy file path
  sym policy validate        # Validate policy file`,
}

var policyPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show or set policy file path",
	Long:  `Display the current policy file path or set a new one.`,
	Run:   runPolicyPath,
}

var policyValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate policy file",
	Long:  `Validate the syntax and structure of the policy file.`,
	Run:   runPolicyValidate,
}

var (
	policyPathSet string
)

func init() {
	policyPathCmd.Flags().StringVar(&policyPathSet, "set", "", "Set new policy file path")

	policyCmd.AddCommand(policyPathCmd)
	policyCmd.AddCommand(policyValidateCmd)
}

func runPolicyPath(cmd *cobra.Command, args []string) {
	cfg, err := config.LoadConfig()
	if err != nil {
		printError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	if policyPathSet != "" {
		// Set new path
		cfg.PolicyPath = policyPathSet
		if err := config.SaveConfig(cfg); err != nil {
			printError(fmt.Sprintf("Failed to save config: %v", err))
			os.Exit(1)
		}

		printOK(fmt.Sprintf("Policy path updated: %s", policyPathSet))
	} else {
		// Show current path
		policyPath := cfg.PolicyPath
		if policyPath == "" {
			policyPath = ".sym/user-policy.json (default)"
		}

		fmt.Printf("Current policy path: %s\n", policyPath)

		// Show full path
		fullPath, err := policy.GetPolicyPath(cfg.PolicyPath)
		if err == nil {
			fmt.Printf("Full path: %s\n", fullPath)
		}

		// Check if file exists
		exists, err := policy.PolicyExists(cfg.PolicyPath)
		if err != nil {
			fmt.Printf("Error checking file: %v\n", err)
		} else if exists {
			printOK("Policy file exists")
		} else {
			printWarn("Policy file does not exist")
		}
	}
}

func runPolicyValidate(cmd *cobra.Command, args []string) {
	cfg, err := config.LoadConfig()
	if err != nil {
		printError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	fmt.Println("Validating policy file...")

	policyData, err := policy.LoadPolicy(cfg.PolicyPath)
	if err != nil {
		printError(fmt.Sprintf("Failed to load policy: %v", err))
		os.Exit(1)
	}

	if err := policy.ValidatePolicy(policyData); err != nil {
		printError(fmt.Sprintf("Validation failed: %v", err))
		os.Exit(1)
	}

	printOK("Policy file is valid")
	fmt.Printf("  Version: %s\n", policyData.Version)
	fmt.Printf("  Rules: %d\n", len(policyData.Rules))

	if policyData.RBAC != nil {
		fmt.Printf("  RBAC roles: %d\n", len(policyData.RBAC.Roles))
	}

	if policyData.Defaults != nil {
		if len(policyData.Defaults.Languages) > 0 {
			fmt.Printf("  Default languages: %v\n", policyData.Defaults.Languages)
		}
	}
}
