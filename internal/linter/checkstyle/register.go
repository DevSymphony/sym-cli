package checkstyle

import (
	"github.com/DevSymphony/sym-cli/internal/linter"
)

func init() {
	_ = linter.Global().RegisterTool(
		New(linter.DefaultToolsDir()),
		NewConverter(),
		"checkstyle.xml",
	)
}
