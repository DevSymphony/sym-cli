package prettier

import (
	"strings"

	"github.com/DevSymphony/sym-cli/internal/adapter"
)

// parseOutput converts Prettier --check output to violations.
//
// Prettier --check output format:
// "Checking formatting...
// src/app.js
// src/utils.js
// [warn] Code style issues found in the above file(s). Forgot to run Prettier?"
//
// Non-zero exit code means files need formatting.
func parseOutput(output *adapter.ToolOutput) ([]adapter.Violation, error) {
	// Exit code 0 = all files formatted
	if output.ExitCode == 0 {
		return nil, nil
	}

	// Parse stdout to find files that need formatting
	var violations []adapter.Violation
	lines := strings.Split(output.Stdout, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and status messages
		if line == "" || strings.HasPrefix(line, "Checking") ||
			strings.HasPrefix(line, "[warn]") || strings.HasPrefix(line, "Code style") {
			continue
		}

		// If line looks like a file path, it needs formatting
		if strings.Contains(line, ".js") || strings.Contains(line, ".ts") ||
			strings.Contains(line, ".jsx") || strings.Contains(line, ".tsx") {
			violations = append(violations, adapter.Violation{
				File:     line,
				Line:     0, // Prettier doesn't report line numbers in --check
				Column:   0,
				Message:  "Code style issues found. Run prettier --write to fix.",
				Severity: "warning",
				RuleID:   "prettier",
			})
		}
	}

	return violations, nil
}
