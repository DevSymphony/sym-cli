package registry

import (
	"github.com/DevSymphony/sym-cli/internal/engine/ast"
	"github.com/DevSymphony/sym-cli/internal/engine/core"
	"github.com/DevSymphony/sym-cli/internal/engine/length"
	"github.com/DevSymphony/sym-cli/internal/engine/pattern"
	"github.com/DevSymphony/sym-cli/internal/engine/style"
	"github.com/DevSymphony/sym-cli/internal/engine/typechecker"
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

	// Register AST engine
	MustRegister("ast", func() (core.Engine, error) {
		return ast.NewEngine(), nil
	})

	// Register type checker engine
	MustRegister("typechecker", func() (core.Engine, error) {
		return typechecker.NewEngine(), nil
	})
}
