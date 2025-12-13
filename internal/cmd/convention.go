package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/policy"
	"github.com/DevSymphony/sym-cli/internal/roles"
	"github.com/DevSymphony/sym-cli/pkg/schema"
	"github.com/spf13/cobra"
)

// ConventionItem represents a convention for batch operations.
type ConventionItem struct {
	ID        string   `json:"id"`
	Say       string   `json:"say"`
	Category  string   `json:"category,omitempty"`
	Languages []string `json:"languages,omitempty"`
	Severity  string   `json:"severity,omitempty"`
	Autofix   bool     `json:"autofix,omitempty"`
	Message   string   `json:"message,omitempty"`
	Example   string   `json:"example,omitempty"`
	Include   []string `json:"include,omitempty"`
	Exclude   []string `json:"exclude,omitempty"`
}

// ConventionEditItem represents a convention edit for batch operations.
type ConventionEditItem struct {
	ID        string   `json:"id"`
	NewID     string   `json:"new_id,omitempty"`
	Say       string   `json:"say,omitempty"`
	Category  string   `json:"category,omitempty"`
	Languages []string `json:"languages,omitempty"`
	Severity  string   `json:"severity,omitempty"`
	Autofix   *bool    `json:"autofix,omitempty"`
	Message   string   `json:"message,omitempty"`
	Example   string   `json:"example,omitempty"`
	Include   []string `json:"include,omitempty"`
	Exclude   []string `json:"exclude,omitempty"`
}

var conventionCmd = &cobra.Command{
	Use:   "convention",
	Short: "Manage conventions (rules)",
	Long: `Manage conventions (rules) in user-policy.json.

Conventions define coding standards and rules that are enforced during validation.

Available subcommands:
  list    - List all conventions
  add     - Add a new convention
  edit    - Edit an existing convention
  remove  - Remove a convention`,
}

var conventionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all conventions",
	Long: `List conventions with their ID, category, description, and severity.

Conventions are defined in user-policy.json and can be customized by the user.
Run 'sym init' to create default conventions.

Examples:
  sym convention list
  sym convention list --category security
  sym convention list --languages go
  sym convention list --languages go,javascript
  sym convention list --category style --languages typescript`,
	RunE: runConventionList,
}

var conventionAddCmd = &cobra.Command{
	Use:   "add [id] [say]",
	Short: "Add a new convention",
	Long: `Add a new convention (rule).

Single mode:
  sym convention add NAMING-001 "Use snake_case for variables" --category naming --languages python --severity error

Batch mode (JSON file):
  sym convention add -f conventions.json

  conventions.json format:
  [
    {
      "id": "NAMING-001",
      "say": "Use snake_case for variables",
      "category": "naming",
      "languages": ["python"],
      "severity": "error",
      "message": "Variable names must use snake_case"
    }
  ]`,
	Args: cobra.MaximumNArgs(2),
	RunE: runConventionAdd,
}

var conventionEditCmd = &cobra.Command{
	Use:   "edit [id]",
	Short: "Edit an existing convention",
	Long: `Edit an existing convention's fields.

Single mode:
  sym convention edit NAMING-001 --say "Updated description" --severity warning
  sym convention edit NAMING-001 --id NAMING-002 --category style
  sym convention edit NAMING-001 --languages python,go

Batch mode (JSON file):
  sym convention edit -f edits.json

  edits.json format:
  [
    {"id": "NAMING-001", "say": "Updated", "severity": "warning"},
    {"id": "STYLE-001", "new_id": "STYLE-002", "category": "formatting"}
  ]`,
	Args: cobra.MaximumNArgs(1),
	RunE: runConventionEdit,
}

var conventionRemoveCmd = &cobra.Command{
	Use:   "remove [ids...]",
	Short: "Remove a convention",
	Long: `Remove conventions from user-policy.json.

Single mode:
  sym convention remove NAMING-001

Batch mode (multiple args):
  sym convention remove NAMING-001 NAMING-002 STYLE-001

Batch mode (JSON file):
  sym convention remove -f ids.json

  ids.json format:
  ["NAMING-001", "NAMING-002", "STYLE-001"]`,
	Args: cobra.ArbitraryArgs,
	RunE: runConventionRemove,
}

