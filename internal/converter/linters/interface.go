package linters

import (
	"github.com/DevSymphony/sym-cli/internal/adapter"
)

// Type aliases for backward compatibility.
// The canonical definitions are now in the adapter package.
type (
	LinterConverter = adapter.LinterConverter
	LinterConfig    = adapter.LinterConfig
)
