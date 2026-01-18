package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/emuntean-godaddy/codeagent-cli/internal/config"
)

func TestCopyFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	src := filepath.Join(dir, "source.txt")
	dst := filepath.Join(dir, "dest.txt")

	if err := os.WriteFile(src, []byte("hello"), 0o700); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := config.CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile() error = %v", err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("dest content = %q, want %q", string(data), "hello")
	}

	info, err := os.Stat(dst)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("dest mode = %v, want %v", info.Mode().Perm(), 0o700)
	}
}

func TestCopyFileRejectsDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	src := filepath.Join(dir, "source-dir")
	dst := filepath.Join(dir, "dest.txt")

	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	if err := config.CopyFile(src, dst); err == nil {
		t.Fatalf("CopyFile() error = nil, want error")
	}
}

func TestCopyFileMissingSource(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	src := filepath.Join(dir, "missing.txt")
	dst := filepath.Join(dir, "dest.txt")

	if err := config.CopyFile(src, dst); err == nil {
		t.Fatalf("CopyFile() error = nil, want error")
	}
}