func init() {
	rootCmd.AddCommand(conventionCmd)

	// Add subcommands
	conventionCmd.AddCommand(conventionListCmd)
	conventionCmd.AddCommand(conventionAddCmd)
	conventionCmd.AddCommand(conventionEditCmd)
	conventionCmd.AddCommand(conventionRemoveCmd)

	// Add command flags
	conventionAddCmd.Flags().StringP("file", "f", "", "JSON file with conventions to add")
	conventionAddCmd.Flags().String("category", "", "Category name")
	conventionAddCmd.Flags().StringSlice("languages", nil, "Programming languages (comma-separated)")
	conventionAddCmd.Flags().String("severity", "", "Severity level (error, warning, info)")
	conventionAddCmd.Flags().Bool("autofix", false, "Enable auto-fix")
	conventionAddCmd.Flags().String("message", "", "Message to display on violation")
	conventionAddCmd.Flags().String("example", "", "Code example")
	conventionAddCmd.Flags().StringSlice("include", nil, "File patterns to include")
	conventionAddCmd.Flags().StringSlice("exclude", nil, "File patterns to exclude")

	// Edit command flags
	conventionEditCmd.Flags().StringP("file", "f", "", "JSON file with convention edits")
	conventionEditCmd.Flags().String("id", "", "New convention ID")
	conventionEditCmd.Flags().String("say", "", "New description")
	conventionEditCmd.Flags().String("category", "", "New category name")
	conventionEditCmd.Flags().StringSlice("languages", nil, "New programming languages (comma-separated)")
	conventionEditCmd.Flags().String("severity", "", "New severity level (error, warning, info)")
	conventionEditCmd.Flags().Bool("autofix", false, "Enable auto-fix")
	conventionEditCmd.Flags().String("message", "", "New message to display on violation")
	conventionEditCmd.Flags().String("example", "", "New code example")
	conventionEditCmd.Flags().StringSlice("include", nil, "New file patterns to include")
	conventionEditCmd.Flags().StringSlice("exclude", nil, "New file patterns to exclude")

	// Remove command flags
	conventionRemoveCmd.Flags().StringP("file", "f", "", "JSON file with convention IDs to remove")

	// List command flags
	conventionListCmd.Flags().String("category", "", "Filter by category (e.g., security, style, documentation)")
	conventionListCmd.Flags().StringSlice("languages", nil, "Filter by programming languages (comma-separated)")
}

func runConventionList(cmd *cobra.Command, args []string) error {
	// Get filter flags
	categoryFilter, _ := cmd.Flags().GetString("category")
	languagesFilter, _ := cmd.Flags().GetStringSlice("languages")

	// Normalize category: "all" or empty means no filtering
	if strings.EqualFold(categoryFilter, "all") {
		categoryFilter = ""
	}

	// Load conventions from user-policy.json
	userPolicy, err := roles.LoadUserPolicyFromRepo()
	if err != nil {
		printWarn("Failed to load user-policy.json")
		fmt.Println("Run 'sym init' to create default conventions")
		return nil
	}

	rules := userPolicy.Rules
	if len(rules) == 0 {
		printWarn("No conventions defined in user-policy.json")
		fmt.Println("Run 'sym init' to create default conventions or use 'sym convention add' to add new ones")
		return nil
	}

	// Filter rules
	var filteredRules []schema.UserRule
	for _, rule := range rules {
		// Category filter
		if categoryFilter != "" && rule.Category != categoryFilter {
			continue
		}

		// Languages filter: if both request and rule have languages, check intersection
		if len(languagesFilter) > 0 && len(rule.Languages) > 0 {
			if !containsAny(rule.Languages, languagesFilter) {
				continue
			}
		}

		filteredRules = append(filteredRules, rule)
	}

	// Build filter description for output
	filterDesc := ""
	if categoryFilter != "" || len(languagesFilter) > 0 {
		parts := []string{}
		if categoryFilter != "" {
			parts = append(parts, fmt.Sprintf("category=%s", categoryFilter))
		}
		if len(languagesFilter) > 0 {
			parts = append(parts, fmt.Sprintf("languages=%s", strings.Join(languagesFilter, ",")))
		}
		filterDesc = fmt.Sprintf(" (filtered: %s)", strings.Join(parts, ", "))
	}

	if len(filteredRules) == 0 {
		printWarn(fmt.Sprintf("No conventions found%s", filterDesc))
		return nil
	}

	printTitle("Conventions", fmt.Sprintf("%d conventions available%s", len(filteredRules), filterDesc))
	fmt.Println()

	for _, rule := range filteredRules {
		// Format: ID [CATEGORY] (languages): description
		languages := ""
		if len(rule.Languages) > 0 {
			languages = fmt.Sprintf(" (%s)", strings.Join(rule.Languages, ", "))
		}
		category := ""
		if rule.Category != "" {
			category = fmt.Sprintf(" [%s]", rule.Category)
		}
		severity := rule.Severity
		if severity == "" {
			severity = "warning"
		}

		fmt.Printf("  %s %s%s%s\n", colorize(bold, "•"), colorize(cyan, rule.ID), colorize(yellow, category), languages)
		fmt.Printf("    %s\n", rule.Say)
		fmt.Printf("    severity: %s\n\n", severity)
	}

	return nil
}

