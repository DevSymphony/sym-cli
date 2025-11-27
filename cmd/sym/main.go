package main

import (
	"github.com/DevSymphony/sym-cli/internal/cmd"

	// Bootstrap: register all adapters
	_ "github.com/DevSymphony/sym-cli/internal/bootstrap"
)

// Version is set by build -ldflags "-X main.Version=x.y.z"
var Version = "dev"

func main() {
	// Set version for version command
	cmd.SetVersion(Version)

	// symphonyclient integration: Execute() doesn't return error
	cmd.Execute()
}
