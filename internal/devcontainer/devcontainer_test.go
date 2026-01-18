package devcontainer_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/emuntean-godaddy/codeagent-cli/internal/devcontainer"
)

func TestValidateDirMissing(t *testing.T) {
	t.Parallel()

	if err := devcontainer.ValidateDir(t.TempDir()); err == nil {
		t.Fatalf("ValidateDir() error = nil, want error")
	}
}

func TestValidateDirNotDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, devcontainer.DirName)
	if err := os.WriteFile(path, []byte("file"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := devcontainer.ValidateDir(dir); err == nil {
		t.Fatalf("ValidateDir() error = nil, want error")
	}
}

func TestUpdateNameAddsField(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "devcontainer.json")
	contents := `{"image":"golang:1.22"}`
	if err := os.WriteFile(jsonPath, []byte(contents), 0o640); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := devcontainer.UpdateName(jsonPath, "project-a"); err != nil {
		t.Fatalf("UpdateName() error = %v", err)
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if payload["name"] != "project-a" {
		t.Fatalf("name = %v, want %v", payload["name"], "project-a")
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

func TestUpdateNameInvalidJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "devcontainer.json")
	if err := os.WriteFile(jsonPath, []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := devcontainer.UpdateName(jsonPath, "project-b"); err == nil {
		t.Fatalf("UpdateName() error = nil, want error")
	}
}

func TestUpdateNameMissingFile(t *testing.T) {
	t.Parallel()

	if err := devcontainer.UpdateName(filepath.Join(t.TempDir(), "devcontainer.json"), "project-c"); err == nil {
		t.Fatalf("UpdateName() error = nil, want error")
	}
}
