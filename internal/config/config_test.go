package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/emuntean-godaddy/codeagent-cli/internal/config"
)

func TestDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir, err := config.Dir()
	if err != nil {
		t.Fatalf("Dir() error = %v", err)
	}

	want := filepath.Join(home, config.ConfigDirName)
	if dir != want {
		t.Fatalf("Dir() = %q, want %q", dir, want)
	}
}

func TestValidateMissingConfig(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), config.ConfigDirName)
	err := config.Validate(dir)
	var missing config.MissingConfigError
	if !errors.As(err, &missing) {
		t.Fatalf("Validate() error = %v, want MissingConfigError", err)
	}
}

func TestValidateMissingFile(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), config.ConfigDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, config.DockerfileName), []byte("FROM scratch\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := config.Validate(dir)
	var missing config.MissingConfigError
	if !errors.As(err, &missing) {
		t.Fatalf("Validate() error = %v, want MissingConfigError", err)
	}
}

func TestValidateFileIsDirectory(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), config.ConfigDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, config.DockerfileName), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, config.DevcontainerJSONName), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := config.Validate(dir)
	var missing config.MissingConfigError
	if !errors.As(err, &missing) {
		t.Fatalf("Validate() error = %v, want MissingConfigError", err)
	}
}

func TestValidateSuccess(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), config.ConfigDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, config.DockerfileName), []byte("FROM scratch\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, config.DevcontainerJSONName), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := config.Validate(dir); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}
