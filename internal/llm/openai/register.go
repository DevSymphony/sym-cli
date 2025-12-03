package openai

import "github.com/DevSymphony/sym-cli/internal/llm"

func init() {
	llm.Register(&llm.Registration{
		Name:     "openai",
		Priority: 10,
		Factory:  New,
	})
}
