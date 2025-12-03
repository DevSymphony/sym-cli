package claudecode

import "github.com/DevSymphony/sym-cli/internal/llm/engine"

func init() {
	engine.Register(&engine.Registration{
		Name:     "claudecode",
		Priority: 20,
		Factory:  New,
	})
}
