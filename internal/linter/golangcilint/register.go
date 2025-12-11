package golangcilint

import (
	"github.com/DevSymphony/sym-cli/internal/linter"
)

func init() {
	// Register golangci-lint linter, converter, and config file
	_ = linter.Global().RegisterTool(
		New(linter.DefaultToolsDir()),
		NewConverter(),
		".golangci.yml",
	)
}
