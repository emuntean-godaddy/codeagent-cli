package docker_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/emuntean-godaddy/codeagent-cli/internal/docker"
)

type runnerFunc func(ctx context.Context, name string, args ...string) (docker.Result, error)

func (f runnerFunc) Run(ctx context.Context, name string, args ...string) (docker.Result, error) {
	return f(ctx, name, args...)
}

func TestContainerStateRequiresName(t *testing.T) {
	t.Parallel()

	called := false
	runner := runnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		called = true
		return docker.Result{}, nil
	})

	_, err := docker.ContainerState(context.Background(), runner, "")
	if err == nil {
		t.Fatalf("ContainerState() error = nil, want error")
	}
	if called {
		t.Fatalf("ContainerState() called runner with empty name")
	}
}

func TestContainerState(t *testing.T) {
	t.Parallel()

	expectedArgs := []string{
		"ps",
		"-a",
		"--filter", "name=codeagent-test",
		"--format", "{{.Names}}\t{{.State}}",
	}

	tests := []struct {
		name            string
		stdout          string
		stderr          string
		runErr          error
		wantState       docker.State
		wantErr         bool
		wantErrContains string
	}{
		{
			name:      "running",
			stdout:    "codeagent-test\trunning\n",
			wantState: docker.StateRunning,
		},
		{
			name:      "stopped",
			stdout:    "codeagent-test\texited\n",
			wantState: docker.StateStopped,
		},
		{
			name:      "paused maps to stopped",
			stdout:    "codeagent-test\tpaused\n",
			wantState: docker.StateStopped,
		},
		{
			name:      "missing",
			stdout:    "",
			wantState: docker.StateMissing,
		},
		{
			name:      "non-exact match ignored",
			stdout:    "codeagent-test-old\trunning\n",
			wantState: docker.StateMissing,
		},
		{
			name:    "multiple exact matches",
			stdout:  "codeagent-test\trunning\ncodeagent-test\texited\n",
			wantErr: true,
		},
		{
			name:    "malformed output",
			stdout:  "codeagent-test",
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

			got, err := docker.ContainerState(context.Background(), runner, "codeagent-test")
			if (err != nil) != tt.wantErr {
				t.Fatalf("ContainerState() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("ContainerState() error = %q, want to contain %q", err.Error(), tt.wantErrContains)
				}
				return
			}
			if got != tt.wantState {
				t.Fatalf("ContainerState() = %q, want %q", got, tt.wantState)
			}
		})
	}
}
