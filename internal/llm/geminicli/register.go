package geminicli

import "github.com/DevSymphony/sym-cli/internal/llm/engine"

func init() {
	engine.Register(&engine.Registration{
		Name:     "geminicli",
		Priority: 20,
		Factory:  New,
	})
}
