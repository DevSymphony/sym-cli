package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/DevSymphony/sym-cli/internal/adapter"
	"github.com/DevSymphony/sym-cli/internal/adapter/eslint"
	"github.com/DevSymphony/sym-cli/internal/adapter/prettier"
	"github.com/DevSymphony/sym-cli/internal/adapter/tsc"
)

// test-adapter is a CLI tool to test individual adapters
// Usage: go run cmd/test-adapter/main.go <adapter-name> [files...]
//
// Examples:
//   go run cmd/test-adapter/main.go eslint src/app.js
//   go run cmd/test-adapter/main.go tsc
//   go run cmd/test-adapter/main.go prettier --check

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: test-adapter <adapter-name> [files...]")
		fmt.Println("\nAvailable adapters:")
		fmt.Println("  - eslint")
		fmt.Println("  - prettier")
		fmt.Println("  - tsc")
		fmt.Println("\nExamples:")
		fmt.Println("  test-adapter eslint")
		fmt.Println("  test-adapter tsc")
		fmt.Println("  test-adapter prettier --check")
		os.Exit(1)
	}

	adapterName := os.Args[1]
	files := os.Args[2:]

	// Get current directory
	workDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	symDir := filepath.Join(workDir, ".sym")
	toolsDir := filepath.Join(os.Getenv("HOME"), ".sym", "tools")

	// Create adapter
	var adp adapter.Adapter
	switch adapterName {
	case "eslint":
		adp = eslint.NewAdapter(toolsDir, workDir)
	case "prettier":
		adp = prettier.NewAdapter(toolsDir, workDir)
	case "tsc":
		adp = tsc.NewAdapter(toolsDir, workDir)
	default:
		fmt.Printf("Unknown adapter: %s\n", adapterName)
		os.Exit(1)
	}

	fmt.Printf("🔧 Testing adapter: %s\n", adp.Name())
	fmt.Printf("📁 Working directory: %s\n", workDir)
	fmt.Printf("📁 .sym directory: %s\n", symDir)
	fmt.Printf("📁 Tools directory: %s\n\n", toolsDir)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Check availability
	fmt.Printf("🔍 Checking availability...\n")
	if err := adp.CheckAvailability(ctx); err != nil {
		fmt.Printf("⚠️  Not available: %v\n", err)
		fmt.Printf("📦 Installing...\n")
		if err := adp.Install(ctx, adapter.InstallConfig{
			ToolsDir: toolsDir,
		}); err != nil {
			fmt.Printf("❌ Installation failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Installed successfully\n\n")
	} else {
		fmt.Printf("✅ Available\n\n")
	}

	// Load config from .sym directory
	var config []byte
	configPath := getConfigPath(symDir, adapterName)

	fmt.Printf("📄 Looking for config: %s\n", configPath)
	if data, err := os.ReadFile(configPath); err == nil {
		config = data
		fmt.Printf("✅ Using config from %s\n\n", configPath)
	} else {
		fmt.Printf("⚠️  No config found, using default\n\n")
		config = []byte("{}")
	}

	// If no files specified, let the adapter use its default file discovery
	if len(files) == 0 {
		fmt.Printf("📂 No files specified, adapter will use its default file discovery\n\n")
	} else {
		fmt.Printf("📂 Files to check: %v\n\n", files)
	}

	// Execute adapter
	fmt.Printf("🚀 Running %s...\n", adapterName)
	output, err := adp.Execute(ctx, config, files)
	if err != nil {
		fmt.Printf("❌ Execution failed: %v\n", err)
		os.Exit(1)
	}

	// Display results
	fmt.Printf("\n📊 Results:\n")
	fmt.Printf("Exit Code: %d\n", output.ExitCode)
	fmt.Printf("Duration: %s\n\n", output.Duration)

	if output.Stdout != "" {
		fmt.Printf("📤 Stdout:\n%s\n\n", output.Stdout)
	}

	if output.Stderr != "" {
		fmt.Printf("📤 Stderr:\n%s\n\n", output.Stderr)
	}

	// Parse violations
	violations, err := adp.ParseOutput(output)
	if err != nil {
		fmt.Printf("⚠️  Failed to parse output: %v\n", err)
	} else {
		fmt.Printf("🔍 Found %d violation(s):\n\n", len(violations))
		for i, v := range violations {
			fmt.Printf("[%d] %s:%d:%d\n", i+1, v.File, v.Line, v.Column)
			fmt.Printf("    Severity: %s\n", v.Severity)
			fmt.Printf("    Message: %s\n", v.Message)
			if v.RuleID != "" {
				fmt.Printf("    Rule: %s\n", v.RuleID)
			}
			fmt.Printf("\n")
		}
	}

	// Print summary as JSON
	summary := map[string]interface{}{
		"adapter":    adapterName,
		"exitCode":   output.ExitCode,
		"duration":   output.Duration,
		"violations": len(violations),
	}

	summaryJSON, _ := json.MarshalIndent(summary, "", "  ")
	fmt.Printf("\n📋 Summary:\n%s\n", string(summaryJSON))

	// Exit with appropriate code
	if output.ExitCode != 0 || len(violations) > 0 {
		os.Exit(1)
	}
}

func getConfigPath(symDir, adapterName string) string {
	switch adapterName {
	case "eslint":
		return filepath.Join(symDir, ".eslintrc.json")
	case "prettier":
		return filepath.Join(symDir, ".prettierrc.json")
	case "tsc":
		return filepath.Join(symDir, "tsconfig.json")
	default:
		return filepath.Join(symDir, adapterName+".json")
	}
}
