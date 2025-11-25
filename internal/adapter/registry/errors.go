package registry

import "fmt"

// errAdapterNotFound is returned when no adapter is found for the given tool name.
type errAdapterNotFound struct {
	ToolName string
}

func (e *errAdapterNotFound) Error() string {
	return fmt.Sprintf("adapter not found: %s", e.ToolName)
}

// errNilAdapter is returned when trying to register a nil adapter.
var errNilAdapter = fmt.Errorf("cannot register nil adapter")
