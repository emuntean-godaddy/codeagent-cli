package cmd_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
		"Devcontainer: default\n" +
		"Config: .devcontainer/devcontainer.json\n" +
		"Container: abc123\n" +
		"State: running\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestStatusMissingOutput(t *testing.T) {
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
		"Devcontainer: default\n" +
		"Config: .devcontainer/devcontainer.json\n" +
		"Container: missing\n" +
		"State: missing\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestStatusRunnerError(t *testing.T) {
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
	if !bytes.Contains(out.Bytes(), []byte("resolve container state")) {
		t.Fatalf("output = %q, want status error output", out.String())
	}
}

func TestStatusWithoutDefaultRequiresTag(t *testing.T) {
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

	restoreRunner := cmd.SetStatusRunner(statusRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		t.Fatalf("runner should not be called")
		return docker.Result{}, nil
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

	err = cmd.Execute()
	if err == nil {
		t.Fatalf("Execute() error = nil, want error")
	}
	if err.Error() != "Error: default .devcontainer/devcontainer.json not found. Use --tag to select a tagged devcontainer." {
		t.Fatalf("Execute() error = %q, want missing default guidance", err.Error())
	}
	if !bytes.Contains(out.Bytes(), []byte("Use --tag")) {
		t.Fatalf("output = %q, want missing default guidance output", out.String())
	}
}

func TestStatusWithTag(t *testing.T) {
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

	projectRoot, err := project.CurrentRoot()
	if err != nil {
		t.Fatalf("CurrentRoot() error = %v", err)
	}
	restoreRunner := cmd.SetStatusRunner(statusRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
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
	os.Args = []string{"codeagent", "status", "--tag", "frontend"}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("Devcontainer: frontend\nConfig: .devcontainer/frontend/devcontainer.json\n")) {
		t.Fatalf("output = %q, want tagged selector and config path", out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("Path: "+projectRoot+"\n")) {
		t.Fatalf("output = %q, want project path", out.String())
	}
}

func TestStatusAllProfiles(t *testing.T) {
	projectDir := t.TempDir()
	if err := writeDefaultDevcontainerJSON(projectDir); err != nil {
		t.Fatalf("writeDefaultDevcontainerJSON() error = %v", err)
	}
	if err := writeDevcontainerJSON(filepath.Join(projectDir, ".devcontainer", "claude", "devcontainer.json"), `{"name":"claude"}`); err != nil {
		t.Fatalf("writeDevcontainerJSON() error = %v", err)
	}
	if err := writeDevcontainerJSON(filepath.Join(projectDir, ".devcontainer", "gocode", "devcontainer.json"), `{"name":"gocode"}`); err != nil {
		t.Fatalf("writeDevcontainerJSON() error = %v", err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	projectRoot, err := project.CurrentRoot()
	if err != nil {
		t.Fatalf("CurrentRoot() error = %v", err)
	}
	defaultArgs := labelArgsForConfig(projectRoot, filepath.Join(projectRoot, ".devcontainer", "devcontainer.json"))
	claudeArgs := labelArgsForConfig(projectRoot, filepath.Join(projectRoot, ".devcontainer", "claude", "devcontainer.json"))
	gocodeArgs := labelArgsForConfig(projectRoot, filepath.Join(projectRoot, ".devcontainer", "gocode", "devcontainer.json"))

	restoreRunner := cmd.SetStatusRunner(statusRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		switch {
		case strings.Join(args, "\x00") == strings.Join(defaultArgs, "\x00"):
			return docker.Result{Stdout: "default123\trunning\n"}, nil
		case strings.Join(args, "\x00") == strings.Join(claudeArgs, "\x00"):
			return docker.Result{Stdout: ""}, nil
		case strings.Join(args, "\x00") == strings.Join(gocodeArgs, "\x00"):
			return docker.Result{Stdout: "gocode123\trunning\n"}, nil
		default:
			t.Fatalf("unexpected args: %v", args)
			return docker.Result{}, nil
		}
	}))
	t.Cleanup(restoreRunner)

	var out bytes.Buffer
	restoreWriter := cmd.SetStatusWriter(&out)
	t.Cleanup(restoreWriter)

	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })
	os.Args = []string{"codeagent", "status", "--all"}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Devcontainer: default") || !strings.Contains(output, "Container: default123") {
		t.Fatalf("output = %q, want default profile", output)
	}
	if !strings.Contains(output, "Devcontainer: claude") || !strings.Contains(output, "Container: missing") {
		t.Fatalf("output = %q, want claude profile missing", output)
	}
	if !strings.Contains(output, "Devcontainer: gocode") || !strings.Contains(output, "Container: gocode123") {
		t.Fatalf("output = %q, want gocode profile", output)
	}
}

func TestStatusAllWithTagFails(t *testing.T) {
	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })
	os.Args = []string{"codeagent", "status", "--all", "--tag", "claude"}

	var out bytes.Buffer
	restoreWriter := cmd.SetStatusWriter(&out)
	t.Cleanup(restoreWriter)

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("Execute() error = nil, want error")
	}
	if err.Error() != "Error: cannot combine --all with --tag" {
		t.Fatalf("Execute() error = %q, want combine flags error", err.Error())
	}
}
