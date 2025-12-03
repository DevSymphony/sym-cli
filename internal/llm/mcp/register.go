package mcp

import "github.com/DevSymphony/sym-cli/internal/llm"

func init() {
	llm.Register(&llm.Registration{
		Name:     "mcp",
		Priority: 30,
		Factory:  New,
	})
}
