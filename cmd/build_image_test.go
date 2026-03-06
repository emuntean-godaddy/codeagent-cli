package cmd_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/emuntean-godaddy/codeagent-cli/cmd"
	"github.com/emuntean-godaddy/codeagent-cli/internal/docker"
	"github.com/emuntean-godaddy/codeagent-cli/internal/project"
)

func TestBuildImageBuildsDefaultConfig(t *testing.T) {
	projectDir := t.TempDir()
	if err := writeDevcontainerJSON(filepath.Join(projectDir, ".devcontainer", "devcontainer.json"), `{"build":{"context":".","dockerfile":"Dockerfile"}}`); err != nil {
		t.Fatalf("writeDevcontainerJSON() error = %v", err)
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
	expectedArgs := []string{
		"build",
		"--workspace-folder", projectRoot,
		"--config", filepath.Join(projectRoot, ".devcontainer", "devcontainer.json"),
		"--image-name", "ans-search-api:devcontainer-base",
	}
	calls := 0
	restoreRunner := cmd.SetBuildImageRunner(startRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		calls++
		if name != "devcontainer" {
			t.Fatalf("runner command = %q, want devcontainer", name)
		}
		if !reflect.DeepEqual(args, expectedArgs) {
			t.Fatalf("runner args = %v, want %v", args, expectedArgs)
		}
		return docker.Result{}, nil
	}))
	t.Cleanup(restoreRunner)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "build-image", "--image-name", "ans-search-api:devcontainer-base"}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("runner calls = %d, want 1", calls)
	}

	data, err := os.ReadFile(filepath.Join(projectDir, ".devcontainer", "devcontainer.json"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if _, ok := payload["build"]; !ok {
		t.Fatalf("build missing, want unchanged build config")
	}
	if _, ok := payload["image"]; ok {
		t.Fatalf("image exists, want no mutation without --set-image")
	}
}

func TestBuildImageMissingImageNameFails(t *testing.T) {
	projectDir := t.TempDir()
	if err := writeDevcontainerJSON(filepath.Join(projectDir, ".devcontainer", "devcontainer.json"), `{"build":{"context":".","dockerfile":"Dockerfile"}}`); err != nil {
		t.Fatalf("writeDevcontainerJSON() error = %v", err)
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

	restoreRunner := cmd.SetBuildImageRunner(startRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		t.Fatalf("runner should not be called")
		return docker.Result{}, nil
	}))
	t.Cleanup(restoreRunner)
	var out bytes.Buffer
	restoreWriter := cmd.SetBuildImageWriter(&out)
	t.Cleanup(restoreWriter)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "build-image"}

	err = cmd.Execute()
	if err == nil {
		t.Fatalf("Execute() error = nil, want error")
	}
	if err.Error() != "Error: image name required: set --image-name" {
		t.Fatalf("Execute() error = %q, want missing image name error", err.Error())
	}
	if out.String() != "Error: image name required: set --image-name\n" {
		t.Fatalf("output = %q, want printed error", out.String())
	}
}

func TestBuildImageSetImageUpdatesConfig(t *testing.T) {
	projectDir := t.TempDir()
	if err := writeDevcontainerJSON(filepath.Join(projectDir, ".devcontainer", "claude", "devcontainer.json"), `{"build":{"context":"..","dockerfile":"../Dockerfile"}}`); err != nil {
		t.Fatalf("writeDevcontainerJSON() error = %v", err)
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
	expectedArgs := []string{
		"build",
		"--workspace-folder", projectRoot,
		"--config", filepath.Join(projectRoot, ".devcontainer", "claude", "devcontainer.json"),
		"--image-name", "harbor.muntean.online/homelab/agent-sandbox:030620260200",
	}
	restoreRunner := cmd.SetBuildImageRunner(startRunnerFunc(func(ctx context.Context, name string, args ...string) (docker.Result, error) {
		if name != "devcontainer" {
			t.Fatalf("runner command = %q, want devcontainer", name)
		}
		if !reflect.DeepEqual(args, expectedArgs) {
			t.Fatalf("runner args = %v, want %v", args, expectedArgs)
		}
		return docker.Result{}, nil
	}))
	t.Cleanup(restoreRunner)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{
		"codeagent", "build-image",
		"--tag", "claude",
		"--image-name", "harbor.muntean.online/homelab/agent-sandbox:030620260200",
		"--set-image",
	}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(projectDir, ".devcontainer", "claude", "devcontainer.json"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if payload["image"] != "harbor.muntean.online/homelab/agent-sandbox:030620260200" {
		t.Fatalf("image = %v, want harbor.muntean.online/homelab/agent-sandbox:030620260200", payload["image"])
	}
	if _, ok := payload["build"]; ok {
		t.Fatalf("build exists, want removed after --set-image")
	}
}
