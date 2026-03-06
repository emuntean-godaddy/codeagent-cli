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
	containerID := "abc123"

	expectedPsArgs := labelArgsFor(projectRoot)
	resolvedPsArgs := labelArgsFor(resolved)
	expectedUpArgs := []string{"up", "--workspace-folder", projectRoot, "--config", filepath.Join(projectRoot, ".devcontainer", "devcontainer.json")}
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
	restoreExec := cmd.SetStartExec(func(path string, args []string, env []string) error {
		execArgs = append([]string{}, args...)
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
		"-e",
		"OPENAI_API_KEY=123",
		"-e",
		"OPENAI_BASE_URL=https://api",
		"abc123",
		"bash",
		"-lc",
		"codex resume abc -yolo",
	}
	if !reflect.DeepEqual(execArgs, wantExec) {
		t.Fatalf("exec args = %v, want %v", execArgs, wantExec)
	}
}

func TestStartInvalidEnv(t *testing.T) {
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
	os.Args = []string{"codeagent", "start", "-e", "bad key=value"}

	if err := cmd.Execute(); err == nil {
		t.Fatalf("Execute() error = nil, want error")
	}
	if !bytes.Contains(out.Bytes(), []byte("invalid env")) {
		t.Fatalf("output = %q, want invalid env error", out.String())
	}
}

func TestStartInvalidEnvKeyFormat(t *testing.T) {
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
		t.Fatalf("exec should not be called on invalid env key")
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
	os.Args = []string{"codeagent", "start", "-e", "1OPENAI_API_KEY=value"}

	if err := cmd.Execute(); err == nil {
		t.Fatalf("Execute() error = nil, want error")
	}
	if !bytes.Contains(out.Bytes(), []byte("key must match")) {
		t.Fatalf("output = %q, want invalid env key format error", out.String())
	}
}

func TestStartEnvFromLocalEnv(t *testing.T) {
	projectDir := t.TempDir()
	if err := writeDefaultDevcontainerJSON(projectDir); err != nil {
		t.Fatalf("writeDefaultDevcontainerJSON() error = %v", err)
	}

	t.Setenv("OPENAI_API_KEY", "secret-value")

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
	restoreExec := cmd.SetStartExec(func(path string, args []string, env []string) error {
		execArgs = append([]string{}, args...)
		return nil
	})
	t.Cleanup(restoreExec)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "start", "-e", "OPENAI_API_KEY"}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !reflect.DeepEqual(execArgs, []string{
		"docker",
		"exec",
		"-it",
		"-e",
		"OPENAI_API_KEY=secret-value",
		"abc123",
		"bash",
		"-lc",
		"codex --yolo",
	}) {
		t.Fatalf("exec args = %v, want env from local variable", execArgs)
	}
}

func TestStartEnvExpansionMissingLocalEnvFails(t *testing.T) {
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
		t.Fatalf("exec should not be called when env reference is missing")
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
	os.Args = []string{"codeagent", "start", "-e", "OPENAI_API_KEY=$MISSING_ENV"}

	if err := cmd.Execute(); err == nil {
		t.Fatalf("Execute() error = nil, want error")
	}
	if !bytes.Contains(out.Bytes(), []byte(`local env "MISSING_ENV" is not set`)) {
		t.Fatalf("output = %q, want missing local env error", out.String())
	}
}

func TestStartEnvExpansionFromLocalEnv(t *testing.T) {
	projectDir := t.TempDir()
	if err := writeDefaultDevcontainerJSON(projectDir); err != nil {
		t.Fatalf("writeDefaultDevcontainerJSON() error = %v", err)
	}

	t.Setenv("SOURCE_KEY", "from-local-env")

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
	restoreExec := cmd.SetStartExec(func(path string, args []string, env []string) error {
		execArgs = append([]string{}, args...)
		return nil
	})
	t.Cleanup(restoreExec)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "start", "-e", "OPENAI_API_KEY=$SOURCE_KEY"}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !reflect.DeepEqual(execArgs, []string{
		"docker",
		"exec",
		"-it",
		"-e",
		"OPENAI_API_KEY=from-local-env",
		"abc123",
		"bash",
		"-lc",
		"codex --yolo",
	}) {
		t.Fatalf("exec args = %v, want env expanded from local variable", execArgs)
	}
}

func TestStartEmptyCommand(t *testing.T) {
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

func TestStartWithoutDefaultRequiresTag(t *testing.T) {
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

	restoreRunner := cmd.SetStartRunner(startRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		t.Fatalf("runner should not be called when default config is missing")
		return docker.Result{}, nil
	}))
	t.Cleanup(restoreRunner)

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
	if !bytes.Contains(out.Bytes(), []byte("Use --tag")) {
		t.Fatalf("output = %q, want tag-required error", out.String())
	}
}

