package main

import (
	"fmt"
	"os"

	"github.com/eggplants/gh-starlist/internal/cmd"
)

// version is set at build time by the release workflow.
var version = "dev"

func main() {
	if err := cmd.NewRoot(version).Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "gh-starlist: %v\n", err)
		os.Exit(1)
	}
}
