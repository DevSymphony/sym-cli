package main

import (
	"github.com/DevSymphony/sym-cli/internal/cmd"
)

// Version is set by build -ldflags "-X main.Version=x.y.z"
var Version = "dev"

func main() {
	// symphonyclient integration: Execute() doesn't return error
	cmd.Execute()
}
