package prettier

import (
	"github.com/DevSymphony/sym-cli/internal/adapter"
	"github.com/DevSymphony/sym-cli/internal/adapter/registry"
)

func init() {
	_ = registry.Global().RegisterTool(
		NewAdapter(adapter.DefaultToolsDir()),
		NewConverter(),
		".prettierrc",
	)
}
