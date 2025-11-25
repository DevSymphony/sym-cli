package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DevSymphony/sym-cli/internal/adapter/registry"
	"github.com/DevSymphony/sym-cli/internal/converter"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
	"github.com/spf13/cobra"
)

var (
	convertInputFile           string
	convertOutputFile          string
	convertTargets             []string
	convertOutputDir           string
	convertOpenAIModel         string
	convertConfidenceThreshold float64
	convertTimeout             int
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

  # Convert for multiple linters with specific model
  sym convert -i user-policy.json --targets checkstyle,pmd --openai-model gpt-4o

  # Use custom output directory
  sym convert -i user-policy.json --targets all --output-dir ./custom-dir

  # Legacy mode (internal policy only)
  sym convert -i user-policy.json -o code-policy.json`,
	RunE: runConvert,
}

func init() {
	convertCmd.Flags().StringVarP(&convertInputFile, "input", "i", "", "input user policy file (default: from .sym/.env POLICY_PATH)")
	convertCmd.Flags().StringVarP(&convertOutputFile, "output", "o", "", "output code policy file (legacy mode)")
	convertCmd.Flags().StringSliceVar(&convertTargets, "targets", []string{}, buildTargetsDescription())
	convertCmd.Flags().StringVar(&convertOutputDir, "output-dir", "", "output directory for linter configs (default: same as input file directory)")
	convertCmd.Flags().StringVar(&convertOpenAIModel, "openai-model", "gpt-4o", "OpenAI model to use for inference")
	convertCmd.Flags().Float64Var(&convertConfidenceThreshold, "confidence-threshold", 0.7, "minimum confidence for LLM inference (0.0-1.0)")
	convertCmd.Flags().IntVar(&convertTimeout, "timeout", 30, "timeout for API calls in seconds")
}

// buildTargetsDescription dynamically builds the --targets flag description
func buildTargetsDescription() string {
	tools := registry.Global().GetAllToolNames()
	if len(tools) == 0 {
		return "target linters (or 'all')"
	}
	return fmt.Sprintf("target linters (%s or 'all')", strings.Join(tools, ","))
}

func runConvert(cmd *cobra.Command, args []string) error {
	// Determine input file path
	if convertInputFile == "" {
		// Try to load from .env
		policyPath := loadPolicyPathFromEnv()
		if policyPath == "" {
			policyPath = ".sym/user-policy.json" // fallback default
		}
		convertInputFile = policyPath
		fmt.Printf("Using policy path from .env: %s\n", convertInputFile)
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

// loadPolicyPathFromEnv reads POLICY_PATH from .sym/.env
func loadPolicyPathFromEnv() string {
	envPath := filepath.Join(".sym", ".env")
	data, err := os.ReadFile(envPath)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	prefix := "POLICY_PATH="

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip comments and empty lines
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		// Check if line starts with POLICY_PATH=
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(line[len(prefix):])
		}
	}

	return ""
}

func runNewConverter(userPolicy *schema.UserPolicy) error {
	// Determine output directory
	if convertOutputDir == "" {
		// Default to .sym directory
		convertOutputDir = ".sym"
	}

	// Setup OpenAI client
	apiKey, err := getAPIKey()
	if err != nil {
		return fmt.Errorf("OpenAI API key required: %w", err)
	}

	timeout := time.Duration(convertTimeout) * time.Second
	llmClient := llm.NewClient(
		apiKey,
		llm.WithModel(convertOpenAIModel),
		llm.WithTimeout(timeout),
	)

	// Create new converter
	conv := converter.NewConverter(llmClient, convertOutputDir)

	// Setup context with generous timeout for parallel processing
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(convertTimeout*10)*time.Second)
	defer cancel()

	fmt.Printf("\n🚀 Converting with language-based routing and parallel LLM inference\n")
	fmt.Printf("📝 Model: %s\n", convertOpenAIModel)
	fmt.Printf("📂 Output: %s\n\n", convertOutputDir)

	// Convert
	result, err := conv.Convert(ctx, userPolicy)
	if err != nil {
		return fmt.Errorf("conversion failed: %w", err)
	}

	// Print results
	fmt.Printf("\n✅ Conversion completed successfully!\n")
	fmt.Printf("📦 Generated %d configuration file(s):\n", len(result.GeneratedFiles))
	for _, file := range result.GeneratedFiles {
		fmt.Printf("   ✓ %s\n", file)
	}

	if len(result.Errors) > 0 {
		fmt.Printf("\n⚠️  Errors (%d):\n", len(result.Errors))
		for linter, err := range result.Errors {
			fmt.Printf("   ✗ %s: %v\n", linter, err)
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Printf("\n⚠️  Warnings (%d):\n", len(result.Warnings))
		for _, warning := range result.Warnings {
			fmt.Printf("   • %s\n", warning)
		}
	}

	return nil
}
