package registry

import (
	"github.com/DevSymphony/sym-cli/internal/engine/core"
	"github.com/DevSymphony/sym-cli/internal/engine/length"
	"github.com/DevSymphony/sym-cli/internal/engine/pattern"
	"github.com/DevSymphony/sym-cli/internal/engine/style"
)

// init registers all built-in engines.
func init() {
	// Register pattern engine
	MustRegister("pattern", func() (core.Engine, error) {
		return pattern.NewEngine(), nil
	})

	// Register length engine
	MustRegister("length", func() (core.Engine, error) {
		return length.NewEngine(), nil
	})

	// Register style engine
	MustRegister("style", func() (core.Engine, error) {
		return style.NewEngine(), nil
	})
}
