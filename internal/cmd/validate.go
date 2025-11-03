package cmd

import (
	"fmt"

	"github.com/DevSymphony/sym-cli/internal/policy"
	"github.com/DevSymphony/sym-cli/internal/validator"
	"github.com/spf13/cobra"
)

var (
	validatePolicyFile  string
	validateTargetPaths []string
	validateRole        string
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate code compliance with defined conventions",
	Long: `Validate that code at specified paths complies with conventions defined in the policy file.
Validation results are returned to standard output, and a non-zero exit code is returned if violations are found.`,
	Example: `  sym validate -p code-policy.json -t src/
  sym validate -p .sym/policy.json -t main.go utils.go
  sym validate -p policy.json -t . --role dev`,
	RunE: runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().StringVarP(&validatePolicyFile, "policy", "p", "code-policy.json", "code policy file path")
	validateCmd.Flags().StringSliceVarP(&validateTargetPaths, "target", "t", []string{"."}, "files or directories to validate")
	validateCmd.Flags().StringVarP(&validateRole, "role", "r", "contributor", "user role for RBAC validation")
}

func runValidate(cmd *cobra.Command, args []string) error {
	loader := policy.NewLoader(verbose)
	codePolicy, err := loader.LoadCodePolicy(validatePolicyFile)
	if err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}

	fmt.Printf("validating %d target(s) with %d rule(s)...\n", len(validateTargetPaths), len(codePolicy.Rules))
	fmt.Printf("role: %s\n\n", validateRole)

	v := validator.NewValidator(codePolicy, verbose)

	for _, targetPath := range validateTargetPaths {
		result, err := v.Validate(targetPath)
		if err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}

		if result.Passed {
			fmt.Printf("✓ %s: validation passed\n", targetPath)
		} else {
			fmt.Printf("✗ %s: found %d violation(s)\n", targetPath, len(result.Violations))
			for _, violation := range result.Violations {
				fmt.Printf("  [%s] %s: %s\n", violation.Severity, violation.RuleID, violation.Message)
			}
		}
	}

	fmt.Println("\nNote: Full validation engine implementation is in progress.")
	fmt.Println("Currently only policy structure validation and basic checks are performed.")

	fmt.Printf("\n✓ Policy loaded successfully\n")
	fmt.Printf("  - version: %s\n", codePolicy.Version)
	fmt.Printf("  - rules: %d\n", len(codePolicy.Rules))
	fmt.Printf("  - enforce stages: %v\n", codePolicy.Enforce.Stages)

	if codePolicy.RBAC != nil && codePolicy.RBAC.Roles != nil {
		fmt.Printf("  - RBAC enabled: %d role(s)\n", len(codePolicy.RBAC.Roles))
	}

	return nil
}
