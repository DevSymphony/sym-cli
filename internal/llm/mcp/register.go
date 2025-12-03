package mcp

import "github.com/DevSymphony/sym-cli/internal/llm/engine"

func init() {
	engine.Register(&engine.Registration{
		Name:     "mcp",
		Priority: 30,
		Factory:  New,
	})
}