func runConventionAdd(cmd *cobra.Command, args []string) error {
	fileFlag, _ := cmd.Flags().GetString("file")

	// Load policy
	userPolicy, err := policy.LoadPolicy("")
	if err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}

	// Build existing IDs map
	existingIDs := make(map[string]bool)
	for _, rule := range userPolicy.Rules {
		existingIDs[rule.ID] = true
	}

	var conventions []ConventionItem

	if fileFlag != "" {
		// Batch mode: load from JSON file
		data, err := os.ReadFile(fileFlag)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		if err := json.Unmarshal(data, &conventions); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
		if len(conventions) == 0 {
			return fmt.Errorf("no conventions found in file")
		}
	} else {
		// Single mode: use args and flags
		if len(args) != 2 {
			return fmt.Errorf("usage: sym convention add <id> <say> [--flags] or sym convention add -f <file>")
		}

		category, _ := cmd.Flags().GetString("category")
		languages, _ := cmd.Flags().GetStringSlice("languages")
		severity, _ := cmd.Flags().GetString("severity")
		autofix, _ := cmd.Flags().GetBool("autofix")
		message, _ := cmd.Flags().GetString("message")
		example, _ := cmd.Flags().GetString("example")
		include, _ := cmd.Flags().GetStringSlice("include")
		exclude, _ := cmd.Flags().GetStringSlice("exclude")

		conventions = []ConventionItem{{
			ID:        args[0],
			Say:       args[1],
			Category:  category,
			Languages: languages,
			Severity:  severity,
			Autofix:   autofix,
			Message:   message,
			Example:   example,
			Include:   include,
			Exclude:   exclude,
		}}
	}

	var succeeded []string
	var failed []string
	var addedRules []schema.UserRule

	// Process each convention
	for _, conv := range conventions {
		if conv.ID == "" {
			failed = append(failed, "(empty): ID is required")
			continue
		}
		if conv.Say == "" {
			failed = append(failed, fmt.Sprintf("%s: say (description) is required", conv.ID))
			continue
		}
		if existingIDs[conv.ID] {
			failed = append(failed, fmt.Sprintf("%s: already exists", conv.ID))
			continue
		}

		rule := schema.UserRule{
			ID:        conv.ID,
			Say:       conv.Say,
			Category:  conv.Category,
			Languages: conv.Languages,
			Severity:  conv.Severity,
			Autofix:   conv.Autofix,
			Message:   conv.Message,
			Example:   conv.Example,
			Include:   conv.Include,
			Exclude:   conv.Exclude,
		}

		userPolicy.Rules = append(userPolicy.Rules, rule)
		addedRules = append(addedRules, rule)
		existingIDs[conv.ID] = true
		succeeded = append(succeeded, conv.ID)
	}

	// Update defaults.languages with new languages from rules
	if len(addedRules) > 0 {
		policy.UpdateDefaultsLanguages(userPolicy, addedRules)
	}

	// Save policy if any succeeded
	if len(succeeded) > 0 {
		if err := policy.SavePolicy(userPolicy, ""); err != nil {
			return fmt.Errorf("failed to save policy: %w", err)
		}
	}

	// Print results
	printConventionBatchResult("Added", succeeded, failed)
	return nil
}

