package cmd

import (
	"fmt"

	"github.com/DevSymphony/sym-cli/internal/roles"
	"github.com/spf13/cobra"
)

var categoryCmd = &cobra.Command{
	Use:   "category",
	Short: "List all available convention categories",
	Long: `List all convention categories with their descriptions.

Categories are defined in user-policy.json and can be customized by the user.
Run 'sym init' to create default categories (security, style, documentation,
error_handling, architecture, performance, testing).

You can add, remove, or modify categories directly in user-policy.json.`,
	RunE: runCategory,
}

func init() {
	rootCmd.AddCommand(categoryCmd)
}

func runCategory(cmd *cobra.Command, args []string) error {
	// Load categories from user-policy.json
	userPolicy, err := roles.LoadUserPolicyFromRepo()
	if err != nil {
		printWarn("Failed to load user-policy.json")
		fmt.Println("Run 'sym init' to create default categories")
		return nil
	}

	categories := userPolicy.Category
	if len(categories) == 0 {
		printWarn("No categories defined in user-policy.json")
		fmt.Println("Run 'sym init' to create default categories")
		return nil
	}

	printTitle("Convention Categories", fmt.Sprintf("%d categories available", len(categories)))
	fmt.Println()

	for _, cat := range categories {
		fmt.Printf("  %s %s\n", colorize(bold, "•"), colorize(cyan, cat.Name))
		fmt.Printf("    %s\n\n", cat.Description)
	}

	return nil
}
