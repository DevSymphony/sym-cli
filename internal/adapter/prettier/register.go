package prettier

import (
	"github.com/DevSymphony/sym-cli/internal/adapter"
	"github.com/DevSymphony/sym-cli/internal/adapter/registry"
	"github.com/DevSymphony/sym-cli/internal/converter/linters"
)

func init() {
	_ = registry.Global().RegisterTool(
		NewAdapter(adapter.DefaultToolsDir()),
		linters.NewPrettierLinterConverter(),
		".prettierrc.json",
	)
}
