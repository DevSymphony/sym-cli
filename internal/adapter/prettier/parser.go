package prettier

import (
	"strings"

	"github.com/DevSymphony/sym-cli/internal/adapter"
)

// parseOutput converts Prettier --check output to violations.
//
// Prettier --check output format (older versions):
// "Checking formatting...
// src/app.js
// src/utils.js
// [warn] Code style issues found in the above file(s). Forgot to run Prettier?"
//
// Prettier --check output format (newer versions like 3.x):
// "Checking formatting...
// [warn] src/app.js
// [warn] src/utils.js
// [warn] Code style issues found in the above file(s). Run Prettier with --write to fix."
//
// Non-zero exit code means files need formatting.
func parseOutput(output *adapter.ToolOutput) ([]adapter.Violation, error) {
	// Exit code 0 = all files formatted
	if output.ExitCode == 0 {
		return nil, nil
	}

	// Parse both stdout and stderr to find files that need formatting
	// Prettier 3.x outputs [warn] messages to stderr
	var violations []adapter.Violation
	combined := output.Stdout + "\n" + output.Stderr
	lines := strings.Split(combined, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and status messages
		if line == "" || strings.HasPrefix(line, "Checking") {
			continue
		}

		// Handle [warn] prefix (Prettier 3.x format)
		if strings.HasPrefix(line, "[warn]") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "[warn]"))
			// Skip the summary message
			if strings.HasPrefix(line, "Code style") {
				continue
			}
		}

		// If line looks like a file path, it needs formatting
		if strings.Contains(line, ".js") || strings.Contains(line, ".ts") ||
			strings.Contains(line, ".jsx") || strings.Contains(line, ".tsx") ||
			strings.Contains(line, ".json") || strings.Contains(line, ".css") ||
			strings.Contains(line, ".md") || strings.Contains(line, ".yaml") ||
			strings.Contains(line, ".yml") {
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
