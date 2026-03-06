package cmd_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/emuntean-godaddy/codeagent-cli/cmd"
	"github.com/emuntean-godaddy/codeagent-cli/internal/docker"
	"github.com/emuntean-godaddy/codeagent-cli/internal/project"
)

type destroyRunnerFunc func(ctx context.Context, name string, args ...string) (docker.Result, error)

func (f destroyRunnerFunc) Run(ctx context.Context, name string, args ...string) (docker.Result, error) {
	return f(ctx, name, args...)
}

func TestDestroyMissingContainer(t *testing.T) {
	projectDir := t.TempDir()
	if err := writeDefaultDevcontainerJSON(projectDir); err != nil {
		t.Fatalf("writeDefaultDevcontainerJSON() error = %v", err)
	}
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

	restoreRunner := cmd.SetDestroyRunner(destroyRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
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
	restoreWriter := cmd.SetDestroyWriter(&out)
	t.Cleanup(restoreWriter)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "destroy"}

	if err := cmd.Execute(); err == nil {
		t.Fatalf("Execute() error = nil, want error")
	}
	if out.Len() == 0 {
		t.Fatalf("output = %q, want error output", out.String())
	}
}

func TestDestroyRemovesContainer(t *testing.T) {
	projectDir := t.TempDir()
	if err := writeDefaultDevcontainerJSON(projectDir); err != nil {
		t.Fatalf("writeDefaultDevcontainerJSON() error = %v", err)
	}
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

	step := 0
	restoreRunner := cmd.SetDestroyRunner(destroyRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		step++
		switch step {
		case 1:
			if name != "docker" {
				t.Fatalf("runner name = %q, want %q", name, "docker")
			}
			if !argsMatchAny(args, expectedArgs, resolvedArgs) {
				t.Fatalf("runner args = %v, want %v or %v", args, expectedArgs, resolvedArgs)
			}
			return docker.Result{Stdout: "abc123\trunning\n"}, nil
		case 2:
			if name != "docker" {
				t.Fatalf("runner name = %q, want %q", name, "docker")
			}
			if !reflect.DeepEqual(args, []string{"rm", "-f", "abc123"}) {
				t.Fatalf("runner args = %v, want %v", args, []string{"rm", "-f", "abc123"})
			}
			return docker.Result{}, nil
		default:
			t.Fatalf("runner called too many times: %d", step)
			return docker.Result{}, nil
		}
	}))
	t.Cleanup(restoreRunner)

	var out bytes.Buffer
	restoreWriter := cmd.SetDestroyWriter(&out)
	t.Cleanup(restoreWriter)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "destroy"}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if out.String() != "Destroyed container for "+filepath.Base(projectRoot)+" (default): abc123\n" {
		t.Fatalf("output = %q, want destroy output", out.String())
	}
}

func TestDestroyRemoveError(t *testing.T) {
	projectDir := t.TempDir()
	if err := writeDefaultDevcontainerJSON(projectDir); err != nil {
		t.Fatalf("writeDefaultDevcontainerJSON() error = %v", err)
	}
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

	step := 0
	restoreRunner := cmd.SetDestroyRunner(destroyRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		step++
		switch step {
		case 1:
			if name != "docker" {
				t.Fatalf("runner name = %q, want %q", name, "docker")
			}
			if !argsMatchAny(args, expectedArgs, resolvedArgs) {
				t.Fatalf("runner args = %v, want %v or %v", args, expectedArgs, resolvedArgs)
			}
			return docker.Result{Stdout: "abc123\trunning\n"}, nil
		case 2:
			return docker.Result{Stderr: "nope"}, errors.New("exit status 1")
		default:
			t.Fatalf("runner called too many times: %d", step)
			return docker.Result{}, nil
		}
	}))
	t.Cleanup(restoreRunner)

	var out bytes.Buffer
	restoreWriter := cmd.SetDestroyWriter(&out)
	t.Cleanup(restoreWriter)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "destroy"}

	if err := cmd.Execute(); err == nil {
		t.Fatalf("Execute() error = nil, want error")
	}
	if !bytes.Contains(out.Bytes(), []byte("stderr: nope")) {
		t.Fatalf("output = %q, want stderr", out.String())
	}
}

func TestDestroyWithoutDefaultRequiresTag(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, ".devcontainer", "frontend"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, ".devcontainer", "frontend", "devcontainer.json"), []byte(`{"name":"frontend"}`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

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

	restoreRunner := cmd.SetDestroyRunner(destroyRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		t.Fatalf("runner should not be called")
		return docker.Result{}, nil
	}))
	t.Cleanup(restoreRunner)

	var out bytes.Buffer
	restoreWriter := cmd.SetDestroyWriter(&out)
	t.Cleanup(restoreWriter)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "destroy"}

	err = cmd.Execute()
	if err == nil {
		t.Fatalf("Execute() error = nil, want error")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("Use --tag")) {
		t.Fatalf("Execute() error = %q, want missing default guidance", err.Error())
	}
}

func TestDestroyWithTag(t *testing.T) {
	projectDir := t.TempDir()
	tagDir := filepath.Join(projectDir, ".devcontainer", "frontend")
	if err := os.MkdirAll(tagDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(tagDir, "devcontainer.json"), []byte(`{"name":"frontend"}`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

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

	step := 0
	restoreRunner := cmd.SetDestroyRunner(destroyRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		step++
		switch step {
		case 1:
			return docker.Result{Stdout: "abc123\trunning\n"}, nil
		case 2:
			if !reflect.DeepEqual(args, []string{"rm", "-f", "abc123"}) {
				t.Fatalf("runner args = %v, want rm -f abc123", args)
			}
			return docker.Result{}, nil
		default:
			t.Fatalf("runner called too many times: %d", step)
			return docker.Result{}, nil
		}
	}))
	t.Cleanup(restoreRunner)

	var out bytes.Buffer
	restoreWriter := cmd.SetDestroyWriter(&out)
	t.Cleanup(restoreWriter)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "destroy", "--tag", "frontend"}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("(frontend): abc123")) {
		t.Fatalf("output = %q, want tagged selector in message", out.String())
	}
}
