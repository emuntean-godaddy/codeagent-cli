package docker_test

import (
	"context"
	"strings"
	"testing"

	"github.com/emuntean-godaddy/codeagent-cli/internal/docker"
)

func TestExecRunnerRunSuccess(t *testing.T) {
	t.Parallel()

	runner := docker.ExecRunner{}
	result, err := runner.Run(context.Background(), "sh", "-c", "echo out; echo err 1>&2")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if strings.TrimSpace(result.Stdout) != "out" {
		t.Fatalf("Stdout = %q, want %q", result.Stdout, "out")
	}
	if strings.TrimSpace(result.Stderr) != "err" {
		t.Fatalf("Stderr = %q, want %q", result.Stderr, "err")
	}
	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want %d", result.ExitCode, 0)
	}
}

func TestExecRunnerRunFailure(t *testing.T) {
	t.Parallel()

	runner := docker.ExecRunner{}
	result, err := runner.Run(context.Background(), "sh", "-c", "echo out; echo err 1>&2; exit 3")
	if err == nil {
		t.Fatalf("Run() error = nil, want error")
	}
	if strings.TrimSpace(result.Stdout) != "out" {
		t.Fatalf("Stdout = %q, want %q", result.Stdout, "out")
	}
	if strings.TrimSpace(result.Stderr) != "err" {
		t.Fatalf("Stderr = %q, want %q", result.Stderr, "err")
	}
	if result.ExitCode != 3 {
		t.Fatalf("ExitCode = %d, want %d", result.ExitCode, 3)
	}
	if !strings.Contains(err.Error(), "command sh -c") {
		t.Fatalf("Run() error = %q, want command context", err.Error())
	}
}
