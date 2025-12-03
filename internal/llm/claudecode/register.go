package claudecode

import "github.com/DevSymphony/sym-cli/internal/llm"

func init() {
	llm.Register(&llm.Registration{
		Name:     "claudecode",
		Priority: 20,
		Factory:  New,
	})
}
