package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/DevSymphony/sym-cli/internal/config"
	"github.com/DevSymphony/sym-cli/internal/linter"
	"github.com/DevSymphony/sym-cli/internal/converter"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/internal/ui"
	"github.com/DevSymphony/sym-cli/pkg/schema"
	"github.com/spf13/cobra"
)

var (
	convertInputFile string
	convertTargets   []string
	convertOutputDir string
)

var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert user policies into linter configurations",
	Long: `Convert natural language policies (Schema A) written by users
into linter-specific configurations and internal validation schema (Schema B).

Supported linters are dynamically determined from registered adapters.
Uses OpenAI API to intelligently analyze natural language rules and
map them to appropriate linter rules.`,
	Example: `  # Convert to all supported linters (outputs to <git-root>/.sym)
  sym convert -i user-policy.json --targets all

  # Convert for specific linter
  sym convert -i user-policy.json --targets eslint

  # Convert for Java with specific model
  sym convert -i user-policy.json --targets checkstyle,pmd

  # Use custom output directory
  sym convert -i user-policy.json --targets all --output-dir ./custom-dir`,
	RunE: runConvert,
}

func init() {
	convertCmd.Flags().StringVarP(&convertInputFile, "input", "i", "", "input user policy file (default: from .sym/.env POLICY_PATH)")
	convertCmd.Flags().StringSliceVar(&convertTargets, "targets", []string{}, buildTargetsDescription())
	convertCmd.Flags().StringVar(&convertOutputDir, "output-dir", "", "output directory for linter configs (default: same as input file directory)")
}

// buildTargetsDescription dynamically builds the --targets flag description
func buildTargetsDescription() string {
	tools := linter.Global().GetAllToolNames()
	if len(tools) == 0 {
		return "target linters (or 'all')"
	}
	return fmt.Sprintf("target linters (%s or 'all')", strings.Join(tools, ","))
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

	ui.PrintTitle("Convert", "Language-based routing with parallel LLM inference")
	fmt.Printf("Output: %s\n\n", convertOutputDir)

	// Convert
	result, err := conv.Convert(ctx, userPolicy)
	if err != nil {
		return fmt.Errorf("conversion failed: %w", err)
	}

	// Print results
	fmt.Println()
	ui.PrintOK("Conversion completed successfully")
	fmt.Printf("Generated %d configuration file(s):\n", len(result.GeneratedFiles))
	for _, file := range result.GeneratedFiles {
		fmt.Printf("  - %s\n", file)
	}

	if len(result.Errors) > 0 {
		fmt.Println()
		ui.PrintWarn(fmt.Sprintf("Errors (%d):", len(result.Errors)))
		for linter, err := range result.Errors {
			fmt.Printf("  - %s: %v\n", linter, err)
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Println()
		ui.PrintWarn(fmt.Sprintf("Warnings (%d):", len(result.Warnings)))
		for _, warning := range result.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
	}

	return nil
}
