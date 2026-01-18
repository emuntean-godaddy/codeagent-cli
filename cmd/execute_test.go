package cmd_test

import (
	"os"
	"testing"

	"github.com/emuntean-godaddy/codeagent-cli/cmd"
)

func TestExecuteNoArgsShowsHelp(t *testing.T) {
	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent"}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestExecuteUnknownCommand(t *testing.T) {
	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "nope"}

	if err := cmd.Execute(); err == nil {
		t.Fatalf("Execute() error = nil, want error")
	}
}
