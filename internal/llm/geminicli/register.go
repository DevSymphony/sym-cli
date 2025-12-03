package geminicli

import "github.com/DevSymphony/sym-cli/internal/llm"

func init() {
	llm.Register(&llm.Registration{
		Name:     "geminicli",
		Priority: 20,
		Factory:  New,
	})
}
