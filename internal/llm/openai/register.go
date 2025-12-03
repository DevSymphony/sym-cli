package openai

import "github.com/DevSymphony/sym-cli/internal/llm/engine"

func init() {
	engine.Register(&engine.Registration{
		Name:     "openai",
		Priority: 10,
		Factory:  New,
	})
}
