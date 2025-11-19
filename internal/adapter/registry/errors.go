package registry

import "fmt"

// ErrAdapterNotFound is returned when no adapter is found for the given tool name.
type ErrAdapterNotFound struct {
	ToolName string
}

func (e *ErrAdapterNotFound) Error() string {
	return fmt.Sprintf("adapter not found: %s", e.ToolName)
}

// ErrNilAdapter is returned when trying to register a nil adapter.
var ErrNilAdapter = fmt.Errorf("cannot register nil adapter")
