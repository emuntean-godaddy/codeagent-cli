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

type startRunnerFunc func(ctx context.Context, name string, args ...string) (docker.Result, error)

func (f startRunnerFunc) Run(ctx context.Context, name string, args ...string) (docker.Result, error) {
	return f(ctx, name, args...)
}

func TestStartMissingDevcontainer(t *testing.T) {
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

	var out bytes.Buffer
	restoreWriter := cmd.SetStartWriter(&out)
	t.Cleanup(restoreWriter)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "start"}

	if err := cmd.Execute(); err == nil {
		t.Fatalf("Execute() error = nil, want error")
	}
	if !bytes.Contains(out.Bytes(), []byte(".devcontainer/ directory not found")) {
		t.Fatalf("output = %q, want missing devcontainer error", out.String())
	}
}

func TestStartMissingContainerRunsUpAndExecsBash(t *testing.T) {
	projectDir := t.TempDir()
	devDir := filepath.Join(projectDir, ".devcontainer")
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
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
	containerID := "abc123"

	expectedPsArgs := labelArgsFor(projectRoot)
	resolvedPsArgs := labelArgsFor(resolved)
	expectedUpArgs := []string{"up", "--workspace-folder", projectRoot}
	expectedBashCheckArgs := []string{"exec", containerID, "bash", "-lc", "exit 0"}

	step := 0
	restoreRunner := cmd.SetStartRunner(startRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		step++
		switch step {
		case 1:
			if name != "docker" || !argsMatchAny(args, expectedPsArgs, resolvedPsArgs) {
				t.Fatalf("docker ps args = %v, want %v or %v", args, expectedPsArgs, resolvedPsArgs)
			}
			return docker.Result{Stdout: ""}, nil
		case 2:
			if name != "devcontainer" || !reflect.DeepEqual(args, expectedUpArgs) {
				t.Fatalf("devcontainer up args = %v, want %v", args, expectedUpArgs)
			}
			return docker.Result{}, nil
		case 3:
			if name != "docker" || !argsMatchAny(args, expectedPsArgs, resolvedPsArgs) {
				t.Fatalf("docker ps args = %v, want %v or %v", args, expectedPsArgs, resolvedPsArgs)
			}
			return docker.Result{Stdout: containerID + "\trunning\n"}, nil
		case 4:
			if name != "docker" || !reflect.DeepEqual(args, expectedBashCheckArgs) {
				t.Fatalf("bash check args = %v, want %v", args, expectedBashCheckArgs)
			}
			return docker.Result{}, nil
		default:
			t.Fatalf("runner called too many times: %d", step)
			return docker.Result{}, nil
		}
	}))
	t.Cleanup(restoreRunner)

	restoreLookPath := cmd.SetStartLookPath(func(name string) (string, error) {
		if name != "docker" {
			t.Fatalf("LookPath name = %q, want %q", name, "docker")
		}
		return "/usr/bin/docker", nil
	})
	t.Cleanup(restoreLookPath)

	var execArgs []string
	var execEnv []string
	restoreExec := cmd.SetStartExec(func(path string, args []string, env []string) error {
		execArgs = append([]string{}, args...)
		execEnv = append([]string{}, env...)
		return nil
	})
	t.Cleanup(restoreExec)

	var out bytes.Buffer
	restoreWriter := cmd.SetStartWriter(&out)
	t.Cleanup(restoreWriter)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "start"}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	wantExec := []string{
		"docker",
		"exec",
		"-it",
		containerID,
		"bash",
		"-lc",
		"codex --yolo",
	}
	if !reflect.DeepEqual(execArgs, wantExec) {
		t.Fatalf("exec args = %v, want %v", execArgs, wantExec)
	}
	if len(execEnv) == 0 {
		t.Fatalf("exec env missing")
	}
	if out.Len() != 0 {
		t.Fatalf("output = %q, want empty", out.String())
	}
}

