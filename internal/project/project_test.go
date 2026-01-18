package project_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/emuntean-godaddy/codeagent-cli/internal/project"
)

func TestCurrentRootReturnsAbsolutePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "codeagent-project-*")
	if err != nil {
		t.Fatalf("MkdirTemp() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(tmpDir)
	})

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	got, err := project.CurrentRoot()
	if err != nil {
		t.Fatalf("CurrentRoot() error = %v", err)
	}

	want, err := filepath.Abs(tmpDir)
	if err != nil {
		t.Fatalf("Abs() error = %v", err)
	}

	gotResolved, err := filepath.EvalSymlinks(got)
	if err != nil {
		t.Fatalf("EvalSymlinks(got) error = %v", err)
	}
	wantResolved, err := filepath.EvalSymlinks(want)
	if err != nil {
		t.Fatalf("EvalSymlinks(want) error = %v", err)
	}

	if gotResolved != wantResolved {
		t.Fatalf("CurrentRoot() = %q, want %q", gotResolved, wantResolved)
	}
}

func TestCurrentRootGetwdFailure(t *testing.T) {
	restore := project.SetGetwd(func() (string, error) {
		return "", errors.New("boom")
	})
	t.Cleanup(restore)

	if _, err := project.CurrentRoot(); err == nil {
		t.Fatalf("CurrentRoot() error = nil, want error")
	}
}
