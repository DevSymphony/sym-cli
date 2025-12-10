package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/DevSymphony/sym-cli/internal/policy"
	"github.com/DevSymphony/sym-cli/internal/roles"
	"github.com/DevSymphony/sym-cli/pkg/schema"
	"github.com/spf13/cobra"
)

// CategoryItem represents a category for batch operations.
type CategoryItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CategoryEditItem represents a category edit for batch operations.
type CategoryEditItem struct {
	Name        string `json:"name"`
	NewName     string `json:"new_name,omitempty"`
	Description string `json:"description,omitempty"`
}

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
	Use:   "add [name] [description]",
	Short: "Add a new category",
	Long: `Add a new convention category.

Single mode:
  sym category add accessibility "Accessibility rules (WCAG, ARIA, etc.)"

Batch mode (JSON file):
  sym category add -f categories.json

  categories.json format:
  [
    {"name": "security", "description": "Security rules"},
    {"name": "performance", "description": "Performance rules"}
  ]`,
	Args: cobra.MaximumNArgs(2),
	RunE: runCategoryAdd,
}

var categoryEditCmd = &cobra.Command{
	Use:   "edit [name]",
	Short: "Edit an existing category",
	Long: `Edit an existing category's name or description.

Single mode:
  sym category edit security --description "Updated security rules"
  sym category edit old-name --name new-name
  sym category edit security --name sec --description "Security conventions"

Batch mode (JSON file):
  sym category edit -f edits.json

  edits.json format:
  [
    {"name": "security", "new_name": "sec"},
    {"name": "performance", "description": "New description"}
  ]`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCategoryEdit,
}

var categoryRemoveCmd = &cobra.Command{
	Use:   "remove [names...]",
	Short: "Remove a category",
	Long: `Remove categories from user-policy.json.

Note: Categories that are referenced by rules cannot be removed.
You must first remove or update the rules that reference the category.

Single mode:
  sym category remove deprecated-category

Batch mode (multiple args):
  sym category remove cat1 cat2 cat3

Batch mode (JSON file):
  sym category remove -f names.json

  names.json format:
  ["cat1", "cat2", "cat3"]`,
	Args: cobra.ArbitraryArgs,
	RunE: runCategoryRemove,
}

func init() {
	rootCmd.AddCommand(categoryCmd)

	// Add subcommands
	categoryCmd.AddCommand(categoryListCmd)
	categoryCmd.AddCommand(categoryAddCmd)
	categoryCmd.AddCommand(categoryEditCmd)
	categoryCmd.AddCommand(categoryRemoveCmd)

	// Add command flags
	categoryAddCmd.Flags().StringP("file", "f", "", "JSON file with categories to add")

	// Edit command flags
	categoryEditCmd.Flags().String("name", "", "New category name")
	categoryEditCmd.Flags().String("description", "", "New category description")
	categoryEditCmd.Flags().StringP("file", "f", "", "JSON file with category edits")

	// Remove command flags
	categoryRemoveCmd.Flags().StringP("file", "f", "", "JSON file with category names to remove")
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
	fileFlag, _ := cmd.Flags().GetString("file")

	// Load policy
	userPolicy, err := policy.LoadPolicy("")
	if err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}

	// Build existing names map
	existingNames := make(map[string]bool)
	for _, cat := range userPolicy.Category {
		existingNames[cat.Name] = true
	}

	var categories []CategoryItem

	if fileFlag != "" {
		// Batch mode: load from JSON file
		data, err := os.ReadFile(fileFlag)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		if err := json.Unmarshal(data, &categories); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
		if len(categories) == 0 {
			return fmt.Errorf("no categories found in file")
		}
	} else {
		// Single mode: use args
		if len(args) != 2 {
			return fmt.Errorf("usage: sym category add <name> <description> or sym category add -f <file>")
		}
		categories = []CategoryItem{{Name: args[0], Description: args[1]}}
	}

	var succeeded []string
	var failed []string

	// Process each category
	for _, cat := range categories {
		if cat.Name == "" {
			failed = append(failed, "(empty): name is required")
			continue
		}
		if cat.Description == "" {
			failed = append(failed, fmt.Sprintf("%s: description is required", cat.Name))
			continue
		}
		if existingNames[cat.Name] {
			failed = append(failed, fmt.Sprintf("%s: already exists", cat.Name))
			continue
		}

		userPolicy.Category = append(userPolicy.Category, schema.CategoryDef{
			Name:        cat.Name,
			Description: cat.Description,
		})
		existingNames[cat.Name] = true
		succeeded = append(succeeded, cat.Name)
	}

	// Save policy if any succeeded
	if len(succeeded) > 0 {
		if err := policy.SavePolicy(userPolicy, ""); err != nil {
			return fmt.Errorf("failed to save policy: %w", err)
		}
	}

	// Print results
	printBatchResult("Added", succeeded, failed)
	return nil
}

