package cmd_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/emuntean-godaddy/codeagent-cli/cmd"
	"github.com/emuntean-godaddy/codeagent-cli/internal/docker"
	"github.com/emuntean-godaddy/codeagent-cli/internal/project"
)

type statusRunnerFunc func(ctx context.Context, name string, args ...string) (docker.Result, error)

func (f statusRunnerFunc) Run(ctx context.Context, name string, args ...string) (docker.Result, error) {
	return f(ctx, name, args...)
}

func TestStatusRunningOutput(t *testing.T) {
	projectDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	projectRoot, err := project.CurrentRoot()
	if err != nil {
		t.Fatalf("CurrentRoot() error = %v", err)
	}
	resolved, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		t.Fatalf("EvalSymlinks() error = %v", err)
	}
	expectedArgs := labelArgsFor(projectRoot)
	resolvedArgs := labelArgsFor(resolved)

	restoreRunner := cmd.SetStatusRunner(statusRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		if name != "docker" {
			t.Fatalf("runner name = %q, want %q", name, "docker")
		}
		if !argsMatchAny(args, expectedArgs, resolvedArgs) {
			t.Fatalf("runner args = %v, want %v or %v", args, expectedArgs, resolvedArgs)
		}
		return docker.Result{Stdout: "abc123\trunning\n"}, nil
	}))
	t.Cleanup(restoreRunner)

	var out bytes.Buffer
	restoreWriter := cmd.SetStatusWriter(&out)
	t.Cleanup(restoreWriter)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "status"}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	want := "Project: " + filepath.Base(projectRoot) + "\n" +
		"Path: " + projectRoot + "\n" +
		"Container: abc123\n" +
		"State: running\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestStatusMissingOutput(t *testing.T) {
	projectDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	projectRoot, err := project.CurrentRoot()
	if err != nil {
		t.Fatalf("CurrentRoot() error = %v", err)
	}
	resolved, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		t.Fatalf("EvalSymlinks() error = %v", err)
	}
	expectedArgs := labelArgsFor(projectRoot)
	resolvedArgs := labelArgsFor(resolved)

	callCount := 0
	restoreRunner := cmd.SetStatusRunner(statusRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		callCount++
		if name != "docker" {
			t.Fatalf("runner name = %q, want %q", name, "docker")
		}
		if !argsMatchAny(args, expectedArgs, resolvedArgs) {
			t.Fatalf("runner args = %v, want %v or %v", args, expectedArgs, resolvedArgs)
		}
		return docker.Result{Stdout: ""}, nil
	}))
	t.Cleanup(restoreRunner)

	var out bytes.Buffer
	restoreWriter := cmd.SetStatusWriter(&out)
	t.Cleanup(restoreWriter)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "status"}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if callCount == 0 {
		t.Fatalf("runner was not called")
	}

	want := "Project: " + filepath.Base(projectRoot) + "\n" +
		"Path: " + projectRoot + "\n" +
		"Container: missing\n" +
		"State: missing\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestStatusRunnerError(t *testing.T) {
	projectDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	restoreRunner := cmd.SetStatusRunner(statusRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		return docker.Result{Stderr: "boom"}, errors.New("exit status 1")
	}))
	t.Cleanup(restoreRunner)

	var out bytes.Buffer
	restoreWriter := cmd.SetStatusWriter(&out)
	t.Cleanup(restoreWriter)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "status"}

	if err := cmd.Execute(); err == nil {
		t.Fatalf("Execute() error = nil, want error")
	}
	if out.Len() != 0 {
		t.Fatalf("output = %q, want empty", out.String())
	}
}
