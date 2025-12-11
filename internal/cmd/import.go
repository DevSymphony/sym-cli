package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/DevSymphony/sym-cli/internal/importer"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/spf13/cobra"
)

var importMode string

var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import conventions from external document",
	Long: `Import coding conventions from an external document into user-policy.json.

Supported formats: txt, md, and code files (go, js, ts, py, java, etc.)

The import process:
1. Reads document content
2. Uses LLM to extract coding conventions
3. Generates categories and rules with unique IDs
4. Merges with existing user-policy.json

Import modes:
  append - Keep existing categories/rules, add new ones (default)
  clear  - Remove all existing categories/rules, then import`,
	Example: `  # Import from a file (append mode, default)
  sym import coding-standards.md

  # Clear existing conventions and import fresh
  sym import new-rules.md --mode clear`,
	Args: cobra.ExactArgs(1),
	RunE: runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)

	importCmd.Flags().StringVarP(&importMode, "mode", "m", "append",
		"Import mode: 'append' (keep existing, add new) or 'clear' (remove existing, then import)")
}

func runImport(cmd *cobra.Command, args []string) error {
	// Validate mode
	mode := importer.ImportModeAppend
	if importMode == "clear" {
		mode = importer.ImportModeClear
		// Confirm clear mode
		fmt.Println("WARNING: Clear mode will remove all existing categories and rules.")
		fmt.Print("Continue? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		confirm, _ := reader.ReadString('\n')
		confirm = strings.TrimSpace(confirm)
		if confirm != "y" && confirm != "Y" {
			fmt.Println("Import cancelled.")
			return nil
		}
	} else if importMode != "append" {
		return fmt.Errorf("invalid mode '%s': must be 'append' or 'clear'", importMode)
	}

	// Setup LLM provider
	llmCfg := llm.LoadConfig()
	llmCfg.Verbose = verbose
	llmProvider, err := llm.New(llmCfg)
	if err != nil {
		return fmt.Errorf("failed to create LLM provider: %w\nTip: configure provider in .sym/config.json", err)
	}
	defer func() { _ = llmProvider.Close() }()

	// Create importer
	imp := importer.NewImporter(llmProvider, verbose)

	// Setup context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Execute import
	input := &importer.ImportInput{
		Path: args[0],
		Mode: mode,
	}

	printTitle("Import Conventions", fmt.Sprintf("Processing: %s", args[0]))
	fmt.Printf("Mode: %s\n\n", importMode)

	result, err := imp.Import(ctx, input)
	if err != nil {
		// Print partial results if available
		if result != nil {
			printImportResults(result)
		}
		return fmt.Errorf("import failed: %w", err)
	}

	// Print results
	printImportResults(result)
	return nil
}

func printImportResults(result *importer.ImportResult) {
	fmt.Println()

	if result.FileProcessed != "" {
		printOK(fmt.Sprintf("Processed: %s", result.FileProcessed))
	}

	if result.CategoriesRemoved > 0 || result.RulesRemoved > 0 {
		fmt.Println()
		printWarn(fmt.Sprintf("Removed %d categories, %d rules (clear mode)",
			result.CategoriesRemoved, result.RulesRemoved))
	}

	if len(result.CategoriesAdded) > 0 {
		fmt.Println()
		printOK(fmt.Sprintf("Added %d categories:", len(result.CategoriesAdded)))
		for _, cat := range result.CategoriesAdded {
			fmt.Printf("    • %s: %s\n", cat.Name, cat.Description)
		}
	}

	if len(result.RulesAdded) > 0 {
		fmt.Println()
		printOK(fmt.Sprintf("Added %d rules:", len(result.RulesAdded)))
		for _, rule := range result.RulesAdded {
			fmt.Printf("    • [%s] %s (%s)\n", rule.ID, rule.Say, rule.Category)
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Println()
		printWarn(fmt.Sprintf("Warnings (%d):", len(result.Warnings)))
		for _, w := range result.Warnings {
			fmt.Printf("    %s\n", w)
		}
	}

	fmt.Println()
	printDone("Import complete")
}
