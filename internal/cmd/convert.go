package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/DevSymphony/sym-cli/internal/config"
	"github.com/DevSymphony/sym-cli/internal/converter"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
	"github.com/spf13/cobra"
)

var (
	convertInputFile string
	convertOutputDir string
)

var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert user policies into linter configurations",
	Long: `Convert natural language policies (Schema A) written by users
into linter-specific configurations and internal validation schema (Schema B).

The conversion uses language-based routing with LLM inference to determine
which linters apply to each rule. Supported linters include ESLint, Prettier,
Pylint, TSC, Checkstyle, and PMD.`,
	Example: `  # Convert policy (outputs to .sym directory)
  sym convert -i user-policy.json

  # Convert with custom output directory
  sym convert -i user-policy.json -o ./custom-dir`,
	RunE: runConvert,
}

func init() {
	convertCmd.Flags().StringVarP(&convertInputFile, "input", "i", "", "input user policy file (default: from .sym/config.json)")
	convertCmd.Flags().StringVarP(&convertOutputDir, "output-dir", "o", "", "output directory for linter configs (default: .sym)")
}

func runConvert(cmd *cobra.Command, args []string) error {
	// Determine input file path
	if convertInputFile == "" {
		// Load from config.json
		projectCfg, _ := config.LoadProjectConfig()
		policyPath := projectCfg.PolicyPath
		if policyPath == "" {
			policyPath = ".sym/user-policy.json" // fallback default
		}
		convertInputFile = policyPath
		fmt.Printf("Using policy path from config: %s\n", convertInputFile)
	}

	// Read input file
	data, err := os.ReadFile(convertInputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	var userPolicy schema.UserPolicy
	if err := json.Unmarshal(data, &userPolicy); err != nil {
		return fmt.Errorf("failed to parse user policy: %w", err)
	}

	fmt.Printf("Loaded user policy with %d rules\n", len(userPolicy.Rules))

	// Use new converter by default (language-based routing with parallel LLM)
	return runNewConverter(&userPolicy)
}

func runNewConverter(userPolicy *schema.UserPolicy) error {
	// Determine output directory
	if convertOutputDir == "" {
		// Default to .sym directory
		convertOutputDir = ".sym"
	}

	// Create LLM provider
	cfg := llm.LoadConfig()
	cfg.Verbose = verbose
	llmProvider, err := llm.New(cfg)
	if err != nil {
		return fmt.Errorf("no available LLM backend for convert: %w\nTip: configure provider in .sym/config.json", err)
	}
	defer llmProvider.Close()

	// Create new converter
	conv := converter.NewConverter(llmProvider, convertOutputDir)

	// Setup context with generous timeout for parallel processing (10 minutes to match validator)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	printTitle("Convert", "Language-based routing with parallel LLM inference")
	fmt.Printf("Output: %s\n\n", convertOutputDir)

	// Convert
	result, err := conv.Convert(ctx, userPolicy)
	if err != nil {
		return fmt.Errorf("conversion failed: %w", err)
	}

	// Print results
	fmt.Println()
	printOK("Conversion completed successfully")
	fmt.Printf("Generated %d configuration file(s):\n", len(result.GeneratedFiles))
	for _, file := range result.GeneratedFiles {
		fmt.Printf("  - %s\n", file)
	}

	if len(result.Errors) > 0 {
		fmt.Println()
		printWarn(fmt.Sprintf("Errors (%d):", len(result.Errors)))
		for linter, err := range result.Errors {
			fmt.Printf("  - %s: %v\n", linter, err)
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Println()
		printWarn(fmt.Sprintf("Warnings (%d):", len(result.Warnings)))
		for _, warning := range result.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
	}

	return nil
}