func runCategoryEdit(cmd *cobra.Command, args []string) error {
	fileFlag, _ := cmd.Flags().GetString("file")
	newName, _ := cmd.Flags().GetString("name")
	newDescription, _ := cmd.Flags().GetString("description")

	// Load policy
	userPolicy, err := policy.LoadPolicy("")
	if err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}

	// Build category index map
	categoryIndex := make(map[string]int)
	for i, cat := range userPolicy.Category {
		categoryIndex[cat.Name] = i
	}

	var edits []CategoryEditItem

	if fileFlag != "" {
		// Batch mode: load from JSON file
		data, err := os.ReadFile(fileFlag)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		if err := json.Unmarshal(data, &edits); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
		if len(edits) == 0 {
			return fmt.Errorf("no edits found in file")
		}
	} else {
		// Single mode: use args
		if len(args) != 1 {
			return fmt.Errorf("usage: sym category edit <name> [--name <new>] [--description <desc>] or sym category edit -f <file>")
		}
		if newName == "" && newDescription == "" {
			return fmt.Errorf("at least one of --name or --description must be provided")
		}
		edits = []CategoryEditItem{{Name: args[0], NewName: newName, Description: newDescription}}
	}

	var succeeded []string
	var failed []string

	// Process each edit
	for _, edit := range edits {
		if edit.Name == "" {
			failed = append(failed, "(empty): name is required")
			continue
		}
		if edit.NewName == "" && edit.Description == "" {
			failed = append(failed, fmt.Sprintf("%s: at least one of new_name or description required", edit.Name))
			continue
		}

		idx, exists := categoryIndex[edit.Name]
		if !exists {
			failed = append(failed, fmt.Sprintf("%s: not found", edit.Name))
			continue
		}

		rulesUpdated := 0
		resultText := edit.Name

		// If renaming
		if edit.NewName != "" && edit.NewName != edit.Name {
			if _, dupExists := categoryIndex[edit.NewName]; dupExists {
				failed = append(failed, fmt.Sprintf("%s: '%s' already exists", edit.Name, edit.NewName))
				continue
			}

			// Update rule references
			for i := range userPolicy.Rules {
				if userPolicy.Rules[i].Category == edit.Name {
					userPolicy.Rules[i].Category = edit.NewName
					rulesUpdated++
				}
			}

			delete(categoryIndex, edit.Name)
			categoryIndex[edit.NewName] = idx
			userPolicy.Category[idx].Name = edit.NewName
			resultText = fmt.Sprintf("%s → %s", edit.Name, edit.NewName)
		}

		// Update description
		if edit.Description != "" {
			userPolicy.Category[idx].Description = edit.Description
			if edit.NewName == "" || edit.NewName == edit.Name {
				resultText = fmt.Sprintf("%s (description updated)", edit.Name)
			}
		}

		if rulesUpdated > 0 {
			resultText = fmt.Sprintf("%s (%d rules updated)", resultText, rulesUpdated)
		}

		succeeded = append(succeeded, resultText)
	}

	// Save policy if any succeeded
	if len(succeeded) > 0 {
		if err := policy.SavePolicy(userPolicy, ""); err != nil {
			return fmt.Errorf("failed to save policy: %w", err)
		}
	}

	// Print results
	printBatchResult("Updated", succeeded, failed)
	return nil
}

func runCategoryRemove(cmd *cobra.Command, args []string) error {
	fileFlag, _ := cmd.Flags().GetString("file")

	// Load policy
	userPolicy, err := policy.LoadPolicy("")
	if err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}

	// Build category index map and rule count map
	categoryIndex := make(map[string]int)
	for i, cat := range userPolicy.Category {
		categoryIndex[cat.Name] = i
	}

	ruleCount := make(map[string]int)
	for _, rule := range userPolicy.Rules {
		ruleCount[rule.Category]++
	}

	var names []string

	if fileFlag != "" {
		// Batch mode: load from JSON file
		data, err := os.ReadFile(fileFlag)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		if err := json.Unmarshal(data, &names); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
		if len(names) == 0 {
			return fmt.Errorf("no category names found in file")
		}
	} else if len(args) > 0 {
		// Single or batch mode: use args
		names = args
	} else {
		return fmt.Errorf("usage: sym category remove <name> [names...] or sym category remove -f <file>")
	}

	var succeeded []string
	var failed []string
	toRemove := make(map[int]bool)

	// Process each name
	for _, name := range names {
		if name == "" {
			failed = append(failed, "(empty): name is required")
			continue
		}

		idx, exists := categoryIndex[name]
		if !exists {
			failed = append(failed, fmt.Sprintf("%s: not found", name))
			continue
		}

		if count := ruleCount[name]; count > 0 {
			failed = append(failed, fmt.Sprintf("%s: used by %d rule(s)", name, count))
			continue
		}

		toRemove[idx] = true
		succeeded = append(succeeded, name)
	}

	// Remove categories
	if len(toRemove) > 0 {
		newCategories := make([]schema.CategoryDef, 0, len(userPolicy.Category)-len(toRemove))
		for i, cat := range userPolicy.Category {
			if !toRemove[i] {
				newCategories = append(newCategories, cat)
			}
		}
		userPolicy.Category = newCategories

		if err := policy.SavePolicy(userPolicy, ""); err != nil {
			return fmt.Errorf("failed to save policy: %w", err)
		}
	}

	// Print results
	printBatchResult("Removed", succeeded, failed)
	return nil
}

// printBatchResult prints the result of a batch operation.
func printBatchResult(action string, succeeded, failed []string) {
	if len(failed) == 0 && len(succeeded) > 0 {
		if len(succeeded) == 1 {
			printDone(fmt.Sprintf("%s category: %s", action, succeeded[0]))
		} else {
			printDone(fmt.Sprintf("%s %d categories:", action, len(succeeded)))
			for _, name := range succeeded {
				fmt.Printf("  • %s\n", name)
			}
		}
	} else if len(succeeded) == 0 && len(failed) > 0 {
		printWarn(fmt.Sprintf("Failed to %s any categories:", action))
		for _, f := range failed {
			fmt.Printf("  ✗ %s\n", f)
		}
	} else if len(succeeded) > 0 && len(failed) > 0 {
		printWarn("Batch operation completed with errors:")
		fmt.Printf("  ✓ %s (%d):\n", action, len(succeeded))
		for _, name := range succeeded {
			fmt.Printf("    • %s\n", name)
		}
		fmt.Printf("  ✗ Failed (%d):\n", len(failed))
		for _, f := range failed {
			fmt.Printf("    • %s\n", f)
		}
	} else {
		printWarn("No categories to process")
	}
}
