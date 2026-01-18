package main

import (
	"os"

	"github.com/emuntean-godaddy/codeagent-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
