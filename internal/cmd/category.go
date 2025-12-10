package cmd

import (
	"fmt"

	"github.com/DevSymphony/sym-cli/internal/policy"
	"github.com/DevSymphony/sym-cli/internal/roles"
	"github.com/DevSymphony/sym-cli/pkg/schema"
	"github.com/spf13/cobra"
)

var categoryCmd = &cobra.Command{
	Use:   "category",
	Short: "Manage convention categories",
	Long: `Manage convention categories in user-policy.json.

Categories help organize rules by concern area (e.g., security, style, performance).

Available subcommands:
  list    - List all categories
  add     - Add a new category
  edit    - Edit an existing category
  remove  - Remove a category`,
}

var categoryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all categories",
	Long: `List all convention categories with their descriptions.

Categories are defined in user-policy.json and can be customized by the user.
Run 'sym init' to create default categories (security, style, documentation,
error_handling, architecture, performance, testing).`,
	RunE: runCategoryList,
}

var categoryAddCmd = &cobra.Command{
	Use:   "add <name> <description>",
	Short: "Add a new category",
	Long: `Add a new convention category.

Example:
  sym category add accessibility "Accessibility rules (WCAG, ARIA, etc.)"`,
	Args: cobra.ExactArgs(2),
	RunE: runCategoryAdd,
}

var categoryEditCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit an existing category",
	Long: `Edit an existing category's name or description.

Use --name to change the category name (will update all rule references).
Use --description to change the category description.

Examples:
  sym category edit security --description "Updated security rules"
  sym category edit old-name --name new-name
  sym category edit security --name sec --description "Security conventions"`,
	Args: cobra.ExactArgs(1),
	RunE: runCategoryEdit,
}

var categoryRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a category",
	Long: `Remove a category from user-policy.json.

Note: Categories that are referenced by rules cannot be removed.
You must first remove or update the rules that reference the category.

Example:
  sym category remove deprecated-category`,
	Args: cobra.ExactArgs(1),
	RunE: runCategoryRemove,
}

func init() {
	rootCmd.AddCommand(categoryCmd)

	// Add subcommands
	categoryCmd.AddCommand(categoryListCmd)
	categoryCmd.AddCommand(categoryAddCmd)
	categoryCmd.AddCommand(categoryEditCmd)
	categoryCmd.AddCommand(categoryRemoveCmd)

	// Edit command flags
	categoryEditCmd.Flags().String("name", "", "New category name")
	categoryEditCmd.Flags().String("description", "", "New category description")
}

func runCategoryList(cmd *cobra.Command, args []string) error {
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

func runCategoryAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	description := args[1]

	// Validate inputs
	if name == "" {
		return fmt.Errorf("category name is required")
	}
	if description == "" {
		return fmt.Errorf("category description is required")
	}

	// Load policy
	userPolicy, err := policy.LoadPolicy("")
	if err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}

	// Check for duplicate
	for _, cat := range userPolicy.Category {
		if cat.Name == name {
			return fmt.Errorf("category '%s' already exists", name)
		}
	}

	// Add new category
	userPolicy.Category = append(userPolicy.Category, schema.CategoryDef{
		Name:        name,
		Description: description,
	})

	// Save policy
	if err := policy.SavePolicy(userPolicy, ""); err != nil {
		return fmt.Errorf("failed to save policy: %w", err)
	}

	printDone(fmt.Sprintf("Category '%s' added successfully", name))
	return nil
}

func runCategoryEdit(cmd *cobra.Command, args []string) error {
	currentName := args[0]
	newName, _ := cmd.Flags().GetString("name")
	newDescription, _ := cmd.Flags().GetString("description")

	// Validate at least one change
	if newName == "" && newDescription == "" {
		return fmt.Errorf("at least one of --name or --description must be provided")
	}

	// Load policy
	userPolicy, err := policy.LoadPolicy("")
	if err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}

	// Find category
	var categoryIndex = -1
	for i, cat := range userPolicy.Category {
		if cat.Name == currentName {
			categoryIndex = i
			break
		}
	}

	if categoryIndex == -1 {
		return fmt.Errorf("category '%s' not found", currentName)
	}

	// If renaming, check for duplicate and update rule references
	affectedRules := 0
	if newName != "" && newName != currentName {
		// Check for duplicate
		for _, cat := range userPolicy.Category {
			if cat.Name == newName {
				return fmt.Errorf("category '%s' already exists", newName)
			}
		}

		// Update rule references
		for i := range userPolicy.Rules {
			if userPolicy.Rules[i].Category == currentName {
				userPolicy.Rules[i].Category = newName
				affectedRules++
			}
		}

		userPolicy.Category[categoryIndex].Name = newName
	}

	// Update description if provided
	if newDescription != "" {
		userPolicy.Category[categoryIndex].Description = newDescription
	}

	// Save policy
	if err := policy.SavePolicy(userPolicy, ""); err != nil {
		return fmt.Errorf("failed to save policy: %w", err)
	}

	if affectedRules > 0 {
		printDone(fmt.Sprintf("Category updated successfully (%d rule(s) updated)", affectedRules))
	} else {
		printDone("Category updated successfully")
	}
	return nil
}

func runCategoryRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Load policy
	userPolicy, err := policy.LoadPolicy("")
	if err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}

	// Find category
	var categoryIndex = -1
	for i, cat := range userPolicy.Category {
		if cat.Name == name {
			categoryIndex = i
			break
		}
	}

	if categoryIndex == -1 {
		return fmt.Errorf("category '%s' not found", name)
	}

	// Check if any rules reference this category
	rulesUsingCategory := 0
	for _, rule := range userPolicy.Rules {
		if rule.Category == name {
			rulesUsingCategory++
		}
	}

	if rulesUsingCategory > 0 {
		return fmt.Errorf("category '%s' is used by %d rule(s). Remove rule references first", name, rulesUsingCategory)
	}

	// Remove category
	userPolicy.Category = append(userPolicy.Category[:categoryIndex], userPolicy.Category[categoryIndex+1:]...)

	// Save policy
	if err := policy.SavePolicy(userPolicy, ""); err != nil {
		return fmt.Errorf("failed to save policy: %w", err)
	}

	printDone(fmt.Sprintf("Category '%s' removed successfully", name))
	return nil
}