func TestStartFallbackToSh(t *testing.T) {
	projectDir := t.TempDir()
	devDir := filepath.Join(projectDir, ".devcontainer")
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
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
	containerID := "abc123"

	expectedPsArgs := labelArgsFor(projectRoot)
	resolvedPsArgs := labelArgsFor(resolved)
	expectedBashCheckArgs := []string{"exec", containerID, "bash", "-lc", "exit 0"}
	expectedShCheckArgs := []string{"exec", containerID, "sh", "-lc", "exit 0"}

	step := 0
	restoreRunner := cmd.SetStartRunner(startRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		step++
		switch step {
		case 1:
			if name != "docker" || !argsMatchAny(args, expectedPsArgs, resolvedPsArgs) {
				t.Fatalf("docker ps args = %v, want %v or %v", args, expectedPsArgs, resolvedPsArgs)
			}
			return docker.Result{Stdout: containerID + "\trunning\n"}, nil
		case 2:
			if name != "docker" || !reflect.DeepEqual(args, expectedBashCheckArgs) {
				t.Fatalf("bash check args = %v, want %v", args, expectedBashCheckArgs)
			}
			return docker.Result{}, errors.New("bash missing")
		case 3:
			if name != "docker" || !reflect.DeepEqual(args, expectedShCheckArgs) {
				t.Fatalf("sh check args = %v, want %v", args, expectedShCheckArgs)
			}
			return docker.Result{}, nil
		default:
			t.Fatalf("runner called too many times: %d", step)
			return docker.Result{}, nil
		}
	}))
	t.Cleanup(restoreRunner)

	restoreLookPath := cmd.SetStartLookPath(func(name string) (string, error) {
		return "/usr/bin/docker", nil
	})
	t.Cleanup(restoreLookPath)

	var execArgs []string
	var execEnv []string
	restoreExec := cmd.SetStartExec(func(path string, args []string, env []string) error {
		execArgs = append([]string{}, args...)
		execEnv = append([]string{}, env...)
		return nil
	})
	t.Cleanup(restoreExec)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "start"}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	wantExec := []string{
		"docker",
		"exec",
		"-it",
		containerID,
		"sh",
		"-lc",
		"codex --yolo",
	}
	if !reflect.DeepEqual(execArgs, wantExec) {
		t.Fatalf("exec args = %v, want %v", execArgs, wantExec)
	}
	if len(execEnv) == 0 {
		t.Fatalf("exec env missing")
	}
}

func TestStartDevcontainerUpError(t *testing.T) {
	projectDir := t.TempDir()
	devDir := filepath.Join(projectDir, ".devcontainer")
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
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
	expectedPsArgs := labelArgsFor(projectRoot)
	resolvedPsArgs := labelArgsFor(resolved)

	step := 0
	restoreRunner := cmd.SetStartRunner(startRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		step++
		switch step {
		case 1:
			if name != "docker" || !argsMatchAny(args, expectedPsArgs, resolvedPsArgs) {
				t.Fatalf("docker ps args = %v, want %v or %v", args, expectedPsArgs, resolvedPsArgs)
			}
			return docker.Result{Stdout: ""}, nil
		case 2:
			return docker.Result{Stderr: "boom"}, errors.New("exit status 1")
		default:
			t.Fatalf("runner called too many times: %d", step)
			return docker.Result{}, nil
		}
	}))
	t.Cleanup(restoreRunner)

	restoreLookPath := cmd.SetStartLookPath(func(name string) (string, error) {
		return "/usr/bin/docker", nil
	})
	t.Cleanup(restoreLookPath)

	restoreExec := cmd.SetStartExec(func(path string, args []string, env []string) error {
		t.Fatalf("exec should not be called on devcontainer up failure")
		return nil
	})
	t.Cleanup(restoreExec)

	var out bytes.Buffer
	restoreWriter := cmd.SetStartWriter(&out)
	t.Cleanup(restoreWriter)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "start"}

	if err := cmd.Execute(); err == nil {
		t.Fatalf("Execute() error = nil, want error")
	}
	if !bytes.Contains(out.Bytes(), []byte("devcontainer up failed")) {
		t.Fatalf("output = %q, want devcontainer up error", out.String())
	}
}

