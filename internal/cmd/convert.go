package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/DevSymphony/sym-cli/internal/converter"
	"github.com/DevSymphony/sym-cli/pkg/schema"
	"github.com/spf13/cobra"
)

var (
	convertInputFile  string
	convertOutputFile string
)

var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert user policies into a validatable format",
	Long: `Convert natural language policies (Schema A) written by users
into a structured schema (Schema B) that the validation engine can read.`,
	Example: `  sym convert -i user-policy.json -o code-policy.json
  sym convert -i conventions.json -o .sym/policy.json`,
	RunE: runConvert,
}

func init() {
	convertCmd.Flags().StringVarP(&convertInputFile, "input", "i", "user-policy.json", "input user policy file")
	convertCmd.Flags().StringVarP(&convertOutputFile, "output", "o", "code-policy.json", "output code policy file")
}

func runConvert(cmd *cobra.Command, args []string) error {
	data, err := os.ReadFile(convertInputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	var userPolicy schema.UserPolicy
	if err := json.Unmarshal(data, &userPolicy); err != nil {
		return fmt.Errorf("failed to parse user policy: %w", err)
	}

	conv := converter.NewConverter(verbose)

	fmt.Printf("converting %d natural language rules into structured policy...\n", len(userPolicy.Rules))

	codePolicy, err := conv.Convert(&userPolicy)
	if err != nil {
		return fmt.Errorf("conversion failed: %w", err)
	}

	output, err := json.MarshalIndent(codePolicy, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize code policy: %w", err)
	}

	if err := os.WriteFile(convertOutputFile, output, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("conversion completed: %s\n", convertOutputFile)
	fmt.Printf("  - processed rules: %d\n", len(codePolicy.Rules))
	if codePolicy.RBAC != nil {
		fmt.Printf("  - RBAC roles: %d\n", len(codePolicy.RBAC.Roles))
	}

	return nil
}
