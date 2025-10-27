package registry

import (
	"github.com/DevSymphony/sym-cli/internal/engine/core"
	"github.com/DevSymphony/sym-cli/internal/engine/length"
	"github.com/DevSymphony/sym-cli/internal/engine/pattern"
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
}
