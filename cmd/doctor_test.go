package cmd_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/emuntean-godaddy/codeagent-cli/cmd"
	"github.com/emuntean-godaddy/codeagent-cli/internal/config"
	"github.com/emuntean-godaddy/codeagent-cli/internal/docker"
)

type doctorRunnerFunc func(ctx context.Context, name string, args ...string) (docker.Result, error)

func (f doctorRunnerFunc) Run(ctx context.Context, name string, args ...string) (docker.Result, error) {
	return f(ctx, name, args...)
}

func TestDoctorReportsAllFailures(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("doctor test relies on sh and docker PATH for supported OS")
	}

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

	home := t.TempDir()
	t.Setenv("HOME", home)

	restoreRunner := cmd.SetDoctorRunner(doctorRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		return docker.Result{Stderr: "no daemon"}, errors.New("exit status 1")
	}))
	t.Cleanup(restoreRunner)

	var out bytes.Buffer
	restoreWriter := cmd.SetDoctorWriter(&out)
	t.Cleanup(restoreWriter)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "doctor"}

	err = cmd.Execute()
	if err == nil {
		t.Fatalf("Execute() error = nil, want error")
	}

	output := out.String()
	if !strings.Contains(output, "Docker Daemon:") {
		t.Fatalf("output = %q, want Docker Daemon line", output)
	}
	if !strings.Contains(output, "Config:") {
		t.Fatalf("output = %q, want Config line", output)
	}
	if !strings.Contains(output, "Devcontainer:") {
		t.Fatalf("output = %q, want Devcontainer line", output)
	}
}

func TestDoctorAllOK(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("doctor test relies on sh and docker PATH for supported OS")
	}

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

	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := setupConfig(home, "FROM scratch\n", `{"name":"x"}`, 0o644); err != nil {
		t.Fatalf("setupConfig() error = %v", err)
	}

	devDir := filepath.Join(projectDir, ".devcontainer")
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(devDir, config.DockerfileName), []byte("FROM scratch\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(devDir, config.DevcontainerJSONName), []byte(`{"name":"x"}`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	restoreRunner := cmd.SetDoctorRunner(doctorRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		if name != "docker" || len(args) != 1 || args[0] != "info" {
			t.Fatalf("unexpected runner call: %s %v", name, args)
		}
		return docker.Result{}, nil
	}))
	t.Cleanup(restoreRunner)

	var out bytes.Buffer
	restoreWriter := cmd.SetDoctorWriter(&out)
	t.Cleanup(restoreWriter)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "doctor"}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Docker CLI: ok") {
		t.Fatalf("output = %q, want Docker CLI ok", output)
	}
	if !strings.Contains(output, "Docker Daemon: ok") {
		t.Fatalf("output = %q, want Docker Daemon ok", output)
	}
	if !strings.Contains(output, "Config: ok") {
		t.Fatalf("output = %q, want Config ok", output)
	}
	if !strings.Contains(output, "Devcontainer: ok") {
		t.Fatalf("output = %q, want Devcontainer ok", output)
	}
}