func runConventionEdit(cmd *cobra.Command, args []string) error {
	fileFlag, _ := cmd.Flags().GetString("file")

	// Load policy
	userPolicy, err := policy.LoadPolicy("")
	if err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}

	// Build rule index map
	ruleIndex := make(map[string]int)
	for i, rule := range userPolicy.Rules {
		ruleIndex[rule.ID] = i
	}

	var edits []ConventionEditItem

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
			return fmt.Errorf("usage: sym convention edit <id> [--flags] or sym convention edit -f <file>")
		}

		newID, _ := cmd.Flags().GetString("id")
		say, _ := cmd.Flags().GetString("say")
		category, _ := cmd.Flags().GetString("category")
		languages, _ := cmd.Flags().GetStringSlice("languages")
		severity, _ := cmd.Flags().GetString("severity")
		message, _ := cmd.Flags().GetString("message")
		example, _ := cmd.Flags().GetString("example")
		include, _ := cmd.Flags().GetStringSlice("include")
		exclude, _ := cmd.Flags().GetStringSlice("exclude")

		// Check if any edit flags were provided
		hasChanges := newID != "" || say != "" || category != "" || len(languages) > 0 ||
			severity != "" || message != "" || example != "" || len(include) > 0 || len(exclude) > 0 ||
			cmd.Flags().Changed("autofix")

		if !hasChanges {
			return fmt.Errorf("at least one edit flag must be provided (--id, --say, --category, etc.)")
		}

		var autofix *bool
		if cmd.Flags().Changed("autofix") {
			val, _ := cmd.Flags().GetBool("autofix")
			autofix = &val
		}

		edits = []ConventionEditItem{{
			ID:        args[0],
			NewID:     newID,
			Say:       say,
			Category:  category,
			Languages: languages,
			Severity:  severity,
			Autofix:   autofix,
			Message:   message,
			Example:   example,
			Include:   include,
			Exclude:   exclude,
		}}
	}

	var succeeded []string
	var failed []string
	var editedRules []schema.UserRule

	// Process each edit
	for _, edit := range edits {
		if edit.ID == "" {
			failed = append(failed, "(empty): ID is required")
			continue
		}

		// Check if at least one field to edit
		hasEdit := edit.NewID != "" || edit.Say != "" || edit.Category != "" ||
			len(edit.Languages) > 0 || edit.Severity != "" || edit.Autofix != nil ||
			edit.Message != "" || edit.Example != "" || len(edit.Include) > 0 || len(edit.Exclude) > 0

		if !hasEdit {
			failed = append(failed, fmt.Sprintf("%s: at least one field to edit is required", edit.ID))
			continue
		}

		idx, exists := ruleIndex[edit.ID]
		if !exists {
			failed = append(failed, fmt.Sprintf("%s: not found", edit.ID))
			continue
		}

		resultText := edit.ID

		// If renaming ID
		if edit.NewID != "" && edit.NewID != edit.ID {
			if _, dupExists := ruleIndex[edit.NewID]; dupExists {
				failed = append(failed, fmt.Sprintf("%s: '%s' already exists", edit.ID, edit.NewID))
				continue
			}

			delete(ruleIndex, edit.ID)
			ruleIndex[edit.NewID] = idx
			userPolicy.Rules[idx].ID = edit.NewID
			resultText = fmt.Sprintf("%s -> %s", edit.ID, edit.NewID)
		}

		// Update other fields
		if edit.Say != "" {
			userPolicy.Rules[idx].Say = edit.Say
		}
		if edit.Category != "" {
			userPolicy.Rules[idx].Category = edit.Category
		}
		if len(edit.Languages) > 0 {
			userPolicy.Rules[idx].Languages = edit.Languages
		}
		if edit.Severity != "" {
			userPolicy.Rules[idx].Severity = edit.Severity
		}
		if edit.Autofix != nil {
			userPolicy.Rules[idx].Autofix = *edit.Autofix
		}
		if edit.Message != "" {
			userPolicy.Rules[idx].Message = edit.Message
		}
		if edit.Example != "" {
			userPolicy.Rules[idx].Example = edit.Example
		}
		if len(edit.Include) > 0 {
			userPolicy.Rules[idx].Include = edit.Include
		}
		if len(edit.Exclude) > 0 {
			userPolicy.Rules[idx].Exclude = edit.Exclude
		}

		editedRules = append(editedRules, userPolicy.Rules[idx])
		succeeded = append(succeeded, resultText)
	}

	// Update defaults.languages with new languages from edited rules
	if len(editedRules) > 0 {
		policy.UpdateDefaultsLanguages(userPolicy, editedRules)
	}

	// Save policy if any succeeded
	if len(succeeded) > 0 {
		if err := policy.SavePolicy(userPolicy, ""); err != nil {
			return fmt.Errorf("failed to save policy: %w", err)
		}
	}

	// Print results
	printConventionBatchResult("Updated", succeeded, failed)
	return nil
}