func TestStartCustomCommandAndEnv(t *testing.T) {
	projectDir := t.TempDir()
	devDir := filepath.Join(projectDir, ".devcontainer")
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
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
	expectedPsArgs := labelArgsFor(projectRoot)
	resolvedPsArgs := labelArgsFor(resolved)

	step := 0
	restoreRunner := cmd.SetStartRunner(startRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		step++
		switch step {
		case 1:
			if name != "docker" || !argsMatchAny(args, expectedPsArgs, resolvedPsArgs) {
				t.Fatalf("docker ps args = %v, want %v or %v", args, expectedPsArgs, resolvedPsArgs)
			}
			return docker.Result{Stdout: "abc123\trunning\n"}, nil
		case 2:
			if name != "docker" || !reflect.DeepEqual(args, []string{"exec", "abc123", "bash", "-lc", "exit 0"}) {
				t.Fatalf("bash check args = %v, want %v", args, []string{"exec", "abc123", "bash", "-lc", "exit 0"})
			}
			return docker.Result{}, nil
		default:
			t.Fatalf("runner called too many times: %d", step)
			return docker.Result{}, nil
		}
	}))
	t.Cleanup(restoreRunner)

	restoreLookPath := cmd.SetStartLookPath(func(name string) (string, error) {
		return "/usr/bin/docker", nil
	})
	t.Cleanup(restoreLookPath)

	var execArgs []string
	var execEnv []string
	restoreExec := cmd.SetStartExec(func(path string, args []string, env []string) error {
		execArgs = append([]string{}, args...)
		execEnv = append([]string{}, env...)
		return nil
	})
	t.Cleanup(restoreExec)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "start", "-c", "codex resume abc -yolo", "-e", "OPENAI_API_KEY=123", "-e", "OPENAI_BASE_URL=https://api"}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	wantExec := []string{
		"docker",
		"exec",
		"-it",
		"abc123",
		"bash",
		"-lc",
		"codex resume abc -yolo",
	}
	if !reflect.DeepEqual(execArgs, wantExec) {
		t.Fatalf("exec args = %v, want %v", execArgs, wantExec)
	}
	if !containsEnv(execEnv, "OPENAI_API_KEY=123") || !containsEnv(execEnv, "OPENAI_BASE_URL=https://api") {
		t.Fatalf("exec env missing expected entries: %v", execEnv)
	}
}

func TestStartInvalidEnv(t *testing.T) {
	projectDir := t.TempDir()
	devDir := filepath.Join(projectDir, ".devcontainer")
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
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
	expectedPsArgs := labelArgsFor(projectRoot)
	resolvedPsArgs := labelArgsFor(resolved)

	restoreRunner := cmd.SetStartRunner(startRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		if name != "docker" || !argsMatchAny(args, expectedPsArgs, resolvedPsArgs) {
			t.Fatalf("docker ps args = %v, want %v or %v", args, expectedPsArgs, resolvedPsArgs)
		}
		return docker.Result{Stdout: "abc123\trunning\n"}, nil
	}))
	t.Cleanup(restoreRunner)

	restoreLookPath := cmd.SetStartLookPath(func(name string) (string, error) {
		return "/usr/bin/docker", nil
	})
	t.Cleanup(restoreLookPath)

	restoreExec := cmd.SetStartExec(func(path string, args []string, env []string) error {
		t.Fatalf("exec should not be called on invalid env")
		return nil
	})
	t.Cleanup(restoreExec)

	var out bytes.Buffer
	restoreWriter := cmd.SetStartWriter(&out)
	t.Cleanup(restoreWriter)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "start", "-e", "NOEQUALS"}

	if err := cmd.Execute(); err == nil {
		t.Fatalf("Execute() error = nil, want error")
	}
	if !bytes.Contains(out.Bytes(), []byte("invalid env")) {
		t.Fatalf("output = %q, want invalid env error", out.String())
	}
}

func TestStartEmptyCommand(t *testing.T) {
	projectDir := t.TempDir()
	devDir := filepath.Join(projectDir, ".devcontainer")
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
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
	expectedPsArgs := labelArgsFor(projectRoot)
	resolvedPsArgs := labelArgsFor(resolved)

	restoreRunner := cmd.SetStartRunner(startRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		if name != "docker" || !argsMatchAny(args, expectedPsArgs, resolvedPsArgs) {
			t.Fatalf("docker ps args = %v, want %v or %v", args, expectedPsArgs, resolvedPsArgs)
		}
		return docker.Result{Stdout: "abc123\trunning\n"}, nil
	}))
	t.Cleanup(restoreRunner)

	restoreLookPath := cmd.SetStartLookPath(func(name string) (string, error) {
		return "/usr/bin/docker", nil
	})
	t.Cleanup(restoreLookPath)

	restoreExec := cmd.SetStartExec(func(path string, args []string, env []string) error {
		t.Fatalf("exec should not be called on empty command")
		return nil
	})
	t.Cleanup(restoreExec)

	var out bytes.Buffer
	restoreWriter := cmd.SetStartWriter(&out)
	t.Cleanup(restoreWriter)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "start", "-c", "   "}

	if err := cmd.Execute(); err == nil {
		t.Fatalf("Execute() error = nil, want error")
	}
	if !bytes.Contains(out.Bytes(), []byte("command must not be empty")) {
		t.Fatalf("output = %q, want empty command error", out.String())
	}
}

func containsEnv(env []string, entry string) bool {
	for _, item := range env {
		if item == entry {
			return true
		}
	}
	return false
}
