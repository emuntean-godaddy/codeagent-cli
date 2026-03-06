package docker_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/emuntean-godaddy/codeagent-cli/internal/docker"
)

func TestContainerByLabel(t *testing.T) {
	t.Parallel()

	expectedArgs := []string{
		"ps",
		"-a",
		"--filter", "label=devcontainer.local_folder=/work/project",
		"--format", "{{.ID}}\t{{.State}}",
	}

	tests := []struct {
		name            string
		stdout          string
		stderr          string
		runErr          error
		wantInfo        docker.ContainerInfo
		wantErr         bool
		wantErrContains string
	}{
		{
			name:   "running",
			stdout: "abc123\trunning\n",
			wantInfo: docker.ContainerInfo{
				ID:    "abc123",
				State: docker.StateRunning,
			},
		},
		{
			name:   "stopped",
			stdout: "abc123\texited\n",
			wantInfo: docker.ContainerInfo{
				ID:    "abc123",
				State: docker.StateStopped,
			},
		},
		{
			name:   "missing",
			stdout: "",
			wantInfo: docker.ContainerInfo{
				State: docker.StateMissing,
			},
		},
		{
			name:    "multiple matches",
			stdout:  "a\trunning\nb\texited\n",
			wantErr: true,
		},
		{
			name:    "malformed output",
			stdout:  "abc123",
			wantErr: true,
		},
		{
			name:            "runner error includes stderr",
			stderr:          "boom",
			runErr:          errors.New("exit status 1"),
			wantErr:         true,
			wantErrContains: "boom",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			runner := runnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
				if name != "docker" {
					t.Fatalf("runner name = %q, want %q", name, "docker")
				}
				if !reflect.DeepEqual(args, expectedArgs) {
					t.Fatalf("runner args = %v, want %v", args, expectedArgs)
				}
				return docker.Result{Stdout: tt.stdout, Stderr: tt.stderr}, tt.runErr
			})

			got, err := docker.ContainerByLabel(context.Background(), runner, "devcontainer.local_folder", "/work/project")
			if (err != nil) != tt.wantErr {
				t.Fatalf("ContainerByLabel() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("ContainerByLabel() error = %q, want to contain %q", err.Error(), tt.wantErrContains)
				}
				return
			}
			if got != tt.wantInfo {
				t.Fatalf("ContainerByLabel() = %+v, want %+v", got, tt.wantInfo)
			}
		})
	}
}

func TestContainerByLocalFolderFallback(t *testing.T) {
	t.Parallel()

	raw := "/var/folders/project"
	resolved := "/private/var/folders/project"

	primaryArgs := []string{
		"ps",
		"-a",
		"--filter", "label=devcontainer.local_folder=" + raw,
		"--format", "{{.ID}}\t{{.State}}",
	}
	resolvedArgs := []string{
		"ps",
		"-a",
		"--filter", "label=devcontainer.local_folder=" + resolved,
		"--format", "{{.ID}}\t{{.State}}",
	}

	step := 0
	runner := runnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		step++
		switch step {
		case 1:
			if name != "docker" || !reflect.DeepEqual(args, primaryArgs) {
				t.Fatalf("primary args = %v, want %v", args, primaryArgs)
			}
			return docker.Result{Stdout: ""}, nil
		case 2:
			if name != "docker" || !reflect.DeepEqual(args, resolvedArgs) {
				t.Fatalf("resolved args = %v, want %v", args, resolvedArgs)
			}
			return docker.Result{Stdout: "abc123\trunning\n"}, nil
		default:
			t.Fatalf("runner called too many times: %d", step)
			return docker.Result{}, nil
		}
	})

	restore := docker.SetEvalSymlinks(func(path string) (string, error) {
		return resolved, nil
	})
	defer restore()

	info, err := docker.ContainerByLocalFolder(context.Background(), runner, raw)
	if err != nil {
		t.Fatalf("ContainerByLocalFolder() error = %v", err)
	}
	if info.ID != "abc123" || info.State != docker.StateRunning {
		t.Fatalf("ContainerByLocalFolder() = %+v, want running abc123", info)
	}
}

func TestContainerByLocalFolderAndConfig(t *testing.T) {
	t.Parallel()

	folder := "/work/project"
	configPath := "/work/project/.devcontainer/claude/devcontainer.json"
	expectedArgs := []string{
		"ps",
		"-a",
		"--filter", "label=devcontainer.config_file=" + configPath,
		"--filter", "label=devcontainer.local_folder=" + folder,
		"--format", "{{.ID}}\t{{.State}}",
	}

	runner := runnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		if name != "docker" {
			t.Fatalf("runner name = %q, want %q", name, "docker")
		}
		if !reflect.DeepEqual(args, expectedArgs) {
			t.Fatalf("runner args = %v, want %v", args, expectedArgs)
		}
		return docker.Result{Stdout: "abc123\trunning\n"}, nil
	})

	info, err := docker.ContainerByLocalFolderAndConfig(context.Background(), runner, folder, configPath)
	if err != nil {
		t.Fatalf("ContainerByLocalFolderAndConfig() error = %v", err)
	}
	if info.ID != "abc123" || info.State != docker.StateRunning {
		t.Fatalf("ContainerByLocalFolderAndConfig() = %+v, want running abc123", info)
	}
}