func runConventionRemove(cmd *cobra.Command, args []string) error {
	fileFlag, _ := cmd.Flags().GetString("file")

	// Load policy
	userPolicy, err := policy.LoadPolicy("")
	if err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}

	// Build rule index map
	ruleIndex := make(map[string]int)
	for i, rule := range userPolicy.Rules {
		ruleIndex[rule.ID] = i
	}

	var ids []string

	if fileFlag != "" {
		// Batch mode: load from JSON file
		data, err := os.ReadFile(fileFlag)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		if err := json.Unmarshal(data, &ids); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
		if len(ids) == 0 {
			return fmt.Errorf("no convention IDs found in file")
		}
	} else if len(args) > 0 {
		// Single or batch mode: use args
		ids = args
	} else {
		return fmt.Errorf("usage: sym convention remove <id> [ids...] or sym convention remove -f <file>")
	}

	var succeeded []string
	var failed []string
	toRemove := make(map[int]bool)

	// Process each ID
	for _, id := range ids {
		if id == "" {
			failed = append(failed, "(empty): ID is required")
			continue
		}

		idx, exists := ruleIndex[id]
		if !exists {
			failed = append(failed, fmt.Sprintf("%s: not found", id))
			continue
		}

		toRemove[idx] = true
		succeeded = append(succeeded, id)
	}

	// Remove rules
	if len(toRemove) > 0 {
		newRules := make([]schema.UserRule, 0, len(userPolicy.Rules)-len(toRemove))
		for i, rule := range userPolicy.Rules {
			if !toRemove[i] {
				newRules = append(newRules, rule)
			}
		}
		userPolicy.Rules = newRules

		if err := policy.SavePolicy(userPolicy, ""); err != nil {
			return fmt.Errorf("failed to save policy: %w", err)
		}
	}

	// Print results
	printConventionBatchResult("Removed", succeeded, failed)
	return nil
}

// containsAny checks if haystack contains any of the needles.
func containsAny(haystack, needles []string) bool {
	for _, needle := range needles {
		for _, hay := range haystack {
			if hay == needle {
				return true
			}
		}
	}
	return false
}

// printConventionBatchResult prints the result of a batch operation for conventions.
func printConventionBatchResult(action string, succeeded, failed []string) {
	if len(failed) == 0 && len(succeeded) > 0 {
		if len(succeeded) == 1 {
			printDone(fmt.Sprintf("%s convention: %s", action, succeeded[0]))
		} else {
			printDone(fmt.Sprintf("%s %d conventions:", action, len(succeeded)))
			for _, id := range succeeded {
				fmt.Printf("  • %s\n", id)
			}
		}
	} else if len(succeeded) == 0 && len(failed) > 0 {
		printWarn(fmt.Sprintf("Failed to %s any conventions:", action))
		for _, f := range failed {
			fmt.Printf("  ✗ %s\n", f)
		}
	} else if len(succeeded) > 0 && len(failed) > 0 {
		printWarn("Batch operation completed with errors:")
		fmt.Printf("  ✓ %s (%d):\n", action, len(succeeded))
		for _, id := range succeeded {
			fmt.Printf("    • %s\n", id)
		}
		fmt.Printf("  ✗ Failed (%d):\n", len(failed))
		for _, f := range failed {
			fmt.Printf("    • %s\n", f)
		}
	} else {
		printWarn("No conventions to process")
	}
}