func TestStartWithTagUsesTaggedConfig(t *testing.T) {
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
	resolved, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		t.Fatalf("EvalSymlinks() error = %v", err)
	}
	expectedPsArgs := labelArgsForConfig(projectRoot, filepath.Join(projectRoot, ".devcontainer", "frontend", "devcontainer.json"))
	resolvedPsArgs := labelArgsForConfig(resolved, filepath.Join(resolved, ".devcontainer", "frontend", "devcontainer.json"))
	expectedUpArgs := []string{"up", "--workspace-folder", projectRoot, "--config", filepath.Join(projectRoot, ".devcontainer", "frontend", "devcontainer.json")}

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
			return docker.Result{Stdout: "abc123\trunning\n"}, nil
		case 4:
			if name != "docker" || !reflect.DeepEqual(args, []string{"exec", "abc123", "bash", "-lc", "exit 0"}) {
				t.Fatalf("bash check args = %v, want default bash check args", args)
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

	restoreExec := cmd.SetStartExec(func(path string, args []string, env []string) error {
		return nil
	})
	t.Cleanup(restoreExec)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "start", "--tag", "frontend"}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestStartUsesDefaultPostStartCommandWhenCommandNotProvided(t *testing.T) {
	projectDir := t.TempDir()
	if err := writeDevcontainerJSON(filepath.Join(projectDir, ".devcontainer", "devcontainer.json"), `{"name":"default","customizations":{"codeagent":{"startCommand":"claude"}}}`); err != nil {
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

	restoreLookPath := cmd.SetStartLookPath(func(name string) (string, error) { return "/usr/bin/docker", nil })
	t.Cleanup(restoreLookPath)

	var execArgs []string
	restoreExec := cmd.SetStartExec(func(path string, args []string, env []string) error {
		execArgs = append([]string{}, args...)
		return nil
	})
	t.Cleanup(restoreExec)

	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })
	os.Args = []string{"codeagent", "start"}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !reflect.DeepEqual(execArgs, []string{"docker", "exec", "-it", "abc123", "bash", "-lc", "claude"}) {
		t.Fatalf("exec args = %v, want codeagent startCommand from default config", execArgs)
	}
}

func TestStartUsesTaggedPostStartCommandWhenCommandNotProvided(t *testing.T) {
	projectDir := t.TempDir()
	if err := writeDevcontainerJSON(filepath.Join(projectDir, ".devcontainer", "claude", "devcontainer.json"), `{"name":"claude","customizations":{"codeagent":{"startCommand":"~/.local/bin/claude"}}}`); err != nil {
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
	resolved, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		t.Fatalf("EvalSymlinks() error = %v", err)
	}
	expectedPsArgs := labelArgsForConfig(projectRoot, filepath.Join(projectRoot, ".devcontainer", "claude", "devcontainer.json"))
	resolvedPsArgs := labelArgsForConfig(resolved, filepath.Join(resolved, ".devcontainer", "claude", "devcontainer.json"))

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

	restoreLookPath := cmd.SetStartLookPath(func(name string) (string, error) { return "/usr/bin/docker", nil })
	t.Cleanup(restoreLookPath)

	var execArgs []string
	restoreExec := cmd.SetStartExec(func(path string, args []string, env []string) error {
		execArgs = append([]string{}, args...)
		return nil
	})
	t.Cleanup(restoreExec)

	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })
	os.Args = []string{"codeagent", "start", "--tag", "claude"}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !reflect.DeepEqual(execArgs, []string{"docker", "exec", "-it", "abc123", "bash", "-lc", "~/.local/bin/claude"}) {
		t.Fatalf("exec args = %v, want codeagent startCommand from tagged config", execArgs)
	}
}

func TestStartFallsBackToLegacyPostStartCommandWhenCodeagentCommandMissing(t *testing.T) {
	projectDir := t.TempDir()
	if err := writeDevcontainerJSON(filepath.Join(projectDir, ".devcontainer", "devcontainer.json"), `{"name":"default","postStartCommand":"legacy-cmd"}`); err != nil {
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

	restoreLookPath := cmd.SetStartLookPath(func(name string) (string, error) { return "/usr/bin/docker", nil })
	t.Cleanup(restoreLookPath)

	var execArgs []string
	restoreExec := cmd.SetStartExec(func(path string, args []string, env []string) error {
		execArgs = append([]string{}, args...)
		return nil
	})
	t.Cleanup(restoreExec)

	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })
	os.Args = []string{"codeagent", "start"}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !reflect.DeepEqual(execArgs, []string{"docker", "exec", "-it", "abc123", "bash", "-lc", "legacy-cmd"}) {
		t.Fatalf("exec args = %v, want legacy postStartCommand fallback", execArgs)
	}
}

func writeDefaultDevcontainerJSON(projectDir string) error {
	return writeDevcontainerJSON(filepath.Join(projectDir, ".devcontainer", "devcontainer.json"), `{"name":"default"}`)
}

func writeDevcontainerJSON(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
