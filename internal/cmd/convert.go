package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/DevSymphony/sym-cli/internal/converter"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
	"github.com/spf13/cobra"
)

var (
	convertInputFile        string
	convertOutputFile       string
	convertTargets          []string
	convertOutputDir        string
	convertOpenAIModel      string
	convertConfidenceThreshold float64
	convertTimeout          int
)

var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert user policies into linter configurations",
	Long: `Convert natural language policies (Schema A) written by users
into linter-specific configurations (ESLint, Checkstyle, PMD, etc.)
and internal validation schema (Schema B).

Uses OpenAI API to intelligently analyze natural language rules and
map them to appropriate linter rules.`,
	Example: `  # Convert to all supported linters (outputs to <git-root>/.sym)
  sym convert -i user-policy.json --targets all

  # Convert only for JavaScript/TypeScript
  sym convert -i user-policy.json --targets eslint

  # Convert for Java with specific model
  sym convert -i user-policy.json --targets checkstyle,pmd --openai-model gpt-4o

  # Use custom output directory
  sym convert -i user-policy.json --targets all --output-dir ./custom-dir

  # Legacy mode (internal policy only)
  sym convert -i user-policy.json -o code-policy.json`,
	RunE: runConvert,
}

func init() {
	rootCmd.AddCommand(convertCmd)

	convertCmd.Flags().StringVarP(&convertInputFile, "input", "i", "user-policy.json", "input user policy file")
	convertCmd.Flags().StringVarP(&convertOutputFile, "output", "o", "", "output code policy file (legacy mode)")
	convertCmd.Flags().StringSliceVar(&convertTargets, "targets", []string{}, "target linters (eslint,checkstyle,pmd or 'all')")
	convertCmd.Flags().StringVar(&convertOutputDir, "output-dir", "", "output directory for linter configs (default: <git-root>/.sym)")
	convertCmd.Flags().StringVar(&convertOpenAIModel, "openai-model", "gpt-4o-mini", "OpenAI model to use for inference")
	convertCmd.Flags().Float64Var(&convertConfidenceThreshold, "confidence-threshold", 0.7, "minimum confidence for LLM inference (0.0-1.0)")
	convertCmd.Flags().IntVar(&convertTimeout, "timeout", 30, "timeout for API calls in seconds")
}

func runConvert(cmd *cobra.Command, args []string) error {
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

	// Check mode: multi-target or legacy
	if len(convertTargets) > 0 {
		return runMultiTargetConvert(&userPolicy)
	}

	// Legacy mode: generate only internal code-policy.json
	return runLegacyConvert(&userPolicy)
}

func runLegacyConvert(userPolicy *schema.UserPolicy) error {
	outputFile := convertOutputFile
	if outputFile == "" {
		outputFile = "code-policy.json"
	}

	conv := converter.NewConverter()

	fmt.Printf("Converting %d natural language rules into structured policy...\n", len(userPolicy.Rules))

	codePolicy, err := conv.Convert(userPolicy)
	if err != nil {
		return fmt.Errorf("conversion failed: %w", err)
	}

	output, err := json.MarshalIndent(codePolicy, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize code policy: %w", err)
	}

	if err := os.WriteFile(outputFile, output, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("✓ Conversion completed: %s\n", outputFile)
	fmt.Printf("  - Processed rules: %d\n", len(codePolicy.Rules))
	if codePolicy.RBAC != nil {
		fmt.Printf("  - RBAC roles: %d\n", len(codePolicy.RBAC.Roles))
	}

	return nil
}

func runMultiTargetConvert(userPolicy *schema.UserPolicy) error {
	// Determine output directory
	if convertOutputDir == "" {
		// Use .sym directory in git root by default
		symDir, err := getSymDir()
		if err != nil {
			return fmt.Errorf("failed to determine output directory: %w (hint: run from within a git repository or use --output-dir)", err)
		}
		convertOutputDir = symDir
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(convertOutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Setup OpenAI client
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Warning: OPENAI_API_KEY not set, using fallback inference")
	}

	timeout := time.Duration(convertTimeout) * time.Second
	llmClient := llm.NewClient(
		apiKey,
		llm.WithModel(convertOpenAIModel),
		llm.WithTimeout(timeout),
		llm.WithVerbose(verbose),
	)

	// Create converter with LLM client
	conv := converter.NewConverter(converter.WithLLMClient(llmClient))

	// Setup context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(convertTimeout*len(userPolicy.Rules))*time.Second)
	defer cancel()

	fmt.Printf("\nConverting with OpenAI model: %s\n", convertOpenAIModel)
	fmt.Printf("Confidence threshold: %.2f\n", convertConfidenceThreshold)
	fmt.Printf("Output directory: %s\n\n", convertOutputDir)

	// Convert for multiple targets
	result, err := conv.ConvertMultiTarget(ctx, userPolicy, converter.MultiTargetConvertOptions{
		Targets:             convertTargets,
		OutputDir:           convertOutputDir,
		ConfidenceThreshold: convertConfidenceThreshold,
	})
	if err != nil {
		return fmt.Errorf("multi-target conversion failed: %w", err)
	}

	// Write linter configuration files
	filesWritten := 0
	for linterName, config := range result.LinterConfigs {
		outputPath := filepath.Join(convertOutputDir, config.Filename)

		if err := os.WriteFile(outputPath, config.Content, 0644); err != nil {
			return fmt.Errorf("failed to write %s config: %w", linterName, err)
		}

		fmt.Printf("✓ Generated %s configuration: %s\n", linterName, outputPath)

		// Print rule count
		if convResult, ok := result.Results[linterName]; ok {
			fmt.Printf("  - Rules: %d\n", len(convResult.Rules))
			if len(convResult.Warnings) > 0 {
				fmt.Printf("  - Warnings: %d\n", len(convResult.Warnings))
			}
		}

		filesWritten++
	}

	// Write internal code policy
	codePolicyPath := filepath.Join(convertOutputDir, "code-policy.json")
	codePolicyJSON, err := json.MarshalIndent(result.CodePolicy, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize code policy: %w", err)
	}

	if err := os.WriteFile(codePolicyPath, codePolicyJSON, 0644); err != nil {
		return fmt.Errorf("failed to write code policy: %w", err)
	}

	fmt.Printf("✓ Generated internal policy: %s\n", codePolicyPath)
	filesWritten++

	// Print summary
	fmt.Printf("\n✓ Conversion complete: %d files written\n", filesWritten)

	if len(result.Warnings) > 0 {
		fmt.Printf("\nWarnings (%d):\n", len(result.Warnings))
		for _, warning := range result.Warnings {
			fmt.Printf("  ⚠ %s\n", warning)
		}
	}

	return nil
}
