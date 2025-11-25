package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/DevSymphony/sym-cli/internal/adapter"
	"github.com/DevSymphony/sym-cli/internal/adapter/eslint"
)

func main() {
	fmt.Println("=== Testing ESLint Adapter ===")
	fmt.Println()

	// 1. Create ESLint adapter
	homeDir, _ := os.UserHomeDir()
	toolsDir := filepath.Join(homeDir, ".sym", "tools")
	workDir, _ := os.Getwd()

	adp := eslint.NewAdapter(toolsDir, workDir)
	fmt.Printf("✓ Created ESLint adapter\n")
	fmt.Printf("  Tools directory: %s\n", adp.ToolsDir)
	fmt.Printf("  Work directory: %s\n\n", adp.WorkDir)

	// 2. Check availability
	ctx := context.Background()
	fmt.Println("Checking ESLint availability...")
	err := adp.CheckAvailability(ctx)
	if err != nil {
		fmt.Printf("⚠️  ESLint not available: %v\n", err)
		fmt.Println("\nInstalling ESLint...")
		// Try to install
		installConfig := adapter.InstallConfig{
			ToolsDir: toolsDir,
		}
		installErr := adp.Install(ctx, installConfig)
		if installErr != nil {
			fmt.Printf("❌ Failed to install: %v\n", installErr)
			os.Exit(1)
		}
		fmt.Println("✓ ESLint installed successfully")
		fmt.Println()
	} else {
		fmt.Println("✓ ESLint is available")
		fmt.Println()
	}

	// 3. Create a simple ESLint config
	config := []byte(`{
		"env": {
			"node": true,
			"es2021": true
		},
		"extends": "eslint:recommended",
		"parserOptions": {
			"ecmaVersion": 12
		},
		"rules": {
			"semi": ["error", "always"],
			"quotes": ["error", "single"],
			"no-unused-vars": "error",
			"no-console": "warn"
		}
	}`)

	// 4. Find test file
	testFile := filepath.Join(workDir, "test_file.js")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		fmt.Printf("❌ Test file not found: %s\n", testFile)
		os.Exit(1)
	}
	fmt.Printf("Test file: %s\n\n", testFile)

	// 5. Execute ESLint
	fmt.Println("Running ESLint...")
	output, err := adp.Execute(ctx, config, []string{testFile})
	if err != nil {
		fmt.Printf("⚠️  ESLint execution error: %v\n", err)
		// Continue to parse output even if there's an error (violations cause non-zero exit)
	}

	// 6. Show raw output
	fmt.Println("\n--- Raw ESLint Output ---")
	if output.Stdout != "" {
		fmt.Printf("STDOUT:\n%s\n", output.Stdout)
	}
	if output.Stderr != "" {
		fmt.Printf("STDERR:\n%s\n", output.Stderr)
	}
	fmt.Printf("Exit Code: %d\n", output.ExitCode)
	fmt.Printf("Duration: %s\n", output.Duration)

	// 7. Parse violations
	fmt.Println("\n--- Parsed Violations ---")
	violations, parseErr := adp.ParseOutput(output)
	if parseErr != nil {
		fmt.Printf("❌ Failed to parse output: %v\n", parseErr)
		os.Exit(1)
	}

	if len(violations) == 0 {
		fmt.Println("✅ No violations found!")
	} else {
		fmt.Printf("Found %d violation(s):\n\n", len(violations))
		for i, v := range violations {
			fmt.Printf("%d. [%s] %s\n", i+1, v.Severity, v.RuleID)
			fmt.Printf("   File: %s:%d:%d\n", v.File, v.Line, v.Column)
			fmt.Printf("   Message: %s\n", v.Message)

			// Show that we have the raw output stored
			if len(output.Stdout) > 0 {
				fmt.Printf("   ✓ Raw output captured: %d bytes\n", len(output.Stdout))
			}
			if len(output.Stderr) > 0 {
				fmt.Printf("   ✓ Raw error captured: %d bytes\n", len(output.Stderr))
			}
			fmt.Printf("   ✓ Execution time: %s\n\n", output.Duration)
		}
	}

	fmt.Println("=== Test Complete ===")
}
