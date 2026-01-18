package cmd_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/emuntean-godaddy/codeagent-cli/cmd"
	"github.com/emuntean-godaddy/codeagent-cli/internal/docker"
	"github.com/emuntean-godaddy/codeagent-cli/internal/project"
)

type stopRunnerFunc func(ctx context.Context, name string, args ...string) (docker.Result, error)

func (f stopRunnerFunc) Run(ctx context.Context, name string, args ...string) (docker.Result, error) {
	return f(ctx, name, args...)
}

func TestStopRunning(t *testing.T) {
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
	containerID := "abc123"

	expectedPsArgs := labelArgsFor(projectRoot)
	resolvedPsArgs := labelArgsFor(resolved)
	expectedStopArgs := []string{"stop", containerID}

	step := 0
	restoreRunner := cmd.SetStopRunner(stopRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		step++
		switch step {
		case 1:
			if name != "docker" {
				t.Fatalf("runner name = %q, want %q", name, "docker")
			}
			if !argsMatchAny(args, expectedPsArgs, resolvedPsArgs) {
				t.Fatalf("runner args = %v, want %v or %v", args, expectedPsArgs, resolvedPsArgs)
			}
			return docker.Result{Stdout: containerID + "\trunning\n"}, nil
		case 2:
			if name != "docker" {
				t.Fatalf("runner name = %q, want %q", name, "docker")
			}
			if !reflect.DeepEqual(args, expectedStopArgs) {
				t.Fatalf("runner args = %v, want %v", args, expectedStopArgs)
			}
			return docker.Result{}, nil
		default:
			t.Fatalf("runner called too many times: %d", step)
			return docker.Result{}, nil
		}
	}))
	t.Cleanup(restoreRunner)

	var out bytes.Buffer
	restoreWriter := cmd.SetStopWriter(&out)
	t.Cleanup(restoreWriter)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "stop"}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	want := "Stopped container for " + filepath.Base(projectRoot) + ": " + containerID + "\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestStopMissing(t *testing.T) {
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
	expectedPsArgs := labelArgsFor(projectRoot)
	resolvedPsArgs := labelArgsFor(resolved)

	restoreRunner := cmd.SetStopRunner(stopRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		if name != "docker" {
			t.Fatalf("runner name = %q, want %q", name, "docker")
		}
		if !argsMatchAny(args, expectedPsArgs, resolvedPsArgs) {
			t.Fatalf("runner args = %v, want %v or %v", args, expectedPsArgs, resolvedPsArgs)
		}
		return docker.Result{Stdout: ""}, nil
	}))
	t.Cleanup(restoreRunner)

	var out bytes.Buffer
	restoreWriter := cmd.SetStopWriter(&out)
	t.Cleanup(restoreWriter)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "stop"}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	want := "Container missing for " + filepath.Base(projectRoot) + ": missing\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestStopStopped(t *testing.T) {
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
	containerID := "abc123"

	restoreRunner := cmd.SetStopRunner(stopRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		return docker.Result{Stdout: containerID + "\texited\n"}, nil
	}))
	t.Cleanup(restoreRunner)

	var out bytes.Buffer
	restoreWriter := cmd.SetStopWriter(&out)
	t.Cleanup(restoreWriter)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "stop"}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	want := "Container already stopped for " + filepath.Base(projectRoot) + ": " + containerID + "\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestStopStopError(t *testing.T) {
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

	containerID := "abc123"

	step := 0
	restoreRunner := cmd.SetStopRunner(stopRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		step++
		switch step {
		case 1:
			return docker.Result{Stdout: containerID + "\trunning\n"}, nil
		case 2:
			return docker.Result{Stderr: "nope"}, errors.New("exit status 1")
		default:
			t.Fatalf("runner called too many times: %d", step)
			return docker.Result{}, nil
		}
	}))
	t.Cleanup(restoreRunner)

	var out bytes.Buffer
	restoreWriter := cmd.SetStopWriter(&out)
	t.Cleanup(restoreWriter)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "stop"}

	if err := cmd.Execute(); err == nil {
		t.Fatalf("Execute() error = nil, want error")
	}
	if !strings.Contains(out.String(), "stderr: nope") {
		t.Fatalf("output = %q, want stderr", out.String())
	}
}
