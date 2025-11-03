package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"github.com/DevSymphony/sym-cli/internal/config"
	"github.com/DevSymphony/sym-cli/internal/policy"

	"github.com/spf13/cobra"
)

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage coding policy",
	Long: `Manage coding conventions and policy settings.

Commands:
  sym policy path           # Show current policy file path
  sym policy path --set PATH  # Set policy file path
  sym policy validate        # Validate policy file
  sym policy history         # Show policy change history`,
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

var policyHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show policy change history",
	Long:  `Display the git commit history for the policy file.`,
	Run:   runPolicyHistory,
}

var (
	policyPathSet string
	historyLimit  int
	jsonOutput    bool
)

func init() {
	policyPathCmd.Flags().StringVar(&policyPathSet, "set", "", "Set new policy file path")
	policyHistoryCmd.Flags().IntVarP(&historyLimit, "limit", "n", 10, "Number of commits to show")
	policyHistoryCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	policyCmd.AddCommand(policyPathCmd)
	policyCmd.AddCommand(policyValidateCmd)
	policyCmd.AddCommand(policyHistoryCmd)
}

func runPolicyPath(cmd *cobra.Command, args []string) {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("❌ Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if policyPathSet != "" {
		// Set new path
		cfg.PolicyPath = policyPathSet
		if err := config.SaveConfig(cfg); err != nil {
			fmt.Printf("❌ Failed to save config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✓ Policy path updated: %s\n", policyPathSet)
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
			fmt.Println("✓ Policy file exists")
		} else {
			fmt.Println("⚠ Policy file does not exist")
		}
	}
}

func runPolicyValidate(cmd *cobra.Command, args []string) {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("❌ Failed to load config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Validating policy file...")

	policyData, err := policy.LoadPolicy(cfg.PolicyPath)
	if err != nil {
		fmt.Printf("❌ Failed to load policy: %v\n", err)
		os.Exit(1)
	}

	if err := policy.ValidatePolicy(policyData); err != nil {
		fmt.Printf("❌ Validation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Policy file is valid")
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

func runPolicyHistory(cmd *cobra.Command, args []string) {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("❌ Failed to load config: %v\n", err)
		os.Exit(1)
	}

	history, err := policy.GetPolicyHistory(cfg.PolicyPath, historyLimit)
	if err != nil {
		fmt.Printf("❌ Failed to get history: %v\n", err)
		os.Exit(1)
	}

	if len(history) == 0 {
		fmt.Println("No policy changes found")
		return
	}

	if jsonOutput {
		data, _ := json.MarshalIndent(history, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Printf("Policy change history (last %d commits):\n\n", len(history))

	for i, commit := range history {
		fmt.Printf("%d. %s\n", i+1, commit.Message)
		fmt.Printf("   %s - %s <%s>\n", commit.Hash[:7], commit.Author, commit.Email)
		fmt.Printf("   %s\n\n", commit.Date.Format("2006-01-02 15:04:05"))
	}
}
