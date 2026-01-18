package cmd_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/emuntean-godaddy/codeagent-cli/cmd"
	"github.com/emuntean-godaddy/codeagent-cli/internal/config"
)

func TestInitMissingConfig(t *testing.T) {
	projectDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "init"}

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

	err = runCommand()
	if err == nil {
		t.Fatalf("init error = nil, want error")
	}
	if err.Error() != "Error: "+config.DisplayConfigDir+" configuration missing.\nExpected:\n  - "+config.DisplayConfigDir+"/Dockerfile\n  - "+config.DisplayConfigDir+"/devcontainer.json" {
		t.Fatalf("init error = %q, want missing config error", err.Error())
	}
}

func TestInitCreatesDevcontainer(t *testing.T) {
	projectDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := setupConfig(home, "FROM scratch\n", `{"image":"golang:1.22"}`, 0o640); err != nil {
		t.Fatalf("setupConfig() error = %v", err)
	}

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "init"}

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

	if err := runCommand(); err != nil {
		t.Fatalf("init error = %v", err)
	}

	jsonPath := filepath.Join(projectDir, ".devcontainer", config.DevcontainerJSONName)
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if payload["name"] != filepath.Base(projectDir) {
		t.Fatalf("name = %v, want %v", payload["name"], filepath.Base(projectDir))
	}
	if payload["image"] != "golang:1.22" {
		t.Fatalf("image = %v, want %v", payload["image"], "golang:1.22")
	}

	info, err := os.Stat(jsonPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %v, want %v", info.Mode().Perm(), 0o640)
	}
}

func TestInitOverwrite(t *testing.T) {
	projectDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := setupConfig(home, "FROM scratch\n", `{"image":"golang:1.22"}`, 0o640); err != nil {
		t.Fatalf("setupConfig() error = %v", err)
	}

	devDir := filepath.Join(projectDir, ".devcontainer")
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(devDir, config.DockerfileName), []byte("old\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(devDir, config.DevcontainerJSONName), []byte(`{"name":"old"}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "init", "--overwrite"}

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

	if err := runCommand(); err != nil {
		t.Fatalf("init error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(devDir, config.DockerfileName))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "FROM scratch\n" {
		t.Fatalf("Dockerfile = %q, want %q", string(data), "FROM scratch\n")
	}
}

func TestInitExistingDevcontainerNoOverwrite(t *testing.T) {
	projectDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := setupConfig(home, "FROM scratch\n", `{"image":"golang:1.22"}`, 0o640); err != nil {
		t.Fatalf("setupConfig() error = %v", err)
	}

	devDir := filepath.Join(projectDir, ".devcontainer")
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "init"}

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

	err = runCommand()
	if err == nil {
		t.Fatalf("init error = nil, want error")
	}
	if err.Error() != "Error: .devcontainer/ already exists. Use --overwrite to regenerate." {
		t.Fatalf("init error = %q, want overwrite message", err.Error())
	}
}

func runCommand() error {
	return cmd.Execute()
}

func setupConfig(home, dockerfile, jsonContent string, mode os.FileMode) error {
	dir := filepath.Join(home, config.ConfigDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, config.DockerfileName), []byte(dockerfile), mode); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, config.DevcontainerJSONName), []byte(jsonContent), mode); err != nil {
		return err
	}
	return nil
}
