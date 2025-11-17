package registry

import "fmt"

// ErrAdapterNotFound is returned when no adapter is found for the given criteria.
type ErrAdapterNotFound struct {
	Language string
	Category string
}

func (e *ErrAdapterNotFound) Error() string {
	return fmt.Sprintf("no adapter found for language=%s category=%s", e.Language, e.Category)
}

// ErrLanguageNotSupported is returned when a language is not supported by any adapter.
type ErrLanguageNotSupported struct {
	Language string
}

func (e *ErrLanguageNotSupported) Error() string {
	return fmt.Sprintf("language %s is not supported by any adapter", e.Language)
}

// ErrNilAdapter is returned when trying to register a nil adapter.
var ErrNilAdapter = fmt.Errorf("cannot register nil adapter")
